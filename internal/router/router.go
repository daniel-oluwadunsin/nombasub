package router

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/daniel-oluwadunsin/nombasub/internal/handlers"
	"github.com/daniel-oluwadunsin/nombasub/internal/middleware"
	"github.com/gin-gonic/gin"
)

func New(cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", handlers.Health)

	auth := r.Group("/auth")
	{
		auth.POST("/register", handlers.Register)
		auth.POST("/login", handlers.Login)
		auth.POST("/refresh", handlers.RefreshToken)
	}

	v1 := r.Group("/v1")
	v1.Use(middleware.JWTAuth(cfg), middleware.ExternalAPIKey(cfg))
	{
		// product routes go here
	}

	return r
}
