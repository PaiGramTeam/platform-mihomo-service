package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	v1 "platform-mihomo-service/api/mihomo/v1"
	"platform-mihomo-service/internal/biz"
	internalcrypto "platform-mihomo-service/internal/crypto"
	"platform-mihomo-service/internal/data"
	platformmihomo "platform-mihomo-service/internal/platform/mihomo"
)

func TestVerifyServiceTicketAcceptsExpectedClaims(t *testing.T) {
	verifier := data.NewTicketVerifier("paigram-account-center", []byte("0123456789abcdef0123456789abcdef"))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":                  "paigram-account-center",
		"aud":                  []string{"platform-mihomo-service"},
		"actor_type":           "bot",
		"actor_id":             "bot-paigram",
		"owner_user_id":        float64(1),
		"binding_id":           float64(101),
		"bot_id":               "bot-paigram",
		"platform":             "mihomo",
		"platform_service_key": "platform-mihomo-service",
		"platform_account_id":  "binding_101_10001",
		"scopes":               []string{"mihomo.status.read"},
		"exp":                  time.Now().Add(time.Minute).Unix(),
	})

	signed, err := token.SignedString([]byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)

	claims, err := verifier.Verify(signed, "platform-mihomo-service")
	require.NoError(t, err)
	require.Equal(t, "bot-paigram", claims.BotID)
	require.Equal(t, "bot", claims.ActorType)
	require.Equal(t, "bot-paigram", claims.ActorID)
	require.Equal(t, uint64(1), claims.OwnerUserID)
	require.Equal(t, uint64(101), claims.BindingID)
	require.Equal(t, "mihomo", claims.Platform)
	require.Equal(t, "platform-mihomo-service", claims.PlatformServiceKey)
	require.Equal(t, "binding_101_10001", claims.PlatformAccountID)
	require.Equal(t, []string{"mihomo.status.read"}, claims.Scopes)
}

func TestVerifyServiceTicketRejectsMissingRequiredClaims(t *testing.T) {
	verifier := data.NewTicketVerifier("paigram-account-center", []byte("0123456789abcdef0123456789abcdef"))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "paigram-account-center",
		"aud": []string{"platform-mihomo-service"},
		"exp": time.Now().Add(time.Minute).Unix(),
	})

	signed, err := token.SignedString([]byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)

	_, err = verifier.Verify(signed, "platform-mihomo-service")
	require.Error(t, err)
}

func TestRefreshCredentialDoesNotMarkRefreshedOnValidationFailure(t *testing.T) {
	credentialRepo := newMemoryCredentialRepo()
	encryptedBlob, err := internalcrypto.EncryptString(testEncryptionKey, `{"account_id":"10001","cookie_token":"abc"}`)
	require.NoError(t, err)
	credentialRepo.byPlatformAccountID["hoyo_10001"] = &biz.Credential{
		PlatformAccountID: "hoyo_10001",
		Platform:          "mihomo",
		AccountID:         "10001",
		Region:            "cn_gf01",
		CredentialBlob:    encryptedBlob,
		CredentialVersion: "v1",
		Status:            "active",
	}
	uc := NewStatusUsecase(credentialRepo, failingStatusClient{err: errors.New("credential expired")}, testEncryptionKey)

	resp, err := uc.RefreshCredential(context.Background(), "hoyo_10001")
	require.NoError(t, err)
	require.Equal(t, v1.CredentialStatus_CREDENTIAL_STATUS_EXPIRED, resp.Status)
	require.Nil(t, resp.RefreshedAt)
	require.Nil(t, credentialRepo.byPlatformAccountID["hoyo_10001"].LastRefreshedAt)
}

func TestValidateCredentialMapsChallengeRequiredStatus(t *testing.T) {
	credentialRepo := newMemoryCredentialRepo()
	encryptedBlob, err := internalcrypto.EncryptString(testEncryptionKey, `{"account_id":"10001","cookie_token":"abc"}`)
	require.NoError(t, err)
	credentialRepo.byPlatformAccountID["hoyo_10001"] = &biz.Credential{
		PlatformAccountID: "hoyo_10001",
		Platform:          "mihomo",
		AccountID:         "10001",
		Region:            "cn_gf01",
		CredentialBlob:    encryptedBlob,
		CredentialVersion: "v1",
		Status:            "active",
	}
	uc := NewStatusUsecase(credentialRepo, failingStatusClient{err: errors.New("challenge required by upstream")}, testEncryptionKey)

	resp, err := uc.ValidateCredential(context.Background(), "hoyo_10001")
	require.NoError(t, err)
	require.Equal(t, v1.CredentialStatus_CREDENTIAL_STATUS_CHALLENGE_REQUIRED, resp.Status)
	require.Equal(t, "CHALLENGE_REQUIRED", resp.ErrorCode)
}

type failingStatusClient struct {
	err error
}

func (c failingStatusClient) ValidateAndDiscover(_ context.Context, _ string, _ string) (string, string, []platformmihomo.DiscoveredProfile, error) {
	return "", "", nil, c.err
}

func (c failingStatusClient) IssueAuthKey(_ context.Context, _ string, _ string) (string, int64, error) {
	return "", 0, errors.New("not implemented")
}
