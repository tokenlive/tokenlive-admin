package monitor

import (
	"time"
)

type trendSeries struct {
	Label   string  `json:"label"`
	Success []int64 `json:"success"`
	Failure []int64 `json:"failure"`
	Total   []int64 `json:"total"`
}

type trafficResult struct {
	Series []trendSeries `json:"series"`
}

type performanceResult struct {
	AvgTTFTMs []*float64 `json:"avg_ttft_ms"`
	Otps      []*float64 `json:"otps"`
}

type trendRangeConfig struct {
	plan         *windowPlan
	minutes      int
	numPoints    int
	stepSeconds  int64
	redisMinutes int
	end          time.Time
}
