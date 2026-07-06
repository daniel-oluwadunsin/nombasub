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
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS())

	r.GET("/health", handlers.Health)
	r.NoRoute(handlers.NoRoute)

	auth := r.Group("/auth")
	{
		auth.POST("/register", handlers.RegisterTenant)
		auth.POST("/login", handlers.LoginTenant)
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

	v1 := r.Group("/v1")
	v1.Use(authenticate)
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
