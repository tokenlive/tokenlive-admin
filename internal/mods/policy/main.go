package policy

import (
	"context"
	"strings"

	"github.com/tokenlive/tokenlive-admin/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"gorm.io/gorm"
)

type legacyPolicyNameIndex struct {
	table string
	name  string
}

type Policy struct {
	DB                    *gorm.DB
	PolicyLoadbalanceAPI  *api.PolicyLoadbalance
	PolicyRouteAPI        *api.PolicyRoute
	PolicyRouteDetailAPI  *api.PolicyRouteDetail
	PolicyLimitAPI        *api.PolicyLimit
	PolicyCircuitBreakAPI *api.PolicyCircuitBreak
	PolicyInvocationAPI   *api.PolicyInvocation
	PolicyBindingAPI      *api.PolicyBinding
	PolicyTaggingAPI      *api.PolicyTagging
}

func (a *Policy) AutoMigrate(ctx context.Context) error {
	return a.DB.AutoMigrate(new(schema.PolicyBinding), new(schema.PolicyLoadbalance), new(schema.PolicyRoute), new(schema.PolicyRouteDetail), new(schema.PolicyLimit), new(schema.PolicyCircuitBreak), new(schema.PolicyInvocation), new(schema.PolicyTagging))
}

func (a *Policy) dropLegacyPolicyNameIndexes() error {
	legacyIndexes := []legacyPolicyNameIndex{
		{table: config.C.FormatTableName("policy_loadbalance"), name: "uniq_policy_loadbalance_name"},
		{table: config.C.FormatTableName("policy_route"), name: "uniq_policy_route_name"},
		{table: config.C.FormatTableName("policy_limit"), name: "uniq_policy_limit_name"},
		{table: config.C.FormatTableName("policy_invocation"), name: "uniq_policy_invocation_name"},
		{table: config.C.FormatTableName("policy_circuit_break"), name: "uniq_policy_circuit_break_name"},
		{table: config.C.FormatTableName("policy_tagging"), name: "uniq_policy_tagging_name"},
	}
	for _, item := range legacyIndexes {
		if err := a.dropLegacyPolicyNameIndex(item); err != nil {
			return err
		}
	}
	return nil
}

func (a *Policy) dropLegacyPolicyNameIndex(item legacyPolicyNameIndex) error {
	switch a.DB.Dialector.Name() {
	case "mysql":
		var count int64
		if err := a.DB.Raw(
			"SELECT COUNT(1) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?",
			item.table,
			item.name,
		).Scan(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
		return a.DB.Exec("DROP INDEX " + quoteIdentifier("mysql", item.name) + " ON " + quoteIdentifier("mysql", item.table)).Error
	case "sqlite", "postgres":
		return a.DB.Exec("DROP INDEX IF EXISTS " + quoteIdentifier(a.DB.Dialector.Name(), item.name)).Error
	default:
		if a.DB.Migrator().HasIndex(item.table, item.name) {
			return a.DB.Migrator().DropIndex(item.table, item.name)
		}
		return nil
	}
}

func quoteIdentifier(dialect string, name string) string {
	if dialect == "mysql" {
		return "`" + strings.ReplaceAll(name, "`", "``") + "`"
	}
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (a *Policy) Init(ctx context.Context) error {
	if err := a.dropLegacyPolicyNameIndexes(); err != nil {
		return err
	}
	if config.C.Storage.DB.AutoMigrate {
		if err := a.AutoMigrate(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (a *Policy) RegisterV1Routers(ctx context.Context, v1 *gin.RouterGroup) error {
	v1 = v1.Group("policy")
	policyLoadbalance := v1.Group("policy-loadbalances")
	{
		policyLoadbalance.GET("", a.PolicyLoadbalanceAPI.Query)
		policyLoadbalance.GET(":id", a.PolicyLoadbalanceAPI.Get)
		policyLoadbalance.POST("", a.PolicyLoadbalanceAPI.Create)
		policyLoadbalance.POST(":id/copy-to-model", a.PolicyLoadbalanceAPI.CopyToModel)
		policyLoadbalance.PUT(":id", a.PolicyLoadbalanceAPI.Update)
		policyLoadbalance.DELETE(":id", a.PolicyLoadbalanceAPI.Delete)
	}
	policyRoute := v1.Group("policy-routes")
	{
		policyRoute.GET("", a.PolicyRouteAPI.Query)
		policyRoute.GET(":id", a.PolicyRouteAPI.Get)
		policyRoute.POST("", a.PolicyRouteAPI.Create)
		policyRoute.POST(":id/copy-to-model", a.PolicyRouteAPI.CopyToModel)
		policyRoute.PUT(":id", a.PolicyRouteAPI.Update)
		policyRoute.DELETE(":id", a.PolicyRouteAPI.Delete)
	}
	policyRouteDetail := v1.Group("policy-route-details")
	{
		policyRouteDetail.GET("", a.PolicyRouteDetailAPI.Query)
		policyRouteDetail.GET(":id", a.PolicyRouteDetailAPI.Get)
		policyRouteDetail.POST("", a.PolicyRouteDetailAPI.Create)
		policyRouteDetail.PUT(":id", a.PolicyRouteDetailAPI.Update)
		policyRouteDetail.DELETE(":id", a.PolicyRouteDetailAPI.Delete)
	}
	policyLimit := v1.Group("policy-limits")
	{
		policyLimit.GET("", a.PolicyLimitAPI.Query)
		policyLimit.GET(":id", a.PolicyLimitAPI.Get)
		policyLimit.POST("", a.PolicyLimitAPI.Create)
		policyLimit.POST(":id/copy-to-model", a.PolicyLimitAPI.CopyToModel)
		policyLimit.PUT(":id", a.PolicyLimitAPI.Update)
		policyLimit.DELETE(":id", a.PolicyLimitAPI.Delete)
	}
	policyCircuitBreak := v1.Group("policy-circuit-breaks")
	{
		policyCircuitBreak.GET("", a.PolicyCircuitBreakAPI.Query)
		policyCircuitBreak.GET(":id", a.PolicyCircuitBreakAPI.Get)
		policyCircuitBreak.POST("", a.PolicyCircuitBreakAPI.Create)
		policyCircuitBreak.POST(":id/copy-to-model", a.PolicyCircuitBreakAPI.CopyToModel)
		policyCircuitBreak.PUT(":id", a.PolicyCircuitBreakAPI.Update)
		policyCircuitBreak.DELETE(":id", a.PolicyCircuitBreakAPI.Delete)
	}

	policyInvocation := v1.Group("policy-invocations")
	{
		policyInvocation.GET("", a.PolicyInvocationAPI.Query)
		policyInvocation.GET(":id", a.PolicyInvocationAPI.Get)
		policyInvocation.POST("", a.PolicyInvocationAPI.Create)
		policyInvocation.POST(":id/copy-to-model", a.PolicyInvocationAPI.CopyToModel)
		policyInvocation.PUT(":id", a.PolicyInvocationAPI.Update)
		policyInvocation.DELETE(":id", a.PolicyInvocationAPI.Delete)
	}

	policyBinding := v1.Group("policy-bindings")
	{
		policyBinding.GET("", a.PolicyBindingAPI.Query)
		policyBinding.GET(":id", a.PolicyBindingAPI.Get)
		policyBinding.POST("", a.PolicyBindingAPI.Create)
		policyBinding.PUT(":id", a.PolicyBindingAPI.Update)
		policyBinding.DELETE(":id", a.PolicyBindingAPI.Delete)
	}

	policyTagging := v1.Group("policy-taggings")
	{
		policyTagging.GET("", a.PolicyTaggingAPI.Query)
		policyTagging.GET(":id", a.PolicyTaggingAPI.Get)
		policyTagging.POST("", a.PolicyTaggingAPI.Create)
		policyTagging.POST(":id/copy-to-model", a.PolicyTaggingAPI.CopyToModel)
		policyTagging.PUT(":id", a.PolicyTaggingAPI.Update)
		policyTagging.DELETE(":id", a.PolicyTaggingAPI.Delete)
	}
	return nil
}

func (a *Policy) Release(ctx context.Context) error {
	return nil
}
