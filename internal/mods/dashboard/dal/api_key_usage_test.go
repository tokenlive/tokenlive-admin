package dal

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
)

type fixtureUsageRows struct {
	values   [][]any
	index    int
	scanErr  error
	finalErr error
	closed   bool
}

func (r *fixtureUsageRows) Next() bool { return r.index < len(r.values) }
func (r *fixtureUsageRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	for i, value := range r.values[r.index] {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
	}
	r.index++
	return nil
}
func (r *fixtureUsageRows) Err() error   { return r.finalErr }
func (r *fixtureUsageRows) Close() error { r.closed = true; return nil }

func fixtureValues(cost string) []any {
	return []any{"hash-a", "key-a", "workspace-a", "user-a", "tenant-a", "sk-***abcd",
		uint64(2), uint64(1), uint64(100), uint64(20), uint64(80), uint64(10), cost}
}

func TestUsageReaderDisabledAndInvalidConfigAreIsolated(t *testing.T) {
	reader, closeReader := NewClickHouseReader(config.ClickHouseConfig{})
	defer closeReader()
	require.False(t, reader.Enabled())
	rows, err := reader.Read(context.Background(), schema.Window{})
	require.NoError(t, err)
	require.Empty(t, rows)

	reader, closeReader = NewClickHouseReader(config.ClickHouseConfig{Enabled: true})
	defer closeReader()
	require.True(t, reader.Enabled())
	_, err = reader.Read(context.Background(), schema.Window{})
	require.Error(t, err)
}

func TestUsageReaderBindsWindowAndReturnsAllCandidates(t *testing.T) {
	window := schema.Window{Start: time.Unix(100, 0), End: time.Unix(200, 0)}
	rows := &fixtureUsageRows{values: [][]any{fixtureValues("0.3"), fixtureValues("0.01")}}
	reader := &ClickHouseReader{enabled: true, queryTimeout: time.Second}
	reader.query = func(ctx context.Context, query string, args ...any) (usageRows, error) {
		require.Contains(t, query, "FROM access_logs FINAL")
		require.Contains(t, query, "time >= ? AND time < ?")
		require.NotContains(t, strings.ToUpper(query), "LIMIT")
		require.Equal(t, []any{window.Start, window.End}, args)
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.LessOrEqual(t, time.Until(deadline), time.Second)
		return rows, nil
	}
	result, err := reader.Read(context.Background(), window)
	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, "hash-a", result[0].Ref.Hash)
	require.Equal(t, uint64(120), result[0].Totals.Tokens())
	require.Equal(t, "0.3", result[0].Totals.Cost.String())
	require.True(t, rows.closed)
}

func TestUsageReaderNeverReturnsPartialOrInvalidAmounts(t *testing.T) {
	for _, tc := range []struct {
		name string
		rows *fixtureUsageRows
	}{
		{"scan", &fixtureUsageRows{values: [][]any{fixtureValues("0.3")}, scanErr: errors.New("scan")}},
		{"iteration", &fixtureUsageRows{values: [][]any{fixtureValues("0.3")}, finalErr: errors.New("iteration")}},
		{"amount", &fixtureUsageRows{values: [][]any{fixtureValues("0.3"), fixtureValues("bad")}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &ClickHouseReader{enabled: true, queryTimeout: time.Second,
				query: func(context.Context, string, ...any) (usageRows, error) { return tc.rows, nil }}
			result, err := reader.Read(context.Background(), schema.Window{})
			require.Error(t, err)
			require.Nil(t, result)
			require.True(t, tc.rows.closed)
		})
	}
}

func TestUsageReaderCancellationReachesTransport(t *testing.T) {
	reader := &ClickHouseReader{enabled: true, queryTimeout: time.Millisecond,
		query: func(ctx context.Context, _ string, _ ...any) (usageRows, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}}
	_, err := reader.Read(context.Background(), schema.Window{})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = reader.Read(ctx, schema.Window{})
	require.ErrorIs(t, err, context.Canceled)
}
