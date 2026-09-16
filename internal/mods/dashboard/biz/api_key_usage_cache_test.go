package biz

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
)

type fixtureReader struct {
	enabled bool
	calls   atomic.Int32
	read    func(context.Context, schema.Window) ([]schema.Candidate, error)
}

func (r *fixtureReader) Enabled() bool { return r.enabled }
func (r *fixtureReader) Read(ctx context.Context, window schema.Window) ([]schema.Candidate, error) {
	r.calls.Add(1)
	return r.read(ctx, window)
}

func TestUsageCacheExpiresAtMidnightAndCopiesResponses(t *testing.T) {
	zone := time.FixedZone("UTC+8", 8*3600)
	var clock atomic.Int64
	clock.Store(time.Date(2026, 9, 16, 23, 59, 50, 0, zone).Unix())
	now := func() time.Time { return time.Unix(clock.Load(), 0).In(zone) }
	reader := &fixtureReader{enabled: true, read: func(context.Context, schema.Window) ([]schema.Candidate, error) {
		return []schema.Candidate{{Ref: schema.KeyRef{Hash: "a"}, Totals: schema.Totals{Requests: 1, Input: 1}}}, nil
	}}
	service := NewAPIKeyUsageService(reader, fixtureResolver{}, now)
	query := schema.Query{"today", "tokens", 10}
	first, err := service.Query(context.Background(), query)
	require.NoError(t, err)
	end := first.Window.End
	first.Items[0].KeyName = "mutated"
	*first.Items[0].TokenShare = 999
	first.Summary.TotalTokens = 999
	second, err := service.Query(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, "App", second.Items[0].KeyName)
	require.Equal(t, float64(100), *second.Items[0].TokenShare)
	require.Equal(t, uint64(1), second.Summary.TotalTokens)
	require.Equal(t, end, second.Window.End)
	require.Equal(t, int32(1), reader.calls.Load())
	clock.Add(11)
	_, err = service.Query(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, int32(2), reader.calls.Load())
	query.Limit = 20
	_, err = service.Query(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, int32(3), reader.calls.Load())
}

func TestUsageCacheCoalescesAndDoesNotCancelOtherWaiters(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	reader := &fixtureReader{enabled: true, read: func(ctx context.Context, _ schema.Window) ([]schema.Candidate, error) {
		close(started)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-release:
			return nil, nil
		}
	}}
	service := NewAPIKeyUsageService(reader, fixtureResolver{}, time.Now)
	query := schema.Query{"today", "tokens", 10}
	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() { _, err := service.Query(ctx, query); first <- err }()
	<-started
	second := make(chan error, 1)
	go func() { _, err := service.Query(context.Background(), query); second <- err }()
	cancel()
	require.ErrorIs(t, <-first, context.Canceled)
	close(release)
	require.NoError(t, <-second)
	require.Equal(t, int32(1), reader.calls.Load())
}

func TestUsageCacheDoesNotCacheErrorsOrAcceptInvalidQuery(t *testing.T) {
	var attempts atomic.Int32
	reader := &fixtureReader{enabled: true, read: func(context.Context, schema.Window) ([]schema.Candidate, error) {
		if attempts.Add(1) == 1 {
			return nil, errors.New("offline")
		}
		return nil, nil
	}}
	service := NewAPIKeyUsageService(reader, nil, time.Now)
	_, err := service.Query(context.Background(), schema.Query{})
	require.Error(t, err)
	require.Zero(t, reader.calls.Load())
	query := schema.Query{"today", "tokens", 10}
	_, err = service.Query(context.Background(), query)
	require.Error(t, err)
	result, err := service.Query(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, "ready", result.State)
	require.Equal(t, uint64(0), result.Summary.RequestCount)
	require.Equal(t, int32(2), reader.calls.Load())
}
