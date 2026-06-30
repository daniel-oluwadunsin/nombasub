package handlers

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
)

func (h *Handler) HandleWebhook(ctx *gin.Context) {
	webhookService := h.sc.WebhookService

	var webhookRequest nomba.NombaWebhookRequest

	if err := ctx.ShouldBindJSON(&webhookRequest); err != nil {
		responses.Error(ctx, responses.BadRequest("Invalid webhook request body"))
		return
	}

	receivedSignature := ctx.GetHeader("nomba-signature")
	timeStamp := ctx.GetHeader("nomba-timestamp")

	if !webhookService.ValidateWebhookSignature(receivedSignature, timeStamp, webhookRequest) {
		responses.Error(ctx, responses.Unauthorized("Invalid webhook signature"))
		return
	}

	if err := webhookService.HandleWebhook(webhookRequest); err != nil {
		responses.Error(ctx, responses.InternalServerError(err))
		return
	}

	responses.SuccessEmpty(ctx, http.StatusOK, "Webhook processed")
}
