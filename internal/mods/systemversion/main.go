package systemversion

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac"
	"github.com/tokenlive/tokenlive-admin/internal/versionstatus"
)

type SystemVersion struct {
	Service *versionstatus.Service
	RBAC    *rbac.RBAC
}

func (a *SystemVersion) Init(ctx context.Context) error {
	a.Service.Start()
	return nil
}

func (a *SystemVersion) RegisterV1Routers(ctx context.Context, v1 *gin.RouterGroup) error {
	v1.GET("current/version", a.CurrentVersion)
	v1.GET("system/updates", a.Updates)
	v1.POST("system/updates/check", a.Check)
	v1.POST("gateway/version", a.Report)
	v1.GET("system/upgrade/capability", a.UpgradeCapability)
	v1.POST("system/upgrade/prepare", a.UpgradePrepare)
	v1.POST("system/upgrade/submit", a.UpgradeSubmit)
	v1.GET("system/upgrade/task", a.UpgradeTask)
	return nil
}

func (a *SystemVersion) Release(ctx context.Context) error {
	a.Service.Close()
	return nil
}
