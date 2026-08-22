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
