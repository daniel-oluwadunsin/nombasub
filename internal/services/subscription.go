package services

import (
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
)

type SubscriptionService struct {
	rc              *repositories.Container
	planService     *PlanService
	customerService *CustomerService
	publisher       *queue.Publisher
	nombaProvider   nomba.Provider
}

func NewSubscriptionService(rc *repositories.Container, planService *PlanService, customerService *CustomerService, publisher *queue.Publisher, nombaProvider nomba.Provider) *SubscriptionService {
	return &SubscriptionService{
		rc:              rc,
		planService:     planService,
		customerService: customerService,
		publisher:       publisher,
		nombaProvider:   nombaProvider,
	}
}

func (s *SubscriptionService) CreateSubscription(tenantId string, body requests.CreateSubscriptionRequest) (*models.Subscription, error) {
	paymentSourceRepository := s.rc.PaymentSourceRepository
	subscriptionRepository := s.rc.SubscriptionRepository

	latestPlan, err := s.planService.GetPlanLatestVersion(tenantId, body.PlanCode)
	if err != nil {
		return nil, err
	}

	customer, err := s.customerService.GetCustomer(tenantId, body.CustomerEmailOrCode)
	if err != nil {
		return nil, err
	}

	var customerPaymentSource *models.PaymentSource

	if body.CardToken != nil {
		customerPaymentSource, err = paymentSourceRepository.Find(&models.PaymentSource{
			TenantID:   tenantId,
			CustomerID: customer.ID,
			Card: &models.CardPaymentSource{
				AuthorizationToken: body.CardToken,
			},
			Status: models.PaymentSourceStatusActive,
		}, nil)

		if err != nil {
			return nil, responses.InternalServerError(err)
		}

		if customerPaymentSource == nil {
			return nil, responses.NotFound("card token is invalid")
		}
	} else if body.MandateID != nil {
		customerPaymentSource, err = paymentSourceRepository.Find(&models.PaymentSource{
			TenantID:   tenantId,
			CustomerID: customer.ID,
			Bank: &models.BankPaymentSource{
				MandateID: body.MandateID,
			},
			Status: models.PaymentSourceStatusActive,
		}, nil)

		if err != nil {
			return nil, responses.InternalServerError(err)
		}

		if customerPaymentSource == nil {
			return nil, responses.NotFound("provided mandate is invalid")
		}
	} else {
		customerPaymentSource, err = paymentSourceRepository.Find(&models.PaymentSource{
			TenantID:   tenantId,
			CustomerID: customer.ID,
			Status:     models.PaymentSourceStatusActive,
		}, &repositories.FindArgs{
			OrderBy: []repositories.OrderBy{{Column: "created_at", Desc: true}},
		})

		if err != nil {
			return nil, responses.InternalServerError(err)
		}
	}

	var paymentSourceID *string
	var paymentSourceType *models.PaymentSourceType
	if customerPaymentSource != nil {
		paymentSourceID = &customerPaymentSource.ID
		paymentSourceType = &customerPaymentSource.Type
	}

	subscription := &models.Subscription{
		TenantID:          tenantId,
		CustomerID:        customer.ID,
		PlanID:            latestPlan.PlanID,
		PlanVersionID:     latestPlan.ID,
		PaymentSourceID:   paymentSourceID,
		PaymentSourceType: paymentSourceType,
		Interval:          latestPlan.Interval,
		Amount:            latestPlan.Amount,
		IntervalCount:     latestPlan.IntervalCount,
		TrialPeriodDays:   latestPlan.TrialPeriodDays,
		Currency:          latestPlan.Currency,
		InvoiceLimit:      latestPlan.InvoiceLimit,
		AllowRetries:      body.AllowRetries,
	}

	if latestPlan.TrialPeriodDays != 0 {
		subscription.TrialStartDate = utils.ToPtr(time.Now())
		subscription.TrialEndDate = utils.ToPtr(time.Now().AddDate(0, 0, latestPlan.TrialPeriodDays))

		startDate, endDate := utils.GetBillingPeriod(*subscription.TrialEndDate, latestPlan.Interval, latestPlan.IntervalCount)
		subscription.CurrentBillingCycleStart = &startDate
		subscription.CurrentBillingCycleEnd = &endDate
	}

	if latestPlan.TrialPeriodDays == 0 {
		startDate, endDate := utils.GetBillingPeriod(time.Now(), latestPlan.Interval, latestPlan.IntervalCount)
		subscription.CurrentBillingCycleStart = &startDate
		subscription.CurrentBillingCycleEnd = &endDate
		subscription.StartedAt = utils.ToPtr(time.Now())
	}

	subscription.Code, err = utils.GenerateCode("SUB")
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	subscription, err = subscriptionRepository.Create(subscription, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	invoice := &models.Invoice{
		TenantID:        tenantId,
		SubscriptionID:  subscription.ID,
		CustomerID:      customer.ID,
		Status:          models.InvoiceStatusPaid,
		AmountDue:       latestPlan.Amount,
		AmountPaid:      latestPlan.Amount,
		AmountRemaining: 0,
		Currency:        latestPlan.Currency,
		DueAt:           subscription.CurrentBillingCycleStart,
	}

	invoice.Code, err = utils.GenerateCode("INV")
	if err != nil {
		return nil, err
	}

	invoice, err = s.rc.InvoiceRepository.Create(invoice, nil)
	if err != nil {
		return nil, err
	}

	subscription.LatestInvoiceID = &invoice.ID
	_, err = s.rc.SubscriptionRepository.Update(subscription, nil)
	if err != nil {
		return nil, err
	}

	if err := queue.EnqueueTenantWebhook(
		s.rc,
		s.publisher,
		tenantId,
		models.WebhookDeliveryEventTypeSubscriptionCreated,
		subscription,
		nil,
	); err != nil {
		return nil, responses.InternalServerError(err)
	}

	enqueueSubscriptionEmail(
		s.rc,
		s.publisher,
		models.EmailTemplateSubscriptionCreated,
		subscription,
		string(models.EmailTemplateSubscriptionCreated)+":"+subscription.ID,
	)
	if subscription.TrialPeriodDays > 0 {
		enqueueSubscriptionEmail(
			s.rc,
			s.publisher,
			models.EmailTemplateTrialStarted,
			subscription,
			string(models.EmailTemplateTrialStarted)+":"+subscription.ID,
		)
	}

	return subscription, nil
}

func (s *SubscriptionService) GetSubscriptions(tenantId string, query requests.GetSubscriptionQuery) (*responses.PaginatedResponse[models.Subscription], error) {
	subscriptionRepository := s.rc.SubscriptionRepository
	planRepository := s.rc.PlanRepository
	customerRepository := s.rc.CustomerRepository
	filter := &models.Subscription{TenantID: tenantId}

	var customer *models.Customer
	var plan *models.Plan
	var err error

	if query.Plan != nil {
		plan, err = planRepository.FindRaw(&repositories.FindArgs{
			Filter: repositories.NewQueryFilter().Where("tenant_id = ? AND (code = ? OR id = ?)", tenantId, *query.Plan, *query.Plan),
		})
		if err != nil {
			return nil, responses.InternalServerError(err)
		}
		if plan == nil {
			return nil, responses.NotFound("Plan does not exist")
		}
		filter.PlanID = plan.ID
	}

	if query.Customer != nil {
		customer, err = customerRepository.FindRaw(&repositories.FindArgs{
			Filter: repositories.NewQueryFilter().Where("tenant_id = ? AND (code = ? OR id = ? OR external_ref = ? or email ILIKE ?)",
				tenantId,
				*query.Customer,
				*query.Customer,
				*query.Customer,
				*query.Customer,
			),
		})
		if err != nil {
			return nil, responses.InternalServerError(err)
		}
		if customer == nil {
			return nil, responses.NotFound("Customer does not exist")
		}
		filter.CustomerID = customer.ID
	}

	response, err := subscriptionRepository.FindManyPaginated(
		filter,
		&repositories.FindArgs{
			Preloads: []repositories.Preload{
				{Association: "Customer"},
				{Association: "Plan"},
				{Association: "PaymentSource"},
			},
			OrderBy: []repositories.OrderBy{{Column: "created_at", Desc: true}},
		},
		&query.PaginationQuery,
	)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return response, nil
}

func (s *SubscriptionService) UpdateDirectDebitMandateStatus(tenantId, idOrCode string, body requests.UpdateMandateStatusRequest) error {
	subscription, err := s.GetSubscription(tenantId, idOrCode)
	if err != nil {
		return err
	}

	if subscription.PaymentSourceType == nil || *subscription.PaymentSourceType != models.PaymentSourceTypeBank {
		return responses.BadRequest("subscription does not have a direct debit payment source")
	}

	paymentSource, err := s.rc.PaymentSourceRepository.FindById(*subscription.PaymentSourceID, nil)
	if err != nil {
		return responses.InternalServerError(err)
	}
	if paymentSource == nil || paymentSource.Bank == nil || paymentSource.Bank.MandateID == nil {
		return responses.NotFound("mandate not found for this subscription")
	}

	_, err = s.nombaProvider.UpdateDirectDebitStatus(nomba.UpdateDirectDebitManadateRequest{
		MandateId:     *paymentSource.Bank.MandateID,
		MandateStatus: nomba.MandateStatus(body.Status),
	})
	if err != nil {
		return responses.InternalServerError(err)
	}

	now := time.Now()
	switch body.Status {
	case "SUSPENDED":
		subscription.Status = models.SubscriptionStatusPaused
		subscription.PausedAt = &now
	case "DELETED":
		subscription.Status = models.SubscriptionStatusCanceled
		subscription.CancelledAt = &now
		paymentSource.Status = models.PaymentSourceStatusInactive
		if _, err := s.rc.PaymentSourceRepository.Update(paymentSource, nil); err != nil {
			return responses.InternalServerError(err)
		}
	}

	if _, err := s.rc.SubscriptionRepository.Update(subscription, nil); err != nil {
		return responses.InternalServerError(err)
	}

	mandateEventType := models.WebhookDeliveryEventTypeMandateSuspended
	subscriptionEventType := models.WebhookDeliveryEventTypeSubscriptionPaused
	if body.Status == "DELETED" {
		mandateEventType = models.WebhookDeliveryEventTypeMandateDeleted
		subscriptionEventType = models.WebhookDeliveryEventTypeSubscriptionCanceled
	}

	mandatePayload := map[string]interface{}{
		"mandateId":    *paymentSource.Bank.MandateID,
		"subscription": subscription,
	}
	if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantId, mandateEventType, mandatePayload, nil); err != nil {
		_ = err
	}
	if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantId, subscriptionEventType, subscription, nil); err != nil {
		_ = err
	}

	return nil
}

func (s *SubscriptionService) GetSubscription(tenantId, idOrCode string) (*models.Subscription, error) {
	subscriptionRepository := s.rc.SubscriptionRepository

	subscription, err := subscriptionRepository.FindRaw(
		&repositories.FindArgs{
			Filter: repositories.NewQueryFilter().Where("tenant_id = ? AND (id = ? or code = ?)", tenantId, idOrCode, idOrCode),
			Preloads: []repositories.Preload{
				{Association: "Customer"},
				{Association: "Plan"},
				{Association: "PaymentSource"},
			},
		},
	)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if subscription == nil {
		return nil, responses.NotFound("Subscription not found")
	}

	return subscription, nil
}
