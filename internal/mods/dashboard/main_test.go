package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/api"
)

func TestSnapshotRoutesAndRetiredQueries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	module := &Dashboard{DashboardAPI: &api.Dashboard{}}
	require.NoError(t, module.RegisterV1Routers(context.Background(), router.Group("/api/v1")))
	routes := map[string]bool{}
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	require.True(t, routes["POST /api/v1/dashboard/snapshot"])
	require.True(t, routes["GET /api/v1/dashboard/circuit-breakers"])
	for _, path := range []string{"qps", "metrics", "overview", "trends", "model-performance-trends", "model-ranking", "provider-ranking"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/"+path, nil))
		require.Equal(t, http.StatusNotFound, w.Code, path)
	}
}

func TestSnapshotMenuPermissionsUseCurrentRoutesInBothLanguages(t *testing.T) {
	for _, file := range []string{"menu.json", "menu_cn.json"} {
		t.Run(file, func(t *testing.T) {
			content, err := os.ReadFile("../../../configs/" + file)
			require.NoError(t, err)
			var menus []struct {
				Children []struct {
					Code      string
					Resources []struct {
						Method string
						Path   string
					}
				}
			}
			require.NoError(t, json.Unmarshal(content, &menus))
			found := false
			for _, parent := range menus {
				for _, menu := range parent.Children {
					if menu.Code != "home" {
						continue
					}
					for _, resource := range menu.Resources {
						if resource.Path == "/api/v1/dashboard/snapshot" && resource.Method == http.MethodPost {
							found = true
						}
						for _, retired := range []string{"qps", "metrics", "overview", "trends", "model-ranking", "provider-ranking", "model-performance-trends"} {
							require.NotEqual(t, "/api/v1/dashboard/"+retired, resource.Path)
						}
					}
				}
			}
			require.True(t, found, "home must grant the shared snapshot query")
		})
	}
}
