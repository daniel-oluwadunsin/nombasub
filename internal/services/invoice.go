package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
)

type InvoiceService struct {
	rc            *repositories.Container
	nombaProvider nomba.Provider
	publisher     *queue.Publisher
}

func NewInvoiceService(rc *repositories.Container, nombaProvider nomba.Provider, publisher *queue.Publisher) *InvoiceService {
	return &InvoiceService{rc: rc, nombaProvider: nombaProvider, publisher: publisher}
}

func (s *InvoiceService) CreateUpcomingInvoices() {
	now := time.Now()
	windowEnd := now.AddDate(0, 0, 3)

	subscriptions, err := s.rc.SubscriptionRepository.FindManyRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"subscriptions.status = ? AND subscriptions.current_billing_cycle_start > ? AND subscriptions.current_billing_cycle_start <= ? AND invoice.id IS NULL",
			models.SubscriptionStatusActive,
			now,
			windowEnd,
		),
		Joins: []repositories.Join{
			*repositories.NewJoin(
				"LEFT JOIN invoices as invoice ON invoice.subscription_id = subscriptions.id AND invoice.due_at = subscriptions.current_billing_cycle_start AND invoice.status IN (?, ?)",
				models.InvoiceStatusOpen,
				models.InvoiceStatusDraft,
			),
		},
	})
	if err != nil {
		log.Printf("invoice upcoming cron: failed to load subscriptions: %v", err)
		return
	}

	for _, subscription := range subscriptions {
		invoice, created, err := s.ensureInvoice(&subscription, models.InvoiceStatusDraft)
		if err != nil {
			log.Printf("invoice upcoming cron: failed for subscription %s: %v", subscription.ID, err)
			continue
		}
		if invoice == nil || !created {
			continue
		}
		s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeInvoiceUpcoming, invoice)
	}
}

func (s *InvoiceService) ProcessDueInvoices() {
	subscriptions, err := s.rc.SubscriptionRepository.FindManyRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"subscriptions.status = ? AND subscriptions.current_billing_cycle_start <= ? AND (subscriptions.trial_period_days = 0 OR subscriptions.started_at IS NOT NULL) AND invoice.id IS NOT NULL",
			models.SubscriptionStatusActive,
			time.Now(),
		),
		Joins: []repositories.Join{
			*repositories.NewJoin(
				"LEFT JOIN invoices as invoice ON invoice.subscription_id = subscriptions.id AND invoice.due_at = subscriptions.current_billing_cycle_start AND invoice.status IN (?, ?)",
				models.InvoiceStatusOpen,
				models.InvoiceStatusDraft,
			),
		},
	})
	if err != nil {
		log.Printf("invoice processing cron: failed to load subscriptions: %v", err)
		return
	}

	for _, subscription := range subscriptions {
		if err := s.processDueSubscription(&subscription); err != nil {
			log.Printf("invoice processing cron: failed for subscription %s: %v", subscription.ID, err)
		}
	}
}

func (s *InvoiceService) processDueSubscription(subscription *models.Subscription) error {
	invoice, created, err := s.ensureInvoice(subscription, models.InvoiceStatusOpen)
	if err != nil {
		return err
	}
	if invoice == nil || invoice.Status == models.InvoiceStatusPaid || invoice.Status == models.InvoiceStatusFailed {
		return nil
	}

	opened := created
	if invoice.Status == models.InvoiceStatusDraft {
		invoice.Status = models.InvoiceStatusOpen
		if _, err := s.rc.InvoiceRepository.Update(invoice, nil); err != nil {
			return err
		}
		opened = true
	}

	subscription.LatestInvoiceID = &invoice.ID
	if _, err := s.rc.SubscriptionRepository.Update(subscription, nil); err != nil {
		return err
	}
	if opened {
		s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeInvoiceCreated, invoice)
	}

	if invoice.CheckoutLink != nil || invoice.AttemptCount > 0 {
		return nil
	}

	paymentSource, err := s.findActivePaymentSource(subscription)
	if err != nil {
		return err
	}
	if paymentSource == nil {
		if subscription.PaymentSourceID != nil && *subscription.PaymentSourceID != "" {
			return s.failInvoice(invoice, subscription, "attached payment source is inactive", models.WebhookDeliveryEventTypeInvoiceMarkedUncollectible)
		}
		return s.createCheckout(invoice, subscription)
	}

	if paymentSource.Type == models.PaymentSourceTypeCard {
		return s.chargeCard(invoice, subscription, paymentSource)
	} else {
	}

	return nil
}

func (s *InvoiceService) ensureInvoice(subscription *models.Subscription, status models.InvoiceStatus) (*models.Invoice, bool, error) {
	existing, err := s.rc.InvoiceRepository.FindRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"subscription_id = ? AND due_at = ?",
			subscription.ID,
			subscription.CurrentBillingCycleStart,
		),
	})
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		return existing, false, nil
	}

	invoice := &models.Invoice{
		TenantID:             subscription.TenantID,
		SubscriptionID:       subscription.ID,
		CustomerID:           subscription.CustomerID,
		Status:               status,
		AmountDue:            subscription.Amount,
		AmountPaid:           0,
		AmountRemaining:      subscription.Amount,
		Currency:             subscription.Currency,
		BillingPeriodStart:   subscription.CurrentBillingCycleStart,
		BillingPeriodEnd:     subscription.CurrentBillingCycleEnd,
		DueAt:                subscription.CurrentBillingCycleStart,
		NextPaymentAttemptAt: subscription.CurrentBillingCycleStart,
	}

	invoice.Code, err = utils.GenerateCode("INV")
	if err != nil {
		return nil, false, err
	}

	invoice, err = s.rc.InvoiceRepository.Create(invoice, nil)
	return invoice, true, err
}

func (s *InvoiceService) findActivePaymentSource(subscription *models.Subscription) (*models.PaymentSource, error) {
	if subscription.PaymentSourceID == nil || *subscription.PaymentSourceID == "" {
		return nil, nil
	}

	paymentSource, err := s.rc.PaymentSourceRepository.FindById(*subscription.PaymentSourceID, nil)
	if err != nil {
		return nil, err
	}
	if paymentSource == nil || paymentSource.Status != models.PaymentSourceStatusActive {
		return nil, nil
	}
	return paymentSource, nil
}

func (s *InvoiceService) createCheckout(invoice *models.Invoice, subscription *models.Subscription) error {
	tenant, err := s.rc.TenantRepository.FindById(subscription.TenantID, nil)
	if err != nil {
		return err
	}
	customer, err := s.rc.CustomerRepository.FindById(subscription.CustomerID, nil)
	if err != nil {
		return err
	}
	planVersion, err := s.rc.PlanVersionRepository.FindById(subscription.PlanVersionID, nil)
	if err != nil {
		return err
	}
	if tenant == nil || customer == nil || planVersion == nil {
		return fmt.Errorf("missing tenant, customer, or plan version for invoice %s", invoice.ID)
	}

	reference, err := utils.GenerateRandomString(24)
	if err != nil {
		return err
	}
	reference = fmt.Sprintf("nombasub_%s", reference)

	metadata := map[string]interface{}{
		"nombaSubTenantId":        subscription.TenantID,
		"nombaSubCustomerCode":    customer.Code,
		"nombaSubPlanCode":        planVersion.Code,
		"nombaSubPlanVersion":     planVersion.Index,
		"nombaSubTenantAccountId": tenant.AccountID,
		"nombaSubInvoiceId":       invoice.ID,
		"nombaSubSubscriptionId":  subscription.ID,
	}

	initiation, err := s.rc.NombaInitiationRepository.Create(&models.NombaInitiation{
		TenantID:  subscription.TenantID,
		Amount:    float64(invoice.AmountDue),
		Currency:  invoice.Currency,
		Reference: reference,
		Purpose:   models.NombaInitiationPurposeCardSubscriptionPayment,
		Status:    models.NombaInitiationStatusPending,
		Metadata:  metadata,
	}, nil)
	if err != nil {
		return err
	}

	response, err := s.nombaProvider.CreateCheckoutOrder(nomba.CreateCheckoutOrderRequest{
		Order: nomba.NombaOrder{
			CallbackUrl:           "",
			CustomerEmail:         customer.Email,
			Amount:                invoice.AmountDue,
			Currency:              &invoice.Currency,
			OrderReference:        &reference,
			AccountId:             &tenant.AccountID,
			AllowedPaymentMethods: &[]nomba.PaymentMethod{nomba.PaymentMethodCard},
			OrderMetaData:         &metadata,
		},
		TokenizeCard: utils.ToPtr(true),
	})
	if err != nil {
		return err
	}

	initiation.NombaOrderID = &response.Data.OrderReference
	if _, err := s.rc.NombaInitiationRepository.Update(initiation, nil); err != nil {
		return err
	}

	invoice.CheckoutLink = &response.Data.CheckoutLink
	if _, err := s.rc.InvoiceRepository.Update(invoice, nil); err != nil {
		return err
	}

	s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeCheckoutCreated, map[string]interface{}{
		"invoice":     invoice,
		"checkoutUrl": response.Data.CheckoutLink,
	})

	return nil
}

func (s *InvoiceService) chargeCard(invoice *models.Invoice, subscription *models.Subscription, paymentSource *models.PaymentSource) error {
	if paymentSource.Card == nil || paymentSource.Card.AuthorizationToken == nil {
		return s.failInvoice(invoice, subscription, "active card payment source is missing an authorization token", models.WebhookDeliveryEventTypeInvoiceMarkedUncollectible)
	}

	reference, err := utils.GenerateRandomString(24)
	if err != nil {
		return err
	}
	reference = fmt.Sprintf("nombasub_%s", reference)

	paymentIntent := &models.PaymentIntent{
		TenantID:          subscription.TenantID,
		CustomerID:        subscription.CustomerID,
		SubscriptionID:    subscription.ID,
		InvoiceID:         &invoice.ID,
		PlanID:            subscription.PlanID,
		PlanVersionID:     subscription.PlanVersionID,
		PaymentSourceID:   &paymentSource.ID,
		PaymentSourceType: &paymentSource.Type,
		Reference:         reference,
		Amount:            invoice.AmountDue,
		Currency:          invoice.Currency,
		Status:            models.PaymentIntentStatusPendingBilling,
		AttemptedAt:       utils.ToPtr(time.Now()),
	}
	paymentIntent.Code, err = utils.GenerateCode("PAY")
	if err != nil {
		return err
	}

	paymentIntent, err = s.rc.PaymentIntentRepository.Create(paymentIntent, nil)
	if err != nil {
		return err
	}

	invoice.AttemptCount++
	if _, err := s.rc.InvoiceRepository.Update(invoice, nil); err != nil {
		return err
	}
	s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeInvoicePaymentAttempted, invoice)

	tenant, err := s.rc.TenantRepository.FindById(subscription.TenantID, nil)
	if err != nil {
		return err
	}
	customer, err := s.rc.CustomerRepository.FindById(subscription.CustomerID, nil)
	if err != nil {
		return err
	}
	if tenant == nil || customer == nil {
		return fmt.Errorf("missing tenant or customer for invoice %s", invoice.ID)
	}

	metadata := map[string]interface{}{
		"nombaSubTenantId":        subscription.TenantID,
		"nombaSubCustomerCode":    customer.Code,
		"nombaSubInvoiceId":       invoice.ID,
		"nombaSubSubscriptionId":  subscription.ID,
		"nombaSubTenantAccountId": tenant.AccountID,
	}

	response, err := s.nombaProvider.ChargeCard(nomba.ChargeCardRequest{
		Order: nomba.NombaOrder{
			CallbackUrl:    "",
			CustomerEmail:  customer.Email,
			Amount:         invoice.AmountDue,
			Currency:       &invoice.Currency,
			OrderReference: &reference,
			AccountId:      &tenant.AccountID,
			OrderMetaData:  &metadata,
		},
		TokenKey: *paymentSource.Card.AuthorizationToken,
	})
	if err != nil {
		paymentIntent.Status = models.PaymentIntentStatusFailed
		paymentIntent.FailureReason = utils.ToPtr(err.Error())
		paymentIntent.FailedAt = utils.ToPtr(time.Now())
		_, _ = s.rc.PaymentIntentRepository.Update(paymentIntent, nil)
		return s.failInvoice(invoice, subscription, err.Error(), models.WebhookDeliveryEventTypeInvoicePaymentFailed)
	}

	responseJSON, _ := json.Marshal(response)
	paymentIntent.Status = models.PaymentIntentStatusSuccess
	paymentIntent.ProviderResponse = utils.ToPtr(string(responseJSON))
	paymentIntent.CompletedAt = utils.ToPtr(time.Now())
	if _, err := s.rc.PaymentIntentRepository.Update(paymentIntent, nil); err != nil {
		return err
	}

	subscription.PaymentSourceID = &paymentSource.ID
	subscription.PaymentSourceType = &paymentSource.Type
	return s.markInvoicePaid(invoice, subscription)
}

func (s *InvoiceService) failInvoice(invoice *models.Invoice, subscription *models.Subscription, reason string, eventType models.WebhookDeliveryEventType) error {
	now := time.Now()
	invoice.Status = models.InvoiceStatusFailed
	invoice.FailedAt = &now
	invoice.FailureReason = &reason
	if _, err := s.rc.InvoiceRepository.Update(invoice, nil); err != nil {
		return err
	}

	subscription.Status = models.SubscriptionStatusPaused
	subscription.PausedAt = &now
	subscription.LatestInvoiceID = &invoice.ID
	if _, err := s.rc.SubscriptionRepository.Update(subscription, nil); err != nil {
		return err
	}

	s.enqueueWebhook(subscription.TenantID, eventType, invoice)
	s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeSubscriptionPaused, subscription)
	return nil
}

func (s *InvoiceService) markInvoicePaid(invoice *models.Invoice, subscription *models.Subscription) error {
	now := time.Now()
	invoice.Status = models.InvoiceStatusPaid
	invoice.AmountPaid = invoice.AmountDue
	invoice.AmountRemaining = 0
	invoice.PaidAt = &now
	if _, err := s.rc.InvoiceRepository.Update(invoice, nil); err != nil {
		return err
	}

	startDate, endDate := utils.GetBillingPeriod(*subscription.CurrentBillingCycleEnd, subscription.Interval, subscription.IntervalCount)
	subscription.CurrentBillingCycleStart = &startDate
	subscription.CurrentBillingCycleEnd = &endDate
	subscription.LatestInvoiceID = &invoice.ID
	subscription.InvoiceCount++
	if _, err := s.rc.SubscriptionRepository.Update(subscription, nil); err != nil {
		return err
	}

	s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeInvoicePaid, invoice)
	return nil
}

func (s *InvoiceService) enqueueWebhook(tenantID string, eventType models.WebhookDeliveryEventType, data any) {
	if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantID, eventType, data); err != nil {
		log.Printf("invoice webhook enqueue failed for tenant %s event %s: %v", tenantID, eventType, err)
	}
}
