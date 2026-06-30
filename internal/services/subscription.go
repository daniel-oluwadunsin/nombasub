package services

import (
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
)

type SubscriptionService struct {
	rc              *repositories.Container
	planService     *PlanService
	customerService *CustomerService
}

func NewSubscriptionService(rc *repositories.Container, planService *PlanService) *SubscriptionService {
	return &SubscriptionService{rc: rc, planService: planService}
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

		if customerPaymentSource == nil {
			return nil, responses.NotFound("Customer does not have any payment method attached")
		}
	}

	subscription := &models.Subscription{
		TenantID:          tenantId,
		CustomerID:        customer.ID,
		PlanID:            latestPlan.PlanID,
		PlanVersionID:     latestPlan.ID,
		PaymentSourceID:   customerPaymentSource.CustomerID,
		PaymentSourceType: models.PaymentSourceTypeCard,
		Interval:          latestPlan.Interval,
		Amount:            latestPlan.Amount,
		IntervalCount:     latestPlan.IntervalCount,
		TrialPeriodDays:   latestPlan.TrialPeriodDays,
		Currency:          latestPlan.Currency,
		InvoiceLimit:      latestPlan.InvoiceLimit,
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

	subscription, err = subscriptionRepository.Create(subscription, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return subscription, nil
}
