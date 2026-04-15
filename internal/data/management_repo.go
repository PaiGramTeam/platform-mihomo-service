package data

import (
	"context"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"platform-mihomo-service/internal/data/model"
)

type ManagementRepo struct {
	db     *gorm.DB
	redis  *redis.Client
	prefix string
}

func NewManagementRepo(db *gorm.DB, redisClient *redis.Client, prefix string) *ManagementRepo {
	return &ManagementRepo{db: db, redis: redisClient, prefix: prefix}
}

func (r *ManagementRepo) DeleteCredentialGraph(ctx context.Context, platformAccountID string) error {
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("platform_account_id = ?", platformAccountID).Delete(&model.RuntimeArtifact{}).Error; err != nil {
			return err
		}
		if err := tx.Where("platform_account_id = ?", platformAccountID).Delete(&model.AccountProfile{}).Error; err != nil {
			return err
		}
		if err := tx.Where("platform_account_id = ?", platformAccountID).Delete(&model.DeviceRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Where("platform_account_id = ?", platformAccountID).Delete(&model.CredentialRecord{}).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	if r.redis != nil {
		artifactRepo := NewArtifactRepo(r.db, r.redis, r.prefix)
		if err := artifactRepo.DeleteByPlatformAccountID(ctx, platformAccountID); err != nil {
			return err
		}
	}

	return nil
}
