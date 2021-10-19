package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"fmt"
	"github.com/mitchellh/mapstructure"
	mathRand "math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

var (
	tagRegexpSpaces = regexp.MustCompile(`[\s]+`)
	format          = "02_Jan_2006"
	Time            = "15:04:05"
)

// inArray checks if a string is present in a list of strings.
func inArray(val string, vals []string) (ok bool) {
	for _, v := range vals {
		if v == val {
			return true
		}
	}
	return false
}

// validateMIME is a helper function to validate uploaded file's MIME type
// against the slice of MIME types is given.
func validateMIME(typ string, mimes []string) (ok bool) {
	if len(mimes) > 0 {
		var (
			ok = false
		)
		for _, m := range mimes {
			if typ == m {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// generateFileName appends the incoming file's name with a small random hash.
func generateFileName(fName string) string {
	name := strings.TrimSpace(fName)
	if name == "" {
		name, _ = generateRandomString(10)
	}
	return name
}

// Given an error, pqErrMsg will try to return pq error details
// if it's a pq error.
func pqErrMsg(err error) string {
	if err, ok := err.(*pq.Error); ok {
		if err.Detail != "" {
			return fmt.Sprintf("%s. %s", err, err.Detail)
		}
	}
	return err.Error()
}

// normalizeTags takes a list of string tags and normalizes them by
// lowercasing and removing all special characters except for dashes.
func normalizeTags(tags []string) []string {
	var (
		out  []string
		dash = []byte("-")
	)

	for _, t := range tags {
		rep := tagRegexpSpaces.ReplaceAll(bytes.TrimSpace([]byte(t)), dash)

		if len(rep) > 0 {
			out = append(out, string(rep))
		}
	}
	return out
}

// makeMsgTpl takes a page title, heading, and message and returns
// a msgTpl that can be rendered as a HTML view. This is used for
// rendering arbitrary HTML views with error and success messages.
func makeMsgTpl(pageTitle, heading, msg string) msgTpl {
	if heading == "" {
		heading = pageTitle
	}
	err := msgTpl{}
	err.Title = pageTitle
	err.MessageTitle = heading
	err.Message = msg
	return err
}

// parseStringIDs takes a slice of numeric string IDs and
// parses each number into an int64 and returns a slice of the
// resultant values.
func parseStringIDs(s []string) ([]int64, error) {
	vals := make([]int64, 0, len(s))
	for _, v := range s {
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}

		if i < 1 {
			return nil, fmt.Errorf("%d is not a valid ID", i)
		}

		vals = append(vals, i)
	}

	return vals, nil
}

// generateRandomString generates a cryptographically random, alphanumeric string of length n.
func generateRandomString(n int) (string, error) {
	const dictionary = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}

	return string(bytes), nil
}

// strHasLen checks if the given string has a length within min-max.
func strHasLen(str string, min, max int) bool {
	return len(str) >= min && len(str) <= max
}

// strSliceContains checks if a string is present in the string slice.
func strSliceContains(str string, sl []string) bool {
	for _, s := range sl {
		if s == str {
			return true
		}
	}

	return false
}

func mergeRow(rows *sql.Rows, dst interface{}) {
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		fmt.Println(err.Error())
	}
	// Get column type
	columns_type, err := rows.ColumnTypes()
	if err != nil {
		fmt.Println(err.Error())
	}

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// get RawBytes from data
	err = rows.Scan(scanArgs...)
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}

	// Now do something with the data.
	// Here we just print each column as a string.
	m := make(map[string]interface{})
	//var value string
	for i, col := range values {
		// Here we can check if the value is nil (NULL value)
		if col != nil {
			if strings.ToLower(columns_type[i].DatabaseTypeName()) == "varchar" {
				m[columns[i]] = string(col)
			} else if strings.ToLower(columns_type[i].DatabaseTypeName()) == "decimal" {
				s, _ := strconv.ParseFloat(string(col), 64)
				m[columns[i]] = s
			} else if strings.ToLower(columns_type[i].DatabaseTypeName()) == "int" {
				m[columns[i]], err = strconv.Atoi(string(col))
			} else if strings.ToLower(columns_type[i].DatabaseTypeName()) == "timestamp" ||
				strings.ToLower(columns_type[i].DatabaseTypeName()) == "datetime" {
				if len(string(col)) > 0 {
					temp, _ := time.Parse("2006-01-02T15:04:05Z", string(col))
					m[columns[i]] = temp.Format("2006-01-02 15:04:05")
				}
			} else if strings.ToLower(columns_type[i].DatabaseTypeName()) == "text" {
				m[columns[i]] = string(col)
			} else {
				m[columns[i]], err = strconv.Atoi(string(col))
				if err != nil {
					m[columns[i]] = string(col)
				}
			}
		}
	}

	config := &mapstructure.DecoderConfig{
		TagName: "json",
		Result:  &dst,
	}
	decoder, _ := mapstructure.NewDecoder(config)
	err = decoder.Decode(m)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func defaultTicker(timeRunning string) *time.Ticker {
	nextTick, err := time.ParseInLocation(format+Time, time.Now().Format(format)+timeRunning, time.Local)
	if err != nil {
		panic("invalid time format")
	}
	if !nextTick.After(time.Now()) {
		nextTick = nextTick.Add(24 * time.Hour)
	}
	diff := nextTick.Sub(time.Now())
	return time.NewTicker(diff)
}

func getRandomTimeScheduler() string {
	mathRand.Seed(time.Now().UnixNano())
	min := 1
	max := 59
	return time.Now().Add(time.Hour*time.Duration(1) +
		time.Minute*time.Duration(mathRand.Intn(max-min+1)+min) +
		time.Second*time.Duration(mathRand.Intn(max-min+1)+min)).Format("15:04:05")
}
