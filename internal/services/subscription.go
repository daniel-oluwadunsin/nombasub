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
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionService struct {
	rc              *repositories.Container
	planService     *PlanService
	customerService *CustomerService
	invoiceService  *InvoiceService
	publisher       *queue.Publisher
	nombaProvider   nomba.Provider
}

func NewSubscriptionService(rc *repositories.Container, planService *PlanService, customerService *CustomerService, invoiceService *InvoiceService, publisher *queue.Publisher, nombaProvider nomba.Provider) *SubscriptionService {
	return &SubscriptionService{
		rc:              rc,
		planService:     planService,
		customerService: customerService,
		invoiceService:  invoiceService,
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
		Status:            models.SubscriptionStatusAttention,
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
		Status:          models.InvoiceStatusOpen,
		AmountDue:       latestPlan.Amount,
		AmountPaid:      0,
		AmountRemaining: latestPlan.Amount,
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

func (s *SubscriptionService) GetSubscriptions(tenantId string, query requests.GetSubscriptionQuery) (*responses.PaginatedResponse[responses.SubscriptionResponse], error) {
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
		customerSearch := "%" + *query.Customer + "%"
		customer, err = customerRepository.FindRaw(&repositories.FindArgs{
			Filter: repositories.NewQueryFilter().Where("tenant_id = ? AND (code = ? OR id = ? OR external_ref = ? or email ILIKE ? OR name ILIKE ?)",
				tenantId,
				*query.Customer,
				*query.Customer,
				*query.Customer,
				customerSearch,
				customerSearch,
			),
		})
		if err != nil {
			return nil, responses.InternalServerError(err)
		}
		if customer == nil {
			page := 1
			limit := 10
			if query.Page != nil {
				page = *query.Page
			}
			if query.Limit != nil {
				limit = *query.Limit
			}
			return responses.NewPaginatedResponse(page, limit, 0, []responses.SubscriptionResponse{}), nil
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
				{Association: "LatestInvoice"},
			},
			OrderBy: []repositories.OrderBy{{Column: "created_at", Desc: true}},
		},
		&query.PaginationQuery,
	)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	data, err := s.formatSubscriptions(response.Data)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &responses.PaginatedResponse[responses.SubscriptionResponse]{
		Data:            data,
		TotalCount:      response.TotalCount,
		Page:            response.Page,
		Limit:           response.Limit,
		TotalPages:      response.TotalPages,
		HasNextPage:     response.HasNextPage,
		HasPreviousPage: response.HasPreviousPage,
	}, nil
}

func (s *SubscriptionService) GenerateCheckoutLink(tenantId, idOrCode string, sendEmail bool) (*responses.GenerateCheckoutLinkResponse, error) {
	subscription, err := s.GetSubscriptionModel(tenantId, idOrCode)
	if err != nil {
		return nil, err
	}
	if !canGenerateCheckoutLink(subscription) {
		return nil, responses.BadRequest("checkout link cannot be generated for this subscription")
	}

	invoice, err := s.checkoutInvoice(subscription)
	if err != nil {
		return nil, err
	}

	link, err := s.invoiceService.CreateCheckoutForSubscription(invoice, subscription, sendEmail)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	subscription.LatestInvoiceID = &invoice.ID
	if _, err := s.rc.SubscriptionRepository.Update(subscription, nil); err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &responses.GenerateCheckoutLinkResponse{CheckoutLink: link}, nil
}

func (s *SubscriptionService) CancelSubscription(tenantId, idOrCode string) error {
	subscription, err := s.GetSubscriptionModel(tenantId, idOrCode)
	if err != nil {
		return err
	}
	if subscription.Status == models.SubscriptionStatusCanceled {
		return responses.BadRequest("subscription is already canceled")
	}

	now := time.Now()
	return s.rc.DB.Transaction(func(trx *gorm.DB) error {
		subscription.Status = models.SubscriptionStatusCanceled
		subscription.CancelledAt = &now
		if _, err := s.rc.SubscriptionRepository.Update(subscription, trx); err != nil {
			return responses.InternalServerError(err)
		}

		if err := trx.Model(&models.Invoice{}).
			Where("tenant_id = ? AND subscription_id = ? AND status IN ?", tenantId, subscription.ID, []models.InvoiceStatus{models.InvoiceStatusDraft, models.InvoiceStatusOpen}).
			Updates(map[string]interface{}{
				"status":         models.InvoiceStatusFailed,
				"failed_at":      now,
				"failure_reason": "subscription canceled",
			}).Error; err != nil {
			return responses.InternalServerError(err)
		}

		if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantId, models.WebhookDeliveryEventTypeSubscriptionCanceled, subscription, trx); err != nil {
			return responses.InternalServerError(err)
		}

		enqueueSubscriptionEmail(
			s.rc,
			s.publisher,
			models.EmailTemplateSubscriptionCanceled,
			subscription,
			string(models.EmailTemplateSubscriptionCanceled)+":"+subscription.ID,
		)

		return nil
	})
}

func (s *SubscriptionService) UpdateDirectDebitMandateStatus(tenantId, idOrCode string, body requests.UpdateMandateStatusRequest) error {
	subscription, err := s.GetSubscriptionModel(tenantId, idOrCode)
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

func (s *SubscriptionService) GetSubscription(tenantId, idOrCode string) (*responses.SubscriptionResponse, error) {
	subscription, err := s.GetSubscriptionModel(tenantId, idOrCode)
	if err != nil {
		return nil, err
	}

	formatted, err := s.formatSubscriptions([]models.Subscription{*subscription})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if len(formatted) == 0 {
		return nil, responses.NotFound("Subscription not found")
	}

	return &formatted[0], nil
}

func (s *SubscriptionService) GetSubscriptionModel(tenantId, idOrCode string) (*models.Subscription, error) {
	subscriptionRepository := s.rc.SubscriptionRepository

	var filter *repositories.QueryFilter

	if uuid.Validate(idOrCode) == nil {
		filter = repositories.NewQueryFilter().Where("tenant_id = ? AND id = ?", tenantId, idOrCode)
	} else {
		filter = repositories.NewQueryFilter().Where("tenant_id = ? AND code = ?", tenantId, idOrCode)
	}

	subscription, err := subscriptionRepository.FindRaw(
		&repositories.FindArgs{
			Filter: filter,
			Preloads: []repositories.Preload{
				{Association: "Customer"},
				{Association: "Plan"},
				{Association: "PaymentSource"},
				{Association: "LatestInvoice"},
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

func (s *SubscriptionService) checkoutInvoice(subscription *models.Subscription) (*models.Invoice, error) {
	if subscription.LatestInvoice != nil {
		return subscription.LatestInvoice, nil
	}
	if subscription.LatestInvoiceID != nil {
		invoice, err := s.rc.InvoiceRepository.FindById(*subscription.LatestInvoiceID, nil)
		if err != nil {
			return nil, responses.InternalServerError(err)
		}
		if invoice != nil {
			return invoice, nil
		}
	}

	invoice := &models.Invoice{
		TenantID:             subscription.TenantID,
		SubscriptionID:       subscription.ID,
		CustomerID:           subscription.CustomerID,
		Status:               models.InvoiceStatusOpen,
		AmountDue:            subscription.Amount,
		AmountPaid:           0,
		AmountRemaining:      subscription.Amount,
		Currency:             subscription.Currency,
		BillingPeriodStart:   subscription.CurrentBillingCycleStart,
		BillingPeriodEnd:     subscription.CurrentBillingCycleEnd,
		DueAt:                subscription.CurrentBillingCycleStart,
		NextPaymentAttemptAt: subscription.CurrentBillingCycleStart,
	}

	code, err := utils.GenerateCode("INV")
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	invoice.Code = code

	invoice, err = s.rc.InvoiceRepository.Create(invoice, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return invoice, nil
}

type subscriptionPaymentFacts struct {
	Payments      int64
	LifetimeValue int64
}

func (s *SubscriptionService) subscriptionPaymentFacts(subscriptionIDs []string) (map[string]subscriptionPaymentFacts, error) {
	facts := make(map[string]subscriptionPaymentFacts, len(subscriptionIDs))
	if len(subscriptionIDs) == 0 {
		return facts, nil
	}

	var rows []struct {
		SubscriptionID string
		Payments       int64
		LifetimeValue  int64
	}
	if err := s.rc.DB.
		Table(models.TableNameInvoices).
		Select("subscription_id, COUNT(*) as payments, COALESCE(SUM(amount_paid), 0) as lifetime_value").
		Where("subscription_id IN ? AND status = ?", subscriptionIDs, models.InvoiceStatusPaid).
		Group("subscription_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		facts[row.SubscriptionID] = subscriptionPaymentFacts{
			Payments:      row.Payments,
			LifetimeValue: row.LifetimeValue,
		}
	}

	return facts, nil
}

func (s *SubscriptionService) formatSubscriptions(subscriptions []models.Subscription) ([]responses.SubscriptionResponse, error) {
	ids := make([]string, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		ids = append(ids, subscription.ID)
	}

	facts, err := s.subscriptionPaymentFacts(ids)
	if err != nil {
		return nil, err
	}

	result := make([]responses.SubscriptionResponse, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		fact := facts[subscription.ID]
		var plan responses.SubscriptionPlanResponse
		if subscription.Plan != nil {
			plan = responses.SubscriptionPlanResponse{
				ID:   subscription.Plan.ID,
				Name: subscription.Plan.Name,
				Code: subscription.Plan.Code,
			}
		}

		var customer responses.SubscriptionCustomerResponse
		if subscription.Customer != nil {
			customer = responses.SubscriptionCustomerResponse{
				ID:    subscription.Customer.ID,
				Code:  subscription.Customer.Code,
				Name:  subscription.Customer.Name,
				Email: subscription.Customer.Email,
			}
		}

		var latestInvoice *responses.SubscriptionInvoiceResponse
		if subscription.LatestInvoice != nil {
			latestInvoice = &responses.SubscriptionInvoiceResponse{
				ID:           subscription.LatestInvoice.ID,
				Code:         subscription.LatestInvoice.Code,
				Status:       subscription.LatestInvoice.Status,
				CheckoutLink: subscription.LatestInvoice.CheckoutLink,
			}
		}

		subscribedFrom := subscription.CreatedAt
		if subscription.StartedAt != nil {
			subscribedFrom = *subscription.StartedAt
		}
		subscribedForDays := int(time.Since(subscribedFrom).Hours() / 24)
		if subscribedForDays < 0 {
			subscribedForDays = 0
		}

		var paymentSource *responses.CustomerPaymentSourceDetail
		if subscription.PaymentSource != nil {
			source := billingPaymentSourceDetail(*subscription.PaymentSource)
			paymentSource = &source
		}

		result = append(result, responses.SubscriptionResponse{
			ID:                       subscription.ID,
			Code:                     subscription.Code,
			Status:                   subscription.Status,
			Amount:                   subscription.Amount,
			Currency:                 subscription.Currency,
			Interval:                 subscription.Interval,
			IntervalCount:            subscription.IntervalCount,
			StartedAt:                subscription.StartedAt,
			CancelledAt:              subscription.CancelledAt,
			PausedAt:                 subscription.PausedAt,
			CurrentBillingCycleStart: subscription.CurrentBillingCycleStart,
			CurrentBillingCycleEnd:   subscription.CurrentBillingCycleEnd,
			LatestInvoiceID:          subscription.LatestInvoiceID,
			AllowRetries:             subscription.AllowRetries,
			CanGenerateCheckoutLink:  canGenerateCheckoutLink(&subscription),
			Payments:                 fact.Payments,
			LifetimeValue:            fact.LifetimeValue,
			SubscribedForDays:        subscribedForDays,
			Plan:                     plan,
			Customer:                 customer,
			PaymentSource:            paymentSource,
			LatestInvoice:            latestInvoice,
			CreatedAt:                subscription.CreatedAt,
			UpdatedAt:                subscription.UpdatedAt,
		})
	}

	return result, nil
}

func billingPaymentSourceDetail(source models.PaymentSource) responses.CustomerPaymentSourceDetail {
	return responses.CustomerPaymentSourceDetail{
		ID:                 source.ID,
		Type:               string(source.Type),
		Status:             string(source.Status),
		CreatedAt:          source.CreatedAt,
		Card:               source.Card,
		Bank:               source.Bank,
		ExpiresSoon:        cardExpiresSoon(source.Card, time.Now()),
		ExpirationMailSent: source.ExpirationMailSent,
	}
}

func canGenerateCheckoutLink(subscription *models.Subscription) bool {
	if subscription.Status == models.SubscriptionStatusCanceled {
		return false
	}
	if subscription.LatestInvoiceID == nil {
		return true
	}
	if subscription.LatestInvoice == nil {
		return true
	}
	if subscription.PaymentSourceID == nil {
		return true
	}

	return subscription.LatestInvoice.Status != models.InvoiceStatusPaid && subscription.LatestInvoice.Status != models.InvoiceStatusRefunded
}
