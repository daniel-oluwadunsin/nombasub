package nomba

import "errors"

var (
	ErrConnectionNotFound      error = errors.New("connection not found")
	FailedToIssueAccessToken   error = errors.New("failed to issue access token")
	FailedToRefreshAccessToken error = errors.New("failed to refresh access token")
)
