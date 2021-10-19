package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Queries contains all prepared SQL queries.
type Queries struct {
	GetDashboardCharts *sqlx.Stmt `query:"get-dashboard-charts"`
	GetDashboardCounts *sqlx.Stmt `query:"get-dashboard-counts"`
	GetPlatformStats   *sqlx.Stmt `query:"get-platform-stats"`

	InsertSubscriber                  *sqlx.Stmt `query:"insert-subscriber"`
	UpsertSubscriber                  *sqlx.Stmt `query:"upsert-subscriber"`
	UpsertBlocklistSubscriber         *sqlx.Stmt `query:"upsert-blocklist-subscriber"`
	GetSubscriber                     *sqlx.Stmt `query:"get-subscriber"`
	GetSubscribersByEmails            *sqlx.Stmt `query:"get-subscribers-by-emails"`
	GetSubscriberLists                *sqlx.Stmt `query:"get-subscriber-lists"`
	GetSubscriberListsLazy            *sqlx.Stmt `query:"get-subscriber-lists-lazy"`
	SubscriberExists                  *sqlx.Stmt `query:"subscriber-exists"`
	UpdateSubscriber                  *sqlx.Stmt `query:"update-subscriber"`
	BlocklistSubscribers              *sqlx.Stmt `query:"blocklist-subscribers"`
	AddSubscribersToLists             *sqlx.Stmt `query:"add-subscribers-to-lists"`
	DeleteSubscriptions               *sqlx.Stmt `query:"delete-subscriptions"`
	ConfirmSubscriptionOptin          *sqlx.Stmt `query:"confirm-subscription-optin"`
	UnsubscribeSubscribersFromLists   *sqlx.Stmt `query:"unsubscribe-subscribers-from-lists"`
	DeleteSubscribers                 *sqlx.Stmt `query:"delete-subscribers"`
	Unsubscribe                       *sqlx.Stmt `query:"unsubscribe"`
	ExportSubscriberData              *sqlx.Stmt `query:"export-subscriber-data"`
	AddSubscribersToListsImports      *sqlx.Stmt `query:"add-subscribers-to-lists-imports"`
	FindSubscribersIdByEmail          *sqlx.Stmt `query:"query-get-subscribers-id-by-email"`
	SyncAttributeBlocklistSubscribers *sqlx.Stmt `query:"sync-attribute-blocklist-subscribers"`
	QueryCheckListId                  *sqlx.Stmt `query:"query-check-list-id"`
	QueryCheckCampaignListId          *sqlx.Stmt `query:"query-check-campaign-list-id"`

	// Non-prepared arbitrary subscriber queries.
	QuerySubscribers                       string `query:"query-subscribers"`
	QuerySubscribersOptimize               string `query:"query-subscribers-optimize"`
	QuerySubscribersForExport              string `query:"query-subscribers-for-export"`
	QuerySubscribersTpl                    string `query:"query-subscribers-template"`
	QuerySubscribersTplNew                 string `query:"query-subscribers-template-new"`
	DeleteSubscribersByQuery               string `query:"delete-subscribers-by-query"`
	AddSubscribersToListsByQuery           string `query:"add-subscribers-to-lists-by-query"`
	BlocklistSubscribersByQuery            string `query:"blocklist-subscribers-by-query"`
	DeleteSubscriptionsByQuery             string `query:"delete-subscriptions-by-query"`
	UnsubscribeSubscribersFromListsByQuery string `query:"unsubscribe-subscribers-from-lists-by-query"`
	InsertAttributeBlocklistSubscribers    string `query:"insert-attribute-blocklist-subscribers"`

	CreateList      *sqlx.Stmt `query:"create-list"`
	QueryLists      string     `query:"query-lists"`
	GetLists        *sqlx.Stmt `query:"get-lists"`
	GetListsByOptin *sqlx.Stmt `query:"get-lists-by-optin"`
	UpdateList      *sqlx.Stmt `query:"update-list"`
	UpdateListsDate *sqlx.Stmt `query:"update-lists-date"`
	DeleteLists     *sqlx.Stmt `query:"delete-lists"`
	DeleteTempLists *sqlx.Stmt `query:"delete-temp-lists"`

	CreateCampaign              *sqlx.Stmt `query:"create-campaign"`
	QueryCampaigns              string     `query:"query-campaigns"`
	GetCampaign                 *sqlx.Stmt `query:"get-campaign"`
	GetCampaignForPreview       *sqlx.Stmt `query:"get-campaign-for-preview"`
	GetCampaignStats            *sqlx.Stmt `query:"get-campaign-stats"`
	GetCampaignStatus           *sqlx.Stmt `query:"get-campaign-status"`
	NextCampaigns               *sqlx.Stmt `query:"next-campaigns"`
	NextCampaignSubscribers     *sqlx.Stmt `query:"next-campaign-subscribers"`
	GetOneCampaignSubscriber    *sqlx.Stmt `query:"get-one-campaign-subscriber"`
	UpdateCampaign              *sqlx.Stmt `query:"update-campaign"`
	UpdateCampaignStatus        *sqlx.Stmt `query:"update-campaign-status"`
	UpdateCampaignCounts        *sqlx.Stmt `query:"update-campaign-counts"`
	RegisterCampaignView        *sqlx.Stmt `query:"register-campaign-view"`
	DeleteCampaign              *sqlx.Stmt `query:"delete-campaign"`
	UpdateLastEmailSent         *sqlx.Stmt `query:"update-last-email-sent"`
	UpdateLastEmailOpen         *sqlx.Stmt `query:"update-last-email-open"`
	UpdateLastEmailClicked      *sqlx.Stmt `query:"update-last-email-clicked"`
	UpdateSendCampaignCounts    *sqlx.Stmt `query:"update-send-campaign-counts"`
	UpdateSettingCampaignCounts *sqlx.Stmt `query:"update-setting-campaign-counts"`
	DeleteEventsScheduler       *sqlx.Stmt `query:"delete-events-scheduler"`
	ValidateStartCampaign       *sqlx.Stmt `query:"validate-start-campaign-by-max-email"`

	InsertMedia *sqlx.Stmt `query:"insert-media"`
	GetMedia    *sqlx.Stmt `query:"get-media"`
	DeleteMedia *sqlx.Stmt `query:"delete-media"`

	CreateTemplate     *sqlx.Stmt `query:"create-template"`
	GetTemplates       *sqlx.Stmt `query:"get-templates"`
	UpdateTemplate     *sqlx.Stmt `query:"update-template"`
	SetDefaultTemplate *sqlx.Stmt `query:"set-default-template"`
	DeleteTemplate     *sqlx.Stmt `query:"delete-template"`

	CreateLink        *sqlx.Stmt `query:"create-link"`
	RegisterLinkClick *sqlx.Stmt `query:"register-link-click"`

	GetSettings                *sqlx.Stmt `query:"get-settings"`
	UpdateSettings             *sqlx.Stmt `query:"update-settings"`
	UpdateSettingsNew          *sqlx.Stmt `query:"update-settings-new"`
	DeleteSettings             *sqlx.Stmt `query:"delete-settings"`
	InsertPlatformUrlSettings  *sqlx.Stmt `query:"create-setting-platform-url"`
	InsertEmailPlanUrlSettings *sqlx.Stmt `query:"create-setting-email-plan-url"`
	InsertStripeKeySettings    *sqlx.Stmt `query:"create-setting-stripe-key"`
	InsertSettings             *sqlx.Stmt `query:"create-settings"`
	InsertSmartListFlag        *sqlx.Stmt `query:"create-smart-list-flag"`

	// GetStats *sqlx.Stmt `query:"get-stats"`
}

// dbConf contains database config required for connecting to a DB.
type dbConf struct {
	Host        string        `koanf:"host"`
	Port        int           `koanf:"port"`
	User        string        `koanf:"user"`
	Password    string        `koanf:"password"`
	DBName      string        `koanf:"database"`
	SSLMode     string        `koanf:"ssl_mode"`
	MaxOpen     int           `koanf:"max_open"`
	MaxIdle     int           `koanf:"max_idle"`
	MaxLifetime time.Duration `koanf:"max_lifetime"`
}

// connectDB initializes a database connection.
func connectDB(c dbConf) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres",
		fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(c.MaxOpen)
	db.SetMaxIdleConns(c.MaxIdle)
	db.SetConnMaxLifetime(c.MaxLifetime)
	return db, nil
}

// compileSubscriberQueryTpl takes a arbitrary WHERE expressions
// to filter subscribers from the subscribers table and prepares a query
// out of it using the raw `query-subscribers-template` query template.
// While doing this, a readonly transaction is created and the query is
// dry run on it to ensure that it is indeed readonly.
func (q *Queries) compileSubscriberQueryTpl(exp string, db *sqlx.DB) (string, error) {
	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Perform the dry run.
	if exp != "" {
		exp = " AND " + exp
	}
	stmt := fmt.Sprintf(q.QuerySubscribersTpl, exp)
	if _, err := tx.Exec(stmt, true, pq.Int64Array{}); err != nil {
		return "", err
	}

	return stmt, nil
}

func (q *Queries) newCompileSubscriberQueryTpl(exp string, listIDs []int64, db *sqlx.DB) (string, error) {
	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Perform the dry run.
	if exp != "" {
		exp = " AND " + exp
	}
	stmt := fmt.Sprintf(q.QuerySubscribersTplNew, exp)
	if _, err := tx.Exec(stmt, pq.Int64Array(listIDs)); err != nil {
		return "", err
	}

	return stmt, nil
}

// compileSubscriberQueryTpl takes a arbitrary WHERE expressions and a subscriber
// query template that depends on the filter (eg: delete by query, blocklist by query etc.)
// combines and executes them.
func (q *Queries) execSubscriberQueryTpl(exp, tpl string, listIDs []int64, db *sqlx.DB, args ...interface{}) error {
	// Perform a dry run.
	filterExp, err := q.compileSubscriberQueryTpl(exp, db)
	if err != nil {
		return err
	}

	if len(listIDs) == 0 {
		listIDs = pq.Int64Array{}
	}
	// First argument is the boolean indicating if the query is a dry run.
	a := append([]interface{}{false, pq.Int64Array(listIDs)}, args...)
	if _, err := db.Exec(fmt.Sprintf(tpl, filterExp), a...); err != nil {
		return err
	}

	return nil
}

// compileSubscriberQueryTpl takes a arbitrary WHERE expressions and a subscriber
// query template that depends on the filter (eg: delete by query, blocklist by query etc.)
// combines and executes them.
func (q *Queries) newExecSubscriberQueryTpl(exp, tpl string, listIDs []int64, db *sqlx.DB, args ...interface{}) error {
	// Perform a dry run.
	filterExp, err := q.newCompileSubscriberQueryTpl(exp, listIDs, db)
	if err != nil {
		return err
	}

	if len(listIDs) == 0 {
		listIDs = pq.Int64Array{}
	}
	// First argument is the boolean indicating if the query is a dry run.
	a := append([]interface{}{pq.Int64Array(listIDs)}, args...)
	if _, err := db.Exec(fmt.Sprintf(tpl, filterExp), a...); err != nil {
		return err
	}

	return nil
}
