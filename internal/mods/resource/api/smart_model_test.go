package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSmartCreateAPIReportsSavedButNotPublished(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close(); biz.ClearGatewayConfigCache() })
	require.NoError(t, db.AutoMigrate(&schema.Model{}, &schema.Endpoint{}, &schema.DataPermission{}))
	for _, id := range []string{"a", "b"} {
		require.NoError(t, db.Create(&schema.Model{ID: id, ModelCode: id, ModelName: id, Enabled: 1, SpaceCode: "test", RequestTypes: `["chat_completion"]`}).Error)
	}
	old := config.C.Sync
	config.C.Sync.Endpoints, config.C.Sync.Policies = true, true
	t.Cleanup(func() { config.C.Sync = old })
	server := miniredis.RunT(t)
	server.SetError("ERR publication unavailable")
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	trans := &util.Trans{DB: db}
	handler := &Model{ModelBIZ: &biz.Model{
		Trans: trans, ModelDAL: &dal.Model{DB: db},
		DataPermissionBIZ: &biz.DataPermission{Trans: trans, DataPermissionDAL: &dal.DataPermission{DB: db}},
		ConfigRedisSync:   &biz.ConfigRedisSync{RedisClient: client, ModelDAL: &dal.Model{DB: db}, EndpointDAL: &dal.Endpoint{DB: db}},
	}}
	engine := gin.New()
	engine.POST("/models", handler.Create)
	body := []byte(`{"model_name":"Smart","model_code":"smart","model_type":"smart","space_code":"test","enabled":1,"request_types":"[\"chat_completion\"]","smart_routing":{"judge_model_id":"a","ranges":[{"min":0,"max":50,"model_id":"a"},{"min":50,"max":100,"model_id":"b"}]}}`)
	request := httptest.NewRequest(http.MethodPost, "/models", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(util.NewIsRootUser(context.Background()))
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			ID         string   `json:"id"`
			Saved      bool     `json:"saved"`
			SyncStatus string   `json:"sync_status"`
			Warnings   []string `json:"warnings"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.True(t, response.Data.Saved)
	require.NotEmpty(t, response.Data.ID)
	require.Equal(t, "failed", response.Data.SyncStatus)
	require.NotEmpty(t, response.Data.Warnings)
	var saved schema.Model
	require.NoError(t, db.First(&saved, "id = ?", response.Data.ID).Error)
	require.Equal(t, int64(1), saved.SmartRouting.Version)
}
