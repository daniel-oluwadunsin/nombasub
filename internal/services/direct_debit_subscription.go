package services

import (
	"log"
	"strings"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"gorm.io/gorm"
)

type DirectDebitSubscriptionService struct {
	rc            *repositories.Container
	nombaProvider nomba.Provider
	publisher     *queue.Publisher
}

func NewDirectDebitSubscriptionService(rc *repositories.Container, nombaProvider nomba.Provider, publisher *queue.Publisher) *DirectDebitSubscriptionService {
	return &DirectDebitSubscriptionService{rc: rc, nombaProvider: nombaProvider, publisher: publisher}
}

// PollPendingMandates is defined in direct_debit_backoff.go. It applies per-
// mandate exponential backoff so we don't hammer Nomba (or bill our own quota)
// for mandates a customer hasn't approved yet.

func (s *DirectDebitSubscriptionService) processPendingMandate(initiation *models.NombaInitiation) error {
	mandateId := initiation.Reference

	statusResp, err := s.nombaProvider.GetDirectDebitManadateStatus(mandateId)
	if err != nil {
		log.Printf("direct debit poll: GetDirectDebitManadateStatus failed for mandate=%s: %v", mandateId, err)
		return err
	}

	data := statusResp.Data
	log.Printf("direct debit poll: mandate=%s mandateStatus=%s adviceStatus=%s", mandateId, data.MandateStatus, data.MandateAdviceStatus)

	// Nomba sometimes returns "Active" / "Deleted" in mixed case, so normalize.
	normalizedStatus := strings.ToUpper(string(data.MandateStatus))

	if normalizedStatus == string(nomba.MandateStatusDeleted) {
		initiation.Status = models.NombaInitiationStatusFailed
		_, err = s.rc.NombaInitiationRepository.Update(initiation, nil)
		if err != nil {
			return err
		}
		tenantId, _ := initiation.Metadata["nombaSubTenantId"].(string)
		if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantId, models.WebhookDeliveryEventTypeMandateActivationFailed, map[string]interface{}{
			"mandateId": mandateId,
			"reason":    "mandate was deleted before activation",
		}, nil); err != nil {
			log.Printf("direct debit poll: failed to enqueue mandate.activation_failed webhook: %v", err)
		}
		return nil
	}

	if normalizedStatus != string(nomba.MandateStatusActive) || strings.ToUpper(data.MandateAdviceStatus) != "ADVICE_SENT" {
		log.Printf("direct debit poll: mandate=%s not yet activatable (status=%s advice=%s), skipping", mandateId, data.MandateStatus, data.MandateAdviceStatus)
		return nil
	}
	log.Printf("direct debit poll: mandate=%s is ACTIVE + ADVICE_SENT, activating subscription", mandateId)

	// Extract fields from anonymous struct before entering the transaction closure
	accountName := data.CustomerAccountName
	accountNumber := data.CustomerAccountNumber

	return s.rc.DB.Transaction(func(trx *gorm.DB) error {
		tenantId, _ := initiation.Metadata["nombaSubTenantId"].(string)
		customerCode, _ := initiation.Metadata["nombaSubCustomerCode"].(string)
		planCode, _ := initiation.Metadata["nombaSubPlanCode"].(string)
		planVersionNumber, _ := initiation.Metadata["nombaSubPlanVersion"].(float64)

		customer, err := s.rc.CustomerRepository.Find(
			&models.Customer{TenantID: tenantId, Code: customerCode},
			&repositories.FindArgs{Trx: trx},
		)
		if err != nil || customer == nil {
			return err
		}

		planVersion, err := s.rc.PlanVersionRepository.Find(
			&models.PlanVersion{TenantID: tenantId, Code: planCode, Index: int(planVersionNumber)},
			&repositories.FindArgs{Trx: trx},
		)
		if err != nil || planVersion == nil {
			return err
		}

		last4 := ""
		if len(accountNumber) >= 4 {
			last4 = accountNumber[len(accountNumber)-4:]
		}

		paymentSource, err := s.rc.PaymentSourceRepository.Create(&models.PaymentSource{
			TenantID:   tenantId,
			CustomerID: customer.ID,
			Type:       models.PaymentSourceTypeBank,
			Bank: &models.BankPaymentSource{
				Name:          &accountName,
				Last4:         &last4,
				MandateID:     &mandateId,
				AccountName:   &accountName,
				AccountNumber: &accountNumber,
			},
			Status: models.PaymentSourceStatusActive,
		}, trx)
		if err != nil {
			return err
		}

		subscription := &models.Subscription{
			TenantID:          tenantId,
			CustomerID:        customer.ID,
			PlanID:            planVersion.PlanID,
			PlanVersionID:     planVersion.ID,
			PaymentSourceID:   &paymentSource.ID,
			PaymentSourceType: utils.ToPtr(models.PaymentSourceTypeBank),
			Interval:          planVersion.Interval,
			Amount:            planVersion.Amount,
			IntervalCount:     planVersion.IntervalCount,
			TrialPeriodDays:   planVersion.TrialPeriodDays,
			Currency:          planVersion.Currency,
			InvoiceLimit:      planVersion.InvoiceLimit,
			Status:            models.SubscriptionStatusActive,
		}

		now := time.Now()

		if planVersion.TrialPeriodDays > 0 {
			subscription.TrialStartDate = &now
			trialEnd := now.AddDate(0, 0, planVersion.TrialPeriodDays)
			subscription.TrialEndDate = &trialEnd
			start, end := utils.GetBillingPeriod(trialEnd, planVersion.Interval, planVersion.IntervalCount)
			subscription.CurrentBillingCycleStart = &start
			subscription.CurrentBillingCycleEnd = &end
		} else {
			start, end := utils.GetBillingPeriod(now, planVersion.Interval, planVersion.IntervalCount)
			subscription.CurrentBillingCycleStart = &start
			subscription.CurrentBillingCycleEnd = &end
			subscription.StartedAt = &now
		}

		subscription.Code, err = utils.GenerateCode("SUB")
		if err != nil {
			return err
		}

		subscription, err = s.rc.SubscriptionRepository.Create(subscription, trx)
		if err != nil {
			return err
		}

		initiation.Status = models.NombaInitiationStatusCompleted
		if _, err = s.rc.NombaInitiationRepository.Update(initiation, trx); err != nil {
			return err
		}

		if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantId, models.WebhookDeliveryEventTypeMandateActivated, map[string]interface{}{
			"mandateId":     mandateId,
			"paymentSource": paymentSource,
			"subscription":  subscription,
		}, trx); err != nil {
			log.Printf("direct debit activation: failed to enqueue mandate.activated webhook: %v", err)
		}
		if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantId, models.WebhookDeliveryEventTypeSubscriptionCreated, subscription, trx); err != nil {
			log.Printf("direct debit activation: failed to enqueue subscription.created webhook: %v", err)
		}
		enqueueSubscriptionEmail(s.rc, s.publisher, models.EmailTemplateSubscriptionCreated, subscription, string(models.EmailTemplateSubscriptionCreated)+":"+subscription.ID)
		if subscription.TrialPeriodDays > 0 {
			enqueueSubscriptionEmail(s.rc, s.publisher, models.EmailTemplateTrialStarted, subscription, string(models.EmailTemplateTrialStarted)+":"+subscription.ID)
		}

		return nil
	})
}
