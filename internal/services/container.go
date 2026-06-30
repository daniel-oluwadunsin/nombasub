package services

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
)

type Container struct {
	AuthService        *AuthService
	CustomerService    *CustomerService
	PlanService        *PlanService
	TransactionService *TransactionService
}

func NewContainer(rc *repositories.Container, nombaProvider nomba.Provider) *Container {
	return &Container{
		AuthService:        NewAuthService(rc),
		CustomerService:    NewCustomerService(rc),
		PlanService:        NewPlanService(rc),
		TransactionService: NewTransactionService(rc, nombaProvider),
	}
}
