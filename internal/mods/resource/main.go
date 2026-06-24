package resource

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"gorm.io/gorm"
)

type Resource struct {
	DB                *gorm.DB
	ProviderAPI       *api.Provider
	EndpointAPI       *api.Endpoint
	ModelAPI          *api.Model
	ModelAliasAPI     *api.ModelAlias
	DataPermissionAPI *api.DataPermission
}

func (a *Resource) AutoMigrate(ctx context.Context) error {
	return a.DB.AutoMigrate(
		new(schema.Provider),
		new(schema.Endpoint),
		new(schema.Model),
		new(schema.ModelAlias),
		new(schema.DataPermission),
	)
}

func (a *Resource) Init(ctx context.Context) error {
	if config.C.Storage.DB.AutoMigrate {
		if err := a.AutoMigrate(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (a *Resource) RegisterV1Routers(ctx context.Context, v1 *gin.RouterGroup) error {
	providers := v1.Group("providers")
	{
		providers.GET("", a.ProviderAPI.Query)
		providers.GET(":id", a.ProviderAPI.Get)
		providers.POST("", a.ProviderAPI.Create)
		providers.PUT(":id", a.ProviderAPI.Update)
		providers.DELETE(":id", a.ProviderAPI.Delete)
		providers.POST(":id/fetch-models", a.ProviderAPI.FetchModels)
		providers.GET(":id/endpoints", a.EndpointAPI.QueryEndpointsByProviderID)
	}

	endpoints := v1.Group("endpoints")
	{
		endpoints.GET("", a.EndpointAPI.Query)
		endpoints.GET(":id", a.EndpointAPI.Get)
		endpoints.POST("", a.EndpointAPI.Create)
		endpoints.PUT(":id", a.EndpointAPI.Update)
		endpoints.PUT(":id/enabled", a.EndpointAPI.UpdateEnabled)
		endpoints.DELETE(":id", a.EndpointAPI.Delete)
		endpoints.POST("test", a.EndpointAPI.Test)
		endpoints.POST(":id/test", a.EndpointAPI.TestByID)
	}

	models := v1.Group("models")
	{
		models.GET("", a.ModelAPI.Query)
		models.GET(":id", a.ModelAPI.Get)
		models.POST("", a.ModelAPI.Create)
		models.PUT(":id", a.ModelAPI.Update)
		models.PUT(":id/enabled", a.ModelAPI.UpdateEnabled)
		models.DELETE(":id", a.ModelAPI.Delete)
		models.GET(":id/endpoints", a.EndpointAPI.QueryEndpointsByModelID)
		models.POST(":id/sync", a.ModelAPI.Sync)
	}
	modelAliases := v1.Group("model-aliases")
	{
		modelAliases.GET("", a.ModelAliasAPI.Query)
		modelAliases.GET(":id", a.ModelAliasAPI.Get)
		modelAliases.POST("", a.ModelAliasAPI.Create)
		modelAliases.PUT(":id", a.ModelAliasAPI.Update)
		modelAliases.DELETE(":id", a.ModelAliasAPI.Delete)
	}
	dataPermission := v1.Group("data-permissions")
	{
		dataPermission.GET("", a.DataPermissionAPI.Query)
		dataPermission.GET(":id", a.DataPermissionAPI.Get)
		dataPermission.POST("", a.DataPermissionAPI.Create)
		dataPermission.PUT(":id", a.DataPermissionAPI.Update)
		dataPermission.DELETE(":id", a.DataPermissionAPI.Delete)
	}

	return nil
}

func (a *Resource) Release(ctx context.Context) error {
	return nil
}
