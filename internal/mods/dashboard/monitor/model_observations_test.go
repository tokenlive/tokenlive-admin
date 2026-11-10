package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
)

func modelObservationReader(t *testing.T) (*Reader, *metrics.MemoryStore, time.Time) {
	t.Helper()
	r, store, now := providerHealthReader(t)
	require.NoError(t, r.db.AutoMigrate(&rschema.Model{}))
	require.NoError(t, r.db.Create(&[]rschema.Model{
		{ID: "model-a", ModelCode: "m-a", ModelName: "Model A", SpaceCode: "default", Enabled: 1},
		{ID: "model-b", ModelCode: "m-b", ModelName: "Model B", SpaceCode: "default", Enabled: 1},
	}).Error)
	return r, store, now
}

func readModelObservation(t *testing.T, r *Reader, model string) ModelRankingItem {
	t.Helper()
	s, err := r.Read(context.Background(), Query{Models: &ModelQuery{Model: model}})
	require.NoError(t, err)
	require.NotNil(t, s.Models.Data, "state=%s error=%s", s.Models.State, s.Models.Error)
	require.Len(t, *s.Models.Data, 1)
	return (*s.Models.Data)[0]
}

func TestModelObservationCountsRequestsRatherThanUpstreamAttempts(t *testing.T) {
	r, store, now := modelObservationReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Model: "m-a", Success: true, DurationMs: 2000, TTFTMs: 120, InputTokens: 10, OutputTokens: 30, Cost: 0.25, Attempts: []healthAttempt{{ProviderCode: "a", Success: false}, {ProviderCode: "b", Success: true}}})
	row := readModelObservation(t, r, "m-a")
	require.Equal(t, &RequestCounts{Total: 1, Success: 1}, row.Requests.Value)
	require.Equal(t, "model_requests", row.Requests.Scope)
	require.Equal(t, "memory", row.Requests.Source)
	require.Equal(t, 100.0, *row.SuccessRate.Value)
	require.Equal(t, 2000.0, *row.AvgLatencyMs.Value)
	require.Equal(t, 120.0, *row.AvgTTFTMs.Value)
	require.Equal(t, int64(40), *row.TotalTokens.Value)
	require.Equal(t, 0.25, *row.TotalCost.Value)
	require.Equal(t, 15.0, *row.Otps.Value)
	for _, state := range []string{row.P50LatencyMs.State, row.P95LatencyMs.State, row.P99LatencyMs.State, row.P50TTFTMs.State, row.P95TTFTMs.State, row.P99TTFTMs.State} {
		require.Equal(t, "unsupported", state)
	}
	require.Nil(t, row.P95LatencyMs.Value)
	require.Equal(t, "no_histogram", row.P95LatencyMs.Reason)
}

func TestModelObservationRetainsQuietModelWithoutInventingPerformance(t *testing.T) {
	r, _, _ := modelObservationReader(t)
	row := readModelObservation(t, r, "model-a")
	require.Equal(t, &RequestCounts{}, row.Requests.Value)
	require.Equal(t, "ok", row.Requests.State)
	require.Nil(t, row.SuccessRate.Value)
	require.Equal(t, "no_samples", row.SuccessRate.State)
	require.Equal(t, "zero_denominator", row.SuccessRate.Reason)
	require.Nil(t, row.TotalTokens.Value)
	require.Nil(t, row.TotalCost.Value)
	require.Nil(t, row.AvgLatencyMs.Value)
	require.Equal(t, "no_samples", row.TotalTokens.State)
}

func TestModelObservationKeepsTrueZeroUsageAndZeroSuccessRate(t *testing.T) {
	r, store, now := modelObservationReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Model: "m-a", Success: false, DurationMs: 1000})
	row := readModelObservation(t, r, "m-a")
	require.Equal(t, &RequestCounts{Total: 1, Failure: 1}, row.Requests.Value)
	require.Equal(t, 0.0, *row.SuccessRate.Value)
	require.Equal(t, int64(0), *row.TotalTokens.Value)
	require.Equal(t, 0.0, *row.TotalCost.Value)
	require.Nil(t, row.AvgTTFTMs.Value)
}

func TestModelObservationKeepsTwentyFourHoursOfAvailableMemoryData(t *testing.T) {
	r, store, now := modelObservationReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-23 * time.Hour).Unix(), Model: "m-a", Success: true, DurationMs: 1000, TTFTMs: 100, InputTokens: 10, OutputTokens: 20, Cost: 0.1})
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Model: "m-a", Success: true, DurationMs: 3000, TTFTMs: 300, InputTokens: 30, OutputTokens: 40, Cost: 0.2})
	s, err := r.Read(context.Background(), Query{Models: &ModelQuery{WindowQuery: WindowQuery{Range: "24h"}}})
	require.NoError(t, err)
	require.Len(t, *s.Models.Data, 1)
	row := (*s.Models.Data)[0]
	require.Equal(t, int64(2), row.Requests.Value.Total)
	require.Equal(t, 2000.0, *row.AvgLatencyMs.Value)
	require.Equal(t, 200.0, *row.AvgTTFTMs.Value)
	require.Equal(t, int64(100), *row.TotalTokens.Value)
	require.InDelta(t, 0.3, *row.TotalCost.Value, 0.000001)
	require.Equal(t, 15.0, *row.Otps.Value)
}

func TestModelObservationCacheDoesNotSharePointersOrCoverage(t *testing.T) {
	r, store, now := modelObservationReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Model: "m-a", Success: true, DurationMs: 1000})
	first := readModelObservation(t, r, "m-a")
	first.Requests.Value.Total = 999
	*first.AvgLatencyMs.Value = 999
	*first.Requests.Coverage.Start = time.Time{}
	again := readModelObservation(t, r, "m-a")
	require.Equal(t, int64(1), again.Requests.Value.Total)
	require.Equal(t, 1000.0, *again.AvgLatencyMs.Value)
	require.False(t, again.Requests.Coverage.Start.IsZero())
}

func TestModelObservationRedisZeroIsObservedButMissingKeysAreUnknown(t *testing.T) {
	r, _, now := modelObservationReader(t)
	minute := now.Truncate(time.Minute).Add(-time.Minute).Unix() / 60
	attachHealthRedis(t, r, &healthRedisHook{values: map[string]interface{}{
		gatewaycontract.Keys.Status.Model("m-a", minute, gatewaycontract.MetricSuccess): "0",
	}})
	zero := readModelObservation(t, r, "m-a")
	require.Equal(t, "redis", zero.Requests.Source)
	require.Equal(t, &RequestCounts{}, zero.Requests.Value)
	require.Nil(t, zero.SuccessRate.Value)
	require.Equal(t, "unsupported", zero.TotalCost.State)
	require.Nil(t, zero.TotalCost.Value)
	missing := readModelObservation(t, r, "m-b")
	require.Nil(t, missing.Requests.Value)
	require.Equal(t, "no_samples", missing.Requests.State)
	require.Equal(t, "no_source_samples", missing.Requests.Reason)
}

func TestModelObservationRedisFailurePreservesMemoryCountsAsPartial(t *testing.T) {
	r, store, now := modelObservationReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Model: "m-a", Success: false, DurationMs: 1000})
	attachHealthRedis(t, r, &healthRedisHook{err: errors.New("offline")})
	row := readModelObservation(t, r, "m-a")
	require.Equal(t, "memory", row.Requests.Source)
	require.Equal(t, "partial", row.Requests.State)
	require.Equal(t, "redis_unavailable", row.Requests.Reason)
	require.Equal(t, &RequestCounts{Total: 1, Failure: 1}, row.Requests.Value)
	require.Equal(t, 0.0, *row.SuccessRate.Value)
	require.Equal(t, 1000.0, *row.AvgLatencyMs.Value)
}

func TestModelObservationHistoricalWindowSkipsExpiredRedis(t *testing.T) {
	r, store, now := modelObservationReader(t)
	at := now.Add(-4 * time.Hour)
	store.Record(metrics.RequestMetric{Time: at.Add(-time.Minute).Unix(), Model: "m-a", Success: true})
	hook := &healthRedisHook{}
	attachHealthRedis(t, r, hook)
	s, err := r.Read(context.Background(), Query{At: &at, Models: &ModelQuery{Model: "m-a"}})
	require.NoError(t, err)
	require.NotNil(t, s.Models.Data)
	row := (*s.Models.Data)[0]
	require.Equal(t, "memory", row.Requests.Source)
	require.Equal(t, int64(1), row.Requests.Value.Total)
	require.Zero(t, hook.mgets)
}

func TestModelObservationReportsActualMemoryCoverage(t *testing.T) {
	r, _, now := modelObservationReader(t)
	r.store = metrics.NewMemoryStoreAt(now.Add(-20 * time.Minute))
	r.store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Model: "m-a", Success: true})
	row := readModelObservation(t, r, "m-a")
	require.Equal(t, "partial", row.Requests.State)
	require.Equal(t, "outside_retention", row.Requests.Reason)
	require.True(t, row.Requests.Coverage.Start.After(row.Requests.Window.Start))
	require.Equal(t, int64(1), row.Requests.Value.Total)
}

func modelPromSample(model, status, value string) map[string]interface{} {
	labels := map[string]string{}
	if model != "" {
		labels["model"] = model
	}
	if status != "" {
		labels["status"] = status
	}
	return map[string]interface{}{"metric": labels, "value": []interface{}{time.Now().Unix(), value}}
}
func writeModelPromVector(w http.ResponseWriter, rows ...map[string]interface{}) {
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "data": map[string]interface{}{"resultType": "vector", "result": rows}})
}
func modelPromHandler(t *testing.T, query func(http.ResponseWriter, string)) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/-/healthy" {
			return
		}
		if r.URL.Path == "/api/v1/query_range" {
			writeHealthPromCoverage(w, r)
			return
		}
		query(w, r.URL.Query().Get("query"))
	}
}

func TestModelObservationPrometheusKeepsSingleFailureAndNaNLocal(t *testing.T) {
	r, _, _ := modelObservationReader(t)
	configureProviderPrometheus(t, modelPromHandler(t, func(w http.ResponseWriter, q string) {
		switch {
		case strings.Contains(q, mRequestTotal):
			writeModelPromVector(w, modelPromSample("m-a", "error", "4"))
		case strings.Contains(q, mCostTotal):
			http.Error(w, "private connection detail", http.StatusServiceUnavailable)
		case strings.Contains(q, mTtftSum), strings.Contains(q, mTtftBucket):
			writeModelPromVector(w, modelPromSample("m-a", "", "NaN"))
		default:
			writeModelPromVector(w, modelPromSample("m-a", "", "0"))
		}
	}))
	row := readModelObservation(t, r, "m-a")
	require.Equal(t, &RequestCounts{Total: 4, Failure: 4}, row.Requests.Value)
	require.Equal(t, 0.0, *row.SuccessRate.Value)
	require.Equal(t, "prometheus", row.Requests.Source)
	require.Equal(t, "model_requests", row.AvgLatencyMs.Scope)
	require.Equal(t, 0.0, *row.AvgLatencyMs.Value)
	require.Equal(t, "no_samples", row.AvgTTFTMs.State)
	require.Nil(t, row.AvgTTFTMs.Value)
	require.Equal(t, int64(0), *row.TotalTokens.Value)
	require.Equal(t, "error", row.TotalCost.State)
	require.Nil(t, row.TotalCost.Value)
	require.NotContains(t, row.TotalCost.Reason, "private")
}

func TestModelObservationPrometheusCountNaNIsNotAnObservedZero(t *testing.T) {
	r, _, _ := modelObservationReader(t)
	configureProviderPrometheus(t, modelPromHandler(t, func(w http.ResponseWriter, q string) {
		if strings.Contains(q, mRequestTotal) {
			writeModelPromVector(w, modelPromSample("m-a", "success", "3"), modelPromSample("m-a", "error", "NaN"))
			return
		}
		writeModelPromVector(w, modelPromSample("m-a", "", "1"))
	}))
	row := readModelObservation(t, r, "m-a")
	require.Nil(t, row.Requests.Value)
	require.Equal(t, "no_samples", row.Requests.State)
	require.Nil(t, row.SuccessRate.Value)
	require.Equal(t, 1000.0, *row.AvgLatencyMs.Value)
}

func TestModelObservationPrometheusMissingDimensionIsUnsupported(t *testing.T) {
	r, _, _ := modelObservationReader(t)
	configureProviderPrometheus(t, modelPromHandler(t, func(w http.ResponseWriter, q string) { writeModelPromVector(w, modelPromSample("", "", "2")) }))
	row := readModelObservation(t, r, "m-a")
	require.Equal(t, "unsupported", row.Requests.State)
	require.Equal(t, "model_dimension_unavailable", row.Requests.Reason)
	require.Nil(t, row.Requests.Value)
	require.Equal(t, "unsupported", row.AvgLatencyMs.State)
	require.Nil(t, row.AvgLatencyMs.Value)
}

func TestModelObservationMissingLatencySortsAfterMeasuredLatency(t *testing.T) {
	r, _, _ := modelObservationReader(t)
	configureProviderPrometheus(t, modelPromHandler(t, func(w http.ResponseWriter, q string) {
		if strings.Contains(q, mRequestTotal) {
			writeModelPromVector(w, modelPromSample("m-a", "success", "2"), modelPromSample("m-b", "success", "1"))
			return
		}
		writeModelPromVector(w, modelPromSample("m-b", "", "0.2"))
	}))
	s, err := r.Read(context.Background(), Query{Models: &ModelQuery{Sort: "avg_latency"}})
	require.NoError(t, err)
	require.Len(t, *s.Models.Data, 2)
	require.Equal(t, "m-b", (*s.Models.Data)[0].ModelCode)
	require.Equal(t, "m-a", (*s.Models.Data)[1].ModelCode)
	require.Nil(t, (*s.Models.Data)[1].AvgLatencyMs.Value)
}

func TestModelObservationFailedMeasurementsAreRetriedAfterRecovery(t *testing.T) {
	r, _, _ := modelObservationReader(t)
	var failed atomic.Bool
	failed.Store(true)
	configureProviderPrometheus(t, modelPromHandler(t, func(w http.ResponseWriter, q string) {
		if strings.Contains(q, mRequestTotal) {
			writeModelPromVector(w, modelPromSample("m-a", "success", "1"))
			return
		}
		if failed.Load() && strings.Contains(q, mCostTotal) {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		writeModelPromVector(w, modelPromSample("m-a", "", "0"))
	}))
	first := readModelObservation(t, r, "m-a")
	require.Equal(t, "error", first.TotalCost.State)
	failed.Store(false)
	again := readModelObservation(t, r, "m-a")
	require.Equal(t, "ok", again.TotalCost.State)
	require.Equal(t, 0.0, *again.TotalCost.Value)
}

func TestModelObservationKeepsMeasuredValuesWhenCoverageCannotBeVerified(t *testing.T) {
	r, _, _ := modelObservationReader(t)
	configureProviderPrometheus(t, func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/-/healthy" {
			return
		}
		if req.URL.Path == "/api/v1/query_range" {
			http.Error(w, "coverage unavailable", http.StatusServiceUnavailable)
			return
		}
		if strings.Contains(req.URL.Query().Get("query"), mRequestTotal) {
			writeModelPromVector(w, modelPromSample("m-a", "success", "1"))
			return
		}
		writeModelPromVector(w, modelPromSample("m-a", "", "0"))
	})
	row := readModelObservation(t, r, "m-a")
	require.Equal(t, "partial", row.Requests.State)
	require.Equal(t, "coverage_unverified", row.Requests.Reason)
	require.Equal(t, int64(1), row.Requests.Value.Total)
	require.Equal(t, "partial", row.AvgLatencyMs.State)
	require.Equal(t, 0.0, *row.AvgLatencyMs.Value)
}

func TestModelObservationEmptyPrometheusResultDoesNotImplyUnsupportedOrZero(t *testing.T) {
	r, _, _ := modelObservationReader(t)
	configureProviderPrometheus(t, modelPromHandler(t, func(w http.ResponseWriter, _ string) { writeModelPromVector(w) }))
	row := readModelObservation(t, r, "m-a")
	require.Equal(t, "no_samples", row.Requests.State)
	require.Nil(t, row.Requests.Value)
	require.Equal(t, "no_samples", row.AvgLatencyMs.State)
	require.Nil(t, row.AvgLatencyMs.Value)
}

func TestModelObservationExactSelectionPreservesEnabledModelFilter(t *testing.T) {
	r, store, now := modelObservationReader(t)
	require.NoError(t, r.db.Model(&rschema.Model{}).Where("id = ?", "model-a").Update("enabled", 0).Error)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Model: "m-a", Success: true})
	s, err := r.Read(context.Background(), Query{Models: &ModelQuery{Model: "m-a"}})
	require.NoError(t, err)
	require.NotNil(t, s.Models.Data)
	require.Empty(t, *s.Models.Data)
}

func TestModelObservationMissingTTFTWithRequestSamplesIsNotRedisUnsupported(t *testing.T) {
	r, store, now := modelObservationReader(t)
	store.Record(metrics.RequestMetric{Time: now.Add(-time.Minute).Unix(), Model: "m-a", Success: true, DurationMs: 1000})
	minute := now.Truncate(time.Minute).Add(-time.Minute).Unix() / 60
	attachHealthRedis(t, r, &healthRedisHook{values: map[string]interface{}{
		gatewaycontract.Keys.Status.Model("m-a", minute, gatewaycontract.MetricSuccess): "1",
	}})
	row := readModelObservation(t, r, "m-a")
	require.Equal(t, "redis", row.Requests.Source)
	require.Equal(t, "memory", row.AvgLatencyMs.Source)
	require.Equal(t, 1000.0, *row.AvgLatencyMs.Value)
	require.Equal(t, "no_samples", row.AvgTTFTMs.State)
	require.Equal(t, "memory", row.AvgTTFTMs.Source)
	require.Nil(t, row.AvgTTFTMs.Value)
}

func TestModelObservationUnsupportedRedisPerformanceCarriesRedisCoverage(t *testing.T) {
	r, _, now := modelObservationReader(t)
	minute := now.Truncate(time.Minute).Add(-time.Minute).Unix() / 60
	attachHealthRedis(t, r, &healthRedisHook{values: map[string]interface{}{
		gatewaycontract.Keys.Status.Model("m-a", minute, gatewaycontract.MetricSuccess): "1",
	}})
	s, err := r.Read(context.Background(), Query{Models: &ModelQuery{Model: "m-a", WindowQuery: WindowQuery{Range: "6h"}}})
	require.NoError(t, err)
	require.NotNil(t, s.Models.Data)
	row := (*s.Models.Data)[0]
	require.Equal(t, "unsupported", row.TotalCost.State)
	require.Equal(t, "redis", row.TotalCost.Source)
	require.Equal(t, row.Requests.Coverage, row.TotalCost.Coverage)
	require.True(t, row.TotalCost.Coverage.Start.After(row.TotalCost.Window.Start))
}
