package monitor

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	rschema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
)

func TestCircuitBreakersDistinguishesEmptyFromFailedRuntimeRead(t *testing.T) {
	r := New(Dependencies{Memory: metrics.NewMemoryStore()})
	data, err := r.CircuitBreakers(context.Background())
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Empty(t, data)
	r.redis = redis.NewClient(&redis.Options{Addr: "unused"})
	require.NoError(t, r.redis.Close())
	data, err = r.CircuitBreakers(context.Background())
	require.Error(t, err)
	require.Nil(t, data)
}

func TestCircuitBreakersCatalogFailureIsNotAnEmptyHealthyList(t *testing.T) {
	r, store, _ := providerHealthReader(t)
	store.UpdateCircuitBreakers([]string{"ep-a"}, nil)
	require.NoError(t, r.db.Migrator().DropTable(&rschema.Endpoint{}))
	data, err := r.CircuitBreakers(context.Background())
	require.Error(t, err)
	require.Nil(t, data)
}

func TestCircuitBreakersKeepsRuntimeIdentityWithoutCatalogMetadata(t *testing.T) {
	store := metrics.NewMemoryStore()
	store.UpdateCircuitBreakers([]string{"removed-endpoint"}, []string{"service"})
	r := New(Dependencies{Memory: store})
	data, err := r.CircuitBreakers(context.Background())
	require.NoError(t, err)
	require.Contains(t, data, CircuitBreakerInfo{ID: "removed-endpoint", Type: "endpoint", Name: "removed-endpoint"})
	require.Contains(t, data, CircuitBreakerInfo{ID: "service", Type: "service", Name: "service"})
}
