package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/biz"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestLoginGetVersion(t *testing.T) {
	origVer := config.C.General.Version
	origName := config.C.General.AppName
	config.C.General.Version = "v0.9.3"
	config.C.General.AppName = "tokenlive-admin"
	defer func() {
		config.C.General.Version = origVer
		config.C.General.AppName = origName
	}()

	loginAPI := &Login{
		LoginBIZ: &biz.Login{},
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/pub/version", nil)

	loginAPI.GetVersion(ctx)

	assert.Equal(t, http.StatusOK, recorder.Code)
	var response util.ResponseResult
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "v0.9.3", data["version"])
	assert.Equal(t, "tokenlive-admin", data["app_name"])
}
