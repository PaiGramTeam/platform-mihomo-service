package data

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"platform-mihomo-service/internal/biz"
	"platform-mihomo-service/internal/data/model"
)

type ProfileRepo struct {
	db *gorm.DB
}

func NewProfileRepo(db *gorm.DB) *ProfileRepo {
	return &ProfileRepo{db: db}
}

func (r *ProfileRepo) Save(ctx context.Context, profile *biz.Profile) error {
	record := model.AccountProfile{
		PlatformAccountID: profile.PlatformAccountID,
		GameBiz:           profile.GameBiz,
		Region:            profile.Region,
		PlayerID:          profile.PlayerID,
		Nickname:          profile.Nickname,
		Level:             profile.Level,
		IsDefault:         profile.IsDefault,
		DiscoveredAt:      profile.DiscoveredAt,
	}

	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "platform_account_id"}, {Name: "player_id"}, {Name: "region"}},
		DoUpdates: clause.AssignmentColumns([]string{"game_biz", "nickname", "level", "is_default", "updated_at"}),
	}).Create(&record).Error
}

func (r *ProfileRepo) ListByPlatformAccountID(ctx context.Context, platformAccountID string) ([]*biz.Profile, error) {
	var records []model.AccountProfile
	if err := r.db.WithContext(ctx).Where("platform_account_id = ?", platformAccountID).Order("id asc").Find(&records).Error; err != nil {
		return nil, err
	}

	profiles := make([]*biz.Profile, 0, len(records))
	for _, record := range records {
		record := record
		profiles = append(profiles, &biz.Profile{
			ID:                record.ID,
			PlatformAccountID: record.PlatformAccountID,
			GameBiz:           record.GameBiz,
			Region:            record.Region,
			PlayerID:          record.PlayerID,
			Nickname:          record.Nickname,
			Level:             record.Level,
			IsDefault:         record.IsDefault,
			DiscoveredAt:      record.DiscoveredAt,
		})
	}

	return profiles, nil
}

func (r *ProfileRepo) DeleteByPlatformAccountID(ctx context.Context, platformAccountID string) error {
	return r.db.WithContext(ctx).Where("platform_account_id = ?", platformAccountID).Delete(&model.AccountProfile{}).Error
}

func (r *ProfileRepo) DeleteMissingByPlatformAccountID(ctx context.Context, platformAccountID string, keep []biz.ProfileIdentity) error {
	var records []model.AccountProfile
	if err := r.db.WithContext(ctx).Where("platform_account_id = ?", platformAccountID).Find(&records).Error; err != nil {
		return err
	}
	keepSet := make(map[string]struct{}, len(keep))
	for _, identity := range keep {
		keepSet[identity.PlayerID+":"+identity.Region] = struct{}{}
	}
	for _, record := range records {
		key := record.PlayerID + ":" + record.Region
		if _, ok := keepSet[key]; ok {
			continue
		}
		if err := r.db.WithContext(ctx).Delete(&model.AccountProfile{}, record.ID).Error; err != nil {
			return err
		}
	}
	return nil
}
