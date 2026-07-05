package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"gorm.io/gorm"
)

type CustomerService struct {
	rc        *repositories.Container
	publisher *queue.Publisher
}

func NewCustomerService(rc *repositories.Container, publisher *queue.Publisher) *CustomerService {
	return &CustomerService{rc: rc, publisher: publisher}
}

func (s *CustomerService) CreateCustomer(tenantId string, body requests.CreateCustomerRequest) (*models.Customer, error) {
	customerRepository := s.rc.CustomerRepository

	existingCustomer, err := customerRepository.ExistsRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where("tenant_id = ? AND email ILIKE ?", tenantId, body.Email),
	})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if existingCustomer {
		return nil, responses.Conflict("A customer with this email already exists")
	}

	code, err := utils.GenerateCode("CUST")
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	customer, err := customerRepository.Create(&models.Customer{
		TenantID:    tenantId,
		Name:        body.Name,
		Email:       body.Email,
		PhoneNumber: body.PhoneNumber,
		ExternalRef: body.ExternalRef,
		Code:        code,
	}, nil)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return customer, nil
}

func (s *CustomerService) GetCustomer(tenantId string, emailOrCode string) (*models.Customer, error) {
	return s.findCustomer(tenantId, emailOrCode)
}

func (s *CustomerService) GetCustomerDetails(tenantId string, emailOrCode string) (*responses.CustomerDetailResponse, error) {
	customerRepository := s.rc.CustomerRepository

	customer, err := customerRepository.FindRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"tenant_id = ? AND (email ILIKE ? OR code = ?)",
			tenantId,
			emailOrCode,
			emailOrCode,
		),
	})

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if customer == nil {
		return nil, responses.NotFound("Customer not found")
	}

	return s.buildCustomerDetail(customer)
}

func (s *CustomerService) GetCustomers(tenantId string, query requests.GetCustomersRequest) (*responses.PaginatedResponse[models.Customer], error) {
	customerRepository := s.rc.CustomerRepository

	queryFilter, err := buildCustomerQueryFilter(query)
	if err != nil {
		return nil, err
	}

	response, err := customerRepository.FindManyPaginated(
		&models.Customer{TenantID: tenantId},
		&repositories.FindArgs{
			Filter:  queryFilter,
			OrderBy: []repositories.OrderBy{{Column: "created_at", Desc: true}},
		},
		&query.PaginationQuery,
	)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return response, nil
}

func (s *CustomerService) UpdateCustomer(tenantId string, emailOrCode string, body requests.UpdateCustomerRequest) (*models.Customer, error) {
	customerRepository := s.rc.CustomerRepository

	customer, err := customerRepository.FindRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"tenant_id = ? AND (email ILIKE ? OR code = ?)",
			tenantId,
			emailOrCode,
			emailOrCode,
		),
	})

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if customer == nil {
		return nil, responses.NotFound("Customer not found")
	}

	if body.Email != nil && strings.TrimSpace(*body.Email) != "" && !strings.EqualFold(customer.Email, strings.TrimSpace(*body.Email)) {
		exists, err := customerRepository.ExistsRaw(&repositories.FindArgs{
			Filter: repositories.NewQueryFilter().Where(
				"tenant_id = ? AND email ILIKE ? AND id <> ?",
				tenantId,
				strings.TrimSpace(*body.Email),
				customer.ID,
			),
		})
		if err != nil {
			return nil, responses.InternalServerError(err)
		}
		if exists {
			return nil, responses.Conflict("A customer with this email already exists")
		}
		customer.Email = strings.TrimSpace(*body.Email)
	}

	if body.Name != nil {
		customer.Name = body.Name
	}

	if body.PhoneNumber != nil {
		customer.PhoneNumber = body.PhoneNumber
	}

	if body.ExternalRef != nil {
		customer.ExternalRef = body.ExternalRef
	}

	updatedCustomer, err := customerRepository.Update(customer, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return updatedCustomer, nil
}

func (s *CustomerService) RemindCustomerCardExpiring(tenantId string, emailOrCode string, paymentSourceID string) error {
	customer, err := s.findCustomer(tenantId, emailOrCode)
	if err != nil {
		return err
	}

	paymentSource, err := s.rc.PaymentSourceRepository.FindRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"tenant_id = ? AND customer_id = ? AND id = ? AND type = ?",
			tenantId,
			customer.ID,
			paymentSourceID,
			models.PaymentSourceTypeCard,
		),
	})
	if err != nil {
		return responses.InternalServerError(err)
	}
	if paymentSource == nil {
		return responses.NotFound("Card payment source not found")
	}

	tenant, err := s.rc.TenantRepository.FindById(tenantId, nil)
	if err != nil {
		return responses.InternalServerError(err)
	}
	if tenant == nil {
		return responses.NotFound("Tenant not found")
	}

	ctx := models.EmailContext{
		GreetingName:  customerDisplayName(customer),
		BusinessName:  valueOrDefault(tenant.BusinessName, "Nomba merchant"),
		CustomerEmail: customer.Email,
	}
	if paymentSource.Card != nil {
		if paymentSource.Card.Last4Digits != nil {
			ctx.CardLast4 = *paymentSource.Card.Last4Digits
		}
		if paymentSource.Card.ExpiryMonth != nil && paymentSource.Card.ExpiryYear != nil {
			ctx.CardExpiry = fmt.Sprintf("%s/%s", *paymentSource.Card.ExpiryMonth, *paymentSource.Card.ExpiryYear)
		}
	}
	applyTemplateCopy(models.EmailTemplateSubscriptionCardExpiring, &ctx)

	if err := queue.EnqueueEmail(
		s.rc,
		s.publisher,
		customer.Email,
		emailSubject(models.EmailTemplateSubscriptionCardExpiring, ctx),
		models.EmailTemplateSubscriptionCardExpiring,
		ctx,
		fmt.Sprintf("%s:%s:%s:manual", models.EmailTemplateSubscriptionCardExpiring, customer.ID, paymentSource.ID),
	); err != nil {
		return responses.InternalServerError(err)
	}

	paymentSource.ExpirationMailSent = true
	if _, err := s.rc.PaymentSourceRepository.Update(paymentSource, nil); err != nil {
		return responses.InternalServerError(err)
	}

	return nil
}

func (s *CustomerService) findCustomer(tenantId string, emailOrCode string) (*models.Customer, error) {
	customer, err := s.rc.CustomerRepository.FindRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"tenant_id = ? AND (email ILIKE ? OR code = ?)",
			tenantId,
			emailOrCode,
			emailOrCode,
		),
	})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if customer == nil {
		return nil, responses.NotFound("Customer not found")
	}
	return customer, nil
}

func (s *CustomerService) buildCustomerDetail(customer *models.Customer) (*responses.CustomerDetailResponse, error) {
	var subscriptions []models.Subscription
	if err := s.rc.DB.
		Where("tenant_id = ? AND customer_id = ?", customer.TenantID, customer.ID).
		Preload("Plan").
		Preload("PaymentSource").
		Order("created_at DESC").
		Find(&subscriptions).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var paymentSources []models.PaymentSource
	if err := s.rc.DB.
		Where("tenant_id = ? AND customer_id = ?", customer.TenantID, customer.ID).
		Order("created_at DESC").
		Find(&paymentSources).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	invoiceStats, err := s.customerInvoiceStats(customer.TenantID, customer.ID)
	if err != nil {
		return nil, err
	}
	paidCounts, lastPaid, err := s.subscriptionInvoiceFacts(customer.TenantID, customer.ID)
	if err != nil {
		return nil, err
	}

	sourceByID := map[string]responses.CustomerPaymentSourceDetail{}
	paymentSourceResponses := make([]responses.CustomerPaymentSourceDetail, 0, len(paymentSources))
	for _, source := range paymentSources {
		response := paymentSourceDetail(source)
		sourceByID[source.ID] = response
		paymentSourceResponses = append(paymentSourceResponses, response)
	}

	subscriptionResponses := make([]responses.CustomerSubscriptionResponse, 0, len(subscriptions))
	var activeSubscriptions int64
	for _, subscription := range subscriptions {
		if subscription.Status == models.SubscriptionStatusActive {
			activeSubscriptions++
		}

		plan := responses.CustomerPlanResponse{}
		if subscription.Plan != nil {
			plan = responses.CustomerPlanResponse{ID: subscription.Plan.ID, Name: subscription.Plan.Name, Code: subscription.Plan.Code}
		}

		var source *responses.CustomerPaymentSourceDetail
		if subscription.PaymentSourceID != nil {
			if found, ok := sourceByID[*subscription.PaymentSourceID]; ok {
				foundCopy := found
				source = &foundCopy
			} else if subscription.PaymentSource != nil {
				found := paymentSourceDetail(*subscription.PaymentSource)
				source = &found
			}
		}

		subscriptionResponses = append(subscriptionResponses, responses.CustomerSubscriptionResponse{
			ID:                subscription.ID,
			Code:              subscription.Code,
			Status:            string(subscription.Status),
			Amount:            subscription.Amount,
			Currency:          subscription.Currency,
			Interval:          string(subscription.Interval),
			IntervalCount:     subscription.IntervalCount,
			StartedAt:         subscription.StartedAt,
			NextChargeAt:      subscription.CurrentBillingCycleEnd,
			LastChargedAt:     lastPaid[subscription.ID],
			TotalInvoicesPaid: paidCounts[subscription.ID],
			Plan:              plan,
			PaymentSource:     source,
		})
	}

	return &responses.CustomerDetailResponse{
		Customer: responses.CustomerProfileResponse{
			ID:          customer.ID,
			Code:        customer.Code,
			Name:        customer.Name,
			Email:       customer.Email,
			PhoneNumber: customer.PhoneNumber,
			ExternalRef: customer.ExternalRef,
			CreatedAt:   customer.CreatedAt,
			UpdatedAt:   customer.UpdatedAt,
		},
		Summary: responses.CustomerSummaryResponse{
			LifetimeValue:         invoiceStats.LifetimeValue,
			Currency:              invoiceStats.Currency,
			ActiveSubscriptions:   activeSubscriptions,
			EarliestInvoicePaidAt: invoiceStats.EarliestPaidAt,
			DateJoined:            customer.CreatedAt,
			TotalPaidInvoices:     invoiceStats.TotalPaidInvoices,
			TotalSubscriptions:    len(subscriptions),
			TotalPaymentSources:   len(paymentSources),
		},
		Subscriptions:  subscriptionResponses,
		PaymentSources: paymentSourceResponses,
	}, nil
}

type customerInvoiceStats struct {
	LifetimeValue     int64
	TotalPaidInvoices int64
	EarliestPaidAt    *time.Time
	Currency          string
}

func (s *CustomerService) customerInvoiceStats(tenantID, customerID string) (customerInvoiceStats, error) {
	var stats customerInvoiceStats
	var row struct {
		LifetimeValue     int64
		TotalPaidInvoices int64
		EarliestPaidAt    *time.Time
		Currency          string
	}
	if err := s.rc.DB.Model(&models.Invoice{}).
		Select("COALESCE(SUM(amount_paid), 0) as lifetime_value, COUNT(*) as total_paid_invoices, MIN(paid_at) as earliest_paid_at, COALESCE(MAX(currency), 'NGN') as currency").
		Where("tenant_id = ? AND customer_id = ? AND status = ?", tenantID, customerID, models.InvoiceStatusPaid).
		Scan(&row).Error; err != nil {
		return stats, responses.InternalServerError(err)
	}
	stats.LifetimeValue = row.LifetimeValue
	stats.TotalPaidInvoices = row.TotalPaidInvoices
	stats.EarliestPaidAt = row.EarliestPaidAt
	stats.Currency = row.Currency
	if stats.Currency == "" {
		stats.Currency = "NGN"
	}
	return stats, nil
}

func (s *CustomerService) subscriptionInvoiceFacts(tenantID, customerID string) (map[string]int64, map[string]*time.Time, error) {
	var rows []struct {
		SubscriptionID string
		PaidCount      int64
		LastPaidAt     *time.Time
	}
	if err := s.rc.DB.Model(&models.Invoice{}).
		Select("subscription_id, COUNT(*) as paid_count, MAX(paid_at) as last_paid_at").
		Where("tenant_id = ? AND customer_id = ? AND status = ?", tenantID, customerID, models.InvoiceStatusPaid).
		Group("subscription_id").
		Scan(&rows).Error; err != nil {
		return nil, nil, responses.InternalServerError(err)
	}

	paidCounts := map[string]int64{}
	lastPaid := map[string]*time.Time{}
	for _, row := range rows {
		paidCounts[row.SubscriptionID] = row.PaidCount
		lastPaid[row.SubscriptionID] = row.LastPaidAt
	}
	return paidCounts, lastPaid, nil
}

func paymentSourceDetail(source models.PaymentSource) responses.CustomerPaymentSourceDetail {
	return responses.CustomerPaymentSourceDetail{
		ID:                 source.ID,
		Type:               string(source.Type),
		Status:             string(source.Status),
		CreatedAt:          source.CreatedAt,
		Card:               source.Card,
		Bank:               source.Bank,
		ExpiresSoon:        cardExpiresWithin(source.Card, time.Now(), 6),
		ExpirationMailSent: source.ExpirationMailSent,
	}
}

func buildCustomerQueryFilter(query requests.GetCustomersRequest) (*repositories.QueryFilter, error) {
	clauses := []string{}
	args := []interface{}{}

	if query.Search != nil && strings.TrimSpace(*query.Search) != "" {
		search := "%" + strings.TrimSpace(*query.Search) + "%"
		clauses = append(clauses, "(name ILIKE ? OR email ILIKE ? OR code ILIKE ? OR phone_number ILIKE ?)")
		args = append(args, search, search, search, search)
	}
	if query.From != nil && strings.TrimSpace(*query.From) != "" {
		from, err := parseCustomerDate(*query.From, false)
		if err != nil {
			return nil, responses.BadRequest("from must use YYYY-MM-DD format")
		}
		clauses = append(clauses, "created_at >= ?")
		args = append(args, *from)
	}
	if query.To != nil && strings.TrimSpace(*query.To) != "" {
		to, err := parseCustomerDate(*query.To, true)
		if err != nil {
			return nil, responses.BadRequest("to must use YYYY-MM-DD format")
		}
		clauses = append(clauses, "created_at <= ?")
		args = append(args, *to)
	}
	if len(clauses) == 0 {
		return nil, nil
	}
	return repositories.NewQueryFilter().Where(strings.Join(clauses, " AND "), args...), nil
}

func parseCustomerDate(value string, endOfDay bool) (*time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}

func cardExpiresWithin(card *models.CardPaymentSource, now time.Time, months int) bool {
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
	limit := now.AddDate(0, months, 0)
	return expiryBoundary.After(now) && (expiryBoundary.Before(limit) || expiryBoundary.Equal(limit))
}

func (s *CustomerService) GetOrCreateCustomer(tenantId string, customer models.Customer, trx *gorm.DB) (*models.Customer, error) {
	if customer.Email == "" {
		return nil, responses.BadRequest("Customer email is required")
	}

	customerRepository := s.rc.CustomerRepository

	customerDetails, err := customerRepository.FindRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"tenant_id = ? AND email ILIKE ?",
			tenantId,
			customer.Email,
		),
		Trx: trx,
	})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if customerDetails == nil {
		code, err := utils.GenerateCode("CUST")
		if err != nil {
			return nil, responses.InternalServerError(err)
		}

		customer.Code = code
		customerDetails, err = customerRepository.Create(&customer, trx)
		if err != nil {
			return nil, responses.InternalServerError(err)
		}
	}

	return customerDetails, nil
}
