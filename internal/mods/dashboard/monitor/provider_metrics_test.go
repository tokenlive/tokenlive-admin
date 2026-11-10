package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
	"gorm.io/gorm"
)

type healthAttempt = struct {
	EndpointID   string `json:"endpoint_id"`
	Provider     string `json:"provider,omitempty"`
	ProviderCode string `json:"provider_code,omitempty"`
	Success      bool   `json:"success"`
}

func providerHealthReader(t *testing.T) (*Reader, *metrics.MemoryStore, time.Time) {
	t.Helper()
	now := time.Now().Truncate(time.Minute).Add(30 * time.Second)
	r, store := memoryReader(t, now, now.Add(-8*24*time.Hour))
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.AutoMigrate(&rschema.Provider{}, &rschema.Endpoint{}))
	require.NoError(t, db.Create(&[]rschema.Provider{
		{ID: "a", Code: "prov-a", Name: "Provider A", Protocol: "openai", Enabled: 1},
		{ID: "b", Code: "prov-b", Name: "Provider B", Protocol: "openai", Enabled: 1},
	}).Error)
	r.db = db
	return r, store, now
}

func readProviderHealth(t *testing.T, r *Reader, provider string) ProviderHealth {
	t.Helper()
	snapshot, err := r.Read(context.Background(), Query{Providers: &ProviderQuery{Provider: provider}})
	require.NoError(t, err)
	require.NotNil(t, snapshot.Providers)
	require.NotNil(t, snapshot.Providers.Data, "reading state=%s error=%s", snapshot.Providers.State, snapshot.Providers.Error)
	require.Len(t, *snapshot.Providers.Data, 1)
	return (*snapshot.Providers.Data)[0]
}

func TestProviderHealthCountsEveryAttemptAndPreservesZeroRate(t *testing.T) {
	r, store, now := providerHealthReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Success: true, InputTokens: 10, OutputTokens: 30, Cost: 0.25, DurationMs: 2000, TTFTMs: 120,
		Attempts: []healthAttempt{{EndpointID: "ep-a", ProviderCode: "prov-a", Success: false}, {EndpointID: "ep-b", ProviderCode: "prov-b", Success: true}}})
	a := readProviderHealth(t, r, "prov-a")
	require.Equal(t, "ok", a.Calls.State)
	require.Equal(t, "upstream_call", a.Calls.Scope)
	require.Equal(t, "memory", a.Calls.Source)
	require.Equal(t, &ProviderCalls{Total: 1, Failure: 1}, a.Calls.Value)
	require.NotNil(t, a.SuccessRate.Value)
	require.Equal(t, 0.0, *a.SuccessRate.Value)
	require.Nil(t, a.AvgLatencyMs.Value)
	require.Equal(t, "no_samples", a.AvgLatencyMs.State)
	b := readProviderHealth(t, r, "prov-b")
	require.Equal(t, &ProviderCalls{Total: 1, Success: 1}, b.Calls.Value)
	require.Equal(t, 100.0, *b.SuccessRate.Value)
	require.Equal(t, 2000.0, *b.AvgLatencyMs.Value)
	require.Equal(t, 120.0, *b.AvgTTFTMs.Value)
	require.Equal(t, int64(40), *b.TotalTokens.Value)
	require.Equal(t, 0.25, *b.TotalCost.Value)
	require.Equal(t, 15.0, *b.Otps.Value)
	require.Equal(t, "successful_provider_requests", b.AvgLatencyMs.Scope)
}

func TestProviderHealthDistinguishesQuietCountsFromMissingPerformance(t *testing.T) {
	r, _, _ := providerHealthReader(t)
	p := readProviderHealth(t, r, "prov-a")
	require.Equal(t, "ok", p.Calls.State)
	require.Equal(t, &ProviderCalls{}, p.Calls.Value)
	require.Nil(t, p.SuccessRate.Value)
	require.Equal(t, "no_samples", p.SuccessRate.State)
	require.Equal(t, "zero_denominator", p.SuccessRate.Reason)
	require.Nil(t, p.TotalTokens.Value)
	require.Nil(t, p.TotalCost.Value)
	require.Nil(t, p.AvgLatencyMs.Value)
	require.Equal(t, "no_samples", p.TotalTokens.State)
	require.NotNil(t, p.EndpointCount.Value)
	require.Equal(t, int64(0), *p.EndpointCount.Value)
}

func TestProviderHealthKeepsObservedZeroUsage(t *testing.T) {
	r, store, now := providerHealthReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Success: true, DurationMs: 1000, Attempts: []healthAttempt{{ProviderCode: "prov-a", Success: true}}})
	p := readProviderHealth(t, r, "a")
	require.Equal(t, "ok", p.TotalTokens.State)
	require.NotNil(t, p.TotalTokens.Value)
	require.Equal(t, int64(0), *p.TotalTokens.Value)
	require.NotNil(t, p.TotalCost.Value)
	require.Equal(t, 0.0, *p.TotalCost.Value)
	require.Nil(t, p.AvgTTFTMs.Value)
	require.Equal(t, "no_samples", p.AvgTTFTMs.State)
}

func TestProviderHealthCacheDoesNotShareMutableObservations(t *testing.T) {
	r, store, now := providerHealthReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Success: true, DurationMs: 1500, Attempts: []healthAttempt{{ProviderCode: "prov-a", Success: true}}})
	first := readProviderHealth(t, r, "prov-a")
	first.Calls.Value.Total = 999
	*first.SuccessRate.Value = 22
	*first.AvgLatencyMs.Value = 999
	*first.Calls.Coverage.Start = time.Time{}
	again := readProviderHealth(t, r, "prov-a")
	require.Equal(t, int64(1), again.Calls.Value.Total)
	require.Equal(t, 100.0, *again.SuccessRate.Value)
	require.Equal(t, 1500.0, *again.AvgLatencyMs.Value)
	require.False(t, again.Calls.Coverage.Start.IsZero())
}

type healthRedisHook struct {
	values map[string]interface{}
	err    error
	mgets  int
}

func (*healthRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) { return next(ctx, network, addr) }
}
func (h *healthRedisHook) ProcessHook(_ redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		switch c := cmd.(type) {
		case *redis.SliceCmd:
			h.mgets++
			if h.err != nil {
				c.SetErr(h.err)
				return h.err
			}
			values := make([]interface{}, len(c.Args())-1)
			for i, key := range c.Args()[1:] {
				values[i] = h.values[fmt.Sprint(key)]
			}
			c.SetVal(values)
		case *redis.StringSliceCmd:
			c.SetVal([]string{})
		}
		return nil
	}
}
func (*healthRedisHook) ProcessPipelineHook(_ redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(context.Context, []redis.Cmder) error { return nil }
}
func attachHealthRedis(t *testing.T, r *Reader, h *healthRedisHook) {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: "unused", MaxRetries: -1})
	client.AddHook(h)
	t.Cleanup(func() { _ = client.Close() })
	r.redis = client
}

func TestProviderHealthRedisCodeZeroWinsOverLegacyName(t *testing.T) {
	r, _, now := providerHealthReader(t)
	minute := now.Truncate(time.Minute).Add(-time.Minute).Unix() / 60
	h := &healthRedisHook{values: map[string]interface{}{
		gatewaycontract.Keys.Status.Provider("prov-a", minute, gatewaycontract.MetricSuccess):     "0",
		gatewaycontract.Keys.Status.Provider("prov-a", minute, gatewaycontract.MetricFailure):     "0",
		gatewaycontract.Keys.Status.Provider("Provider A", minute, gatewaycontract.MetricSuccess): "99",
	}}
	attachHealthRedis(t, r, h)
	p := readProviderHealth(t, r, "prov-a")
	require.Equal(t, "redis", p.Calls.Source)
	require.Equal(t, "provider_code", p.Calls.Identity)
	require.Equal(t, &ProviderCalls{}, p.Calls.Value)
	require.Nil(t, p.SuccessRate.Value)
	require.Equal(t, "zero_denominator", p.SuccessRate.Reason)
}

func TestProviderHealthRedisMissingKeysAreNotObservedZero(t *testing.T) {
	r, _, _ := providerHealthReader(t)
	attachHealthRedis(t, r, &healthRedisHook{})
	p := readProviderHealth(t, r, "prov-a")
	require.Equal(t, "redis", p.Calls.Source)
	require.Equal(t, "no_samples", p.Calls.State)
	require.Equal(t, "no_source_samples", p.Calls.Reason)
	require.Nil(t, p.Calls.Value)
	require.Nil(t, p.SuccessRate.Value)
}

func TestProviderHealthRedisFailureKeepsMemoryValuesWithReason(t *testing.T) {
	r, store, now := providerHealthReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Success: false, Attempts: []healthAttempt{{ProviderCode: "prov-a", Success: false}}})
	attachHealthRedis(t, r, &healthRedisHook{err: errors.New("redis offline")})
	p := readProviderHealth(t, r, "prov-a")
	require.Equal(t, "memory", p.Calls.Source)
	require.Equal(t, "partial", p.Calls.State)
	require.Equal(t, "redis_unavailable", p.Calls.Reason)
	require.Equal(t, &ProviderCalls{Total: 1, Failure: 1}, p.Calls.Value)
	require.NotNil(t, p.SuccessRate.Value)
	require.Equal(t, 0.0, *p.SuccessRate.Value)
}

func TestProviderHealthHistoricalWindowDoesNotReadExpiredRedisCounts(t *testing.T) {
	r, store, now := providerHealthReader(t)
	at := now.Add(-4 * time.Hour)
	store.Record(metrics.RequestMetric{Time: at.Add(-time.Minute).Unix(), Success: true, Attempts: []healthAttempt{{ProviderCode: "prov-a", Success: true}}})
	h := &healthRedisHook{}
	attachHealthRedis(t, r, h)
	s, err := r.Read(context.Background(), Query{At: &at, Providers: &ProviderQuery{Provider: "prov-a"}})
	require.NoError(t, err)
	require.NotNil(t, s.Providers.Data)
	require.Equal(t, "memory", (*s.Providers.Data)[0].Calls.Source)
	require.Equal(t, int64(1), (*s.Providers.Data)[0].Calls.Value.Total)
	require.Equal(t, 0, h.mgets)
}

func configureProviderPrometheus(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	old := config.C.Util.PrometheusServer.Address
	config.C.Util.PrometheusServer.Address = srv.URL
	t.Cleanup(func() { config.C.Util.PrometheusServer.Address = old })
}

func writeHealthPromVector(w http.ResponseWriter, provider, value string) {
	metric := map[string]string{}
	if provider != "" {
		metric["provider"] = provider
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "data": map[string]interface{}{"resultType": "vector", "result": []interface{}{map[string]interface{}{"metric": metric, "value": []interface{}{time.Now().Unix(), value}}}}})
}

func writeHealthPromCoverage(w http.ResponseWriter, req *http.Request) {
	start, _ := strconv.ParseInt(req.URL.Query().Get("start"), 10, 64)
	end, _ := strconv.ParseInt(req.URL.Query().Get("end"), 10, 64)
	values := [][]interface{}{}
	for at := start; at <= end; at += 60 {
		values = append(values, []interface{}{at, "1"})
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "data": map[string]interface{}{"resultType": "matrix", "result": []interface{}{map[string]interface{}{"metric": map[string]string{}, "values": values}}}})
}

func TestProviderHealthPrometheusFailureAndNaNStayLocal(t *testing.T) {
	r, store, now := providerHealthReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Success: false, Attempts: []healthAttempt{{ProviderCode: "prov-a", Success: false}}})
	configureProviderPrometheus(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/-/healthy" {
			return
		}
		if req.URL.Path == "/api/v1/query_range" {
			writeHealthPromCoverage(w, req)
			return
		}
		query := req.URL.Query().Get("query")
		if strings.Contains(query, mRequestTotal) {
			t.Errorf("final Request counts cannot measure upstream calls: %s", query)
		}
		switch {
		case strings.Contains(query, mTtftSum):
			writeHealthPromVector(w, "prov-a", "NaN")
		case strings.Contains(query, mTokensTotal):
			writeHealthPromVector(w, "prov-a", "0")
		case strings.Contains(query, mCostTotal):
			writeHealthPromVector(w, "prov-a", "0")
		default:
			http.Error(w, "latency unavailable", http.StatusServiceUnavailable)
		}
	})
	p := readProviderHealth(t, r, "prov-a")
	require.Equal(t, &ProviderCalls{Total: 1, Failure: 1}, p.Calls.Value)
	require.Equal(t, "memory", p.Calls.Source)
	require.Equal(t, "error", p.AvgLatencyMs.State)
	require.Nil(t, p.AvgLatencyMs.Value)
	require.Equal(t, "no_samples", p.AvgTTFTMs.State)
	require.Nil(t, p.AvgTTFTMs.Value)
	require.NotNil(t, p.TotalTokens.Value)
	require.Equal(t, int64(0), *p.TotalTokens.Value)
	require.Equal(t, "prometheus", p.TotalTokens.Source)
	require.Equal(t, "final_provider_requests", p.TotalTokens.Scope)
	require.NotNil(t, p.TotalCost.Value)
	require.Equal(t, 0.0, *p.TotalCost.Value)
}

func TestProviderHealthPrometheusMissingDimensionIsUnsupported(t *testing.T) {
	r, _, _ := providerHealthReader(t)
	configureProviderPrometheus(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/-/healthy" {
			return
		}
		if req.URL.Path == "/api/v1/query_range" {
			writeHealthPromCoverage(w, req)
			return
		}
		writeHealthPromVector(w, "", "2")
	})
	p := readProviderHealth(t, r, "prov-a")
	require.Equal(t, "unsupported", p.AvgLatencyMs.State)
	require.Equal(t, "provider_dimension_unavailable", p.AvgLatencyMs.Reason)
	require.Nil(t, p.AvgLatencyMs.Value)
	require.NotNil(t, p.Calls.Value)
}

func TestProviderHealthPrometheusChoosesOneIdentityIncludingZero(t *testing.T) {
	r, _, _ := providerHealthReader(t)
	configureProviderPrometheus(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/-/healthy" {
			return
		}
		if req.URL.Path == "/api/v1/query_range" {
			writeHealthPromCoverage(w, req)
			return
		}
		query := req.URL.Query().Get("query")
		if strings.Contains(query, mTtftSum) {
			writeHealthPromVector(w, "Provider A", "0.2")
			return
		}
		writeHealthPromVector(w, "prov-a", "0")
	})
	p := readProviderHealth(t, r, "prov-a")
	require.NotNil(t, p.AvgLatencyMs.Value)
	require.Equal(t, 0.0, *p.AvgLatencyMs.Value)
	require.Equal(t, "provider_code", p.AvgLatencyMs.Identity)
	require.Nil(t, p.AvgTTFTMs.Value, "a selected code identity must not borrow a denominator or value from the legacy name")
	require.Equal(t, "no_samples", p.AvgTTFTMs.State)
}

func TestProviderHealthInventoryFailureDoesNotDiscardRequestPerformance(t *testing.T) {
	r, store, now := providerHealthReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Provider: "prov-a", Success: true, DurationMs: 1500})
	require.NoError(t, r.db.Migrator().DropTable(&rschema.Endpoint{}))
	p := readProviderHealth(t, r, "prov-a")
	require.Equal(t, "error", p.EndpointCount.State)
	require.Nil(t, p.EndpointCount.Value)
	require.Equal(t, "error", p.OpenBreakerCount.State)
	require.NotNil(t, p.AvgLatencyMs.Value)
	require.Equal(t, 1500.0, *p.AvgLatencyMs.Value)
}

func TestProviderHealthSortsMissingLatencyAfterMeasuredLatency(t *testing.T) {
	r, store, now := providerHealthReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Success: true, DurationMs: 1500, Attempts: []healthAttempt{
		{ProviderCode: "prov-a", Success: false}, {ProviderCode: "prov-b", Success: true},
	}})
	s, err := r.Read(context.Background(), Query{Providers: &ProviderQuery{Sort: "avg_latency"}})
	require.NoError(t, err)
	require.NotNil(t, s.Providers.Data)
	require.Len(t, *s.Providers.Data, 2)
	require.Equal(t, "prov-b", (*s.Providers.Data)[0].ProviderCode)
	require.Equal(t, "prov-a", (*s.Providers.Data)[1].ProviderCode)
	require.Nil(t, (*s.Providers.Data)[1].AvgLatencyMs.Value)
}

func TestProviderHealthLegacyRequestKeepsPerformanceButCannotCountAttempts(t *testing.T) {
	r, store, now := providerHealthReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Provider: "prov-a", Success: false, DurationMs: 1500, TTFTMs: 100, InputTokens: 10, OutputTokens: 20, Cost: 0.25})
	p := readProviderHealth(t, r, "prov-a")
	require.Equal(t, "no_samples", p.Calls.State)
	require.Equal(t, "upstream_samples_unavailable", p.Calls.Reason)
	require.Nil(t, p.Calls.Value)
	require.Nil(t, p.SuccessRate.Value)
	require.Equal(t, "attributed_provider_requests", p.AvgLatencyMs.Scope)
	require.NotNil(t, p.AvgLatencyMs.Value)
	require.Equal(t, 1500.0, *p.AvgLatencyMs.Value)
	require.NotNil(t, p.AvgTTFTMs.Value)
	require.Equal(t, 100.0, *p.AvgTTFTMs.Value)
	require.NotNil(t, p.TotalTokens.Value)
	require.Equal(t, int64(30), *p.TotalTokens.Value)
	require.Equal(t, "attributed_provider_requests", p.TotalTokens.Scope)
	require.NotNil(t, p.TotalCost.Value)
	require.Equal(t, 0.25, *p.TotalCost.Value)
}

func TestProviderHealthMixedLegacyAndAttemptSamplesKeepKnownCounts(t *testing.T) {
	r, store, now := providerHealthReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Provider: "prov-a", Success: false, DurationMs: 2000, InputTokens: 10, Cost: 0.25})
	store.Record(metrics.RequestMetric{Time: now.Add(-2 * time.Minute).Unix(), Success: true, DurationMs: 1000, InputTokens: 20, Cost: 0.5, Attempts: []healthAttempt{{ProviderCode: "prov-a", Success: true}}})
	p := readProviderHealth(t, r, "prov-a")
	require.Equal(t, "partial", p.Calls.State)
	require.Equal(t, "upstream_samples_unavailable", p.Calls.Reason)
	require.Equal(t, &ProviderCalls{Total: 1, Success: 1}, p.Calls.Value)
	require.Equal(t, "partial", p.SuccessRate.State)
	require.NotNil(t, p.SuccessRate.Value)
	require.Equal(t, 100.0, *p.SuccessRate.Value)
	require.Equal(t, "attributed_provider_requests", p.TotalTokens.Scope)
	require.NotNil(t, p.TotalTokens.Value)
	require.Equal(t, int64(30), *p.TotalTokens.Value)
	require.NotNil(t, p.TotalCost.Value)
	require.Equal(t, 0.75, *p.TotalCost.Value)
}

func TestProviderHealthRetainsMeasuredValuesWhenCoverageProbeFails(t *testing.T) {
	r, _, _ := providerHealthReader(t)
	configureProviderPrometheus(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/-/healthy" {
			return
		}
		if req.URL.Path == "/api/v1/query_range" {
			http.Error(w, "probe failed", http.StatusServiceUnavailable)
			return
		}
		writeHealthPromVector(w, "prov-a", "2")
	})
	p := readProviderHealth(t, r, "prov-a")
	require.Equal(t, "partial", p.AvgLatencyMs.State)
	require.Equal(t, "coverage_unverified", p.AvgLatencyMs.Reason)
	require.Equal(t, 2000.0, *p.AvgLatencyMs.Value)
	require.Equal(t, "ok", p.Calls.State)
}

func TestProviderHealthDoesNotCacheFailedMeasurements(t *testing.T) {
	r, _, _ := providerHealthReader(t)
	var failed atomic.Bool
	failed.Store(true)
	configureProviderPrometheus(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/-/healthy" {
			return
		}
		if req.URL.Path == "/api/v1/query_range" {
			writeHealthPromCoverage(w, req)
			return
		}
		if failed.Load() && strings.Contains(req.URL.Query().Get("query"), mCostTotal) {
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		writeHealthPromVector(w, "prov-a", "0")
	})
	first := readProviderHealth(t, r, "prov-a")
	require.Equal(t, "error", first.TotalCost.State)
	require.Nil(t, first.TotalCost.Value)
	failed.Store(false)
	recovered := readProviderHealth(t, r, "prov-a")
	require.Equal(t, "ok", recovered.TotalCost.State)
	require.NotNil(t, recovered.TotalCost.Value)
	require.Equal(t, 0.0, *recovered.TotalCost.Value)
}
