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

func (uc *ProfileUsecase) ListProfilesWithScope(ctx context.Context, guard ScopeGuard, platformAccountID string) ([]*v1.ProfileSummary, error) {
	profiles, err := uc.listScopedProfiles(ctx, guard, platformAccountID)
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

func (uc *ProfileUsecase) GetPrimaryProfileWithScope(ctx context.Context, guard ScopeGuard, platformAccountID string) (*v1.ProfileSummary, error) {
	profiles, err := uc.listScopedProfiles(ctx, guard, platformAccountID)
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

func (uc *ProfileUsecase) ConfirmPrimaryProfileWithScope(ctx context.Context, guard ScopeGuard, platformAccountID string, playerID string) (*v1.ProfileSummary, error) {
	if err := guard.RequirePlatformAccountID(platformAccountID); err != nil {
		return nil, err
	}
	profiles, selected, err := uc.findProfileByPlayerID(ctx, platformAccountID, playerID)
	if err != nil {
		return nil, err
	}
	if err := guard.RequireProfile(selected.BindingID, selected.ID); err != nil {
		return nil, err
	}
	for _, profile := range profiles {
		profile.IsDefault = profile.PlayerID == playerID
		if err := uc.profileRepo.Save(ctx, profile); err != nil {
			return nil, err
		}
	}
	return toProfileSummary(selected), nil
}

func (uc *ProfileUsecase) RequireProfileAccessByPlayerID(ctx context.Context, guard ScopeGuard, platformAccountID string, playerID string) error {
	if err := guard.RequirePlatformAccountID(platformAccountID); err != nil {
		return err
	}
	_, selected, err := uc.findProfileByPlayerID(ctx, platformAccountID, playerID)
	if err != nil {
		return err
	}
	return guard.RequireProfile(selected.BindingID, selected.ID)
}

func (uc *ProfileUsecase) listScopedProfiles(ctx context.Context, guard ScopeGuard, platformAccountID string) ([]*biz.Profile, error) {
	if err := guard.RequirePlatformAccountID(platformAccountID); err != nil {
		return nil, err
	}
	profiles, err := uc.profileRepo.ListByPlatformAccountID(ctx, platformAccountID)
	if err != nil {
		return nil, err
	}
	if guard.ProfileID == 0 {
		return profiles, nil
	}
	filtered := make([]*biz.Profile, 0, len(profiles))
	for _, profile := range profiles {
		if guard.RequireProfile(profile.BindingID, profile.ID) == nil {
			filtered = append(filtered, profile)
		}
	}
	if len(filtered) == 0 {
		return nil, ErrProfileScopeDenied
	}
	return filtered, nil
}

func (uc *ProfileUsecase) findProfileByPlayerID(ctx context.Context, platformAccountID string, playerID string) ([]*biz.Profile, *biz.Profile, error) {
	profiles, err := uc.profileRepo.ListByPlatformAccountID(ctx, platformAccountID)
	if err != nil {
		return nil, nil, err
	}
	for _, profile := range profiles {
		if profile.PlayerID == playerID {
			return profiles, profile, nil
		}
	}
	return profiles, nil, ErrProfileNotFound
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
