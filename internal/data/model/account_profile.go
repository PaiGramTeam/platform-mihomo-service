package model

import "time"

type AccountProfile struct {
	ID                uint64    `gorm:"primaryKey"`
	PlatformAccountID string    `gorm:"size:64;not null;uniqueIndex:uniq_platform_profile,priority:1;index:idx_profile_platform_account_id"`
	GameBiz           string    `gorm:"size:64;not null"`
	Region            string    `gorm:"size:32;not null;uniqueIndex:uniq_platform_profile,priority:3"`
	PlayerID          string    `gorm:"size:64;not null;uniqueIndex:uniq_platform_profile,priority:2"`
	Nickname          string    `gorm:"size:255;not null"`
	Level             int       `gorm:"not null;default:0"`
	IsDefault         bool      `gorm:"not null;default:false"`
	DiscoveredAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP(3)"`
	UpdatedAt         time.Time
}
