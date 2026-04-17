package service

import (
	"context"
	"testing"
	"time"

	mihomov1 "github.com/PaiGramTeam/proto-contracts/mihomo/v1"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"platform-mihomo-service/internal/data"
	platformmihomo "platform-mihomo-service/internal/platform/mihomo"
	"platform-mihomo-service/internal/usecase"
)

type mihomoCredentialServiceTestHarness struct {
	service *MihomoCredentialService
	bindUC  *usecase.BindUsecase
}

func TestMihomoCredentialServiceGetCredentialSummary(t *testing.T) {
	harness := newMihomoCredentialServiceForTest(t)

	bindResp, err := harness.bindUC.BindCredential(context.Background(), usecase.BindCredentialInput{
		BindingID:        101,
		CookieBundleJSON: `{"account_id":"10001","cookie_token":"abc"}`,
		DeviceID:         "12345678-1234-1234-1234-123456789abc",
		DeviceFP:         "abcdefghijklmn",
		DeviceName:       "iPhone",
	})
	require.NoError(t, err)

	resp, err := harness.service.GetCredentialSummary(context.Background(), &mihomov1.GetCredentialSummaryRequest{
		ServiceTicket:     signedMihomoSummaryTicket(t, bindResp.PlatformAccountID, "mihomo.credential.read_meta"),
		PlatformAccountId: bindResp.PlatformAccountID,
	})
	require.NoError(t, err)
	require.Equal(t, bindResp.PlatformAccountID, resp.PlatformAccountId)
	require.NotEmpty(t, resp.Profiles)
	require.NotEmpty(t, resp.Devices)
}

func TestMihomoCredentialServiceRejectsTicketMissingPlatformAccountID(t *testing.T) {
	harness := newMihomoCredentialServiceForTest(t)

	_, err := harness.service.GetCredentialSummary(context.Background(), &mihomov1.GetCredentialSummaryRequest{
		ServiceTicket:     signedMihomoSummaryTicketWithoutAccount(t),
		PlatformAccountId: "binding_101_10001",
	})
	require.Error(t, err)
}

func TestMihomoCredentialServiceRejectsMissingSummaryScope(t *testing.T) {
	harness := newMihomoCredentialServiceForTest(t)

	bindResp, err := harness.bindUC.BindCredential(context.Background(), usecase.BindCredentialInput{
		BindingID:        101,
		CookieBundleJSON: `{"account_id":"10001","cookie_token":"abc"}`,
		DeviceID:         "12345678-1234-1234-1234-123456789abc",
		DeviceFP:         "abcdefghijklmn",
		DeviceName:       "iPhone",
	})
	require.NoError(t, err)

	_, err = harness.service.GetCredentialSummary(context.Background(), &mihomov1.GetCredentialSummaryRequest{
		ServiceTicket:     signedMihomoSummaryTicket(t, bindResp.PlatformAccountID),
		PlatformAccountId: bindResp.PlatformAccountID,
	})
	require.Error(t, err)
}

func signedMihomoSummaryTicket(t *testing.T, platformAccountID string, scopes ...string) string {
	t.Helper()
	return signedServiceTicketForAccount(t, platformAccountID, scopes...)
}

func signedMihomoSummaryTicketWithoutAccount(t *testing.T) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss":                  serviceTestIssuer,
		"aud":                  []string{serviceTestAudience},
		"actor_type":           "bot",
		"actor_id":             "bot-paigram",
		"owner_user_id":        float64(1),
		"binding_id":           float64(101),
		"bot_id":               "bot-paigram",
		"platform":             "mihomo",
		"platform_service_key": serviceTestAudience,
		"scopes":               []string{"mihomo.credential.read_meta"},
		"exp":                  time.Now().Add(time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(serviceTestSigningKey)
	require.NoError(t, err)
	return signed
}

func newMihomoCredentialServiceForTest(t *testing.T) *mihomoCredentialServiceTestHarness {
	t.Helper()

	credentialRepo := newMemoryCredentialRepo()
	deviceRepo := newMemoryDeviceRepo()
	profileRepo := newMemoryProfileRepo()
	artifactRepo := newMemoryArtifactRepo()
	client := platformmihomo.StubClient{}

	bindUC := usecase.NewBindUsecase(credentialRepo, deviceRepo, profileRepo, client, serviceTestSigningKey)
	profileUC := usecase.NewProfileUsecase(profileRepo)
	managementUC := usecase.NewManagementUsecase(
		credentialRepo,
		deviceRepo,
		profileRepo,
		artifactRepo,
		newMemoryManagementRepo(credentialRepo, deviceRepo, profileRepo, artifactRepo),
		bindUC,
		profileUC,
	)

	return &mihomoCredentialServiceTestHarness{
		service: NewMihomoCredentialService(
			data.NewTicketVerifier(serviceTestIssuer, serviceTestSigningKey),
			managementUC,
		),
		bindUC: bindUC,
	}
}
