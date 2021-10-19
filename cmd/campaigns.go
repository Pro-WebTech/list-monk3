package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/knadh/listmonk/usecase/campaign"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo"
	"github.com/lib/pq"
	null "gopkg.in/volatiletech/null.v6"
)

// campaignReq is a wrapper over the Campaign model for receiving
// campaign creation and updation data from APIs.
type campaignReq struct {
	models.Campaign

	// Indicates if the "send_at" date should be written or set to null.
	SendLater bool `db:"-" json:"send_later"`

	// This overrides Campaign.Lists to receive and
	// write a list of int IDs during creation and updation.
	// Campaign.Lists is JSONText for sending lists children
	// to the outside world.
	ListIDs pq.Int64Array `db:"-" json:"lists"`

	// This is only relevant to campaign test requests.
	SubscriberEmails pq.StringArray `json:"subscribers"`

	Type string `json:"type"`
}

// campaignContentReq wraps params coming from API requests for converting
// campaign content formats.
type campaignContentReq struct {
	models.Campaign
	From string `json:"from"`
	To   string `json:"to"`
}

type campaignStats struct {
	ID        int       `db:"id" json:"id"`
	Status    string    `db:"status" json:"status"`
	ToSend    int       `db:"to_send" json:"to_send"`
	Sent      int       `db:"sent" json:"sent"`
	Started   null.Time `db:"started_at" json:"started_at"`
	UpdatedAt null.Time `db:"updated_at" json:"updated_at"`
	Rate      float64   `json:"rate"`
}

type campsWrap struct {
	Results models.Campaigns `json:"results"`

	Query   string `json:"query"`
	Total   int    `json:"total"`
	PerPage int    `json:"per_page"`
	Page    int    `json:"page"`
}

var (
	regexFromAddress   = regexp.MustCompile(`(.+?)\s<(.+?)@(.+?)>`)
	regexFullTextQuery = regexp.MustCompile(`\s+`)

	campaignQuerySortFields = []string{"name", "status", "created_at", "updated_at"}
)

// handleGetCampaigns handles retrieval of campaigns.
func handleGetCampaigns(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pg  = getPagination(c.QueryParams(), 20)
		out campsWrap

		id, _     = strconv.Atoi(c.Param("id"))
		status    = c.QueryParams()["status"]
		query     = strings.TrimSpace(c.FormValue("query"))
		orderBy   = c.FormValue("order_by")
		order     = c.FormValue("order")
		noBody, _ = strconv.ParseBool(c.QueryParam("no_body"))
	)

	// Fetch one list.
	single := false
	if id > 0 {
		single = true
	}
	if query != "" {
		query = `%` +
			string(regexFullTextQuery.ReplaceAll([]byte(query), []byte("&"))) + `%`
	}

	// Sort params.
	if !strSliceContains(orderBy, campaignQuerySortFields) {
		orderBy = "created_at"
	}
	if order != sortAsc && order != sortDesc {
		order = sortDesc
	}

	stmt := fmt.Sprintf(app.queries.QueryCampaigns, orderBy, order)

	// Unsafe to ignore scanning fields not present in models.Campaigns.
	if err := db.Select(&out.Results, stmt, id, pq.StringArray(status), query, pg.Offset, pg.Limit); err != nil {
		app.log.Printf("error fetching campaigns: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}
	if single && len(out.Results) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("campaigns.notFound", "name", "{globals.terms.campaign}"))
	}
	if len(out.Results) == 0 {
		out.Results = []models.Campaign{}
		return c.JSON(http.StatusOK, okResp{out})
	}

	for i := 0; i < len(out.Results); i++ {
		// Replace null tags.
		if out.Results[i].Tags == nil {
			out.Results[i].Tags = make(pq.StringArray, 0)
		}

		if noBody {
			out.Results[i].Body = ""
		}
	}

	// Lazy load stats.
	if err := out.Results.LoadStats(app.queries.GetCampaignStats); err != nil {
		app.log.Printf("error fetching campaign stats: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	log.Print(out.Results)

	for v, each := range out.Results {
		out.Results[v].SentPercentage = calPercentage(float64(each.Sent*100), float64(each.ToSend))
	}

	if single {
		return c.JSON(http.StatusOK, okResp{out.Results[0]})
	}

	// Meta.
	out.Total = out.Results[0].Total
	out.Page = pg.Page
	out.PerPage = pg.PerPage

	return c.JSON(http.StatusOK, okResp{out})
}

//Check if divider is zero and return zero or return percentage
func calPercentage(s float64, d float64) int {
	if d == 0 {
		return 0
	}

	return int(math.Round(s / d))
}

// handlePreviewCampaign renders the HTML preview of a campaign body.
func handlePreviewCampaign(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	var camp models.Campaign
	err := app.queries.GetCampaignForPreview.Get(&camp, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
		}

		app.log.Printf("error fetching campaign: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	// There's a body in the request to preview instead of the body in the DB.
	if c.Request().Method == http.MethodPost {
		camp.ContentType = c.FormValue("content_type")
		camp.Body = c.FormValue("body")
	}

	// Use a dummy campaign ID to prevent views and clicks from {{ TrackView }}
	// and {{ TrackLink }} being registered on preview.
	camp.UUID = dummySubscriber.UUID
	if err := camp.CompileTemplate(app.manager.TemplateFuncs(&camp)); err != nil {
		app.log.Printf("error compiling template: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("templates.errorCompiling", "error", err.Error()))
	}

	// Render the message body.
	msg, err := app.manager.NewCampaignMessage(&camp, dummySubscriber)
	if err != nil {
		app.log.Printf("error rendering message: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("templates.errorRendering", "error", err.Error()))
	}

	return c.HTML(http.StatusOK, string(msg.Body()))
}

// handleCampaignContent handles campaign content (body) format conversions.
func handleCampaignContent(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	var camp campaignContentReq
	if err := c.Bind(&camp); err != nil {
		return err
	}

	out, err := camp.ConvertContent(camp.From, camp.To)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleCreateCampaign handles campaign creation.
// Newly created campaigns are always drafts.
func handleCreateCampaign(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		o   campaignReq
	)

	if err := c.Bind(&o); err != nil {
		return err
	}

	// If the campaign's 'opt-in', prepare a default message.
	if o.Type == models.CampaignTypeOptin {
		op, err := makeOptinCampaignMessage(o, app)
		if err != nil {
			return err
		}
		o = op
	}

	// Validate.
	if c, err := validateCampaignFields(o, app); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else {
		o = c
	}

	uu, err := uuid.NewV4()
	if err != nil {
		app.log.Printf("error generating UUID: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}

	// Insert and read ID.
	var newID int
	if err := app.queries.CreateCampaign.Get(&newID,
		uu,
		o.Type,
		o.Name,
		o.Subject,
		o.FromEmail,
		o.Body,
		o.AltBody,
		o.ContentType,
		o.SendAt,
		pq.StringArray(normalizeTags(o.Tags)),
		o.Messenger,
		o.TemplateID,
		o.ListIDs,
	); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("campaigns.noSubs"))
		}

		app.log.Printf("error creating campaign: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorCreating",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	// Hand over to the GET handler to return the last insertion.
	return handleGetCampaigns(copyEchoCtx(c, map[string]string{
		"id": fmt.Sprintf("%d", newID),
	}))
}

// handleUpdateCampaign handles campaign modification.
// Campaigns that are done cannot be modified.
func handleUpdateCampaign(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))

	}

	var cm models.Campaign
	if err := app.queries.GetCampaign.Get(&cm, id, nil); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.campaign}"))
		}

		app.log.Printf("error fetching campaign: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	if isCampaignalMutable(cm.Status) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("campaigns.cantUpdate"))
	}

	// Read the incoming params into the existing campaign fields from the DB.
	// This allows updating of values that have been sent where as fields
	// that are not in the request retain the old values.
	o := campaignReq{Campaign: cm}
	if err := c.Bind(&o); err != nil {
		return err
	}

	if c, err := validateCampaignFields(o, app); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else {
		o = c
	}

	_, err := app.queries.UpdateCampaign.Exec(cm.ID,
		o.Name,
		o.Subject,
		o.FromEmail,
		o.Body,
		o.AltBody,
		o.ContentType,
		o.SendAt,
		o.SendLater,
		pq.StringArray(normalizeTags(o.Tags)),
		o.Messenger,
		o.TemplateID,
		o.ListIDs)
	if err != nil {
		app.log.Printf("error updating campaign: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	return handleGetCampaigns(c)
}

// handleUpdateCampaignStatus handles campaign status modification.
func handleUpdateCampaignStatus(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	var cm models.Campaign
	if err := app.queries.GetCampaign.Get(&cm, id, nil); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("globals.message.notFound", "name", "{globals.terms.campaign}"))
		}

		app.log.Printf("error fetching campaign: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	// Incoming params.
	var o campaignReq
	if err := c.Bind(&o); err != nil {
		return err
	}

	if o.Status == models.CampaignStatusRunning || o.Status == models.CampaignStatusScheduled {
		var cmEmails models.CampaignMeta
		if err := app.queries.ValidateStartCampaign.Get(&cmEmails, id); err != nil {
			if err == sql.ErrNoRows {
				return echo.NewHTTPError(http.StatusBadRequest,
					app.i18n.Ts("globals.message.notFound", "name", "{globals.terms.campaign}"))
			}

			app.log.Printf("error fetching campaign: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorFetching",
					"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
		}

		if cmEmails.EmailAllowed < (cmEmails.ToSend + cmEmails.TotalSent) {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("globals.messages.errorUpdating",
					"name", "{globals.terms.campaign}", "error", "You have reached your sending limit!"))
		}
	}

	errMsg := ""
	switch o.Status {
	case models.CampaignStatusDraft:
		if cm.Status != models.CampaignStatusScheduled {
			errMsg = app.i18n.T("campaigns.onlyScheduledAsDraft")
		}
	case models.CampaignStatusScheduled:
		if cm.Status != models.CampaignStatusDraft {
			errMsg = app.i18n.T("campaigns.onlyDraftAsScheduled")
		}
		if !cm.SendAt.Valid {
			errMsg = app.i18n.T("campaigns.needsSendAt")
		}

	case models.CampaignStatusRunning:
		if cm.Status != models.CampaignStatusPaused && cm.Status != models.CampaignStatusDraft {
			errMsg = app.i18n.T("campaigns.onlyPausedDraft")
		}
	case models.CampaignStatusPaused:
		if cm.Status != models.CampaignStatusRunning {
			errMsg = app.i18n.T("campaigns.onlyActivePause")
		}
	case models.CampaignStatusCancelled:
		if cm.Status != models.CampaignStatusRunning && cm.Status != models.CampaignStatusPaused {
			errMsg = app.i18n.T("campaigns.onlyActiveCancel")
		}
	}

	if len(errMsg) > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, errMsg)
	}

	res, err := app.queries.UpdateCampaignStatus.Exec(cm.ID, o.Status)
	if err != nil {
		app.log.Printf("error updating campaign status: %v", err)

		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	if n, _ := res.RowsAffected(); n == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	if o.Status == models.CampaignStatusCancelled {
		go func() {
			<-time.After(time.Millisecond * 500)
			app.sigChan <- syscall.SIGHUP
		}()
	}

	return handleGetCampaigns(c)
}

// handleDeleteCampaign handles campaign deletion.
// Only scheduled campaigns that have not started yet can be deleted.
func handleDeleteCampaign(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	var cm models.Campaign
	if err := app.queries.GetCampaign.Get(&cm, id, nil); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("globals.messages.notFound",
					"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
		}

		app.log.Printf("error fetching campaign: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	if _, err := app.queries.DeleteCampaign.Exec(cm.ID); err != nil {
		app.log.Printf("error deleting campaign: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorDeleting",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))

	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleGetRunningCampaignStats returns stats of a given set of campaign IDs.
func handleGetRunningCampaignStats(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out []campaignStats
	)

	if err := app.queries.GetCampaignStatus.Select(&out, models.CampaignStatusRunning); err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusOK, okResp{[]struct{}{}})
		}

		app.log.Printf("error fetching campaign stats: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	} else if len(out) == 0 {
		return c.JSON(http.StatusOK, okResp{[]struct{}{}})
	}

	// Compute rate.
	for i, c := range out {
		if c.Started.Valid && c.UpdatedAt.Valid {
			diff := c.UpdatedAt.Time.Sub(c.Started.Time).Minutes()
			if diff > 0 {
				var (
					sent = float64(c.Sent)
					rate = sent / diff
				)
				if rate > sent || rate > float64(c.ToSend) {
					rate = sent
				}
				out[i].Rate = rate
			}
		}
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleTestCampaign handles the sending of a campaign message to
// arbitrary subscribers for testing.
func handleTestCampaign(c echo.Context) error {
	var (
		app       = c.Get("app").(*App)
		campID, _ = strconv.Atoi(c.Param("id"))
		req       campaignReq
	)

	if campID < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.errorID"))
	}

	// Get and validate fields.
	if err := c.Bind(&req); err != nil {
		return err
	}

	// Validate.
	if c, err := validateCampaignFields(req, app); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else {
		req = c
	}
	if len(req.SubscriberEmails) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("campaigns.noSubsToTest"))
	}

	// Get the subscribers.
	for i := 0; i < len(req.SubscriberEmails); i++ {
		req.SubscriberEmails[i] = strings.ToLower(strings.TrimSpace(req.SubscriberEmails[i]))
	}
	var subs models.Subscribers
	if err := app.queries.GetSubscribersByEmails.Select(&subs, req.SubscriberEmails); err != nil {
		app.log.Printf("error fetching subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	} else if len(subs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("campaigns.noKnownSubsToTest"))
	}

	// The campaign.
	var camp models.Campaign
	if err := app.queries.GetCampaignForPreview.Get(&camp, campID); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("globals.messages.notFound",
					"name", "{globals.terms.campaign}"))
		}

		app.log.Printf("error fetching campaign: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
	}

	// Override certain values from the DB with incoming values.
	camp.Name = req.Name
	camp.Subject = req.Subject
	camp.FromEmail = req.FromEmail
	camp.Body = req.Body
	camp.AltBody = req.AltBody
	camp.Messenger = req.Messenger
	camp.ContentType = req.ContentType
	camp.TemplateID = req.TemplateID

	// Send the test messages.
	for _, s := range subs {
		sub := s
		if err := sendTestMessage(sub, &camp, app); err != nil {
			app.log.Printf("error sending test message: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("campaigns.errorSendTest", "error", err.Error()))
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleTestCampaign handles the sending of a campaign message to
// arbitrary subscribers for testing.
func handleSendTestEmailCampaign(c echo.Context) error {
	var (
		app              = c.Get("app").(*App)
		campID, _        = strconv.Atoi(c.QueryParam("campaignid"))
		subscribers      = strings.Split(c.QueryParam("subscribers"), ",")
		SubscriberEmails pq.StringArray
	)

	if campID < 1 {
		return echo.NewHTTPError(http.StatusBadRequest,
			defaultResponse{Message: app.i18n.T("globals.messages.errorCampID"), Code: http.StatusBadRequest, Data: struct{}{}})
	}

	if len(subscribers) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			defaultResponse{Message: app.i18n.T("campaigns.noSubsEmailToSend"), Code: http.StatusBadRequest, Data: struct{}{}})
	}

	// Get the subscribers.
	for i := 0; i < len(subscribers); i++ {
		SubscriberEmails = append(SubscriberEmails, strings.ToLower(strings.TrimSpace(subscribers[i])))
	}
	var subs models.Subscribers
	if err := app.queries.GetSubscribersByEmails.Select(&subs, SubscriberEmails); err != nil {
		app.log.Printf("error fetching subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			defaultResponse{Message: app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)), Code: http.StatusInternalServerError, Data: struct{}{}})
	} else if len(subs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			defaultResponse{Message: app.i18n.T("campaigns.noKnownSubsToTest"), Code: http.StatusBadRequest, Data: struct{}{}})
	}

	// The campaign.
	var camp models.Campaign
	if err := app.queries.GetCampaignForPreview.Get(&camp, campID); err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusBadRequest,
				defaultResponse{Message: app.i18n.Ts("globals.messages.notFound",
					"name", "{globals.terms.campaign}"), Code: http.StatusBadRequest, Data: struct{}{}})
		}

		app.log.Printf("error fetching campaign: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			defaultResponse{Message: app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.campaign}", "error", pqErrMsg(err)), Code: http.StatusInternalServerError, Data: struct{}{}})
	}

	// Send the test messages.

	for _, s := range subs {
		sub := s
		if err := sendTestMessage(sub, &camp, app); err != nil {
			app.log.Printf("error sending test message: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				defaultResponse{Message: app.i18n.Ts("campaigns.errorSendTest", "error", err.Error()), Code: http.StatusInternalServerError, Data: struct{}{}})
		}
	}

	return c.JSON(http.StatusOK, defaultResponse{Message: "success", Code: http.StatusOK, Data: struct{}{}})
}

// sendTestMessage takes a campaign and a subsriber and sends out a sample campaign message.
func sendTestMessage(sub models.Subscriber, camp *models.Campaign, app *App) error {
	if err := camp.CompileTemplate(app.manager.TemplateFuncs(camp)); err != nil {
		app.log.Printf("error compiling template: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("templates.errorCompiling", "error", err.Error()))
	}

	// Create a sample campaign message.
	msg, err := app.manager.NewCampaignMessage(camp, sub)
	if err != nil {
		app.log.Printf("error rendering message: %v", err)
		return echo.NewHTTPError(http.StatusNotFound,
			app.i18n.Ts("templates.errorRendering", "error", err.Error()))
	}

	return app.manager.PushCampaignMessage(msg)
}

// validateCampaignFields validates incoming campaign field values.
func validateCampaignFields(c campaignReq, app *App) (campaignReq, error) {
	if c.FromEmail == "" {
		c.FromEmail = app.constants.FromEmail
	} else if !regexFromAddress.Match([]byte(c.FromEmail)) {
		if !subimporter.IsEmail(c.FromEmail) {
			return c, errors.New(app.i18n.T("campaigns.fieldInvalidFromEmail"))
		}
	}

	if !strHasLen(c.Name, 1, stdInputMaxLen) {
		return c, errors.New(app.i18n.T("campaigns.fieldInvalidName"))
	}
	if !strHasLen(c.Subject, 1, stdInputMaxLen) {
		return c, errors.New(app.i18n.T("campaigns.fieldInvalidSubject"))
	}

	// if !hasLen(c.Body, 1, bodyMaxLen) {
	// 	return c,errors.New("invalid length for `body`")
	// }

	// If there's a "send_at" date, it should be in the future.
	if c.SendAt.Valid {
		if c.SendAt.Time.Before(time.Now()) {
			return c, errors.New(app.i18n.T("campaigns.fieldInvalidSendAt"))
		}
	}

	if len(c.ListIDs) == 0 {
		return c, errors.New(app.i18n.T("campaigns.fieldInvalidListIDs"))
	}

	if !app.manager.HasMessenger(c.Messenger) {
		return c, errors.New(app.i18n.Ts("campaigns.fieldInvalidMessenger", "name", c.Messenger))
	}

	camp := models.Campaign{Body: c.Body, TemplateBody: tplTag}
	if err := c.CompileTemplate(app.manager.TemplateFuncs(&camp)); err != nil {
		return c, errors.New(app.i18n.Ts("campaigns.fieldInvalidBody", "error", err.Error()))
	}

	return c, nil
}

// isCampaignalMutable tells if a campaign's in a state where it's
// properties can be mutated.
func isCampaignalMutable(status string) bool {
	return status == models.CampaignStatusRunning ||
		status == models.CampaignStatusCancelled ||
		status == models.CampaignStatusFinished
}

// makeOptinCampaignMessage makes a default opt-in campaign message body.
func makeOptinCampaignMessage(o campaignReq, app *App) (campaignReq, error) {
	if len(o.ListIDs) == 0 {
		return o, echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("campaigns.fieldInvalidListIDs"))
	}

	// Fetch double opt-in lists from the given list IDs.
	var lists []models.List
	err := app.queries.GetListsByOptin.Select(&lists, models.ListOptinDouble, pq.Int64Array(o.ListIDs), nil)
	if err != nil {
		app.log.Printf("error fetching lists for opt-in: %s", pqErrMsg(err))
		return o, echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	// No opt-in lists.
	if len(lists) == 0 {
		return o, echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("campaigns.noOptinLists"))
	}

	// Construct the opt-in URL with list IDs.
	listIDs := url.Values{}
	for _, l := range lists {
		listIDs.Add("l", l.UUID)
	}
	// optinURLFunc := template.URL("{{ OptinURL }}?" + listIDs.Encode())
	optinURLAttr := template.HTMLAttr(fmt.Sprintf(`href="{{ OptinURL }}%s"`, listIDs.Encode()))

	// Prepare sample opt-in message for the campaign.
	var b bytes.Buffer
	if err := app.notifTpls.ExecuteTemplate(&b, "optin-campaign", struct {
		Lists        []models.List
		OptinURLAttr template.HTMLAttr
	}{lists, optinURLAttr}); err != nil {
		app.log.Printf("error compiling 'optin-campaign' template: %v", err)
		return o, echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("templates.errorCompiling", "error", err.Error()))
	}

	o.Body = b.String()
	return o, nil
}

type campaignHandler struct {
	svc campaign.Service
	lo  *log.Logger
}

func setupCampaignHandler(svc campaign.Service, lo *log.Logger) *campaignHandler {
	return &campaignHandler{
		svc: svc,
		lo:  lo,
	}
}

func (h *campaignHandler) getListMessenger(c echo.Context) error {
	r, err := h.svc.GetListMessengers(h.lo)
	if err != nil {
		h.lo.Println("err getSettings: ", err)
		return err
	}
	return c.JSON(http.StatusOK, defaultResponse{Message: "success", Code: http.StatusOK, Data: r})
}
