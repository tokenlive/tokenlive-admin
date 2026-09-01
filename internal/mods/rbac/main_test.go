package rbac

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/api"
)

func TestRegisterV1RoutersIncludesPubVersion(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	module := &RBAC{
		LoginAPI:          &api.Login{},
		OAuthAPI:          &api.OAuth{},
		MenuAPI:           &api.Menu{},
		RoleAPI:           &api.Role{},
		UserAPI:           &api.User{},
		UserAPIKeyAPI:     &api.UserAPIKey{},
		TenantAPI:         &api.Tenant{},
		LoggerAPI:         &api.Logger{},
		TenantModelAPI:    &api.TenantModel{},
		TenantEndpointAPI: &api.TenantEndpoint{},
	}

	require.NoError(t, module.RegisterV1Routers(context.Background(), router.Group("/api/v1")))

	found := false
	for _, route := range router.Routes() {
		if route.Method == http.MethodGet && route.Path == "/api/v1/pub/version" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected /api/v1/pub/version route to be registered")
}
