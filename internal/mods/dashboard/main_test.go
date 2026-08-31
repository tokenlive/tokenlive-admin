package dashboard

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/api"
)

func TestRegisterV1RoutersIncludesModelPerformanceTrends(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	module := &Dashboard{DashboardAPI: &api.Dashboard{}}

	require.NoError(t, module.RegisterV1Routers(context.Background(), router.Group("/api/v1")))

	found := false
	for _, route := range router.Routes() {
		if route.Method == http.MethodGet && route.Path == "/api/v1/dashboard/model-performance-trends" {
			found = true
			break
		}
	}
	assert.True(t, found)
}
