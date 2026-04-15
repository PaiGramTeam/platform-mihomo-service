package biz

import (
	"context"
	"time"
)

type Device struct {
	PlatformAccountID string
	DeviceID          string
	DeviceFP          string
	DeviceName        *string
	IsValid           bool
	LastSeenAt        *time.Time
}

type DeviceRepository interface {
	Save(ctx context.Context, device *Device) error
	ListByPlatformAccountID(ctx context.Context, platformAccountID string) ([]*Device, error)
	DeleteByPlatformAccountID(ctx context.Context, platformAccountID string) error
}
