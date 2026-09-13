package biz

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
)

func TestFillProvidersStatusPoints_Redis(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	p := &Provider{
		RedisClient: rdb,
	}

	ctx := context.Background()
	currentMin := time.Now().Unix() / 60

	// Set 5 successes and 2 failures at 5 minutes ago for provider "openai"
	targetMin := currentMin - 5
	s.Set(fmt.Sprintf("aigw:status:provider:openai:%d:s", targetMin), "5")
	s.Set(fmt.Sprintf("aigw:status:provider:openai:%d:f", targetMin), "2")

	providers := []*schema.Provider{
		{
			Code: "openai",
			Name: "OpenAI",
		},
	}

	p.fillProvidersStatusPoints(ctx, providers)

	assert.Len(t, providers[0].StatusPoints, 10)
	// The most recent slice (pIdx = 9) covers [currentMin-9, currentMin]
	lastSlice := providers[0].StatusPoints[9]
	assert.Equal(t, int64(5), lastSlice.SuccessCount)
	assert.Equal(t, int64(2), lastSlice.FailCount)
}

func TestFillProvidersStatusPoints_RedisFallbackToName(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	p := &Provider{
		RedisClient: rdb,
	}

	ctx := context.Background()
	currentMin := time.Now().Unix() / 60

	// Set data under Name "Azure-OpenAI" instead of Code "azure"
	targetMin := currentMin - 3
	s.Set(fmt.Sprintf("aigw:status:provider:Azure-OpenAI:%d:s", targetMin), "8")
	s.Set(fmt.Sprintf("aigw:status:provider:Azure-OpenAI:%d:f", targetMin), "1")

	providers := []*schema.Provider{
		{
			Code: "azure",
			Name: "Azure-OpenAI",
		},
	}

	p.fillProvidersStatusPoints(ctx, providers)

	assert.Len(t, providers[0].StatusPoints, 10)
	lastSlice := providers[0].StatusPoints[9]
	assert.Equal(t, int64(8), lastSlice.SuccessCount)
	assert.Equal(t, int64(1), lastSlice.FailCount)
}

func TestFillProvidersStatusPoints_MemoryStore(t *testing.T) {
	originalStore := metrics.GlobalStore
	metrics.GlobalStore = metrics.NewMemoryStore()
	defer func() { metrics.GlobalStore = originalStore }()

	p := &Provider{
		RedisClient: nil,
	}

	ctx := context.Background()
	now := time.Now().Unix()

	metrics.GlobalStore.Record(metrics.RequestMetric{
		Time:     now,
		Provider: "anthropic",
		Success:  true,
		Attempts: []struct {
			EndpointID   string `json:"endpoint_id"`
			Provider     string `json:"provider,omitempty"`
			ProviderCode string `json:"provider_code,omitempty"`
			Success      bool   `json:"success"`
		}{
			{
				EndpointID:   "ep-claude",
				ProviderCode: "anthropic",
				Success:      true,
			},
		},
	})

	providers := []*schema.Provider{
		{
			Code: "anthropic",
			Name: "Anthropic Official",
		},
	}

	p.fillProvidersStatusPoints(ctx, providers)

	assert.Len(t, providers[0].StatusPoints, 10)
	lastSlice := providers[0].StatusPoints[9]
	assert.Equal(t, int64(1), lastSlice.SuccessCount)
	assert.Equal(t, int64(0), lastSlice.FailCount)
}
