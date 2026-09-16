package schema

import (
	"net/url"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestParseQuery(t *testing.T) {
	query, err := ParseQuery(url.Values{})
	require.NoError(t, err)
	require.Equal(t, Query{"today", "tokens", 10}, query)
	for _, raw := range []string{
		"sort_by=key_hash", "time_range=30d", "limit=100", "limit=",
		"sort_by=", "time_range=", "limit=10&limit=20", "api_key=secret",
	} {
		values, err := url.ParseQuery(raw)
		require.NoError(t, err)
		_, err = ParseQuery(values)
		require.Error(t, err, raw)
	}
	query, err = ParseQuery(url.Values{"time_range": {"7d"}, "sort_by": {"cost"}, "limit": {"50"}})
	require.NoError(t, err)
	require.Equal(t, Query{"7d", "cost", 50}, query)
}

func TestResolveWindow(t *testing.T) {
	zone := time.FixedZone("UTC+8", 8*3600)
	end := time.Date(2026, 9, 16, 12, 1, 2, 0, zone)
	for _, tc := range []struct {
		rangeName string
		start     time.Time
	}{
		{"today", time.Date(2026, 9, 16, 0, 0, 0, 0, zone)},
		{"1h", time.Date(2026, 9, 16, 11, 1, 2, 0, zone)},
		{"6h", time.Date(2026, 9, 16, 6, 1, 2, 0, zone)},
		{"24h", time.Date(2026, 9, 15, 12, 1, 2, 0, zone)},
		{"7d", time.Date(2026, 9, 9, 12, 1, 2, 0, zone)},
	} {
		window := ResolveWindow(Query{TimeRange: tc.rangeName}, end)
		require.Equal(t, tc.start, window.Start)
		require.Equal(t, end, window.End)
		require.Equal(t, "UTC+8", window.Timezone)
	}
}

func TestTotalsPreservesDecimalAndDoesNotDoubleCountCache(t *testing.T) {
	totals := Totals{Requests: 1, Success: 1, Input: 100, Output: 20, Cached: 80, Cost: decimal.RequireFromString("0.1")}
	totals.Add(Totals{Requests: 2, Success: 1, Input: 30, Output: 5, CacheCreation: 10, Cost: decimal.RequireFromString("0.2")})
	require.Equal(t, uint64(155), totals.Tokens())
	require.Equal(t, uint64(3), totals.Requests)
	require.Equal(t, uint64(2), totals.Success)
	require.Equal(t, uint64(80), totals.Cached)
	require.Equal(t, uint64(10), totals.CacheCreation)
	require.Equal(t, "0.3", totals.Cost.String())
}
