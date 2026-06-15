package policy

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"gorm.io/gorm"
)

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

func (a *Policy) Init(ctx context.Context) error {
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
		policyLoadbalance.PUT(":id", a.PolicyLoadbalanceAPI.Update)
		policyLoadbalance.DELETE(":id", a.PolicyLoadbalanceAPI.Delete)
	}
	policyRoute := v1.Group("policy-routes")
	{
		policyRoute.GET("", a.PolicyRouteAPI.Query)
		policyRoute.GET(":id", a.PolicyRouteAPI.Get)
		policyRoute.POST("", a.PolicyRouteAPI.Create)
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
		policyLimit.PUT(":id", a.PolicyLimitAPI.Update)
		policyLimit.DELETE(":id", a.PolicyLimitAPI.Delete)
	}
	policyCircuitBreak := v1.Group("policy-circuit-breaks")
	{
		policyCircuitBreak.GET("", a.PolicyCircuitBreakAPI.Query)
		policyCircuitBreak.GET(":id", a.PolicyCircuitBreakAPI.Get)
		policyCircuitBreak.POST("", a.PolicyCircuitBreakAPI.Create)
		policyCircuitBreak.PUT(":id", a.PolicyCircuitBreakAPI.Update)
		policyCircuitBreak.DELETE(":id", a.PolicyCircuitBreakAPI.Delete)
	}

	policyInvocation := v1.Group("policy-invocations")
	{
		policyInvocation.GET("", a.PolicyInvocationAPI.Query)
		policyInvocation.GET(":id", a.PolicyInvocationAPI.Get)
		policyInvocation.POST("", a.PolicyInvocationAPI.Create)
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
		policyTagging.PUT(":id", a.PolicyTaggingAPI.Update)
		policyTagging.DELETE(":id", a.PolicyTaggingAPI.Delete)
	}
	return nil
}

func (a *Policy) Release(ctx context.Context) error {
	return nil
}
