package main

import (
	"crypto/subtle"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

const (
	// stdInputMaxLen is the maximum allowed length for a standard input field.
	stdInputMaxLen = 200

	sortAsc  = "asc"
	sortDesc = "desc"
)

type okResp struct {
	Data interface{} `json:"data"`
}

type defaultResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
	Code    int         `json:"code"`
}

// pagination represents a query's pagination (limit, offset) related values.
type pagination struct {
	PerPage int `json:"per_page"`
	Page    int `json:"page"`
	Offset  int `json:"offset"`
	Limit   int `json:"limit"`
}

var (
	reUUID     = regexp.MustCompile("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$")
	reLangCode = regexp.MustCompile("[^a-zA-Z_0-9]")
)

// registerHandlers registers HTTP handlers.
func registerHTTPHandlers(e *echo.Echo, app *App) {
	// Group of private handlers with BasicAuth.
	var g *echo.Group

	if len(app.constants.AdminUsername) == 0 ||
		len(app.constants.AdminPassword) == 0 {
		g = e.Group("")
	} else {
		g = e.Group("", middleware.BasicAuth(basicAuth))
	}

	e.GET("/", handleIndexPage)
	e.GET("/api/health", handleHealthCheck)
	e.GET("/api/config", handleGetServerConfig)
	e.GET("/api/lang/:lang", handleGetI18nLang)
	//g.GET("/api/dashboard/charts", handleGetDashboardCharts)
	//g.GET("/api/dashboard/counts", handleGetDashboardCounts)
	//
	//g.GET("/api/settings", handleGetSettings)
	g.PUT("/api/settings", handleUpdateSettings)
	//g.POST("/api/admin/reload", handleReloadApp)
	//g.GET("/api/logs", handleGetLogs)
	//
	//g.GET("/api/subscribers/:id", handleGetSubscriber)
	//g.GET("/api/subscribers/:id/export", handleExportSubscriberData)
	//g.POST("/api/subscribers", handleCreateSubscriber)
	//g.PUT("/api/subscribers/:id", handleUpdateSubscriber)
	//g.POST("/api/subscribers/:id/optin", handleSubscriberSendOptin)
	//g.PUT("/api/subscribers/blocklist", handleBlocklistSubscribers)
	//g.PUT("/api/subscribers/:id/blocklist", handleBlocklistSubscribers)
	//g.PUT("/api/subscribers/lists/:id", handleManageSubscriberLists)
	//g.PUT("/api/subscribers/lists", handleManageSubscriberLists)
	//g.DELETE("/api/subscribers/:id", handleDeleteSubscribers)
	//g.DELETE("/api/subscribers", handleDeleteSubscribers)
	//
	//// Subscriber operations based on arbitrary SQL queries.
	//// These aren't very REST-like.
	//g.POST("/api/subscribers/query/delete", handleDeleteSubscribersByQuery)
	g.PUT("/api/subscribers/query/blocklist", handleBlocklistSubscribersByQuery)
	//g.PUT("/api/subscribers/query/lists", handleManageSubscriberListsByQuery)
	//g.GET("/api/subscribers", handleQuerySubscribers)
	//g.GET("/api/subscribers/export",
	//	middleware.GzipWithConfig(middleware.GzipConfig{Level: 9})(handleExportSubscribers))
	//g.GET("/api/subscribers/filter", handleQueryFilterSubscribers)
	//g.GET("/api/subscribers/smart/filter", handleQuerySmartFilterSubscribers)
	//
	//g.GET("/api/import/subscribers", handleGetImportSubscribers)
	//g.GET("/api/import/subscribers/logs", handleGetImportSubscriberStats)
	//g.POST("/api/import/subscribers", handleImportSubscribers)
	//g.DELETE("/api/import/subscribers", handleStopImportSubscribers)
	//
	//g.GET("/api/lists", handleGetLists)
	//g.GET("/api/lists/:id", handleGetLists)
	//g.POST("/api/lists", handleCreateList)
	//g.PUT("/api/lists/:id", handleUpdateList)
	//g.DELETE("/api/lists/:id", handleDeleteLists)
	//
	//g.GET("/api/campaigns", handleGetCampaigns)
	//g.GET("/api/campaigns/running/stats", handleGetRunningCampaignStats)
	//g.GET("/api/campaigns/:id", handleGetCampaigns)
	//g.GET("/api/campaigns/:id/preview", handlePreviewCampaign)
	//g.POST("/api/campaigns/:id/preview", handlePreviewCampaign)
	//g.POST("/api/campaigns/:id/content", handleCampaignContent)
	//g.POST("/api/campaigns/:id/text", handlePreviewCampaign)
	//g.POST("/api/campaigns/:id/test", handleTestCampaign)
	//g.POST("/api/automation/sendemail", handleSendTestEmailCampaign)
	//g.POST("/api/campaigns", handleCreateCampaign)
	//g.PUT("/api/campaigns/:id", handleUpdateCampaign)
	//g.PUT("/api/campaigns/:id/status", handleUpdateCampaignStatus)
	//g.DELETE("/api/campaigns/:id", handleDeleteCampaign)
	//
	//g.GET("/api/media", handleGetMedia)
	//g.POST("/api/media", handleUploadMedia)
	//g.DELETE("/api/media/:id", handleDeleteMedia)
	//
	//g.GET("/api/templates", handleGetTemplates)
	//g.GET("/api/templates/:id", handleGetTemplates)
	//g.GET("/api/templates/:id/preview", handlePreviewTemplate)
	//g.POST("/api/templates/preview", handlePreviewTemplate)
	//g.POST("/api/templates", handleCreateTemplate)
	//g.PUT("/api/templates/:id", handleUpdateTemplate)
	//g.PUT("/api/templates/:id/default", handleTemplateSetDefault)
	//g.DELETE("/api/templates/:id", handleDeleteTemplate)

	// Static admin views.
	e.GET("/dashboard", handleIndexPage)
	e.GET("/lists", handleIndexPage)
	e.GET("/lists/forms", handleIndexPage)
	e.GET("/subscribers", handleIndexPage)
	e.GET("/subscribers/lists/:listID", handleIndexPage)
	e.GET("/subscribers/import", handleIndexPage)
	e.GET("/campaigns", handleIndexPage)
	e.GET("/campaigns/new", handleIndexPage)
	e.GET("/campaigns/media", handleIndexPage)
	e.GET("/campaigns/templates", handleIndexPage)
	e.GET("/campaigns/:campignID", handleIndexPage)
	e.GET("/settings", handleIndexPage)
	e.GET("/settings/logs", handleIndexPage)
	e.GET("/settings/billing", handleIndexPage)
	e.GET("/success", handleIndexPage)

	// Public subscriber facing views.
	e.GET("/subscription/form", handleSubscriptionFormPage)
	e.POST("/subscription/form", handleSubscriptionForm)
	e.GET("/subscription/:campUUID/:subUUID", noIndex(validateUUID(subscriberExists(handleSubscriptionPage),
		"campUUID", "subUUID")))
	e.POST("/subscription/:campUUID/:subUUID", validateUUID(subscriberExists(handleSubscriptionPage),
		"campUUID", "subUUID"))
	e.GET("/subscription/optin/:subUUID", noIndex(validateUUID(subscriberExists(handleOptinPage), "subUUID")))
	e.POST("/subscription/optin/:subUUID", validateUUID(subscriberExists(handleOptinPage), "subUUID"))
	e.POST("/subscription/export/:subUUID", validateUUID(subscriberExists(handleSelfExportSubscriberData),
		"subUUID"))
	e.POST("/subscription/wipe/:subUUID", validateUUID(subscriberExists(handleWipeSubscriberData),
		"subUUID"))
	e.GET("/link/:linkUUID/:campUUID/:subUUID", noIndex(validateUUID(handleLinkRedirect,
		"linkUUID", "campUUID", "subUUID")))
	e.GET("/link/:linkUUID/:campUUID/:subUUID/:email", noIndex(validateUUID(handleLinkRedirect,
		"linkUUID", "campUUID", "subUUID")))
	e.GET("/link/check", handleLinkCheck)
	e.GET("/campaign/:campUUID/:subUUID", noIndex(validateUUID(handleViewCampaignMessage,
		"campUUID", "subUUID")))
	e.GET("/campaign/:campUUID/:subUUID/px.png", noIndex(validateUUID(handleRegisterCampaignView,
		"campUUID", "subUUID")))
	// Public health API endpoint.
	e.GET("/health", handleHealthCheck)

	bg := e.Group("", checkAuth)
	bg.POST("/webhook/amazon", handleAwsEvents)
	bg.POST("/webhook/mailparser", handleEmailParserEvents)
	bg.POST("/webhook/postmarkapp", handlePostMarkAppEvents)
}

// handleIndex is the root handler that renders the Javascript frontend.
func handleIndexPage(c echo.Context) error {
	app := c.Get("app").(*App)

	b, err := app.fs.Read("/frontend/index.html")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	c.Response().Header().Set("Content-Type", "text/html")
	return c.String(http.StatusOK, string(b))
}

// handleHealthCheck is a healthcheck endpoint that returns a 200 response.
func handleHealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, okResp{true})
}

// basicAuth middleware does an HTTP BasicAuth authentication for admin handlers.
func basicAuth(username, password string, c echo.Context) (bool, error) {
	app := c.Get("app").(*App)

	// Auth is disabled.
	if len(app.constants.AdminUsername) == 0 &&
		len(app.constants.AdminPassword) == 0 {
		return true, nil
	}

	if subtle.ConstantTimeCompare([]byte(username), app.constants.AdminUsername) == 1 &&
		subtle.ConstantTimeCompare([]byte(password), app.constants.AdminPassword) == 1 {
		return true, nil
	}
	return false, nil
}

// validateUUID middleware validates the UUID string format for a given set of params.
func validateUUID(next echo.HandlerFunc, params ...string) echo.HandlerFunc {
	return func(c echo.Context) error {
		app := c.Get("app").(*App)

		for _, p := range params {
			if !reUUID.MatchString(c.Param(p)) {
				return c.Render(http.StatusBadRequest, tplMessage,
					makeMsgTpl(app.i18n.T("public.errorTitle"), "",
						app.i18n.T("globals.messages.invalidUUID")))
			}
		}
		return next(c)
	}
}

// subscriberExists middleware checks if a subscriber exists given the UUID
// param in a request.
func subscriberExists(next echo.HandlerFunc, params ...string) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			app     = c.Get("app").(*App)
			subUUID = c.Param("subUUID")
		)

		var exists bool
		if err := app.queries.SubscriberExists.Get(&exists, 0, subUUID); err != nil {
			app.log.Printf("error checking subscriber existence: %v", err)
			return c.Render(http.StatusInternalServerError, tplMessage,
				makeMsgTpl(app.i18n.T("public.errorTitle"), "",
					app.i18n.T("public.errorProcessingRequest")))
		}

		if !exists {
			return c.Render(http.StatusNotFound, tplMessage,
				makeMsgTpl(app.i18n.T("public.notFoundTitle"), "",
					app.i18n.T("public.subNotFound")))
		}
		return next(c)
	}
}

// noIndex adds the HTTP header requesting robots to not crawl the page.
func noIndex(next echo.HandlerFunc, params ...string) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("X-Robots-Tag", "noindex")
		return next(c)
	}
}

// getPagination takes form values and extracts pagination values from it.
func getPagination(q url.Values, perPage int) pagination {
	var (
		page, _ = strconv.Atoi(q.Get("page"))
		pp      = q.Get("per_page")
	)

	if pp == "all" {
		// No limit.
		perPage = 0
	} else {
		ppi, _ := strconv.Atoi(pp)
		if ppi > 0 {
			perPage = ppi
		}
	}

	if page < 1 {
		page = 0
	} else {
		page--
	}

	return pagination{
		Page:    page + 1,
		PerPage: perPage,
		Offset:  page * perPage,
		Limit:   perPage,
	}
}

// copyEchoCtx returns a copy of the the current echo.Context in a request
// with the given params set for the active handler to proxy the request
// to another handler without mutating its context.
func copyEchoCtx(c echo.Context, params map[string]string) echo.Context {
	var (
		keys = make([]string, 0, len(params))
		vals = make([]string, 0, len(params))
	)
	for k, v := range params {
		keys = append(keys, k)
		vals = append(vals, v)
	}

	b := c.Echo().NewContext(c.Request(), c.Response())
	b.Set("app", c.Get("app").(*App))
	b.SetParamNames(keys...)
	b.SetParamValues(vals...)
	return b
}
