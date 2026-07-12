package router

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/daniel-oluwadunsin/nombasub/internal/handlers"
	"github.com/daniel-oluwadunsin/nombasub/internal/middleware"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/services"
	"github.com/gin-gonic/gin"
)

func New(
	cfg *config.Config,
	handlers *handlers.Handler,
	rc *repositories.Container,
	sc *services.Container,
) *gin.Engine {
	r := gin.New()
	authenticate := middleware.Authenticate(cfg, rc.TenantRepository, sc.AuthService)
	r.Use(
		gin.Logger(),
		gin.Recovery(),
		middleware.SecurityHeaders(),
		middleware.BodyLimit(1<<20), // 1 MiB
		middleware.CORS(),
	)

	r.GET("/health", handlers.Health)
	r.NoRoute(handlers.NoRoute)

	authenticatePortal := middleware.AuthenticatePortal(cfg, rc.PortalSessionRepository)

	// Strict throttle for unauthenticated, abuse-prone endpoints (credential
	// stuffing, code brute-force, email flooding).
	sensitiveRateLimit := middleware.RateLimit(10)

	auth := r.Group("/auth")
	{
		auth.POST("/register", sensitiveRateLimit, handlers.RegisterTenant)
		auth.POST("/login", sensitiveRateLimit, handlers.LoginTenant)
		auth.POST("/sign-out", authenticate, handlers.SignOutTenant)
		auth.GET("/settings", authenticate, handlers.GetTenantSettings)
		auth.PATCH("/settings", authenticate, handlers.UpdateTenantSettings)
		auth.POST("/api-key/rotate", authenticate, handlers.RotateTenantApiKey)
		auth.POST("/webhook-secret/rotate", authenticate, handlers.RotateTenantWebhookSecret)
		auth.POST("/change-password", authenticate, handlers.ChangeTenantPassword)
		auth.POST("/set-webhook-url", authenticate, handlers.SetWebhookUrl)
	}

	webhook := r.Group("/webhook")
	{
		webhook.POST("/nomba", handlers.HandleWebhook)
		webhook.POST("/tenant/sample", handlers.TenantSampleWebhook)
	}

	portal := r.Group("/portal")
	{
		portal.POST("/session/initiate", sensitiveRateLimit, handlers.InitiatePortalSession)
		portal.POST("/session/verify", sensitiveRateLimit, handlers.VerifyPortalSession)
		portal.GET("/me", authenticatePortal, handlers.GetPortalSession)
		portal.PATCH("/profile", authenticatePortal, handlers.UpdatePortalProfile)
		portal.GET("/analytics", authenticatePortal, handlers.GetPortalAnalytics)
		portal.GET("/payment-sources", authenticatePortal, handlers.GetPortalPaymentSources)
		portal.POST("/payment-sources/:paymentSourceID/card-update", authenticatePortal, handlers.InitiatePortalCardUpdate)
		portal.POST("/payment-sources/:paymentSourceID/disable", authenticatePortal, handlers.DisablePortalPaymentSource)
		portal.GET("/subscriptions", authenticatePortal, handlers.GetPortalSubscriptions)
		portal.GET("/subscriptions/:idOrCode", authenticatePortal, handlers.GetPortalSubscription)
		portal.POST("/subscriptions/:idOrCode/cancel", authenticatePortal, handlers.CancelPortalSubscription)
		portal.PATCH("/subscriptions/:idOrCode/payment-method", authenticatePortal, handlers.UpdatePortalSubscriptionPaymentMethod)
		portal.GET("/invoices", authenticatePortal, handlers.GetPortalInvoices)
		portal.GET("/invoices/:idOrCode", authenticatePortal, handlers.GetPortalInvoice)
		portal.POST("/invoices/:idOrCode/retry", authenticatePortal, handlers.RetryPortalInvoicePayment)
		portal.GET("/refunds", authenticatePortal, handlers.GetPortalRefunds)
	}

	v1 := r.Group("/v1")
	v1.Use(authenticate, middleware.RateLimitByTenant(300))
	{
		customers := v1.Group("/customer")
		{
			customers.POST("/", handlers.CreateCustomer)
			customers.GET("/", handlers.GetCustomers)
			customers.GET("/:emailOrCode", handlers.GetCustomer)
			customers.PUT("/:emailOrCode", handlers.UpdateCustomer)
			customers.POST("/:emailOrCode/payment-sources/:paymentSourceID/remind-card-expiry", handlers.RemindCustomerCardExpiring)
		}

		plans := v1.Group("/plan")
		{
			plans.POST("/", handlers.CreatePlan)
			plans.GET("/", handlers.GetPlans)
			plans.GET("/:planCode", handlers.GetPlan)
			plans.PUT("/:planCode", handlers.UpdatePlan)
		}

		transactions := v1.Group("/checkout")
		{
			transactions.POST("/order", handlers.InitializeCardTransaction)
			transactions.POST("/direct-debit", handlers.InitializeDirectDebitTransaction)
			transactions.GET("/refunds", handlers.GetRefunds)
			transactions.GET("/payment-attempts", handlers.GetPaymentIntents)
			transactions.POST("/refunds", handlers.RefundPaymentOrInvoice)
		}

		subscriptions := v1.Group("/subscription")
		{
			subscriptions.POST("/", handlers.CreateSubscription)
			subscriptions.GET("/", handlers.GetSubscriptions)
			subscriptions.GET("/:idOrCode", handlers.GetSubscription)
			subscriptions.POST("/:idOrCode/checkout-link", handlers.GenerateSubscriptionCheckoutLink)
			subscriptions.POST("/:idOrCode/cancel", handlers.CancelSubscription)
			subscriptions.PUT("/:idOrCode/mandate", handlers.UpdateSubscriptionMandateStatus)
		}

		invoices := v1.Group("/invoice")
		{
			invoices.GET("/", handlers.GetInvoices)
			invoices.GET("/:idOrCode", handlers.GetInvoice)
			invoices.POST("/:idOrCode/checkout-link", handlers.GenerateInvoiceCheckoutLink)
			invoices.POST("/:idOrCode/retry", handlers.RetryInvoicePayment)
			invoices.POST("/:idOrCode/send-reminder", handlers.SendInvoiceReminder)
		}

		dashboard := v1.Group("/dashboard")
		{
			dashboard.GET("/analytics", handlers.GetDashboardAnalytics)
		}

		webhookDeliveries := v1.Group("/webhook-deliveries")
		{
			webhookDeliveries.GET("/", handlers.GetWebhookDeliveries)
			webhookDeliveries.GET("/:id", handlers.GetWebhookDelivery)
			webhookDeliveries.POST("/:id/retry", handlers.RetryWebhookDelivery)
		}

		settlementPayouts := v1.Group("/settlement-payouts")
		{
			settlementPayouts.GET("/", handlers.GetSettlementPayouts)
			settlementPayouts.GET("/:id", handlers.GetSettlementPayout)
		}
	}

	return r
}
