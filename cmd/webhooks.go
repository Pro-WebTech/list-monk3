package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo"
)

type bodyMessage struct {
	Message string `json:"Message"`
}

type message struct {
	NotificationType string `json:"notificationType"`
}

type BounceEvent struct {
	NotificationType string `json:"notificationType"`
	Bounce           struct {
		BounceType        string `json:"bounceType"`
		ReportingMTA      string `json:"reportingMTA"`
		BouncedRecipients []struct {
			EmailAddress   string `json:"emailAddress"`
			Status         string `json:"status"`
			Action         string `json:"action"`
			DiagnosticCode string `json:"diagnosticCode"`
		} `json:"bouncedRecipients"`
		BounceSubType string    `json:"bounceSubType"`
		Timestamp     time.Time `json:"timestamp"`
		FeedbackID    string    `json:"feedbackId"`
		RemoteMtaIP   string    `json:"remoteMtaIp"`
	} `json:"bounce"`
	Mail struct {
		Timestamp        time.Time `json:"timestamp"`
		Source           string    `json:"source"`
		SourceArn        string    `json:"sourceArn"`
		SourceIP         string    `json:"sourceIp"`
		SendingAccountID string    `json:"sendingAccountId"`
		MessageID        string    `json:"messageId"`
		Destination      []string  `json:"destination"`
		HeadersTruncated bool      `json:"headersTruncated"`
		Headers          []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
		CommonHeaders struct {
			From      []string `json:"from"`
			Date      string   `json:"date"`
			To        []string `json:"to"`
			MessageID string   `json:"messageId"`
			Subject   string   `json:"subject"`
		} `json:"commonHeaders"`
	} `json:"mail"`
}

type ComplaintEvent struct {
	NotificationType string `json:"notificationType"`
	Complaint        struct {
		UserAgent            string `json:"userAgent"`
		ComplainedRecipients []struct {
			EmailAddress string `json:"emailAddress"`
		} `json:"complainedRecipients"`
		ComplaintFeedbackType string    `json:"complaintFeedbackType"`
		ArrivalDate           time.Time `json:"arrivalDate"`
		Timestamp             time.Time `json:"timestamp"`
		FeedbackID            string    `json:"feedbackId"`
	} `json:"complaint"`
	Mail struct {
		Timestamp        time.Time `json:"timestamp"`
		MessageID        string    `json:"messageId"`
		Source           string    `json:"source"`
		SourceArn        string    `json:"sourceArn"`
		SourceIP         string    `json:"sourceIp"`
		SendingAccountID string    `json:"sendingAccountId"`
		Destination      []string  `json:"destination"`
		HeadersTruncated bool      `json:"headersTruncated"`
		Headers          []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
		CommonHeaders struct {
			From      []string `json:"from"`
			Date      string   `json:"date"`
			To        []string `json:"to"`
			MessageID string   `json:"messageId"`
			Subject   string   `json:"subject"`
		} `json:"commonHeaders"`
	} `json:"mail"`
}

type mailparserReq struct {
	SenderFieldAddress     string `json:"sender_field_address"`
	HardBounceParseAddress string `json:"hard_bounce_parse_address"`
	ReceivedAt             string `json:"received_at"`
	Id                     string `json:"id"`
}

var layoutISO = "2006-01-02 15:04:05"

const (
	TypeBounce    = "Bounced"
	TypeComplaint = "Complained"
)

type SubQueryReq struct {
	Query          string    `json:"query"`
	Email          string    `json:"email"`
	SubscriberIDs  int64     `json:"subscriberId"`
	EventType      string    `json:"eventType"`
	EventReason    string    `json:"eventReason"`
	EventTimeStamp time.Time `json:"eventTimeStamp"`
}

type EmailAttribute struct {
	EventType      string    `json:"eventType"`
	EventReason    string    `json:"eventReason"`
	EventTimeStamp time.Time `json:"eventTimeStamp"`
}

const (
	postmarkappHardBounce          = "hardbounce"
	postmarkappBadEmail            = "bademailaddress"
	postmarkappSpamComplaint       = "spamcomplaint"
	postmarkappSpamNotification    = "spamnotification"
	postmarkappManuallyDeactivated = "manuallydeactivated"
)

type postmarkappReq struct {
	ID          int64     `json:"ID"`
	Email       string    `json:"Email"`
	From        string    `json:"From"`
	Type        string    `json:"Type"`
	TypeCode    int64     `json:"TypeCode"`
	RecordType  string    `json:"RecordType"`
	Description string    `json:"Description"`
	Details     string    `json:"Details"`
	BouncedAt   time.Time `json:"BouncedAt"`
}

func amazonSubscriptionHandler(r *http.Request, c echo.Context) error {
	requestData := make(map[string]interface{})
	app := c.Get("app").(*App)

	// check header value
	if r.Header.Get("x-amz-sns-message-type") == "SubscriptionConfirmation" {
		// read request body
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			app.log.Println("Error! unable to read request body[amazonSubscriptionHandler]:", err)
			return err
		}
		defer r.Body.Close()

		err = json.Unmarshal(body, &requestData)
		if err != nil {
			app.log.Println("Error! unable to unmarshal request body[amazonSubscriptionHandler]:", err)
			return err
		}

		if subscribeURL, ok := requestData["SubscribeURL"]; ok {
			// append subscribe url to file
			err = VisitSubscriptionURL(subscribeURL.(string), c)
			if err != nil {
				return err
			}
		} else {
			app.log.Println("Error! subscribed url doesn't exist in payload")
			return errors.New("Subscribed url doesn't exist in payload")
		}
	} else {
		app.log.Println("Invalid sns message type:", r.Header.Get("x-amz-sns-message-type"))
		return errors.New("Invalid sns message type")
	}

	return nil
}

// verify auth key using middleware
func checkAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		app := c.Get("app").(*App)
		if c.QueryParam("auth_key") != "" && c.QueryParam("auth_key") == app.constants.ApiKey {
			return next(c)
		}
		return c.JSON(http.StatusBadRequest, "bad request")
	}
}

// handle email events like bounced, complaint etc.
func handleAwsEvents(c echo.Context) error {
	eventData := make(map[string]interface{})
	app := c.Get("app").(*App)
	r := c.Request()

	if r.Header.Get("x-amz-sns-message-type") == "SubscriptionConfirmation" {
		err := amazonSubscriptionHandler(r, c)
		if err != nil {
			return c.JSON(http.StatusBadRequest, err.Error())
		} else {
			return c.JSON(http.StatusCreated, "ok")
		}
	}

	b := bytes.NewBuffer(make([]byte, 0))
	reader := io.TeeReader(r.Body, b)
	err := json.NewDecoder(reader).Decode(&eventData)
	if err != nil {
		app.log.Println("Error! unable to decode json body[handleAwsEvents]:", err)
		return c.JSON(http.StatusBadRequest, "bad request")
	}
	r.Body = ioutil.NopCloser(b)
	r.Body.Close()

	var bounceEventData BounceEvent
	var complaintData ComplaintEvent
	var recipients []string

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		app.log.Println("Error! unable to read request body[handleAwsEvents]:", err)
		return c.JSON(http.StatusBadRequest, "bad request")
	}
	defer r.Body.Close()

	bodyMessage := &bodyMessage{}
	err = json.Unmarshal(body, bodyMessage)
	message := &message{}
	err = json.Unmarshal([]byte(bodyMessage.Message), message)
	var awsAttributes []EmailAttribute
	if message.NotificationType == "Bounce" {
		err = json.Unmarshal([]byte(bodyMessage.Message), &bounceEventData)
		if err != nil {
			app.log.Println("Error! unable to unmarshal json body[handleAwsEvents]:", err)
			return c.JSON(http.StatusBadRequest, "bad request")
		} else if strings.ToLower(bounceEventData.Bounce.BounceType) == "permanent" {
			recipientsData := bounceEventData.Bounce.BouncedRecipients

			i := 0
			for i < len(recipientsData) {
				var awsAttribute EmailAttribute
				awsAttribute.EventType = TypeBounce //bounceEventData.NotificationType
				awsAttribute.EventReason = recipientsData[i].DiagnosticCode
				awsAttribute.EventTimeStamp = bounceEventData.Bounce.Timestamp
				awsAttributes = append(awsAttributes, awsAttribute)
				recipients = append(recipients, recipientsData[i].EmailAddress)
				i++
			}
		}
	} else if message.NotificationType == "Complaint" {
		err = json.Unmarshal([]byte(bodyMessage.Message), &complaintData)
		if err != nil {
			app.log.Println("Error! unable to unmarshal json body[handleAwsEvents]:", err)
			return c.JSON(http.StatusBadRequest, "bad request")
		}
		recipientsData := complaintData.Complaint.ComplainedRecipients
		i := 0
		for i < len(recipientsData) {
			var awsAttribute EmailAttribute
			awsAttribute.EventType = TypeComplaint //complaintData.NotificationType
			awsAttribute.EventReason = complaintData.Complaint.ComplaintFeedbackType
			awsAttribute.EventTimeStamp = complaintData.Complaint.Timestamp
			awsAttributes = append(awsAttributes, awsAttribute)
			recipients = append(recipients, recipientsData[i].EmailAddress)
			i++
		}
	}

	// Blacklist all subscribers
	i := 0
	for _, each := range awsAttributes {
		err = BlacklistSubscriber(recipients[i], each, c)
		if err != nil {
			app.log.Println("Error! unable to blacklist subscriber:", recipients[i])
			return c.JSON(http.StatusBadRequest, err.Error())
		} else {
			app.log.Println("Blacklisted:", recipients[i])
		}
		i++
	}

	return c.JSON(http.StatusCreated, "ok")
}

// blacklist subscriber api
func BlacklistSubscriber(email string, attribs EmailAttribute, c echo.Context) error {
	app := c.Get("app").(*App)
	url := app.constants.RootURL + "/api/subscribers/query/blocklist"
	method := "PUT"
	data := "subscribers.email LIKE " + "'" + email + "'"
	app.log.Println("BlacklistSubscriber URL: ", url)

	reqBody := &SubQueryReq{
		Query:          data,
		Email:          email,
		EventType:      attribs.EventType,
		EventReason:    attribs.EventReason,
		EventTimeStamp: attribs.EventTimeStamp,
	}
	postBody, _ := json.Marshal(reqBody)
	app.log.Println("BlacklistSubscriber body: ", string(postBody))

	client := &http.Client{}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(postBody))
	if err != nil {
		app.log.Printf("Error! unable to blacklist subscriber - create HTTP NewRequest[BlacklistSubscriber]: %v", err)
		return err
	}

	req.SetBasicAuth(string(app.constants.AdminUsername), string(app.constants.AdminPassword))
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		app.log.Printf("Error! unable to blacklist subscriber - call HTTP client[BlacklistSubscriber]: %v", err)
		return err
	}
	defer res.Body.Close()

	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		app.log.Printf("Error! unable to blacklist subscriber - read response body[BlacklistSubscriber]: %v", err)
		return err
	}

	return nil
}

// handle email events like bounced, complaint etc.
func handleEmailParserEvents(c echo.Context) error {
	app := c.Get("app").(*App)

	req := &mailparserReq{}
	if err := c.Bind(req); err != nil {
		app.log.Println("Error! unable to bind json body[handleEmailParserEvents]:", err)
		return c.JSON(http.StatusBadRequest, "bad request")
	}

	if len(req.SenderFieldAddress) == 0 {
		app.log.Println("Error! SenderFieldAddress value is nil [handleEmailParserEvents]")
		return c.JSON(http.StatusBadRequest, "bad request")
	}

	t, err := time.Parse(layoutISO, req.ReceivedAt)
	EmailParserAttribute := EmailAttribute{
		EventType:      TypeBounce,
		EventReason:    "DO NOT email requests from MailParser",
		EventTimeStamp: t,
	}
	err = BlacklistSubscriber(req.SenderFieldAddress, EmailParserAttribute, c)
	if err != nil {
		app.log.Println("Error! unable to blacklist subscriber [handleEmailParserEvents]:", req.SenderFieldAddress)
		return c.JSON(http.StatusBadRequest, err.Error())
	} else {
		app.log.Println("Blacklisted[handleEmailParserEvents]:", req.SenderFieldAddress)
	}

	return c.JSON(http.StatusCreated, "ok")
}

// handle email events like bounced, complaint etc.
func handlePostMarkAppEvents(c echo.Context) error {
	app := c.Get("app").(*App)

	req := &postmarkappReq{}
	if err := c.Bind(req); err != nil {
		app.log.Println("Error! unable to bind json body[handlePostMarkAppEvents]:", err)
		return c.JSON(http.StatusBadRequest, "bad request")
	}

	switch strings.ToLower(req.Type) {
	case postmarkappBadEmail, postmarkappHardBounce, postmarkappManuallyDeactivated, postmarkappSpamComplaint, postmarkappSpamNotification:
		if strings.ToLower(req.RecordType) != "bounce" {
			req.RecordType = TypeComplaint // "Complaint"
		} else {
			req.RecordType = TypeBounce
		}
		postMartAttribute := EmailAttribute{
			EventType:      req.RecordType,
			EventReason:    fmt.Sprint(req.Details, ": ", req.Description),
			EventTimeStamp: req.BouncedAt,
		}
		err := BlacklistSubscriber(req.Email, postMartAttribute, c)
		if err != nil {
			app.log.Println("Error! unable to blacklist subscriber [handlePostMarkAppEvents]:", req.Email)
			return c.JSON(http.StatusBadRequest, err.Error())
		} else {
			app.log.Println("Blacklisted[handlePostMarkAppEvents]:", req.Email)
		}
	}

	return c.JSON(http.StatusCreated, "ok")
}

func VisitSubscriptionURL(subscribedURL string, c echo.Context) error {
	app := c.Get("app").(*App)
	_, err := http.Get(subscribedURL)
	if err != nil {
		app.log.Println("Error! unable to visit subscription url:", err)
		return err
	}
	return nil
}

// write to file
// func AppendToFile(subscribedURL string, c echo.Context) error {
// 	app := c.Get("app").(*App)
// 	filePath:= "/home/emailitapp/devx.emailitapp.com"
// 	fileName := "output.txt"
// 	f, err := os.OpenFile(filePath+"/"+fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
// 	if err != nil {
// 		app.log.Println("Error! unable to open a file:", err)
// 		return err
// 	}

// 	defer f.Close()

// 	if _, err = f.WriteString(subscribedURL + "\r\n"); err != nil {
// 		app.log.Println("Error! unable to write to a file:", err)
// 		return err
// 	}

// 	return nil
// }
