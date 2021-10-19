// Package subimporter implements a bulk ZIP/CSV importer of subscribers.
// It implements a simple queue for buffering imports and committing records
// to DB along with ZIP and CSV handling utilities. It is meant to be used as
// a singleton as each Importer instance is stateful, where it keeps track of
// an import in progress. Only one import should happen on a single importer
// instance at a time.
package subimporter

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/mail"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
)

const (
	// stdInputMaxLen is the maximum allowed length for a standard input field.
	stdInputMaxLen = 200

	// commitBatchSize is the number of inserts to commit in a single SQL transaction.
	commitBatchSize = 10000
)

// Various import statuses.
const (
	StatusNone      = "none"
	StatusImporting = "importing"
	StatusStopping  = "stopping"
	StatusFinished  = "finished"
	StatusFailed    = "failed"

	ModeSubscribe = "subscribe"
	ModeBlocklist = "blocklist"
)

// Importer represents the bulk CSV subscriber import system.
type Importer struct {
	opt Options
	db  *sql.DB

	stop   chan bool
	status Status
	sync.RWMutex
}

// Options represents inport options.
type Options struct {
	UpsertStmt         *sql.Stmt
	BlocklistStmt      *sql.Stmt
	UpdateListDateStmt *sql.Stmt
	NotifCB            models.AdminNotifCallback
	SubsListStmt       *sql.Stmt
}

// Session represents a single import session.
type Session struct {
	im       *Importer
	subQueue chan SubReq
	log      *log.Logger

	opt SessionOpt
}

// SessionOpt represents the options for an importer session.
type SessionOpt struct {
	Filename  string `json:"filename"`
	Mode      string `json:"mode"`
	SubStatus string `json:"subscription_status"`
	Overwrite bool   `json:"overwrite"`
	Delim     string `json:"delim"`
	ListIDs   []int  `json:"lists"`
}

// Status reporesents statistics from an ongoing import session.
type Status struct {
	Name     string `json:"name"`
	Total    int    `json:"total"`
	Imported int    `json:"imported"`
	Status   string `json:"status"`
	logBuf   *bytes.Buffer
}

// SubReq is a wrapper over the Subscriber model.
type SubReq struct {
	models.Subscriber
	Lists          pq.Int64Array  `json:"lists"`
	ListUUIDs      pq.StringArray `json:"list_uuids"`
	PreconfirmSubs bool           `json:"preconfirm_subscriptions"`
}

type importStatusTpl struct {
	Name     string
	Status   string
	Imported int
	Total    int
}

var (
	// ErrIsImporting is thrown when an import request is made while an
	// import is already running.
	ErrIsImporting = errors.New("import is already running")

	csvHeaders = map[string]bool{
		"email":      true,
		"name":       true,
		"attributes": true}

	regexCleanStr = regexp.MustCompile("[[:^ascii:]]")
)

var (
	emailAbuseRegexp   = regexp.MustCompile("^abuse@(.*)$")
	emailSpamRegexp    = regexp.MustCompile("^spam@(.*)$")
	emailRegexp        = regexp.MustCompile("^(.*)@(?:(yahoo|ymail|rocketmail|aol|aim|hotmail|outlook|live|msn)).(.*)$")
	emailYahooRegexp   = regexp.MustCompile("^(.*)@(?:(yahoo|ymail|rocketmail)).(.*)$")
	emailHotmailRegexp = regexp.MustCompile("^(.*)@(?:(hotmail|outlook|live|msn)).(.*)$")
	emailGmailRegexp   = regexp.MustCompile("^(.*)@(?:(gmail|googlemail)).(.*)$")
	emailAolRegexp     = regexp.MustCompile("^(.*)@(?:(aol|aim|love|ygm|games|wow)).(.*)$")
)

// New returns a new instance of Importer.
func New(opt Options, db *sql.DB) *Importer {
	im := Importer{
		opt:    opt,
		stop:   make(chan bool, 1),
		db:     db,
		status: Status{Status: StatusNone, logBuf: bytes.NewBuffer(nil)},
	}
	return &im
}

// NewSession returns an new instance of Session. It takes the name
// of the uploaded file, but doesn't do anything with it but retains it for stats.
func (im *Importer) NewSession(opt SessionOpt) (*Session, error) {
	if im.getStatus() != StatusNone {
		return nil, errors.New("an import is already running")
	}

	im.Lock()
	im.status = Status{Status: StatusImporting,
		Name:   opt.Filename,
		logBuf: bytes.NewBuffer(nil)}
	im.Unlock()

	s := &Session{
		im:       im,
		log:      log.New(im.status.logBuf, "", log.Ldate|log.Ltime|log.Lshortfile),
		subQueue: make(chan SubReq, commitBatchSize),
		opt:      opt,
	}

	s.log.Printf("processing '%s'", opt.Filename)
	return s, nil
}

// GetStats returns the global Stats of the importer.
func (im *Importer) GetStats() Status {
	im.RLock()
	defer im.RUnlock()
	return Status{
		Name:     im.status.Name,
		Status:   im.status.Status,
		Total:    im.status.Total,
		Imported: im.status.Imported,
	}
}

// GetLogs returns the log entries of the last import session.
func (im *Importer) GetLogs() []byte {
	im.RLock()
	defer im.RUnlock()

	if im.status.logBuf == nil {
		return []byte{}
	}
	return im.status.logBuf.Bytes()
}

// setStatus sets the Importer's status.
func (im *Importer) setStatus(status string) {
	im.Lock()
	im.status.Status = status
	im.Unlock()
}

// getStatus get's the Importer's status.
func (im *Importer) getStatus() string {
	im.RLock()
	status := im.status.Status
	im.RUnlock()
	return status
}

// isDone returns true if the importer is working (importing|stopping).
func (im *Importer) isDone() bool {
	s := true
	im.RLock()
	if im.getStatus() == StatusImporting || im.getStatus() == StatusStopping {
		s = false
	}
	im.RUnlock()
	return s
}

// incrementImportCount sets the Importer's "imported" counter.
func (im *Importer) incrementImportCount(n int) {
	im.Lock()
	im.status.Imported += n
	im.Unlock()
}

// sendNotif sends admin notifications for import completions.
func (im *Importer) sendNotif(status string) error {
	var (
		s   = im.GetStats()
		out = importStatusTpl{
			Name:     s.Name,
			Status:   status,
			Imported: s.Imported,
			Total:    s.Total,
		}
		subject = fmt.Sprintf("%s: %s import",
			strings.Title(status),
			s.Name)
	)
	return im.opt.NotifCB(subject, out)
}

// Start is a blocking function that selects on a channel queue until all
// subscriber entries in the import session are imported. It should be
// invoked as a goroutine.
func (s *Session) Start() {
	var (
		tx     *sql.Tx
		stmt   *sql.Stmt
		stmtSL *sql.Stmt
		err    error
		total  = 0
		cur    = 0

		listIDs = make(pq.Int64Array, len(s.opt.ListIDs))
	)

	for i, v := range s.opt.ListIDs {
		listIDs[i] = int64(v)
	}

	for sub := range s.subQueue {
		if cur == 0 {
			// New transaction batch.
			tx, err = s.im.db.Begin()
			if err != nil {
				s.log.Printf("error creating DB transaction: %v", err)
				continue
			}

			if s.opt.Mode == ModeSubscribe {
				stmt = tx.Stmt(s.im.opt.UpsertStmt)
			} else {
				stmt = tx.Stmt(s.im.opt.BlocklistStmt)
			}

			stmtSL = tx.Stmt(s.im.opt.SubsListStmt)
		}

		uu, err := uuid.NewV4()
		if err != nil {
			s.log.Printf("error generating UUID: %v", err)
			tx.Rollback()
			break
		}

		if s.opt.Mode == ModeSubscribe && !(emailAbuseRegexp.MatchString(sub.Email) || emailSpamRegexp.MatchString(sub.Email) || !emailRegexp.MatchString(sub.Email)) {
			//if s.opt.Mode == ModeSubscribe {
			_, err = stmt.Exec(uu, sub.Email, sub.Name, sub.Attribs, listIDs, s.opt.SubStatus, s.opt.Overwrite)
		} else if s.opt.Mode == ModeBlocklist {
			_, err = stmt.Exec(uu, sub.Email, sub.Name, sub.Attribs)
		}

		listName := ValidateSmartEmail(sub.Email)
		if listName != "" {
			_, err = stmtSL.Exec(sub.Email, listName)
		}
		if err != nil {
			s.log.Printf("error executing insert: %v", err)
			tx.Rollback()
			break
		}
		cur++
		total++

		// Batch size is met. Commit.
		if cur%commitBatchSize == 0 {
			if err := tx.Commit(); err != nil {
				tx.Rollback()
				s.log.Printf("error committing to DB: %v", err)
			} else {
				s.im.incrementImportCount(cur)
				s.log.Printf("imported %d", total)
			}

			cur = 0
		}
	}

	// Queue's closed and there's nothing left to commit.
	if cur == 0 {
		s.im.setStatus(StatusFinished)
		s.log.Printf("imported finished")
		if _, err := s.im.opt.UpdateListDateStmt.Exec(listIDs); err != nil {
			s.log.Printf("error updating lists date: %v", err)
		}
		s.im.sendNotif(StatusFinished)
		return
	}

	// Queue's closed and there are records left to commit.
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		s.im.setStatus(StatusFailed)
		s.log.Printf("error committing to DB: %v", err)
		s.im.sendNotif(StatusFailed)
		return
	}

	s.im.incrementImportCount(cur)
	s.im.setStatus(StatusFinished)
	s.log.Printf("imported finished")
	if _, err := s.im.opt.UpdateListDateStmt.Exec(listIDs); err != nil {
		s.log.Printf("error updating lists date: %v", err)
	}
	s.im.sendNotif(StatusFinished)
}

// Stop stops an active import session.
func (s *Session) Stop() {
	close(s.subQueue)
}

// ExtractZIP takes a ZIP file's path and extracts all .csv files in it to
// a temporary directory, and returns the name of the temp directory and the
// list of extracted .csv files.
func (s *Session) ExtractZIP(srcPath string, maxCSVs int) (string, []string, error) {
	if s.im.isDone() {
		return "", nil, ErrIsImporting
	}

	failed := true
	defer func() {
		if failed {
			s.im.setStatus(StatusFailed)
		}
	}()

	z, err := zip.OpenReader(srcPath)
	if err != nil {
		return "", nil, err
	}
	defer z.Close()

	// Create a temporary directory to extract the files.
	dir, err := ioutil.TempDir("", "listmonk")
	if err != nil {
		s.log.Printf("error creating temporary directory for extracting ZIP: %v", err)
		return "", nil, err
	}

	files := make([]string, 0, len(z.File))
	for _, f := range z.File {
		fName := f.FileInfo().Name()

		// Skip directories.
		if f.FileInfo().IsDir() {
			s.log.Printf("skipping directory '%s'", fName)
			continue
		}

		// Skip files without the .csv extension.
		if !strings.HasSuffix(strings.ToLower(fName), ".csv") {
			s.log.Printf("skipping non .csv file '%s'", fName)
			continue
		}

		s.log.Printf("extracting '%s'", fName)
		src, err := f.Open()
		if err != nil {
			s.log.Printf("error opening '%s' from ZIP: '%v'", fName, err)
			return "", nil, err
		}
		defer src.Close()

		out, err := os.OpenFile(dir+"/"+fName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			s.log.Printf("error creating '%s/%s': '%v'", dir, fName, err)
			return "", nil, err
		}
		defer out.Close()

		if _, err := io.Copy(out, src); err != nil {
			s.log.Printf("error extracting to '%s/%s': '%v'", dir, fName, err)
			return "", nil, err
		}
		s.log.Printf("extracted '%s'", fName)

		files = append(files, fName)
		if len(files) > maxCSVs {
			s.log.Printf("won't extract any more files. Maximum is %d", maxCSVs)
			break
		}
	}

	if len(files) == 0 {
		s.log.Println("no CSV files found in the ZIP")
		return "", nil, errors.New("no CSV files found in the ZIP")
	}

	failed = false
	return dir, files, nil
}

// LoadCSV loads a CSV file and validates and imports the subscriber entries in it.
func (s *Session) LoadCSV(srcPath string, delim rune) error {
	if s.im.isDone() {
		return ErrIsImporting
	}

	// Default status is "failed" in case the function
	// returns at one of the many possible errors.
	failed := true
	defer func() {
		if failed {
			s.im.setStatus(StatusFailed)
		}
	}()

	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}

	// Count the total number of lines in the file. This doesn't distinguish
	// between "blank" and non "blank" lines, and is only used to derive
	// the progress percentage for the frontend.
	numLines, err := countLines(f)
	if err != nil {
		s.log.Printf("error counting lines in '%s': '%v'", srcPath, err)
		return err
	}

	if numLines == 0 {
		return errors.New("empty file")
	}

	s.im.Lock()
	// Exclude the header from count.
	s.im.status.Total = numLines - 1
	s.im.Unlock()

	// Rewind, now that we've done a linecount on the same handler.
	_, _ = f.Seek(0, 0)
	rd := csv.NewReader(f)
	rd.Comma = delim

	// Read the header.
	csvHdr, err := rd.Read()
	if err != nil {
		s.log.Printf("error reading header from '%s': '%v'", srcPath, err)
		return err
	}

	hdrKeys := s.mapCSVHeaders(csvHdr, csvHeaders)
	// email, and name are required headers.
	if _, ok := hdrKeys["email"]; !ok {
		s.log.Printf("'email' column not found in '%s'", srcPath)
		return errors.New("'email' column not found")
	}
	if _, ok := hdrKeys["name"]; !ok {
		s.log.Printf("'name' column not found in '%s'", srcPath)
		return errors.New("'name' column not found")
	}

	var (
		lnHdr = len(hdrKeys)
		i     = 0
		cur   = 0
	)
	for {
		i++

		// Check for the stop signal.
		select {
		case <-s.im.stop:
			failed = false
			close(s.subQueue)
			s.log.Println("stop request received")
			return nil
		default:
		}

		cols, err := rd.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			if err, ok := err.(*csv.ParseError); ok && err.Err == csv.ErrFieldCount {
				s.log.Printf("skipping line %d. %v", i, err)
				continue
			} else {
				s.log.Printf("error reading CSV '%s'", err)
				return err
			}
		}

		lnCols := len(cols)
		if lnCols < lnHdr {
			s.log.Printf("skipping line %d. column count (%d) does not match minimum header count (%d)", i, lnCols, lnHdr)
			continue
		}

		// Iterate the key map and based on the indices mapped earlier,
		// form a map of key: csv_value, eg: email: user@user.com.
		row := make(map[string]string, lnCols)
		for key := range hdrKeys {
			row[key] = cols[hdrKeys[key]]
		}

		sub := SubReq{}
		// Lowercase to ensure uniqueness in the DB.
		sub.Email = strings.ToLower(strings.TrimSpace(row["email"]))
		sub.Name = row["name"]
		if err := ValidateFields(sub); err != nil {
			s.log.Printf("skipping line %d: %v", i, err)
			continue
		}

		// JSON attributes.
		if len(row["attributes"]) > 0 {
			var (
				attribs models.SubscriberAttribs
				b       = []byte(row["attributes"])
			)
			if err := json.Unmarshal(b, &attribs); err != nil {
				s.log.Printf("skipping invalid attributes JSON on line %d for '%s': %v", i, sub.Email, err)
			} else {
				sub.Attribs = attribs
			}
		}

		// Send the subscriber to the queue.
		s.subQueue <- sub
		cur++
		if cur%commitBatchSize == 0 {
			time.Sleep(100 * time.Millisecond)
			cur = 0
		}
	}

	close(s.subQueue)
	failed = false
	return nil
}

// Stop sends a signal to stop the existing import.
func (im *Importer) Stop() {
	if im.getStatus() != StatusImporting {
		im.Lock()
		im.status = Status{Status: StatusNone}
		im.Unlock()
		return
	}

	select {
	case im.stop <- true:
		im.setStatus(StatusStopping)
	default:
	}
}

// mapCSVHeaders takes a list of headers obtained from a CSV file, a map of known headers,
// and returns a new map with each of the headers in the known map mapped by the position (0-n)
// in the given CSV list.
func (s *Session) mapCSVHeaders(csvHdrs []string, knownHdrs map[string]bool) map[string]int {
	// Map 0-n column index to the header keys, name: 0, email: 1 etc.
	// This is to allow dynamic ordering of columns in th CSV.
	hdrKeys := make(map[string]int)
	for i, h := range csvHdrs {
		// Clean the string of non-ASCII characters (BOM etc.).
		h := regexCleanStr.ReplaceAllString(h, "")
		if _, ok := knownHdrs[h]; !ok {
			s.log.Printf("ignoring unknown header '%s'", h)
			continue
		}
		hdrKeys[h] = i
	}

	return hdrKeys
}

// ValidateFields validates incoming subscriber field values.
func ValidateFields(s SubReq) error {
	if len(s.Email) > 1000 {
		return errors.New(`e-mail too long`)
	}
	if !IsEmail(s.Email) {
		return errors.New(`invalid e-mail "` + s.Email + `"`)
	}
	if len(s.Name) == 0 || len(s.Name) > stdInputMaxLen {
		return errors.New(`invalid or empty name "` + s.Name + `"`)
	}
	return nil
}

func ValidateEmail(email string) error {
	if emailAbuseRegexp.MatchString(email) || emailSpamRegexp.MatchString(email) || !emailRegexp.MatchString(email) {
		return errors.New(`invalid e-mail "` + email + `"`)
	}
	return nil
}

func ValidateSmartEmail(email string) string {
	if emailYahooRegexp.MatchString(email) {
		return "SMART-YAHOO"
	} else if emailHotmailRegexp.MatchString(email) {
		return "SMART-HOTMAIL"
	} else if emailGmailRegexp.MatchString(email) {
		return "SMART-GMAIL"
	} else if emailAolRegexp.MatchString(email) {
		return "SMART-AOL"
	}
	return ""
}

// countLines counts the number of line breaks in a file. This does not
// distinguish between "blank" and non "blank" lines.
// Credit: https://stackoverflow.com/a/24563853
func countLines(r io.Reader) (int, error) {
	var (
		buf     = make([]byte, 32*1024)
		count   = 0
		lineSep = []byte{'\n'}
	)

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

// IsEmail checks whether the given string is a valid e-mail address.
func IsEmail(email string) bool {
	// Since `mail.ParseAddress` parses an email address which can also contain optional name component
	// here we check if incoming email string is same as the parsed email.Address. So this eliminates
	// any valid email address with name and also valid address with empty name like `<abc@example.com>`.
	if em, err := mail.ParseAddress(email); err != nil || em.Address != email {
		return false
	}
	return true
}
