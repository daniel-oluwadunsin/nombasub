package handlers

import (
	"encoding/json"
	"fmt"
	"log"
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

	// TODO: signature verification temporarily disabled while the Nomba signature
	// format is validated against live payloads. Re-enable before treating this
	// endpoint as trusted — until then, forged webhook events are accepted.
	// receivedSignature := ctx.GetHeader("nomba-signature")
	// timeStamp := ctx.GetHeader("nomba-timestamp")
	//
	// if !webhookService.ValidateWebhookSignature(receivedSignature, timeStamp, webhookRequest) {
	// 	log.Printf("[webhook] rejected nomba webhook: invalid signature event=%q", webhookRequest.EventType)
	// 	responses.Error(ctx, responses.Unauthorized("Invalid webhook signature"))
	// 	return
	// }

	if err := webhookService.HandleWebhook(webhookRequest); err != nil {
		log.Printf("[webhook] handler error: %v", err)
		responses.Error(ctx, responses.InternalServerError(err))
		return
	}

	responses.SuccessEmpty(ctx, http.StatusOK, "Webhook processed")
}

func (h *Handler) TenantSampleWebhook(ctx *gin.Context) {
	var requestBody map[string]interface{}
	if err := ctx.ShouldBindBodyWith(&requestBody, binding.JSON); err != nil {
		responses.Error(ctx, responses.BadRequest(err.Error()))
		return
	}
	jsonBytes, err := json.Marshal(requestBody)
	if err != nil {
		return
	}
	fmt.Println(string(jsonBytes))

	responses.SuccessEmpty(ctx, http.StatusOK, "Webhook processed")
}
