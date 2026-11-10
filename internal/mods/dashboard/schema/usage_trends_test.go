package schema

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUsageTrendsWindowsAndNullableMetrics(t *testing.T) {
	zone := time.FixedZone("UTC+8", 8*3600)
	end := time.Date(2026, 11, 9, 23, 59, 30, 123456789, zone)
	for _, span := range []string{"today", "1h", "6h", "24h", "7d"} {
		q := UsageTrendsQuery{TimeRange: span}
		window := q.Window(end)
		step := time.Duration(q.BucketSeconds(window)) * time.Second
		points := (window.End.Sub(window.Start) + step - 1) / step
		require.LessOrEqual(t, int(points), 120)
		require.Equal(t, "UTC+8", window.Timezone)
		require.Equal(t, 0, window.End.Nanosecond()%int(time.Millisecond))
		if span == "today" {
			require.Equal(t, 0, window.Start.Hour())
			require.Equal(t, end.Day(), window.Start.Day())
		}
	}
	m := (UsageTrendsTotals{}).Metrics()
	require.Nil(t, m.FailureRate)
	require.Nil(t, m.AvgLatencyMs)
	require.Nil(t, m.AvgTTFTMs)
	m = (UsageTrendsTotals{Requests: 1}).Metrics()
	require.Equal(t, float64(0), *m.AvgLatencyMs)
	require.Equal(t, float64(100), *m.FailureRate)
	for _, raw := range []string{"time_range=year", "time_range=", "time_range=1h&time_range=7d", "tenant=all"} {
		values, err := url.ParseQuery(raw)
		require.NoError(t, err)
		_, err = ParseUsageTrendsQuery(values)
		require.Error(t, err)
	}
}
