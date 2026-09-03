package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryStoreRecordsEndpointPerfOnWinningEndpointOnly(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()
	store.Record(RequestMetric{
		Time:         now.Unix(),
		Model:        "gpt-4",
		Success:      true,
		OutputTokens: 40,
		EndpointID:   "ep-2",
		TTFTMs:       180,
		DurationMs:   2000,
		Attempts: []struct {
			EndpointID string `json:"endpoint_id"`
			Success    bool   `json:"success"`
		}{
			{EndpointID: "ep-1", Success: false},
			{EndpointID: "ep-2", Success: true},
		},
	})

	minute := now.Unix() / 60
	succ, fail := store.GetEndpointStatus("ep-1", minute)
	assert.Equal(t, int64(0), succ)
	assert.Equal(t, int64(1), fail)
	retry := store.GetEndpointMinutePerf("ep-1", minute)
	assert.Equal(t, int64(0), retry.Output)
	assert.Equal(t, int64(0), retry.TTFTSum)

	winner := store.GetEndpointMinutePerf("ep-2", minute)
	assert.Equal(t, int64(1), winner.Success)
	assert.Equal(t, int64(40), winner.Output)
	assert.Equal(t, int64(180), winner.TTFTSum)
	assert.Equal(t, int64(1), winner.TTFTCount)
	assert.Equal(t, int64(2000), winner.DurationMs)

	modelPerf := store.GetModelMinutePerf("gpt-4", minute)
	assert.Equal(t, int64(1), modelPerf.Success)
	assert.Equal(t, int64(40), modelPerf.Output)
	assert.Equal(t, int64(180), modelPerf.TTFTSum)
	assert.Equal(t, int64(2000), modelPerf.DurationMs)
}

func TestMemoryStoreRecordsDashboardMetricsForGlobalModelAndProvider(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()
	store.Record(RequestMetric{
		Time:         now.Unix(),
		Model:        "gpt-4",
		Provider:     "openai",
		Success:      true,
		InputTokens:  10,
		OutputTokens: 30,
		Cost:         0.25,
		TTFTMs:       120,
		DurationMs:   2000,
	})

	minute := now.Unix() / 60
	global := store.GetGlobalMinutePerf(minute)
	assert.Equal(t, int64(1), global.Success)
	assert.Equal(t, int64(10), global.InputTokens)
	assert.Equal(t, int64(30), global.OutputTokens)
	assert.Equal(t, int64(30), global.Output)
	assert.Equal(t, 0.25, global.Cost)
	assert.Equal(t, int64(120), global.TTFTSum)
	assert.Equal(t, int64(1), global.TTFTCount)
	assert.Equal(t, int64(2000), global.LatencySumMs)
	assert.Equal(t, int64(1), global.LatencyCount)

	model := store.GetModelMinutePerf("gpt-4", minute)
	assert.Equal(t, global, model)

	provider := store.GetProviderMinutePerf("openai", minute)
	assert.Equal(t, global, provider)
	assert.Equal(t, []string{"gpt-4"}, store.GetModelCodes())
	assert.Equal(t, []string{"openai"}, store.GetProviderNames())
}

func TestMemoryStoreRetainsSevenDaysOfMinuteMetrics(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()
	sixDaysAgo := now.Add(-6 * 24 * time.Hour)

	store.Record(RequestMetric{
		Time:     sixDaysAgo.Unix(),
		Model:    "gpt-4",
		Provider: "openai",
		Success:  true,
	})
	store.Record(RequestMetric{
		Time:     now.Unix(),
		Model:    "gpt-4",
		Provider: "openai",
		Success:  true,
	})

	minute := sixDaysAgo.Unix() / 60
	assert.Equal(t, int64(1), store.GetGlobalMinutePerf(minute).Success)
	assert.Equal(t, int64(1), store.GetModelMinutePerf("gpt-4", minute).Success)
	assert.Equal(t, int64(1), store.GetProviderMinutePerf("openai", minute).Success)
}

func TestMemoryStoreKeepsFailedTokensOutOfOtpsTotals(t *testing.T) {
	store := NewMemoryStore()
	now := time.Now()
	store.Record(RequestMetric{
		Time:         now.Unix(),
		Model:        "gpt-4",
		Success:      true,
		InputTokens:  10,
		OutputTokens: 30,
		DurationMs:   3000,
	})
	store.Record(RequestMetric{
		Time:         now.Unix(),
		Model:        "gpt-4",
		Success:      false,
		InputTokens:  20,
		OutputTokens: 90,
		DurationMs:   1000,
	})

	perf := store.GetModelMinutePerf("gpt-4", now.Unix()/60)
	assert.Equal(t, int64(30), perf.InputTokens)
	assert.Equal(t, int64(120), perf.OutputTokens)
	assert.Equal(t, int64(30), perf.Output)
	assert.Equal(t, int64(3000), perf.DurationMs)
}
