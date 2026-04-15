package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "platform-mihomo-service/api/mihomo/v1"
	"platform-mihomo-service/internal/biz"
	"platform-mihomo-service/internal/data"
	"platform-mihomo-service/internal/usecase"
)

const serviceTicketAudience = "platform-mihomo-service"

type MihomoAccountService struct {
	v1.UnimplementedMihomoAccountServiceServer

	ticketVerifier *data.TicketVerifier
	bindUC         *usecase.BindUsecase
	statusUC       *usecase.StatusUsecase
	profileUC      *usecase.ProfileUsecase
	authkeyUC      *usecase.AuthkeyUsecase
	managementUC   *usecase.ManagementUsecase
}

func NewMihomoAccountService(
	ticketVerifier *data.TicketVerifier,
	bindUC *usecase.BindUsecase,
	statusUC *usecase.StatusUsecase,
	profileUC *usecase.ProfileUsecase,
	authkeyUC *usecase.AuthkeyUsecase,
	managementUC *usecase.ManagementUsecase,
) *MihomoAccountService {
	return &MihomoAccountService{
		ticketVerifier: ticketVerifier,
		bindUC:         bindUC,
		statusUC:       statusUC,
		profileUC:      profileUC,
		authkeyUC:      authkeyUC,
		managementUC:   managementUC,
	}
}

func (s *MihomoAccountService) BindCredential(ctx context.Context, req *v1.BindCredentialRequest) (*v1.BindCredentialResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	claims, err := s.verifyServiceTicket(req.GetServiceTicket())
	if err != nil {
		return nil, err
	}

	output, err := s.bindUC.BindCredential(ctx, toBindCredentialInput(req, claims))
	if err != nil {
		return nil, mapUsecaseError(err)
	}

	return toBindCredentialResponse(output), nil
}

func (s *MihomoAccountService) GetCredentialStatus(ctx context.Context, req *v1.GetCredentialStatusRequest) (*v1.GetCredentialStatusResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	claims, err := s.verifyServiceTicket(req.GetServiceTicket())
	if err != nil {
		return nil, err
	}
	if err := validateScopedPlatformAccountID(claims, req.GetPlatformAccountId()); err != nil {
		return nil, err
	}

	output, err := s.statusUC.GetCredentialStatus(ctx, req.GetPlatformAccountId())
	if err != nil {
		return nil, mapUsecaseError(err)
	}

	return &v1.GetCredentialStatusResponse{
		Status:          output.Status,
		LastValidatedAt: toTimestamp(output.LastValidatedAt),
	}, nil
}

func (s *MihomoAccountService) ValidateCredential(ctx context.Context, req *v1.ValidateCredentialRequest) (*v1.ValidateCredentialResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	claims, err := s.verifyServiceTicket(req.GetServiceTicket())
	if err != nil {
		return nil, err
	}
	if err := validateScopedPlatformAccountID(claims, req.GetPlatformAccountId()); err != nil {
		return nil, err
	}

	output, err := s.statusUC.ValidateCredential(ctx, req.GetPlatformAccountId())
	if err != nil {
		return nil, mapUsecaseError(err)
	}

	return &v1.ValidateCredentialResponse{Status: output.Status, ErrorCode: output.ErrorCode}, nil
}

func (s *MihomoAccountService) RefreshCredential(ctx context.Context, req *v1.RefreshCredentialRequest) (*v1.RefreshCredentialResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	claims, err := s.verifyServiceTicket(req.GetServiceTicket())
	if err != nil {
		return nil, err
	}
	if err := validateScopedPlatformAccountID(claims, req.GetPlatformAccountId()); err != nil {
		return nil, err
	}

	output, err := s.statusUC.RefreshCredential(ctx, req.GetPlatformAccountId())
	if err != nil {
		return nil, mapUsecaseError(err)
	}

	return &v1.RefreshCredentialResponse{Status: output.Status, RefreshedAt: toTimestamp(output.RefreshedAt)}, nil
}

func (s *MihomoAccountService) ListProfiles(ctx context.Context, req *v1.ListProfilesRequest) (*v1.ListProfilesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	claims, err := s.verifyServiceTicket(req.GetServiceTicket())
	if err != nil {
		return nil, err
	}
	if err := validateScopedPlatformAccountID(claims, req.GetPlatformAccountId()); err != nil {
		return nil, err
	}

	profiles, err := s.profileUC.ListProfiles(ctx, req.GetPlatformAccountId())
	if err != nil {
		return nil, mapUsecaseError(err)
	}

	return &v1.ListProfilesResponse{Profiles: profiles}, nil
}

func (s *MihomoAccountService) GetPrimaryProfile(ctx context.Context, req *v1.GetPrimaryProfileRequest) (*v1.GetPrimaryProfileResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	claims, err := s.verifyServiceTicket(req.GetServiceTicket())
	if err != nil {
		return nil, err
	}
	if err := validateScopedPlatformAccountID(claims, req.GetPlatformAccountId()); err != nil {
		return nil, err
	}

	profile, err := s.profileUC.GetPrimaryProfile(ctx, req.GetPlatformAccountId())
	if err != nil {
		return nil, mapUsecaseError(err)
	}

	return &v1.GetPrimaryProfileResponse{Profile: profile}, nil
}

func (s *MihomoAccountService) GetAuthKey(ctx context.Context, req *v1.GetAuthKeyRequest) (*v1.GetAuthKeyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	claims, err := s.verifyServiceTicket(req.GetServiceTicket())
	if err != nil {
		return nil, err
	}
	if err := validateScopedPlatformAccountID(claims, req.GetPlatformAccountId()); err != nil {
		return nil, err
	}

	output, err := s.authkeyUC.GetAuthKey(ctx, req.GetPlatformAccountId(), req.GetPlayerId())
	if err != nil {
		return nil, mapUsecaseError(err)
	}

	return &v1.GetAuthKeyResponse{Authkey: output.AuthKey, ExpiresAt: toTimestamp(&output.ExpiresAt)}, nil
}

func (s *MihomoAccountService) GetCredentialSummary(ctx context.Context, req *v1.GetCredentialSummaryRequest) (*v1.GetCredentialSummaryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	claims, err := s.verifyServiceTicket(req.GetServiceTicket())
	if err != nil {
		return nil, err
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

	return toCredentialSummaryResponse(output), nil
}

func (s *MihomoAccountService) UpdateCredential(ctx context.Context, req *v1.UpdateCredentialRequest) (*v1.UpdateCredentialResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	claims, err := s.verifyServiceTicket(req.GetServiceTicket())
	if err != nil {
		return nil, err
	}
	if err := validateScopedPlatformAccountID(claims, req.GetPlatformAccountId()); err != nil {
		return nil, err
	}
	if err := requireScopes(claims, "mihomo.credential.update"); err != nil {
		return nil, err
	}
	if req.GetDevice() == nil {
		return nil, status.Error(codes.InvalidArgument, "device is required")
	}
	device := req.GetDevice()

	input := usecase.UpdateCredentialInput{
		PlatformAccountID: req.GetPlatformAccountId(),
		BindCredentialInput: usecase.BindCredentialInput{
			PlatformAccountRefID: claims.PlatformAccountRefID,
			CookieBundleJSON:     req.GetCookieBundleJson(),
			RegionHint:           req.GetRegionHint(),
		},
	}
	if device != nil {
		input.DeviceID = device.GetDeviceId()
		input.DeviceFP = device.GetDeviceFp()
		input.DeviceName = device.GetDeviceName()
	}

	output, err := s.managementUC.UpdateCredential(ctx, input)
	if err != nil {
		return nil, mapUsecaseError(err)
	}

	return &v1.UpdateCredentialResponse{Summary: toCredentialSummary(output)}, nil
}

func (s *MihomoAccountService) DeleteCredential(ctx context.Context, req *v1.DeleteCredentialRequest) (*v1.DeleteCredentialResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	claims, err := s.verifyServiceTicket(req.GetServiceTicket())
	if err != nil {
		return nil, err
	}
	if err := validateScopedPlatformAccountID(claims, req.GetPlatformAccountId()); err != nil {
		return nil, err
	}
	if err := requireScopes(claims, "mihomo.credential.delete"); err != nil {
		return nil, err
	}

	if err := s.managementUC.DeleteCredential(ctx, req.GetPlatformAccountId()); err != nil {
		return nil, mapUsecaseError(err)
	}

	return &v1.DeleteCredentialResponse{Success: true}, nil
}

func (s *MihomoAccountService) UpsertDevice(ctx context.Context, req *v1.UpsertDeviceRequest) (*v1.UpsertDeviceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := s.verifyServiceTicket(req.GetServiceTicket())
	if err != nil {
		return nil, err
	}
	if err := validateScopedPlatformAccountID(claims, req.GetPlatformAccountId()); err != nil {
		return nil, err
	}
	device := req.GetDevice()
	if device == nil {
		return nil, status.Error(codes.InvalidArgument, "device is required")
	}
	if err := s.bindUC.UpsertDevice(ctx, req.GetPlatformAccountId(), device.GetDeviceId(), device.GetDeviceFp(), device.GetDeviceName()); err != nil {
		return nil, mapUsecaseError(err)
	}
	return &v1.UpsertDeviceResponse{Success: true}, nil
}

func (s *MihomoAccountService) ConfirmPrimaryProfile(ctx context.Context, req *v1.ConfirmPrimaryProfileRequest) (*v1.ConfirmPrimaryProfileResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	claims, err := s.verifyServiceTicket(req.GetServiceTicket())
	if err != nil {
		return nil, err
	}
	if err := validateScopedPlatformAccountID(claims, req.GetPlatformAccountId()); err != nil {
		return nil, err
	}
	profile, err := s.profileUC.ConfirmPrimaryProfile(ctx, req.GetPlatformAccountId(), req.GetPlayerId())
	if err != nil {
		return nil, mapUsecaseError(err)
	}
	return &v1.ConfirmPrimaryProfileResponse{Profile: profile}, nil
}

func (s *MihomoAccountService) verifyServiceTicket(raw string) (*biz.ServiceTicketClaims, error) {
	claims, err := s.ticketVerifier.Verify(raw, serviceTicketAudience)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid service ticket")
	}

	return claims, nil
}

func mapUsecaseError(err error) error {
	if errors.Is(err, usecase.ErrCredentialNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, usecase.ErrProfileNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	if errors.Is(err, usecase.ErrPlatformAccountMismatch) {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	return status.Error(codes.Internal, err.Error())
}

func validateScopedPlatformAccountID(claims *biz.ServiceTicketClaims, platformAccountID string) error {
	if claims.Platform != "mihomo" {
		return status.Error(codes.PermissionDenied, "platform is outside ticket scope")
	}
	if claims.PlatformAccountID == "" {
		return status.Error(codes.PermissionDenied, "platform account is outside ticket scope")
	}
	if claims.PlatformAccountID != platformAccountID {
		return status.Error(codes.PermissionDenied, "platform account is outside ticket scope")
	}
	expectedPrefix := fmt.Sprintf("hoyo_ref_%d_", claims.PlatformAccountRefID)
	if !strings.HasPrefix(platformAccountID, expectedPrefix) {
		return status.Error(codes.PermissionDenied, "platform account is outside ticket scope")
	}
	return nil
}

func requireScopes(claims *biz.ServiceTicketClaims, required ...string) error {
	granted := make(map[string]struct{}, len(claims.Scopes))
	for _, scope := range claims.Scopes {
		granted[scope] = struct{}{}
	}
	for _, scope := range required {
		if _, ok := granted[scope]; !ok {
			return status.Error(codes.PermissionDenied, "scope is not granted")
		}
	}
	return nil
}
