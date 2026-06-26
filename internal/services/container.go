package services

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
)

type Container struct {
	AuthService             *AuthService
	TenantConnectionService *TenantConnectionService
	CustomerService         *CustomerService
	PlanService             *PlanService
	NombaFactory            *nomba.Factory
}

func NewContainer(rc *repositories.Container) *Container {
	tenantConnectionService := NewTenantConnectionService(rc)
	nombaFactory := nomba.NewFactory(tenantConnectionService) // can be passed to other services struct that need it.

	return &Container{
		AuthService:             NewAuthService(rc),
		CustomerService:         NewCustomerService(rc),
		PlanService:             NewPlanService(rc),
		TenantConnectionService: tenantConnectionService,
		NombaFactory:            nombaFactory,
	}
}
