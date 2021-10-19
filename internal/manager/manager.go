package manager

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/textproto"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/messenger"
	"github.com/knadh/listmonk/models"
)

const (
	// BaseTPL is the name of the base template.
	BaseTPL = "base"

	// ContentTpl is the name of the compiled message.
	ContentTpl = "content"

	dummyUUID = "00000000-0000-0000-0000-000000000000"
)

// DataSource represents a data backend, such as a database,
// that provides subscriber and campaign records.
type DataSource interface {
	NextCampaigns(excludeIDs []int64) ([]*models.Campaign, error)
	NextSubscribers(campID, limit int) ([]models.Subscriber, error)
	GetCampaign(campID int) (*models.Campaign, error)
	UpdateCampaignStatus(campID int, status string) error
	CreateLink(url string) (string, error)
	UpdateLastEmailSent(email string) error
	UpdateSentCampaign(campID, limit, lastSubsId int) error
}

// Manager handles the scheduling, processing, and queuing of campaigns
// and message pushes.
type Manager struct {
	Cfg        Config
	src        DataSource
	i18n       *i18n.I18n
	messengers map[string]messenger.Messenger
	notifCB    models.AdminNotifCallback
	logger     *log.Logger

	// Campaigns that are currently running.
	camps    map[int]*models.Campaign
	campsMut sync.RWMutex

	// Links generated using Track() are cached here so as to not query
	// the database for the link UUID for every message sent. This has to
	// be locked as it may be used externally when previewing campaigns.
	links    map[string]string
	linksMut sync.RWMutex

	subFetchQueue      chan *models.Campaign
	campMsgQueue       chan CampaignMessage
	campMsgErrorQueue  chan msgError
	campMsgErrorCounts map[int]int
	msgQueue           chan Message

	// Sliding window keeps track of the total number of messages sent in a period
	// and on reaching the specified limit, waits until the window is over before
	// sending further messages.
	slidingWindowNumMsg int
	slidingWindowStart  time.Time
}

// CampaignMessage represents an instance of campaign message to be pushed out,
// specific to a subscriber, via the campaign's messenger.
type CampaignMessage struct {
	Campaign   *models.Campaign
	Subscriber models.Subscriber

	from     string
	to       string
	subject  string
	body     []byte
	altBody  []byte
	unsubURL string
}

// Message represents a generic message to be pushed to a messenger.
type Message struct {
	messenger.Message
	Subscriber models.Subscriber

	// Messenger is the messenger backend to use: email|postback.
	Messenger string
}

// Config has parameters for configuring the manager.
type Config struct {
	// Number of subscribers to pull from the DB in a single iteration.
	BatchSize int

	Concurrency           int
	MessageRate           int
	MaxSendErrors         int
	SlidingWindow         bool
	SlidingWindowDuration time.Duration
	SlidingWindowRate     int
	RequeueOnError        bool
	FromEmail             string
	IndividualTracking    bool
	LinkTrackURL          string
	UnsubURL              string
	OptinURL              string
	MessageURL            string
	ViewTrackURL          string
	UnsubHeader           bool
}

type msgError struct {
	camp *models.Campaign
	err  error
}

// New returns a new instance of Mailer.
func New(cfg Config, src DataSource, notifCB models.AdminNotifCallback, i *i18n.I18n, l *log.Logger) *Manager {
	if cfg.BatchSize < 1 {
		cfg.BatchSize = 1000
	}
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	if cfg.MessageRate < 1 {
		cfg.MessageRate = 1
	}

	return &Manager{
		Cfg:                cfg,
		src:                src,
		i18n:               i,
		notifCB:            notifCB,
		logger:             l,
		messengers:         make(map[string]messenger.Messenger),
		camps:              make(map[int]*models.Campaign),
		links:              make(map[string]string),
		subFetchQueue:      make(chan *models.Campaign, cfg.Concurrency),
		campMsgQueue:       make(chan CampaignMessage, cfg.Concurrency*2),
		msgQueue:           make(chan Message, cfg.Concurrency),
		campMsgErrorQueue:  make(chan msgError, cfg.MaxSendErrors),
		campMsgErrorCounts: make(map[int]int),
		slidingWindowStart: time.Now(),
	}
}

// NewCampaignMessage creates and returns a CampaignMessage that is made available
// to message templates while they're compiled. It represents a message from
// a campaign that's bound to a single Subscriber.
func (m *Manager) NewCampaignMessage(c *models.Campaign, s models.Subscriber) (CampaignMessage, error) {
	msg := CampaignMessage{
		Campaign:   c,
		Subscriber: s,

		subject:  c.Subject,
		from:     c.FromEmail,
		to:       s.Email,
		unsubURL: fmt.Sprintf(m.Cfg.UnsubURL, c.UUID, s.UUID),
	}

	if err := msg.render(); err != nil {
		return msg, err
	}

	return msg, nil
}

// AddMessenger adds a Messenger messaging backend to the manager.
func (m *Manager) AddMessenger(msg messenger.Messenger) error {
	id := msg.Name()
	if _, ok := m.messengers[id]; ok {
		return fmt.Errorf("messenger '%s' is already loaded", id)
	}
	m.messengers[id] = msg
	return nil
}

// PushMessage pushes an arbitrary non-campaign Message to be sent out by the workers.
// It times out if the queue is busy.
func (m *Manager) PushMessage(msg Message) error {
	t := time.NewTicker(time.Second * 3)
	defer t.Stop()

	select {
	case m.msgQueue <- msg:
	case <-t.C:
		m.logger.Printf("message push timed out: '%s'", msg.Subject)
		return errors.New("message push timed out")
	}
	return nil
}

// PushCampaignMessage pushes a campaign messages to be sent out by the workers.
// It times out if the queue is busy.
func (m *Manager) PushCampaignMessage(msg CampaignMessage) error {
	t := time.NewTicker(time.Second * 3)
	defer t.Stop()

	select {
	case m.campMsgQueue <- msg:
	case <-t.C:
		m.logger.Printf("message push timed out: '%s'", msg.Subject())
		return errors.New("message push timed out")
	}
	return nil
}

// HasMessenger checks if a given messenger is registered.
func (m *Manager) HasMessenger(id string) bool {
	_, ok := m.messengers[id]
	return ok
}

// HasRunningCampaigns checks if there are any active campaigns.
func (m *Manager) HasRunningCampaigns() bool {
	m.campsMut.Lock()
	defer m.campsMut.Unlock()
	return len(m.camps) > 0
}

// Run is a blocking function (that should be invoked as a goroutine)
// that scans the data source at regular intervals for pending campaigns,
// and queues them for processing. The process queue fetches batches of
// subscribers and pushes messages to them for each queued campaign
// until all subscribers are exhausted, at which point, a campaign is marked
// as "finished".
func (m *Manager) Run(tick time.Duration) {
	go m.scanCampaigns(tick)

	// Spawn N message workers.
	for i := 0; i < m.Cfg.Concurrency; i++ {
		go m.messageWorker(i)
	}

	// Fetch the next set of subscribers for a campaign and process them.
	for c := range m.subFetchQueue {
		has, err := m.nextSubscribers(c, m.Cfg.BatchSize)
		if err != nil {
			m.logger.Printf("error processing campaign batch (%s): %v", c.Name, err)
			continue
		}

		if has {
			// There are more subscribers to fetch.
			m.subFetchQueue <- c
		} else if m.isCampaignProcessing(c.ID) {
			// There are no more subscribers. Either the campaign status
			// has changed or all subscribers have been processed.
			newC, err := m.exhaustCampaign(c, "")
			if err != nil {
				m.logger.Printf("error exhausting campaign (%s): %v", c.Name, err)
				continue
			}
			m.sendNotif(newC, newC.Status, "")
		}
	}
}

// messageWorker is a blocking function that listens to the message queue
// and pushes out incoming messages on it to the messenger.
func (m *Manager) messageWorker(worker int) {
	// Counter to keep track of the message / sec rate limit.
	//rl := ratelimit.New(m.Cfg.MessageRate)
	//start := time.Now()
	for {
		select {
		// Campaign message.
		case msg, ok := <-m.campMsgQueue:
			if !ok {
				return
			}
			//now := rl.Take()

			// Outgoing message.
			out := messenger.Message{
				From:        msg.from,
				To:          []string{msg.to},
				Subject:     msg.subject,
				ContentType: msg.Campaign.ContentType,
				Body:        msg.body,
				AltBody:     msg.altBody,
				Subscriber:  msg.Subscriber,
				Campaign:    msg.Campaign,
			}

			// Attach List-Unsubscribe headers?
			if m.Cfg.UnsubHeader {
				h := textproto.MIMEHeader{}
				h.Set("List-Unsubscribe-Post", "List-Unsubscribe=One-Click")
				h.Set("List-Unsubscribe", `<`+msg.unsubURL+`>`)
				out.Headers = h
			}

			if err := m.messengers[msg.Campaign.Messenger].Push(out, m.Cfg.Concurrency); err != nil {
				m.logger.Printf("error sending message in campaign %s: subscriber %s: %v",
					msg.Campaign.Name, msg.Subscriber.UUID, err)

				select {
				case m.campMsgErrorQueue <- msgError{camp: msg.Campaign, err: err}:
				default:
				}
			}
			//m.logger.Printf("[%v] sent to %v at worker %v, taking at %v - %v", msg.Campaign.Messenger, out.To[0], worker, now, now.Sub(start))
			//start = now

			go func(email string) {
				if err := m.src.UpdateLastEmailSent(email); err != nil {
					m.logger.Printf("error updating last email sent (%s) : %v", out.To[0], err)
				}
			}(out.To[0])

		// Arbitrary message.
		case msg, ok := <-m.msgQueue:
			if !ok {
				return
			}

			err := m.messengers[msg.Messenger].Push(messenger.Message{
				From:        msg.From,
				To:          msg.To,
				Subject:     msg.Subject,
				ContentType: msg.ContentType,
				Body:        msg.Body,
				AltBody:     msg.AltBody,
				Subscriber:  msg.Subscriber,
				Campaign:    msg.Campaign,
			}, m.Cfg.Concurrency)
			if err != nil {
				m.logger.Printf("error sending message '%s': %v", msg.Subject, err)
			}
		}
	}
}

// TemplateFuncs returns the template functions to be applied into
// compiled campaign templates.
func (m *Manager) TemplateFuncs(c *models.Campaign) template.FuncMap {
	f := template.FuncMap{
		"TrackLink": func(url string, baseUrl string, msg *CampaignMessage) string {
			subUUID := msg.Subscriber.UUID
			if !m.Cfg.IndividualTracking {
				subUUID = dummyUUID
			}

			urlTemplate := m.Cfg.LinkTrackURL
			if len(strings.TrimSpace(baseUrl)) > 0 {
				urlTemplate = fmt.Sprintf("%s/link/%%s/%%s/%%s", baseUrl)
			}

			return m.trackLink(urlTemplate, url, msg.Campaign.UUID, subUUID)
		},
		"TrackView": func(url string, msg *CampaignMessage) template.HTML {
			subUUID := msg.Subscriber.UUID
			if !m.Cfg.IndividualTracking {
				subUUID = dummyUUID
			}

			urlTemplate := m.Cfg.ViewTrackURL
			if len(strings.TrimSpace(url)) > 0 {
				urlTemplate = strings.ReplaceAll(fmt.Sprintf("%s/campaign/%%s/%%s/px.png", strings.TrimSpace(url)), "//campaign", "/campaign")
			}

			return template.HTML(fmt.Sprintf(`<img src="%s" alt="" />`,
				fmt.Sprintf(urlTemplate, msg.Campaign.UUID, subUUID)))
		},
		"UnsubscribeURL": func(msg *CampaignMessage) string {
			return msg.unsubURL
		},
		"OptinURL": func(msg *CampaignMessage) string {
			// Add list IDs.
			// TODO: Show private lists list on optin e-mail
			return fmt.Sprintf(m.Cfg.OptinURL, msg.Subscriber.UUID, "")
		},
		"MessageURL": func(msg *CampaignMessage) string {
			return fmt.Sprintf(m.Cfg.MessageURL, c.UUID, msg.Subscriber.UUID)
		},
		"Date": func(layout string) string {
			if layout == "" {
				layout = time.ANSIC
			}
			return time.Now().Format(layout)
		},
		"L": func() *i18n.I18n {
			return m.i18n
		},
		"Safe": func(safeHTML string) template.HTML {
			return template.HTML(safeHTML)
		},
	}
	for k, v := range sprig.GenericFuncMap() {
		f[k] = v
	}
	return f
}

// Close closes and exits the campaign manager.
func (m *Manager) Close() {
	close(m.subFetchQueue)
	close(m.campMsgErrorQueue)
	close(m.msgQueue)
}

// scanCampaigns is a blocking function that periodically scans the data source
// for campaigns to process and dispatches them to the manager.
func (m *Manager) scanCampaigns(tick time.Duration) {
	t := time.NewTicker(tick)
	defer t.Stop()

	for {
		select {
		// Periodically scan the data source for campaigns to process.
		case <-t.C:
			campaigns, err := m.src.NextCampaigns(m.getPendingCampaignIDs())
			if err != nil {
				m.logger.Printf("error fetching campaigns: %v", err)
				continue
			}

			for _, c := range campaigns {
				if err := m.addCampaign(c); err != nil {
					m.logger.Printf("error processing campaign (%s): %v", c.Name, err)
					continue
				}
				m.logger.Printf("start processing campaign (%s)", c.Name)

				// If subscriber processing is busy, move on. Blocking and waiting
				// can end up in a race condition where the waiting campaign's
				// state in the data source has changed.
				select {
				case m.subFetchQueue <- c:
				default:
				}
			}

			// Aggregate errors from sending messages to check against the error threshold
			// after which a campaign is paused.
		case e, ok := <-m.campMsgErrorQueue:
			if !ok {
				return
			}
			if m.Cfg.MaxSendErrors < 1 {
				continue
			}

			// If the error threshold is met, pause the campaign.
			m.campMsgErrorCounts[e.camp.ID]++
			if m.campMsgErrorCounts[e.camp.ID] >= m.Cfg.MaxSendErrors {
				m.logger.Printf("error counted exceeded %d. pausing campaign %s",
					m.Cfg.MaxSendErrors, e.camp.Name)

				if m.isCampaignProcessing(e.camp.ID) {
					m.exhaustCampaign(e.camp, models.CampaignStatusPaused)
				}
				delete(m.campMsgErrorCounts, e.camp.ID)

				// Notify admins.
				m.sendNotif(e.camp, models.CampaignStatusPaused, "Too many errors")
			}
		}
	}
}

// addCampaign adds a campaign to the process queue.
func (m *Manager) addCampaign(c *models.Campaign) error {
	// Validate messenger.
	if _, ok := m.messengers[c.Messenger]; !ok {
		m.src.UpdateCampaignStatus(c.ID, models.CampaignStatusCancelled)
		return fmt.Errorf("unknown messenger %s on campaign %s", c.Messenger, c.Name)
	}

	// Load the template.
	if err := c.CompileTemplate(m.TemplateFuncs(c)); err != nil {
		return err
	}

	// Add the campaign to the active map.
	m.campsMut.Lock()
	m.camps[c.ID] = c
	m.campsMut.Unlock()
	return nil
}

// getPendingCampaignIDs returns the IDs of campaigns currently being processed.
func (m *Manager) getPendingCampaignIDs() []int64 {
	// Needs to return an empty slice in case there are no campaigns.
	m.campsMut.RLock()
	ids := make([]int64, 0, len(m.camps))
	for _, c := range m.camps {
		ids = append(ids, int64(c.ID))
	}
	m.campsMut.RUnlock()
	return ids
}

// nextSubscribers processes the next batch of subscribers in a given campaign.
// It returns a bool indicating whether any subscribers were processed
// in the current batch or not. A false indicates that all subscribers
// have been processed, or that a campaign has been paused or cancelled.
func (m *Manager) nextSubscribers(c *models.Campaign, batchSize int) (bool, error) {
	// Fetch a batch of subscribers.
	if len(c.SubsCamp) == 0 {
		subs, err := m.src.NextSubscribers(c.ID, batchSize)
		if err != nil {
			return false, fmt.Errorf("error fetching campaign subscribers (%s): %v", c.Name, err)
		}
		for _, eachSub := range subs {
			c.SubsCamp = append(c.SubsCamp, eachSub)
		}
	}

	// There are no subscribers.
	if len(c.SubsCamp) == 0 {
		return false, nil
	}

	// Is there a sliding window limit configured?
	hasSliding := m.Cfg.SlidingWindow &&
		m.Cfg.SlidingWindowRate > 0 &&
		m.Cfg.SlidingWindowDuration.Seconds() > 1

	// Push messages.
	sleep := 0
	tmp := c.SubsCamp[:0]
	lastSubID := 0

	for _, s := range c.SubsCamp {
		if sleep >= batchSize {
			tmp = append(tmp, s)
			sleep++
			continue
		}
		// Send the message.
		msg, err := m.NewCampaignMessage(c, s)
		if err != nil {
			m.logger.Printf("error rendering message (%s) (%s): %v", c.Name, s.Email, err)
			continue
		}

		// Push the message to the queue while blocking and waiting until
		// the queue is drained.
		m.campMsgQueue <- msg

		// Check if the sliding window is active.
		if hasSliding {
			diff := time.Now().Sub(m.slidingWindowStart)

			// Window has expired. Reset the clock.
			if diff >= m.Cfg.SlidingWindowDuration {
				m.slidingWindowStart = time.Now()
				m.slidingWindowNumMsg = 0
				continue
			}

			// Have the messages exceeded the limit?
			m.slidingWindowNumMsg++
			if m.slidingWindowNumMsg >= m.Cfg.SlidingWindowRate {
				wait := m.Cfg.SlidingWindowDuration - diff

				m.logger.Printf("messages exceeded (%d) for the window (%v since %s). Sleeping for %s.",
					m.slidingWindowNumMsg,
					m.Cfg.SlidingWindowDuration,
					m.slidingWindowStart.Format(time.RFC822Z),
					wait.Round(time.Second)*1)

				m.slidingWindowNumMsg = 0
				time.Sleep(wait)
			}
		}
		sleep++
		lastSubID = s.ID
	}

	count := batchSize
	if batchSize > len(c.SubsCamp) {
		count = len(c.SubsCamp)
	}

	go m.src.UpdateSentCampaign(c.ID, count, lastSubID)

	if batchSize >= len(c.SubsCamp) {
		time.Sleep(5 * time.Second)
	}

	c.SubsCamp = tmp

	return true, nil
}

// isCampaignProcessing checks if the campaign is bing processed.
func (m *Manager) isCampaignProcessing(id int) bool {
	m.campsMut.RLock()
	_, ok := m.camps[id]
	m.campsMut.RUnlock()
	return ok
}

func (m *Manager) exhaustCampaign(c *models.Campaign, status string) (*models.Campaign, error) {
	m.campsMut.Lock()
	delete(m.camps, c.ID)
	m.campsMut.Unlock()

	// A status has been passed. Change the campaign's status
	// without further checks.
	if status != "" {
		if err := m.src.UpdateCampaignStatus(c.ID, status); err != nil {
			m.logger.Printf("error updating campaign (%s) status to %s: %v", c.Name, status, err)
		} else {
			m.logger.Printf("set campaign (%s) to %s", c.Name, status)
		}
		return c, nil
	}

	// Fetch the up-to-date campaign status from the source.
	cm, err := m.src.GetCampaign(c.ID)
	if err != nil {
		return nil, err
	}

	// If a running campaign has exhausted subscribers, it's finished.
	if cm.Status == models.CampaignStatusRunning {
		cm.Status = models.CampaignStatusFinished
		if err := m.src.UpdateCampaignStatus(c.ID, models.CampaignStatusFinished); err != nil {
			m.logger.Printf("error finishing campaign (%s): %v", c.Name, err)
		} else {
			m.logger.Printf("campaign (%s) finished", c.Name)
		}
	} else {
		m.logger.Printf("stop processing campaign (%s)", c.Name)
	}

	return cm, nil
}

// trackLink register a URL and return its UUID to be used in message templates
// for tracking links.
func (m *Manager) trackLink(urlTemplate, url, campUUID, subUUID string) string {
	m.linksMut.RLock()
	if uu, ok := m.links[url]; ok {
		m.linksMut.RUnlock()
		return fmt.Sprintf(urlTemplate, uu, campUUID, subUUID)
	}
	m.linksMut.RUnlock()

	// Register link.
	uu, err := m.src.CreateLink(url)
	if err != nil {
		m.logger.Printf("error registering tracking for link '%s': %v", url, err)

		// If the registration fails, fail over to the original URL.
		return url
	}

	m.linksMut.Lock()
	m.links[url] = uu
	m.linksMut.Unlock()

	return fmt.Sprintf(urlTemplate, uu, campUUID, subUUID)
}

// sendNotif sends a notification to registered admin e-mails.
func (m *Manager) sendNotif(c *models.Campaign, status, reason string) error {
	var (
		subject = fmt.Sprintf("%s: %s", strings.Title(status), c.Name)
		data    = map[string]interface{}{
			"ID":     c.ID,
			"Name":   c.Name,
			"Status": status,
			"Sent":   c.Sent,
			"ToSend": c.ToSend,
			"Reason": reason,
		}
	)
	return m.notifCB(subject, data)
}

// render takes a Message, executes its pre-compiled Campaign.Tpl
// and applies the resultant bytes to Message.body to be used in messages.
func (m *CampaignMessage) render() error {
	out := bytes.Buffer{}

	// Render the subject if it's a template.
	if m.Campaign.SubjectTpl != nil {
		if err := m.Campaign.SubjectTpl.ExecuteTemplate(&out, models.ContentTpl, m); err != nil {
			return err
		}
		m.subject = out.String()
		out.Reset()
	}

	// Compile the main template.
	if err := m.Campaign.Tpl.ExecuteTemplate(&out, models.BaseTpl, m); err != nil {
		return err
	}
	m.body = out.Bytes()

	// Is there an alt body?
	if m.Campaign.ContentType != models.CampaignContentTypePlain && m.Campaign.AltBody.Valid {
		if m.Campaign.AltBodyTpl != nil {
			b := bytes.Buffer{}
			if err := m.Campaign.AltBodyTpl.ExecuteTemplate(&b, models.ContentTpl, m); err != nil {
				return err
			}
			m.altBody = b.Bytes()
		} else {
			m.altBody = []byte(m.Campaign.AltBody.String)
		}
	}

	return nil
}

// Subject returns a copy of the message subject
func (m *CampaignMessage) Subject() string {
	return m.subject
}

// Body returns a copy of the message body.
func (m *CampaignMessage) Body() []byte {
	out := make([]byte, len(m.body))
	copy(out, m.body)
	return out
}

// AltBody returns a copy of the message's alt body.
func (m *CampaignMessage) AltBody() []byte {
	out := make([]byte, len(m.altBody))
	copy(out, m.altBody)
	return out
}
