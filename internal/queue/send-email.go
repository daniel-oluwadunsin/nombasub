package queue

import (
	"encoding/json"
	"fmt"

	"github.com/daniel-oluwadunsin/nombasub/internal/mail"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
)

const SendEmailQueue = "send-email"

type SendEmailJob struct {
	EmailDeliveryID string `json:"emailDeliveryId"`
}

func EnqueueEmail(
	rc *repositories.Container,
	publisher *Publisher,
	recipient string,
	subject string,
	templateName models.EmailTemplateName,
	context models.EmailContext,
	idempotencyKey string,
) error {
	if publisher == nil || recipient == "" || idempotencyKey == "" {
		return nil
	}

	exists, err := rc.EmailDeliveryRepository.Exists(&models.EmailDelivery{IdempotencyKey: idempotencyKey}, nil)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	delivery, err := rc.EmailDeliveryRepository.Create(&models.EmailDelivery{
		Recipient:      recipient,
		Subject:        subject,
		TemplateName:   templateName,
		Context:        context,
		IdempotencyKey: idempotencyKey,
		Status:         models.EmailDeliveryStatusPending,
	}, nil)
	if err != nil {
		return err
	}

	return publisher.Publish(SendEmailQueue, SendEmailJob{EmailDeliveryID: delivery.ID})
}

func SendEmailHandler(rc *repositories.Container, mailer *mail.Mailer) HandlerFunc {
	return func(body []byte) error {
		var job SendEmailJob
		if err := json.Unmarshal(body, &job); err != nil {
			return err
		}

		delivery, err := rc.EmailDeliveryRepository.FindById(job.EmailDeliveryID, nil)
		if err != nil {
			return err
		}
		if delivery == nil || delivery.Status == models.EmailDeliveryStatusDelivered {
			return nil
		}

		delivery.AttemptCount++
		if err := mailer.SendMail(delivery.Recipient, delivery.Subject, "", delivery.TemplateName, delivery.Context); err != nil {
			delivery.Status = models.EmailDeliveryStatusFailed
			reason := err.Error()
			delivery.FailureReason = &reason
			_, updateErr := rc.EmailDeliveryRepository.Update(delivery, nil)
			if updateErr != nil {
				return updateErr
			}
			return fmt.Errorf("failed to send email %s: %w", delivery.ID, err)
		}

		delivery.Status = models.EmailDeliveryStatusDelivered
		_, err = rc.EmailDeliveryRepository.Update(delivery, nil)
		return err
	}
}
