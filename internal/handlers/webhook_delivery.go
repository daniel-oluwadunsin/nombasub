package handlers

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/middleware"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetWebhookDeliveries(ctx *gin.Context) {
	var query requests.WebhookDeliveriesQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	result, err := h.sc.WebhookDeliveryService.ListWebhookDeliveries(tenantID, query)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Webhook deliveries retrieved successfully", result)
}

func (h *Handler) GetWebhookDelivery(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	deliveryID := ctx.Param("id")

	result, err := h.sc.WebhookDeliveryService.GetWebhookDelivery(tenantID, deliveryID)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, "Webhook delivery retrieved successfully", result)
}

func (h *Handler) RetryWebhookDelivery(ctx *gin.Context) {
	tenantID := ctx.GetString(middleware.TenantIdCtxKey)
	deliveryID := ctx.Param("id")

	result, message, err := h.sc.WebhookDeliveryService.RetryWebhookDelivery(tenantID, deliveryID)
	if err != nil {
		responses.Error(ctx, err)
		return
	}

	responses.Success(ctx, http.StatusOK, message, result)
}
