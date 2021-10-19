package main

import (
	"fmt"
	"github.com/knadh/listmonk/models"
	"github.com/knadh/listmonk/usecase/admin"
	"log"
	"net/http"
	"sort"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx/types"
	"github.com/labstack/echo"
)

type serverConfig struct {
	Messengers   []string   `json:"messengers"`
	Langs        []i18nLang `json:"langs"`
	Lang         string     `json:"lang"`
	Update       *AppUpdate `json:"update"`
	NeedsRestart bool       `json:"needs_restart"`
}

// handleGetServerConfig returns general server config.
func handleGetServerConfig(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out = serverConfig{}
	)

	// Language list.
	langList, err := getI18nLangList(app.constants.Lang, app)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error loading language list: %v", err))
	}
	out.Langs = langList
	out.Lang = app.constants.Lang

	// Sort messenger names with `email` always as the first item.
	var names []string
	for name := range app.messengers {
		if name == emailMsgr {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	out.Messengers = append(out.Messengers, emailMsgr)
	out.Messengers = append(out.Messengers, names...)

	app.Lock()
	out.NeedsRestart = app.needsRestart
	out.Update = app.update
	app.Unlock()

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardCharts returns chart data points to render ont he dashboard.
func handleGetDashboardCharts(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out types.JSONText
	)

	if err := app.queries.GetDashboardCharts.Get(&out); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching", "name", "dashboard charts", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardCounts returns stats counts to show on the dashboard.
func handleGetDashboardCounts(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out types.JSONText
	)

	if err := app.queries.GetDashboardCounts.Get(&out); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching", "name", "dashboard stats", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleGetDashboardCounts returns stats counts to show on the dashboard.
func handleGetPlatformStats(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out types.JSONText
	)

	if err := app.queries.GetPlatformStats.Get(&out); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorFetching", "name", "platform stats", "error", pqErrMsg(err)))
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// handleReloadApp restarts the app.
func handleReloadApp(c echo.Context) error {
	app := c.Get("app").(*App)
	go func() {
		<-time.After(time.Millisecond * 500)
		app.sigChan <- syscall.SIGHUP
	}()
	return c.JSON(http.StatusOK, okResp{true})
}

type adminHandler struct {
	svc admin.Service
	lo  *log.Logger
}

func setupAdminHandler(svc admin.Service, lo *log.Logger) *adminHandler {
	return &adminHandler{
		svc: svc,
		lo:  lo,
	}
}

func (h *adminHandler) getPlatformStats(c echo.Context) error {
	r, err := h.svc.GetPlatformStats(h.lo)
	if err != nil {
		h.lo.Println("err getPlatformStats: ", err)
		return err
	}
	return c.JSON(http.StatusOK, r)
}

func (h *adminHandler) getSettings(c echo.Context) error {
	r, err := h.svc.GetPlatformSettings(h.lo)
	if err != nil {
		h.lo.Println("err getSettings: ", err)
		return err
	}
	return c.JSON(http.StatusOK, r)
}

func (h *adminHandler) updateSettings(c echo.Context) error {
	req := &models.SettingReq{}
	if err := c.Bind(req); err != nil {
		return err
	}
	r, err := h.svc.UpdatePlatformSettings(h.lo, req)
	if err != nil {
		h.lo.Println("err getSettings: ", err)
		return err
	}
	return c.JSON(http.StatusOK, r)
}
