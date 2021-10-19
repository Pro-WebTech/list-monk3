package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo"
	"github.com/lib/pq"
)

const (
	dummyUUID = "00000000-0000-0000-0000-000000000000"
)

// subQueryReq is a "catch all" struct for reading various
// subscriber related requests.
type subQueryReq struct {
	Query          string        `json:"query"`
	ListIDs        pq.Int64Array `json:"list_ids"`
	TargetListIDs  pq.Int64Array `json:"target_list_ids"`
	SubscriberIDs  pq.Int64Array `json:"ids"`
	Action         string        `json:"action"`
	List           []SubQueryReq `json:"list"`
	EventType      string        `json:"eventType"`
	EventReason    string        `json:"eventReason"`
	EventTimeStamp time.Time     `json:"eventTimeStamp"`
	Email          string        `json:"email"`
}

type subsWrap struct {
	Results models.Subscribers `json:"results"`

	Query           string `json:"query"`
	Total           int    `json:"total"`
	PerPage         int    `json:"per_page"`
	Page            int    `json:"page"`
	Id              int    `json:"id"`
	Name            string `json:"name"`
	SubscriberCount int    `json:"subscriber_count"`
}

type subUpdateReq struct {
	models.Subscriber
	RawAttribs json.RawMessage `json:"attribs"`
	Lists      pq.Int64Array   `json:"lists"`
	ListUUIDs  pq.StringArray  `json:"list_uuids"`
}

// subProfileData represents a subscriber's collated data in JSON
// for export.
type subProfileData struct {
	Email         string          `db:"email" json:"-"`
	Profile       json.RawMessage `db:"profile" json:"profile,omitempty"`
	Subscriptions json.RawMessage `db:"subscriptions" json:"subscriptions,omitempty"`
	CampaignViews json.RawMessage `db:"campaign_views" json:"campaign_views,omitempty"`
	LinkClicks    json.RawMessage `db:"link_clicks" json:"link_clicks,omitempty"`
}

// subOptin contains the data that's passed to the double opt-in e-mail template.
type subOptin struct {
	*models.Subscriber

	OptinURL string
	Lists    []models.List
}

var (
	dummySubscriber = models.Subscriber{
		Email: "dummy@listmonk.app",
		Name:  "Dummy Subscriber",
		UUID:  dummyUUID,
	}

	subQuerySortFields = []string{"email", "name", "created_at", "updated_at"}

	errSubscriberExists = errors.New("subscriber already exists")
)

// handleGetSubscriber handles the retrieval of a single subscriber by ID.
func handleGetSubscriber(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	sub, err := getSubscriber(id, "", "", app)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{sub})
}

// handleQuerySubscribers handles querying subscribers based on an arbitrary SQL expression.
func handleQuerySubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pg  = getPagination(c.QueryParams(), 30)

		// Limit the subscribers to a particular list?
		listID, _ = strconv.Atoi(c.FormValue("list_id"))

		// The "WHERE ?" bit.
		query   = sanitizeSQLExp(c.FormValue("query"))
		orderBy = c.FormValue("order_by")
		order   = c.FormValue("order")
		out     subsWrap
	)

	listIDs := pq.Int64Array{}
	if listID < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.errorID"))
	} else if listID > 0 {
		listIDs = append(listIDs, int64(listID))
	}

	// There's an arbitrary query condition.
	cond := ""
	if query != "" {
		cond = " AND " + query
	}

	// Sort params.
	if !strSliceContains(orderBy, subQuerySortFields) {
		orderBy = "updated_at"
	}
	if order != sortAsc && order != sortDesc {
		order = sortAsc
	}

	stmt := fmt.Sprintf(app.queries.QuerySubscribers, cond, orderBy, order)

	// Create a readonly transaction to prevent mutations.
	tx, err := app.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		app.log.Printf("error preparing subscriber query: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
	}
	defer tx.Rollback()

	// Run the query. stmt is the raw SQL query.
	if err := tx.Select(&out.Results, stmt, listIDs, pg.Offset, pg.Limit); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	// Lazy load lists for each subscriber.
	if err := out.Results.LoadLists(app.queries.GetSubscriberListsLazy); err != nil {
		app.log.Printf("error fetching subscriber lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	out.Query = query
	if len(out.Results) == 0 {
		out.Results = make(models.Subscribers, 0)
		return c.JSON(http.StatusOK, okResp{out})
	}

	// Meta.
	out.Total = out.Results[0].Total
	out.Page = pg.Page
	out.PerPage = pg.PerPage

	return c.JSON(http.StatusOK, okResp{out})
}

// handleExportSubscribers handles querying subscribers based on an arbitrary SQL expression.
func handleExportSubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)

		// Limit the subscribers to a particular list?
		listID, _ = strconv.Atoi(c.FormValue("list_id"))

		// The "WHERE ?" bit.
		query = sanitizeSQLExp(c.FormValue("query"))
	)

	listIDs := pq.Int64Array{}
	if listID < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.errorID"))
	} else if listID > 0 {
		listIDs = append(listIDs, int64(listID))
	}

	// There's an arbitrary query condition.
	cond := ""
	if query != "" {
		cond = " AND " + query
	}

	stmt := fmt.Sprintf(app.queries.QuerySubscribersForExport, cond)

	// Verify that the arbitrary SQL search expression is read only.
	if cond != "" {
		tx, err := app.db.Unsafe().BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
		if err != nil {
			app.log.Printf("error preparing subscriber query: %v", err)
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
		}
		defer tx.Rollback()

		if _, err := tx.Query(stmt, nil, 0, 1); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
		}
	}

	// Prepare the actual query statement.
	tx, err := db.Preparex(stmt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
	}

	// Run the query until all rows are exhausted.
	var (
		id = 0

		h  = c.Response().Header()
		wr = csv.NewWriter(c.Response())
	)

	h.Set(echo.HeaderContentType, echo.MIMEOctetStream)
	h.Set("Content-type", "text/csv")
	h.Set(echo.HeaderContentDisposition, "attachment; filename="+"subscribers.csv")
	h.Set("Content-Transfer-Encoding", "binary")
	h.Set("Cache-Control", "no-cache")
	wr.Write([]string{"uuid", "email", "name", "attributes", "status", "created_at", "updated_at"})

loop:
	for {
		var out []models.SubscriberExport
		if err := tx.Select(&out, listIDs, id, app.constants.DBBatchSize); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorFetching",
					"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
		}
		if len(out) == 0 {
			break loop
		}

		for _, r := range out {
			if err = wr.Write([]string{r.UUID, r.Email, r.Name, r.Attribs, r.Status,
				r.CreatedAt.Time.String(), r.UpdatedAt.Time.String()}); err != nil {
				app.log.Printf("error streaming CSV export: %v", err)
				break loop
			}
		}
		wr.Flush()

		id = out[len(out)-1].ID
	}

	return nil
}

// handleCreateSubscriber handles the creation of a new subscriber.
func handleCreateSubscriber(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req subimporter.SubReq
	)

	// Get and validate fields.
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := subimporter.ValidateFields(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	//if err := subimporter.ValidateEmail(req.Email); err != nil {
	//	return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	//}

	// Insert the subscriber into the DB.
	sub, isNew, _, err := insertSubscriber(req, app)
	if err != nil {
		return err
	}
	if !isNew {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.emailExists"))
	}

	listName := subimporter.ValidateSmartEmail(req.Email)

	if len(listName) > 0 {
		var listResult []models.List
		var pg = getPagination(c.QueryParams(), 30)
		filterTempList := fmt.Sprintf(" AND name = '%s' ", listName)
		if err := db.Select(&listResult, fmt.Sprintf(app.queries.QueryLists, filterTempList, "created_at", "asc"), 0, pg.Offset, pg.Limit); err != nil {
			app.log.Printf("error fetching lists: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorFetching",
					"name", "{globals.terms.lists}", "error", pqErrMsg(err)))
		}

		var newListID int
		if len(listResult) > 0 {
			newListID = listResult[0].ID
		}

		if newListID > 0 {
			var IDs pq.Int64Array
			IDs = append(IDs, int64(sub.ID))

			_, err = app.queries.AddSubscribersToLists.Exec(IDs, pq.Int64Array{int64(newListID)})
			if err != nil {
				app.log.Printf("error updating subscriptions: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError,
					app.i18n.Ts("globals.messages.errorUpdating",
						"name", "{globals.terms.subscribers}", "error", err.Error()))
			}
		}
	}

	return c.JSON(http.StatusOK, okResp{sub})
}

// handleUpdateSubscriber handles modification of a subscriber.
func handleUpdateSubscriber(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.ParseInt(c.Param("id"), 10, 64)
		req   subUpdateReq
	)
	// Get and validate fields.
	if err := c.Bind(&req); err != nil {
		return err
	}

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}
	if req.Email != "" && !subimporter.IsEmail(req.Email) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidEmail"))
	}
	if req.Name != "" && !strHasLen(req.Name, 1, stdInputMaxLen) {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidName"))
	}

	// If there's an attribs value, validate it.
	if len(req.RawAttribs) > 0 {
		var a models.SubscriberAttribs
		if err := json.Unmarshal(req.RawAttribs, &a); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorUpdating",
					"name", "{globals.terms.subscriber}", "error", err.Error()))
		}
	}

	_, err := app.queries.UpdateSubscriber.Exec(id,
		strings.ToLower(strings.TrimSpace(req.Email)),
		strings.TrimSpace(req.Name),
		req.Status,
		req.RawAttribs,
		req.Lists)
	if err != nil {
		app.log.Printf("error updating subscriber: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}

	// Send a confirmation e-mail (if there are any double opt-in lists).
	sub, err := getSubscriber(int(id), "", "", app)
	if err != nil {
		return err
	}
	_, _ = sendOptinConfirmation(sub, []int64(req.Lists), app)

	return c.JSON(http.StatusOK, okResp{sub})
}

// handleGetSubscriberSendOptin sends an optin confirmation e-mail to a subscriber.
func handleSubscriberSendOptin(c echo.Context) error {
	var (
		app   = c.Get("app").(*App)
		id, _ = strconv.Atoi(c.Param("id"))
	)

	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Fetch the subscriber.
	out, err := getSubscriber(id, "", "", app)
	if err != nil {
		app.log.Printf("error fetching subscriber: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	if _, err := sendOptinConfirmation(out, nil, app); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.T("subscribers.errorSendingOptin"))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleBlocklistSubscribers handles the blocklisting of one or more subscribers.
// It takes either an ID in the URI, or a list of IDs in the request body.
func handleBlocklistSubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pID = c.Param("id")
		IDs pq.Int64Array
	)

	// Is it a /:id call?
	if pID != "" {
		id, _ := strconv.ParseInt(pID, 10, 64)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
		}
		IDs = append(IDs, id)
	} else {
		// Multiple IDs.
		var req subQueryReq
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("subscribers.errorInvalidIDs", "error", err.Error()))
		}
		if len(req.SubscriberIDs) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest,
				"No IDs given.")
		}
		IDs = req.SubscriberIDs
	}

	if _, err := app.queries.BlocklistSubscribers.Exec(IDs); err != nil {
		app.log.Printf("error blocklisting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("subscribers.errorBlocklisting", "error", err.Error()))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleManageSubscriberLists handles bulk addition or removal of subscribers
// from or to one or more target lists.
// It takes either an ID in the URI, or a list of IDs in the request body.
func handleManageSubscriberLists(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pID = c.Param("id")
		IDs pq.Int64Array
	)

	// Is it a /:id call?
	if pID != "" {
		id, _ := strconv.ParseInt(pID, 10, 64)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
		}
		IDs = append(IDs, id)
	}

	var req subQueryReq
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("subscribers.errorInvalidIDs", "error", err.Error()))
	}
	if len(req.SubscriberIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.errorNoIDs"))
	}
	if len(IDs) == 0 {
		IDs = req.SubscriberIDs
	}
	if len(req.TargetListIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.errorNoListsGiven"))
	}

	// Action.
	var err error
	switch req.Action {
	case "add":
		_, err = app.queries.AddSubscribersToLists.Exec(IDs, req.TargetListIDs)
	case "remove":
		_, err = app.queries.DeleteSubscriptions.Exec(IDs, req.TargetListIDs)
	case "unsubscribe":
		_, err = app.queries.UnsubscribeSubscribersFromLists.Exec(IDs, req.TargetListIDs)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidAction"))
	}

	if err != nil {
		app.log.Printf("error updating subscriptions: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleDeleteSubscribers handles subscriber deletion.
// It takes either an ID in the URI, or a list of IDs in the request body.
func handleDeleteSubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pID = c.Param("id")
		IDs pq.Int64Array
	)

	// Is it an /:id call?
	if pID != "" {
		id, _ := strconv.ParseInt(pID, 10, 64)
		if id < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
		}
		IDs = append(IDs, id)
	} else {
		// Multiple IDs.
		i, err := parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("subscribers.errorInvalidIDs", "error", err.Error()))
		}
		if len(i) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest,
				app.i18n.Ts("subscribers.errorNoIDs", "error", err.Error()))
		}
		IDs = i
	}

	if _, err := app.queries.DeleteSubscribers.Exec(IDs, nil); err != nil {
		app.log.Printf("error deleting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorDeleting",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleDeleteSubscribersByQuery bulk deletes based on an
// arbitrary SQL expression.
func handleDeleteSubscribersByQuery(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req subQueryReq
	)

	if err := c.Bind(&req); err != nil {
		return err
	}

	err := app.queries.execSubscriberQueryTpl(sanitizeSQLExp(req.Query),
		app.queries.DeleteSubscribersByQuery,
		req.ListIDs, app.db)
	if err != nil {
		app.log.Printf("error deleting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorDeleting",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleBlocklistSubscribersByQuery bulk blocklists subscribers
// based on an arbitrary SQL expression.
func handleBlocklistSubscribersByQuery(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req subQueryReq
	)

	err := c.Bind(&req)
	if err != nil {
		return err
	}

	buf := strings.Builder{}
	filter := ""
	insertValues := "(%d, '%s', '%s', '%s', %d)"
	if len(req.Email) > 0 {
		var IDs int64
		err = app.queries.FindSubscribersIdByEmail.Get(&IDs, req.Email)
		if err != nil {
			app.log.Printf("error FindSubsribersIdByEmail: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("subscribers.errorBlocklisting", "error", pqErrMsg(err)))
		}
		buf.WriteString(fmt.Sprintf(insertValues, IDs, req.EventType, req.EventReason, req.EventTimeStamp.Format("2006-01-02T15:04:05Z"), 0))
		filter = buf.String()
	} else {
		if len(req.List) == 0 {
			return c.JSON(http.StatusOK, okResp{true})
		}
		for _, eachList := range req.List {
			err = app.queries.FindSubscribersIdByEmail.Get(&eachList.SubscriberIDs, eachList.Email)
			if err != nil {
				app.log.Printf("error FindSubsribersIdByEmail: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError,
					app.i18n.Ts("subscribers.errorBlocklisting", "error", pqErrMsg(err)))
			}
			if eachList.SubscriberIDs == 0 {
				continue
			}
			buf.WriteString(fmt.Sprintf(insertValues, eachList.SubscriberIDs, eachList.EventType, eachList.EventReason, eachList.EventTimeStamp.Format("2006-01-02T15:04:05Z"), 1))
			buf.WriteString(", ")
			req.SubscriberIDs = append(req.SubscriberIDs, eachList.SubscriberIDs)
		}
		if len(buf.String()) > 2 {
			filter = buf.String()
			filter = filter[:len(filter)-2]
		}
	}

	if len(req.SubscriberIDs) > 0 {
		err = app.queries.newExecSubscriberQueryTpl(sanitizeSQLExp(req.Query),
			app.queries.BlocklistSubscribersByQuery,
			req.SubscriberIDs, app.db)
	} else {
		err = app.queries.execSubscriberQueryTpl(sanitizeSQLExp(req.Query),
			app.queries.BlocklistSubscribersByQuery,
			req.ListIDs, app.db)
	}
	if err != nil {
		app.log.Printf("error blocklisting subscribers: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("subscribers.errorBlocklisting", "error", pqErrMsg(err)))
	}

	if _, err := app.db.Exec(fmt.Sprintf(app.queries.InsertAttributeBlocklistSubscribers, filter)); err != nil {
		return err
	}
	if err != nil {
		app.log.Printf("error updating subscriber: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleManageSubscriberListsByQuery bulk adds/removes/unsubscribers subscribers
// from one or more lists based on an arbitrary SQL expression.
func handleManageSubscriberListsByQuery(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		req subQueryReq
	)

	if err := c.Bind(&req); err != nil {
		return err
	}
	if len(req.TargetListIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.T("subscribers.errorNoListsGiven"))
	}

	// Action.
	var stmt string
	switch req.Action {
	case "add":
		stmt = app.queries.AddSubscribersToListsByQuery
	case "remove":
		stmt = app.queries.DeleteSubscriptionsByQuery
	case "unsubscribe":
		stmt = app.queries.UnsubscribeSubscribersFromListsByQuery
	default:
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("subscribers.invalidAction"))
	}

	err := app.queries.execSubscriberQueryTpl(sanitizeSQLExp(req.Query),
		stmt, req.ListIDs, app.db, req.TargetListIDs)
	if err != nil {
		app.log.Printf("error updating subscriptions: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUpdating",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleExportSubscriberData pulls the subscriber's profile,
// list subscriptions, campaign views and clicks and produces
// a JSON report. This is a privacy feature and depends on the
// configuration in app.Constants.Privacy.
func handleExportSubscriberData(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pID = c.Param("id")
	)
	id, _ := strconv.ParseInt(pID, 10, 64)
	if id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	_, b, err := exportSubscriberData(id, "", app.constants.Privacy.Exportable, app)
	if err != nil {
		app.log.Printf("error exporting subscriber data: %s", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", err.Error()))
	}

	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="data.json"`)
	return c.Blob(http.StatusOK, "application/json", b)
}

// insertSubscriber inserts a subscriber and returns the ID. The first bool indicates if
// it was a new subscriber, and the second bool indicates if the subscriber was sent an optin confirmation.
func insertSubscriber(req subimporter.SubReq, app *App) (models.Subscriber, bool, bool, error) {
	uu, err := uuid.NewV4()
	if err != nil {
		return req.Subscriber, false, false, err
	}
	req.UUID = uu.String()

	var (
		isNew     = true
		subStatus = models.SubscriptionStatusUnconfirmed
	)
	if req.PreconfirmSubs {
		subStatus = models.SubscriptionStatusConfirmed
	}

	if err = app.queries.InsertSubscriber.Get(&req.ID,
		req.UUID,
		req.Email,
		strings.TrimSpace(req.Name),
		req.Status,
		req.Attribs,
		req.Lists,
		req.ListUUIDs,
		subStatus); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Constraint == "subscribers_email_key" {
			isNew = false
		} else {
			// return req.Subscriber, errSubscriberExists
			app.log.Printf("error inserting subscriber: %v", err)
			return req.Subscriber, false, false, echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorCreating",
					"name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
		}
	}

	// Fetch the subscriber's full data. If the subscriber already existed and wasn't
	// created, the id will be empty. Fetch the details by e-mail then.
	sub, err := getSubscriber(req.ID, "", strings.ToLower(req.Email), app)
	if err != nil {
		return sub, false, false, err
	}

	hasOptin := false
	if !req.PreconfirmSubs {
		// Send a confirmation e-mail (if there are any double opt-in lists).
		num, _ := sendOptinConfirmation(sub, []int64(req.Lists), app)
		hasOptin = num > 0
	}
	return sub, isNew, hasOptin, nil
}

// getSubscriber gets a single subscriber by ID, uuid, or e-mail in that order.
// Only one of these params should have a value.
func getSubscriber(id int, uuid, email string, app *App) (models.Subscriber, error) {
	var out models.Subscribers

	if err := app.queries.GetSubscriber.Select(&out, id, uuid, email); err != nil {
		app.log.Printf("error fetching subscriber: %v", err)
		return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscriber}", "error", pqErrMsg(err)))
	}
	if len(out) == 0 {
		return models.Subscriber{}, echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.subscriber}"))
	}
	if err := out.LoadLists(app.queries.GetSubscriberListsLazy); err != nil {
		app.log.Printf("error loading subscriber lists: %v", err)
		return models.Subscriber{}, echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	return out[0], nil
}

// exportSubscriberData collates the data of a subscriber including profile,
// subscriptions, campaign_views, link_clicks (if they're enabled in the config)
// and returns a formatted, indented JSON payload. Either takes a numeric id
// and an empty subUUID or takes 0 and a string subUUID.
func exportSubscriberData(id int64, subUUID string, exportables map[string]bool, app *App) (subProfileData, []byte, error) {
	// Get the subscriber's data. A single query that gets the profile,
	// list subscriptions, campaign views, and link clicks. Names of
	// private lists are replaced with "Private list".
	var (
		data subProfileData
		uu   interface{}
	)
	// UUID should be a valid value or a nil.
	if subUUID != "" {
		uu = subUUID
	}
	if err := app.queries.ExportSubscriberData.Get(&data, id, uu); err != nil {
		app.log.Printf("error fetching subscriber export data: %v", err)
		return data, nil, err
	}

	// Filter out the non-exportable items.
	if _, ok := exportables["profile"]; !ok {
		data.Profile = nil
	}
	if _, ok := exportables["subscriptions"]; !ok {
		data.Subscriptions = nil
	}
	if _, ok := exportables["campaign_views"]; !ok {
		data.CampaignViews = nil
	}
	if _, ok := exportables["link_clicks"]; !ok {
		data.LinkClicks = nil
	}

	// Marshal the data into an indented payload.
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		app.log.Printf("error marshalling subscriber export data: %v", err)
		return data, nil, err
	}
	return data, b, nil
}

// sendOptinConfirmation sends a double opt-in confirmation e-mail to a subscriber
// if at least one of the given listIDs is set to optin=double. It returns the number of
// opt-in lists that were found.
func sendOptinConfirmation(sub models.Subscriber, listIDs []int64, app *App) (int, error) {
	var lists []models.List

	// Fetch double opt-in lists from the given list IDs.
	// Get the list of subscription lists where the subscriber hasn't confirmed.
	if err := app.queries.GetSubscriberLists.Select(&lists, sub.ID, nil,
		pq.Int64Array(listIDs), nil, models.SubscriptionStatusUnconfirmed, models.ListOptinDouble); err != nil {
		app.log.Printf("error fetching lists for opt-in: %s", pqErrMsg(err))
		return 0, err
	}

	// None.
	if len(lists) == 0 {
		return 0, nil
	}

	var (
		out      = subOptin{Subscriber: &sub, Lists: lists}
		qListIDs = url.Values{}
	)
	// Construct the opt-in URL with list IDs.
	for _, l := range out.Lists {
		qListIDs.Add("l", l.UUID)
	}
	out.OptinURL = fmt.Sprintf(app.constants.OptinURL, sub.UUID, qListIDs.Encode())

	// Send the e-mail.
	if err := app.sendNotification([]string{sub.Email},
		app.i18n.T("subscribers.optinSubject"), notifSubscriberOptin, out); err != nil {
		app.log.Printf("error sending opt-in e-mail: %s", err)
		return 0, err
	}
	return len(lists), nil
}

// sanitizeSQLExp does basic sanitisation on arbitrary
// SQL query expressions coming from the frontend.
func sanitizeSQLExp(q string) string {
	if len(q) == 0 {
		return ""
	}
	q = strings.TrimSpace(q)

	// Remove semicolon suffix.
	if q[len(q)-1] == ';' {
		q = q[:len(q)-1]
	}
	return q
}

// handleQuerySubscribers handles querying subscribers based on an arbitrary SQL expression.
func handleQueryFilterSubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pg  = getPagination(c.QueryParams(), 200)

		// Limit the subscribers to a particular list?
		listID, _ = strconv.Atoi(c.FormValue("list_id"))

		// The "WHERE ?" bit.
		selection      = c.QueryParam("selection")
		typ            = c.QueryParam("type")
		timeRange      = c.QueryParam("timerange")
		maxsubscribers = c.QueryParam("maxsubscribers")
		list           = c.QueryParam("list")
		campaignlist   = c.QueryParam("campaignlist")
		orderBy        = c.FormValue("order_by")
		order          = c.FormValue("order")
		out            subsWrap
	)

	listIDs := pq.Int64Array{}
	if listID < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.errorID"))
	} else if listID > 0 {
		listIDs = append(listIDs, int64(listID))
	}

	cond := ""

	if len(list) > 0 {
		ls := strings.Split(list, ",")
		var lsID pq.Int64Array
		for _, each := range ls {
			i, err := strconv.ParseInt(each, 10, 64)
			if err != nil {
				continue
			}
			lsID = append(lsID, i)
		}
		var IDs int64
		err := app.queries.QueryCheckListId.Get(&IDs, lsID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("List Id not found"))
		}
		if IDs == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("List Id not found"))
		}
		cond = cond + fmt.Sprintf(" AND subscribers.id in (select subscriber_id from subscriber_lists where list_id in (%s)) ", list)
	}

	campaignQuery := false
	if len(campaignlist) > 0 {
		ls := strings.Split(campaignlist, ",")
		var lsID pq.Int64Array
		for _, each := range ls {
			i, err := strconv.ParseInt(each, 10, 64)
			if err != nil {
				continue
			}
			lsID = append(lsID, i)
		}
		var IDs int64
		err := app.queries.QueryCheckCampaignListId.Get(&IDs, lsID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("Err Find Campaign List Id"))
		}
		if IDs == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("Campaign List Id not found"))
		}
		campaignQuery = true
	}

	if typ == "emailed" {
		if campaignQuery {
			cond = cond + " AND %s (SELECT distinct sl.subscriber_id FROM campaigns c, campaign_lists cl, subscriber_lists sl where c.id in (%s) AND c.id = cl.campaign_id and cl.list_id = sl.list_id AND %s) AND %s "
		} else {
			cond = cond + " AND %s (SELECT distinct id FROM subscribers where %s) "
		}
	} else {
		// There's an arbitrary query condition.
		if campaignQuery {
			cond = cond + " AND %s (SELECT distinct subscriber_id FROM %s where campaign_id in (%s) AND %s) "
		} else {
			cond = cond + " AND %s (SELECT distinct subscriber_id FROM %s where %s) "
		}
	}

	selectionCond := ""
	refTbl := ""
	timeRangeCond := ""

	if len(selection) > 0 {
		if selection == "include" {
			selectionCond = "EXISTS"
		} else if selection == "exclude" {
			selectionCond = "NOT EXISTS"
		}
	} else {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("Please choose a valid criterion!"))
	}

	if len(typ) > 0 {
		if typ == "opened" {
			refTbl = "campaign_views"
		} else if typ == "clicked" {
			refTbl = "link_clicks"
		} else if typ == "emailed" {
			refTbl = "emailed"
		}
	}

	timeStartedAt := ""
	if len(timeRange) > 0 {
		timeType := timeRange[len(timeRange)-1:]
		timeDetail := ""
		switch timeType {
		case "s":
			timeDetail = "second"
		case "m":
			timeDetail = "minute"
		case "h":
			timeDetail = "hour"
		case "d":
			timeDetail = "day"
		default:
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("Please choose a time range for the segmenting!"))
		}
		if len(timeDetail) > 0 {
			if typ == "emailed" {
				timeRangeCond = fmt.Sprintf(" last_email_sent between (now() - '%s %s'::interval) and now() ", timeRange[:len(timeRange)-1], timeDetail)
				if campaignQuery {
					timeStartedAt = fmt.Sprintf(" c.started_at between (now() - '%s %s'::interval) and now() ", timeRange[:len(timeRange)-1], timeDetail)
				}
			} else {
				timeRangeCond = fmt.Sprintf(" created_at between (now() - '%s %s'::interval) and now() and subscribers.id = %s.subscriber_id", timeRange[:len(timeRange)-1], timeDetail, refTbl)
			}
		}
	} else {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("Please choose a time range for the segmenting!"))
	}

	if selectionCond == "" || refTbl == "" || len(timeRange) == 0 {

	} else if typ == "emailed" {
		if campaignQuery {
			cond = fmt.Sprintf(cond, selectionCond, campaignlist, timeStartedAt, timeRangeCond)
		} else {
			cond = fmt.Sprintf(cond, selectionCond, timeRangeCond)
		}
	} else {
		if campaignQuery {
			cond = fmt.Sprintf(cond, selectionCond, refTbl, campaignlist, timeRangeCond)
		} else {
			cond = fmt.Sprintf(cond, selectionCond, refTbl, timeRangeCond)
		}
	}

	// Sort params.
	orderBy = "subscribers.id, subscribers.uuid"
	order = "asc"

	stmtCount := fmt.Sprintf(app.queries.QuerySubscribersOptimize, "COUNT(*) over () as total", cond, orderBy, order)
	stmt := fmt.Sprintf(app.queries.QuerySubscribersOptimize, "subscribers.*", cond, orderBy, order)

	app.log.Println("Query stmt: ", stmt)

	// Create a readonly transaction to prevent mutations.
	tx, err := app.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		app.log.Printf("error preparing subscriber query: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
	}
	defer tx.Rollback()

	// Run the query. stmt is the raw SQL query.
	var limitSubscribers int
	if len(maxsubscribers) > 0 {
		limitSubscribers, _ = strconv.Atoi(maxsubscribers)
	}
	limit := pg.Limit
	if limitSubscribers < limit && limitSubscribers > 0 {
		limit = limitSubscribers
	}

	if err := tx.Select(&out.Results, stmtCount, 0, 1); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}
	listName := fmt.Sprint("FLTRD-", time.Now().Unix())
	if len(out.Results) == 0 {
		err = errors.New("data not found")
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}
	if out.Results[0].Total == 0 {
		err = errors.New("data not found")
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}

	//Create List
	uu, err := uuid.NewV4()
	if err != nil {
		app.log.Printf("error generating UUID: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
	}
	var newListID int
	if err := app.queries.CreateList.Get(&newListID,
		uu.String(),
		listName,
		"private",
		"single",
		pq.StringArray(normalizeTags([]string{}))); err != nil {
		app.log.Printf("error creating list: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorCreating",
				"name", "{globals.terms.list}", "error", pqErrMsg(err)))
	}

	out.Query = ""
	if len(out.Results) == 0 {
		out.Results = make(models.Subscribers, 0)
		out.Id = newListID
		out.Name = listName
		return c.JSON(http.StatusOK, okResp{out})
	}

	out.SubscriberCount = out.Results[0].Total
	pgPerPage := pg.PerPage
	if out.Results[0].Total < 1000 {
		limit = limit
	} else if out.Results[0].Total < 10000 {
		pgPerPage = 1000
		limit = 1000
	} else if out.Results[0].Total < 100000 {
		pgPerPage = 10000
		limit = 10000
	} else if out.Results[0].Total < 1000000 {
		pgPerPage = 100000
		limit = 100000
	} else {
		pgPerPage = 1000000
		limit = 1000000
	}

	//if out.Results[0].Total > pg.PerPage && limitSubscribers > pg.PerPage {

	if out.Results[0].Total > pg.PerPage {
		if limitSubscribers > 0 && out.Results[0].Total > limitSubscribers {
			out.Results[0].Total = limitSubscribers
			out.SubscriberCount = limitSubscribers
		}
		counter := int(math.Ceil(float64(out.Results[0].Total) / float64(pgPerPage)))
		var wg = new(errgroup.Group)
		var countSleep = 0
		for i := 0; i < counter; i++ {
			wg.Go(func() error {
				offset := i * pgPerPage
				pgLimit := limit
				if (offset + pgLimit) > out.Results[0].Total {
					pgLimit = pgLimit - (offset + pgLimit - out.Results[0].Total)
				}
				var res models.Subscribers
				txG, err := app.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
				defer txG.Rollback()
				err = txG.Select(&res, stmt, offset, pgLimit)
				if err != nil {
					app.log.Println("go routine err: ", err)
					return err
				}

				countLimit := 0
				var IDs pq.Int64Array
				for _, each := range res {
					IDs = append(IDs, int64(each.ID))
					countLimit++
					if countLimit == 5000 {
						go func(iDs pq.Int64Array, newLsID int) {
							_, err = app.queries.AddSubscribersToLists.Exec(iDs, pq.Int64Array{int64(newLsID)})
							if err != nil {
								app.log.Printf("error updating list subscriptions: %v", err)
							}
						}(IDs, newListID)
						countLimit = 0
						IDs = nil
						time.Sleep(10 * time.Millisecond)
					}
				}

				go func(iDs pq.Int64Array, newLsID int) {
					_, err = app.queries.AddSubscribersToLists.Exec(iDs, pq.Int64Array{int64(newLsID)})
					if err != nil {
						app.log.Printf("error updating list subscriptions: %v", err)
					}
				}(IDs, newListID)
				return err
			})
			countSleep++
			if countSleep >= 10 {
				countSleep = 0
				time.Sleep(10 * time.Millisecond)
			}
			time.Sleep(10 * time.Millisecond)
		}

		if err = wg.Wait(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorFetching",
					"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
		}
	} else {
		var res models.Subscribers
		txG, err := app.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
		defer txG.Rollback()
		err = txG.Select(&res, stmt, 0, pg.PerPage)
		if err != nil {
			app.log.Println("go routine err: ", err)
			return err
		}
		var IDs pq.Int64Array
		for _, each := range res {
			IDs = append(IDs, int64(each.ID))
		}
		_, err = app.queries.AddSubscribersToLists.Exec(IDs, pq.Int64Array{int64(newListID)})
		if err != nil {
			app.log.Printf("error updating list subscriptions: %v", err)
		}
	}

	// Meta.
	out.Total = out.Results[0].Total
	out.Page = pg.Page
	out.PerPage = pg.PerPage
	out.Id = newListID
	out.Name = listName

	if len(out.Results) > 10 {
		out.Results = nil
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleQuerySubscribers handles querying subscribers based on an arbitrary SQL expression.
func handleQuerySmartFilterSubscribers(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		pg  = getPagination(c.QueryParams(), 200)

		// Limit the subscribers to a particular list?
		listID, _ = strconv.Atoi(c.FormValue("list_id"))

		// The "WHERE ?" bit.
		selection      = c.QueryParam("selection")
		typ            = c.QueryParam("type")
		timeRange      = c.QueryParam("timerange")
		maxsubscribers = c.QueryParam("maxsubscribers")
		orderBy        = c.FormValue("order_by")
		order          = c.FormValue("order")
		out            subsWrap
	)

	listIDs := pq.Int64Array{}
	if listID < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.errorID"))
	} else if listID > 0 {
		listIDs = append(listIDs, int64(listID))
	}

	cond := ""

	if typ == "emailed" {
		cond = cond + " AND %s (SELECT distinct id FROM subscribers where %s) "
	} else {
		// There's an arbitrary query condition.
		cond = cond + " AND %s (SELECT distinct subscriber_id FROM %s where %s) "
	}

	selectionCond := ""
	refTbl := ""
	timeRangeCond := ""

	if len(selection) > 0 {
		if selection == "include" {
			selectionCond = "EXISTS"
		} else if selection == "exclude" {
			selectionCond = "NOT EXISTS"
		}
	}

	listName := "SMART-"
	if len(typ) > 0 {
		if typ == "opened" {
			refTbl = "campaign_views"
			listName = listName + "OPENED"
		} else if typ == "clicked" {
			refTbl = "link_clicks"
			listName = listName + "CLICKED"
		} else if typ == "emailed" {
			refTbl = "emailed"
		}
	}

	if len(timeRange) > 0 {
		timeType := timeRange[len(timeRange)-1:]
		timeDetail := ""
		switch timeType {
		case "s":
			timeDetail = "second"
		case "m":
			timeDetail = "minute"
		case "h":
			timeDetail = "hour"
		case "d":
			timeDetail = "day"
		default:
			return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("Please ensure all the options are specified..."))
		}
		if len(timeDetail) > 0 {
			if typ == "emailed" {
				timeRangeCond = fmt.Sprintf(" last_email_sent between (now() - '%s %s'::interval) and now() ", timeRange[:len(timeRange)-1], timeDetail)
			} else {
				timeRangeCond = fmt.Sprintf(" created_at between (now() - '%s %s'::interval) and now() and subscribers.id = %s.subscriber_id", timeRange[:len(timeRange)-1], timeDetail, refTbl)
			}
		}
	}

	if selectionCond == "" || refTbl == "" || len(timeRange) == 0 {

	} else if typ == "emailed" {
		cond = fmt.Sprintf(cond, selectionCond, timeRangeCond)
	} else {
		cond = fmt.Sprintf(cond, selectionCond, refTbl, timeRangeCond)
	}

	// Sort params.
	orderBy = "subscribers.id, subscribers.uuid"
	order = "asc"

	stmtCount := fmt.Sprintf(app.queries.QuerySubscribersOptimize, "COUNT(*) over () as total", cond, orderBy, order)
	stmt := fmt.Sprintf(app.queries.QuerySubscribersOptimize, "subscribers.*", cond, orderBy, order)

	// Create a readonly transaction to prevent mutations.
	tx, err := app.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		app.log.Printf("error preparing subscriber query: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
	}
	defer tx.Rollback()

	// Run the query. stmt is the raw SQL query.
	var limitSubscribers int
	if len(maxsubscribers) > 0 {
		limitSubscribers, _ = strconv.Atoi(maxsubscribers)
	}
	limit := pg.Limit
	if limitSubscribers < limit && limitSubscribers > 0 {
		limit = limitSubscribers
	}

	if err := tx.Select(&out.Results, stmtCount, 0, 1); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
	}
	if len(out.Results) == 0 {
		lo.Println("INFO total list subscriber - ", listName, ": ", len(out.Results))
		return c.JSON(http.StatusOK, okResp{out})
	}
	if out.Results[0].Total == 0 {
		lo.Println("INFO total result list subscriber - ", listName, ": ", out.Results[0].Total)
		return c.JSON(http.StatusOK, okResp{out})
	}

	var listResult []models.List
	filterTempList := fmt.Sprintf(" AND name = '%s' ", listName)
	if err := db.Select(&listResult, fmt.Sprintf(app.queries.QueryLists, filterTempList, "created_at", "asc"), 0, pg.Offset, pg.Limit); err != nil {
		app.log.Printf("error fetching lists: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching",
				"name", "{globals.terms.lists}", "error", pqErrMsg(err)))
	}

	var newListID int
	if len(listResult) == 0 {
		//Create List
		uu, err := uuid.NewV4()
		if err != nil {
			app.log.Printf("error generating UUID: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorUUID", "error", err.Error()))
		}
		if err := app.queries.CreateList.Get(&newListID,
			uu.String(),
			listName,
			"private",
			"single",
			pq.StringArray(normalizeTags([]string{}))); err != nil {
			app.log.Printf("error creating list: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorCreating",
					"name", "{globals.terms.list}", "error", pqErrMsg(err)))
		}
	} else {
		newListID = listResult[0].ID
	}

	out.Query = ""
	if len(out.Results) == 0 {
		out.Results = make(models.Subscribers, 0)
		out.Id = newListID
		out.Name = listName
		return c.JSON(http.StatusOK, okResp{out})
	}

	out.SubscriberCount = out.Results[0].Total
	pgPerPage := pg.PerPage
	if out.Results[0].Total < 1000 {
		limit = limit
	} else if out.Results[0].Total < 10000 {
		pgPerPage = 1000
		limit = 1000
	} else if out.Results[0].Total < 100000 {
		pgPerPage = 10000
		limit = 10000
	} else if out.Results[0].Total < 1000000 {
		pgPerPage = 100000
		limit = 100000
	} else {
		pgPerPage = 1000000
		limit = 1000000
	}

	if out.Results[0].Total > pg.PerPage {
		if limitSubscribers > 0 && out.Results[0].Total > limitSubscribers {
			out.Results[0].Total = limitSubscribers
			out.SubscriberCount = limitSubscribers
		}
		counter := int(math.Ceil(float64(out.Results[0].Total) / float64(pgPerPage)))
		var wg = new(errgroup.Group)
		var countSleep = 0
		for i := 0; i < counter; i++ {
			wg.Go(func() error {
				offset := i * pgPerPage
				pgLimit := limit
				if (offset + pgLimit) > out.Results[0].Total {
					pgLimit = pgLimit - (offset + pgLimit - out.Results[0].Total)
				}
				var res models.Subscribers
				txG, err := app.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
				defer txG.Rollback()
				err = txG.Select(&res, stmt, offset, pgLimit)
				if err != nil {
					app.log.Println("go routine err: ", err)
					return err
				}

				countLimit := 0
				var IDs pq.Int64Array
				for _, each := range res {
					IDs = append(IDs, int64(each.ID))
					countLimit++
					if countLimit == 5000 {
						go func(iDs pq.Int64Array, newLsID int) {
							_, err = app.queries.AddSubscribersToLists.Exec(iDs, pq.Int64Array{int64(newLsID)})
							if err != nil {
								app.log.Printf("error updating list subscriptions: %v", err)
							}
						}(IDs, newListID)
						countLimit = 0
						IDs = nil
						time.Sleep(10 * time.Millisecond)
					}
				}

				go func(iDs pq.Int64Array, newLsID int) {
					_, err = app.queries.AddSubscribersToLists.Exec(iDs, pq.Int64Array{int64(newLsID)})
					if err != nil {
						app.log.Printf("error updating list subscriptions: %v", err)
					}
				}(IDs, newListID)
				return err
			})
			countSleep++
			if countSleep >= 10 {
				countSleep = 0
				time.Sleep(10 * time.Millisecond)
			}
			time.Sleep(10 * time.Millisecond)
		}

		if err = wg.Wait(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError,
				app.i18n.Ts("globals.messages.errorFetching",
					"name", "{globals.terms.subscribers}", "error", pqErrMsg(err)))
		}
	} else {
		var res models.Subscribers
		txG, err := app.db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
		defer txG.Rollback()
		err = txG.Select(&res, stmt, 0, pg.PerPage)
		if err != nil {
			app.log.Println("go routine err: ", err)
			return err
		}
		var IDs pq.Int64Array
		for _, each := range res {
			IDs = append(IDs, int64(each.ID))
		}
		_, err = app.queries.AddSubscribersToLists.Exec(IDs, pq.Int64Array{int64(newListID)})
		if err != nil {
			app.log.Printf("error updating list subscriptions: %v", err)
		}
	}

	// Meta.
	out.Total = out.Results[0].Total
	out.Page = pg.Page
	out.PerPage = pg.PerPage
	out.Id = newListID
	out.Name = listName

	if len(out.Results) > 10 {
		out.Results = nil
	}

	return c.JSON(http.StatusOK, okResp{out})
}
