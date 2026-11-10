package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/biz"
	"github.com/tokenlive/tokenlive-admin/pkg/middleware"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestUsageTrendsUsesDashboardMenuPermission(t *testing.T) {
	const route = "/api/v1/dashboard/usage-trends"
	content, err := os.ReadFile("../../../configs/menu.json")
	require.NoError(t, err)
	var menus []struct {
		Code     string `json:"code"`
		Children []struct {
			Code      string `json:"code"`
			Resources []struct {
				Method string `json:"method"`
				Path   string `json:"path"`
			} `json:"resources"`
		} `json:"children"`
	}
	require.NoError(t, json.Unmarshal(content, &menus))
	found := false
	for _, parent := range menus {
		for _, menu := range parent.Children {
			if parent.Code == "monitor" && menu.Code == "home" {
				for _, resource := range menu.Resources {
					if resource.Method == "GET" && resource.Path == route {
						found = true
					}
				}
			}
		}
	}
	require.True(t, found, "the existing home menu must grant the new endpoint")
	m, err := model.NewModelFromString(`[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub == p.sub && r.obj == p.obj && r.act == p.act`)
	require.NoError(t, err)
	enforcer, err := casbin.NewEnforcer(m)
	require.NoError(t, err)
	_, err = enforcer.AddPolicy("overview-reader", route, "GET")
	require.NoError(t, err)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(util.NewUserID(c.Request.Context(), "ordinary-user"))
		c.Next()
	})
	router.Use(middleware.CasbinWithConfig(middleware.CasbinConfig{
		AllowedPathPrefixes: []string{"/api/"},
		GetEnforcer:         func(*gin.Context) *casbin.Enforcer { return enforcer },
		GetSubjects:         func(c *gin.Context) []string { return []string{c.GetHeader("Test-Role")} },
	}))
	module := &Dashboard{DashboardAPI: &api.Dashboard{}, APIKeyUsageAPI: &api.APIKeyUsage{},
		UsageTrendsAPI: &api.UsageTrends{Service: biz.NewUsageTrendsService(nil, nil)}}
	require.NoError(t, module.RegisterV1Routers(context.Background(), router.Group("/api/v1")))
	for _, tc := range []struct {
		role string
		code int
	}{{"overview-reader", 200}, {"other-role", 403}} {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		req.Header.Set("Test-Role", tc.role)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, tc.code, res.Code)
	}
}
