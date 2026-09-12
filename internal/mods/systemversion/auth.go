package systemversion

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// CanManage checks the single update-management capability. Role names have no
// special meaning; only trusted Root context or an explicit policy grants it.
func CanManage(root bool, roles []string, enforce func(...interface{}) (bool, error)) bool {
	if root {
		return true
	}
	if enforce == nil {
		return false
	}
	for _, role := range roles {
		allowed, err := enforce(role, "/api/v1/system/updates/check", "POST")
		if err == nil && allowed {
			return true
		}
	}
	return false
}

func (a *SystemVersion) canManage(ctx context.Context) bool {
	if util.FromIsRootUser(ctx) {
		return true
	}
	// Disabling the global route middleware does not grant this capability.
	if config.C.Middleware.Casbin.Disable || a.RBAC == nil {
		return false
	}
	enforcer := a.RBAC.Casbinx.GetEnforcer()
	if enforcer == nil {
		return false
	}
	return CanManage(false, util.FromUserCache(ctx).RoleIDs, enforcer.Enforce)
}
