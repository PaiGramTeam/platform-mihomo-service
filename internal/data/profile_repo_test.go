package data

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"platform-mihomo-service/internal/biz"
)

func TestProfileRepoListByBindingIDReturnsOnlyBindingProfiles(t *testing.T) {
	db := newRepoTestDB(t)
	repo := NewProfileRepo(db)
	now := time.Now().UTC()

	require.NoError(t, repo.Save(context.Background(), &biz.Profile{
		BindingID:         42,
		PlatformAccountID: "binding_42_10001",
		GameBiz:           "hk4e_cn",
		Region:            "cn_gf01",
		PlayerID:          "1008611",
		Nickname:          "Traveler",
		Level:             60,
		IsDefault:         true,
		DiscoveredAt:      now,
	}))
	require.NoError(t, repo.Save(context.Background(), &biz.Profile{
		BindingID:         7,
		PlatformAccountID: "binding_7_30003",
		GameBiz:           "hk4e_cn",
		Region:            "cn_gf01",
		PlayerID:          "700001",
		Nickname:          "Other",
		Level:             50,
		IsDefault:         true,
		DiscoveredAt:      now,
	}))

	profiles, err := repo.ListByBindingID(context.Background(), 42)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	require.Equal(t, uint64(42), profiles[0].BindingID)
	require.Equal(t, "1008611", profiles[0].PlayerID)
}
