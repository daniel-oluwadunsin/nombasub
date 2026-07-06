package services

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/queue"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"gorm.io/gorm"
)

type TransactionService struct {
	rc              *repositories.Container
	nombaProvider   nomba.Provider
	customerService *CustomerService
	publisher       *queue.Publisher
}

func NewTransactionService(rc *repositories.Container, nombaProvider nomba.Provider, customerService *CustomerService, publisher *queue.Publisher) *TransactionService {
	return &TransactionService{
		rc:              rc,
		nombaProvider:   nombaProvider,
		customerService: customerService,
		publisher:       publisher,
	}
}

func (ts *TransactionService) InitializeCardTransaction(tenantId, tenantAccountId string, body requests.CreateCheckoutOrderRequest) (*nomba.CreateCheckoutOrderResponse, error) {
	db := ts.rc.DB
	planVersionRepository := ts.rc.PlanVersionRepository
	nombaInitiationRepository := ts.rc.NombaInitiationRepository
	subscriptionRepository := ts.rc.SubscriptionRepository
	nombaProvider := ts.nombaProvider
	customerService := ts.customerService

	plan, err := planVersionRepository.Find(
		&models.PlanVersion{TenantID: tenantId, Code: body.PlanCode}, &repositories.FindArgs{
			OrderBy: []repositories.OrderBy{
				{Column: "index", Desc: true},
			},
		},
	)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if plan == nil {
		return nil, responses.NotFound("Plan not found")
	}
	if plan.Status != models.PlanStatusActive {
		return nil, responses.BadRequest("Plan is not active")
	}

	var nombaResponse *nomba.CreateCheckoutOrderResponse

	err = db.Transaction(func(trx *gorm.DB) error {
		customer, err := customerService.GetOrCreateCustomer(
			tenantId,
			models.Customer{TenantID: tenantId, Email: body.Order.CustomerEmail},
			trx,
		)
		if err != nil {
			return responses.InternalServerError(err)
		}

		subscriptionExists, err := subscriptionRepository.Exists(&models.Subscription{
			TenantID:   tenantId,
			CustomerID: customer.ID,
			PlanID:     plan.PlanID,
			Status:     models.SubscriptionStatusActive,
		}, nil)
		if err != nil {
			return responses.InternalServerError(err)
		}
		if subscriptionExists {
			return responses.BadRequest("Customer already has an active subscription for this plan")
		}

		checkoutOrder := body.CreateCheckoutOrderRequest

		tenantAccountId = *utils.Or(checkoutOrder.Order.AccountId, &tenantAccountId)
		metadata := *utils.Or(checkoutOrder.Order.OrderMetaData, new(map[string]interface{}))
		metadata["nombaSubTenantId"] = tenantId
		metadata["nombaSubCustomerCode"] = customer.Code
		metadata["nombaSubPlanCode"] = plan.Code
		metadata["nombaSubPlanVersion"] = plan.Index
		metadata["nombaSubTenantAccountId"] = tenantAccountId
		if checkoutOrder.Order.OrderReference != nil {
			metadata["nombaSubTenantOrderReference"] = *checkoutOrder.Order.OrderReference
		}
		checkoutOrder.Order.OrderMetaData = &metadata
		checkoutOrder.Order.AllowedPaymentMethods = utils.ToPtr([]nomba.PaymentMethod{nomba.PaymentMethodCard})
		checkoutOrder.Order.Amount = &plan.Amount
		checkoutOrder.Order.Currency = &plan.Currency
		checkoutOrder.TokenizeCard = utils.ToPtr(true)
		checkoutOrder.Order.AccountId = utils.ToPtr(tenantAccountId)

		reference, err := utils.GenerateRandomString(24)
		if err != nil {
			return responses.InternalServerError(err)
		}
		reference = fmt.Sprintf("nombasub_%s", reference)
		checkoutOrder.Order.OrderReference = &reference

		nombaInitiation, err := nombaInitiationRepository.Create(&models.NombaInitiation{
			TenantID:  tenantId,
			Amount:    float64(*checkoutOrder.Order.Amount),
			Currency:  *checkoutOrder.Order.Currency,
			Reference: reference,
			Purpose:   models.NombaInitiationPurposeCardSubscriptionPayment,
			Metadata:  metadata,
		}, trx)

		if err != nil {
			return responses.InternalServerError(err)
		}

		nombaResponse, err = nombaProvider.CreateCheckoutOrder(checkoutOrder)
		if err != nil {
			return responses.InternalServerError(err)
		}

		nombaInitiation.NombaOrderID = &nombaResponse.Data.OrderReference
		_, err = nombaInitiationRepository.Update(nombaInitiation, trx)
		if err != nil {
			return responses.InternalServerError(err)
		}

		return nil
	})

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return nombaResponse, nil
}

func (ts *TransactionService) InitializeDirectDebitSubscription(tenantId string, body requests.InitializeDirectDebitRequest) (*responses.InitializeDirectDebitResponse, error) {
	db := ts.rc.DB
	planVersionRepository := ts.rc.PlanVersionRepository
	nombaInitiationRepository := ts.rc.NombaInitiationRepository
	subscriptionRepository := ts.rc.SubscriptionRepository
	nombaProvider := ts.nombaProvider
	customerService := ts.customerService

	plan, err := planVersionRepository.Find(
		&models.PlanVersion{TenantID: tenantId, Code: body.PlanCode},
		&repositories.FindArgs{
			OrderBy: []repositories.OrderBy{{Column: "index", Desc: true}},
		},
	)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if plan == nil {
		return nil, responses.NotFound("Plan not found")
	}
	if plan.Status != models.PlanStatusActive {
		return nil, responses.BadRequest("Plan is not active")
	}

	var result *responses.InitializeDirectDebitResponse

	err = db.Transaction(func(trx *gorm.DB) error {
		customer, err := customerService.GetOrCreateCustomer(
			tenantId,
			models.Customer{TenantID: tenantId, Email: body.CustomerEmail},
			trx,
		)
		if err != nil {
			return responses.InternalServerError(err)
		}

		subscriptionExists, err := subscriptionRepository.Exists(&models.Subscription{
			TenantID:   tenantId,
			CustomerID: customer.ID,
			PlanID:     plan.PlanID,
			Status:     models.SubscriptionStatusActive,
		}, nil)
		if err != nil {
			return responses.InternalServerError(err)
		}
		if subscriptionExists {
			return responses.BadRequest("Customer already has an active subscription for this plan")
		}

		merchantReference, err := utils.GenerateNumericString(12)
		if err != nil {
			return responses.InternalServerError(err)
		}

		metadata := map[string]interface{}{
			"nombaSubTenantId":     tenantId,
			"nombaSubCustomerCode": customer.Code,
			"nombaSubPlanCode":     plan.Code,
			"nombaSubPlanVersion":  plan.Index,
		}
		if body.OrderReference != nil {
			metadata["nombaSubTenantOrderReference"] = *body.OrderReference
		}

		nombaResponse, err := nombaProvider.CreateDirectDebitManadate(nomba.CreateDirectDebitManadateRequest{
			CustomerAccountNumber: body.CustomerAccountNumber,
			CustomerAccountName:   body.CustomerAccountName,
			CustomerName:          body.CustomerName,
			CustomerAddress:       body.CustomerAddress,
			BankCode:              body.BankCode,
			Frequency:             body.Frequency,
			Narration:             body.Narration,
			CustomerPhoneNumber:   body.CustomerPhoneNumber,
			MerchantReference:     merchantReference,
			StartDate:             body.StartDate,
			EndDate:               body.EndDate,
			StartImmediately:      body.StartImmediately,
		})
		if err != nil {
			return responses.InternalServerError(err)
		}
		fmt.Println(nombaResponse)

		mandateId := nombaResponse.Data.MandateID

		_, err = nombaInitiationRepository.Create(&models.NombaInitiation{
			TenantID:  tenantId,
			Amount:    float64(plan.Amount),
			Currency:  plan.Currency,
			Reference: mandateId,
			Purpose:   models.NombaInitiationPurposeDirectDebitSubscription,
			Status:    models.NombaInitiationStatusPending,
			Metadata:  metadata,
		}, trx)
		if err != nil {
			return responses.InternalServerError(err)
		}

		result = &responses.InitializeDirectDebitResponse{
			MandateID:           mandateId,
			MerchantReference:   nombaResponse.Data.MerchantReference,
			CustomerPhoneNumber: nombaResponse.Data.CustomerPhoneNumber,
			Description:         nombaResponse.Data.Description,
		}

		if err := queue.EnqueueTenantWebhook(ts.rc, ts.publisher, tenantId, models.WebhookDeliveryEventTypeMandateCreated, map[string]interface{}{
			"mandateId":           mandateId,
			"merchantReference":   nombaResponse.Data.MerchantReference,
			"customerPhoneNumber": nombaResponse.Data.CustomerPhoneNumber,
			"planCode":            plan.Code,
			"customerCode":        customer.Code,
		}, trx); err != nil {
			log.Printf("direct debit: failed to enqueue mandate.created webhook: %v", err)
		}

		return nil
	})

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return result, nil
}

func (ts *TransactionService) GetRefunds(tenantID string, query requests.RefundsQuery) (*responses.RefundsResponse, error) {
	page, limit := paginationValues(query.Page, query.Limit, 20, 100)

	db := ts.rc.DB.Model(&models.Refund{}).
		Joins("LEFT JOIN invoices ON invoices.id = refunds.invoice_id").
		Where("refunds.tenant_id = ?", tenantID)

	if query.Search != nil && strings.TrimSpace(*query.Search) != "" {
		search := "%" + strings.TrimSpace(*query.Search) + "%"
		db = db.Where(
			"refunds.id ILIKE ? OR refunds.nomba_transaction_id ILIKE ? OR invoices.code ILIKE ?",
			search,
			search,
			search,
		)
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

func (ts *TransactionService) RefundPaymentOrInvoice(tenantId string, body requests.RefundPaymentOrInvoiceRequest) error {
	invoiceRepository := ts.rc.InvoiceRepository
	paymentIntentRepository := ts.rc.PaymentIntentRepository
	paymentSourceRepository := ts.rc.PaymentSourceRepository
	initiationRepository := ts.rc.NombaInitiationRepository
	refundRepository := ts.rc.RefundRepository
	settlementRepository := ts.rc.SettlementRepository
	nombaProvider := ts.nombaProvider

	var paymentIntent *models.PaymentIntent
	var invoice *models.Invoice
	var settlement *models.Settlement
	var err error
	if (body.PaymentIntentId == nil || *body.PaymentIntentId == "") && (body.InvoiceId == nil || *body.InvoiceId == "") {
		return responses.BadRequest("paymentIntentId or invoiceId is required")
	}

	if body.PaymentIntentId != nil && *body.PaymentIntentId != "" {
		paymentIntent, err = paymentIntentRepository.Find(&models.PaymentIntent{
			TenantID: tenantId,
			BaseModel: models.BaseModel{
				ID: *body.PaymentIntentId,
			},
		}, nil)
		if err != nil {
			return responses.InternalServerError(err)
		}
		if paymentIntent == nil {
			return responses.NotFound("Payment intent not found")
		}

		if paymentIntent.InvoiceID != nil {
			body.InvoiceId = paymentIntent.InvoiceID
		}
	}

	if body.InvoiceId != nil && *body.InvoiceId != "" {
		invoice, err = invoiceRepository.Find(&models.Invoice{
			TenantID: tenantId,
			BaseModel: models.BaseModel{
				ID: *body.InvoiceId,
			},
		}, nil)
		if err != nil {
			return responses.InternalServerError(err)
		}
		if invoice == nil {
			return responses.NotFound("Invoice not found")
		}

		if paymentIntent == nil {
			paymentIntent, err = paymentIntentRepository.Find(&models.PaymentIntent{
				TenantID:  tenantId,
				InvoiceID: &invoice.ID,
				Status:    models.PaymentIntentStatusSuccess,
			}, nil)
			if err != nil {
				return responses.InternalServerError(err)
			}
			if paymentIntent == nil {
				return responses.NotFound("Payment intent not found")
			}
		}
	}

	if invoice == nil {
		return responses.NotFound("Invoice not found")
	}
	if !CanRefundInvoice(invoice) {
		return responses.BadRequest("invoice can only be refunded within 12 hours of payment")
	}

	if paymentIntent.Status != models.PaymentIntentStatusSuccess {
		return responses.BadRequest("Payment intent is not successful")
	}

	if paymentIntent.Status == models.PaymentIntentStatusRefund {
		return responses.BadRequest("Payment intent is already refunded")
	}

	if paymentIntent.Status == models.PaymentIntentStatusCancelled {
		return responses.BadRequest("Payment intent is cancelled")
	}

	if paymentIntent.Status == models.PaymentIntentStatusPendingBilling {
		return responses.BadRequest("Payment intent is pending billing")
	}

	settlement, err = settlementRepository.Find(&models.Settlement{
		InvoiceID: &invoice.ID,
	}, nil)

	if err != nil {
		return responses.InternalServerError(err)
	}

	var transactionId string

	if paymentIntent.ProviderTransactionID != nil {
		transactionId = *paymentIntent.ProviderTransactionID
	} else {

		initiation, err := initiationRepository.Find(
			&models.NombaInitiation{
				PaymentIntentId: &paymentIntent.ID,
			}, nil)
		if err != nil {
			return responses.InternalServerError(err)
		}
		if initiation == nil {
			return responses.NotFound("Nomba initiation not found")
		}
		transactionId = *initiation.NombaTransactionId
	}

	if transactionId == "" {
		return responses.BadRequest("Nomba transaction couldn't be traced")
	}

	var refundRequest *nomba.RefundRequest
	refund := &models.Refund{
		TenantID:    tenantId,
		Reason:      body.Reason,
		InitiatedAt: time.Now(),
		ETAFrom:     time.Now(),
		ETATo:       time.Now().Add(7 * 24 * time.Hour),
		Amount:      float64(paymentIntent.Amount) / 100,
		Currency:    paymentIntent.Currency,
	}

	paymentSource, err := paymentSourceRepository.FindById(paymentIntent.PaymentSourceID, nil)
	if err != nil {
		return responses.InternalServerError(err)
	}
	if paymentSource == nil {
		return responses.NotFound("Payment source not found")
	}

	if paymentSource.Type == models.PaymentSourceTypeCard {
		refundRequest = &nomba.RefundRequest{
			TransactionId: &transactionId,
		}
		refund.Card = paymentSource.Card
	} else {
		refundRequest = &nomba.RefundRequest{
			TransactionId: &transactionId,
			AccountNumber: paymentSource.Bank.AccountNumber,
			BankCode:      paymentSource.Bank.Code,
		}

		refund.Bank = paymentSource.Bank
	}

	err = nombaProvider.RequestRefund(*refundRequest)
	if err != nil {
		return responses.InternalServerError(err)
	}

	return ts.rc.DB.Transaction(func(trx *gorm.DB) error {
		paymentIntent.Status = models.PaymentIntentStatusRefund
		if _, err := paymentIntentRepository.Update(paymentIntent, trx); err != nil {
			return responses.InternalServerError(err)
		}

		if invoice != nil {
			now := time.Now()
			invoice.Status = models.InvoiceStatusRefunded
			invoice.RefundedAt = &now
			if _, err := invoiceRepository.Update(invoice, trx); err != nil {
				return responses.InternalServerError(err)
			}
		}

		if settlement != nil {
			settlement.Status = models.SettlementStatusRefunded
			if _, err := settlementRepository.Update(settlement, trx); err != nil {
				return responses.InternalServerError(err)
			}
		}

		refund.PaymentID = &paymentIntent.ID
		refund.InvoiceID = paymentIntent.InvoiceID
		refund.NombaTransactionId = &transactionId
		refund, err = refundRepository.Create(refund, trx)
		if err != nil {
			return responses.InternalServerError(err)
		}

		if err := queue.EnqueueTenantWebhook(
			ts.rc,
			ts.publisher,
			tenantId,
			models.WebhookDeliveryEventTypeInvoiceRefunded,
			map[string]interface{}{
				"invoice": invoice,
				"refund":  refund,
			},
			trx,
		); err != nil {
			return responses.InternalServerError(err)
		}

		return nil
	})
}

func refundResponse(refund models.Refund) responses.RefundResponse {
	var invoice *responses.RefundInvoiceResponse
	if refund.Invoice != nil {
		invoice = &responses.RefundInvoiceResponse{
			ID:              refund.Invoice.ID,
			Code:            refund.Invoice.Code,
			Status:          string(refund.Invoice.Status),
			AmountDue:       refund.Invoice.AmountDue,
			AmountPaid:      refund.Invoice.AmountPaid,
			AmountRemaining: refund.Invoice.AmountRemaining,
			Currency:        refund.Invoice.Currency,
			PaidAt:          refund.Invoice.PaidAt,
			RefundedAt:      refund.Invoice.RefundedAt,
			CreatedAt:       refund.Invoice.CreatedAt,
		}
	}

	return responses.RefundResponse{
		ID:                 refund.ID,
		PaymentID:          refund.PaymentID,
		InvoiceID:          refund.InvoiceID,
		NombaTransactionID: refund.NombaTransactionId,
		Amount:             refund.Amount,
		Currency:           refund.Currency,
		Reason:             refund.Reason,
		InitiatedAt:        refund.InitiatedAt,
		ETAFrom:            refund.ETAFrom,
		ETATo:              refund.ETATo,
		Metadata:           refund.Metadata,
		Card:               refund.Card,
		Bank:               refund.Bank,
		CreatedAt:          refund.CreatedAt,
		UpdatedAt:          refund.UpdatedAt,
		Invoice:            invoice,
	}
}

func parseRefundDate(value string, endOfDay bool) (*time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return nil, err
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}
