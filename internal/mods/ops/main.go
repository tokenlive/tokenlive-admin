package ops

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"gorm.io/gorm"
)

// Ops is the operations module for event monitoring, dashboard, and audit logging.
type Ops struct {
	DB            *gorm.DB
	EventAPI      *api.EventAPI
	AuditLogAPI   *api.AuditLog
	PortalUserAPI *api.PortalUserAPI
	Consumer      *biz.Consumer
	CleanupTask   *biz.CleanupTask
	Hub           *api.WSHub
}

// AutoMigrate creates or updates the event_log and audit_log tables.
func (a *Ops) AutoMigrate(ctx context.Context) error {
	return a.DB.AutoMigrate(
		new(schema.EventLog),
		new(schema.AuditLog),
	)
}

// Init initializes the ops module: auto-migrate, start consumer and cleanup.
func (a *Ops) Init(ctx context.Context) error {
	if config.C.Storage.DB.AutoMigrate {
		if err := a.AutoMigrate(ctx); err != nil {
			return err
		}
	}

	// Start WebSocket hub
	go a.Hub.Run()

	// Connect consumer to WebSocket hub and start
	a.Consumer.SetHub(a.Hub)
	a.Consumer.Start(ctx)
	a.CleanupTask.Start(ctx)

	return nil
}

// RegisterV1Routers registers the ops API routes.
func (a *Ops) RegisterV1Routers(ctx context.Context, v1 *gin.RouterGroup) error {
	g := v1.Group("ops")
	{
		g.GET("events", a.EventAPI.Query)
		g.GET("events/statistics", a.EventAPI.GetStatistics)
		g.GET("events/ws", a.EventAPI.HandleWebSocket)
		g.GET("portal/users", a.PortalUserAPI.Query)
		g.GET("portal/workspaces/:workspace_id/api-keys", a.PortalUserAPI.ListWorkspaceAPIKeys)
		g.POST("portal/workspaces/:workspace_id/runtime-sync", a.PortalUserAPI.SyncWorkspaceRuntime)
	}

	auditLogs := v1.Group("audit-logs")
	{
		auditLogs.GET("", a.AuditLogAPI.Query)
		auditLogs.GET(":id", a.AuditLogAPI.Get)
	}

	return nil
}

// Release cleans up resources.
func (a *Ops) Release(ctx context.Context) error {
	return nil
}
