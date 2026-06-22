package rbac

import (
	"context"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RBAC struct {
	DB                     *gorm.DB
	MenuAPI                *api.Menu
	RoleAPI                *api.Role
	UserAPI                *api.User
	LoginAPI               *api.Login
	LoggerAPI              *api.Logger
	UserAPIKeyAPI          *api.UserAPIKey
	TenantAPI              *api.Tenant
	TenantModelAPI         *api.TenantModel
	TenantEndpointAPI      *api.TenantEndpoint
	Casbinx                *Casbinx
}

func (a *RBAC) AutoMigrate(ctx context.Context) error {
	return a.DB.AutoMigrate(
		new(schema.Menu),
		new(schema.MenuResource),
		new(schema.Role),
		new(schema.RoleMenu),
		new(schema.User),
		new(schema.UserRole),
		new(schema.UserAPIKey),
		new(schema.Tenant),
		new(schema.TenantModel),
		new(schema.TenantEndpoint),
	)
}

func (a *RBAC) Init(ctx context.Context) error {
	if config.C.Storage.DB.AutoMigrate {
		var err error
		for i := 0; i < 3; i++ {
			err = a.AutoMigrate(ctx)
			if err == nil {
				break
			}
			logging.Context(ctx).Warn("AutoMigrate failed, retrying in 500ms...", zap.Error(err), zap.Int("attempt", i+1))
			time.Sleep(500 * time.Millisecond)
		}
		if err != nil {
			return err
		}
	}

	if err := a.Casbinx.Load(ctx); err != nil {
		return err
	}

	if name := config.C.General.MenuFile; name != "" {
		fullPath := filepath.Join(config.C.General.WorkDir, name)
		if err := a.MenuAPI.MenuBIZ.InitFromFile(ctx, fullPath); err != nil {
			logging.Context(ctx).Error("failed to init menu data", zap.Error(err), zap.String("file", fullPath))
		}
	}

	return nil
}

func (a *RBAC) RegisterV1Routers(ctx context.Context, v1 *gin.RouterGroup) error {
	captcha := v1.Group("captcha")
	{
		captcha.GET("id", a.LoginAPI.GetCaptcha)
		captcha.GET("image", a.LoginAPI.ResponseCaptcha)
	}

	v1.POST("login", a.LoginAPI.Login)

	current := v1.Group("current")
	{
		current.POST("refresh-token", a.LoginAPI.RefreshToken)
		current.GET("user", a.LoginAPI.GetUserInfo)
		current.GET("menus", a.LoginAPI.QueryMenus)
		current.PUT("password", a.LoginAPI.UpdatePassword)
		current.PUT("user", a.LoginAPI.UpdateUser)
		current.POST("logout", a.LoginAPI.Logout)
	}

	menu := v1.Group("menus")
	{
		menu.GET("", a.MenuAPI.Query)
		menu.GET(":id", a.MenuAPI.Get)
		menu.POST("", a.MenuAPI.Create)
		menu.PUT(":id", a.MenuAPI.Update)
		menu.DELETE(":id", a.MenuAPI.Delete)
	}

	role := v1.Group("roles")
	{
		role.GET("", a.RoleAPI.Query)
		role.GET(":id", a.RoleAPI.Get)
		role.POST("", a.RoleAPI.Create)
		role.PUT(":id", a.RoleAPI.Update)
		role.DELETE(":id", a.RoleAPI.Delete)
	}

	user := v1.Group("users")
	{
		user.GET("", a.UserAPI.Query)
		user.GET(":id", a.UserAPI.Get)
		user.POST("", a.UserAPI.Create)
		user.PUT(":id", a.UserAPI.Update)
		user.DELETE(":id", a.UserAPI.Delete)
		user.PATCH(":id/reset-pwd", a.UserAPI.ResetPassword)
	}

	userAPIKeys := v1.Group("user-api-keys")
	{
		userAPIKeys.GET("", a.UserAPIKeyAPI.Query)
		userAPIKeys.GET(":id", a.UserAPIKeyAPI.Get)
		userAPIKeys.GET(":id/plaintext", a.UserAPIKeyAPI.GetPlaintext)
		userAPIKeys.POST("", a.UserAPIKeyAPI.Create)
		userAPIKeys.PUT(":id", a.UserAPIKeyAPI.Update)
		userAPIKeys.DELETE(":id", a.UserAPIKeyAPI.Delete)
	}

	tenant := v1.Group("tenants")
	{
		tenant.GET("", a.TenantAPI.Query)
		tenant.GET(":id", a.TenantAPI.Get)
		tenant.POST("", a.TenantAPI.Create)
		tenant.PUT(":id", a.TenantAPI.Update)
		tenant.DELETE(":id", a.TenantAPI.Delete)
	}

	logger := v1.Group("loggers")
	{
		logger.GET("", a.LoggerAPI.Query)
	}

	tenantModels := v1.Group("tenant-models")
	{
		tenantModels.GET(":tenantCode", a.TenantModelAPI.GetAuthorizedModelIDs)
		tenantModels.POST("bindings", a.TenantModelAPI.SaveBindings)
		tenantModels.GET("endpoints", a.TenantEndpointAPI.GetAllowedEndpointIDs)
		tenantModels.POST("endpoints", a.TenantEndpointAPI.SaveEndpoints)
	}

	return nil
}

func (a *RBAC) Release(ctx context.Context) error {
	if err := a.Casbinx.Release(ctx); err != nil {
		return err
	}
	return nil
}
