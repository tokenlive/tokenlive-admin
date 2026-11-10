package dashboard

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/api"
)

type Dashboard struct {
	DashboardAPI   *api.Dashboard
	APIKeyUsageAPI *api.APIKeyUsage
	UsageTrendsAPI *api.UsageTrends
}

func (a *Dashboard) Init(ctx context.Context) error {
	return nil
}

func (a *Dashboard) RegisterV1Routers(ctx context.Context, v1 *gin.RouterGroup) error {
	g := v1.Group("dashboard")
	{
		g.POST("snapshot", a.DashboardAPI.QuerySnapshot)
		g.GET("circuit-breakers", a.DashboardAPI.QueryCircuitBreakers)
		g.POST("sync-redis", a.DashboardAPI.SyncRedis)
		g.GET("api-key-ranking", a.APIKeyUsageAPI.Query)
		g.GET("usage-trends", a.UsageTrendsAPI.Query)
		g.GET("ws", a.DashboardAPI.HandleWebSocket)
	}
	return nil
}

func (a *Dashboard) Release(ctx context.Context) error {
	return nil
}
