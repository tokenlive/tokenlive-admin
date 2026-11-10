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
	GroupBy  string `json:"group_by,omitempty"`
	Model    string `json:"model,omitempty"`
	Provider string `json:"provider,omitempty"`
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
	Cutoff      time.Time                    `json:"cutoff"`
	Overview    *Reading[Overview]           `json:"overview,omitempty"`
	Traffic     *Reading[Traffic]            `json:"traffic,omitempty"`
	Performance *Reading[Performance]        `json:"performance,omitempty"`
	Models      *Reading[[]ModelRankingItem] `json:"models,omitempty"`
	Providers   *Reading[[]ProviderHealth]   `json:"providers,omitempty"`
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

// ModelRankingItem reports Request observations independently of upstream calls.
type ModelRankingItem struct {
	ModelID      string                           `json:"model_id"`
	ModelCode    string                           `json:"model_code"`
	ModelName    string                           `json:"model_name"`
	Requests     MetricObservation[RequestCounts] `json:"requests"`
	SuccessRate  MetricObservation[float64]       `json:"success_rate"`
	AvgLatencyMs MetricObservation[float64]       `json:"avg_latency_ms"`
	P50LatencyMs MetricObservation[float64]       `json:"p50_latency_ms"`
	P95LatencyMs MetricObservation[float64]       `json:"p95_latency_ms"`
	P99LatencyMs MetricObservation[float64]       `json:"p99_latency_ms"`
	AvgTTFTMs    MetricObservation[float64]       `json:"avg_ttft_ms"`
	P50TTFTMs    MetricObservation[float64]       `json:"p50_ttft_ms"`
	P95TTFTMs    MetricObservation[float64]       `json:"p95_ttft_ms"`
	P99TTFTMs    MetricObservation[float64]       `json:"p99_ttft_ms"`
	TotalCost    MetricObservation[float64]       `json:"total_cost"`
	Otps         MetricObservation[float64]       `json:"otps"`
	TotalTokens  MetricObservation[int64]         `json:"total_tokens"`
}

// RequestCounts is an atomic partition of final request outcomes in one source.
type RequestCounts struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"`
	Failure int64 `json:"failure"`
}

// MetricObservation keeps a value distinct from the availability of its samples.
// Scope names the population counted or timed; Source names the observation source.
type MetricObservation[T any] struct {
	Value    *T       `json:"value"`
	State    string   `json:"state"`
	Source   string   `json:"source"`
	Scope    string   `json:"scope"`
	Identity string   `json:"identity,omitempty"`
	Window   Window   `json:"window"`
	Coverage Coverage `json:"coverage"`
	Reason   string   `json:"reason,omitempty"`
}

type ProviderCalls struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"`
	Failure int64 `json:"failure"`
}

// ProviderHealth separates upstream attempts from attributed request performance.
type ProviderHealth struct {
	ProviderID       string                           `json:"provider_id"`
	ProviderCode     string                           `json:"provider_code"`
	ProviderName     string                           `json:"provider_name"`
	Calls            MetricObservation[ProviderCalls] `json:"calls"`
	SuccessRate      MetricObservation[float64]       `json:"success_rate"`
	AvgLatencyMs     MetricObservation[float64]       `json:"avg_latency_ms"`
	AvgTTFTMs        MetricObservation[float64]       `json:"avg_ttft_ms"`
	TotalTokens      MetricObservation[int64]         `json:"total_tokens"`
	TotalCost        MetricObservation[float64]       `json:"total_cost"`
	Otps             MetricObservation[float64]       `json:"otps"`
	EndpointCount    MetricObservation[int64]         `json:"endpoint_count"`
	OpenBreakerCount MetricObservation[int64]         `json:"open_breaker_count"`
}
