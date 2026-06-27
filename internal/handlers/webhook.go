package handlers

import (
	"net/http"

	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func (h *Handler) HandleWebhook(ctx *gin.Context) {
	webhookService := h.sc.WebhookService

	var webhookRequest nomba.NombaWebhookRequest
	if err := ctx.ShouldBindBodyWith(&webhookRequest, binding.JSON); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}

	// read body in string format for logging for now
	var requestBody map[string]interface{}
	if err := ctx.ShouldBindBodyWith(&requestBody, binding.JSON); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
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
