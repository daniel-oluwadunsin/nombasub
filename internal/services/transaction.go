package services

import (
	"fmt"
	"log"

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
			return err
		}

		subscriptionExists, err := subscriptionRepository.Exists(&models.Subscription{
			TenantID:   tenantId,
			CustomerID: customer.ID,
			PlanID:     plan.PlanID,
			Status:     models.SubscriptionStatusActive,
		}, nil)
		if err != nil {
			return err
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
		checkoutOrder.Order.Amount = plan.Amount
		checkoutOrder.Order.Currency = &plan.Currency
		checkoutOrder.TokenizeCard = utils.ToPtr(true)
		checkoutOrder.Order.AccountId = utils.ToPtr(tenantAccountId)

		reference, err := utils.GenerateRandomString(24)
		if err != nil {
			return err
		}
		reference = fmt.Sprintf("nombasub_%s", reference)
		checkoutOrder.Order.OrderReference = &reference

		nombaInitiation, err := nombaInitiationRepository.Create(&models.NombaInitiation{
			TenantID:  tenantId,
			Amount:    float64(checkoutOrder.Order.Amount),
			Currency:  *checkoutOrder.Order.Currency,
			Reference: reference,
			Purpose:   models.NombaInitiationPurposeCardSubscriptionPayment,
			Metadata:  metadata,
		}, trx)

		if err != nil {
			return err
		}

		nombaResponse, err = nombaProvider.CreateCheckoutOrder(checkoutOrder)
		if err != nil {
			return err
		}

		nombaInitiation.NombaOrderID = &nombaResponse.Data.OrderReference
		_, err = nombaInitiationRepository.Update(nombaInitiation, trx)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return nombaResponse, nil
}

type InitializeDirectDebitResponse struct {
	MandateID           string `json:"mandateId"`
	MerchantReference   string `json:"merchantReference"`
	CustomerPhoneNumber string `json:"customerPhoneNumber"`
	Description         string `json:"description"`
}

func (ts *TransactionService) InitializeDirectDebitSubscription(tenantId string, body requests.InitializeDirectDebitRequest) (*InitializeDirectDebitResponse, error) {
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

	var result *InitializeDirectDebitResponse

	err = db.Transaction(func(trx *gorm.DB) error {
		customer, err := customerService.GetOrCreateCustomer(
			tenantId,
			models.Customer{TenantID: tenantId, Email: body.CustomerEmail},
			trx,
		)
		if err != nil {
			return err
		}

		subscriptionExists, err := subscriptionRepository.Exists(&models.Subscription{
			TenantID:   tenantId,
			CustomerID: customer.ID,
			PlanID:     plan.PlanID,
			Status:     models.SubscriptionStatusActive,
		}, nil)
		if err != nil {
			return err
		}
		if subscriptionExists {
			return responses.BadRequest("Customer already has an active subscription for this plan")
		}

		merchantReference, err := utils.GenerateNumericString(12)
		if err != nil {
			return err
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
			return err
		}

		mandateId := nombaResponse.Data.MandateID

		_, err = nombaInitiationRepository.Create(&models.NombaInitiation{
			TenantID:     tenantId,
			Amount:       float64(plan.Amount),
			Currency:     plan.Currency,
			Reference:    mandateId,
			Purpose:      models.NombaInitiationPurposeDirectDebitSubscription,
			Status:       models.NombaInitiationStatusPending,
			Metadata:     metadata,
		}, trx)
		if err != nil {
			return err
		}

		result = &InitializeDirectDebitResponse{
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
		}); err != nil {
			log.Printf("direct debit: failed to enqueue mandate.created webhook: %v", err)
		}

		return nil
	})

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return result, nil
}
