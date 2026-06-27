package services

import (
	"fmt"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransactionService struct {
	rc              *repositories.Container
	nombaProvider   nomba.Provider
	customerService *CustomerService
}

func NewTransactionService(rc *repositories.Container, nombaProvider nomba.Provider, customerService *CustomerService) *TransactionService {
	return &TransactionService{
		rc:              rc,
		nombaProvider:   nombaProvider,
		customerService: customerService,
	}
}

func (ts *TransactionService) InitializeCardTransaction(tenantId, tenantAccountId string, body requests.CreateCheckoutOrderRequest) (*nomba.CreateCheckoutOrderResponse, error) {
	db := ts.rc.DB
	planRepository := ts.rc.PlanRepository
	nombaInitiationRepository := ts.rc.NombaInitiationRepository
	nombaProvider := ts.nombaProvider
	customerService := ts.customerService

	plan, err := planRepository.Find(&models.Plan{Code: body.PlanCode, TenantID: tenantId}, nil)
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

		checkoutOrder := body.CreateCheckoutOrderRequest

		tenantAccountId = *utils.Or(checkoutOrder.Order.AccountId, &tenantAccountId)
		metadata := *utils.Or(checkoutOrder.Order.OrderMetaData, new(map[string]interface{}))
		metadata["nombaSubTenantId"] = tenantId
		metadata["nombaSubCustomerCode"] = customer.Code
		metadata["nombaSubPlanCode"] = plan.Code
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

		reference := fmt.Sprintf("nombasub_%s_%s", tenantId, uuid.New().String())
		checkoutOrder.Order.OrderReference = &reference

		_, err = nombaInitiationRepository.Create(&models.NombaInitiation{
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
		fmt.Println(err)
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
