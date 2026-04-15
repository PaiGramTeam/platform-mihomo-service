package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"platform-mihomo-service/internal/biz"
	"platform-mihomo-service/internal/data/model"
)

type ArtifactRepo struct {
	db     *gorm.DB
	redis  *redis.Client
	prefix string
}

func NewArtifactRepo(db *gorm.DB, redisClient *redis.Client, prefix string) *ArtifactRepo {
	return &ArtifactRepo{db: db, redis: redisClient, prefix: prefix}
}

func (r *ArtifactRepo) Put(ctx context.Context, artifact *biz.Artifact) error {
	record := model.RuntimeArtifact{
		PlatformAccountID: artifact.PlatformAccountID,
		ArtifactType:      artifact.ArtifactType,
		ArtifactValue:     artifact.ArtifactValue,
		ScopeKey:          artifact.ScopeKey,
		ExpiresAt:         artifact.ExpiresAt,
	}

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "platform_account_id"}, {Name: "artifact_type"}, {Name: "scope_key"}},
		UpdateAll: true,
	}).Create(&record).Error; err != nil {
		return err
	}

	if r.redis == nil {
		return nil
	}

	payload, err := json.Marshal(artifact)
	if err != nil {
		return err
	}

	ttl := time.Until(artifact.ExpiresAt)
	if ttl <= 0 {
		return r.redis.Del(ctx, r.cacheKey(artifact.PlatformAccountID, artifact.ArtifactType, artifact.ScopeKey)).Err()
	}

	return r.redis.Set(ctx, r.cacheKey(artifact.PlatformAccountID, artifact.ArtifactType, artifact.ScopeKey), payload, ttl).Err()
}

func (r *ArtifactRepo) Get(ctx context.Context, platformAccountID, artifactType, scopeKey string) (*biz.Artifact, error) {
	if r.redis != nil {
		payload, err := r.redis.Get(ctx, r.cacheKey(platformAccountID, artifactType, scopeKey)).Bytes()
		if err == nil {
			artifact := &biz.Artifact{}
			if err := json.Unmarshal(payload, artifact); err == nil {
				return artifact, nil
			}
		}
	}

	var record model.RuntimeArtifact
	err := r.db.WithContext(ctx).Where(
		"platform_account_id = ? AND artifact_type = ? AND scope_key = ? AND expires_at > ?",
		platformAccountID,
		artifactType,
		scopeKey,
		time.Now(),
	).Take(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	artifact := &biz.Artifact{
		PlatformAccountID: record.PlatformAccountID,
		ArtifactType:      record.ArtifactType,
		ArtifactValue:     record.ArtifactValue,
		ScopeKey:          record.ScopeKey,
		ExpiresAt:         record.ExpiresAt,
	}

	if r.redis != nil {
		payload, err := json.Marshal(artifact)
		if err == nil {
			ttl := time.Until(artifact.ExpiresAt)
			if ttl <= 0 {
				_ = r.redis.Del(ctx, r.cacheKey(platformAccountID, artifactType, scopeKey)).Err()
				return artifact, nil
			}
			_ = r.redis.Set(ctx, r.cacheKey(platformAccountID, artifactType, scopeKey), payload, ttl).Err()
		}
	}

	return artifact, nil
}

func (r *ArtifactRepo) DeleteByPlatformAccountID(ctx context.Context, platformAccountID string) error {
	if err := r.db.WithContext(ctx).Where("platform_account_id = ?", platformAccountID).Delete(&model.RuntimeArtifact{}).Error; err != nil {
		return err
	}

	if r.redis != nil {
		pattern := fmt.Sprintf("%sartifact:%s:*", r.prefix, platformAccountID)
		var cursor uint64
		for {
			keys, nextCursor, err := r.redis.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				return err
			}
			if len(keys) > 0 {
				if err := r.redis.Del(ctx, keys...).Err(); err != nil {
					return err
				}
			}
			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
	}

	return nil
}

func (r *ArtifactRepo) cacheKey(platformAccountID, artifactType, scopeKey string) string {
	return fmt.Sprintf("%sartifact:%s:%s:%s", r.prefix, platformAccountID, artifactType, scopeKey)
}
