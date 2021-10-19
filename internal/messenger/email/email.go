package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"net/smtp"
	"net/textproto"
	"sync"
	"time"

	"github.com/knadh/listmonk/internal/messenger"
	"github.com/knadh/smtppool"
)

const emName = "smtp_email"

// Server represents an SMTP server's credentials.
type Server struct {
	Username      string            `json:"username"`
	Password      string            `json:"password"`
	AuthProtocol  string            `json:"auth_protocol"`
	TLSEnabled    bool              `json:"tls_enabled"`
	TLSSkipVerify bool              `json:"tls_skip_verify"`
	EmailHeaders  map[string]string `json:"email_headers"`

	// Rest of the options are embedded directly from the smtppool lib.
	// The JSON tag is for config unmarshal to work.
	smtppool.Opt `json:",squash"`

	pool *smtppool.Pool
}

// Emailer is the SMTP e-mail messenger.
type Emailer struct {
	servers []*Server
	mu      sync.Mutex
	v       int
	lo      *log.Logger
	name    string
}

// New returns an SMTP e-mail Messenger backend with a the given SMTP servers.
func New(lo *log.Logger, name string, servers ...Server) (*Emailer, error) {
	e := &Emailer{
		servers: make([]*Server, 0, len(servers)),
		lo:      lo,
		name:    name,
	}

	for _, srv := range servers {
		s := srv
		var auth smtp.Auth
		switch s.AuthProtocol {
		case "cram":
			auth = smtp.CRAMMD5Auth(s.Username, s.Password)
		case "plain":
			auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
		case "login":
			auth = &smtppool.LoginAuth{Username: s.Username, Password: s.Password}
		case "", "none":
		default:
			return nil, fmt.Errorf("unknown SMTP auth type '%s'", s.AuthProtocol)
		}
		s.Opt.Auth = auth

		// TLS config.
		if s.TLSEnabled {
			s.TLSConfig = &tls.Config{}
			if s.TLSSkipVerify {
				s.TLSConfig.InsecureSkipVerify = s.TLSSkipVerify
			} else {
				s.TLSConfig.ServerName = s.Host
			}
		}

		pool, err := smtppool.New(s.Opt)
		if err != nil {
			return nil, err
		}

		s.pool = pool
		e.servers = append(e.servers, &s)
	}

	return e, nil
}

// Name returns the Server's name.
func (e *Emailer) Name() string {
	if len(e.name) > 0 {
		return e.name
	}
	return emName
}

// Push pushes a message to the server.
func (e *Emailer) Push(m messenger.Message, threshold int) error {
	// If there are more than one SMTP servers, send to a random
	// one from the list.
	var (
		ln  = len(e.servers)
		srv *Server
	)
	if ln > 1 {
		srv = e.servers[rand.Intn(ln)]
	} else {
		srv = e.servers[0]
	}

	// Are there attachments?
	var files []smtppool.Attachment
	if m.Attachments != nil {
		files = make([]smtppool.Attachment, 0, len(m.Attachments))
		for _, f := range m.Attachments {
			a := smtppool.Attachment{
				Filename: f.Name,
				Header:   f.Header,
				Content:  make([]byte, len(f.Content)),
			}
			copy(a.Content, f.Content)
			files = append(files, a)
		}
	}

	em := smtppool.Email{
		From:        m.From,
		To:          m.To,
		Subject:     m.Subject,
		Attachments: files,
	}

	em.Headers = textproto.MIMEHeader{}
	// Attach e-mail level headers.
	if len(m.Headers) > 0 {
		em.Headers = m.Headers
	}

	// Attach SMTP level headers.
	if len(srv.EmailHeaders) > 0 {
		for k, v := range srv.EmailHeaders {
			em.Headers.Set(k, v)
		}
	}

	switch m.ContentType {
	case "plain":
		em.Text = []byte(m.Body)
	default:
		em.HTML = m.Body
		if len(m.AltBody) > 0 {
			em.Text = m.AltBody
		}
	}

	e.Inc(threshold)

	return srv.pool.Send(em)
}

func (e *Emailer) Inc(threshold int) {
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
func (e *Emailer) Flush() error {
	return nil
}

// Close closes the SMTP pools.
func (e *Emailer) Close() error {
	for _, s := range e.servers {
		s.pool.Close()
	}
	return nil
}
