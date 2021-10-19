package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/stripe/stripe-go/v72"

	"github.com/gofrs/uuid"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
	"golang.org/x/sync/errgroup"

	"github.com/go-playground/validator"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/goyesql/v2"
	goyesqlx "github.com/knadh/goyesql/v2/sqlx"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/listmonk/cmd/cors"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/manager"
	"github.com/knadh/listmonk/internal/media"
	"github.com/knadh/listmonk/internal/media/providers/filesystem"
	"github.com/knadh/listmonk/internal/media/providers/s3"
	"github.com/knadh/listmonk/internal/messenger"
	aws_email "github.com/knadh/listmonk/internal/messenger/aws-email"
	"github.com/knadh/listmonk/internal/messenger/email"
	"github.com/knadh/listmonk/internal/messenger/postback"
	"github.com/knadh/listmonk/internal/subimporter"
	"github.com/knadh/stuffbin"
	"github.com/labstack/echo"
	flag "github.com/spf13/pflag"
)

const (
	queryFilePath = "queries.sql"
)

// constants contains static, constant config values required by the app.
type constants struct {
	RootURL             string   `koanf:"root_url"`
	LogoURL             string   `koanf:"logo_url"`
	FaviconURL          string   `koanf:"favicon_url"`
	FromEmail           string   `koanf:"from_email"`
	NotifyEmails        []string `koanf:"notify_emails"`
	EnablePublicSubPage bool     `koanf:"enable_public_subscription_page"`
	Lang                string   `koanf:"lang"`
	DBBatchSize         int      `koanf:"batch_size"`
	Privacy             struct {
		IndividualTracking bool            `koanf:"individual_tracking"`
		AllowBlocklist     bool            `koanf:"allow_blocklist"`
		AllowExport        bool            `koanf:"allow_export"`
		AllowWipe          bool            `koanf:"allow_wipe"`
		Exportable         map[string]bool `koanf:"-"`
	} `koanf:"privacy"`
	AdminUsername []byte `koanf:"admin_username"`
	AdminPassword []byte `koanf:"admin_password"`
	ApiKey        string `koanf:"api_key"`

	UnsubURL                   string
	LinkTrackURL               string
	ViewTrackURL               string
	OptinURL                   string
	MessageURL                 string
	MediaProvider              string
	PlatformFileUrl            string `koanf:"platform_file_url"`
	EmailPlanFileUrl           string `koanf:"email_plan_file_url"`
	DelTempListSchedulerTime   string `koanf:"del_temp_list_scheduler_time"`
	PruningSchedulerTime       string `koanf:"pruning_scheduler_time"`
	SyncBlacklistSchedulerTime string `koanf:"sync_blacklist_scheduler_time"`
	SmartListFlag              string `koanf:"smart_list_flag"`
	EmailSentAllowed           int    `koanf:"allowed"`
	StripeKey                  string `koanf:"stripe_key"`
	Platform                   []PlatformConfig
	EmailPlan                  []EmailPlanConfig
}

type PlatformConfig struct {
	Platform string `json:"platform"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type EmailPlanConfig struct {
	PlanName  string `json:"plan_name"`
	PlanQty   string `json:"plan_qty"`
	PlanPrice string `json:"plan_price"`
}

type ProvidersConfig struct {
	Messenger string             `json:"messenger"`
	Name      string             `json:"name"`
	Product   []ProductProviders `json:"product"`
}

type ProductProviders struct {
	Name       string            `json:"name"`
	Connection []EmailConnection `json:"connection"`
}

type EmailConnection struct {
	UUID          string              `json:"uuid" map:"uuid"`
	Enabled       bool                `json:"enabled" map:"enabled"`
	Host          string              `json:"host" map:"host"`
	HelloHostname string              `json:"hello_hostname" map:"hello_hostname"`
	Port          int                 `json:"port" map:"port"`
	AuthProtocol  string              `json:"auth_protocol" map:"auth_protocol"`
	Username      string              `json:"username" map:"username"`
	Password      string              `json:"password,omitempty" map:"password"`
	EmailHeaders  []map[string]string `json:"email_headers" map:"email_headers"`
	MaxConns      int                 `json:"max_conns" map:"max_conns"`
	MaxMsgRetries int                 `json:"max_msg_retries" map:"max_msg_retries"`
	IdleTimeout   string              `json:"idle_timeout" map:"idle_timeout"`
	WaitTimeout   string              `json:"wait_timeout" map:"wait_timeout"`
	TLSEnabled    bool                `json:"tls_enabled" map:"tls_enabled"`
	TLSSkipVerify bool                `json:"tls_skip_verify" map:"tls_skip_verify"`
	Tag           string              `json:"tag" map:"tag"`
	Details       []EmailDetails      `json:"details" map:"details"`
}

type EmailDetails struct {
	Name        string `json:"name"`
	Summary     string `json:"summary"`
	Category    string `json:"category"`
	Status      string `json:"status"`
	Icon        string `json:"icon"`
	ProductCode string `json:"product_code"`
	Messenger   string `json:"messenger"`
}

func initFlags() {
	f := flag.NewFlagSet("config", flag.ContinueOnError)
	f.Usage = func() {
		// Register --help handler.
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	// Register the commandline flags.
	f.StringSlice("config", []string{"config.toml"},
		"path to one or more config files (will be merged in order)")
	f.Bool("install", false, "run first time installation")
	f.Bool("upgrade", false, "upgrade database to the current version")
	f.Bool("version", false, "current version of the build")
	f.Bool("new-config", false, "generate sample config file")
	f.String("static-dir", "", "(optional) path to directory with static files")
	f.String("i18n-dir", "", "(optional) path to directory with i18n language files")
	f.Bool("yes", false, "assume 'yes' to prompts, eg: during --install")
	if err := f.Parse(os.Args[1:]); err != nil {
		lo.Fatalf("error loading flags: %v", err)
	}

	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		lo.Fatalf("error loading config: %v", err)
	}
}

// initConfigFiles loads the given config files into the koanf instance.
func initConfigFiles(files []string, ko *koanf.Koanf) {
	for _, f := range files {
		lo.Printf("reading config: %s", f)
		if err := ko.Load(file.Provider(f), toml.Parser()); err != nil {
			if os.IsNotExist(err) {
				lo.Fatal("config file not found. If there isn't one yet, run --new-config to generate one.")
			}
			lo.Fatalf("error loadng config from file: %v.", err)
		}
	}
}

// initFileSystem initializes the stuffbin FileSystem to provide
// access to bunded static assets to the app.
func initFS(staticDir, i18nDir string) stuffbin.FileSystem {
	// Get the executable's path.
	path, err := os.Executable()
	if err != nil {
		lo.Fatalf("error getting executable path: %v", err)
	}

	// Load the static files stuffed in the binary.
	fs, err := stuffbin.UnStuff(path)
	if err != nil {
		// Running in local mode. Load local assets into
		// the in-memory stuffbin.FileSystem.
		lo.Printf("unable to initialize embedded filesystem: %v", err)
		lo.Printf("using local filesystem for static assets")
		files := []string{
			"config.toml.sample",
			"auth_model.conf",
			"queries.sql",
			"schema.sql",
			"static/email-templates",

			// Alias /static/public to /public for the HTTP fileserver.
			"static/public:/public",

			// The frontend app's static assets are aliased to /frontend
			// so that they are accessible at /frontend/js/* etc.
			// Alias all files inside dist/ and dist/frontend to frontend/*.
			"frontend/dist/favicon.png:/frontend/favicon.png",
			"frontend/dist/frontend:/frontend",
			"i18n:/i18n",
		}

		// If no external static dir is provided, try to load from the working dir.
		if staticDir == "" {
			files = append(files, "static/email-templates", "static/public:/public")
		}

		fs, err = stuffbin.NewLocalFS("./", files...)
		if err != nil {
			lo.Fatalf("failed to initialize local file for assets: %v", err)
		}
	}

	// Optional static directory to override static files.
	if staticDir != "" {
		lo.Printf("loading static files from: %v", staticDir)
		fStatic, err := stuffbin.NewLocalFS("/", []string{
			filepath.Join(staticDir, "/email-templates") + ":/static/email-templates",

			// Alias /static/public to /public for the HTTP fileserver.
			filepath.Join(staticDir, "/public") + ":/public",
		}...)
		if err != nil {
			lo.Fatalf("failed reading static directory: %s: %v", staticDir, err)
		}

		if err := fs.Merge(fStatic); err != nil {
			lo.Fatalf("error merging static directory: %s: %v", staticDir, err)
		}
	}

	// Optional static directory to override i18n language files.
	if i18nDir != "" {
		lo.Printf("loading i18n language files from: %v", i18nDir)
		fi18n, err := stuffbin.NewLocalFS("/", []string{i18nDir + ":/i18n"}...)
		if err != nil {
			lo.Fatalf("failed reading i18n directory: %s: %v", i18nDir, err)
		}

		if err := fs.Merge(fi18n); err != nil {
			lo.Fatalf("error merging i18n directory: %s: %v", i18nDir, err)
		}
	}
	return fs
}

// initDB initializes the main DB connection pool and parse and loads the app's
// SQL queries into a prepared query map.
func initDB() *sqlx.DB {
	var dbCfg dbConf
	if err := ko.Unmarshal("db", &dbCfg); err != nil {
		lo.Fatalf("error loading db config: %v", err)
	}

	lo.Printf("connecting to db: %s:%d/%s", dbCfg.Host, dbCfg.Port, dbCfg.DBName)
	db, err := connectDB(dbCfg)
	if err != nil {
		lo.Fatalf("error connecting to DB: %v", err)
	}
	return db
}

// initQueries loads named SQL queries from the queries file and optionally
// prepares them.
func initQueries(sqlFile string, db *sqlx.DB, fs stuffbin.FileSystem, prepareQueries bool) (goyesql.Queries, *Queries) {
	// Load SQL queries.
	qB, err := fs.Read(sqlFile)
	if err != nil {
		lo.Fatalf("error reading SQL file %s: %v", sqlFile, err)
	}
	qMap, err := goyesql.ParseBytes(qB)
	if err != nil {
		lo.Fatalf("error parsing SQL queries: %v", err)
	}

	if !prepareQueries {
		return qMap, nil
	}

	// Prepare queries.
	var q Queries
	if err := goyesqlx.ScanToStruct(&q, qMap, db.Unsafe()); err != nil {
		lo.Fatalf("error preparing SQL queries: %v", err)
	}

	return qMap, &q
}

// initSettings loads settings from the DB.
func initSettings(q *sqlx.Stmt) {
	var s types.JSONText
	if err := q.Get(&s); err != nil {
		lo.Fatalf("error reading settings from DB: %s", pqErrMsg(err))
	}

	// Setting keys are dot separated, eg: app.favicon_url. Unflatten them into
	// nested maps {app: {favicon_url}}.
	var out map[string]interface{}
	if err := json.Unmarshal(s, &out); err != nil {
		lo.Fatalf("error unmarshalling settings from DB: %v", err)
	}
	if err := ko.Load(confmap.Provider(out, "."), nil); err != nil {
		lo.Fatalf("error parsing settings from DB: %v", err)
	}
}

func initConstants() *constants {
	// Read constants.
	var c constants
	if err := ko.Unmarshal("app", &c); err != nil {
		lo.Fatalf("error loading app config: %v", err)
	}
	if err := ko.Unmarshal("privacy", &c.Privacy); err != nil {
		lo.Fatalf("error loading app config: %v", err)
	}

	c.RootURL = strings.TrimRight(c.RootURL, "/")
	c.Lang = ko.String("app.lang")
	c.Privacy.Exportable = maps.StringSliceToLookupMap(ko.Strings("privacy.exportable"))
	c.MediaProvider = ko.String("upload.provider")

	// Static URLS.
	// url.com/subscription/{campaign_uuid}/{subscriber_uuid}
	c.UnsubURL = fmt.Sprintf("%s/subscription/%%s/%%s", c.RootURL)

	// url.com/subscription/optin/{subscriber_uuid}
	c.OptinURL = fmt.Sprintf("%s/subscription/optin/%%s?%%s", c.RootURL)

	// url.com/link/{campaign_uuid}/{subscriber_uuid}/{link_uuid}
	c.LinkTrackURL = fmt.Sprintf("%s/link/%%s/%%s/%%s", c.RootURL)

	// url.com/link/{campaign_uuid}/{subscriber_uuid}
	c.MessageURL = fmt.Sprintf("%s/campaign/%%s/%%s", c.RootURL)

	// url.com/campaign/{campaign_uuid}/{subscriber_uuid}/px.png
	c.ViewTrackURL = fmt.Sprintf("%s/campaign/%%s/%%s/px.png", c.RootURL)
	return &c
}

// initI18n initializes a new i18n instance with the selected language map
// loaded from the filesystem. English is a loaded first as the default map
// and then the selected language is loaded on top of it so that if there are
// missing translations in it, the default English translations show up.
func initI18n(lang string, fs stuffbin.FileSystem) *i18n.I18n {
	i, ok, err := getI18nLang(lang, fs)
	if err != nil {
		if ok {
			lo.Println(err)
		} else {
			lo.Fatal(err)
		}
	}
	return i
}

// initCampaignManager initializes the campaign manager.
func initCampaignManager(q *Queries, cs *constants, app *App) *manager.Manager {
	campNotifCB := func(subject string, data interface{}) error {
		return app.sendNotification(cs.NotifyEmails, subject, notifTplCampaign, data)
	}

	if ko.Int("app.concurrency") < 1 {
		lo.Fatal("app.concurrency should be at least 1")
	}
	if ko.Int("app.message_rate") < 1 {
		lo.Fatal("app.message_rate should be at least 1")
	}

	return manager.New(manager.Config{
		BatchSize:             ko.Int("app.batch_size"),
		Concurrency:           ko.Int("app.concurrency"),
		MessageRate:           ko.Int("app.message_rate"),
		MaxSendErrors:         ko.Int("app.max_send_errors"),
		FromEmail:             cs.FromEmail,
		IndividualTracking:    ko.Bool("privacy.individual_tracking"),
		UnsubURL:              cs.UnsubURL,
		OptinURL:              cs.OptinURL,
		LinkTrackURL:          cs.LinkTrackURL,
		ViewTrackURL:          cs.ViewTrackURL,
		MessageURL:            cs.MessageURL,
		UnsubHeader:           ko.Bool("privacy.unsubscribe_header"),
		SlidingWindow:         ko.Bool("app.message_sliding_window"),
		SlidingWindowDuration: ko.Duration("app.message_sliding_window_duration"),
		SlidingWindowRate:     ko.Int("app.message_sliding_window_rate"),
	}, newManagerDB(q, lo), campNotifCB, app.i18n, lo)

}

func initProviders(app *App) {
	items := ko.Slices("providers")

	if len(items) == 0 {
		lo.Fatalf("no providers servers found in config")
	}

	for _, item := range items {
		messengerName := item.String("messenger")
		switch messengerName {
		case "email_api":
			for _, product := range item.Slices("product") {
				productName := product.String("name")
				switch productName {
				case "AWS":
					req := []EmailConnection{}
					for _, con := range product.Slices("connection") {
						emailCon := EmailConnection{}
						if err := con.UnmarshalWithConf("", &emailCon, koanf.UnmarshalConf{Tag: "json"}); err != nil {
							lo.Fatalf("error reading SMTP config: %v", err)
						}
						req = append(req, emailCon)
					}
					name := fmt.Sprintf("%v_%v", messengerName, productName)
					app.messengers[name] = initAWSMessenger(app.manager, name, req)
				}
			}
		case "email_smtp":
			for _, product := range item.Slices("product") {
				productName := product.String("name")
				name := fmt.Sprintf("%v_%v", messengerName, productName)
				app.messengers[name] = initSMTPMessenger(app.manager, name, product.Slices("connection"))
			}
		}
	}
}

func initAWSMessenger(m *manager.Manager, name string, cfg []EmailConnection) messenger.Messenger {
	var (
		configs = make([]aws_email.AWSConfig, 0, len(cfg))
	)

	if len(cfg) == 0 {
		lo.Fatalf("no SMTP servers found in config")
	}

	for _, item := range cfg {
		if !item.Enabled {
			continue
		}

		c := aws_email.AWSConfig{}
		if strings.Contains(item.Host, ".amazonaws.com") {
			c.Username = item.Username
			c.Password = item.Password

			hostPartitions := strings.Split(item.Host, ".")
			if len(hostPartitions) > 2 {
				c.Region = hostPartitions[1]
			}
		}
		configs = append(configs, c)
		lo.Printf("loaded email (AWS) messenger: %s@%s",
			item.Username, item.Host)
	}

	if len(configs) == 0 {
		lo.Fatalf("no AWS servers enabled in settings")
	}

	// Initialize the e-mail messenger with multiple SMTP servers.
	msgr, err := aws_email.New(lo, name, configs...)
	if err != nil {
		lo.Fatalf("error loading e-mail messenger: %v", err)
	}

	return msgr
}

func initStripe(q *Queries, cs *constants) {
	var value string
	if len(cs.StripeKey) == 0 {
		err := q.InsertStripeKeySettings.Get(&value)
		if err != nil {
			lo.Println("error insert key value email plan url into tbl settings: %v", err)
		}
		cs.StripeKey = value
	}
	stripe.Key = cs.StripeKey

	if len(cs.EmailPlanFileUrl) == 0 {
		err := q.InsertEmailPlanUrlSettings.Get(&value)
		if err != nil {
			lo.Println("error insert key value email plan url into tbl settings: %v", err)
		}
		cs.EmailPlanFileUrl = value
	}

	// Get the data
	resp, err := http.Get(cs.EmailPlanFileUrl)
	if err != nil {
		lo.Println("error download email plan config from Dropbox: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	buffer := bytes.NewBuffer(make([]byte, 0))
	part := make([]byte, 1024)
	var count int
	for {
		if count, err = reader.Read(part); err != nil {
			break
		}
		buffer.Write(part[:count])
	}
	if err != io.EOF {
		lo.Println("error reading email plan config from Dropbox: %v", err)
	} else {
		err = nil
	}

	emailPlanList := []EmailPlanConfig{}
	err = json.Unmarshal(buffer.Bytes(), &emailPlanList)
	if err != nil {
		lo.Println("error json unmarshal platform config from Dropbox: %v", err)
	}

	cs.EmailPlan = emailPlanList

	q.DeleteSettings.Exec("emailsent.plan")

	err = q.InsertSettings.Get(&value, "emailsent.plan", string(buffer.Bytes()))
	if err != nil {
		lo.Printf("error insert emailsent.plan: %v", err)
	}
}

// initImporter initializes the bulk subscriber importer.
func initImporter(q *Queries, db *sqlx.DB, app *App) *subimporter.Importer {
	return subimporter.New(
		subimporter.Options{
			UpsertStmt:         q.UpsertSubscriber.Stmt,
			BlocklistStmt:      q.UpsertBlocklistSubscriber.Stmt,
			SubsListStmt:       q.AddSubscribersToListsImports.Stmt,
			UpdateListDateStmt: q.UpdateListsDate.Stmt,
			NotifCB: func(subject string, data interface{}) error {
				app.sendNotification(app.constants.NotifyEmails, subject, notifTplImport, data)
				return nil
			},
		}, db.DB)
}

// initSMTPMessenger initializes the SMTP messenger.
func initSMTPMessenger(m *manager.Manager, name string, cfg []*koanf.Koanf) messenger.Messenger {
	var (
		servers = make([]email.Server, 0, len(cfg))
	)

	if len(cfg) == 0 {
		lo.Fatalf("no SMTP servers found in config")
	}

	// Load the config for multipme SMTP servers.
	for _, item := range cfg {
		if !item.Bool("enabled") {
			continue
		}

		// Read the SMTP config.
		var s email.Server
		if err := item.UnmarshalWithConf("", &s, koanf.UnmarshalConf{Tag: "json"}); err != nil {
			lo.Fatalf("error reading SMTP config: %v", err)
		}

		servers = append(servers, s)
		lo.Printf("loaded email (SMTP) messenger: %s@%s",
			item.String("username"), item.String("host"))
	}
	if len(servers) == 0 {
		lo.Fatalf("no SMTP servers enabled in settings")
	}

	// Initialize the e-mail messenger with multiple SMTP servers.
	msgr, err := email.New(lo, name, servers...)
	if err != nil {
		lo.Fatalf("error loading e-mail messenger: %v", err)
	}

	return msgr
}

// initPostbackMessengers initializes and returns all the enabled
// HTTP postback messenger backends.
func initPostbackMessengers(m *manager.Manager) []messenger.Messenger {
	items := ko.Slices("messengers")
	if len(items) == 0 {
		return nil
	}

	var out []messenger.Messenger
	for _, item := range items {
		if !item.Bool("enabled") {
			continue
		}

		// Read the Postback server config.
		var (
			name = item.String("name")
			o    postback.Options
		)
		if err := item.UnmarshalWithConf("", &o, koanf.UnmarshalConf{Tag: "json"}); err != nil {
			lo.Fatalf("error reading Postback config: %v", err)
		}

		// Initialize the Messenger.
		p, err := postback.New(o)
		if err != nil {
			lo.Fatalf("error initializing Postback messenger %s: %v", name, err)
		}
		out = append(out, p)

		lo.Printf("loaded Postback messenger: %s", name)
	}

	return out
}

// initMediaStore initializes Upload manager with a custom backend.
func initMediaStore() media.Store {
	switch provider := ko.String("upload.provider"); provider {
	case "s3":
		var o s3.Opts
		ko.Unmarshal("upload.s3", &o)
		up, err := s3.NewS3Store(o)
		if err != nil {
			lo.Fatalf("error initializing s3 upload provider %s", err)
		}
		lo.Println("media upload provider: s3")
		return up

	case "filesystem":
		var o filesystem.Opts

		ko.Unmarshal("upload.filesystem", &o)
		o.RootURL = ko.String("app.root_url")
		o.UploadPath = filepath.Clean(o.UploadPath)
		o.UploadURI = filepath.Clean(o.UploadURI)
		up, err := filesystem.NewDiskStore(o)
		if err != nil {
			lo.Fatalf("error initializing filesystem upload provider %s", err)
		}
		lo.Println("media upload provider: filesystem")
		return up

	default:
		lo.Fatalf("unknown provider. select filesystem or s3")
	}
	return nil
}

// initNotifTemplates compiles and returns e-mail notification templates that are
// used for sending ad-hoc notifications to admins and subscribers.
func initNotifTemplates(path string, fs stuffbin.FileSystem, i *i18n.I18n, cs *constants) *template.Template {
	// Register utility functions that the e-mail templates can use.
	funcs := template.FuncMap{
		"RootURL": func() string {
			return cs.RootURL
		},
		"LogoURL": func() string {
			return cs.LogoURL
		},
		"L": func() *i18n.I18n {
			return i
		},
	}

	tpl, err := stuffbin.ParseTemplatesGlob(funcs, fs, "/static/email-templates/*.html")
	if err != nil {
		lo.Fatalf("error parsing e-mail notif templates: %v", err)
	}
	return tpl
}

// initHTTPServer sets up and runs the app's main HTTP server and blocks forever.
func initHTTPServer(app *App) *echo.Echo {
	// Initialize the HTTP server.
	var srv = echo.New()
	srv.Use(echo.WrapMiddleware(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		MaxAge:           86400,
		AllowedMethods:   []string{"OPTIONS", "POST", "GET", "PUT", "DELETE", "PATCH", "HEAD"},
		AllowedHeaders:   []string{"*"},
		Debug:            true,
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: true,
	}).Handler))
	srv.HideBanner = true

	srv.Validator = &CustomValidator{V: validator.New()}
	srv.Binder = &CustomBinder{b: &echo.DefaultBinder{}}

	// Register app (*App) to be injected into all HTTP handlers.
	srv.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("app", app)
			return next(c)
		}
	})

	// Parse and load user facing templates.
	tpl, err := stuffbin.ParseTemplatesGlob(template.FuncMap{
		"L": func() *i18n.I18n {
			return app.i18n
		}}, app.fs, "/public/templates/*.html")
	if err != nil {
		lo.Fatalf("error parsing public templates: %v", err)
	}
	srv.Renderer = &tplRenderer{
		templates:  tpl,
		RootURL:    app.constants.RootURL,
		LogoURL:    app.constants.LogoURL,
		FaviconURL: app.constants.FaviconURL}

	// Initialize the static file server.
	fSrv := app.fs.FileServer()
	srv.GET("/public/*", echo.WrapHandler(fSrv))
	srv.GET("/frontend/*", echo.WrapHandler(fSrv))
	if ko.String("upload.provider") == "filesystem" {
		srv.Static(ko.String("upload.filesystem.upload_uri"),
			ko.String("upload.filesystem.upload_path"))
	}

	// Register all HTTP handlers.
	setupRouter(srv, app.db, app.log)
	registerHTTPHandlers(srv, app)

	// Start the server.
	go func() {
		if err := srv.Start(ko.String("app.address")); err != nil {
			if strings.Contains(err.Error(), "Server closed") {
				lo.Println("HTTP server shut down")
			} else {
				lo.Fatalf("error starting HTTP server: %v", err)
			}
		}
	}()

	return srv
}

func InitPlatform(q *Queries, cs *constants) {
	var value string
	if len(cs.PlatformFileUrl) == 0 {
		err := q.InsertPlatformUrlSettings.Get(&value)
		if err != nil {
			lo.Fatalf("error insert key value platform utl into tbl settings: %v", err)
		}
		cs.PlatformFileUrl = value
	}

	// Get the data
	resp, err := http.Get(cs.PlatformFileUrl)
	if err != nil {
		lo.Fatalf("error download platform config from Dropbox: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	buffer := bytes.NewBuffer(make([]byte, 0))
	part := make([]byte, 1024)
	var count int
	for {
		if count, err = reader.Read(part); err != nil {
			break
		}
		buffer.Write(part[:count])
	}
	if err != io.EOF {
		lo.Fatalf("error reading platform config from Dropbox: %v", err)
	} else {
		err = nil
	}

	platformList := []PlatformConfig{}
	err = json.Unmarshal(buffer.Bytes(), &platformList)
	if err != nil {
		lo.Fatalf("error json unmarshal platform config from Dropbox: %v", err)
	}

	cs.Platform = platformList
}

func InitSmartOpenList(cs *constants) {
	ticker := defaultTicker(time.Now().Add(time.Hour*time.Duration(1) +
		time.Minute*time.Duration(0) +
		time.Second*time.Duration(0)).Format("15:04:05"))
	for {
		select {
		case <-ticker.C:
			{
				lo.Println("running scheduler InitSmartOpenList")
				url := cs.RootURL + "/api/subscribers/smart/filter?selection=include&type=opened&timerange=2h&maxsubscribers=10000000"
				method := "GET"

				client := &http.Client{}
				req, err := http.NewRequest(method, url, nil)
				if err != nil {
					lo.Printf("Error! unable to InitSmartOpenList - create HTTP NewRequest[SmartFilterSubscribers]: %v", err)
					return
				}

				req.SetBasicAuth(string(cs.AdminUsername), string(cs.AdminPassword))
				req.Header.Set("Content-Type", "application/json")
				_, err = client.Do(req)
				if err != nil {
					lo.Printf("Error! InitSmartOpenList - call HTTP client[SmartFilterSubscribers]: %v", err)
					return
				}

				ticker = time.NewTicker(2 * time.Hour)
			}
		}
	}
}

func InitSmartClickList(cs *constants) {
	ticker := defaultTicker(time.Now().Add(time.Hour*time.Duration(1) +
		time.Minute*time.Duration(15) +
		time.Second*time.Duration(10)).Format("15:04:05"))
	for {
		select {
		case <-ticker.C:
			{

				lo.Println("running scheduler InitSmartClickList")
				url := cs.RootURL + "/api/subscribers/smart/filter?selection=include&type=clicked&timerange=2h&maxsubscribers=100000"
				method := "GET"

				client := &http.Client{}
				req, err := http.NewRequest(method, url, nil)
				if err != nil {
					lo.Printf("Error! unable to InitSmartClickList - create HTTP NewRequest[InitSmartClickList]: %v", err)
					return
				}

				req.SetBasicAuth(string(cs.AdminUsername), string(cs.AdminPassword))
				req.Header.Set("Content-Type", "application/json")
				_, err = client.Do(req)
				if err != nil {
					lo.Printf("Error! unable to InitSmartClickList - create HTTP NewRequest[InitSmartClickList]: %v", err)
					return
				}

				ticker = time.NewTicker(2 * time.Hour)
			}
		}
	}
}

func InitSmartEmailList(q *Queries, cs *constants) {
	if len(cs.SmartListFlag) > 0 {
		return
	}

	_, err := q.InsertSmartListFlag.Exec()
	if err != nil {
		lo.Printf("error insert key value flag smart list into tbl settings: %v", err)
	}

	emailsPrefix := []string{"YAHOO", "AOL", "HOTMAIL", "GMAIL"}
	for _, eachPrefix := range emailsPrefix {
		go runSmartEmailList(q, eachPrefix)
		time.Sleep(3 * time.Minute)
	}
}

func runSmartEmailList(q *Queries, prefixEmail string) {
	smartEmailName := fmt.Sprint("SMART-", prefixEmail)
	lo.Println("running InitSmartEmailList : ", smartEmailName)
	uu, err := uuid.NewV4()
	if err != nil {
		lo.Printf("error generating UUID: %v", err)
		return
	}
	var newListID int
	if err := q.CreateList.Get(&newListID,
		uu.String(),
		smartEmailName,
		"private",
		"single",
		pq.StringArray(normalizeTags([]string{}))); err != nil {
		lo.Println("error create new list ", smartEmailName, " [runSmartEmailList]: ", err)
		return
	}
	limitSubscribers := 200

	pg := pagination{
		Page:    1,
		PerPage: 200,
		Offset:  0,
		Limit:   200,
	}

	cond := ""
	switch prefixEmail {
	case "YAHOO":
		cond = " AND subscribers.status = 'enabled' AND (subscribers.email like '%@yahoo%' " +
			"OR subscribers.email like '%@ymail%' OR subscribers.email like '%@rocketmail%') "
	case "HOTMAIL":
		cond = " AND subscribers.status = 'enabled' AND (subscribers.email like '%@hotmail%' " +
			"OR subscribers.email like '%@outlook%' OR subscribers.email like '%@live%' " +
			"OR subscribers.email like '%@msn%' OR subscribers.email like '%@passport%' ) "
	case "GMAIL":
		cond = " AND subscribers.status = 'enabled' AND (subscribers.email like '%@gmail%' " +
			"OR subscribers.email like '%@googlemail%') "
	case "AOL":
		cond = " AND subscribers.status = 'enabled' AND (subscribers.email like '%@aol%' " +
			"OR subscribers.email like '%@aim%' OR subscribers.email like '%@love%' " +
			"OR subscribers.email like '%@ygm%' OR subscribers.email like '%@games%' " +
			"OR subscribers.email like '%@wow%') "
	}

	tx, err := db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		lo.Printf("error preparing subscriber query [runSmartEmailList]: %v", err)
		return
	}
	defer tx.Rollback()

	var subscribers models.Subscribers
	orderBy := "subscribers.id, subscribers.uuid"
	order := "asc"
	stmt := fmt.Sprintf(q.QuerySubscribers, cond, orderBy, order)
	listIDs := pq.Int64Array{}
	limit := pg.Limit
	if limitSubscribers < limit && limitSubscribers > 0 {
		limit = limitSubscribers
	}
	if err := tx.Select(&subscribers, stmt, listIDs, pg.Offset, limit); err != nil {
		lo.Println("error get subscribers [runSmartEmailList]: ", err)
		return
	}

	var IDs pq.Int64Array
	for _, each := range subscribers {
		IDs = append(IDs, int64(each.ID))
	}
	if len(subscribers) == 0 {
		lo.Println("INFO total list subscriber - ", smartEmailName, ": ", len(subscribers))
		return
	}
	pgPerPage := pg.PerPage
	if subscribers[0].Total < 1000 {
		limit = limit
	} else if subscribers[0].Total < 10000 {
		pgPerPage = 1000
		limit = 1000
	} else if subscribers[0].Total < 100000 {
		pgPerPage = 10000
		limit = 10000
	} else if subscribers[0].Total < 1000000 {
		pgPerPage = 100000
		limit = 100000
	} else {
		pgPerPage = 1000000
		limit = 1000000
	}
	limitSubscribers = subscribers[0].Total

	if subscribers[0].Total > pg.PerPage && limitSubscribers > pg.PerPage {
		if limitSubscribers > 0 && subscribers[0].Total > limitSubscribers {
			subscribers[0].Total = limitSubscribers
		}
		counter := int(math.Ceil(float64(subscribers[0].Total) / float64(pgPerPage)))
		var wg = new(errgroup.Group)
		var countSleep = 0
		var upLimit bool
		if pg.PerPage != pgPerPage {
			counter += 1
			upLimit = true
		}
		for i := 1; i < counter; i++ {
			wg.Go(func() error {
				curri := i
				offset := i * pgPerPage
				if upLimit && curri == 1 {
					offset = i * pg.PerPage
				}
				if curri > 1 {
					offset = (curri - 1) * pgPerPage
					offset += pg.PerPage
				}
				pgLimit := limit
				if (offset + pgLimit) > subscribers[0].Total {
					pgLimit = pgLimit - (offset + pgLimit - subscribers[0].Total)
				}
				var res models.Subscribers
				txG, err := db.BeginTxx(context.Background(), &sql.TxOptions{ReadOnly: true})
				defer txG.Rollback()
				err = txG.Select(&res, stmt, listIDs, offset, pgLimit)

				var ID pq.Int64Array
				if curri > 1 {
					for _, each := range res {
						IDs = append(IDs, int64(each.ID))
					}
					ID = IDs
				} else {
					for _, each := range res {
						ID = append(ID, int64(each.ID))
					}
				}
				if err != nil {
					lo.Println("go routine err: ", err)
					return err
				}

				_, err = q.AddSubscribersToLists.Exec(ID, pq.Int64Array{int64(newListID)})
				if err != nil {
					lo.Printf("error updating list subscriptions: %v", err)
				}
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
		}
	} else {
		_, err = q.AddSubscribersToLists.Exec(IDs, pq.Int64Array{int64(newListID)})
		if err != nil {
			lo.Println("error insert subscribers to list ", smartEmailName, " [runSmartEmailList]: ", err)
		}
	}
	lo.Println("INFO total list subscriber - ", smartEmailName, ": ", subscribers[0].Total)
}

func SchedulerSyncBlacklistSubscribers(q *Queries, cs *constants) {
	timeTIcker := getRandomTimeScheduler()
	lo.Println("timeTIcker SchedulerSyncBlacklistSubscribers: ", timeTIcker)
	ticker := defaultTicker(timeTIcker)
	for {
		select {
		case <-ticker.C:
			{
				ticker = time.NewTicker(2 * time.Hour)
				lo.Println("running scheduler SchedulerSyncBlacklistSubscribers")
				var listEvent []models.Event
				err := q.SyncAttributeBlocklistSubscribers.Select(&listEvent)
				if err == nil {
					if len(listEvent) > 150 {
						count := 0
						tempLsEvent := []models.Event{}
						for _, ls := range listEvent {
							tempLsEvent = append(tempLsEvent, ls)
							if count > 150 {
								go doBlacklistSubscriber(cs, tempLsEvent)
								count = 0
								tempLsEvent = []models.Event{}
								time.Sleep(1 * time.Second)
								continue
							}
							count++
						}
						go doBlacklistSubscriber(cs, tempLsEvent)
					} else {
						doBlacklistSubscriber(cs, listEvent)
					}
				} else {
					lo.Println("err SchedulerSyncBlacklistSubscribers: ", err)
				}
			}
		}
	}
}

func doBlacklistSubscriber(cs *constants, events []models.Event) {
	for _, eachPlatform := range cs.Platform {
		if eachPlatform.Platform == cs.RootURL {
			continue
		}
		url := eachPlatform.Platform + "/api/subscribers/query/blocklist"
		method := "PUT"

		reqBody := &subQueryReq{}
		for _, event := range events {
			reqBody.List = append(reqBody.List, SubQueryReq{
				SubscriberIDs:  event.ID,
				EventType:      event.EventType,
				EventReason:    event.EventReason,
				EventTimeStamp: event.EventTimeStamp,
			})
		}
		postBody, _ := json.Marshal(reqBody)

		client := &http.Client{}
		req, err := http.NewRequest(method, url, bytes.NewBuffer(postBody))
		if err != nil {
			lo.Printf("Error! unable to blacklist subscriber - create HTTP NewRequest[doBlacklistSubscriber]: %v", err)
			continue
		}

		req.SetBasicAuth(eachPlatform.Username, eachPlatform.Password)
		req.Header.Set("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			lo.Printf("Error! unable to blacklist subscriber - call HTTP client[doBlacklistSubscriber]: %v", err)
			continue
		}
		defer res.Body.Close()

		_, err = ioutil.ReadAll(res.Body)
		if err != nil {
			lo.Printf("Error! unable to blacklist subscriber - read response body[doBlacklistSubscriber]: %v", err)
			continue
		}

		time.Sleep(20 * time.Second)
	}
}

func SchedulerDeleteTblEvents(q *Queries) {
	timeTIcker := getRandomTimeScheduler()
	lo.Println("timeTIcker SchedulerDeleteTblEvents: ", timeTIcker)
	ticker := defaultTicker(timeTIcker)
	for {
		select {
		case <-ticker.C:
			{
				ticker = time.NewTicker(24 * time.Hour)
				lo.Println("running scheduler SchedulerDeleteTblEvents")
				res, err := q.DeleteEventsScheduler.Exec()
				if err != nil {
					lo.Printf(" Error SchedulerDeleteTblEvents: ", err)
				} else {
					total, _ := res.RowsAffected()
					lo.Print(" Finished delete tbl Event more than 2 days. Total rows: ", total)
				}
			}
		}
	}
}

func InitSchedulerDeleteTempList(q *Queries, cs *constants) {
	var value string
	if len(cs.DelTempListSchedulerTime) == 0 {
		err := q.InsertSettings.Get(&value, "app.del_temp_list_scheduler_time", fmt.Sprintf("\"%v\"", getRandomTimeScheduler()))
		if err != nil {
			lo.Printf("error insert key value scheduler time into tbl settings: %v", err)
		}
		cs.DelTempListSchedulerTime = value
	}
	lo.Println("starting scheduler delete temp list every day at ", cs.DelTempListSchedulerTime)

	ticker := defaultTicker(cs.DelTempListSchedulerTime)
	for {
		select {
		case <-ticker.C:
			{
				lo.Println("running scheduler delete temp list!")
				//to make sure the scheduler always runs at the same time
				ticker = defaultTicker(cs.DelTempListSchedulerTime)

				query := "FLTRD-%"
				if _, err := q.DeleteTempLists.Exec(query); err != nil {
					lo.Printf("error deleting temp lists: %v", err)
				}
			}
		}
	}
}

type CustomValidator struct {
	V *validator.Validate
}

// Validate validates the request
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.V.Struct(i)
}

type CustomBinder struct {
	b echo.Binder
}

func (cb *CustomBinder) Bind(i interface{}, c echo.Context) error {
	if err := cb.b.Bind(i, c); err != nil && err != echo.ErrUnsupportedMediaType {
		return err
	}
	return c.Validate(i)
}

func awaitReload(sigChan chan os.Signal, closerWait chan bool, closer func()) chan bool {
	// The blocking signal handler that main() waits on.
	out := make(chan bool)

	// Respawn a new process and exit the running one.
	respawn := func() {
		if err := syscall.Exec(os.Args[0], os.Args, os.Environ()); err != nil {
			lo.Fatalf("error spawning process: %v", err)
		}
		os.Exit(0)
	}

	// Listen for reload signal.
	go func() {
		for range sigChan {
			lo.Println("reloading on signal ...")

			go closer()
			select {
			case <-closerWait:
				// Wait for the closer to finish.
				respawn()
			case <-time.After(time.Second * 3):
				// Or timeout and force close.
				respawn()
			}
		}
	}()

	return out
}
