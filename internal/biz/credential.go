package biz

import (
	"context"
	"time"
)

type Credential struct {
	PlatformAccountID string
	Platform          string
	AccountID         string
	Region            string
	CredentialBlob    string
	CredentialVersion string
	Status            string
	LastValidatedAt   *time.Time
	LastRefreshedAt   *time.Time
	ExpiresAt         *time.Time
}

type CredentialRepository interface {
	Save(ctx context.Context, credential *Credential) error
	GetByPlatformAccountID(ctx context.Context, platformAccountID string) (*Credential, error)
	DeleteByPlatformAccountID(ctx context.Context, platformAccountID string) error
}
