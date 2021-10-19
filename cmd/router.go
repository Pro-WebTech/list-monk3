package main

import (
	"crypto/sha1"
	"github.com/knadh/listmonk/usecase/campaign"
	"github.com/knadh/listmonk/usecase/payment"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	"github.com/knadh/listmonk/dao/impl"
	"github.com/knadh/listmonk/usecase/admin"
	"github.com/knadh/listmonk/usecase/auth"
	"github.com/knadh/listmonk/usecase/public"
	"github.com/knadh/listmonk/utl/middleware/jwt"
	"github.com/knadh/listmonk/utl/secure"
)

func setupRouter(server *echo.Echo, db *sqlx.DB, lo *log.Logger) {

	userRepo := impl.NewUserDaoImpl()
	roleRepo := impl.NewRoleDaoImpl(lo)
	privilegeRepo := impl.NewPrivilegeDaoImpl(lo)
	settingsRepo := impl.NewSettingsDaoImpl(lo)
	stripePaymentHistory := impl.NewStripePaymentHistoryDaoImpl(lo)

	sec := secure.New(1, sha1.New())
	jwt := jwt.New(lo, db, roleRepo, privilegeRepo, "L1stm0nkr34lm", "HS256", 1555200)

	authSvc := auth.New(db, jwt, sec, userRepo)
	adminSvc := admin.New(db, settingsRepo)
	publicSvc := public.New(db, settingsRepo)
	paymentSvc := payment.New(db, settingsRepo, stripePaymentHistory)
	campaignSvc := campaign.New(db, settingsRepo)
	authHandler := setupAuthHandler(authSvc, lo)
	adminHandler := setupAdminHandler(adminSvc, lo)
	publicHandler := setupPublicHandler(publicSvc, lo)
	paymentHandler := setupPaymentHandler(paymentSvc, lo)
	campaignHandler := setupCampaignHandler(campaignSvc, lo)

	server.POST("/login", authHandler.login)

	public := server.Group("/public")
	public.GET("/asset/logo", publicHandler.getLogoUrl)
	public.GET("/email/plan", publicHandler.getEmailPlan)

	v1 := server.Group("/v1")
	v1.Use(jwt.MWFunc())

	v1.POST("/api/checkout/email/plan", paymentHandler.checkoutEmailPlan)
	bg := server.Group("", checkAuth)
	bg.POST("/webhook/stripe", paymentHandler.handlerStripe)

	v1.GET("/api/dashboard/charts", handleGetDashboardCharts)
	v1.GET("/api/dashboard/counts", handleGetDashboardCounts)

	v1.GET("/api/admin/platform/stats", adminHandler.getPlatformStats)
	v1.GET("/api/admin/platform/settings", adminHandler.getSettings)
	v1.PUT("/api/admin/platform/settings", adminHandler.updateSettings)

	v1.GET("/api/initsettings", handleGetSettings)
	v1.GET("/api/settings", handleGetSettings)
	v1.POST("/api/settings/proxy/graphql", handleSettingProxy)
	v1.PUT("/api/settings", handleUpdateSettings)
	v1.POST("/api/admin/reload", handleReloadApp)
	v1.GET("/api/logs", handleGetLogs)

	v1.GET("/api/subscribers/:id", handleGetSubscriber)
	v1.GET("/api/subscribers/:id/export", handleExportSubscriberData)
	v1.POST("/api/subscribers", handleCreateSubscriber)
	v1.PUT("/api/subscribers/:id", handleUpdateSubscriber)
	v1.POST("/api/subscribers/:id/optin", handleSubscriberSendOptin)
	v1.PUT("/api/subscribers/blocklist", handleBlocklistSubscribers)
	v1.PUT("/api/subscribers/:id/blocklist", handleBlocklistSubscribers)
	v1.PUT("/api/subscribers/lists/:id", handleManageSubscriberLists)
	v1.PUT("/api/subscribers/lists", handleManageSubscriberLists)
	v1.DELETE("/api/subscribers/:id", handleDeleteSubscribers)
	v1.DELETE("/api/subscribers", handleDeleteSubscribers)

	// Subscriber operations based on arbitrary SQL queries.
	// These aren't very REST-like.
	v1.POST("/api/subscribers/query/delete", handleDeleteSubscribersByQuery)
	v1.PUT("/api/subscribers/query/blocklist", handleBlocklistSubscribersByQuery)
	v1.PUT("/api/subscribers/query/lists", handleManageSubscriberListsByQuery)
	v1.GET("/api/subscribers", handleQuerySubscribers)
	v1.GET("/api/subscribers/export",
		middleware.GzipWithConfig(middleware.GzipConfig{Level: 9})(handleExportSubscribers))
	v1.GET("/api/subscribers/filter", handleQueryFilterSubscribers)
	v1.GET("/api/subscribers/smart/filter", handleQuerySmartFilterSubscribers)

	v1.GET("/api/import/subscribers", handleGetImportSubscribers)
	v1.GET("/api/import/subscribers/logs", handleGetImportSubscriberStats)
	v1.POST("/api/import/subscribers", handleImportSubscribers)
	v1.DELETE("/api/import/subscribers", handleStopImportSubscribers)

	v1.GET("/api/initlists", handleGetLists)
	v1.GET("/api/lists", handleGetLists)
	v1.GET("/api/lists/:id", handleGetLists)
	v1.POST("/api/lists", handleCreateList)
	v1.PUT("/api/lists/:id", handleUpdateList)
	v1.DELETE("/api/lists/:id", handleDeleteLists)

	v1.GET("/api/campaigns", handleGetCampaigns)
	v1.GET("/api/campaigns/running/stats", handleGetRunningCampaignStats)
	v1.GET("/api/campaigns/:id", handleGetCampaigns)
	v1.GET("/api/campaigns/:id/preview", handlePreviewCampaign)
	v1.POST("/api/campaigns/:id/preview", handlePreviewCampaign)
	v1.POST("/api/campaigns/:id/text", handlePreviewCampaign)
	v1.POST("/api/campaigns/:id/test", handleTestCampaign)
	v1.POST("/api/automation/sendemail", handleSendTestEmailCampaign)
	v1.POST("/api/campaigns", handleCreateCampaign)
	v1.PUT("/api/campaigns/:id", handleUpdateCampaign)
	v1.PUT("/api/campaigns/:id/status", handleUpdateCampaignStatus)
	v1.DELETE("/api/campaigns/:id", handleDeleteCampaign)
	v1.GET("/api/campaigns/list/messenger", campaignHandler.getListMessenger)

	v1.GET("/api/media", handleGetMedia)
	v1.POST("/api/media", handleUploadMedia)
	v1.DELETE("/api/media/:id", handleDeleteMedia)

	v1.GET("/api/templates", handleGetTemplates)
	v1.GET("/api/templates/:id", handleGetTemplates)
	v1.GET("/api/templates/:id/preview", handlePreviewTemplate)
	v1.POST("/api/templates/preview", handlePreviewTemplate)
	v1.POST("/api/templates", handleCreateTemplate)
	v1.PUT("/api/templates/:id", handleUpdateTemplate)
	v1.PUT("/api/templates/:id/default", handleTemplateSetDefault)
	v1.DELETE("/api/templates/:id", handleDeleteTemplate)
}
