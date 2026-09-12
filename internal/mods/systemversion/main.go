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
	// HTTP handlers and authorization are added by the API integration task.
	return nil
}

func (a *SystemVersion) Release(ctx context.Context) error {
	a.Service.Close()
	return nil
}
