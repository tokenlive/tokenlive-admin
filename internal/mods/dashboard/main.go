package dashboard

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/api"
)

type Dashboard struct {
	DashboardAPI *api.Dashboard
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
		g.POST("sync-redis", a.DashboardAPI.SyncRedis)
		g.GET("model-ranking", a.DashboardAPI.QueryModelRanking)
		g.GET("overview", a.DashboardAPI.QueryOverview)
	}
	return nil
}

func (a *Dashboard) Release(ctx context.Context) error {
	return nil
}
