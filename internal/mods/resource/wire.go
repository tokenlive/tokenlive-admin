package resource

import (
	"github.com/google/wire"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
)

// Collection of wire providers
var Set = wire.NewSet(
	wire.Struct(new(Resource), "*"),
	wire.Struct(new(dal.Provider), "*"),
	wire.Struct(new(biz.Provider), "*"),
	wire.Struct(new(api.Provider), "*"),
	wire.Struct(new(dal.Endpoint), "*"),
	wire.Struct(new(biz.Endpoint), "*"),
	wire.Struct(new(api.Endpoint), "*"),
	wire.Struct(new(dal.Model), "*"),
	wire.Struct(new(biz.Model), "*"),
	wire.Struct(new(api.Model), "*"),
	wire.Struct(new(biz.ConfigRedisSync), "*"),
	wire.Struct(new(dal.ModelAlias), "*"),
	wire.Struct(new(biz.ModelAlias), "*"),
	wire.Struct(new(api.ModelAlias), "*"),
	wire.Struct(new(dal.DataPermission), "*"),
	wire.Struct(new(biz.DataPermission), "*"),
	wire.Struct(new(api.DataPermission), "*"),
	// Model Catalog
	wire.Struct(new(dal.ModelCatalog), "*"),
	wire.Struct(new(biz.ModelCatalog), "*"),
	wire.Struct(new(api.ModelCatalog), "ModelCatalogBIZ"),
	// Model Catalog I18n
	wire.Struct(new(dal.ModelCatalogI18n), "*"),
	wire.Struct(new(biz.ModelCatalogI18n), "*"),
	wire.Struct(new(api.ModelCatalogI18n), "*"),
	// Model Price Version
	wire.Struct(new(dal.ModelPriceVersion), "*"),
	wire.Struct(new(biz.ModelPriceVersion), "*"),
	wire.Struct(new(api.ModelPriceVersion), "*"),
	// Gateway Pull Sync
	wire.Struct(new(biz.GatewaySync), "*"),
	wire.Struct(new(api.GatewaySync), "*"),
)
