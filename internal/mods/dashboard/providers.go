package dashboard

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/dal"
	opsbiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	"gorm.io/gorm"
)

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
