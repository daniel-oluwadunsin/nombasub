package nomba

type Factory struct {
	tenantConnectionService TenantConnectionService
}

func NewFactory(tenantConnectionService TenantConnectionService) *Factory {
	return &Factory{tenantConnectionService: tenantConnectionService}
}

func (f *Factory) ForTenant(tenantId string) (*Client, error) {
	conn, err := f.tenantConnectionService.GetTenantNombaConnection(tenantId)

	if err != nil {
		return nil, err
	}

	if conn == nil {
		return nil, ErrTenantConnectionNotFound
	}

	client, err := NewClient(
		conn.ClientID,
		conn.ClientSecret,
		conn.AccountID,
		conn.AccessToken,
		conn.RefreshToken,
		conn.AccessTokenExpiresAt,
		f.tenantConnectionService,
	)

	if err != nil {
		return nil, err
	}

	return client, nil
}
