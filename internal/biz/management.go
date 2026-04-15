package biz

import "context"

type CredentialManagementRepository interface {
	DeleteCredentialGraph(ctx context.Context, platformAccountID string) error
}
