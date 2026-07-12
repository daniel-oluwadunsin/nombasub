package services

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"gorm.io/gorm"
)

// maxPortalCodeAttempts is the number of incorrect portal sign-in code guesses
// allowed before the code is locked and the customer must request a new one.
const maxPortalCodeAttempts = 5

type PortalService struct {
	rc                  *repositories.Container
	publisher           *queue.Publisher
	cfg                 *config.Config
	nombaProvider       nomba.Provider
	subscriptionService *SubscriptionService
	invoiceService      *InvoiceService
}

func NewPortalService(rc *repositories.Container, publisher *queue.Publisher, cfg *config.Config, nombaProvider nomba.Provider, subscriptionService *SubscriptionService, invoiceService *InvoiceService) *PortalService {
	return &PortalService{
		rc:                  rc,
		publisher:           publisher,
		cfg:                 cfg,
		nombaProvider:       nombaProvider,
		subscriptionService: subscriptionService,
		invoiceService:      invoiceService,
	}
}

func (s *PortalService) InitiateSession(body requests.InitiatePortalSessionRequest) (*responses.PortalSessionInitiatedResponse, error) {
	tenant, customer, err := s.portalTenantAndCustomer(body.TenantID, body.CustomerID)
	if err != nil {
		return nil, err
	}

	code, err := utils.GenerateNumericString(6)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	codeHash, err := utils.Hash(code)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	codeExpiresAt := time.Now().Add(10 * time.Minute)
	session, err := s.rc.PortalSessionRepository.Create(&models.PortalSession{
		TenantID:      tenant.ID,
		CustomerID:    customer.ID,
		CodeHash:      &codeHash,
		CodeExpiresAt: &codeExpiresAt,
	}, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	ctx := models.EmailContext{
		Title:         "Your customer portal code",
		Preheader:     "Use this code to sign in to your subscription portal.",
		GreetingName:  customerDisplayName(customer),
		Intro:         "Use this code to continue to your customer portal.",
		Body:          "This code expires in 10 minutes. If you did not request it, you can safely ignore this email.",
		BusinessName:  valueOrDefault(tenant.BusinessName, "Nomba merchant"),
		CustomerEmail: customer.Email,
		PortalURL:     buildCustomerPortalURL(s.cfg.ClientURL, tenant.ID, customer.ID),
		PortalCode:    code,
		SecondaryNote: "For your security, do not share this code with anyone.",
	}

	if err := queue.EnqueueEmail(
		s.rc,
		s.publisher,
		customer.Email,
		fmt.Sprintf("Your %s portal sign-in code", ctx.BusinessName),
		models.EmailTemplatePortalSessionCode,
		ctx,
		fmt.Sprintf("%s:%s", models.EmailTemplatePortalSessionCode, session.ID),
	); err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &responses.PortalSessionInitiatedResponse{
		SessionID:     session.ID,
		CustomerEmail: customer.Email,
		CodeExpiresAt: session.CodeExpiresAt,
	}, nil
}

func (s *PortalService) VerifySession(body requests.VerifyPortalSessionRequest) (*responses.PortalSessionResponse, error) {
	tenant, customer, err := s.portalTenantAndCustomer(body.TenantID, body.CustomerID)
	if err != nil {
		return nil, err
	}

	session, err := s.rc.PortalSessionRepository.Find(&models.PortalSession{
		TenantID:   tenant.ID,
		CustomerID: customer.ID,
	}, &repositories.FindArgs{
		OrderBy: []repositories.OrderBy{{Column: "created_at", Desc: true}},
	})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if session == nil || session.CodeHash == nil || session.CodeExpiresAt == nil {
		return nil, responses.BadRequest("No active portal session code found")
	}
	if session.VerifiedAt != nil {
		return nil, responses.BadRequest("Portal session code has already been used")
	}
	if session.CodeExpiresAt.Before(time.Now()) {
		return nil, responses.BadRequest("Portal session code has expired")
	}
	if session.FailedAttempts >= maxPortalCodeAttempts {
		return nil, responses.Unauthorized("Too many incorrect attempts; request a new code")
	}
	if !utils.ValidateHash(*session.CodeHash, body.Code) {
		// Count the failed attempt and lock the code once the limit is hit so a
		// 6-digit code can't be brute-forced within its validity window.
		session.FailedAttempts++
		if session.FailedAttempts >= maxPortalCodeAttempts {
			expired := time.Now()
			session.CodeExpiresAt = &expired
		}
		if _, updateErr := s.rc.PortalSessionRepository.Update(session, nil); updateErr != nil {
			return nil, responses.InternalServerError(updateErr)
		}
		return nil, responses.Unauthorized("Invalid portal session code")
	}

	now := time.Now()
	tokenExpiresAt := now.Add(24 * time.Hour)
	token, err := utils.GeneratePortalJwt(tenant.ID, customer.ID, session.ID, tokenExpiresAt, s.cfg)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	tokenHash := utils.DigestToken(token)

	session.VerifiedAt = &now
	session.AccessTokenHash = &tokenHash
	session.AccessTokenExpiresAt = &tokenExpiresAt
	if _, err := s.rc.PortalSessionRepository.Update(session, nil); err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &responses.PortalSessionResponse{
		AccessToken: token,
		ExpiresAt:   session.AccessTokenExpiresAt,
		Tenant: responses.PortalTenantResponse{
			ID:           tenant.ID,
			BusinessName: tenant.BusinessName,
			AccountID:    tenant.AccountID,
		},
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
	}, nil
}

func (s *PortalService) CurrentSession(tenantID, customerID string) (*responses.PortalSessionResponse, error) {
	tenant, customer, err := s.portalTenantAndCustomer(tenantID, customerID)
	if err != nil {
		return nil, err
	}

	return &responses.PortalSessionResponse{
		AccessToken: "",
		ExpiresAt:   nil,
		Tenant: responses.PortalTenantResponse{
			ID:           tenant.ID,
			BusinessName: tenant.BusinessName,
			AccountID:    tenant.AccountID,
		},
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
	}, nil
}

func (s *PortalService) UpdateProfile(tenantID, customerID string, body requests.UpdatePortalProfileRequest) (*responses.PortalSessionResponse, error) {
	tenant, customer, err := s.portalTenantAndCustomer(tenantID, customerID)
	if err != nil {
		return nil, err
	}

	if body.Name != nil {
		customer.Name = body.Name
	}
	if body.PhoneNumber != nil {
		customer.PhoneNumber = body.PhoneNumber
	}

	updatedCustomer, err := s.rc.CustomerRepository.Update(customer, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantID, models.WebhookDeliveryEventTypeCustomerUpdated, updatedCustomer, nil); err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &responses.PortalSessionResponse{
		AccessToken: "",
		ExpiresAt:   nil,
		Tenant: responses.PortalTenantResponse{
			ID:           tenant.ID,
			BusinessName: tenant.BusinessName,
			AccountID:    tenant.AccountID,
		},
		Customer: responses.CustomerProfileResponse{
			ID:          updatedCustomer.ID,
			Code:        updatedCustomer.Code,
			Name:        updatedCustomer.Name,
			Email:       updatedCustomer.Email,
			PhoneNumber: updatedCustomer.PhoneNumber,
			ExternalRef: updatedCustomer.ExternalRef,
			CreatedAt:   updatedCustomer.CreatedAt,
			UpdatedAt:   updatedCustomer.UpdatedAt,
		},
	}, nil
}

func (s *PortalService) Analytics(tenantID, customerID string) (*responses.PortalAnalyticsResponse, error) {
	var subscriptionRows []struct {
		Status string
		Count  int64
	}
	if err := s.rc.DB.
		Table(models.TableNameSubscription).
		Select("status, COUNT(*) as count").
		Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Group("status").
		Scan(&subscriptionRows).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var sourceRows []struct {
		Type  string
		Count int64
	}
	if err := s.rc.DB.
		Table(models.TableNamePaymentSource).
		Select("type, COUNT(*) as count").
		Where("tenant_id = ? AND customer_id = ? AND status = ?", tenantID, customerID, models.PaymentSourceStatusActive).
		Group("type").
		Scan(&sourceRows).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var trendRows []struct {
		Date     string
		Amount   int64
		Currency string
	}
	if err := s.rc.DB.
		Table(models.TableNameInvoices).
		Select("TO_CHAR(DATE_TRUNC('day', paid_at), 'YYYY-MM-DD') as date, COALESCE(SUM(amount_paid), 0) as amount, MAX(currency) as currency").
		Where("tenant_id = ? AND customer_id = ? AND status = ? AND paid_at IS NOT NULL", tenantID, customerID, models.InvoiceStatusPaid).
		Group("DATE_TRUNC('day', paid_at)").
		Order("DATE_TRUNC('day', paid_at) ASC").
		Scan(&trendRows).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var totalSubscriptions, activeSubscriptions, totalCards, totalDirectDebits, totalSpent int64
	statusData := make([]responses.PortalBreakdownItem, 0, len(subscriptionRows))
	for _, row := range subscriptionRows {
		totalSubscriptions += row.Count
		if row.Status == string(models.SubscriptionStatusActive) {
			activeSubscriptions = row.Count
		}
		statusData = append(statusData, responses.PortalBreakdownItem{Label: row.Status, Count: row.Count})
	}

	sourceData := make([]responses.PortalBreakdownItem, 0, len(sourceRows))
	for _, row := range sourceRows {
		switch row.Type {
		case string(models.PaymentSourceTypeCard):
			totalCards = row.Count
		case string(models.PaymentSourceTypeBank):
			totalDirectDebits = row.Count
		}
		sourceData = append(sourceData, responses.PortalBreakdownItem{Label: row.Type, Count: row.Count})
	}

	trend := make([]responses.PortalAmountTrendPoint, 0, len(trendRows))
	currency := "NGN"
	for _, row := range trendRows {
		totalSpent += row.Amount
		if row.Currency != "" {
			currency = row.Currency
		}
		trend = append(trend, responses.PortalAmountTrendPoint{Date: row.Date, Amount: row.Amount})
	}

	return &responses.PortalAnalyticsResponse{
		Currency:               currency,
		TotalSubscriptions:     totalSubscriptions,
		ActiveSubscriptions:    activeSubscriptions,
		TotalCards:             totalCards,
		TotalDirectDebits:      totalDirectDebits,
		TotalSpent:             totalSpent,
		AmountSpentTrend:       trend,
		SubscriptionStatusData: statusData,
		PaymentSourceData:      sourceData,
	}, nil
}

func (s *PortalService) PaymentSources(tenantID, customerID string) ([]responses.CustomerPaymentSourceDetail, error) {
	sources, err := s.rc.PaymentSourceRepository.FindMany(&models.PaymentSource{
		TenantID:   tenantID,
		CustomerID: customerID,
	}, &repositories.FindArgs{OrderBy: []repositories.OrderBy{{Column: "created_at", Desc: true}}})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	result := make([]responses.CustomerPaymentSourceDetail, 0, len(sources))
	for _, source := range sources {
		result = append(result, billingPaymentSourceDetail(source))
	}
	return result, nil
}

func (s *PortalService) InitiateCardUpdate(tenantID, customerID, paymentSourceID string) (*responses.PortalCardUpdateResponse, error) {
	tenant, customer, err := s.portalTenantAndCustomer(tenantID, customerID)
	if err != nil {
		return nil, err
	}

	paymentSource, err := s.rc.PaymentSourceRepository.Find(&models.PaymentSource{
		TenantID:   tenantID,
		CustomerID: customerID,
		BaseModel:  models.BaseModel{ID: paymentSourceID},
		Type:       models.PaymentSourceTypeCard,
	}, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if paymentSource == nil {
		return nil, responses.NotFound("Card payment method not found")
	}
	if paymentSource.Status != models.PaymentSourceStatusActive {
		return nil, responses.BadRequest("payment method is not active")
	}

	reference, err := utils.GenerateRandomString(24)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	reference = fmt.Sprintf("nombasub_%s", reference)
	amount := int64(10000)
	currency := "NGN"
	metadata := map[string]interface{}{
		"nombaSubTenantId":        tenantID,
		"nombaSubCustomerId":      customerID,
		"nombaSubCustomerCode":    customer.Code,
		"nombaSubPaymentSourceId": paymentSource.ID,
		"nombaSubTenantAccountId": tenant.AccountID,
	}

	initiation, err := s.rc.NombaInitiationRepository.Create(&models.NombaInitiation{
		TenantID:  tenantID,
		Amount:    float64(amount),
		Currency:  currency,
		Reference: reference,
		Purpose:   models.NombaInitiationPurposeUpdateCard,
		Status:    models.NombaInitiationStatusPending,
		Metadata:  metadata,
	}, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	result, err := s.nombaProvider.CreateCheckoutOrder(nomba.CreateCheckoutOrderRequest{
		Order: nomba.NombaOrder{
			CallbackUrl:           "",
			CustomerEmail:         customer.Email,
			Amount:                &amount,
			Currency:              &currency,
			OrderReference:        &reference,
			AccountId:             &tenant.AccountID,
			AllowedPaymentMethods: &[]nomba.PaymentMethod{nomba.PaymentMethodCard},
			OrderMetaData:         &metadata,
		},
		TokenizeCard: utils.ToPtr(true),
	})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	initiation.NombaOrderID = &result.Data.OrderReference
	if _, err := s.rc.NombaInitiationRepository.Update(initiation, nil); err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &responses.PortalCardUpdateResponse{CheckoutLink: result.Data.CheckoutLink}, nil
}

func (s *PortalService) DisablePaymentSource(tenantID, customerID, paymentSourceID string) error {
	paymentSource, err := s.rc.PaymentSourceRepository.Find(&models.PaymentSource{
		TenantID:   tenantID,
		CustomerID: customerID,
		BaseModel:  models.BaseModel{ID: paymentSourceID},
	}, nil)
	if err != nil {
		return responses.InternalServerError(err)
	}
	if paymentSource == nil {
		return responses.NotFound("Payment method not found")
	}
	if paymentSource.Status == models.PaymentSourceStatusInactive {
		return responses.BadRequest("payment method is already disabled")
	}

	now := time.Now()
	var linkedSubscriptions []models.Subscription
	if err := s.rc.DB.Transaction(func(trx *gorm.DB) error {
		paymentSource.Status = models.PaymentSourceStatusInactive
		if _, err := s.rc.PaymentSourceRepository.Update(paymentSource, trx); err != nil {
			return responses.InternalServerError(err)
		}

		if err := trx.
			Where("tenant_id = ? AND customer_id = ? AND payment_source_id = ? AND status <> ?", tenantID, customerID, paymentSourceID, models.SubscriptionStatusCanceled).
			Find(&linkedSubscriptions).Error; err != nil {
			return responses.InternalServerError(err)
		}

		for index := range linkedSubscriptions {
			linkedSubscriptions[index].Status = models.SubscriptionStatusPaused
			linkedSubscriptions[index].PausedAt = &now
			if _, err := s.rc.SubscriptionRepository.Update(&linkedSubscriptions[index], trx); err != nil {
				return responses.InternalServerError(err)
			}
		}

		return queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantID, models.WebhookDeliveryEventTypePaymentMethodDisabled, map[string]interface{}{
			"paymentSource": paymentSource,
			"subscriptions": linkedSubscriptions,
		}, trx)
	}); err != nil {
		return err
	}

	return nil
}

func (s *PortalService) Subscriptions(tenantID, customerID string, query requests.GetSubscriptionQuery) (*responses.PaginatedResponse[responses.SubscriptionResponse], error) {
	query.Customer = &customerID
	return s.subscriptionService.GetSubscriptions(tenantID, query)
}

func (s *PortalService) Subscription(tenantID, customerID, idOrCode string) (*responses.SubscriptionResponse, error) {
	subscription, err := s.subscriptionService.GetSubscriptionModel(tenantID, idOrCode)
	if err != nil {
		return nil, err
	}
	if subscription.CustomerID != customerID {
		return nil, responses.NotFound("Subscription not found")
	}
	formatted, err := s.subscriptionService.formatSubscriptions([]models.Subscription{*subscription})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if len(formatted) == 0 {
		return nil, responses.NotFound("Subscription not found")
	}
	return &formatted[0], nil
}

func (s *PortalService) CancelSubscription(tenantID, customerID, idOrCode string) error {
	subscription, err := s.subscriptionService.GetSubscriptionModel(tenantID, idOrCode)
	if err != nil {
		return err
	}
	if subscription.CustomerID != customerID {
		return responses.NotFound("Subscription not found")
	}
	return s.subscriptionService.CancelSubscription(tenantID, idOrCode)
}

func (s *PortalService) UpdateSubscriptionPaymentMethod(tenantID, customerID, idOrCode string, body requests.UpdatePortalSubscriptionPaymentMethodRequest) (*responses.SubscriptionResponse, error) {
	subscription, err := s.subscriptionService.GetSubscriptionModel(tenantID, idOrCode)
	if err != nil {
		return nil, err
	}
	if subscription.CustomerID != customerID {
		return nil, responses.NotFound("Subscription not found")
	}
	if subscription.Status == models.SubscriptionStatusCanceled {
		return nil, responses.BadRequest("cannot update payment method for a canceled subscription")
	}

	paymentSource, err := s.rc.PaymentSourceRepository.Find(&models.PaymentSource{
		TenantID:   tenantID,
		CustomerID: customerID,
		BaseModel:  models.BaseModel{ID: body.PaymentSourceID},
	}, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if paymentSource == nil {
		return nil, responses.NotFound("Payment method not found")
	}
	if paymentSource.Status != models.PaymentSourceStatusActive {
		return nil, responses.BadRequest("payment method is not active")
	}

	previousPaymentSourceID := subscription.PaymentSourceID
	subscription.PaymentSourceID = &paymentSource.ID
	subscription.PaymentSourceType = &paymentSource.Type
	if _, err := s.rc.SubscriptionRepository.Update(subscription, nil); err != nil {
		return nil, responses.InternalServerError(err)
	}

	if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantID, models.WebhookDeliveryEventTypeSubscriptionPaymentUpdated, map[string]interface{}{
		"subscription":            subscription,
		"paymentSource":           paymentSource,
		"previousPaymentSourceId": previousPaymentSourceID,
	}, nil); err != nil {
		return nil, responses.InternalServerError(err)
	}

	return s.Subscription(tenantID, customerID, idOrCode)
}

func (s *PortalService) Invoices(tenantID, customerID string, query requests.GetInvoiceQuery) (*responses.PaginatedResponse[responses.InvoiceResponse], error) {
	query.CustomerID = &customerID
	return s.invoiceService.GetInvoices(tenantID, query)
}

func (s *PortalService) Invoice(tenantID, customerID, idOrCode string) (*responses.InvoiceResponse, error) {
	invoice, err := s.invoiceService.invoiceModel(tenantID, idOrCode)
	if err != nil {
		return nil, err
	}
	if invoice.CustomerID != customerID {
		return nil, responses.NotFound("Invoice not found")
	}
	formatted, err := s.invoiceService.formatInvoices([]models.Invoice{*invoice})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if len(formatted) == 0 {
		return nil, responses.NotFound("Invoice not found")
	}
	return &formatted[0], nil
}

func (s *PortalService) RetryInvoicePayment(tenantID, customerID, idOrCode string) (*responses.InvoiceResponse, error) {
	invoice, err := s.invoiceService.invoiceModel(tenantID, idOrCode)
	if err != nil {
		return nil, err
	}
	if invoice.CustomerID != customerID {
		return nil, responses.NotFound("Invoice not found")
	}
	return s.invoiceService.RetryInvoicePayment(tenantID, idOrCode)
}

func (s *PortalService) Refunds(tenantID, customerID string, query requests.RefundsQuery) (*responses.RefundsResponse, error) {
	page, limit := paginationValues(query.Page, query.Limit, 20, 100)
	db := s.rc.DB.Model(&models.Refund{}).
		Joins("LEFT JOIN invoices ON invoices.id = refunds.invoice_id").
		Where("refunds.tenant_id = ? AND invoices.customer_id = ?", tenantID, customerID)

	if query.Search != nil && strings.TrimSpace(*query.Search) != "" {
		search := "%" + strings.TrimSpace(*query.Search) + "%"
		db = db.Where("refunds.id ILIKE ? OR refunds.nomba_transaction_id ILIKE ? OR invoices.code ILIKE ?", search, search, search)
	}
	if query.From != nil && strings.TrimSpace(*query.From) != "" {
		from, err := parseRefundDate(*query.From, false)
		if err != nil {
			return nil, responses.BadRequest("from must use YYYY-MM-DD format")
		}
		db = db.Where("refunds.created_at >= ?", *from)
	}
	if query.To != nil && strings.TrimSpace(*query.To) != "" {
		to, err := parseRefundDate(*query.To, true)
		if err != nil {
			return nil, responses.BadRequest("to must use YYYY-MM-DD format")
		}
		db = db.Where("refunds.created_at <= ?", *to)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	var refunds []models.Refund
	if err := db.
		Preload("Invoice").
		Order("refunds.created_at DESC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&refunds).Error; err != nil {
		return nil, responses.InternalServerError(err)
	}

	items := make([]responses.RefundResponse, 0, len(refunds))
	for _, refund := range refunds {
		items = append(items, refundResponse(refund))
	}

	return &responses.RefundsResponse{
		Data: items,
		Meta: responses.PaginationMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: int(math.Ceil(float64(total) / float64(limit))),
		},
	}, nil
}

func (s *PortalService) portalTenantAndCustomer(tenantID, customerID string) (*models.Tenant, *models.Customer, error) {
	tenant, err := s.rc.TenantRepository.FindById(tenantID, nil)
	if err != nil {
		return nil, nil, responses.InternalServerError(err)
	}
	if tenant == nil {
		return nil, nil, responses.NotFound("Tenant not found")
	}

	customer, err := s.rc.CustomerRepository.FindRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"tenant_id = ? AND (id = ? OR code = ?)",
			tenant.ID,
			customerID,
			customerID,
		),
	})
	if err != nil {
		return nil, nil, responses.InternalServerError(err)
	}
	if customer == nil {
		return nil, nil, responses.NotFound("Customer not found")
	}

	return tenant, customer, nil
}
