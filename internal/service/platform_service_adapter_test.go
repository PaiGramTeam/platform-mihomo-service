package service

import (
	"context"
	"net"
	"testing"

	platformv1 "github.com/PaiGramTeam/proto-contracts/platform/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"platform-mihomo-service/internal/data"
	platformmihomo "platform-mihomo-service/internal/platform/mihomo"
	"platform-mihomo-service/internal/usecase"
)

func TestGenericPlatformServiceGetCredentialSummary(t *testing.T) {
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
	adapter := NewGenericPlatformService(
		data.NewTicketVerifier(serviceTestIssuer, serviceTestSigningKey),
		managementUC,
	)

	bindResp, err := bindUC.BindCredential(context.Background(), usecase.BindCredentialInput{
		PlatformAccountRefID: 101,
		CookieBundleJSON:     `{"account_id":"10001","cookie_token":"abc"}`,
		DeviceID:             "12345678-1234-1234-1234-123456789abc",
		DeviceFP:             "abcdefghijklmn",
		DeviceName:           "iPhone",
	})
	require.NoError(t, err)

	resp, err := adapter.GetCredentialSummary(context.Background(), &platformv1.GetCredentialSummaryRequest{
		ServiceTicket:     signedMihomoSummaryTicket(t, bindResp.PlatformAccountID, "mihomo.credential.read_meta"),
		PlatformAccountId: bindResp.PlatformAccountID,
	})
	require.NoError(t, err)
	require.Equal(t, bindResp.PlatformAccountID, resp.PlatformAccountId)
	require.NotEmpty(t, resp.Profiles)
	require.NotEmpty(t, resp.Devices)
}

func TestGenericPlatformServiceDescribePlatform(t *testing.T) {
	adapter := NewGenericPlatformService(
		data.NewTicketVerifier(serviceTestIssuer, serviceTestSigningKey),
		usecase.NewManagementUsecase(newMemoryCredentialRepo(), newMemoryDeviceRepo(), newMemoryProfileRepo(), newMemoryArtifactRepo(), newMemoryManagementRepo(newMemoryCredentialRepo(), newMemoryDeviceRepo(), newMemoryProfileRepo(), newMemoryArtifactRepo()), usecase.NewBindUsecase(newMemoryCredentialRepo(), newMemoryDeviceRepo(), newMemoryProfileRepo(), platformmihomo.StubClient{}, serviceTestSigningKey), usecase.NewProfileUsecase(newMemoryProfileRepo())),
	)

	resp, err := adapter.DescribePlatform(context.Background(), &platformv1.DescribePlatformRequest{})
	require.NoError(t, err)
	require.Equal(t, "mihomo", resp.PlatformKey)
	require.Equal(t, "Mihomo", resp.DisplayName)
	require.Equal(t, serviceTicketAudience, resp.ServiceAudience)
	require.Equal(t, []string{"summary"}, resp.SupportedActions)
	require.NotNil(t, resp.CredentialSchema)
	require.NotEmpty(t, resp.CredentialSchema.Fields)
}

func TestGenericPlatformServiceRegisteredOnGRPCServer(t *testing.T) {
	adapter := NewGenericPlatformService(
		data.NewTicketVerifier(serviceTestIssuer, serviceTestSigningKey),
		usecase.NewManagementUsecase(newMemoryCredentialRepo(), newMemoryDeviceRepo(), newMemoryProfileRepo(), newMemoryArtifactRepo(), newMemoryManagementRepo(newMemoryCredentialRepo(), newMemoryDeviceRepo(), newMemoryProfileRepo(), newMemoryArtifactRepo()), usecase.NewBindUsecase(newMemoryCredentialRepo(), newMemoryDeviceRepo(), newMemoryProfileRepo(), platformmihomo.StubClient{}, serviceTestSigningKey), usecase.NewProfileUsecase(newMemoryProfileRepo())),
	)

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	platformv1.RegisterPlatformServiceServer(server, adapter)
	go server.Serve(listener)
	defer server.Stop()

	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	resp, err := platformv1.NewPlatformServiceClient(conn).DescribePlatform(context.Background(), &platformv1.DescribePlatformRequest{})
	require.NoError(t, err)
	require.Equal(t, "mihomo", resp.PlatformKey)
}
