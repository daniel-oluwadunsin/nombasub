package services

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/providers/nomba"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
)

type TenantConnectionService struct {
	rc *repositories.Container
}

func NewTenantConnectionService(rc *repositories.Container) *TenantConnectionService {
	return &TenantConnectionService{rc: rc}
}

func (s *TenantConnectionService) GetTenantNombaConnection(tenantID string) (*nomba.TenantConnection, error) {
	tenantRepository := s.rc.TenantRepository

	tenant, err := tenantRepository.Find(&models.Tenant{BaseModel: models.BaseModel{ID: tenantID}}, nil)

	if err != nil {
		return nil, err
	}

	if tenant == nil {
		return nil, nomba.ErrTenantConnectionNotFound
	}

	return &nomba.TenantConnection{
		ClientID:             tenant.ClientID,
		ClientSecret:         tenant.ClientSecret,
		AccountID:            tenant.AccountID,
		AccessToken:          tenant.AccessToken,
		RefreshToken:         tenant.RefreshToken,
		AccessTokenExpiresAt: tenant.AccessTokenExpiresAt,
	}, nil
}

func (s *TenantConnectionService) SaveTenantNombaConnection(tenantID string, connection *nomba.TenantConnection) error {
	tenantRepository := s.rc.TenantRepository

	tenant, err := tenantRepository.Find(&models.Tenant{BaseModel: models.BaseModel{ID: tenantID}}, nil)

	if err != nil {
		return err
	}

	if tenant == nil {
		return nomba.ErrTenantConnectionNotFound
	}

	tenant.ClientID = connection.ClientID
	tenant.ClientSecret = connection.ClientSecret
	tenant.AccountID = connection.AccountID
	tenant.AccessToken = connection.AccessToken
	tenant.RefreshToken = connection.RefreshToken
	tenant.AccessTokenExpiresAt = connection.AccessTokenExpiresAt

	_, err = tenantRepository.Update(tenant)

	if err != nil {
		return err
	}

	return nil
}
