package services

import (
	"github.com/daniel-oluwadunsin/nombasub/internal/helpers/encryption"
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

func (s *AuthService) RegisterTenant(body requests.AuthTenantRequest) (*string, error) {
	tenantRepository := s.rc.TenantRepository

	account, err := tenantRepository.Find(&models.Tenant{AccountID: body.AccountID}, nil)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if account != nil {
		return nil, responses.Conflict("A tenant has already been created for this account id")
	}

	encryptedClientSecret, err := encryption.Encrypt(body.ClientSecret, body.AccountID)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	apiKey, err := s.assignNewApiKey()
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	tenant := &models.Tenant{
		AccountID:    body.AccountID,
		ClientID:     body.ClientID,
		ClientSecret: encryptedClientSecret.Ciphertext,
		Nonce:        encryptedClientSecret.Nonce,
		Algorithm:    encryptedClientSecret.Algorithm,
		KeyVersion:   encryptedClientSecret.KeyVersion,
		ApiKey:       apiKey,
	}

	_, err = tenantRepository.Create(tenant)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	return &tenant.ApiKey, nil
}

func (s *AuthService) LoginTenant(body requests.AuthTenantRequest) (*string, error) {
	tenantRepository := s.rc.TenantRepository

	tenant, err := tenantRepository.Find(&models.Tenant{AccountID: body.AccountID}, nil)
	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if tenant == nil {
		return nil, responses.NotFound("Tenant not found")
	}

	decryptedClientSecret, err := encryption.Decrypt(encryption.EncryptedSecret{
		Ciphertext: tenant.ClientSecret,
		Nonce:      tenant.Nonce,
		Algorithm:  tenant.Algorithm,
		KeyVersion: tenant.KeyVersion,
	}, body.AccountID)

	if err != nil {
		return nil, responses.InternalServerError(err)
	}

	if decryptedClientSecret != body.ClientSecret {
		return nil, responses.Unauthorized("Invalid client secret")
	}

	return &tenant.ApiKey, nil
}

func (s *AuthService) SetWebhookUrl(tenantId string, webhookUrl string) error {
	tenantRepository := s.rc.TenantRepository

	tenant, err := tenantRepository.FindById(tenantId, nil)
	if err != nil {
		return responses.InternalServerError(err)
	}

	if tenant == nil {
		return responses.NotFound("Tenant not found")
	}

	tenant.WebhookUrl = &webhookUrl

	_, err = tenantRepository.Update(tenant)
	if err != nil {
		return responses.InternalServerError(err)
	}

	return nil
}
