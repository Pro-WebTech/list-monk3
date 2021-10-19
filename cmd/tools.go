package main

import (
	"net/http"
	"regexp"

	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/listmonk/models"

	"github.com/labstack/echo"
)

var (
	emailAbuseRegexp = regexp.MustCompile("^abuse@(.*)$")
	emailSpamRegexp  = regexp.MustCompile("^spam@(.*)$")
	emailRegexp      = regexp.MustCompile("^(.*)@(?:(yahoo|ymail|gmail|rocketmail|aol|aim|hotmail|outlook|live|msn)).(.*)$")
)

// handleDeleteTemplate handles template deletion.
func handleDeleteBlocklistSubscriber(c echo.Context) error {
	var (
		app = c.Get("app").(*App)
		out []models.Subscriber
	)

	rows, err := db.Query("SELECT * FROM subscribers")
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	if err != nil {
		app.log.Printf("error preparing subscriber query: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest,
			app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
	}
	var list []models.Subscriber
	for rows.Next() {
		var each models.Subscriber
		mergeRow(rows, &each)
		list = append(list, each)
	}

	for _, eachList := range list {
		if err := subimporter.ValidateEmail(eachList.Email); err != nil {
			stmt, err := db.Prepare("DELETE FROM subscribers where uuid = $1 AND email = $2")
			defer stmt.Close()
			_, err = stmt.Exec(eachList.UUID, eachList.Email)
			if err != nil {
				app.log.Printf("error delete subscriber: %v", err)
				return echo.NewHTTPError(http.StatusBadRequest,
					app.i18n.Ts("subscribers.errorPreparingQuery", "error", pqErrMsg(err)))
			}
			out = append(out, eachList)
		}
	}

	return c.JSON(http.StatusOK, okResp{out})
}
