package services

import (
	"fmt"

	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
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
		// checkoutOrder.Order.AllowedPaymentMethods = utils.ToPtr([]nomba.PaymentMethod{nomba.PaymentMethodCard})
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
