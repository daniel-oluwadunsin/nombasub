package services

import (
	"log"
	"strconv"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
)

type SubscriptionLifecycleService struct {
	rc        *repositories.Container
	publisher *queue.Publisher
}

func NewSubscriptionLifecycleService(rc *repositories.Container, publisher *queue.Publisher) *SubscriptionLifecycleService {
	return &SubscriptionLifecycleService{rc: rc, publisher: publisher}
}

func (s *SubscriptionLifecycleService) ProcessTrials() {
	s.sendTrialEndingSoonWebhooks()
	s.startBillingForEndedTrials()
}

func (s *SubscriptionLifecycleService) ProcessCardExpirations() {
	paymentSources, err := s.rc.PaymentSourceRepository.FindManyRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"type = ? AND expiration_mail_sent = ?",
			models.PaymentSourceTypeCard,
			false,
		),
	})
	if err != nil {
		log.Printf("card expiration cron: failed to load payment sources: %v", err)
		return
	}

	for _, paymentSource := range paymentSources {
		if !cardExpiresSoon(paymentSource.Card, time.Now()) {
			continue
		}

		subscriptions, err := s.rc.SubscriptionRepository.FindMany(&models.Subscription{
			PaymentSourceID: &paymentSource.ID,
			Status:          models.SubscriptionStatusActive,
		}, nil)
		if err != nil {
			log.Printf("card expiration cron: failed to load subscriptions for payment source %s: %v", paymentSource.ID, err)
			continue
		}

		for _, subscription := range subscriptions {
			s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeSubscriptionCardExpiring, map[string]interface{}{
				"subscription":  subscription,
				"paymentSource": paymentSource,
			})
			enqueueCardExpiringEmail(s.rc, s.publisher, &subscription, &paymentSource)
		}

		paymentSource.ExpirationMailSent = true
		if _, err := s.rc.PaymentSourceRepository.Update(&paymentSource, nil); err != nil {
			log.Printf("card expiration cron: failed to update payment source %s: %v", paymentSource.ID, err)
		}
	}
}

func (s *SubscriptionLifecycleService) sendTrialEndingSoonWebhooks() {
	now := time.Now()
	windowEnd := now.AddDate(0, 0, 3)

	subscriptions, err := s.rc.SubscriptionRepository.FindManyRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"status = ? AND trial_end_date IS NOT NULL AND trial_ending_soon_sent = ? AND trial_end_date > ? AND trial_end_date <= ?",
			models.SubscriptionStatusActive,
			false,
			now,
			windowEnd,
		),
	})
	if err != nil {
		log.Printf("trial lifecycle cron: failed to load ending trials: %v", err)
		return
	}

	for _, subscription := range subscriptions {
		s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeSubscriptionTrialEnding, subscription)
		enqueueSubscriptionEmail(s.rc, s.publisher, models.EmailTemplateTrialEndingSoon, &subscription, string(models.EmailTemplateTrialEndingSoon)+":"+subscription.ID)
		subscription.TrialEndingSoonSent = true
		if _, err := s.rc.SubscriptionRepository.Update(&subscription, nil); err != nil {
			log.Printf("trial lifecycle cron: failed to update subscription %s: %v", subscription.ID, err)
		}
	}
}

func (s *SubscriptionLifecycleService) startBillingForEndedTrials() {
	now := time.Now()

	subscriptions, err := s.rc.SubscriptionRepository.FindManyRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"status = ? AND trial_end_date IS NOT NULL AND trial_end_date <= ? AND started_at IS NULL",
			models.SubscriptionStatusActive,
			now,
		),
	})
	if err != nil {
		log.Printf("trial lifecycle cron: failed to load ended trials: %v", err)
		return
	}

	for _, subscription := range subscriptions {
		subscription.StartedAt = &now
		if _, err := s.rc.SubscriptionRepository.Update(&subscription, nil); err != nil {
			log.Printf("trial lifecycle cron: failed to start billing for subscription %s: %v", subscription.ID, err)
			continue
		}
		s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeSubscriptionBillingStarted, subscription)
		enqueueSubscriptionEmail(s.rc, s.publisher, models.EmailTemplateTrialEndedBillingStarted, &subscription, string(models.EmailTemplateTrialEndedBillingStarted)+":"+subscription.ID)
	}
}

func (s *SubscriptionLifecycleService) enqueueWebhook(tenantID string, eventType models.WebhookDeliveryEventType, data any) {
	if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantID, eventType, data, nil); err != nil {
		log.Printf("subscription lifecycle webhook enqueue failed for tenant %s event %s: %v", tenantID, eventType, err)
	}
}

func cardExpiresSoon(card *models.CardPaymentSource, now time.Time) bool {
	if card == nil || card.ExpiryMonth == nil || card.ExpiryYear == nil {
		return false
	}

	month, err := strconv.Atoi(*card.ExpiryMonth)
	if err != nil || month < 1 || month > 12 {
		return false
	}

	year, err := strconv.Atoi(*card.ExpiryYear)
	if err != nil {
		return false
	}
	if year < 100 {
		year += 2000
	}

	expiryBoundary := time.Date(year, time.Month(month)+1, 1, 0, 0, 0, 0, now.Location())
	return expiryBoundary.After(now) && (expiryBoundary.Before(now.AddDate(0, 0, 31)) || expiryBoundary.Equal(now.AddDate(0, 0, 31)))
}
