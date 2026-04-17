package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "platform-mihomo-service/api/mihomo/v1"
	"platform-mihomo-service/internal/biz"
	platformmihomo "platform-mihomo-service/internal/platform/mihomo"
)

func TestGetCredentialSummaryReturnsDevicesAndProfiles(t *testing.T) {
	uc := newManagementUsecaseForTest(t)
	platformAccountID := bindCredentialForManagementTest(t, uc)

	summary, err := uc.GetCredentialSummary(context.Background(), platformAccountID)
	require.NoError(t, err)
	require.Equal(t, platformAccountID, summary.PlatformAccountID)
	require.Equal(t, v1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE, summary.Status)
	require.NotNil(t, summary.LastValidatedAt)
	require.NotEmpty(t, summary.Profiles)
	require.NotEmpty(t, summary.Devices)
	require.Equal(t, "12345678-1234-1234-1234-123456789abc", summary.Devices[0].DeviceID)
	require.Equal(t, "1008611", summary.Profiles[0].PlayerId)
}

func TestUpdateCredentialReturnsUpdatedSummary(t *testing.T) {
	uc := newManagementUsecaseForTest(t)
	platformAccountID := bindCredentialForManagementTest(t, uc)

	summary, err := uc.UpdateCredential(context.Background(), UpdateCredentialInput{
		PlatformAccountID: platformAccountID,
		BindCredentialInput: BindCredentialInput{
			BindingID:        101,
			CookieBundleJSON: `{"account_id":"10001","cookie_token":"updated"}`,
			DeviceID:         "device-2",
			DeviceFP:         "fp-2",
			DeviceName:       "iPad",
			RegionHint:       "cn_gf01",
		},
	})
	require.NoError(t, err)
	require.Equal(t, platformAccountID, summary.PlatformAccountID)
	require.NotEmpty(t, summary.Devices)
	require.True(t, containsDevice(summary.Devices, "device-2"))
	require.NotEmpty(t, summary.Profiles)
}

func TestUpdateCredentialRejectsPlatformAccountMismatch(t *testing.T) {
	uc := newManagementUsecaseForTest(t)
	platformAccountID := bindCredentialForManagementTest(t, uc)
	uc.bindUC = NewBindUsecase(uc.credentialRepo, uc.deviceRepo, uc.profileRepo, mismatchClient{}, testEncryptionKey)

	_, err := uc.UpdateCredential(context.Background(), UpdateCredentialInput{
		PlatformAccountID: platformAccountID,
		BindCredentialInput: BindCredentialInput{
			BindingID:        101,
			CookieBundleJSON: `{"account_id":"20002","cookie_token":"updated"}`,
			DeviceID:         "device-3",
			DeviceFP:         "fp-3",
			DeviceName:       "iPad",
		},
	})
	require.ErrorIs(t, err, ErrPlatformAccountMismatch)
	require.Nil(t, uc.credentialRepo.byPlatformAccountID["binding_101_20002"])
}

func TestUpdateCredentialRemovesStaleProfiles(t *testing.T) {
	uc := newManagementUsecaseForTest(t)
	platformAccountID := bindCredentialForManagementTest(t, uc)
	uc.bindUC = NewBindUsecase(uc.credentialRepo, uc.deviceRepo, uc.profileRepo, profileSwitchClient{}, testEncryptionKey)

	summary, err := uc.UpdateCredential(context.Background(), UpdateCredentialInput{
		PlatformAccountID: platformAccountID,
		BindCredentialInput: BindCredentialInput{
			BindingID:        101,
			CookieBundleJSON: `{"account_id":"10001","cookie_token":"updated"}`,
			DeviceID:         "device-4",
			DeviceFP:         "fp-4",
			DeviceName:       "switch-device",
		},
	})
	require.NoError(t, err)
	require.Len(t, summary.Profiles, 1)
	require.Equal(t, "2008611", summary.Profiles[0].PlayerId)
}

func TestDeleteCredentialRemovesCredentialArtifactsAndRelations(t *testing.T) {
	uc := newManagementUsecaseForTest(t)
	platformAccountID := bindCredentialForManagementTest(t, uc)
	require.NoError(t, uc.artifactRepo.Put(context.Background(), &biz.Artifact{
		PlatformAccountID: platformAccountID,
		ArtifactType:      authKeyArtifactType,
		ArtifactValue:     "authkey-1",
		ScopeKey:          "1008611",
		ExpiresAt:         time.Now().UTC().Add(5 * time.Minute),
	}))

	err := uc.DeleteCredential(context.Background(), platformAccountID)
	require.NoError(t, err)
	require.Empty(t, uc.credentialRepo.byPlatformAccountID)
	require.Empty(t, uc.deviceRepo.byPlatformAccountID)
	require.Empty(t, uc.profileRepo.byPlatformAccountID)
	require.Empty(t, uc.artifactRepo.artifacts)

	_, err = uc.GetCredentialSummary(context.Background(), platformAccountID)
	require.ErrorIs(t, err, ErrCredentialNotFound)
}

type managementUsecaseTestHarness struct {
	*ManagementUsecase
	credentialRepo *memoryCredentialRepo
	deviceRepo     *memoryDeviceRepo
	profileRepo    *memoryProfileRepo
	artifactRepo   *memoryArtifactRepo
	managementRepo *memoryManagementRepo
}

func newManagementUsecaseForTest(t *testing.T) *managementUsecaseTestHarness {
	t.Helper()
	return newManagementUsecaseForTestWithClient(t, nil)
}

func newManagementUsecaseForTestWithClient(t *testing.T, client platformmihomo.Client) *managementUsecaseTestHarness {
	t.Helper()

	bindHarness := newBindUsecaseForTest()
	if client != nil {
		bindHarness.BindUsecase = NewBindUsecase(bindHarness.credentialRepo, bindHarness.deviceRepo, bindHarness.profileRepo, client, testEncryptionKey)
	}
	artifactRepo := newMemoryArtifactRepo()
	managementRepo := &memoryManagementRepo{
		credentialRepo: bindHarness.credentialRepo,
		deviceRepo:     bindHarness.deviceRepo,
		profileRepo:    bindHarness.profileRepo,
		artifactRepo:   artifactRepo,
	}
	managementUsecase := NewManagementUsecase(
		bindHarness.credentialRepo,
		bindHarness.deviceRepo,
		bindHarness.profileRepo,
		artifactRepo,
		managementRepo,
		bindHarness.BindUsecase,
		bindHarness.profileUsecase,
	)

	return &managementUsecaseTestHarness{
		ManagementUsecase: managementUsecase,
		credentialRepo:    bindHarness.credentialRepo,
		deviceRepo:        bindHarness.deviceRepo,
		profileRepo:       bindHarness.profileRepo,
		artifactRepo:      artifactRepo,
		managementRepo:    managementRepo,
	}
}

func bindCredentialForManagementTest(t *testing.T, uc *managementUsecaseTestHarness) string {
	t.Helper()

	resp, err := uc.bindUC.BindCredential(context.Background(), BindCredentialInput{
		BindingID:        101,
		CookieBundleJSON: `{"account_id":"10001","cookie_token":"abc"}`,
		DeviceID:         "12345678-1234-1234-1234-123456789abc",
		DeviceFP:         "abcdefghijklmn",
		DeviceName:       "iPhone",
	})
	require.NoError(t, err)
	return resp.PlatformAccountID
}

type memoryManagementRepo struct {
	credentialRepo *memoryCredentialRepo
	deviceRepo     *memoryDeviceRepo
	profileRepo    *memoryProfileRepo
	artifactRepo   *memoryArtifactRepo
}

func (r *memoryManagementRepo) DeleteCredentialGraph(_ context.Context, platformAccountID string) error {
	requireDeleteArtifacts(r.artifactRepo, platformAccountID)
	delete(r.credentialRepo.byPlatformAccountID, platformAccountID)
	delete(r.deviceRepo.byPlatformAccountID, platformAccountID)
	delete(r.profileRepo.byPlatformAccountID, platformAccountID)
	return nil
}

func requireDeleteArtifacts(repo *memoryArtifactRepo, platformAccountID string) {
	for key, artifact := range repo.artifacts {
		if artifact.PlatformAccountID == platformAccountID {
			delete(repo.artifacts, key)
		}
	}
}

func containsDevice(devices []*biz.Device, deviceID string) bool {
	for _, device := range devices {
		if device.DeviceID == deviceID {
			return true
		}
	}
	return false
}

type mismatchClient struct{}

func (mismatchClient) ValidateAndDiscover(_ context.Context, _ string, _ string) (string, string, []platformmihomo.DiscoveredProfile, error) {
	return "20002", "cn_gf01", []platformmihomo.DiscoveredProfile{{GameBiz: "hk4e_cn", Region: "cn_gf01", PlayerID: "2008611", Nickname: "Mismatch", Level: 60}}, nil
}

func (mismatchClient) IssueAuthKey(_ context.Context, _ string, _ string) (string, int64, error) {
	return "", 0, nil
}

type profileSwitchClient struct{}

func (profileSwitchClient) ValidateAndDiscover(_ context.Context, _ string, _ string) (string, string, []platformmihomo.DiscoveredProfile, error) {
	return "10001", "cn_gf01", []platformmihomo.DiscoveredProfile{{GameBiz: "hk4e_cn", Region: "cn_gf01", PlayerID: "2008611", Nickname: "Switched", Level: 50}}, nil
}

func (profileSwitchClient) IssueAuthKey(_ context.Context, _ string, _ string) (string, int64, error) {
	return "", 0, nil
}
