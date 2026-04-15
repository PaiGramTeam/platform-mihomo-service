package data

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"

	"platform-mihomo-service/internal/biz"
)

type TicketVerifier struct {
	issuer string
	key    []byte
}

type serviceTicketJWTClaims struct {
	ActorType            string   `json:"actor_type"`
	ActorID              string   `json:"actor_id"`
	OwnerUserID          uint64   `json:"owner_user_id"`
	BotID                string   `json:"bot_id"`
	UserID               uint64   `json:"user_id"`
	Platform             string   `json:"platform"`
	PlatformServiceKey   string   `json:"platform_service_key"`
	PlatformAccountID    string   `json:"platform_account_id"`
	PlatformAccountRefID uint64   `json:"platform_account_ref_id"`
	Scopes               []string `json:"scopes"`
	jwt.RegisteredClaims
}

func NewTicketVerifier(issuer string, key []byte) *TicketVerifier {
	return &TicketVerifier{issuer: issuer, key: key}
}

func (v *TicketVerifier) Verify(raw string, expectedAudience string) (*biz.ServiceTicketClaims, error) {
	claims := &serviceTicketJWTClaims{}

	parsed, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return v.key, nil
	}, jwt.WithAudience(expectedAudience), jwt.WithIssuer(v.issuer))
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, fmt.Errorf("invalid service ticket")
	}
	if claims.ActorType == "" {
		return nil, fmt.Errorf("service ticket missing actor_type")
	}
	if claims.ActorID == "" {
		return nil, fmt.Errorf("service ticket missing actor_id")
	}
	if claims.OwnerUserID == 0 {
		return nil, fmt.Errorf("service ticket missing owner_user_id")
	}
	if claims.PlatformAccountRefID == 0 {
		return nil, fmt.Errorf("service ticket missing platform_account_ref_id")
	}
	if claims.Platform == "" {
		return nil, fmt.Errorf("service ticket missing platform")
	}
	userID := claims.OwnerUserID
	if claims.UserID != 0 {
		userID = claims.UserID
	}

	return &biz.ServiceTicketClaims{
		ActorType:            claims.ActorType,
		ActorID:              claims.ActorID,
		OwnerUserID:          claims.OwnerUserID,
		BotID:                claims.BotID,
		UserID:               userID,
		Platform:             claims.Platform,
		PlatformServiceKey:   claims.PlatformServiceKey,
		PlatformAccountID:    claims.PlatformAccountID,
		PlatformAccountRefID: claims.PlatformAccountRefID,
		Scopes:               claims.Scopes,
		Audience:             expectedAudience,
	}, nil
}
