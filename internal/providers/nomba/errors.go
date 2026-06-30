package nomba

import "errors"

var (
	ErrTenantConnectionNotFound         error = errors.New("tenant connection not found")
	FailedToIssueAccessTokenForTenant   error = errors.New("failed to issue access token for tenant")
	FailedToRefreshAccessTokenForTenant error = errors.New("failed to refresh access token for tenant")
)
