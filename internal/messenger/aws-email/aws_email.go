package aws_email

import (
	"bytes"
	"errors"
	"log"
	"math/rand"
	"mime/multipart"
	"net/textproto"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	AwsSes "github.com/aws/aws-sdk-go/service/ses"
	"github.com/knadh/listmonk/internal/messenger"
)

const emName = "email"

// Server represents an AWS server's credentials.

type AWSEmailer struct {
	sesClients []*AwsSes.SES
	mu         sync.Mutex
	v          int
	lo         *log.Logger
	name       string
}

type AWSConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Region   string `json:"region"`
}

func New(lo *log.Logger, name string, conf ...AWSConfig) (*AWSEmailer, error) {
	e := &AWSEmailer{
		sesClients: make([]*AwsSes.SES, 0, len(conf)),
		lo:         lo,
		name:       name,
	}

	for _, c := range conf {
		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String(c.Region),
			Credentials: credentials.NewStaticCredentials(c.Username, c.Password, ""),
		})
		if err != nil {
			return nil, err
		}

		// Create an SES session.
		client := AwsSes.New(sess)
		e.sesClients = append(e.sesClients, client)
	}

	return e, nil
}

// Name returns the Server's name.
func (e *AWSEmailer) Name() string {
	if len(e.name) > 0 {
		return e.name
	}
	return emName
}

// Push pushes a message to the server.
func (e *AWSEmailer) Push(m messenger.Message, threshold int) error {
	var (
		ln  = len(e.sesClients)
		srv *AwsSes.SES
	)
	if ln > 1 {
		srv = e.sesClients[rand.Intn(ln)]
	} else if ln == 1 {
		srv = e.sesClients[0]
	} else {
		return errors.New("no AWS emailer")
	}

	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	// email main header:
	h := textproto.MIMEHeader{}
	// Attach e-mail level headers.
	if len(m.Headers) > 0 {
		h = m.Headers
	}
	h.Set("From", m.From)
	h.Set("To", m.To[0])
	h.Set("Return-Path", m.From)
	cleanSubject := strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1
		}
		return r
	}, m.Subject)
	h.Set("Subject", cleanSubject)
	h.Set("Content-Language", "en-US")
	h.Set("Content-Type", "multipart/alternative; boundary=\""+writer.Boundary()+"\"")
	h.Set("MIME-Version", "1.0")

	_, err := writer.CreatePart(h)
	if err != nil {
		log.Println("Error: email main header - ", err)
	}

	// body:
	h = make(textproto.MIMEHeader)
	h.Set("Content-Transfer-Encoding", "quoted-printable")
	h.Set("Content-Type", "text/plain; charset=iso-8859-1")
	h.Set("MIME-Version", "1.0")
	part, err := writer.CreatePart(h)
	if err != nil {
		log.Println("Error: email createPart body - ", err)
	}
	clean := strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1
		}
		return r
	}, string(m.AltBody))
	_, err = part.Write([]byte(clean))
	if err != nil {
		log.Println("Error: email write body - ", err)
	}

	// body:
	h = make(textproto.MIMEHeader)
	h.Set("Content-Transfer-Encoding", "quoted-printable")
	h.Set("Content-Type", "text/html; charset=us-ascii")
	h.Set("MIME-Version", "1.0")
	partHtml, err := writer.CreatePart(h)
	if err != nil {
		return err
	}
	cleanBody := strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1
		}
		return r
	}, string(m.Body))
	_, err = partHtml.Write([]byte(cleanBody))
	if err != nil {
		return err
	}

	// Strip boundary line before header (doesn't work with it present)
	s := buf.String()
	if strings.Count(s, "\n") < 2 {
		return errors.New("error: invalid e-mail content")
	}
	s = strings.SplitN(s, "\n", 2)[1]

	raw := AwsSes.RawMessage{
		Data: []byte(s),
	}

	toAddresses := make([]*string, 0)
	for _, to := range m.To {
		toAddresses = append(toAddresses, aws.String(to))
	}

	inputs := &AwsSes.SendRawEmailInput{
		Destinations: toAddresses,
		Source:       aws.String(m.From),
		RawMessage:   &raw,
	}

	e.Inc(threshold)

	// Attempt to send the email.
	//defer func(begin time.Time) {
	//	e.lo.Printf("AWS Send Email Destinations: %s , took: %v", toAddresses, time.Since(begin))
	//}(time.Now())

	_, err = srv.SendRawEmail(inputs)

	return err
}

func (e *AWSEmailer) Inc(threshold int) {
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
func (e *AWSEmailer) Flush() error {
	return nil
}

// Close closes the aws emailer.
func (e *AWSEmailer) Close() error {
	return nil
}
