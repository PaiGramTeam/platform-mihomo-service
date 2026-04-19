package data

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"platform-mihomo-service/internal/biz"
	"platform-mihomo-service/internal/data/model"
)

type DeviceRepo struct {
	db *gorm.DB
}

func NewDeviceRepo(db *gorm.DB) *DeviceRepo {
	return &DeviceRepo{db: db}
}

func (r *DeviceRepo) Save(ctx context.Context, device *biz.Device) error {
	record := model.DeviceRecord{
		PlatformAccountID: device.PlatformAccountID,
		DeviceID:          device.DeviceID,
		DeviceFP:          device.DeviceFP,
		DeviceName:        device.DeviceName,
		IsValid:           device.IsValid,
		LastSeenAt:        device.LastSeenAt,
	}

	return dbFromContext(ctx, r.db).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "platform_account_id"}, {Name: "device_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"device_fp", "device_name", "is_valid", "last_seen_at", "updated_at"}),
	}).Create(&record).Error
}

func (r *DeviceRepo) ListByPlatformAccountID(ctx context.Context, platformAccountID string) ([]*biz.Device, error) {
	var records []model.DeviceRecord
	if err := dbFromContext(ctx, r.db).Where("platform_account_id = ?", platformAccountID).Order("id asc").Find(&records).Error; err != nil {
		return nil, err
	}

	devices := make([]*biz.Device, 0, len(records))
	for _, record := range records {
		record := record
		devices = append(devices, &biz.Device{
			PlatformAccountID: record.PlatformAccountID,
			DeviceID:          record.DeviceID,
			DeviceFP:          record.DeviceFP,
			DeviceName:        record.DeviceName,
			IsValid:           record.IsValid,
			LastSeenAt:        record.LastSeenAt,
		})
	}

	return devices, nil
}

func (r *DeviceRepo) DeleteByPlatformAccountID(ctx context.Context, platformAccountID string) error {
	return dbFromContext(ctx, r.db).Where("platform_account_id = ?", platformAccountID).Delete(&model.DeviceRecord{}).Error
}
