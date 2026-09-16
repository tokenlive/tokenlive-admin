package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
)

func TestAggregateMergesBeforeRankingAndKeepsCompleteDenominator(t *testing.T) {
	var rows []schema.Candidate
	for i := 0; i < 51; i++ {
		rows = append(rows, schema.Candidate{Ref: schema.KeyRef{Hash: fmt.Sprint(i)},
			Totals: schema.Totals{Requests: 1, Input: 100, Cost: decimal.RequireFromString("0.1")}})
	}
	rows = append(rows,
		schema.Candidate{Ref: schema.KeyRef{Hash: "winner", KeyID: "old"}, Totals: schema.Totals{Requests: 1, Input: 60}},
		schema.Candidate{Ref: schema.KeyRef{Hash: "winner", KeyID: "new"}, Totals: schema.Totals{Requests: 1, Input: 60}},
		schema.Candidate{Totals: schema.Totals{Requests: 1, Input: 80}})
	result := Aggregate(rows, CanonicalKeys(rows, nil), schema.Query{"today", "tokens", 10})
	require.Len(t, result.Groups, 10)
	require.Equal(t, "h:winner", result.Groups[0].Canonical)
	require.Equal(t, uint64(52), result.KeyCount)
	require.Equal(t, uint64(5300), result.All.Tokens())
	require.Equal(t, uint64(80), result.Unattributed.Tokens())
	require.Equal(t, "5.1", result.All.Cost.String())
}

func TestAggregateSortsByExactCostRequestsAndStableIdentity(t *testing.T) {
	rows := []schema.Candidate{
		{Ref: schema.KeyRef{Hash: "b"}, Totals: schema.Totals{Requests: 3, Input: 10, Cost: decimal.RequireFromString("9007199254740992.000000001")}},
		{Ref: schema.KeyRef{Hash: "a"}, Totals: schema.Totals{Requests: 1, Input: 10, Cost: decimal.RequireFromString("9007199254740992.000000002")}},
	}
	for _, tc := range []struct{ sort, want string }{{"cost", "h:a"}, {"request_count", "h:b"}, {"tokens", "h:a"}} {
		result := Aggregate(rows, CanonicalKeys(rows, nil), schema.Query{"today", tc.sort, 10})
		require.Equal(t, tc.want, result.Groups[0].Canonical)
	}
}

func TestAggregateDoesNotCountCachedTokensTwice(t *testing.T) {
	rows := []schema.Candidate{{Ref: schema.KeyRef{Hash: "a"}, Totals: schema.Totals{
		Requests: 1, Input: 100, Output: 20, Cached: 80, CacheCreation: 10,
	}}}
	result := Aggregate(rows, CanonicalKeys(rows, nil), schema.Query{"today", "tokens", 10})
	require.Equal(t, uint64(120), result.All.Tokens())
	require.Equal(t, uint64(80), result.All.Cached)
}

type fixtureResolver struct{}

func (fixtureResolver) Resolve(_ context.Context, rows []schema.Candidate) schema.Resolution {
	return CanonicalKeys(rows, nil)
}
func (fixtureResolver) Describe(_ context.Context, groups []schema.Group) map[string]schema.Metadata {
	result := map[string]schema.Metadata{}
	for _, group := range groups {
		result[group.Canonical] = schema.Metadata{Display: "sk-fixture-sensitive-secret", KeyName: "App", Source: "admin_user",
			KeyStatus: "enabled", MetadataStatus: "ready", Owner: schema.Owner{Kind: "user", ID: "u"}}
	}
	return result
}

func TestUsageServiceDTOHasAllMetricsButNoCredentialMaterial(t *testing.T) {
	reader := &fixtureReader{enabled: true, read: func(context.Context, schema.Window) ([]schema.Candidate, error) {
		return []schema.Candidate{
			{Ref: schema.KeyRef{Hash: "private-fixture-hash"}, Totals: schema.Totals{
				Requests: 2, Success: 1, Input: 10, Output: 20, Cached: 8, CacheCreation: 2, Cost: decimal.RequireFromString("0.3")}},
			{Totals: schema.Totals{Requests: 1, Input: 70}},
		}, nil
	}}
	service := NewAPIKeyUsageService(reader, fixtureResolver{}, time.Now)
	result, err := service.Query(context.Background(), schema.Query{"today", "tokens", 10})
	require.NoError(t, err)
	require.Equal(t, uint64(100), result.Summary.TotalTokens)
	require.Equal(t, float64(30), *result.Items[0].TokenShare)
	require.Equal(t, float64(50), result.Items[0].SuccessRate)
	require.Equal(t, float64(70), *result.Unattributed.TokenShare)
	require.Equal(t, "0.3", result.Items[0].TotalCost)
	require.Contains(t, result.Warnings, "unattributed_usage")
	raw, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "private-fixture-hash")
	require.NotContains(t, string(raw), "sk-fixture-sensitive-secret")
	require.Len(t, result.Items[0].RowID, 64)
}

func TestUsageServiceDisabledUnknownAndZeroTokensRemainDistinct(t *testing.T) {
	service := NewAPIKeyUsageService(&fixtureReader{}, nil, time.Now)
	disabled, err := service.Query(context.Background(), schema.Query{"today", "tokens", 10})
	require.NoError(t, err)
	require.Equal(t, "disabled", disabled.State)
	require.Nil(t, disabled.Summary)
	require.NotNil(t, disabled.Items)
	reader := &fixtureReader{enabled: true, read: func(context.Context, schema.Window) ([]schema.Candidate, error) {
		return []schema.Candidate{{Ref: schema.KeyRef{Hash: "a"}, Totals: schema.Totals{Requests: 1}}, {Totals: schema.Totals{Requests: 1}}}, nil
	}}
	service = NewAPIKeyUsageService(reader, nil, time.Now)
	result, err := service.Query(context.Background(), schema.Query{"today", "tokens", 10})
	require.NoError(t, err)
	require.Equal(t, "ready", result.State)
	require.Equal(t, uint64(2), result.Summary.RequestCount)
	require.Nil(t, result.Items[0].TokenShare)
	require.Contains(t, result.Warnings, "metadata_unavailable")
	require.Equal(t, uint64(1), result.Unattributed.RequestCount)
}
