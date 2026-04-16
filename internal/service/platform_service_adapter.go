package service

import (
	"context"

	platformv1 "github.com/PaiGramTeam/proto-contracts/platform/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	mihomoapiv1 "platform-mihomo-service/api/mihomo/v1"
	"platform-mihomo-service/internal/data"
	"platform-mihomo-service/internal/usecase"
)

type GenericPlatformService struct {
	platformv1.UnimplementedPlatformServiceServer

	ticketVerifier *data.TicketVerifier
	managementUC   *usecase.ManagementUsecase
}

func NewGenericPlatformService(ticketVerifier *data.TicketVerifier, managementUC *usecase.ManagementUsecase) *GenericPlatformService {
	return &GenericPlatformService{ticketVerifier: ticketVerifier, managementUC: managementUC}
}

func (s *GenericPlatformService) DescribePlatform(context.Context, *platformv1.DescribePlatformRequest) (*platformv1.DescribePlatformResponse, error) {
	credentialSchema, err := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"cookie_bundle": map[string]any{"type": "string"},
			"device_id":     map[string]any{"type": "string"},
			"device_fp":     map[string]any{"type": "string"},
			"device_name":   map[string]any{"type": "string"},
		},
		"required": []any{"cookie_bundle", "device_id", "device_fp"},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to build credential schema")
	}

	return &platformv1.DescribePlatformResponse{
		PlatformKey:      "mihomo",
		DisplayName:      "Mihomo",
		ServiceAudience:  serviceTicketAudience,
		SupportedActions: []string{"summary"},
		CredentialSchema: credentialSchema,
		Version:          "v1",
	}, nil
}

func (s *GenericPlatformService) GetCredentialSummary(ctx context.Context, req *platformv1.GetCredentialSummaryRequest) (*platformv1.GetCredentialSummaryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	claims, err := s.ticketVerifier.Verify(req.GetServiceTicket(), serviceTicketAudience)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid service ticket")
	}
	if err := validateScopedPlatformAccountID(claims, req.GetPlatformAccountId()); err != nil {
		return nil, err
	}
	if err := requireScopes(claims, "mihomo.credential.read_meta"); err != nil {
		return nil, err
	}

	output, err := s.managementUC.GetCredentialSummary(ctx, req.GetPlatformAccountId())
	if err != nil {
		return nil, mapUsecaseError(err)
	}

	return toGenericCredentialSummary(output), nil
}

func toGenericCredentialSummary(output *usecase.CredentialSummaryOutput) *platformv1.GetCredentialSummaryResponse {
	profiles := make([]*platformv1.ProfileSummary, 0, len(output.Profiles))
	for _, profile := range output.Profiles {
		profiles = append(profiles, &platformv1.ProfileSummary{
			Id:                profile.Id,
			PlatformAccountId: profile.PlatformAccountId,
			GameBiz:           profile.GameBiz,
			Region:            profile.Region,
			PlayerId:          profile.PlayerId,
			Nickname:          profile.Nickname,
			Level:             profile.Level,
			IsDefault:         profile.IsDefault,
		})
	}

	devices := make([]*platformv1.DeviceSummary, 0, len(output.Devices))
	for _, device := range output.Devices {
		devices = append(devices, &platformv1.DeviceSummary{
			DeviceId:   device.DeviceID,
			DeviceFp:   device.DeviceFP,
			DeviceName: derefString(device.DeviceName),
			IsValid:    device.IsValid,
			LastSeenAt: toTimestamp(device.LastSeenAt),
		})
	}

	return &platformv1.GetCredentialSummaryResponse{
		PlatformAccountId: output.PlatformAccountID,
		Status:            toGenericCredentialStatus(output.Status),
		LastValidatedAt:   toTimestamp(output.LastValidatedAt),
		LastRefreshedAt:   toTimestamp(output.LastRefreshedAt),
		Devices:           devices,
		Profiles:          profiles,
	}
}

func toGenericCredentialStatus(statusValue mihomoapiv1.CredentialStatus) platformv1.CredentialStatus {
	switch statusValue {
	case mihomoapiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE:
		return platformv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE
	case mihomoapiv1.CredentialStatus_CREDENTIAL_STATUS_EXPIRED:
		return platformv1.CredentialStatus_CREDENTIAL_STATUS_EXPIRED
	case mihomoapiv1.CredentialStatus_CREDENTIAL_STATUS_INVALID:
		return platformv1.CredentialStatus_CREDENTIAL_STATUS_INVALID
	case mihomoapiv1.CredentialStatus_CREDENTIAL_STATUS_CHALLENGE_REQUIRED:
		return platformv1.CredentialStatus_CREDENTIAL_STATUS_CHALLENGE_REQUIRED
	default:
		return platformv1.CredentialStatus_CREDENTIAL_STATUS_UNSPECIFIED
	}
}
