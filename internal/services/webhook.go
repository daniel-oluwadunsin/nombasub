package services

import (
	"encoding/json"
	"fmt"

	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
)

type WebhookService struct {
	rc          *repositories.Container
	nombaClient nomba.Provider
}

func NewWebhookService(rc *repositories.Container, nombaClient nomba.Provider) *WebhookService {
	return &WebhookService{rc: rc, nombaClient: nombaClient}
}

func (ws WebhookService) convertRequestBodyToJson(body interface{}) (*string, error) {
	var jsonData string

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	jsonData = string(jsonBytes)
	fmt.Print("Webhook data received", jsonData)
	return &jsonData, nil
}

func (ws *WebhookService) ValidateWebhookSignature(receivedSignature, timestamp string, payload nomba.NombaWebhookRequest) bool {
	payloadJson, err := ws.convertRequestBodyToJson(payload)
	if err != nil {
		return false
	}

	expectedSignature, err :=
		ws.nombaClient.GenerateSignature(*payloadJson, timestamp)
	if err != nil {
		return false
	}

	return receivedSignature == expectedSignature
}

func (ws *WebhookService) handlePaymentSuccess(payload nomba.NombaWebhookRequest) error {
	// initiationRequest, err := ws.rc.NombaInitiationRepository.Find(&models.NombaInitiation{Reference: payload.Data.Transaction.})
	// Implement the logic to handle payment success webhook event
	return nil
}

func (ws *WebhookService) handlePaymentFailed(payload nomba.NombaWebhookRequest) error {
	// Implement the logic to handle payment failed webhook event
	return nil
}

func (ws *WebhookService) handlePaymentReversal(payload nomba.NombaWebhookRequest) error {
	// Implement the logic to handle payment reversal webhook event
	return nil
}

func (ws *WebhookService) handlePayoutSuccess(payload nomba.NombaWebhookRequest) error {
	// Implement the logic to handle payout success webhook event
	return nil
}

func (ws *WebhookService) handlePayoutFailed(payload nomba.NombaWebhookRequest) error {
	// Implement the logic to handle payout failed webhook event
	return nil
}

func (ws *WebhookService) handlePayoutRefund(payload nomba.NombaWebhookRequest) error {
	// Implement the logic to handle payout refund webhook event
	return nil
}

func (ws *WebhookService) HandleWebhook(payload nomba.NombaWebhookRequest) error {
	eventType := payload.EventType

	switch eventType {
	case nomba.WebhookEventTypePaymentSuccess:
		return ws.handlePaymentSuccess(payload)
	case nomba.WebhookEventTypePaymentFailed:
		return ws.handlePaymentFailed(payload)
	case nomba.WebhookEventTypePaymentReversal:
		return ws.handlePaymentReversal(payload)
	case nomba.WebhookEventTypePayoutSuccess:
		return ws.handlePayoutSuccess(payload)
	case nomba.WebhookEventTypePayoutFailed:
		return ws.handlePayoutFailed(payload)
	case nomba.WebhookEventTypePayoutRefund:
		return ws.handlePayoutRefund(payload)
	default:
		return nil
	}
}
