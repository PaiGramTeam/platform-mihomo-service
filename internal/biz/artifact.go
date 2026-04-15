package biz

import (
	"context"
	"time"
)

type Artifact struct {
	PlatformAccountID string
	ArtifactType      string
	ArtifactValue     string
	ScopeKey          string
	ExpiresAt         time.Time
}

type ArtifactRepository interface {
	Put(ctx context.Context, artifact *Artifact) error
	Get(ctx context.Context, platformAccountID, artifactType, scopeKey string) (*Artifact, error)
	DeleteByPlatformAccountID(ctx context.Context, platformAccountID string) error
}
