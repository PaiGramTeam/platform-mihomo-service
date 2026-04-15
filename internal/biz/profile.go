package biz

import (
	"context"
	"time"
)

type ProfileIdentity struct {
	PlayerID string
	Region   string
}

type Profile struct {
	ID                uint64
	PlatformAccountID string
	GameBiz           string
	Region            string
	PlayerID          string
	Nickname          string
	Level             int
	IsDefault         bool
	DiscoveredAt      time.Time
}

type ProfileRepository interface {
	Save(ctx context.Context, profile *Profile) error
	ListByPlatformAccountID(ctx context.Context, platformAccountID string) ([]*Profile, error)
	DeleteByPlatformAccountID(ctx context.Context, platformAccountID string) error
	DeleteMissingByPlatformAccountID(ctx context.Context, platformAccountID string, keep []ProfileIdentity) error
}
