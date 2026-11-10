package dashboard

import (
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/monitor"
	opsbiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	resourcebiz "github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"gorm.io/gorm"
)

func ProvideMonitor(db *gorm.DB, client *redis.Client) *monitor.Reader {
	return monitor.New(monitor.Dependencies{DB: db, Redis: client})
}

func ProvideDashboard(reader *monitor.Reader, sync *resourcebiz.ConfigRedisSync) *api.Dashboard {
	return &api.Dashboard{Monitor: reader, RedisSync: sync}
}

func ProvideUsageReader() (*dal.ClickHouseReader, func()) {
	return dal.NewClickHouseReader(config.C.Storage.ClickHouse)
}

func ProvideUsageResolver(db *gorm.DB) *biz.IdentityResolver {
	return biz.NewIdentityResolver(db, &opsbiz.PortalUser{}, config.C.Gateway.APIKeyPepper, time.Now)
}

func ProvideUsageService(reader *dal.ClickHouseReader, resolver *biz.IdentityResolver) *biz.APIKeyUsageService {
	return biz.NewAPIKeyUsageService(reader, resolver, time.Now)
}

func ProvideUsageAPI(service *biz.APIKeyUsageService) *api.APIKeyUsage {
	return &api.APIKeyUsage{Service: service}
}

func ProvideUsageTrendsService(reader *dal.ClickHouseReader) *biz.UsageTrendsService {
	return biz.NewUsageTrendsService(reader, time.Now)
}

func ProvideUsageTrendsAPI(service *biz.UsageTrendsService) *api.UsageTrends {
	return &api.UsageTrends{Service: service}
}
