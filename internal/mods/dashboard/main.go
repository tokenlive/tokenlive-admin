package dashboard

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/api"
)

type Dashboard struct {
	DashboardAPI   *api.Dashboard
	APIKeyUsageAPI *api.APIKeyUsage
}

func (a *Dashboard) Init(ctx context.Context) error {
	return nil
}

func (a *Dashboard) RegisterV1Routers(ctx context.Context, v1 *gin.RouterGroup) error {
	g := v1.Group("dashboard")
	{
		g.GET("qps", a.DashboardAPI.QueryQPS)
		g.GET("circuit-breakers", a.DashboardAPI.QueryCircuitBreakers)
		g.GET("trends", a.DashboardAPI.QueryTrends)
		g.GET("model-performance-trends", a.DashboardAPI.QueryModelPerformanceTrends)
		g.POST("sync-redis", a.DashboardAPI.SyncRedis)
		g.GET("model-ranking", a.DashboardAPI.QueryModelRanking)
		g.GET("api-key-ranking", a.APIKeyUsageAPI.Query)
		g.GET("provider-ranking", a.DashboardAPI.QueryProviderRanking)
		g.GET("overview", a.DashboardAPI.QueryOverview)
		g.GET("ws", a.DashboardAPI.HandleWebSocket)
	}
	return nil
}

func (a *Dashboard) Release(ctx context.Context) error {
	return nil
}
