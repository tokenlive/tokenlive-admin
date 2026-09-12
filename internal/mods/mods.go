package mods

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource"
	"github.com/tokenlive/tokenlive-admin/internal/mods/space"
	"github.com/tokenlive/tokenlive-admin/internal/mods/systemversion"
)

const (
	apiPrefix = "/api/"
)

// Collection of wire providers
var Set = wire.NewSet(
	wire.Struct(new(Mods), "*"),
	rbac.Set,
	resource.Set,
	space.Set,
	policy.Set,
	dashboard.Set,
	ops.Set,
	systemversion.Set,
)

type Mods struct {
	RBAC          *rbac.RBAC
	Resource      *resource.Resource
	Space         *space.Space
	Policy        *policy.Policy
	Dashboard     *dashboard.Dashboard
	Ops           *ops.Ops
	SystemVersion *systemversion.SystemVersion
}

func (a *Mods) Init(ctx context.Context) error {
	if err := a.RBAC.Init(ctx); err != nil {
		return err
	}
	if err := a.Resource.Init(ctx); err != nil {
		return err
	}
	if err := a.Space.Init(ctx); err != nil {
		return err
	}
	if err := a.Policy.Init(ctx); err != nil {
		return err
	}
	if err := a.Dashboard.Init(ctx); err != nil {
		return err
	}
	if err := a.Ops.Init(ctx); err != nil {
		return err
	}
	if err := a.SystemVersion.Init(ctx); err != nil {
		return err
	}
	return nil
}

func (a *Mods) RouterPrefixes() []string {
	return []string{
		apiPrefix,
	}
}

func (a *Mods) RegisterRouters(ctx context.Context, e *gin.Engine) error {
	gAPI := e.Group(apiPrefix)
	v1 := gAPI.Group("v1")

	if err := a.RBAC.RegisterV1Routers(ctx, v1); err != nil {
		return err
	}
	if err := a.Resource.RegisterV1Routers(ctx, v1); err != nil {
		return err
	}
	if err := a.Space.RegisterV1Routers(ctx, v1); err != nil {
		return err
	}
	if err := a.Policy.RegisterV1Routers(ctx, v1); err != nil {
		return err
	}
	if err := a.Dashboard.RegisterV1Routers(ctx, v1); err != nil {
		return err
	}
	if err := a.Ops.RegisterV1Routers(ctx, v1); err != nil {
		return err
	}
	if err := a.SystemVersion.RegisterV1Routers(ctx, v1); err != nil {
		return err
	}
	return nil
}

func (a *Mods) Release(ctx context.Context) error {
	if err := a.SystemVersion.Release(ctx); err != nil {
		return err
	}
	if err := a.RBAC.Release(ctx); err != nil {
		return err
	}
	if err := a.Resource.Release(ctx); err != nil {
		return err
	}
	if err := a.Space.Release(ctx); err != nil {
		return err
	}
	if err := a.Policy.Release(ctx); err != nil {
		return err
	}
	if err := a.Dashboard.Release(ctx); err != nil {
		return err
	}
	if err := a.Ops.Release(ctx); err != nil {
		return err
	}
	return nil
}
