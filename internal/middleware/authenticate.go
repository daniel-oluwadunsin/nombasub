package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/daniel-oluwadunsin/nombasub/internal/services"
	"github.com/gin-gonic/gin"
)

const (
	TenantIdCtxKey  = "tenant_id"
	AccountIdCtxKey = "tenant_account_id"
)

func Authenticate(
	cfg *config.Config,
	tenantRepo *repositories.Repository[models.Tenant],
	authService *services.AuthService,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tenant *models.Tenant
		var err error

		apiKey := c.GetHeader(cfg.APIKeyHeader)

		if apiKey != "" {
			keyHash := utils.HashAPIKey(apiKey)
			tenant, err = tenantRepo.Find(&models.Tenant{ApiKeyHash: &keyHash}, nil)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, responses.Format(&responses.Response{
					Success: false,
					Message: "Internal server error",
				}))
				return
			}
		}

		token := c.GetHeader("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			accessToken := strings.TrimPrefix(token, "Bearer ")
			tenantId, err := utils.ValidateJwt(accessToken, cfg)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, responses.Format(&responses.Response{
					Success: false,
					Message: "Invalid or expired token",
				}))
				return
			}

			tenant, err = tenantRepo.Find(&models.Tenant{BaseModel: models.BaseModel{ID: tenantId}}, nil)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, responses.Format(&responses.Response{
					Success: false,
					Message: "Internal server error",
				}))
				return
			}

			// Reject tokens that don't match the currently-stored session token
			// (sign-out clears it) or whose server-side expiry has passed. This
			// is what makes sign-out and revocation actually take effect.
			if tenant == nil ||
				tenant.AccessToken == nil ||
				*tenant.AccessToken != accessToken ||
				tenant.AccessTokenExpiresAt == nil ||
				tenant.AccessTokenExpiresAt.Before(time.Now()) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, responses.Format(&responses.Response{
					Success: false,
					Message: "Invalid or expired token",
				}))
				return
			}
		}

		if tenant == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, responses.Format(&responses.Response{
				Success: false,
				Message: "Invalid api key/bearer token",
			}))
			return
		}

		c.Set(TenantIdCtxKey, tenant.ID)
		c.Set(AccountIdCtxKey, tenant.AccountID)
		c.Next()
	}
}
