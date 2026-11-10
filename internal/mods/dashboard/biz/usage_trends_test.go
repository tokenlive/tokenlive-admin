package biz

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
)

type trendsReaderFixture struct {
	enabled bool
	calls   int
	read    func(schema.Window, int64) ([]schema.UsageTrendsRow, error)
}

func (r *trendsReaderFixture) Enabled() bool { return r.enabled }
func (r *trendsReaderFixture) ReadUsageTrends(_ context.Context, w schema.Window, step int64) ([]schema.UsageTrendsRow, error) {
	r.calls++
	return r.read(w, step)
}

func TestUsageTrendsWeightedSnapshotAndWindowBoundaries(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 30, 45, 0, time.FixedZone("UTC+8", 8*3600))
	r := &trendsReaderFixture{enabled: true}
	r.read = func(w schema.Window, step int64) ([]schema.UsageTrendsRow, error) {
		require.Equal(t, now.Add(-time.Hour), w.Start)
		require.Equal(t, now, w.End)
		require.Equal(t, int64(60), step)
		return []schema.UsageTrendsRow{
			{Index: 0, UsageTrendsTotals: schema.UsageTrendsTotals{Requests: 1, Success: 1, Input: 100, Output: 20, Cached: 60, CacheCreation: 10, LatencySumMs: 100}},
			{Index: 59, UsageTrendsTotals: schema.UsageTrendsTotals{Requests: 9, Success: 7, LatencySumMs: 90, TTFTSumMs: 40, TTFTCount: 2}},
		}, nil
	}
	s := NewUsageTrendsService(r, func() time.Time { return now })
	q := schema.UsageTrendsQuery{TimeRange: "1h"}
	res, err := s.Query(context.Background(), q)
	require.NoError(t, err)
	require.Len(t, res.Buckets, 60)
	require.Equal(t, uint64(10), res.Summary.Requests)
	require.Equal(t, uint64(2), res.Summary.Failure)
	require.Equal(t, float64(20), *res.Summary.FailureRate)
	require.Equal(t, float64(19), *res.Summary.AvgLatencyMs)
	require.Equal(t, float64(20), *res.Summary.AvgTTFTMs)
	require.Equal(t, uint64(120), res.Summary.TotalTokens)
	require.Nil(t, res.Buckets[1].AvgLatencyMs)
	require.Nil(t, res.Buckets[1].FailureRate)
	require.Equal(t, res.Window.Start, res.Buckets[0].Start)
	require.Equal(t, res.Window.End, res.Buckets[59].End)
	var count uint64
	for i, bucket := range res.Buckets {
		count += bucket.Requests
		if i > 0 {
			require.Equal(t, res.Buckets[i-1].End, bucket.Start)
		}
	}
	require.Equal(t, res.Summary.Requests, count)
	_, err = s.Query(context.Background(), q)
	require.NoError(t, err)
	require.Equal(t, 1, r.calls)
}

func TestUsageTrendsFailureDisabledAndMidnightExpiry(t *testing.T) {
	now := time.Date(2026, 11, 9, 23, 59, 50, 0, time.UTC)
	r := &trendsReaderFixture{enabled: true}
	fail := true
	r.read = func(w schema.Window, _ int64) ([]schema.UsageTrendsRow, error) {
		if fail {
			return nil, errors.New("unavailable")
		}
		require.Equal(t, now.Day(), w.Start.Day())
		return nil, nil
	}
	s := NewUsageTrendsService(r, func() time.Time { return now })
	q := schema.UsageTrendsQuery{TimeRange: "today"}
	_, err := s.Query(context.Background(), q)
	require.Error(t, err)
	fail = false
	res, err := s.Query(context.Background(), q)
	require.NoError(t, err)
	require.Equal(t, 2, r.calls)
	require.Zero(t, res.Summary.Requests)
	now = now.Add(11 * time.Second)
	res, err = s.Query(context.Background(), q)
	require.NoError(t, err)
	require.Equal(t, 3, r.calls)
	require.Equal(t, 10, res.Window.Start.Day())
	require.Len(t, res.Buckets, 1)
	require.Equal(t, now, res.Buckets[0].End)
	r.enabled = false
	res, err = s.Query(context.Background(), q)
	require.NoError(t, err)
	require.Equal(t, "disabled", res.State)
	require.Nil(t, res.Summary)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.Query(ctx, q)
	require.ErrorIs(t, err, context.Canceled)
}
