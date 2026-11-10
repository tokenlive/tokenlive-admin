package dal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
)

func TestUsageTrendsReaderBindsDeduplicatedWindowAndSamples(t *testing.T) {
	w := schema.Window{Start: time.Unix(100, 123000000), End: time.Unix(200, 123000000)}
	values := []any{uint64(0), uint64(2), uint64(1), uint64(100), uint64(20), uint64(60), uint64(10), uint64(0), uint64(30), uint64(1)}
	rows := &fixtureUsageRows{values: [][]any{values}}
	r := &ClickHouseReader{enabled: true, queryTimeout: time.Second}
	r.query = func(_ context.Context, sql string, args ...any) (usageRows, error) {
		require.Contains(t, sql, "FROM access_logs FINAL")
		require.Contains(t, sql, "time >= fromUnixTimestamp64Milli(?) AND time < fromUnixTimestamp64Milli(?)")
		require.Contains(t, sql, "is_stream = 1 AND ttft_ms > 0")
		require.NotContains(t, sql, "api_key")
		require.Equal(t, []any{w.Start.UnixMilli(), int64(60000), w.Start.UnixMilli(), w.End.UnixMilli()}, args)
		return rows, nil
	}
	result, err := r.ReadUsageTrends(context.Background(), w, 60)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, uint64(1), result[0].Metrics().Failure)
	require.Equal(t, float64(0), *result[0].Metrics().AvgLatencyMs)
	require.Equal(t, uint64(120), result[0].Metrics().TotalTokens)
	require.True(t, rows.closed)
	rows.index = 0
	rows.finalErr = errors.New("partial response")
	result, err = r.ReadUsageTrends(context.Background(), w, 60)
	require.Error(t, err)
	require.Nil(t, result)
}
