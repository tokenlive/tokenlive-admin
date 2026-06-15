package rbac

import (
	"github.com/google/wire"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/dal"
)

// Collection of wire providers
var Set = wire.NewSet(
	wire.Struct(new(RBAC), "*"),
	wire.Struct(new(Casbinx), "*"),
	wire.Struct(new(dal.Menu), "*"),
	wire.Struct(new(biz.Menu), "*"),
	wire.Struct(new(api.Menu), "*"),
	wire.Struct(new(dal.MenuResource), "*"),
	wire.Struct(new(dal.Role), "*"),
	wire.Struct(new(biz.Role), "*"),
	wire.Struct(new(api.Role), "*"),
	wire.Struct(new(dal.RoleMenu), "*"),
	wire.Struct(new(dal.User), "*"),
	wire.Struct(new(biz.User), "*"),
	wire.Struct(new(api.User), "*"),
	wire.Struct(new(dal.UserRole), "*"),
	wire.Struct(new(biz.Login), "*"),
	wire.Struct(new(api.Login), "*"),
	wire.Struct(new(api.Logger), "*"),
	wire.Struct(new(biz.Logger), "*"),
	wire.Struct(new(dal.Logger), "*"),
	biz.ProvideRedisClient,
	wire.Struct(new(dal.UserAPIKey), "*"),
	wire.Struct(new(biz.UserAPIKey), "*"),
	wire.Struct(new(api.UserAPIKey), "*"),
	wire.Struct(new(dal.Tenant), "*"),
	wire.Struct(new(biz.Tenant), "*"),
	wire.Struct(new(api.Tenant), "*"),
	wire.Struct(new(dal.TenantModel), "*"),
	wire.Struct(new(biz.TenantModel), "*"),
	wire.Struct(new(api.TenantModel), "*"),
	wire.Struct(new(dal.TenantModelProvider), "*"),
	wire.Struct(new(biz.TenantModelProvider), "*"),
	wire.Struct(new(api.TenantModelProvider), "*"),
	wire.Struct(new(dal.TenantEndpoint), "*"),
	wire.Struct(new(biz.TenantEndpoint), "*"),
	wire.Struct(new(api.TenantEndpoint), "*"),
)
