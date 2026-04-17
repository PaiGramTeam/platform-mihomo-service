//go:build integration

package integration

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	v1 "platform-mihomo-service/api/mihomo/v1"
	"platform-mihomo-service/internal/data"
	platformmihomo "platform-mihomo-service/internal/platform/mihomo"
	"platform-mihomo-service/internal/service"
	"platform-mihomo-service/internal/usecase"
)

const (
	integrationTicketIssuer   = "paigram-account-center"
	integrationTicketAudience = "platform-mihomo-service"
	bufConnAddress            = "bufnet"
)

var (
	integrationEncryptionKey = []byte("0123456789abcdef0123456789abcdef")
	integrationSigningKey    = []byte("abcdef0123456789abcdef0123456789")
)

func TestBindThenGetAuthKeyFlow(t *testing.T) {
	stack := newIntegrationStack(t)
	t.Cleanup(stack.cleanup)
	client := newMihomoClientForTest(t, stack)

	bindResp, err := client.BindCredential(testTicket(t), validBindRequest())
	require.NoError(t, err)
	require.NotEmpty(t, bindResp.PlatformAccountId)
	require.NotEmpty(t, bindResp.Profiles)

	authResp, err := client.GetAuthKey(testTicketForAccount(t, bindResp.PlatformAccountId, "mihomo.authkey.issue"), bindResp.PlatformAccountId, bindResp.Profiles[0].PlayerId)
	require.NoError(t, err)
	require.NotEmpty(t, authResp.Authkey)
}

func TestUpdateThenDeleteCredentialFlow(t *testing.T) {
	stack := newIntegrationStack(t)
	t.Cleanup(stack.cleanup)
	client := newMihomoClientForTest(t, stack)

	bindResp, err := client.BindCredential(testTicket(t), validBindRequest())
	require.NoError(t, err)

	summaryResp, err := client.GetCredentialSummary(testTicketForAccount(t, bindResp.PlatformAccountId, "mihomo.credential.read_meta"), bindResp.PlatformAccountId)
	require.NoError(t, err)
	require.NotNil(t, summaryResp.Summary)
	require.Equal(t, bindResp.PlatformAccountId, summaryResp.Summary.PlatformAccountId)
	require.Len(t, summaryResp.Summary.Devices, 1)
	require.Len(t, summaryResp.Summary.Profiles, 1)

	updateResp, err := client.UpdateCredential(testTicketForAccount(t, bindResp.PlatformAccountId, "mihomo.credential.update"), &v1.UpdateCredentialRequest{
		PlatformAccountId: bindResp.PlatformAccountId,
		CookieBundleJson:  `{"account_id":"10001","cookie_token":"updated"}`,
		Device: &v1.DeviceInfo{
			DeviceId:   "87654321-4321-4321-4321-cba987654321",
			DeviceFp:   "zyxwvutsrqponm",
			DeviceName: "updated-device",
		},
		RegionHint: "cn_gf01",
	})
	require.NoError(t, err)
	require.NotNil(t, updateResp.Summary)
	require.Len(t, updateResp.Summary.Devices, 2)
	require.Equal(t, "87654321-4321-4321-4321-cba987654321", updateResp.Summary.Devices[1].DeviceId)

	deleteResp, err := client.DeleteCredential(testTicketForAccount(t, bindResp.PlatformAccountId, "mihomo.credential.delete"), bindResp.PlatformAccountId)
	require.NoError(t, err)
	require.True(t, deleteResp.Success)

	_, err = client.GetCredentialSummary(testTicketForAccount(t, bindResp.PlatformAccountId, "mihomo.credential.read_meta"), bindResp.PlatformAccountId)
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

type testMihomoClient struct {
	client v1.MihomoAccountServiceClient
	t      *testing.T
}

func newMihomoClientForTest(t *testing.T, stack *integrationStack) *testMihomoClient {
	t.Helper()

	requireMigrationsApplied(t, stack.SQLDB)

	db, err := gorm.Open(mysql.Open(stack.DatabaseCfg.Source), &gorm.Config{})
	require.NoError(t, err)

	credentialRepo := data.NewCredentialRepo(db)
	deviceRepo := data.NewDeviceRepo(db)
	profileRepo := data.NewProfileRepo(db)
	artifactRepo := data.NewArtifactRepo(db, nil, "integration:")
	managementRepo := data.NewManagementRepo(db, nil, "integration:")
	hoyoClient := platformmihomo.StubClient{}
	bindUC := usecase.NewBindUsecase(credentialRepo, deviceRepo, profileRepo, hoyoClient, integrationEncryptionKey)
	profileUC := usecase.NewProfileUsecase(profileRepo)

	svc := service.NewMihomoAccountService(
		data.NewTicketVerifier(integrationTicketIssuer, integrationSigningKey),
		bindUC,
		usecase.NewStatusUsecase(credentialRepo, hoyoClient, integrationEncryptionKey),
		profileUC,
		usecase.NewAuthkeyUsecase(credentialRepo, artifactRepo, hoyoClient, integrationEncryptionKey),
		usecase.NewManagementUsecase(credentialRepo, deviceRepo, profileRepo, artifactRepo, managementRepo, bindUC, profileUC),
	)

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	v1.RegisterMihomoAccountServiceServer(server, svc)

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	conn, err := grpc.DialContext(ctx, bufConnAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return &testMihomoClient{client: v1.NewMihomoAccountServiceClient(conn), t: t}
}

func (c *testMihomoClient) BindCredential(serviceTicket string, request *v1.BindCredentialRequest) (*v1.BindCredentialResponse, error) {
	c.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request.ServiceTicket = serviceTicket
	return c.client.BindCredential(ctx, request)
}

func (c *testMihomoClient) GetAuthKey(serviceTicket string, platformAccountID string, playerID string) (*v1.GetAuthKeyResponse, error) {
	c.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.GetAuthKey(ctx, &v1.GetAuthKeyRequest{
		ServiceTicket:     serviceTicket,
		PlatformAccountId: platformAccountID,
		PlayerId:          playerID,
	})
}

func (c *testMihomoClient) GetCredentialSummary(serviceTicket string, platformAccountID string) (*v1.GetCredentialSummaryResponse, error) {
	c.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.GetCredentialSummary(ctx, &v1.GetCredentialSummaryRequest{
		ServiceTicket:     serviceTicket,
		PlatformAccountId: platformAccountID,
	})
}

func (c *testMihomoClient) UpdateCredential(serviceTicket string, request *v1.UpdateCredentialRequest) (*v1.UpdateCredentialResponse, error) {
	c.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request.ServiceTicket = serviceTicket
	return c.client.UpdateCredential(ctx, request)
}

func (c *testMihomoClient) DeleteCredential(serviceTicket string, platformAccountID string) (*v1.DeleteCredentialResponse, error) {
	c.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.DeleteCredential(ctx, &v1.DeleteCredentialRequest{
		ServiceTicket:     serviceTicket,
		PlatformAccountId: platformAccountID,
	})
}

func testTicket(t *testing.T) string { return testTicketForAccount(t, "", "mihomo.credential.bind") }

func testTicketForAccount(t *testing.T, platformAccountID string, scopes ...string) string {
	t.Helper()

	claims := jwt.MapClaims{
		"iss":                     integrationTicketIssuer,
		"aud":                     []string{integrationTicketAudience},
		"actor_type":              "bot",
		"actor_id":                "bot-paigram",
		"owner_user_id":           float64(1),
		"binding_id":              float64(101),
		"bot_id":                  "bot-paigram",
		"platform":                "mihomo",
		"user_id":                 float64(1),
		"platform_service_key":    integrationTicketAudience,
		"platform_account_ref_id": float64(101),
		"exp":                     time.Now().Add(time.Minute).Unix(),
	}
	if platformAccountID != "" {
		claims["platform_account_id"] = platformAccountID
	}
	if len(scopes) > 0 {
		claims["scopes"] = scopes
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(integrationSigningKey)
	require.NoError(t, err)

	return signed
}

func validBindRequest() *v1.BindCredentialRequest {
	return &v1.BindCredentialRequest{
		CookieBundleJson: `{"account_id":"10001","cookie_token":"abc"}`,
		Device: &v1.DeviceInfo{
			DeviceId:   "12345678-1234-1234-1234-123456789abc",
			DeviceFp:   "abcdefghijklmn",
			DeviceName: "integration-device",
		},
	}
}
