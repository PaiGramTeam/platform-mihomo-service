package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	v1 "platform-mihomo-service/api/mihomo/v1"
	"platform-mihomo-service/internal/biz"
	internalcrypto "platform-mihomo-service/internal/crypto"
	platformmihomo "platform-mihomo-service/internal/platform/mihomo"
)

func TestBindCredentialPersistsCredentialDeviceAndProfiles(t *testing.T) {
	uc := newBindUsecaseForTest()

	resp, err := uc.BindCredential(context.Background(), BindCredentialInput{
		BindingID:        101,
		CookieBundleJSON: `{"account_id":"10001","cookie_token":"abc"}`,
		DeviceID:         "12345678-1234-1234-1234-123456789abc",
		DeviceFP:         "abcdefghijklmn",
		DeviceName:       "iPhone",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(101), resp.BindingID)
	require.Equal(t, "binding_101_10001", resp.PlatformAccountID)
	require.Equal(t, v1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE, resp.Status)
	require.Len(t, resp.Profiles, 1)
	require.Equal(t, "1008611", resp.Profiles[0].PlayerId)
	require.True(t, resp.Profiles[0].IsDefault)

	credential := uc.credentialRepo.byPlatformAccountID[resp.PlatformAccountID]
	require.NotNil(t, credential)
	require.Equal(t, uint64(101), credential.BindingID)
	require.Equal(t, "mihomo", credential.Platform)
	require.Equal(t, "10001", credential.AccountID)
	require.Equal(t, "cn_gf01", credential.Region)
	require.Equal(t, "v1", credential.CredentialVersion)
	require.Equal(t, "active", credential.Status)
	require.NotEmpty(t, credential.CredentialBlob)
	require.NotNil(t, credential.LastValidatedAt)

	decrypted, err := internalcrypto.DecryptString(testEncryptionKey, credential.CredentialBlob)
	require.NoError(t, err)
	require.Equal(t, `{"account_id":"10001","cookie_token":"abc"}`, decrypted)

	devices := uc.deviceRepo.byPlatformAccountID[resp.PlatformAccountID]
	require.Len(t, devices, 1)
	require.Equal(t, "12345678-1234-1234-1234-123456789abc", devices[0].DeviceID)
	require.Equal(t, "abcdefghijklmn", devices[0].DeviceFP)
	require.NotNil(t, devices[0].DeviceName)
	require.Equal(t, "iPhone", *devices[0].DeviceName)
	require.True(t, devices[0].IsValid)
	require.NotNil(t, devices[0].LastSeenAt)

	persistedProfiles := uc.profileRepo.byPlatformAccountID[resp.PlatformAccountID]
	require.Len(t, persistedProfiles, 1)
	require.Equal(t, uint64(101), persistedProfiles[0].BindingID)
	require.Equal(t, "Traveler", persistedProfiles[0].Nickname)
	require.Equal(t, 60, persistedProfiles[0].Level)
	require.True(t, persistedProfiles[0].IsDefault)
	require.False(t, persistedProfiles[0].DiscoveredAt.IsZero())

	listed, err := uc.profileUsecase.ListProfiles(context.Background(), resp.PlatformAccountID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, "1008611", listed[0].PlayerId)

	primary, err := uc.profileUsecase.GetPrimaryProfile(context.Background(), resp.PlatformAccountID)
	require.NoError(t, err)
	require.NotNil(t, primary)
	require.Equal(t, "1008611", primary.PlayerId)
}

func TestBindCredentialPersistsBindingIDOnCredentialAndProfiles(t *testing.T) {
	uc := newBindUsecaseForTest()

	out, err := uc.BindCredential(context.Background(), BindCredentialInput{
		BindingID:        42,
		CookieBundleJSON: `{"account_id":"10001","cookie_token":"abc"}`,
		DeviceID:         "12345678-1234-1234-1234-123456789abc",
		DeviceFP:         "abcdefghijklmn",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(42), out.BindingID)

	credential, err := uc.credentialRepo.GetByBindingID(context.Background(), 42)
	require.NoError(t, err)
	require.NotNil(t, credential)
	require.Equal(t, uint64(42), credential.BindingID)
	require.Equal(t, out.PlatformAccountID, credential.PlatformAccountID)

	profiles, err := uc.profileRepo.ListByBindingID(context.Background(), 42)
	require.NoError(t, err)
	require.NotEmpty(t, profiles)
	require.Equal(t, uint64(42), profiles[0].BindingID)
	require.Equal(t, out.PlatformAccountID, profiles[0].PlatformAccountID)
}

func TestBindCredentialRejectsMissingBindingID(t *testing.T) {
	uc := newBindUsecaseForTest()

	_, err := uc.BindCredential(context.Background(), BindCredentialInput{
		CookieBundleJSON: `{"account_id":"10001","cookie_token":"abc"}`,
		DeviceID:         "12345678-1234-1234-1234-123456789abc",
		DeviceFP:         "abcdefghijklmn",
	})
	require.Error(t, err)
}

func TestFormatPlatformAccountIDUsesBindingPrefix(t *testing.T) {
	require.Equal(t, "binding_42_10001", FormatPlatformAccountID(42, "10001"))
}

func TestBindingIDFromPlatformAccountIDAcceptsLegacyPrefix(t *testing.T) {
	bindingID, err := BindingIDFromPlatformAccountID("hoyo_ref_42_10001")
	require.NoError(t, err)
	require.Equal(t, uint64(42), bindingID)
}

func TestBindCredentialRollsBackWhenProfileSaveFails(t *testing.T) {
	uc := newBindUsecaseForTest()
	uc.profileRepo.failSave = true

	_, err := uc.BindCredential(context.Background(), BindCredentialInput{
		BindingID:        101,
		CookieBundleJSON: `{"account_id":"10001","cookie_token":"abc"}`,
		DeviceID:         "12345678-1234-1234-1234-123456789abc",
		DeviceFP:         "abcdefghijklmn",
		DeviceName:       "iPhone",
	})
	require.Error(t, err)
	require.Empty(t, uc.credentialRepo.byPlatformAccountID)
	require.Empty(t, uc.deviceRepo.byPlatformAccountID)
	require.Empty(t, uc.profileRepo.byPlatformAccountID)
}

var testEncryptionKey = []byte("0123456789abcdef0123456789abcdef")

type bindUsecaseTestHarness struct {
	*BindUsecase
	credentialRepo *memoryCredentialRepo
	deviceRepo     *memoryDeviceRepo
	profileRepo    *memoryProfileRepo
	profileUsecase *ProfileUsecase
}

func newBindUsecaseForTest() *bindUsecaseTestHarness {
	credentialRepo := newMemoryCredentialRepo()
	deviceRepo := newMemoryDeviceRepo()
	profileRepo := newMemoryProfileRepo()

	bindUsecase := NewBindUsecase(credentialRepo, deviceRepo, profileRepo, platformmihomo.StubClient{}, testEncryptionKey)

	return &bindUsecaseTestHarness{
		BindUsecase:    bindUsecase,
		credentialRepo: credentialRepo,
		deviceRepo:     deviceRepo,
		profileRepo:    profileRepo,
		profileUsecase: NewProfileUsecase(profileRepo),
	}
}

type memoryCredentialRepo struct {
	byPlatformAccountID map[string]*biz.Credential
	byBindingID         map[uint64]*biz.Credential
}

func newMemoryCredentialRepo() *memoryCredentialRepo {
	return &memoryCredentialRepo{
		byPlatformAccountID: make(map[string]*biz.Credential),
		byBindingID:         make(map[uint64]*biz.Credential),
	}
}

func (r *memoryCredentialRepo) Save(_ context.Context, credential *biz.Credential) error {
	clone := *credential
	r.byPlatformAccountID[credential.PlatformAccountID] = &clone
	r.byBindingID[credential.BindingID] = &clone
	return nil
}

func (r *memoryCredentialRepo) GetByPlatformAccountID(_ context.Context, platformAccountID string) (*biz.Credential, error) {
	credential := r.byPlatformAccountID[platformAccountID]
	if credential == nil {
		return nil, nil
	}
	clone := *credential
	return &clone, nil
}

func (r *memoryCredentialRepo) GetByBindingID(_ context.Context, bindingID uint64) (*biz.Credential, error) {
	credential := r.byBindingID[bindingID]
	if credential == nil {
		return nil, nil
	}
	clone := *credential
	return &clone, nil
}

func (r *memoryCredentialRepo) DeleteByPlatformAccountID(_ context.Context, platformAccountID string) error {
	if credential := r.byPlatformAccountID[platformAccountID]; credential != nil {
		delete(r.byBindingID, credential.BindingID)
	}
	delete(r.byPlatformAccountID, platformAccountID)
	return nil
}

type memoryDeviceRepo struct {
	byPlatformAccountID map[string][]*biz.Device
}

func newMemoryDeviceRepo() *memoryDeviceRepo {
	return &memoryDeviceRepo{byPlatformAccountID: make(map[string][]*biz.Device)}
}

func (r *memoryDeviceRepo) Save(_ context.Context, device *biz.Device) error {
	clone := *device
	current := r.byPlatformAccountID[device.PlatformAccountID]
	for index, existing := range current {
		if existing.DeviceID == device.DeviceID {
			current[index] = &clone
			r.byPlatformAccountID[device.PlatformAccountID] = current
			return nil
		}
	}
	r.byPlatformAccountID[device.PlatformAccountID] = append(current, &clone)
	return nil
}

func (r *memoryDeviceRepo) ListByPlatformAccountID(_ context.Context, platformAccountID string) ([]*biz.Device, error) {
	devices := r.byPlatformAccountID[platformAccountID]
	result := make([]*biz.Device, 0, len(devices))
	for _, device := range devices {
		clone := *device
		result = append(result, &clone)
	}
	return result, nil
}

func (r *memoryDeviceRepo) DeleteByPlatformAccountID(_ context.Context, platformAccountID string) error {
	delete(r.byPlatformAccountID, platformAccountID)
	return nil
}

type memoryProfileRepo struct {
	byPlatformAccountID map[string][]*biz.Profile
	byBindingID         map[uint64][]*biz.Profile
	failSave            bool
}

func newMemoryProfileRepo() *memoryProfileRepo {
	return &memoryProfileRepo{
		byPlatformAccountID: make(map[string][]*biz.Profile),
		byBindingID:         make(map[uint64][]*biz.Profile),
	}
}

func (r *memoryProfileRepo) Save(_ context.Context, profile *biz.Profile) error {
	if r.failSave {
		return errors.New("save profile failed")
	}
	clone := *profile
	current := r.byPlatformAccountID[profile.PlatformAccountID]
	byBinding := r.byBindingID[profile.BindingID]
	for index, existing := range current {
		if existing.PlayerID == profile.PlayerID && existing.Region == profile.Region {
			current[index] = &clone
			r.byPlatformAccountID[profile.PlatformAccountID] = current
			for bindingIndex, bindingProfile := range byBinding {
				if bindingProfile.PlayerID == profile.PlayerID && bindingProfile.Region == profile.Region {
					byBinding[bindingIndex] = &clone
					r.byBindingID[profile.BindingID] = byBinding
					return nil
				}
			}
			return nil
		}
	}
	r.byPlatformAccountID[profile.PlatformAccountID] = append(current, &clone)
	r.byBindingID[profile.BindingID] = append(byBinding, &clone)
	return nil
}

func (r *memoryProfileRepo) ListByPlatformAccountID(_ context.Context, platformAccountID string) ([]*biz.Profile, error) {
	profiles := r.byPlatformAccountID[platformAccountID]
	result := make([]*biz.Profile, 0, len(profiles))
	for _, profile := range profiles {
		clone := *profile
		result = append(result, &clone)
	}
	return result, nil
}

func (r *memoryProfileRepo) ListByBindingID(_ context.Context, bindingID uint64) ([]*biz.Profile, error) {
	profiles := r.byBindingID[bindingID]
	result := make([]*biz.Profile, 0, len(profiles))
	for _, profile := range profiles {
		clone := *profile
		result = append(result, &clone)
	}
	return result, nil
}

func (r *memoryProfileRepo) DeleteByPlatformAccountID(_ context.Context, platformAccountID string) error {
	if profiles := r.byPlatformAccountID[platformAccountID]; len(profiles) > 0 {
		delete(r.byBindingID, profiles[0].BindingID)
	}
	delete(r.byPlatformAccountID, platformAccountID)
	return nil
}

func (r *memoryProfileRepo) DeleteMissingByPlatformAccountID(_ context.Context, platformAccountID string, keep []biz.ProfileIdentity) error {
	profiles := r.byPlatformAccountID[platformAccountID]
	keepSet := make(map[string]struct{}, len(keep))
	for _, identity := range keep {
		keepSet[identity.PlayerID+":"+identity.Region] = struct{}{}
	}
	filtered := make([]*biz.Profile, 0, len(profiles))
	for _, profile := range profiles {
		if _, ok := keepSet[profile.PlayerID+":"+profile.Region]; ok {
			filtered = append(filtered, profile)
		}
	}
	r.byPlatformAccountID[platformAccountID] = filtered
	if len(filtered) == 0 {
		if len(profiles) > 0 {
			delete(r.byBindingID, profiles[0].BindingID)
		}
		return nil
	}
	r.byBindingID[filtered[0].BindingID] = filtered
	return nil
}

var _ biz.CredentialRepository = (*memoryCredentialRepo)(nil)
var _ biz.DeviceRepository = (*memoryDeviceRepo)(nil)
var _ biz.ProfileRepository = (*memoryProfileRepo)(nil)
