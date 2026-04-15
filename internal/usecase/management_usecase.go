package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "platform-mihomo-service/api/mihomo/v1"
	"platform-mihomo-service/internal/biz"
)

var ErrPlatformAccountMismatch = errors.New("platform account does not match requested credential")

type CredentialSummaryOutput struct {
	PlatformAccountID string
	Status            v1.CredentialStatus
	LastValidatedAt   *time.Time
	LastRefreshedAt   *time.Time
	Devices           []*biz.Device
	Profiles          []*v1.ProfileSummary
}

type ManagementUsecase struct {
	credentials biz.CredentialRepository
	devices     biz.DeviceRepository
	profiles    biz.ProfileRepository
	artifacts   biz.ArtifactRepository
	management  biz.CredentialManagementRepository
	bindUC      *BindUsecase
	profileUC   *ProfileUsecase
}

func NewManagementUsecase(
	credentials biz.CredentialRepository,
	devices biz.DeviceRepository,
	profiles biz.ProfileRepository,
	artifacts biz.ArtifactRepository,
	management biz.CredentialManagementRepository,
	bindUC *BindUsecase,
	profileUC *ProfileUsecase,
) *ManagementUsecase {
	return &ManagementUsecase{
		credentials: credentials,
		devices:     devices,
		profiles:    profiles,
		artifacts:   artifacts,
		management:  management,
		bindUC:      bindUC,
		profileUC:   profileUC,
	}
}

func (uc *ManagementUsecase) GetCredentialSummary(ctx context.Context, platformAccountID string) (*CredentialSummaryOutput, error) {
	credential, err := uc.credentials.GetByPlatformAccountID(ctx, platformAccountID)
	if err != nil {
		return nil, err
	}
	if credential == nil {
		return nil, ErrCredentialNotFound
	}

	devices, err := uc.devices.ListByPlatformAccountID(ctx, platformAccountID)
	if err != nil {
		return nil, err
	}

	profiles, err := uc.profileUC.ListProfiles(ctx, platformAccountID)
	if err != nil {
		return nil, err
	}

	return &CredentialSummaryOutput{
		PlatformAccountID: credential.PlatformAccountID,
		Status:            toCredentialStatus(credential.Status),
		LastValidatedAt:   credential.LastValidatedAt,
		LastRefreshedAt:   credential.LastRefreshedAt,
		Devices:           devices,
		Profiles:          profiles,
	}, nil
}

type UpdateCredentialInput struct {
	PlatformAccountID string
	BindCredentialInput
}

func (uc *ManagementUsecase) UpdateCredential(ctx context.Context, input UpdateCredentialInput) (*CredentialSummaryOutput, error) {
	credential, err := uc.credentials.GetByPlatformAccountID(ctx, input.PlatformAccountID)
	if err != nil {
		return nil, err
	}
	if credential == nil {
		return nil, ErrCredentialNotFound
	}

	output, err := uc.bindUC.BindCredential(ctx, input.BindCredentialInput)
	if err != nil {
		return nil, err
	}
	if output.PlatformAccountID != input.PlatformAccountID {
		if cleanupErr := uc.management.DeleteCredentialGraph(ctx, output.PlatformAccountID); cleanupErr != nil {
			return nil, fmt.Errorf("%w: cleanup failed: %v", ErrPlatformAccountMismatch, cleanupErr)
		}
		return nil, ErrPlatformAccountMismatch
	}
	if err := uc.pruneStaleProfiles(ctx, input.PlatformAccountID, output.Profiles); err != nil {
		return nil, err
	}

	return uc.GetCredentialSummary(ctx, input.PlatformAccountID)
}

func (uc *ManagementUsecase) DeleteCredential(ctx context.Context, platformAccountID string) error {
	return uc.management.DeleteCredentialGraph(ctx, platformAccountID)
}

func (uc *ManagementUsecase) pruneStaleProfiles(ctx context.Context, platformAccountID string, profiles []v1.ProfileSummary) error {
	keep := make([]biz.ProfileIdentity, 0, len(profiles))
	for i := range profiles {
		profile := &profiles[i]
		keep = append(keep, biz.ProfileIdentity{PlayerID: profile.PlayerId, Region: profile.Region})
	}
	return uc.profiles.DeleteMissingByPlatformAccountID(ctx, platformAccountID, keep)
}
