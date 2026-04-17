package usecase

import (
	"context"
	"errors"
	"time"

	v1 "platform-mihomo-service/api/mihomo/v1"
	"platform-mihomo-service/internal/biz"
	internalcrypto "platform-mihomo-service/internal/crypto"
	platformmihomo "platform-mihomo-service/internal/platform/mihomo"
)

type BindCredentialInput struct {
	BindingID        uint64
	CookieBundleJSON string
	DeviceID         string
	DeviceFP         string
	DeviceName       string
	RegionHint       string
}

type BindCredentialOutput struct {
	BindingID         uint64
	PlatformAccountID string
	Profiles          []v1.ProfileSummary
	Status            v1.CredentialStatus
}

type BindUsecase struct {
	credentialRepo biz.CredentialRepository
	deviceRepo     biz.DeviceRepository
	profileRepo    biz.ProfileRepository
	client         platformmihomo.Client
	encryptionKey  []byte
}

func NewBindUsecase(
	credentialRepo biz.CredentialRepository,
	deviceRepo biz.DeviceRepository,
	profileRepo biz.ProfileRepository,
	client platformmihomo.Client,
	encryptionKey []byte,
) *BindUsecase {
	return &BindUsecase{
		credentialRepo: credentialRepo,
		deviceRepo:     deviceRepo,
		profileRepo:    profileRepo,
		client:         client,
		encryptionKey:  encryptionKey,
	}
}

func (uc *BindUsecase) BindCredential(ctx context.Context, input BindCredentialInput) (*BindCredentialOutput, error) {
	if input.BindingID == 0 {
		return nil, errors.New("binding id is required")
	}

	accountID, region, discoveredProfiles, err := uc.client.ValidateAndDiscover(ctx, input.CookieBundleJSON, input.RegionHint)
	if err != nil {
		return nil, err
	}

	platformAccountID := FormatPlatformAccountID(input.BindingID, accountID)
	encryptedBlob, err := internalcrypto.EncryptString(uc.encryptionKey, input.CookieBundleJSON)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if err := uc.credentialRepo.Save(ctx, &biz.Credential{
		BindingID:         input.BindingID,
		PlatformAccountID: platformAccountID,
		Platform:          "mihomo",
		AccountID:         accountID,
		Region:            region,
		CredentialBlob:    encryptedBlob,
		CredentialVersion: "v1",
		Status:            "active",
		LastValidatedAt:   &now,
	}); err != nil {
		return nil, err
	}
	rollback := true
	defer func() {
		if rollback {
			_ = uc.profileRepo.DeleteByPlatformAccountID(ctx, platformAccountID)
			_ = uc.deviceRepo.DeleteByPlatformAccountID(ctx, platformAccountID)
			_ = uc.credentialRepo.DeleteByPlatformAccountID(ctx, platformAccountID)
		}
	}()

	device := &biz.Device{
		PlatformAccountID: platformAccountID,
		DeviceID:          input.DeviceID,
		DeviceFP:          input.DeviceFP,
		IsValid:           true,
		LastSeenAt:        &now,
	}
	if input.DeviceName != "" {
		deviceName := input.DeviceName
		device.DeviceName = &deviceName
	}
	if err := uc.deviceRepo.Save(ctx, device); err != nil {
		return nil, err
	}

	outputProfiles := make([]v1.ProfileSummary, 0, len(discoveredProfiles))
	for index, discoveredProfile := range discoveredProfiles {
		profile := &biz.Profile{
			BindingID:         input.BindingID,
			PlatformAccountID: platformAccountID,
			GameBiz:           discoveredProfile.GameBiz,
			Region:            discoveredProfile.Region,
			PlayerID:          discoveredProfile.PlayerID,
			Nickname:          discoveredProfile.Nickname,
			Level:             int(discoveredProfile.Level),
			IsDefault:         index == 0,
			DiscoveredAt:      now,
		}
		if err := uc.profileRepo.Save(ctx, profile); err != nil {
			return nil, err
		}

		outputProfiles = append(outputProfiles, *toProfileSummary(profile))
	}

	rollback = false

	return &BindCredentialOutput{
		BindingID:         input.BindingID,
		PlatformAccountID: platformAccountID,
		Profiles:          outputProfiles,
		Status:            v1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
	}, nil
}

func (uc *BindUsecase) UpsertDevice(ctx context.Context, platformAccountID string, deviceID string, deviceFP string, deviceName string) error {
	credential, err := uc.credentialRepo.GetByPlatformAccountID(ctx, platformAccountID)
	if err != nil {
		return err
	}
	if credential == nil {
		return ErrCredentialNotFound
	}

	device := &biz.Device{
		PlatformAccountID: platformAccountID,
		DeviceID:          deviceID,
		DeviceFP:          deviceFP,
		IsValid:           true,
	}
	if deviceName != "" {
		name := deviceName
		device.DeviceName = &name
	}
	now := time.Now().UTC()
	device.LastSeenAt = &now
	return uc.deviceRepo.Save(ctx, device)
}
