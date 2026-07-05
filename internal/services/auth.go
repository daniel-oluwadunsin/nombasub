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

type TenantSettings struct {
	BusinessName  *string `json:"businessName"`
	AccountID     string  `json:"accountId"`
	ApiKey        string  `json:"apiKey"`
	WebhookUrl    *string `json:"webhookUrl"`
	WebhookSecret *string `json:"webhookSecret"`
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

func (s *AuthService) SignOutTenant(tenantId string) error {
	tenantRepository := s.rc.TenantRepository

	tenant, err := tenantRepository.FindById(tenantId, nil)
	if err != nil {
		return responses.InternalServerError(err)
	}

	if tenant == nil {
		return responses.NotFound("Tenant not found")
	}

	tenant.AccessToken = nil
	tenant.AccessTokenExpiresAt = nil

	if _, err := tenantRepository.Update(tenant, nil); err != nil {
		return responses.InternalServerError(err)
	}

	return nil
}

func (s *AuthService) GetTenantSettings(tenantId string) (*TenantSettings, error) {
	tenant, err := s.rc.TenantRepository.FindById(tenantId, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if tenant == nil {
		return nil, responses.NotFound("Tenant not found")
	}

	return settingsFromTenant(tenant), nil
}

func (s *AuthService) UpdateTenantSettings(tenantId string, body requests.UpdateTenantSettingsRequest) (*TenantSettings, error) {
	tenant, err := s.rc.TenantRepository.FindById(tenantId, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if tenant == nil {
		return nil, responses.NotFound("Tenant not found")
	}

	if body.BusinessName != nil {
		tenant.BusinessName = body.BusinessName
	}
	if body.WebhookUrl != nil {
		tenant.WebhookUrl = body.WebhookUrl
	}

	updatedTenant, err := s.rc.TenantRepository.Update(tenant, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return settingsFromTenant(updatedTenant), nil
}

func (s *AuthService) RotateTenantApiKey(tenantId string) (*TenantSettings, error) {
	tenant, err := s.rc.TenantRepository.FindById(tenantId, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if tenant == nil {
		return nil, responses.NotFound("Tenant not found")
	}

	apiKey, err := s.assignNewApiKey()
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	tenant.ApiKey = apiKey
	updatedTenant, err := s.rc.TenantRepository.Update(tenant, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return settingsFromTenant(updatedTenant), nil
}

func (s *AuthService) RotateTenantWebhookSecret(tenantId string) (*TenantSettings, error) {
	tenant, err := s.rc.TenantRepository.FindById(tenantId, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if tenant == nil {
		return nil, responses.NotFound("Tenant not found")
	}

	webhookSecret, err := utils.GenerateRandomString(64)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	tenant.WebhookSecret = &webhookSecret
	updatedTenant, err := s.rc.TenantRepository.Update(tenant, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return settingsFromTenant(updatedTenant), nil
}

func (s *AuthService) ChangeTenantPassword(tenantId string, body requests.ChangePasswordRequest) error {
	tenant, err := s.rc.TenantRepository.FindById(tenantId, nil)
	if err != nil {
		return responses.InternalServerError(err)
	}
	if tenant == nil {
		return responses.NotFound("Tenant not found")
	}
	if tenant.Password == nil {
		return responses.Forbidden("password not set")
	}
	if !utils.ValidateHash(*tenant.Password, body.OldPassword) {
		return responses.Unauthorized("Old password is incorrect")
	}

	hashedPassword, err := utils.Hash(body.NewPassword)
	if err != nil {
		return responses.InternalServerError(err)
	}

	tenant.Password = &hashedPassword
	if _, err := s.rc.TenantRepository.Update(tenant, nil); err != nil {
		return responses.InternalServerError(err)
	}

	return nil
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
	webhookSecret, err := utils.GenerateRandomString(64)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}
	if tenant.WebhookSecret == nil {
		tenant.WebhookSecret = &webhookSecret
	}

	_, err = tenantRepository.Update(tenant, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &webhookSecret, nil
}

func settingsFromTenant(tenant *models.Tenant) *TenantSettings {
	return &TenantSettings{
		BusinessName:  tenant.BusinessName,
		AccountID:     tenant.AccountID,
		ApiKey:        tenant.ApiKey,
		WebhookUrl:    tenant.WebhookUrl,
		WebhookSecret: tenant.WebhookSecret,
	}
}
