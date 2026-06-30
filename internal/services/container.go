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
	authService := NewAuthService(rc)
	customerService := NewCustomerService(rc)
	planService := NewPlanService(rc)
	transactionService := NewTransactionService(rc, nombaProvider, customerService)

	return &Container{
		authService,
		customerService,
		planService,
		transactionService,
	}
}
