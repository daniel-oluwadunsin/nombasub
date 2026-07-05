package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
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
	trx *gorm.DB,
) error {
	log.Printf("[send-tenant-webhook] enqueue requested tenant=%s event=%s hasPublisher=%t hasTransaction=%t", tenantID, eventType, publisher != nil, trx != nil)
	if publisher == nil {
		log.Printf("[send-tenant-webhook] enqueue skipped tenant=%s event=%s reason=publisher_nil", tenantID, eventType)
		return nil
	}

	tenant, err := rc.TenantRepository.FindById(tenantID, nil)
	if err != nil {
		log.Printf("[send-tenant-webhook] enqueue failed tenant=%s event=%s stage=find_tenant error=%v", tenantID, eventType, err)
		return err
	}
	if tenant == nil {
		log.Printf("[send-tenant-webhook] enqueue skipped tenant=%s event=%s reason=tenant_not_found", tenantID, eventType)
		return nil
	}
	if tenant.WebhookUrl == nil || *tenant.WebhookUrl == "" {
		log.Printf("[send-tenant-webhook] enqueue skipped tenant=%s event=%s reason=webhook_url_missing", tenantID, eventType)
		return nil
	}
	if tenant.WebhookSecret == nil || *tenant.WebhookSecret == "" {
		log.Printf("[send-tenant-webhook] enqueue skipped tenant=%s event=%s endpoint=%s reason=webhook_secret_missing", tenantID, eventType, *tenant.WebhookUrl)
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
		log.Printf("[send-tenant-webhook] enqueue failed tenant=%s event=%s stage=marshal_payload error=%v", tenantID, eventType, err)
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
	}, trx)
	if err != nil {
		log.Printf("[send-tenant-webhook] enqueue failed tenant=%s event=%s delivery=%s stage=create_delivery error=%v", tenantID, eventType, deliveryID, err)
		return err
	}
	log.Printf("[send-tenant-webhook] delivery created tenant=%s event=%s delivery=%s endpoint=%s payloadBytes=%d", tenantID, eventType, delivery.ID, delivery.EndpointURL, len(payloadBytes))

	if err := publisher.Publish(SendTenantWebhookQueue, SendTenantWebhookJob{WebhookDeliveryID: delivery.ID}); err != nil {
		log.Printf("[send-tenant-webhook] publish failed tenant=%s event=%s delivery=%s queue=%s error=%v", tenantID, eventType, delivery.ID, SendTenantWebhookQueue, err)
		return err
	}
	log.Printf("[send-tenant-webhook] publish ok tenant=%s event=%s delivery=%s queue=%s", tenantID, eventType, delivery.ID, SendTenantWebhookQueue)

	return nil
}

func SendTenantWebhookHandler(rc *repositories.Container) HandlerFunc {
	return func(body []byte) error {
		log.Printf("[send-tenant-webhook] handler received bodyBytes=%d", len(body))
		var job SendTenantWebhookJob
		if err := json.Unmarshal(body, &job); err != nil {
			log.Printf("[send-tenant-webhook] handler failed stage=unmarshal_job body=%s error=%v", string(body), err)
			return err
		}
		log.Printf("[send-tenant-webhook] handler decoded delivery=%s", job.WebhookDeliveryID)

		delivery, err := rc.WebhookDeliveryRepository.FindById(job.WebhookDeliveryID, nil)
		if err != nil {
			log.Printf("[send-tenant-webhook] handler failed delivery=%s stage=find_delivery error=%v", job.WebhookDeliveryID, err)
			return err
		}
		if delivery == nil {
			log.Printf("[send-tenant-webhook] handler skipped delivery=%s reason=delivery_not_found possible_reason=transaction_not_committed_or_deleted", job.WebhookDeliveryID)
			return nil
		}
		log.Printf("[send-tenant-webhook] handler loaded delivery=%s tenant=%s event=%s status=%s attempts=%d/%d endpoint=%s", delivery.ID, delivery.TenantID, delivery.EventType, delivery.Status, delivery.AttempsCount, delivery.MaxAttemptCount, delivery.EndpointURL)
		if delivery.Status == models.WebhookDeliveryStatusDelivered {
			log.Printf("[send-tenant-webhook] handler skipped delivery=%s reason=already_delivered", delivery.ID)
			return nil
		}
		if delivery.AttempsCount >= delivery.MaxAttemptCount {
			delivery.Status = models.WebhookDeliveryStatusFailed
			_, err := rc.WebhookDeliveryRepository.Update(delivery, nil)
			if err != nil {
				log.Printf("[send-tenant-webhook] handler failed delivery=%s stage=mark_max_attempts_failed error=%v", delivery.ID, err)
			} else {
				log.Printf("[send-tenant-webhook] handler marked failed delivery=%s reason=max_attempts_reached attempts=%d/%d", delivery.ID, delivery.AttempsCount, delivery.MaxAttemptCount)
			}
			return err
		}

		tenant, err := rc.TenantRepository.FindById(delivery.TenantID, nil)
		if err != nil {
			log.Printf("[send-tenant-webhook] handler failed delivery=%s tenant=%s stage=find_tenant error=%v", delivery.ID, delivery.TenantID, err)
			return err
		}
		if tenant == nil {
			log.Printf("[send-tenant-webhook] handler failed delivery=%s tenant=%s reason=tenant_not_found", delivery.ID, delivery.TenantID)
			return fmt.Errorf("tenant %s not found", delivery.TenantID)
		}

		timestamp := time.Now().UTC().Format(time.RFC3339)

		request := resty.New().R().
			SetHeader("Content-Type", "application/json").
			SetHeader("x-nombasub-event", string(delivery.EventType)).
			SetHeader("x-nombasub-webhook-id", delivery.ID).
			SetHeader("x-nombasub-tenant-id", delivery.TenantID).
			SetHeader("x-nombasub-timestamp", timestamp)
		if tenant.WebhookSecret != nil && *tenant.WebhookSecret != "" {
			signature := utils.HashOutgoingPayload(string(delivery.EventType), delivery.ID, delivery.TenantID, timestamp, *tenant.WebhookSecret)
			request = request.SetHeader("x-nombasub-signature", signature)
		} else {
			log.Printf("[send-tenant-webhook] handler warning delivery=%s tenant=%s reason=webhook_secret_missing signature_header_omitted", delivery.ID, delivery.TenantID)
		}

		log.Printf("[send-tenant-webhook] http send delivery=%s endpoint=%s event=%s attempt=%d", delivery.ID, delivery.EndpointURL, delivery.EventType, delivery.AttempsCount+1)
		resp, err := request.SetBody(json.RawMessage(delivery.Payload)).Post(delivery.EndpointURL)
		statusCode := 0
		responseBody := ""
		if resp != nil {
			statusCode = resp.StatusCode()
			responseBody = resp.String()
		}
		if err != nil {
			log.Printf("[send-tenant-webhook] http error delivery=%s endpoint=%s status=%d error=%v", delivery.ID, delivery.EndpointURL, statusCode, err)
		} else {
			log.Printf("[send-tenant-webhook] http response delivery=%s endpoint=%s status=%d responseBytes=%d", delivery.ID, delivery.EndpointURL, statusCode, len(responseBody))
		}

		delivery.AttempsCount++
		if err == nil && resp != nil && resp.IsStatusSuccess() {
			delivery.Status = models.WebhookDeliveryStatusDelivered
		} else if delivery.AttempsCount >= delivery.MaxAttemptCount {
			delivery.Status = models.WebhookDeliveryStatusFailed
		} else {
			delivery.Status = models.WebhookDeliveryStatusPending
		}
		log.Printf("[send-tenant-webhook] delivery status transition delivery=%s attempts=%d/%d newStatus=%s", delivery.ID, delivery.AttempsCount, delivery.MaxAttemptCount, delivery.Status)

		if _, createErr := rc.WebhookDeliveryAttemptRepository.Create(&models.WebhookDeliveryAttempt{
			WebhookDeliveryID: delivery.ID,
			StatusCode:        statusCode,
			ResponseBody:      responseBody,
			AttemptCount:      delivery.AttempsCount,
		}, nil); createErr != nil {
			log.Printf("[send-tenant-webhook] handler failed delivery=%s stage=create_attempt attempt=%d statusCode=%d error=%v", delivery.ID, delivery.AttempsCount, statusCode, createErr)
			return createErr
		}
		log.Printf("[send-tenant-webhook] attempt recorded delivery=%s attempt=%d statusCode=%d", delivery.ID, delivery.AttempsCount, statusCode)
		if _, updateErr := rc.WebhookDeliveryRepository.Update(delivery, nil); updateErr != nil {
			log.Printf("[send-tenant-webhook] handler failed delivery=%s stage=update_delivery status=%s error=%v", delivery.ID, delivery.Status, updateErr)
			return updateErr
		}
		log.Printf("[send-tenant-webhook] delivery updated delivery=%s status=%s attempts=%d/%d", delivery.ID, delivery.Status, delivery.AttempsCount, delivery.MaxAttemptCount)

		if err != nil {
			if delivery.Status == models.WebhookDeliveryStatusFailed {
				log.Printf("[send-tenant-webhook] handler completed delivery=%s finalStatus=%s reason=http_error_max_attempts_reached", delivery.ID, delivery.Status)
				return nil
			}
			log.Printf("[send-tenant-webhook] handler requeue delivery=%s reason=http_error status=%s error=%v", delivery.ID, delivery.Status, err)
			return err
		}
		if resp != nil && resp.IsStatusFailure() {
			if delivery.Status == models.WebhookDeliveryStatusFailed {
				log.Printf("[send-tenant-webhook] handler completed delivery=%s finalStatus=%s reason=status_failure_max_attempts_reached statusCode=%d", delivery.ID, delivery.Status, statusCode)
				return nil
			}
			log.Printf("[send-tenant-webhook] handler requeue delivery=%s reason=status_failure statusCode=%d", delivery.ID, statusCode)
			return fmt.Errorf("tenant webhook returned status %d", statusCode)
		}

		log.Printf("[send-tenant-webhook] handler completed delivery=%s finalStatus=%s", delivery.ID, delivery.Status)
		return nil
	}
}
