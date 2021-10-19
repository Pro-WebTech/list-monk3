package postback

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/knadh/listmonk/internal/messenger"
	"github.com/knadh/listmonk/models"
)

// postback is the payload that's posted as JSON to the HTTP Postback server.
//easyjson:json
type postback struct {
	Subject     string      `json:"subject"`
	ContentType string      `json:"content_type"`
	Body        string      `json:"body"`
	Recipients  []recipient `json:"recipients"`
	Campaign    *campaign   `json:"campaign"`
}

type campaign struct {
	UUID string   `db:"uuid" json:"uuid"`
	Name string   `db:"name" json:"name"`
	Tags []string `db:"tags" json:"tags"`
}

type recipient struct {
	UUID    string                   `db:"uuid" json:"uuid"`
	Email   string                   `db:"email" json:"email"`
	Name    string                   `db:"name" json:"name"`
	Attribs models.SubscriberAttribs `db:"attribs" json:"attribs"`
	Status  string                   `db:"status" json:"status"`
}

// Options represents HTTP Postback server options.
type Options struct {
	Name     string        `json:"name"`
	Username string        `json:"username"`
	Password string        `json:"password"`
	RootURL  string        `json:"root_url"`
	MaxConns int           `json:"max_conns"`
	Retries  int           `json:"retries"`
	Timeout  time.Duration `json:"timeout"`
}

// Postback represents an HTTP Message server.
type Postback struct {
	authStr string
	o       Options
	c       *http.Client
	mu      sync.Mutex
	v       int
}

// New returns a new instance of the HTTP Postback messenger.
func New(o Options) (*Postback, error) {
	authStr := ""
	if o.Username != "" && o.Password != "" {
		authStr = fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString(
			[]byte(o.Username+":"+o.Password)))
	}

	return &Postback{
		authStr: authStr,
		o:       o,
		c: &http.Client{
			Timeout: o.Timeout,
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   o.MaxConns,
				MaxConnsPerHost:       o.MaxConns,
				ResponseHeaderTimeout: o.Timeout,
				IdleConnTimeout:       o.Timeout,
			},
		},
	}, nil
}

// Name returns the messenger's name.
func (p *Postback) Name() string {
	return p.o.Name
}

// Push pushes a message to the server.
func (p *Postback) Push(m messenger.Message, threshold int) error {

	pb := postback{
		Subject:     m.Subject,
		ContentType: m.ContentType,
		Body:        string(m.Body),
		Recipients: []recipient{{
			UUID:    m.Subscriber.UUID,
			Email:   m.Subscriber.Email,
			Name:    m.Subscriber.Name,
			Status:  m.Subscriber.Status,
			Attribs: m.Subscriber.Attribs,
		}},
	}

	if m.Campaign != nil {
		pb.Campaign = &campaign{
			UUID: m.Campaign.UUID,
			Name: m.Campaign.Name,
			Tags: m.Campaign.Tags,
		}
	}

	b, err := pb.MarshalJSON()
	if err != nil {
		return err
	}

	p.Inc(threshold)

	return p.exec(http.MethodPost, p.o.RootURL, b, nil)
}

func (e *Postback) Inc(threshold int) {
	e.mu.Lock()
	// Lock so only one goroutine at a time can access the map c.v.
	if e.v >= threshold {
		time.Sleep(time.Second)
		//atomic.CompareAndSwapInt32(&numMsg, numMsg, 0)
		e.v = 0
	}
	e.v++
	e.mu.Unlock()
}

// Flush flushes the message queue to the server.
func (p *Postback) Flush() error {
	return nil
}

// Close closes idle HTTP connections.
func (p *Postback) Close() error {
	p.c.CloseIdleConnections()
	return nil
}

func (p *Postback) exec(method, rURL string, reqBody []byte, headers http.Header) error {
	var (
		err      error
		postBody io.Reader
	)

	// Encode POST / PUT params.
	if method == http.MethodPost || method == http.MethodPut {
		postBody = bytes.NewReader(reqBody)
	}

	req, err := http.NewRequest(method, rURL, postBody)
	if err != nil {
		return err
	}

	if headers != nil {
		req.Header = headers
	} else {
		req.Header = http.Header{}
	}
	req.Header.Set("User-Agent", "listmonk")

	// Optional BasicAuth.
	if p.authStr != "" {
		req.Header.Set("Authorization", p.authStr)
	}

	// If a content-type isn't set, set the default one.
	if req.Header.Get("Content-Type") == "" {
		if method == http.MethodPost || method == http.MethodPut {
			req.Header.Add("Content-Type", "application/json")
		}
	}

	// If the request method is GET or DELETE, add the params as QueryString.
	if method == http.MethodGet || method == http.MethodDelete {
		req.URL.RawQuery = string(reqBody)
	}

	r, err := p.c.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		// Drain and close the body to let the Transport reuse the connection
		io.Copy(ioutil.Discard, r.Body)
		r.Body.Close()
	}()

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("non-OK response from Postback server: %d", r.StatusCode)
	}

	return nil
}
