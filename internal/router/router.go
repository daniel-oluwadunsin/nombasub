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
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", handlers.Health)
	r.NoRoute(handlers.NoRoute)

	auth := r.Group("/auth")
	{
		auth.POST("/register", handlers.RegisterTenant)
		auth.POST("/login", handlers.LoginTenant)
	}

	v1 := r.Group("/v1")
	v1.Use(middleware.APIKey(cfg, rc.TenantRepository, sc.AuthService))
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
			transactions.POST("/order", handlers.InitializeCardTransaction) // same nomba route path for card transactions.
		}

		webhook := v1.Group("/webhook")
		{
			webhook.POST("/nomba", handlers.HandleWebhook)
		}

	}

	return r
}
