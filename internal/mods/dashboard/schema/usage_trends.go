package schema

import (
	"fmt"
	"net/url"
	"time"
)

type UsageTrendsQuery struct {
	TimeRange string
}

func (q UsageTrendsQuery) Validate() error {
	switch q.TimeRange {
	case "today", "1h", "6h", "24h", "7d":
		return nil
	default:
		return fmt.Errorf("invalid trends time range")
	}
}

func ParseUsageTrendsQuery(values url.Values) (UsageTrendsQuery, error) {
	q := UsageTrendsQuery{TimeRange: "today"}
	for name, items := range values {
		if name != "time_range" || len(items) != 1 || items[0] == "" {
			return q, fmt.Errorf("invalid trends query parameter")
		}
		q.TimeRange = items[0]
	}
	return q, q.Validate()
}

func (q UsageTrendsQuery) Window(end time.Time) Window {
	return ResolveWindow(Query{TimeRange: q.TimeRange}, end.Truncate(time.Millisecond))
}

func (q UsageTrendsQuery) BucketSeconds(window Window) int64 {
	switch q.TimeRange {
	case "1h":
		return 60
	case "6h":
		return 300
	case "24h":
		return 900
	case "7d":
		return 7200
	default:
		for _, seconds := range []int64{60, 300, 600, 900} {
			if window.End.Sub(window.Start) <= time.Duration(120*seconds)*time.Second {
				return seconds
			}
		}
		return 900
	}
}

// UsageTrendsTotals is an internal aggregate; it never contains request identities.
type UsageTrendsTotals struct {
	Requests, Success, Input, Output, Cached, CacheCreation uint64
	LatencySumMs, TTFTSumMs, TTFTCount                      uint64
}

func (t *UsageTrendsTotals) Add(other UsageTrendsTotals) {
	t.Requests += other.Requests
	t.Success += other.Success
	t.Input += other.Input
	t.Output += other.Output
	t.Cached += other.Cached
	t.CacheCreation += other.CacheCreation
	t.LatencySumMs += other.LatencySumMs
	t.TTFTSumMs += other.TTFTSumMs
	t.TTFTCount += other.TTFTCount
}

type UsageTrendsRow struct {
	Index uint64
	UsageTrendsTotals
}

type UsageTrendsMetrics struct {
	Requests      uint64   `json:"requests"`
	Success       uint64   `json:"success"`
	Failure       uint64   `json:"failure"`
	Input         uint64   `json:"input_tokens"`
	Output        uint64   `json:"output_tokens"`
	Cached        uint64   `json:"cached_tokens"`
	CacheCreation uint64   `json:"cache_creation_tokens"`
	TotalTokens   uint64   `json:"total_tokens"`
	FailureRate   *float64 `json:"failure_rate"`
	AvgLatencyMs  *float64 `json:"avg_latency_ms"`
	AvgTTFTMs     *float64 `json:"avg_ttft_ms"`
}

func (t UsageTrendsTotals) Metrics() UsageTrendsMetrics {
	m := UsageTrendsMetrics{
		Requests: t.Requests, Success: t.Success, Failure: t.Requests - t.Success,
		Input: t.Input, Output: t.Output, Cached: t.Cached, CacheCreation: t.CacheCreation,
		TotalTokens: t.Input + t.Output,
	}
	if t.Requests > 0 {
		rate := float64(m.Failure) / float64(t.Requests) * 100
		latency := float64(t.LatencySumMs) / float64(t.Requests)
		m.FailureRate, m.AvgLatencyMs = &rate, &latency
	}
	if t.TTFTCount > 0 {
		ttft := float64(t.TTFTSumMs) / float64(t.TTFTCount)
		m.AvgTTFTMs = &ttft
	}
	return m
}

type UsageTrendsBucket struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	UsageTrendsMetrics
}

type UsageTrendsResponse struct {
	State         string              `json:"state"`
	DataSource    string              `json:"data_source"`
	Window        *Window             `json:"window"`
	GeneratedAt   *time.Time          `json:"generated_at"`
	BucketSeconds int64               `json:"bucket_seconds"`
	Summary       *UsageTrendsMetrics `json:"summary"`
	Buckets       []UsageTrendsBucket `json:"buckets"`
}
