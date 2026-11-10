package monitor

import "time"

// Query selects named domain readings; source details are private to Reader.
type Query struct {
	At          *time.Time        `json:"at,omitempty"`
	Timezone    string            `json:"timezone,omitempty"`
	Overview    bool              `json:"overview,omitempty"`
	Traffic     *TrafficQuery     `json:"traffic,omitempty"`
	Performance *PerformanceQuery `json:"performance,omitempty"`
	Models      *ModelQuery       `json:"models,omitempty"`
	Providers   *ProviderQuery    `json:"providers,omitempty"`
}

type WindowQuery struct {
	Range string     `json:"range,omitempty"`
	Start *time.Time `json:"start,omitempty"`
}

type TrafficQuery struct {
	WindowQuery
	GroupBy string `json:"group_by,omitempty"`
	Model   string `json:"model,omitempty"`
}

type PerformanceQuery struct {
	WindowQuery
	Model string `json:"model"`
}

type ModelQuery struct {
	WindowQuery
	Model string `json:"model,omitempty"`
	Sort  string `json:"sort,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

type ProviderQuery struct {
	WindowQuery
	Provider string `json:"provider,omitempty"`
	Sort     string `json:"sort,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type Window struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Timezone string    `json:"timezone"`
}

type Bucket struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type Coverage struct {
	Start  *time.Time `json:"start,omitempty"`
	End    *time.Time `json:"end,omitempty"`
	Reason string     `json:"reason,omitempty"`
}

type Reading[T any] struct {
	State    string            `json:"state"`
	Source   string            `json:"source,omitempty"`
	Window   Window            `json:"window"`
	Windows  map[string]Window `json:"windows,omitempty"`
	Buckets  []Bucket          `json:"buckets,omitempty"`
	Coverage Coverage          `json:"coverage"`
	Data     *T                `json:"data,omitempty"`
	Error    string            `json:"error,omitempty"`
}

type TrafficSeries struct {
	Label   string   `json:"label"`
	Success []*int64 `json:"success"`
	Failure []*int64 `json:"failure"`
	Total   []*int64 `json:"total"`
}

type Traffic struct {
	Series []TrafficSeries `json:"series"`
}

type Performance struct {
	AvgTTFTMs []*float64 `json:"avg_ttft_ms"`
	Otps      []*float64 `json:"otps"`
}

type Snapshot struct {
	Cutoff      time.Time                       `json:"cutoff"`
	Overview    *Reading[Overview]              `json:"overview,omitempty"`
	Traffic     *Reading[Traffic]               `json:"traffic,omitempty"`
	Performance *Reading[Performance]           `json:"performance,omitempty"`
	Models      *Reading[[]ModelRankingItem]    `json:"models,omitempty"`
	Providers   *Reading[[]ProviderRankingItem] `json:"providers,omitempty"`
}

// Overview keeps missing live/daily measurements distinct from a genuine zero.
type Overview struct {
	QPS                      *float64             `json:"qps"`
	DailyRequests            *int64               `json:"daily_requests"`
	DailyPromptTokens        *int64               `json:"daily_prompt_tokens"`
	DailyCompletionTokens    *int64               `json:"daily_completion_tokens"`
	DailyCachedTokens        *int64               `json:"daily_cached_tokens"`
	DailyCacheCreationTokens *int64               `json:"daily_cache_creation_tokens"`
	DailyCost                *float64             `json:"daily_cost"`
	AvgLatencyMs             *float64             `json:"avg_latency_ms"`
	AvgTTFTMs                *float64             `json:"avg_ttft_ms"`
	ActiveCircuitBreakers    []CircuitBreakerInfo `json:"active_circuit_breakers"`
}

type CircuitBreakerInfo struct {
	ID           string `json:"id"`
	Type         string `json:"type"`          // "endpoint" 或 "service"
	Name         string `json:"name"`          // 显示名称
	ModelID      string `json:"model_id"`      // 关联模型 ID
	ModelName    string `json:"model_name"`    // 关联模型名称
	ProviderID   string `json:"provider_id"`   // 关联供应商 ID
	ProviderName string `json:"provider_name"` // 关联供应商名称
	URL          string `json:"url"`           // 关联的 URL 地址
}

type ModelRankingItem struct {
	ModelID      string  `json:"model_id"`
	ModelCode    string  `json:"model_code"`
	ModelName    string  `json:"model_name"`
	RequestCount int64   `json:"request_count"`
	SuccessCount int64   `json:"success_count"`
	FailCount    int64   `json:"fail_count"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	P50LatencyMs float64 `json:"p50_latency_ms"`
	P95LatencyMs float64 `json:"p95_latency_ms"`
	P99LatencyMs float64 `json:"p99_latency_ms"`
	AvgTTFTMs    float64 `json:"avg_ttft_ms"`
	P50TTFTMs    float64 `json:"p50_ttft_ms"`
	P95TTFTMs    float64 `json:"p95_ttft_ms"`
	P99TTFTMs    float64 `json:"p99_ttft_ms"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost"`
	Otps         float64 `json:"otps"`
}

// ProviderRankingItem 单个供应商在给定时间窗口内的聚合指标。
//
// 注意：不含 P50/P95/P99 分位数。内存 store 只累积 sum/count 而无直方图桶，
// 分位数仅 Prometheus 可算；一个在部分部署下静默消失的指标不如不提供。详见 ADR-0008。
type ProviderRankingItem struct {
	ProviderID   string `json:"provider_id"`
	ProviderCode string `json:"provider_code"`
	ProviderName string `json:"provider_name"`
	// UpstreamCallCount 上游调用次数。故障转移产生的多次尝试各自计入对应供应商。
	UpstreamCallCount int64 `json:"upstream_call_count"`
	SuccessCount      int64 `json:"success_count"`
	FailCount         int64 `json:"fail_count"`
	// UpstreamCallSuccessRate 上游调用成功率（百分比）。
	// 与 Request Success Rate 分母不同：故障转移走掉的失败尝试计入此处的失败数。
	UpstreamCallSuccessRate float64 `json:"upstream_call_success_rate"`
	AvgLatencyMs            float64 `json:"avg_latency_ms"`
	AvgTTFTMs               float64 `json:"avg_ttft_ms"`
	TotalTokens             int64   `json:"total_tokens"`
	TotalCost               float64 `json:"total_cost"`
	Otps                    float64 `json:"otps"`
	EndpointCount           int64   `json:"endpoint_count"`
	OpenBreakerCount        int64   `json:"open_breaker_count"`
}
