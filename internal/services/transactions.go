package services

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
)

type TransactionService struct {
	rc            *repositories.Container
	nombaProvider nomba.Provider
}

func NewTransactionService(rc *repositories.Container, nombaProvider nomba.Provider) *TransactionService {
	return &TransactionService{
		rc:            rc,
		nombaProvider: nombaProvider,
	}
}

func (ts *TransactionService) InitializeTransaction(tenantId string, body requests.CreateCheckoutOrderRequest) (interface{}, error) {
	planRepository := ts.rc.PlanRepository
	nombaProvider := ts.nombaProvider

	plan, err := planRepository.Find(&models.Plan{Code: body.PlanCode, TenantID: tenantId}, nil)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if plan == nil {
		return nil, responses.NotFound("Plan not found")
	}

	nombaResponse, err := nombaProvider.CreateCheckoutOrder(body.CreateCheckoutOrderRequest)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return nombaResponse, nil
}
