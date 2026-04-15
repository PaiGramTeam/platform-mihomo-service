package usecase

import (
	"context"
	"errors"

	v1 "platform-mihomo-service/api/mihomo/v1"
	"platform-mihomo-service/internal/biz"
)

var ErrProfileNotFound = errors.New("profile not found")

type ProfileUsecase struct {
	profileRepo biz.ProfileRepository
}

func NewProfileUsecase(profileRepo biz.ProfileRepository) *ProfileUsecase {
	return &ProfileUsecase{profileRepo: profileRepo}
}

func (uc *ProfileUsecase) ListProfiles(ctx context.Context, platformAccountID string) ([]*v1.ProfileSummary, error) {
	profiles, err := uc.profileRepo.ListByPlatformAccountID(ctx, platformAccountID)
	if err != nil {
		return nil, err
	}

	summaries := make([]*v1.ProfileSummary, 0, len(profiles))
	for _, profile := range profiles {
		summaries = append(summaries, toProfileSummary(profile))
	}

	return summaries, nil
}

func (uc *ProfileUsecase) GetPrimaryProfile(ctx context.Context, platformAccountID string) (*v1.ProfileSummary, error) {
	profiles, err := uc.profileRepo.ListByPlatformAccountID(ctx, platformAccountID)
	if err != nil {
		return nil, err
	}

	if len(profiles) == 0 {
		return nil, nil
	}

	for _, profile := range profiles {
		if profile.IsDefault {
			return toProfileSummary(profile), nil
		}
	}

	return toProfileSummary(profiles[0]), nil
}

func (uc *ProfileUsecase) ConfirmPrimaryProfile(ctx context.Context, platformAccountID string, playerID string) (*v1.ProfileSummary, error) {
	profiles, err := uc.profileRepo.ListByPlatformAccountID(ctx, platformAccountID)
	if err != nil {
		return nil, err
	}

	var selected *biz.Profile
	for _, profile := range profiles {
		if profile.PlayerID == playerID {
			selected = profile
			break
		}
	}
	if selected == nil {
		return nil, ErrProfileNotFound
	}

	for _, profile := range profiles {
		profile.IsDefault = profile.PlayerID == playerID
		if err := uc.profileRepo.Save(ctx, profile); err != nil {
			return nil, err
		}
	}
	return toProfileSummary(selected), nil
}

func toProfileSummary(profile *biz.Profile) *v1.ProfileSummary {
	return &v1.ProfileSummary{
		Id:                profile.ID,
		PlatformAccountId: profile.PlatformAccountID,
		GameBiz:           profile.GameBiz,
		Region:            profile.Region,
		PlayerId:          profile.PlayerID,
		Nickname:          profile.Nickname,
		Level:             int32(profile.Level),
		IsDefault:         profile.IsDefault,
	}
}
