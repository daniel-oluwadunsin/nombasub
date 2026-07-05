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
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", handlers.Health)
	r.NoRoute(handlers.NoRoute)

	auth := r.Group("/auth")
	{
		auth.POST("/register", handlers.RegisterTenant)
		auth.POST("/login", handlers.LoginTenant)
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
		}

		subscriptions := v1.Group("/subscription")
		{
			subscriptions.POST("/", handlers.CreateSubscription)
			subscriptions.GET("/", handlers.GetSubscriptions)
			subscriptions.GET("/:idOrCode", handlers.GetSubscription)
			subscriptions.PUT("/:idOrCode/mandate", handlers.UpdateSubscriptionMandateStatus)
		}

	}

	return r
}
