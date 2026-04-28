package data

import (
	"context"
	"strings"

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
		BindingID:         profile.BindingID,
		PlatformAccountID: profile.PlatformAccountID,
		GameBiz:           profile.GameBiz,
		Region:            profile.Region,
		PlayerID:          profile.PlayerID,
		Nickname:          profile.Nickname,
		Level:             profile.Level,
		IsDefault:         profile.IsDefault,
		DiscoveredAt:      profile.DiscoveredAt,
	}

	return dbFromContext(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "binding_id"}, {Name: "player_id"}, {Name: "region"}},
		DoUpdates: clause.AssignmentColumns([]string{"platform_account_id", "game_biz", "nickname", "level", "is_default", "updated_at"}),
	}).Create(&record).Error
}

func (r *ProfileRepo) ListByBindingID(ctx context.Context, bindingID uint64) ([]*biz.Profile, error) {
	var records []model.AccountProfile
	if err := dbFromContext(ctx, r.db).Where("binding_id = ?", bindingID).Order("id asc").Find(&records).Error; err != nil {
		return nil, err
	}

	return profilesFromRecords(records), nil
}

func (r *ProfileRepo) ListByPlatformAccountID(ctx context.Context, platformAccountID string) ([]*biz.Profile, error) {
	var records []model.AccountProfile
	if err := dbFromContext(ctx, r.db).Where("platform_account_id = ?", platformAccountID).Order("id asc").Find(&records).Error; err != nil {
		return nil, err
	}

	return profilesFromRecords(records), nil
}

func (r *ProfileRepo) DeleteByPlatformAccountID(ctx context.Context, platformAccountID string) error {
	return dbFromContext(ctx, r.db).Where("platform_account_id = ?", platformAccountID).Delete(&model.AccountProfile{}).Error
}

func (r *ProfileRepo) DeleteMissingByBindingID(ctx context.Context, bindingID uint64, keep []biz.ProfileIdentity) error {
	query := dbFromContext(ctx, r.db).Where("binding_id = ?", bindingID)
	if len(keep) == 0 {
		return query.Delete(&model.AccountProfile{}).Error
	}

	clauses := make([]string, 0, len(keep))
	args := make([]any, 0, len(keep)*2)
	for _, identity := range keep {
		clauses = append(clauses, "(player_id = ? AND region = ?)")
		args = append(args, identity.PlayerID, identity.Region)
	}
	return query.Where("NOT ("+strings.Join(clauses, " OR ")+")", args...).Delete(&model.AccountProfile{}).Error
}

func (r *ProfileRepo) DeleteMissingByPlatformAccountID(ctx context.Context, platformAccountID string, keep []biz.ProfileIdentity) error {
	var records []model.AccountProfile
	if err := dbFromContext(ctx, r.db).Where("platform_account_id = ?", platformAccountID).Find(&records).Error; err != nil {
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
		if err := dbFromContext(ctx, r.db).Delete(&model.AccountProfile{}, record.ID).Error; err != nil {
			return err
		}
	}
	return nil
}

func profilesFromRecords(records []model.AccountProfile) []*biz.Profile {
	profiles := make([]*biz.Profile, 0, len(records))
	for _, record := range records {
		record := record
		profiles = append(profiles, &biz.Profile{
			ID:                record.ID,
			BindingID:         record.BindingID,
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

	return profiles
}
