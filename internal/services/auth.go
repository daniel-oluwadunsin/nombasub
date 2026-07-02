package services

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
)

type AuthService struct {
	rc *repositories.Container
}

func NewAuthService(rc *repositories.Container) *AuthService {
	return &AuthService{rc: rc}
}

func (s *AuthService) assignNewApiKey() (string, error) {
	apiKey, err := utils.GenerateRandomString(32)
	if err != nil {
		return "", err
	}

	existingTenant, err := s.rc.TenantRepository.Find(&models.Tenant{ApiKey: apiKey}, nil)
	if err != nil {
		return "", err
	}

	if existingTenant != nil {
		return s.assignNewApiKey()
	}

	return apiKey, nil
}

func (s *AuthService) RegisterTenant(body requests.SignUpTenantRequest) (*string, error) {
	tenantRepository := s.rc.TenantRepository

	account, err := tenantRepository.Find(&models.Tenant{AccountID: body.AccountID}, nil)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if account != nil {
		return nil, responses.Conflict("A tenant has already been created for this account id")
	}

	apiKey, err := s.assignNewApiKey()
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	tenant := &models.Tenant{
		BusinessName: &body.BusinessName,
		AccountID:    body.AccountID,
		ApiKey:       apiKey,
	}

	_, err = tenantRepository.Create(tenant, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &tenant.ApiKey, nil
}

func (s *AuthService) LoginTenant(body requests.LoginTenantRequest) (*string, error) {
	tenantRepository := s.rc.TenantRepository

	tenant, err := tenantRepository.Find(&models.Tenant{AccountID: body.AccountID}, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if tenant == nil {
		return nil, responses.NotFound("Tenant not found")
	}

	return &tenant.ApiKey, nil
}

func (s *AuthService) SetWebhookUrl(tenantId string, webhookUrl string) (*string, error) {
	tenantRepository := s.rc.TenantRepository

	tenant, err := tenantRepository.FindById(tenantId, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if tenant == nil {
		return nil, responses.NotFound("Tenant not found")
	}

	tenant.WebhookUrl = &webhookUrl

	_, err = tenantRepository.Update(tenant, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	webhookSecret, err := utils.GenerateRandomString(64)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &webhookSecret, nil
}
