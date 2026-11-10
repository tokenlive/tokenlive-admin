package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func memoryReader(t *testing.T, now time.Time, started time.Time) (*Reader, *metrics.MemoryStore) {
	t.Helper()
	old := config.C.Util.PrometheusServer.Address
	config.C.Util.PrometheusServer.Address = ""
	t.Cleanup(func() { config.C.Util.PrometheusServer.Address = old })
	store := metrics.NewMemoryStoreAt(started)
	return New(Dependencies{Memory: store, Clock: func() time.Time { return now }}), store
}

func TestReadCapturesOneCutoffAndKeepsFailuresLocal(t *testing.T) {
	now := time.Now().Truncate(time.Minute).Add(42 * time.Second)
	r, store := memoryReader(t, now, now.Add(-48*time.Hour))
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Model: "m", Success: true, DurationMs: 1000, TTFTMs: 120, OutputTokens: 20})
	calls := 0
	r.clock = func() time.Time { calls++; return now.Add(time.Duration(calls-1) * time.Minute) }
	s, err := r.Read(context.Background(), Query{Overview: true, Traffic: &TrafficQuery{}, Performance: &PerformanceQuery{Model: "m"}, Models: &ModelQuery{}})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, now, s.Cutoff)
	require.Equal(t, "ok", s.Traffic.State)
	require.Equal(t, "ok", s.Performance.State)
	require.Equal(t, now.Truncate(time.Minute), s.Traffic.Window.End)
	require.Equal(t, s.Traffic.Window.End, s.Performance.Window.End)
	require.Equal(t, "error", s.Models.State)
	require.Equal(t, "source_unavailable", s.Models.Error)
	require.NotNil(t, s.Overview.Data.DailyRequests)
	require.Equal(t, now, s.Overview.Windows["daily"].End)
	require.Equal(t, now.Add(-5*time.Minute), s.Overview.Windows["latency"].Start)
	require.Equal(t, int64(1), *s.Traffic.Data.Series[0].Total[59])
	require.Equal(t, 120.0, *s.Performance.Data.AvgTTFTMs[59])
}

func TestReadSevenDaysAndHistoricalTodayUsePlannedWindow(t *testing.T) {
	now := time.Now().Truncate(time.Minute).Add(42 * time.Second)
	r, store := memoryReader(t, now, now.Add(-8*24*time.Hour))
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&rschema.Model{}))
	r.db = db
	require.NoError(t, db.Create(&rschema.Model{ModelCode: "m", ModelName: "Model", Enabled: 1}).Error)
	store.Record(metrics.RequestMetric{Time: now.Add(-3 * 24 * time.Hour).Unix(), Model: "m", Success: true})
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Model: "m", Success: true})
	s, err := r.Read(context.Background(), Query{Models: &ModelQuery{WindowQuery: WindowQuery{Range: "7d"}}})
	require.NoError(t, err)
	require.Equal(t, "partial", s.Models.State)
	require.Len(t, *s.Models.Data, 1)
	require.Equal(t, int64(2), (*s.Models.Data)[0].RequestCount)
	at := now.Add(-24 * time.Hour)
	a, err := r.Read(context.Background(), Query{At: &at, Timezone: "+08:00", Traffic: &TrafficQuery{WindowQuery: WindowQuery{Range: "today"}}})
	require.NoError(t, err)
	want := at.In(time.FixedZone("+08:00", 8*3600))
	want = time.Date(want.Year(), want.Month(), want.Day(), 0, 0, 0, 0, want.Location())
	require.Equal(t, want, a.Traffic.Window.Start)
	require.Equal(t, at.Truncate(time.Minute).Unix(), a.Traffic.Window.End.Unix())
	b, err := r.Read(context.Background(), Query{At: &at, Timezone: "-05:00", Traffic: &TrafficQuery{WindowQuery: WindowQuery{Range: "today"}}})
	require.NoError(t, err)
	require.NotEqual(t, a.Traffic.Window.Start.Unix(), b.Traffic.Window.Start.Unix())
}

func TestReadRetentionAndCacheDoNotInventZerosOrMutateResults(t *testing.T) {
	now := time.Now().Truncate(time.Minute).Add(42 * time.Second)
	r, store := memoryReader(t, now, now.Add(-32*time.Minute))
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Model: "m", Success: true, TTFTMs: 100, DurationMs: 1000})
	q := Query{Traffic: &TrafficQuery{}, Performance: &PerformanceQuery{Model: "m"}}
	s, err := r.Read(context.Background(), q)
	require.NoError(t, err)
	require.Equal(t, "partial", s.Traffic.State)
	require.Nil(t, s.Traffic.Data.Series[0].Total[0])
	require.NotNil(t, s.Traffic.Data.Series[0].Total[59])
	require.Nil(t, s.Performance.Data.AvgTTFTMs[0])
	require.NotNil(t, s.Performance.Data.AvgTTFTMs[59])
	s.Traffic.Data.Series[0].Total[59] = nil
	s.Performance.Data.AvgTTFTMs[59] = nil
	s, err = r.Read(context.Background(), q)
	require.NoError(t, err)
	require.Equal(t, int64(1), *s.Traffic.Data.Series[0].Total[59])
	require.Equal(t, 100.0, *s.Performance.Data.AvgTTFTMs[59])
	at := now.Add(-8 * 24 * time.Hour)
	s, err = r.Read(context.Background(), Query{At: &at, Traffic: &TrafficQuery{}})
	require.NoError(t, err)
	require.Equal(t, "unavailable", s.Traffic.State)
	require.Nil(t, s.Traffic.Data)
}

func TestReadRejectsInvalidWholeQueryBeforeSourceWork(t *testing.T) {
	now := time.Now()
	r, _ := memoryReader(t, now, now.Add(-time.Hour))
	future := now.Add(time.Minute)
	for _, q := range []Query{{}, {Timezone: "bad-zone", Overview: true}, {Timezone: "++1:00", Overview: true}, {Traffic: &TrafficQuery{WindowQuery: WindowQuery{Range: "8d"}}}, {At: &future, Traffic: &TrafficQuery{}}, {Traffic: &TrafficQuery{WindowQuery: WindowQuery{Range: "garbage"}}}, {Overview: true, Traffic: &TrafficQuery{GroupBy: "unknown"}}, {Performance: &PerformanceQuery{}}, {Models: &ModelQuery{Limit: 101}}} {
		_, err := r.Read(context.Background(), q)
		require.Error(t, err)
	}
	require.Empty(t, r.cacheMap)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := r.Read(ctx, Query{Traffic: &TrafficQuery{}})
	require.ErrorIs(t, err, context.Canceled)
}

func TestReadPrometheusPartialBucketAndCoverage(t *testing.T) {
	// A custom start retains the six-hour preset grid and its real final bucket.
	now := time.Now().Truncate(24 * time.Hour).Add(10*time.Hour + 37*time.Minute + 42*time.Second).UTC()
	var mu sync.Mutex
	var queries []string
	missing := now.Truncate(time.Minute).Add(-10 * time.Minute).Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/-/healthy" {
			fmt.Fprint(w, "ok")
			return
		}
		q := r.URL.Query()
		query := q.Get("query")
		mu.Lock()
		queries = append(queries, query)
		mu.Unlock()
		start, _ := strconv.ParseInt(q.Get("start"), 10, 64)
		end, _ := strconv.ParseInt(q.Get("end"), 10, 64)
		step, _ := strconv.ParseInt(q.Get("step"), 10, 64)
		values := [][]any{}
		for at := start; at <= end && step > 0; at += step {
			if strings.HasPrefix(query, "count(count_over_time") && at == missing {
				continue
			}
			values = append(values, []any{at, "1"})
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{"resultType": "matrix", "result": []any{map[string]any{"metric": map[string]string{}, "values": values}}}})
	}))
	defer server.Close()
	old := config.C.Util.PrometheusServer.Address
	config.C.Util.PrometheusServer.Address = server.URL
	t.Cleanup(func() { config.C.Util.PrometheusServer.Address = old })
	r := New(Dependencies{Clock: func() time.Time { return now }})
	start := now.Truncate(time.Hour)
	s, err := r.Read(context.Background(), Query{Timezone: "UTC", Traffic: &TrafficQuery{WindowQuery: WindowQuery{Range: "6h", Start: &start}}})
	require.NoError(t, err)
	require.Equal(t, "partial", s.Traffic.State)
	last := s.Traffic.Buckets[len(s.Traffic.Buckets)-1]
	require.Equal(t, 2*time.Minute, last.End.Sub(last.Start))
	require.Equal(t, now.Truncate(time.Minute), last.End)
	require.Equal(t, int64(2), *s.Traffic.Data.Series[0].Total[len(s.Traffic.Buckets)-1])
	require.Contains(t, strings.Join(queries, "\n"), "[120s]")
	require.Nil(t, s.Traffic.Data.Series[0].Total[5])
}

func TestReadMidnightHasNoCompletedMinutes(t *testing.T) {
	now := time.Now()
	r, _ := memoryReader(t, now, now.Add(-48*time.Hour))
	at := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 42, 0, time.Local)
	s, err := r.Read(context.Background(), Query{At: &at, Traffic: &TrafficQuery{WindowQuery: WindowQuery{Range: "today"}}, Models: &ModelQuery{WindowQuery: WindowQuery{Range: "today"}}})
	require.NoError(t, err)
	require.Equal(t, "unavailable", s.Traffic.State)
	require.Equal(t, "no_completed_minutes", s.Traffic.Coverage.Reason)
	require.Equal(t, "unavailable", s.Models.State)
	require.Nil(t, s.Models.Data)
}

func TestReadDailyTotalsNeverRelabelAnotherCalendar(t *testing.T) {
	now := time.Now()
	r, _ := memoryReader(t, now, now.Add(-48*time.Hour))
	// Pick a fixed zone whose midnight differs from the local calendar.
	_, offset := now.Zone()
	zone := "+14:00"
	if offset == 14*3600 {
		zone = "-12:00"
	}
	s, err := r.Read(context.Background(), Query{Timezone: zone, Overview: true})
	require.NoError(t, err)
	require.Equal(t, "partial", s.Overview.State)
	require.Nil(t, s.Overview.Data.DailyRequests)
	require.NotNil(t, s.Overview.Data.QPS)
	require.Nil(t, s.Overview.Coverage.Start)
}

func TestReadProviderRangeKeepsSevenDayCeiling(t *testing.T) {
	now := time.Now()
	r, _ := memoryReader(t, now, now.Add(-8*24*time.Hour))
	s, err := r.Read(context.Background(), Query{Providers: &ProviderQuery{WindowQuery: WindowQuery{Range: "30d"}}})
	require.NoError(t, err)
	require.Equal(t, 7*24*time.Hour, s.Providers.Window.End.Sub(s.Providers.Window.Start))
	// A missing catalog is local to the result, not a window-validation failure.
	require.Equal(t, "error", s.Providers.State)
}
