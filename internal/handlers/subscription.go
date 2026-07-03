package handlers

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/middleware"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateSubscription(ctx *gin.Context) {
	var body requests.CreateSubscriptionRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	data, err := h.sc.SubscriptionService.CreateSubscription(tenantId, body)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusCreated, "Subscription Created Successfully", data)
}

func (h *Handler) GetSubscriptions(ctx *gin.Context) {
	var query requests.GetSubscriptionQuery

	if err := ctx.ShouldBindQuery(&query); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	data, err := h.sc.SubscriptionService.GetSubscriptions(tenantId, query)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Subscriptions retrieved successfully", data)
}

func (h *Handler) GetSubscription(ctx *gin.Context) {
	idOrCode := ctx.Param("idOrCode")
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	data, err := h.sc.SubscriptionService.GetSubscription(tenantId, idOrCode)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Subscription retrieved successfully", data)
}

func (h *Handler) UpdateSubscriptionMandateStatus(ctx *gin.Context) {
	var body requests.UpdateMandateStatusRequest

	if err := ctx.ShouldBindJSON(&body); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	idOrCode := ctx.Param("idOrCode")
	tenantId := ctx.GetString(middleware.TenantIdCtxKey)

	if err := h.sc.SubscriptionService.UpdateDirectDebitMandateStatus(tenantId, idOrCode, body); err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.SuccessEmpty(ctx, http.StatusOK, "Mandate status updated successfully")
}
