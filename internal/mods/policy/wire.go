package policy

import (
	"github.com/google/wire"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
)

var Set = wire.NewSet(
	wire.Struct(new(Policy), "*"),
	wire.Struct(new(dal.PolicyLoadbalance), "*"),
	wire.Struct(new(biz.PolicyLoadbalance), "*"),
	wire.Struct(new(api.PolicyLoadbalance), "*"),
	wire.Struct(new(dal.PolicyRoute), "*"),
	wire.Struct(new(biz.PolicyRoute), "*"),
	wire.Struct(new(api.PolicyRoute), "*"),
	wire.Struct(new(dal.PolicyRouteDetail), "*"),
	wire.Struct(new(biz.PolicyRouteDetail), "*"),
	wire.Struct(new(api.PolicyRouteDetail), "*"),
	wire.Struct(new(dal.PolicyLimit), "*"),
	wire.Struct(new(biz.PolicyLimit), "*"),
	wire.Struct(new(api.PolicyLimit), "*"),
	wire.Struct(new(dal.PolicyCircuitBreak), "*"),
	wire.Struct(new(biz.PolicyCircuitBreak), "*"),
	wire.Struct(new(api.PolicyCircuitBreak), "*"),

	wire.Struct(new(dal.PolicyInvocation), "*"),
	wire.Struct(new(biz.PolicyInvocation), "*"),
	wire.Struct(new(api.PolicyInvocation), "*"),

	wire.Struct(new(dal.PolicyBinding), "*"),
	wire.Struct(new(biz.PolicyBinding), "*"),
	wire.Struct(new(api.PolicyBinding), "*"),

	wire.Struct(new(dal.PolicyTagging), "*"),
	wire.Struct(new(biz.PolicyTagging), "*"),
	wire.Struct(new(api.PolicyTagging), "*"),

	wire.Struct(new(biz.PolicyRedisSync), "*"),
)
