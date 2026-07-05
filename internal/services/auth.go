package services

import (
	"time"

	"github.com/daniel-oluwadunsin/nombasub/internal/config"
	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/utils"
	"github.com/daniel-oluwadunsin/nombasub/internal/models"
	"github.com/daniel-oluwadunsin/nombasub/internal/repositories"
	"github.com/daniel-oluwadunsin/nombasub/internal/requests"
	"github.com/daniel-oluwadunsin/nombasub/internal/responses"
)

type AuthService struct {
	rc  *repositories.Container
	cfg *config.Config
}

func NewAuthService(rc *repositories.Container, cfg *config.Config) *AuthService {
	return &AuthService{rc, cfg}
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

	hashedPassword, err := utils.Hash(body.Password)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	tenant := &models.Tenant{
		BusinessName: &body.BusinessName,
		AccountID:    body.AccountID,
		ApiKey:       apiKey,
		Password:     &hashedPassword,
	}

	_, err = tenantRepository.Create(tenant, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &tenant.ApiKey, nil
}

func (s *AuthService) LoginTenant(body requests.LoginTenantRequest) (*models.Tenant, error) {
	tenantRepository := s.rc.TenantRepository

	tenant, err := tenantRepository.Find(&models.Tenant{AccountID: body.AccountID}, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if tenant == nil {
		return nil, responses.NotFound("Tenant not found")
	}

	if tenant.Password == nil {
		return nil, responses.Forbidden("password not set")
	}

	if !utils.ValidateHash(*tenant.Password, body.Password) {
		return nil, responses.Unauthorized("Password incorrect")
	}

	accessToken, err := utils.GenerateJwt(tenant.ID, s.cfg)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	tenant.AccessToken = &accessToken
	tenant.AccessTokenExpiresAt = utils.ToPtr(time.Now().Add(24 * time.Hour))
	_, err = tenantRepository.Update(tenant, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return tenant, nil
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
