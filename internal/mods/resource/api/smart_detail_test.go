package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSmartDetailAPIStagedConfigurationAndBasicEdit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	conn, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(); biz.ClearGatewayConfigCache() })
	require.NoError(t, db.AutoMigrate(&schema.Model{}, &schema.Endpoint{}, &schema.DataPermission{}))
	for _, id := range []string{"a", "b"} {
		require.NoError(t, db.Create(&schema.Model{ID: id, ModelCode: id, ModelName: id, SpaceCode: "test", Enabled: 1, RequestTypes: `["chat_completion"]`}).Error)
	}
	trans := &util.Trans{DB: db}
	b := &biz.Model{
		Trans: trans, ModelDAL: &dal.Model{DB: db},
		DataPermissionBIZ: &biz.DataPermission{Trans: trans, DataPermissionDAL: &dal.DataPermission{DB: db}},
		ConfigRedisSync:   &biz.ConfigRedisSync{ModelDAL: &dal.Model{DB: db}, EndpointDAL: &dal.Endpoint{DB: db}},
	}
	handler := &api.Model{ModelBIZ: b}
	engine := gin.New()
	mod := &resource.Resource{ModelAPI: handler}
	require.NoError(t, mod.RegisterV1Routers(context.Background(), engine.Group("/api/v1")))
	request := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
		r = r.WithContext(util.NewIsRootUser(context.Background()))
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, r)
		return w
	}
	base := `{"model_name":"Draft","model_code":"smart","space_code":"test","model_type":"smart","enabled":0,"request_types":"[\"chat_completion\"]","abilities":"[]"}`
	w := request(http.MethodPost, "/api/v1/models", base)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var created struct {
		Data schema.Model `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.NotEmpty(t, created.Data.ID)
	require.False(t, created.Data.SmartRoutingReady)
	path := "/api/v1/models/" + created.Data.ID
	require.Equal(t, http.StatusBadRequest, request(http.MethodPut, path+"/enabled", `{"enabled":1}`).Code)
	routing := `{"version":0,"judge_model_id":"a","ranges":[{"min":0,"max":50,"model_id":"a"},{"min":50,"max":100,"model_id":"b"}]}`
	w = request(http.MethodPut, path+"/smart-routing", routing)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = request(http.MethodPut, path+"/enabled", `{"enabled":1}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = request(http.MethodPut, path, `{"model_name":"Changed","model_code":"smart","space_code":"test","model_type":"smart","enabled":1,"request_types":"[\"chat_completion\"]","abilities":"[]"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	get := request(http.MethodGet, path, "")
	require.Equal(t, http.StatusOK, get.Code, get.Body.String())
	var updated struct {
		Data schema.Model `json:"data"`
	}
	require.NoError(t, json.Unmarshal(get.Body.Bytes(), &updated))
	require.Equal(t, "Changed", updated.Data.ModelName)
	require.True(t, updated.Data.SmartRoutingReady)
	require.Equal(t, int64(1), updated.Data.SmartRouting.Version)
	require.Equal(t, 1, updated.Data.Enabled)
	require.Equal(t, http.StatusConflict, request(http.MethodPut, path+"/smart-routing", routing).Code)
}
