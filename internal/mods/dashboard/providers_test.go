package dashboard

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
)

func TestUsageProviderDisabledAndUnavailableDoNotBlockConstruction(t *testing.T) {
	previous := config.C
	config.C = new(config.Config)
	t.Cleanup(func() { config.C = previous })
	for _, enabled := range []bool{false, true} {
		config.C.Storage.ClickHouse.Enabled = enabled
		reader, cleanup := ProvideUsageReader()
		resolver := ProvideUsageResolver(nil)
		service := ProvideUsageService(reader, resolver)
		handler := ProvideUsageAPI(service)
		require.NotNil(t, handler)
		result, err := service.Query(context.Background(), schema.Query{"today", "tokens", 10})
		if enabled {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, "disabled", result.State)
		}
		cleanup()
	}
}

func TestRegisterV1RoutersIncludesAPIKeyRanking(t *testing.T) {
	router := gin.New()
	module := &Dashboard{DashboardAPI: &api.Dashboard{}, APIKeyUsageAPI: &api.APIKeyUsage{}}
	require.NoError(t, module.RegisterV1Routers(context.Background(), router.Group("/api/v1")))
	found := false
	for _, route := range router.Routes() {
		if route.Method == http.MethodGet && route.Path == "/api/v1/dashboard/api-key-ranking" {
			found = true
		}
	}
	require.True(t, found)
}
