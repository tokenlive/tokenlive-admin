package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusPointJSONKeepsSuccessFailAndAddsMetrics(t *testing.T) {
	raw, err := json.Marshal(StatusPoint{
		SuccessCount: 3,
		FailCount:    1,
		StartTime:    "10:00",
		EndTime:      "10:10",
		AvgTTFTMs:    180,
		Otps:         34.5,
	})
	assert.NoError(t, err)

	var payload map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &payload))
	assert.Equal(t, float64(3), payload["success_count"])
	assert.Equal(t, float64(1), payload["fail_count"])
	assert.Equal(t, "10:00", payload["start_time"])
	assert.Equal(t, 180.0, payload["avg_ttft_ms"])
	assert.Equal(t, 34.5, payload["otps"])
}

func TestAggregateEndpointStatusPointComputesTTFTAndRequestPeriodOTPS(t *testing.T) {
	point := AggregateEndpointStatusPoint([]EndpointMinutePerf{
		{Success: 2, Fail: 1, TTFTSum: 200, TTFTCount: 2, Output: 30, DurationMs: 1000},
		{Success: 1, Fail: 0, TTFTSum: 100, TTFTCount: 1, Output: 10, DurationMs: 1000},
	}, "10:00", "10:10")

	assert.Equal(t, int64(3), point.SuccessCount)
	assert.Equal(t, int64(1), point.FailCount)
	assert.Equal(t, 100.0, point.AvgTTFTMs)
	assert.Equal(t, 20.0, point.Otps)
	assert.Equal(t, "10:00", point.StartTime)
}

func TestAggregateEndpointStatusPointLeavesMetricsZeroWithoutSamples(t *testing.T) {
	point := AggregateEndpointStatusPoint([]EndpointMinutePerf{
		{Success: 4, Fail: 0},
	}, "11:00", "11:10")

	assert.Equal(t, int64(4), point.SuccessCount)
	assert.Equal(t, 0.0, point.AvgTTFTMs)
	assert.Equal(t, 0.0, point.Otps)
}
