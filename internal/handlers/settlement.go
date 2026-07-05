package handlers

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/middleware"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetSettlementPayouts(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	var query requests.SettlementPayoutsQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	payouts, err := h.sc.SettlementService.GetSettlementPayouts(tenantID, query)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Settlement payouts retrieved successfully", payouts)
}

func (h *Handler) GetSettlementPayout(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	payoutID := ctx.Param("id")

	payout, err := h.sc.SettlementService.GetSettlementPayout(tenantID, payoutID)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Settlement payout retrieved successfully", payout)
}
