package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
)

func TestProviderStoreKeysPrefersCodeOverName(t *testing.T) {
	// ADR-0009 确立 code 为端到端主标识，name 仅作历史数据兜底。
	keys := providerStoreKeys(&rschema.Provider{Code: "openai", Name: "OpenAI 官方"})
	assert.Equal(t, []string{"openai", "OpenAI 官方"}, keys)
}

func TestProviderStoreKeysSkipsDuplicateWhenCodeEqualsName(t *testing.T) {
	keys := providerStoreKeys(&rschema.Provider{Code: "openai", Name: "openai"})
	assert.Equal(t, []string{"openai"}, keys)
}

func TestProviderStoreKeysOmitsEmptyFields(t *testing.T) {
	assert.Equal(t, []string{"openai"}, providerStoreKeys(&rschema.Provider{Code: "openai"}))
	assert.Equal(t, []string{"OpenAI"}, providerStoreKeys(&rschema.Provider{Name: "OpenAI"}))
	assert.Empty(t, providerStoreKeys(&rschema.Provider{}))
}

func TestPickProviderMetricFallsBackToNextKey(t *testing.T) {
	values := map[string]float64{"OpenAI 官方": 42}
	// code 无数据时回退到 name——网关完成 provider_code 穿透之前写入的历史数据。
	assert.Equal(t, float64(42), pickProviderMetric(values, []string{"openai", "OpenAI 官方"}))
}

func TestPickProviderMetricReturnsZeroWhenNoKeyMatches(t *testing.T) {
	assert.Equal(t, float64(0), pickProviderMetric(map[string]float64{"other": 7}, []string{"openai"}))
}

func TestResolveProviderMetricRangeClampsToSevenDays(t *testing.T) {
	end := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)

	// 超过 7 天的请求被夹取，而非静默返回不完整数据（ADR-0008）。
	promRange, minutes := resolveProviderMetricRange("30d", end)
	assert.Equal(t, "10080m", promRange)
	assert.Equal(t, 7*24*60, minutes)
}

func TestResolveProviderMetricRangeParsesDayUnit(t *testing.T) {
	end := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)

	// time.ParseDuration 不认 "d"，而前端快捷范围里就有 "7d"。
	promRange, minutes := resolveProviderMetricRange("7d", end)
	assert.Equal(t, "10080m", promRange)
	assert.Equal(t, 7*24*60, minutes)
}

func TestResolveProviderMetricRangeHandlesNormalRanges(t *testing.T) {
	end := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)

	promRange, minutes := resolveProviderMetricRange("1h", end)
	assert.Equal(t, "60m", promRange)
	assert.Equal(t, 60, minutes)

	promRange, minutes = resolveProviderMetricRange("24h", end)
	assert.Equal(t, "1440m", promRange)
	assert.Equal(t, 1440, minutes)
}

func TestResolveProviderMetricRangeTodayCountsFromMidnight(t *testing.T) {
	end := time.Date(2026, 9, 13, 2, 30, 0, 0, time.UTC)

	promRange, minutes := resolveProviderMetricRange("today", end)
	assert.Equal(t, "151m", promRange)
	assert.Equal(t, 151, minutes)
}

func TestResolveProviderMetricRangeFallsBackOnGarbage(t *testing.T) {
	end := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)

	promRange, minutes := resolveProviderMetricRange("not-a-duration", end)
	assert.Equal(t, "60m", promRange)
	assert.Equal(t, 60, minutes)
}

func TestFillProviderMetricsFromMemoryUsesUpstreamCallScope(t *testing.T) {
	store := metrics.NewMemoryStore()
	original := metrics.GlobalStore
	metrics.GlobalStore = store
	defer func() { metrics.GlobalStore = original }()

	now := time.Now().Truncate(time.Minute)

	// 一次用户请求：供应商 a 失败后转移到 b 成功。
	// 上游调用口径下 a 记失败、b 记成功——a 的故障不被转移掩盖（ADR-0008）。
	store.Record(metrics.RequestMetric{
		Time:    now.Unix(),
		Model:   "gpt-4",
		Success: true,
		Attempts: []struct {
			EndpointID   string `json:"endpoint_id"`
			Provider     string `json:"provider,omitempty"`
			ProviderCode string `json:"provider_code,omitempty"`
			Success      bool   `json:"success"`
		}{
			{EndpointID: "ep-a", ProviderCode: "prov-a", Success: false},
			{EndpointID: "ep-b", ProviderCode: "prov-b", Success: true},
		},
	})

	items := []ProviderRankingItem{
		{ProviderID: "1", ProviderCode: "prov-a"},
		{ProviderID: "2", ProviderCode: "prov-b"},
	}
	storeKeys := [][]string{{"prov-a"}, {"prov-b"}}

	fillProviderMetricsFromMemory(items, storeKeys, 5, now)

	assert.Equal(t, int64(1), items[0].UpstreamCallCount)
	assert.Equal(t, int64(1), items[0].FailCount)
	assert.Equal(t, float64(0), items[0].UpstreamCallSuccessRate)

	assert.Equal(t, int64(1), items[1].UpstreamCallCount)
	assert.Equal(t, int64(1), items[1].SuccessCount)
	assert.Equal(t, float64(100), items[1].UpstreamCallSuccessRate)
}

func TestFillProviderMetricsFromMemoryFallsBackToNameKey(t *testing.T) {
	store := metrics.NewMemoryStore()
	original := metrics.GlobalStore
	metrics.GlobalStore = store
	defer func() { metrics.GlobalStore = original }()

	now := time.Now().Truncate(time.Minute)

	// 网关完成 provider_code 穿透之前，上报的是名称。
	store.Record(metrics.RequestMetric{
		Time:         now.Unix(),
		Model:        "gpt-4",
		Provider:     "OpenAI 官方",
		Success:      true,
		OutputTokens: 30,
		InputTokens:  10,
		Cost:         0.25,
		TTFTMs:       120,
		DurationMs:   2000,
	})

	items := []ProviderRankingItem{{ProviderID: "1", ProviderCode: "openai"}}
	fillProviderMetricsFromMemory(items, [][]string{{"openai", "OpenAI 官方"}}, 5, now)

	assert.Equal(t, int64(1), items[0].UpstreamCallCount)
	assert.Equal(t, float64(100), items[0].UpstreamCallSuccessRate)
	assert.Equal(t, int64(40), items[0].TotalTokens)
	assert.Equal(t, 0.25, items[0].TotalCost)
	assert.Equal(t, float64(120), items[0].AvgTTFTMs)
	assert.Equal(t, float64(2000), items[0].AvgLatencyMs)
	assert.Equal(t, float64(15), items[0].Otps)
}

func TestFillProviderMetricsFromMemoryCarriesFullKpiSetForAttemptTraffic(t *testing.T) {
	store := metrics.NewMemoryStore()
	original := metrics.GlobalStore
	metrics.GlobalStore = store
	defer func() { metrics.GlobalStore = original }()

	now := time.Now().Truncate(time.Minute)

	// 真实网关上报带 attempts。KPI 卡的 TTFT/Token/费用/延迟必须在这条路径下有值——
	// 若只在无 attempts 的数据上验证，指标归零的缺陷会被静默放过。
	store.Record(metrics.RequestMetric{
		Time:         now.Unix(),
		Model:        "gpt-4",
		Success:      true,
		InputTokens:  10,
		OutputTokens: 30,
		Cost:         0.25,
		TTFTMs:       120,
		DurationMs:   2000,
		Attempts: []struct {
			EndpointID   string `json:"endpoint_id"`
			Provider     string `json:"provider,omitempty"`
			ProviderCode string `json:"provider_code,omitempty"`
			Success      bool   `json:"success"`
		}{
			{EndpointID: "ep-1", ProviderCode: "openai", Success: true},
		},
	})

	items := []ProviderRankingItem{{ProviderID: "1", ProviderCode: "openai"}}
	fillProviderMetricsFromMemory(items, [][]string{{"openai"}}, 5, now)

	assert.Equal(t, int64(1), items[0].UpstreamCallCount)
	assert.Equal(t, float64(100), items[0].UpstreamCallSuccessRate)
	assert.Equal(t, int64(40), items[0].TotalTokens)
	assert.Equal(t, 0.25, items[0].TotalCost)
	assert.Equal(t, float64(120), items[0].AvgTTFTMs)
	assert.Equal(t, float64(2000), items[0].AvgLatencyMs)
	assert.Equal(t, float64(15), items[0].Otps)
}

func TestFillProviderMetricsFromMemoryLeavesZeroWhenNoTraffic(t *testing.T) {
	store := metrics.NewMemoryStore()
	original := metrics.GlobalStore
	metrics.GlobalStore = store
	defer func() { metrics.GlobalStore = original }()

	items := []ProviderRankingItem{{ProviderID: "1", ProviderCode: "quiet"}}
	fillProviderMetricsFromMemory(items, [][]string{{"quiet"}}, 5, time.Now())

	assert.Equal(t, int64(0), items[0].UpstreamCallCount)
	assert.Equal(t, float64(0), items[0].UpstreamCallSuccessRate)
}

func TestSortProviderRankingItemsByRequestCountDescending(t *testing.T) {
	items := []ProviderRankingItem{
		{ProviderCode: "a", UpstreamCallCount: 5},
		{ProviderCode: "b", UpstreamCallCount: 50},
	}
	sortProviderRankingItems(items, "request_count")
	assert.Equal(t, "b", items[0].ProviderCode)
}

func TestSortProviderRankingItemsByLatencyAscending(t *testing.T) {
	items := []ProviderRankingItem{
		{ProviderCode: "slow", AvgLatencyMs: 900},
		{ProviderCode: "fast", AvgLatencyMs: 120},
	}
	sortProviderRankingItems(items, "avg_latency")
	assert.Equal(t, "fast", items[0].ProviderCode)
}

func TestSortProviderRankingItemsBySuccessRateDescending(t *testing.T) {
	items := []ProviderRankingItem{
		{ProviderCode: "flaky", UpstreamCallSuccessRate: 42},
		{ProviderCode: "solid", UpstreamCallSuccessRate: 99},
	}
	sortProviderRankingItems(items, "success_rate")
	assert.Equal(t, "solid", items[0].ProviderCode)
}
