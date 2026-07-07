package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/google/uuid"
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
		enqueueInvoiceEmail(s.rc, s.publisher, models.EmailTemplateUpcomingInvoice, invoice, string(models.EmailTemplateUpcomingInvoice)+":"+invoice.ID)
	}
}

func (s *InvoiceService) ProcessDueInvoices() {
	subscriptions, err := s.rc.SubscriptionRepository.FindManyRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(
			"subscriptions.status IN (?, ?) AND subscriptions.current_billing_cycle_start <= ? AND (subscriptions.trial_period_days = 0 OR subscriptions.started_at IS NOT NULL)",
			models.SubscriptionStatusActive,
			models.SubscriptionStatusPastDue,
			time.Now(),
		),
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
		enqueueInvoiceEmail(s.rc, s.publisher, models.EmailTemplateInvoiceCreated, invoice, string(models.EmailTemplateInvoiceCreated)+":"+invoice.ID)
	}

	if invoice.CheckoutLink != nil {
		return nil
	}
	if invoice.AttemptCount > 0 && (invoice.NextPaymentAttemptAt == nil || invoice.NextPaymentAttemptAt.After(time.Now())) {
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
		return s.createCheckout(invoice, subscription, true)
	}

	if paymentSource.Type == models.PaymentSourceTypeCard {
		return s.chargeCard(invoice, subscription, paymentSource)
	}

	return s.chargeDirectDebit(invoice, subscription, paymentSource)
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

func (s *InvoiceService) createCheckout(invoice *models.Invoice, subscription *models.Subscription, sendEmail bool) error {
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
			Amount:                &invoice.AmountDue,
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
	if sendEmail {
		enqueueCheckoutEmail(s.rc, s.publisher, invoice, response.Data.CheckoutLink)
	}

	return nil
}

func (s *InvoiceService) CreateCheckoutForSubscription(invoice *models.Invoice, subscription *models.Subscription, sendEmail bool) (string, error) {
	if invoice.CheckoutLink != nil && *invoice.CheckoutLink != "" {
		if sendEmail {
			enqueueCheckoutEmail(s.rc, s.publisher, invoice, *invoice.CheckoutLink)
		}
		return *invoice.CheckoutLink, nil
	}

	if err := s.createCheckout(invoice, subscription, sendEmail); err != nil {
		return "", err
	}

	if invoice.CheckoutLink == nil || *invoice.CheckoutLink == "" {
		return "", fmt.Errorf("checkout provider did not return a checkout link")
	}

	return *invoice.CheckoutLink, nil
}

func (s *InvoiceService) GetInvoices(tenantID string, query requests.GetInvoiceQuery) (*responses.PaginatedResponse[responses.InvoiceResponse], error) {
	conditions := []string{"invoices.tenant_id = ?"}
	args := []interface{}{tenantID}
	if query.Search != nil && *query.Search != "" {
		search := "%" + *query.Search + "%"
		conditions = append(conditions, "(invoices.code ILIKE ? OR plans.name ILIKE ? OR subscriptions.code ILIKE ?)")
		args = append(args, search, search, search)
	}
	if query.Status != nil && *query.Status != "" {
		conditions = append(conditions, "invoices.status = ?")
		args = append(args, *query.Status)
	}
	if query.CustomerID != nil && *query.CustomerID != "" {
		conditions = append(conditions, "invoices.customer_id = ?")
		args = append(args, *query.CustomerID)
	}

	response, err := s.rc.InvoiceRepository.FindManyPaginatedRaw(&repositories.FindArgs{
		Filter: repositories.NewQueryFilter().Where(strings.Join(conditions, " AND "), args...),
		Joins: []repositories.Join{
			*repositories.NewJoin("LEFT JOIN subscriptions ON subscriptions.id = invoices.subscription_id"),
			*repositories.NewJoin("LEFT JOIN plans ON plans.id = subscriptions.plan_id"),
		},
		Preloads: []repositories.Preload{
			{Association: "Customer"},
			{Association: "Subscription"},
			{Association: "Subscription.Plan"},
		},
		OrderBy: []repositories.OrderBy{{Column: "invoices.created_at", Desc: true}},
	}, &query.PaginationQuery)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	data, err := s.formatInvoices(response.Data)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &responses.PaginatedResponse[responses.InvoiceResponse]{
		Data:            data,
		TotalCount:      response.TotalCount,
		Page:            response.Page,
		Limit:           response.Limit,
		TotalPages:      response.TotalPages,
		HasNextPage:     response.HasNextPage,
		HasPreviousPage: response.HasPreviousPage,
	}, nil
}

func (s *InvoiceService) GetInvoice(tenantID, idOrCode string) (*responses.InvoiceResponse, error) {
	invoice, err := s.invoiceModel(tenantID, idOrCode)
	if err != nil {
		return nil, err
	}

	data, err := s.formatInvoices([]models.Invoice{*invoice})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if len(data) == 0 {
		return nil, responses.NotFound("Invoice not found")
	}

	return &data[0], nil
}

func (s *InvoiceService) RetryInvoicePayment(tenantID, idOrCode string) (*responses.InvoiceResponse, error) {
	invoice, err := s.invoiceModel(tenantID, idOrCode)
	if err != nil {
		return nil, err
	}
	if invoice.Status == models.InvoiceStatusPaid {
		return nil, responses.BadRequest("invoice is already paid")
	}
	if invoice.Status == models.InvoiceStatusRefunded {
		return nil, responses.BadRequest("invoice has been refunded")
	}

	subscription, err := s.rc.SubscriptionRepository.FindById(invoice.SubscriptionID, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if subscription == nil {
		return nil, responses.NotFound("Subscription not found")
	}

	if invoice.Status == models.InvoiceStatusFailed {
		invoice.Status = models.InvoiceStatusOpen
		invoice.FailedAt = nil
		invoice.FailureReason = nil
	}
	invoice.NextPaymentAttemptAt = utils.ToPtr(time.Now())
	if _, err := s.rc.InvoiceRepository.Update(invoice, nil); err != nil {
		return nil, responses.InternalServerError(err)
	}

	if subscription.Status == models.SubscriptionStatusPaused {
		subscription.Status = models.SubscriptionStatusPastDue
		subscription.PausedAt = nil
		if _, err := s.rc.SubscriptionRepository.Update(subscription, nil); err != nil {
			return nil, responses.InternalServerError(err)
		}
	}

	if err := s.processDueSubscription(subscription); err != nil {
		return nil, responses.InternalServerError(err)
	}

	return s.GetInvoice(tenantID, idOrCode)
}

func (s *InvoiceService) SendPaymentReminder(tenantID, idOrCode string) (*responses.GenerateInvoiceCheckoutLinkResponse, error) {
	invoice, err := s.invoiceModel(tenantID, idOrCode)
	if err != nil {
		return nil, err
	}
	if invoice.Status == models.InvoiceStatusPaid {
		return nil, responses.BadRequest("invoice is already paid")
	}
	if invoice.Status == models.InvoiceStatusRefunded {
		return nil, responses.BadRequest("invoice has been refunded")
	}

	subscription, err := s.rc.SubscriptionRepository.FindById(invoice.SubscriptionID, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if subscription == nil {
		return nil, responses.NotFound("Subscription not found")
	}

	if invoice.Status == models.InvoiceStatusDraft {
		invoice.Status = models.InvoiceStatusOpen
		if _, err := s.rc.InvoiceRepository.Update(invoice, nil); err != nil {
			return nil, responses.InternalServerError(err)
		}
	}

	link, err := s.CreateCheckoutForSubscription(invoice, subscription, true)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &responses.GenerateInvoiceCheckoutLinkResponse{CheckoutLink: link, Sent: true}, nil
}

func (s *InvoiceService) GenerateCheckoutLink(tenantID, idOrCode string, sendEmail bool) (*responses.GenerateInvoiceCheckoutLinkResponse, error) {
	invoice, err := s.invoiceModel(tenantID, idOrCode)
	if err != nil {
		return nil, err
	}
	if invoice.Status != models.InvoiceStatusOpen && invoice.Status != models.InvoiceStatusDraft && invoice.Status != models.InvoiceStatusFailed {
		return nil, responses.BadRequest("checkout link can only be generated for draft, open, or failed invoices")
	}

	subscription, err := s.rc.SubscriptionRepository.FindById(invoice.SubscriptionID, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if subscription == nil {
		return nil, responses.NotFound("Subscription not found")
	}

	if invoice.Status == models.InvoiceStatusDraft {
		invoice.Status = models.InvoiceStatusOpen
		if _, err := s.rc.InvoiceRepository.Update(invoice, nil); err != nil {
			return nil, responses.InternalServerError(err)
		}
	}

	link, err := s.CreateCheckoutForSubscription(invoice, subscription, sendEmail)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &responses.GenerateInvoiceCheckoutLinkResponse{CheckoutLink: link, Sent: sendEmail}, nil
}

func (s *InvoiceService) invoiceModel(tenantID, idOrCode string) (*models.Invoice, error) {
	var filter *repositories.QueryFilter

	if uuid.Validate(idOrCode) == nil {
		filter = repositories.NewQueryFilter().Where("tenant_id = ? AND id = ?", tenantID, idOrCode)
	} else {
		filter = repositories.NewQueryFilter().Where("tenant_id = ? AND code = ?", tenantID, idOrCode)
	}

	invoice, err := s.rc.InvoiceRepository.FindRaw(&repositories.FindArgs{
		Filter: filter,
		Preloads: []repositories.Preload{
			{Association: "Customer"},
			{Association: "Subscription"},
			{Association: "Subscription.Plan"},
		},
	})
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if invoice == nil {
		return nil, responses.NotFound("Invoice not found")
	}
	return invoice, nil
}

func (s *InvoiceService) formatInvoices(invoices []models.Invoice) ([]responses.InvoiceResponse, error) {
	result := make([]responses.InvoiceResponse, 0, len(invoices))
	for _, invoice := range invoices {
		paymentIntent, err := s.invoicePaymentIntent(invoice.TenantID, invoice.ID)
		if err != nil {
			return nil, err
		}
		refund, err := s.invoiceRefund(invoice.TenantID, invoice.ID)
		if err != nil {
			return nil, err
		}

		var plan responses.InvoicePlanResponse
		var customer responses.CustomerProfileResponse
		var subscription responses.InvoiceSubscriptionResponse
		if invoice.Customer != nil {
			customer = responses.CustomerProfileResponse{
				ID:          invoice.Customer.ID,
				Code:        invoice.Customer.Code,
				Name:        invoice.Customer.Name,
				Email:       invoice.Customer.Email,
				PhoneNumber: invoice.Customer.PhoneNumber,
				ExternalRef: invoice.Customer.ExternalRef,
				CreatedAt:   invoice.Customer.CreatedAt,
				UpdatedAt:   invoice.Customer.UpdatedAt,
			}
		}
		if invoice.Subscription != nil {
			subscription = responses.InvoiceSubscriptionResponse{ID: invoice.Subscription.ID, Code: invoice.Subscription.Code}
			if invoice.Subscription.Plan != nil {
				plan = responses.InvoicePlanResponse{
					ID:   invoice.Subscription.Plan.ID,
					Name: invoice.Subscription.Plan.Name,
					Code: invoice.Subscription.Plan.Code,
				}
			}
		}

		var paymentIntentResponse *responses.InvoicePaymentIntentResponse
		if paymentIntent != nil {
			var paymentSource *responses.CustomerPaymentSourceDetail
			if paymentIntent.PaymentSource != nil {
				source := billingPaymentSourceDetail(*paymentIntent.PaymentSource)
				paymentSource = &source
			}

			paymentIntentResponse = &responses.InvoicePaymentIntentResponse{
				ID:                     paymentIntent.ID,
				Code:                   paymentIntent.Code,
				Status:                 paymentIntent.Status,
				ProviderTransactionID:  paymentIntent.ProviderTransactionID,
				ProviderTransactionRef: paymentIntent.ProviderTransactionReference,
				PaymentSource:          paymentSource,
			}
		}

		var refundData *responses.RefundResponse
		if refund != nil {
			formattedRefund := refundResponse(*refund)
			refundData = &formattedRefund
		}

		result = append(result, responses.InvoiceResponse{
			ID:                 invoice.ID,
			Code:               invoice.Code,
			Status:             invoice.Status,
			AmountDue:          invoice.AmountDue,
			AmountPaid:         invoice.AmountPaid,
			AmountRemaining:    invoice.AmountRemaining,
			Currency:           invoice.Currency,
			BillingPeriodStart: invoice.BillingPeriodStart,
			BillingPeriodEnd:   invoice.BillingPeriodEnd,
			DueAt:              invoice.DueAt,
			PaidAt:             invoice.PaidAt,
			FailedAt:           invoice.FailedAt,
			RefundedAt:         invoice.RefundedAt,
			CheckoutLink:       invoice.CheckoutLink,
			FailureReason:      invoice.FailureReason,
			CanBeRefunded:      CanRefundInvoice(&invoice),
			Plan:               plan,
			Customer:           customer,
			Subscription:       subscription,
			PaymentIntent:      paymentIntentResponse,
			Refund:             refundData,
			CreatedAt:          invoice.CreatedAt,
			UpdatedAt:          invoice.UpdatedAt,
		})
	}

	return result, nil
}

func (s *InvoiceService) invoicePaymentIntent(tenantID, invoiceID string) (*models.PaymentIntent, error) {
	paymentIntent, err := s.rc.PaymentIntentRepository.Find(&models.PaymentIntent{
		TenantID:  tenantID,
		InvoiceID: &invoiceID,
	}, &repositories.FindArgs{
		Preloads: []repositories.Preload{{Association: "PaymentSource"}},
		OrderBy:  []repositories.OrderBy{{Column: "created_at", Desc: true}},
	})
	if err != nil {
		return nil, err
	}
	return paymentIntent, nil
}

func (s *InvoiceService) invoiceRefund(tenantID, invoiceID string) (*models.Refund, error) {
	refund, err := s.rc.RefundRepository.Find(&models.Refund{
		TenantID:  tenantID,
		InvoiceID: &invoiceID,
	}, &repositories.FindArgs{
		Preloads: []repositories.Preload{{Association: "Invoice"}},
		OrderBy:  []repositories.OrderBy{{Column: "created_at", Desc: true}},
	})
	if err != nil {
		return nil, err
	}
	return refund, nil
}

func CanRefundInvoice(invoice *models.Invoice) bool {
	if invoice == nil || invoice.Status != models.InvoiceStatusPaid || invoice.PaidAt == nil {
		return false
	}
	return time.Since(*invoice.PaidAt) <= 12*time.Hour
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
		"nombaSubPaymentSourceId": paymentSource.ID,
	}

	initiation, err := s.rc.NombaInitiationRepository.Create(&models.NombaInitiation{
		TenantID:  subscription.TenantID,
		Amount:    float64(invoice.AmountDue),
		Currency:  invoice.Currency,
		Reference: reference,
		Purpose:   models.NombaInitiationPurposeChargeCardPayment,
		Status:    models.NombaInitiationStatusPending,
		Metadata:  metadata,
	}, nil)
	if err != nil {
		initiation.Status = models.NombaInitiationStatusFailed
		s.rc.NombaInitiationRepository.Update(initiation, nil)
		return err
	}

	response, err := s.nombaProvider.ChargeCard(nomba.ChargeCardRequest{
		Order: nomba.NombaOrder{
			CallbackUrl:    "",
			CustomerEmail:  customer.Email,
			Amount:         &invoice.AmountDue,
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
	isTransient := eventType == models.WebhookDeliveryEventTypeInvoicePaymentFailed
	if isTransient && subscription.AllowRetries && invoice.AttemptCount < 3 {
		nextAttempt := time.Now().Add(24 * time.Hour)
		invoice.NextPaymentAttemptAt = &nextAttempt
		invoice.FailureReason = &reason
		if _, err := s.rc.InvoiceRepository.Update(invoice, nil); err != nil {
			return err
		}

		subscription.Status = models.SubscriptionStatusPastDue
		subscription.LatestInvoiceID = &invoice.ID
		if _, err := s.rc.SubscriptionRepository.Update(subscription, nil); err != nil {
			return err
		}

		s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeInvoicePaymentFailed, invoice)
		s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeSubscriptionPastDue, subscription)
		return nil
	}

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
	enqueueSubscriptionPausedEmail(s.rc, s.publisher, subscription, invoice, reason)
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
	enqueueInvoiceEmail(s.rc, s.publisher, models.EmailTemplatePaymentSuccessful, invoice, string(models.EmailTemplatePaymentSuccessful)+":"+invoice.ID)
	enqueueInvoiceEmail(s.rc, s.publisher, models.EmailTemplatePaymentReceipt, invoice, string(models.EmailTemplatePaymentReceipt)+":"+invoice.ID)
	enqueueInvoiceEmail(s.rc, s.publisher, models.EmailTemplateInvoicePaid, invoice, string(models.EmailTemplateInvoicePaid)+":"+invoice.ID)
	return nil
}

func (s *InvoiceService) chargeDirectDebit(invoice *models.Invoice, subscription *models.Subscription, paymentSource *models.PaymentSource) error {
	if paymentSource.Bank == nil || paymentSource.Bank.MandateID == nil {
		return s.failInvoice(invoice, subscription, "bank payment source is missing a mandate ID", models.WebhookDeliveryEventTypeInvoiceMarkedUncollectible)
	}

	mandateId := *paymentSource.Bank.MandateID

	// Verify mandate is still active before attempting debit
	statusResp, err := s.nombaProvider.GetDirectDebitManadateStatus(mandateId)
	if err != nil {
		return s.failInvoice(invoice, subscription, fmt.Sprintf("could not verify mandate status: %s", err.Error()), models.WebhookDeliveryEventTypeInvoicePaymentFailed)
	}
	if statusResp.Data.MandateStatus != nomba.MandateStatusActive {
		return s.failInvoice(invoice, subscription, "mandate is not active", models.WebhookDeliveryEventTypeInvoiceMarkedUncollectible)
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

	initiation, err := s.rc.NombaInitiationRepository.Create(&models.NombaInitiation{
		TenantID:  subscription.TenantID,
		Amount:    float64(invoice.AmountDue),
		Currency:  invoice.Currency,
		Reference: reference,
		Purpose:   models.NombaInitiationPurposeDirectDebitCharge,
		Status:    models.NombaInitiationStatusPending,
		Metadata: map[string]interface{}{
			"nombaSubSubscriptionId": subscription.ID,
			"nombaSubInvoiceId":      invoice.ID,
		},
		PaymentIntentId: &paymentIntent.ID,
	}, nil)
	if err != nil {
		return err
	}

	invoice.AttemptCount++
	if _, err := s.rc.InvoiceRepository.Update(invoice, nil); err != nil {
		return err
	}
	s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeInvoicePaymentAttempted, invoice)

	debitResp, err := s.nombaProvider.DebitMandate(nomba.DebitMandateRequest{
		MandateId: mandateId,
		Amount:    float64(invoice.AmountDue) / 100,
	})

	if err != nil {
		paymentIntent.Status = models.PaymentIntentStatusFailed
		paymentIntent.FailureReason = utils.ToPtr(err.Error())
		paymentIntent.FailedAt = utils.ToPtr(time.Now())
		_, _ = s.rc.PaymentIntentRepository.Update(paymentIntent, nil)
		initiation.Status = models.NombaInitiationStatusFailed
		_, _ = s.rc.NombaInitiationRepository.Update(initiation, nil)
		s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeMandateDebitFailed, map[string]interface{}{
			"mandateId":     mandateId,
			"invoice":       invoice,
			"subscription":  subscription,
			"failureReason": err.Error(),
		})
		return s.failInvoice(invoice, subscription, err.Error(), models.WebhookDeliveryEventTypeInvoicePaymentFailed)
	}

	paymentIntent.Status = models.PaymentIntentStatusSuccess
	paymentIntent.ProviderTransactionReference = &debitResp.Data.MandateId
	paymentIntent.CompletedAt = utils.ToPtr(time.Now())
	if _, err := s.rc.PaymentIntentRepository.Update(paymentIntent, nil); err != nil {
		return err
	}

	initiation.Status = models.NombaInitiationStatusCompleted
	if _, err := s.rc.NombaInitiationRepository.Update(initiation, nil); err != nil {
		return err
	}

	amountAfterFee := s.nombaProvider.DeductFee(float64(invoice.AmountDue) / 100)
	_, _ = s.rc.SettlementRepository.Create(&models.Settlement{
		TenantID:       subscription.TenantID,
		Purpose:        models.NombaInitiationPurposeDirectDebitCharge,
		Amount:         amountAfterFee,
		Currency:       invoice.Currency,
		Status:         models.SettlementStatusPending,
		Reference:      reference,
		SettlementTime: time.Now().Add(25 * time.Hour),
		SubscriptionID: &subscription.ID,
		InvoiceID:      &invoice.ID,
	}, nil)

	s.enqueueWebhook(subscription.TenantID, models.WebhookDeliveryEventTypeMandateDebitSuccess, map[string]interface{}{
		"mandateId":    mandateId,
		"invoice":      invoice,
		"subscription": subscription,
		"amount":       float64(invoice.AmountDue) / 100,
	})

	return s.markInvoicePaid(invoice, subscription)
}

func (s *InvoiceService) enqueueWebhook(tenantID string, eventType models.WebhookDeliveryEventType, data any) {
	if err := queue.EnqueueTenantWebhook(s.rc, s.publisher, tenantID, eventType, data, nil); err != nil {
		log.Printf("invoice webhook enqueue failed for tenant %s event %s: %v", tenantID, eventType, err)
	}
}
