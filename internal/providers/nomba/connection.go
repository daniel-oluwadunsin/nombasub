package nomba

import "time"

type TenantConnection struct {
	ClientID             string
	ClientSecret         string
	AccountID            string
	AccessToken          *string
	RefreshToken         *string
	AccessTokenExpiresAt *time.Time
}

type TenantConnectionService interface {
	GetTenantNombaConnection(tenantID string) (*TenantConnection, error)
	SaveTenantNombaConnection(tenantID string, connection *TenantConnection) error
}
