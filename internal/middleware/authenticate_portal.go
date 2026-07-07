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
	"github.com/gin-gonic/gin"
)

const (
	PortalCustomerIdCtxKey = "portal_customer_id"
	PortalSessionIdCtxKey  = "portal_session_id"
)

func AuthenticatePortal(
	cfg *config.Config,
	portalSessionRepo *repositories.Repository[models.PortalSession],
) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if strings.Index(token, "Bearer ") != 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, responses.Format(&responses.Response{
				Success: false,
				Message: "Invalid portal bearer token",
			}))
			return
		}

		accessToken := strings.Split(token, " ")[1]
		claims, err := utils.ValidatePortalJwt(accessToken, cfg)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, responses.Format(&responses.Response{
				Success: false,
				Message: "Invalid portal bearer token",
			}))
			return
		}

		session, err := portalSessionRepo.Find(&models.PortalSession{
			TenantID:   claims.TenantID,
			CustomerID: claims.CustomerID,
			BaseModel:  models.BaseModel{ID: claims.SessionID},
		}, nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, responses.Format(&responses.Response{
				Success: false,
				Message: "Internal server error",
			}))
			return
		}
		if session == nil ||
			session.VerifiedAt == nil ||
			session.RevokedAt != nil ||
			session.AccessTokenHash == nil ||
			session.AccessTokenExpiresAt == nil ||
			session.AccessTokenExpiresAt.Before(time.Now()) ||
			*session.AccessTokenHash != utils.DigestToken(accessToken) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, responses.Format(&responses.Response{
				Success: false,
				Message: "Invalid portal bearer token",
			}))
			return
		}

		c.Set(TenantIdCtxKey, claims.TenantID)
		c.Set(PortalCustomerIdCtxKey, claims.CustomerID)
		c.Set(PortalSessionIdCtxKey, claims.SessionID)
		c.Next()
	}
}
