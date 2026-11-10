package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func providerTrafficReader(t *testing.T, now, started time.Time) (*Reader, *metrics.MemoryStore) {
	t.Helper()
	r, store := memoryReader(t, now, started)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&rschema.Provider{}))
	require.NoError(t, db.Create(&rschema.Provider{ID: "provider-a", Code: "a", Name: "Provider A", Enabled: 1}).Error)
	require.NoError(t, db.Create(&rschema.Provider{ID: "provider-b", Code: "b", Name: "Provider B", Enabled: 1}).Error)
	r.db = db
	return r, store
}

func providerTrafficAttempt(at time.Time, provider string, success bool) metrics.RequestMetric {
	return metrics.RequestMetric{Time: at.Unix(), Success: success, Attempts: []struct {
		EndpointID   string `json:"endpoint_id"`
		Provider     string `json:"provider,omitempty"`
		ProviderCode string `json:"provider_code,omitempty"`
		Success      bool   `json:"success"`
	}{{EndpointID: "endpoint-" + provider, ProviderCode: provider, Success: success}}}
}

func TestReadProviderTrafficCountsAttemptsWithoutPrometheusRequests(t *testing.T) {
	now := time.Now().Truncate(time.Minute).Add(20 * time.Second)
	r, store := providerTrafficReader(t, now, now.Add(-time.Hour*2))
	var promQueries atomic.Int32
	prom := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		promQueries.Add(1)
		http.Error(w, "request counters are not attempts", 500)
	}))
	defer prom.Close()
	config.C.Util.PrometheusServer.Address = prom.URL
	record := metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Success: true}
	record.Attempts = []struct {
		EndpointID   string `json:"endpoint_id"`
		Provider     string `json:"provider,omitempty"`
		ProviderCode string `json:"provider_code,omitempty"`
		Success      bool   `json:"success"`
	}{{EndpointID: "endpoint-a", ProviderCode: "a", Success: false}, {EndpointID: "endpoint-b", ProviderCode: "b", Success: true}}
	store.Record(record)
	for _, tc := range []struct {
		filter           string
		success, failure int64
	}{{"provider-a", 0, 1}, {"b", 1, 0}} {
		got, err := r.Read(context.Background(), Query{Traffic: &TrafficQuery{Provider: tc.filter, GroupBy: "provider"}})
		require.NoError(t, err)
		require.Equal(t, "ok", got.Traffic.State)
		require.Equal(t, "memory", got.Traffic.Source)
		row := got.Traffic.Data.Series[0]
		last := len(row.Total) - 1
		require.Equal(t, int64(1), *row.Total[last])
		require.Equal(t, tc.success, *row.Success[last])
		require.Equal(t, tc.failure, *row.Failure[last])
	}
	require.Zero(t, promQueries.Load())
}

func TestReadProviderTrafficKeepsRedisCodeZeroInsteadOfNameTraffic(t *testing.T) {
	now := time.Now().Truncate(time.Minute).Add(20 * time.Second)
	r, _ := providerTrafficReader(t, now, now.Add(-2*time.Hour))
	client := redis.NewClient(&redis.Options{Addr: "unused"})
	defer client.Close()
	minute := now.Add(-time.Minute).Unix() / 60
	client.AddHook(performanceRedisHook{values: map[string]interface{}{
		gatewaycontract.Keys.Status.Provider("a", minute, gatewaycontract.MetricSuccess):          "0",
		gatewaycontract.Keys.Status.Provider("Provider A", minute, gatewaycontract.MetricSuccess): "17",
	}})
	r.redis = client
	got, err := r.Read(context.Background(), Query{Traffic: &TrafficQuery{Provider: "a", GroupBy: "provider"}})
	require.NoError(t, err)
	require.Equal(t, "redis", got.Traffic.Source)
	for _, value := range got.Traffic.Data.Series[0].Total {
		require.NotNil(t, value)
		require.Zero(t, *value)
	}
}

func TestReadProviderTrafficHistoricalRetentionUsesMemory(t *testing.T) {
	now := time.Now().Truncate(time.Minute).Add(20 * time.Second)
	r, store := providerTrafficReader(t, now, now.Add(-24*time.Hour))
	client := redis.NewClient(&redis.Options{Addr: "unused"})
	defer client.Close()
	r.redis = client
	at := now.Add(-5 * time.Hour)
	store.Record(providerTrafficAttempt(at.Add(-time.Minute), "a", true))
	got, err := r.Read(context.Background(), Query{At: &at, Traffic: &TrafficQuery{Provider: "a", GroupBy: "provider"}})
	require.NoError(t, err)
	require.Equal(t, "memory", got.Traffic.Source)
	require.Equal(t, "ok", got.Traffic.State)
	require.Equal(t, int64(1), *got.Traffic.Data.Series[0].Total[59])
}

func TestReadProviderTrafficMasksUnobservedBucketsAndKeepsFreshValues(t *testing.T) {
	now := time.Now().Truncate(time.Minute).Add(20 * time.Second)
	r, store := providerTrafficReader(t, now, now.Add(-15*time.Minute))
	store.Record(providerTrafficAttempt(now.Add(-time.Minute), "a", false))
	q := Query{Traffic: &TrafficQuery{Provider: "a", GroupBy: "provider"}}
	got, err := r.Read(context.Background(), q)
	require.NoError(t, err)
	require.Equal(t, "partial", got.Traffic.State)
	require.Nil(t, got.Traffic.Data.Series[0].Total[0])
	require.Equal(t, int64(1), *got.Traffic.Data.Series[0].Total[59])
	*got.Traffic.Data.Series[0].Total[59] = 99
	again, err := r.Read(context.Background(), q)
	require.NoError(t, err)
	require.Equal(t, int64(1), *again.Traffic.Data.Series[0].Total[59])
}

func TestReadProviderTrafficRejectsAmbiguousPopulation(t *testing.T) {
	r := New(Dependencies{})
	for _, q := range []TrafficQuery{{Provider: "a"}, {Provider: "a", GroupBy: "model"}, {Provider: "a", GroupBy: "provider", Model: "m"}} {
		_, err := r.Read(context.Background(), Query{Traffic: &q})
		require.ErrorContains(t, err, "provider traffic requires")
	}
}

func TestReadProviderTrafficDoesNotTurnAbsentRedisSamplesIntoZero(t *testing.T) {
	now := time.Now().Truncate(time.Minute).Add(20 * time.Second)
	r, _ := providerTrafficReader(t, now, now.Add(-2*time.Hour))
	client := redis.NewClient(&redis.Options{Addr: "unused"})
	defer client.Close()
	client.AddHook(performanceRedisHook{values: map[string]interface{}{}})
	r.redis = client
	got, err := r.Read(context.Background(), Query{Traffic: &TrafficQuery{Provider: "a", GroupBy: "provider"}})
	require.NoError(t, err)
	require.Equal(t, "unavailable", got.Traffic.State)
	require.Equal(t, "no_source_samples", got.Traffic.Coverage.Reason)
	require.Nil(t, got.Traffic.Data)
}
