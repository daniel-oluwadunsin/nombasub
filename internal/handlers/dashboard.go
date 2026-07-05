package handlers

import (
	"net/http"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/middleware"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetDashboardAnalytics(ctx *gin.Context) {
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)
	var from *time.Time
	var to *time.Time

	if rawFrom := ctx.Query("from"); rawFrom != "" {
		parsedFrom, err := time.Parse("2006-01-02", rawFrom)
		if err != nil {
			responses.Error(ctx, responses.BadRequest("from must use YYYY-MM-DD format"))
			return
		}
		from = &parsedFrom
	}

	if rawTo := ctx.Query("to"); rawTo != "" {
		parsedTo, err := time.Parse("2006-01-02", rawTo)
		if err != nil {
			responses.Error(ctx, responses.BadRequest("to must use YYYY-MM-DD format"))
			return
		}
		to = &parsedTo
	}

	analytics, err := h.sc.DashboardAnalyticsService.GetAnalytics(tenantId, from, to)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Dashboard analytics retrieved successfully", analytics)
}
