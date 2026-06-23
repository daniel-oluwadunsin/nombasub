package middleware

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/gin-gonic/gin"
)

const ExternalAPIKeyCtxKey = "external_api_key"

func ExternalAPIKey(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader(cfg.ExternalAPIKeyHeader)
		if key == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "missing " + cfg.ExternalAPIKeyHeader + " header"})
			return
		}
		c.Set(ExternalAPIKeyCtxKey, key)
		c.Next()
	}
}
