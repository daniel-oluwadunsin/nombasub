package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/google/uuid"
	"resty.dev/v3"
)

const SendTenantWebhookQueue = "send-tenant-webhook"

type SendTenantWebhookJob struct {
	WebhookDeliveryID string `json:"webhookDeliveryId"`
}

type TenantWebhookPayload struct {
	ID        string                          `json:"id"`
	EventType models.WebhookDeliveryEventType `json:"eventType"`
	TenantID  string                          `json:"tenantId"`
	Data      any                             `json:"data"`
	CreatedAt time.Time                       `json:"createdAt"`
}

func EnqueueTenantWebhook(
	rc *repositories.Container,
	publisher *Publisher,
	tenantID string,
	eventType models.WebhookDeliveryEventType,
	data any,
) error {
	if publisher == nil {
		return nil
	}

	tenant, err := rc.TenantRepository.FindById(tenantID, nil)
	if err != nil {
		return err
	}
	if tenant == nil ||
		tenant.WebhookUrl == nil || *tenant.WebhookUrl == "" ||
		tenant.WebhookSecret == nil || *tenant.WebhookSecret == "" {
		return nil
	}

	deliveryID := uuid.New().String()
	payloadBytes, err := json.Marshal(TenantWebhookPayload{
		ID:        deliveryID,
		EventType: eventType,
		TenantID:  tenantID,
		Data:      data,
		CreatedAt: time.Now(),
	})
	if err != nil {
		return err
	}

	delivery, err := rc.WebhookDeliveryRepository.Create(&models.WebhookDelivery{
		BaseModel:       models.BaseModel{ID: deliveryID},
		TenantID:        tenantID,
		EndpointURL:     *tenant.WebhookUrl,
		EventType:       eventType,
		Payload:         string(payloadBytes),
		Status:          models.WebhookDeliveryStatusPending,
		MaxAttemptCount: 3,
	}, nil)
	if err != nil {
		return err
	}

	return publisher.Publish(SendTenantWebhookQueue, SendTenantWebhookJob{WebhookDeliveryID: delivery.ID})
}

func SendTenantWebhookHandler(rc *repositories.Container) HandlerFunc {
	client := resty.New().R()

	return func(body []byte) error {
		var job SendTenantWebhookJob
		if err := json.Unmarshal(body, &job); err != nil {
			return err
		}

		delivery, err := rc.WebhookDeliveryRepository.FindById(job.WebhookDeliveryID, nil)
		if err != nil {
			return err
		}
		if delivery == nil || delivery.Status == models.WebhookDeliveryStatusDelivered {
			return nil
		}
		if delivery.AttempsCount >= delivery.MaxAttemptCount {
			delivery.Status = models.WebhookDeliveryStatusFailed
			_, err := rc.WebhookDeliveryRepository.Update(delivery, nil)
			return err
		}

		tenant, err := rc.TenantRepository.FindById(delivery.TenantID, nil)
		if err != nil {
			return err
		}
		if tenant == nil {
			return fmt.Errorf("tenant %s not found", delivery.TenantID)
		}

		timestamp := time.Now().UTC().Format(time.RFC3339)

		client = client.SetHeader("Content-Type", "application/json").
			SetHeader("x-nombasub-event", string(delivery.EventType)).
			SetHeader("x-nombasub-webhook-id", delivery.ID).
			SetHeader("x-nombasub-tenant-id", delivery.TenantID).
			SetHeader("x-nombasub-timestamp", timestamp)
		if tenant.WebhookSecret != nil && *tenant.WebhookSecret != "" {
			signature := utils.HashOutgoingPayload(string(delivery.EventType), delivery.ID, delivery.TenantID, timestamp, *tenant.WebhookSecret)
			client = client.SetHeader("x-nombasub-signature", signature)
		}

		resp, err := client.Post(delivery.EndpointURL)
		statusCode := resp.StatusCode()
		responseBody := resp.String()

		delivery.AttempsCount++
		if err == nil && resp.IsStatusSuccess() {
			delivery.Status = models.WebhookDeliveryStatusDelivered
		} else if delivery.AttempsCount >= delivery.MaxAttemptCount {
			delivery.Status = models.WebhookDeliveryStatusFailed
		}

		if _, createErr := rc.WebhookDeliveryAttemptRepository.Create(&models.WebhookDeliveryAttempt{
			WebhookDeliveryID: delivery.ID,
			StatusCode:        statusCode,
			ResponseBody:      responseBody,
			AttemptCount:      delivery.AttempsCount,
		}, nil); createErr != nil {
			return createErr
		}
		if _, updateErr := rc.WebhookDeliveryRepository.Update(delivery, nil); updateErr != nil {
			return updateErr
		}

		if err != nil {
			if delivery.Status == models.WebhookDeliveryStatusFailed {
				return nil
			}
			return err
		}
		if resp.IsStatusFailure() {
			if delivery.Status == models.WebhookDeliveryStatusFailed {
				return nil
			}
			return fmt.Errorf("tenant webhook returned status %d", statusCode)
		}

		return nil
	}
}
