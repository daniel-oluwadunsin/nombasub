package middleware

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/daniel-oluwadunsin/nombasub/internal/services"
	"github.com/gin-gonic/gin"
)

const (
	TenantIdCtxKey = "tenant_id"
)

func APIKey(
	cfg *config.Config,
	tenantRepo *repositories.Repository[models.Tenant],
	authService *services.AuthService,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader(cfg.APIKeyHeader)

		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, responses.Format(&responses.Response{
				Success: false,
				Message: "Api key header not provided",
			}))
			return
		}

		tenant, err := tenantRepo.Find(&models.Tenant{ApiKey: apiKey}, nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responses.Format(&responses.Response{
				Success: false,
				Message: "Internal server error",
			}))
			return
		}

		if tenant == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, responses.Format(&responses.Response{
				Success: false,
				Message: "Invalid api key",
			}))
			return
		}

		c.Set(TenantIdCtxKey, tenant.ID)
		c.Next()
	}
}
