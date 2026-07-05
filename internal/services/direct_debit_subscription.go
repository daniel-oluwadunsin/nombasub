package services

import (
	"log"
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

// PollPendingMandates is the cron job that checks mandate status for all pending
// direct debit initiations and activates the subscription once the mandate is
// ACTIVE with ADVICE_SENT.
func (s *DirectDebitSubscriptionService) PollPendingMandates() {
	initiations, err := s.rc.NombaInitiationRepository.FindManyRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"purpose = ? AND status = ?",
			models.NombaInitiationPurposeDirectDebitSubscription,
			models.NombaInitiationStatusPending,
		),
	})
	if err != nil {
		log.Printf("direct debit poll cron: failed to load pending initiations: %v", err)
		return
	}

	for _, initiation := range initiations {
		if err := s.processPendingMandate(&initiation); err != nil {
			log.Printf("direct debit poll cron: failed for mandateId=%s: %v", initiation.Reference, err)
		}
	}
}

func (s *DirectDebitSubscriptionService) processPendingMandate(initiation *models.NombaInitiation) error {
	mandateId := initiation.Reference

	statusResp, err := s.nombaProvider.GetDirectDebitManadateStatus(mandateId)
	if err != nil {
		return err
	}

	data := statusResp.Data

	if data.MandateStatus == nomba.MandateStatusDeleted {
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

	if data.MandateStatus != nomba.MandateStatusActive || data.MandateAdviceStatus != "ADVICE_SENT" {
		return nil
	}

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
