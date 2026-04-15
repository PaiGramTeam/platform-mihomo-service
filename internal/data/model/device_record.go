package model

import "time"

type DeviceRecord struct {
	ID                uint64  `gorm:"primaryKey"`
	PlatformAccountID string  `gorm:"size:64;not null;uniqueIndex:uniq_device_record;index:idx_device_platform_account_id"`
	DeviceID          string  `gorm:"size:64;not null;uniqueIndex:uniq_device_record;index:idx_device_device_id"`
	DeviceFP          string  `gorm:"size:64;not null"`
	DeviceName        *string `gorm:"size:128"`
	IsValid           bool    `gorm:"not null;default:true"`
	LastSeenAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
