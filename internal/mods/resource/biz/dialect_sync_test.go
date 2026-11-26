package biz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy"
	policyAPI "github.com/tokenlive/tokenlive-admin/internal/mods/policy/api"
	policyBiz "github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestModelDialectLifecycleAndSnapshots(t *testing.T) {
	db, models, ctx := smartFixture(t)
	redisSync, redisServer := smartRedisFixture(t, db)
	models.ConfigRedisSync = redisSync
	seedSmartEndpoints(t, db)
	store := policySchema.DialectStoreUnsupported
	developer := policySchema.DialectDeveloperToSystem
	dialects := &policyBiz.PolicyDialect{
		Trans: models.Trans, ModelDAL: models.ModelDAL, DataPermissionDAL: models.DataPermissionBIZ.DataPermissionDAL,
		Publisher: redisSync, AuditLogBIZ: models.AuditLogBIZ,
	}
	module := &policy.Policy{PolicyDialectAPI: &policyAPI.PolicyDialect{PolicyDialectBIZ: dialects}}
	router := gin.New()
	api := router.Group("/api/v1")
	require.NoError(t, module.RegisterV1Routers(ctx, api))

	path := "/api/v1/policy/policy-dialects"
	request := func(auth context.Context, method, suffix, body string, status int) json.RawMessage {
		t.Helper()
		req := httptest.NewRequest(method, path+suffix, strings.NewReader(body)).WithContext(auth)
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, status, res.Code, "%s %s: %s", method, suffix, res.Body.String())
		var envelope struct {
			Success bool            `json:"success"`
			Data    json.RawMessage `json:"data"`
		}
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &envelope))
		require.Equal(t, status == http.StatusOK, envelope.Success)
		return envelope.Data
	}
	create := func(model, implementation string) policySchema.PolicyDialect {
		t.Helper()
		data := request(ctx, http.MethodPost, "", `{"model_id":"`+model+`","implementation":"`+implementation+`","enabled":1}`, http.StatusOK)
		var result policySchema.PolicyDialect
		require.NoError(t, json.Unmarshal(data, &result))
		require.Zero(t, result.Enabled)
		require.Equal(t, "global", result.ScopeType)
		require.Empty(t, result.ScopeCode)
		require.Equal(t, implementation, result.Name)
		require.Equal(t, int64(1), result.Version)
		return result
	}
	pull := &GatewaySync{DB: db}
	redisKey := gatewaycontract.Keys.Config.Endpoints("a")
	assertPublication := func(want []string, count int) {
		t.Helper()
		raw, err := redisServer.Get(redisKey)
		require.NoError(t, err)
		var endpoints []ResolvedEndpoint
		require.NoError(t, json.Unmarshal([]byte(raw), &endpoints))
		require.Len(t, endpoints, count)
		for _, ep := range endpoints {
			require.Equal(t, want, ep.DialectImplementations)
		}
		config, err := pull.GetGatewayConfig(ctx, "")
		require.NoError(t, err)
		require.Len(t, config.Models["a"].Endpoints, count)
		for _, ep := range config.Models["a"].Endpoints {
			require.Equal(t, want, ep.DialectImplementations)
		}
		require.Empty(t, config.Models["b"].Endpoints[0].DialectImplementations)
	}

	// Endpoint snapshots apply to the entire model, so other scopes cannot
	// be published to every caller of a model.
	require.NoError(t, db.Create(&policySchema.PolicyDialect{
		ID: "tenant-dialect", ModelID: "b", Name: store, Implementation: store,
		ScopeType: "tenant", ScopeCode: "tenant-a", Enabled: 1,
	}).Error)
	require.NoError(t, redisSync.SyncModelByCode(ctx, "b"))
	raw, err := redisServer.Get(gatewaycontract.Keys.Config.Endpoints("b"))
	require.NoError(t, err)
	require.NotContains(t, raw, "dialect_implementations")

	require.NoError(t, redisSync.SyncModelByCode(ctx, "a"))
	assertPublication(nil, 1)
	request(ctx, http.MethodPost, "", `{"model_id":"a","implementation":"custom_script"}`, http.StatusBadRequest)
	smart := createSmartFixture(t, models, ctx)
	request(ctx, http.MethodPost, "", `{"model_id":"`+smart.ID+`","implementation":"`+store+`"}`, http.StatusBadRequest)
	request(ctx, http.MethodGet, "", "", http.StatusBadRequest)
	storePolicy := create("a", store)
	developerPolicy := create("a", developer)
	request(ctx, http.MethodPost, "", `{"model_id":"a","implementation":"`+store+`"}`, http.StatusConflict)
	assertPublication(nil, 1)
	page := request(ctx, http.MethodGet, "?model_id=a&current=2&pageSize=1", "", http.StatusOK)
	var items []policySchema.PolicyDialect
	require.NoError(t, json.Unmarshal(page, &items))
	require.Len(t, items, 1)
	require.Equal(t, storePolicy.ID, items[0].ID)

	reader := util.NewTenant(util.NewUsername(context.Background(), "reader"), "tenant-a")
	require.NoError(t, db.Create(&schema.DataPermission{
		ID: "dialect-reader", Type: schema.DataPermissionTypeModel, DataId: "a", User: "reader", Tenant: "tenant-a",
		Role: "member", Permission: 1,
	}).Error)
	request(reader, http.MethodGet, "?model_id=a", "", http.StatusOK)
	request(reader, http.MethodPut, "/"+storePolicy.ID+"/enabled", `{"enabled":1}`, http.StatusNotFound)
	request(reader, http.MethodDelete, "/"+storePolicy.ID, "", http.StatusNotFound)
	require.NoError(t, db.Model(&schema.DataPermission{}).Where("id = ?", "dialect-reader").Update("deleted", "dialect-reader").Error)
	request(reader, http.MethodGet, "?model_id=a", "", http.StatusNotFound)

	request(ctx, http.MethodPut, "/"+storePolicy.ID+"/enabled", `{}`, http.StatusBadRequest)
	request(ctx, http.MethodPut, "/"+storePolicy.ID+"/enabled", `{"enabled":2}`, http.StatusBadRequest)
	request(ctx, http.MethodPut, "/"+storePolicy.ID+"/enabled", `{"enabled":1}`, http.StatusOK)
	assertPublication([]string{store}, 1)
	request(ctx, http.MethodPut, "/"+storePolicy.ID+"/enabled", `{"enabled":1}`, http.StatusOK)
	var storeStatus policySchema.PolicyDialect
	require.NoError(t, db.First(&storeStatus, "id = ?", storePolicy.ID).Error)
	require.Equal(t, int64(2), storeStatus.Version, "same-state sync retry must not increment the version")
	request(ctx, http.MethodPut, "/"+developerPolicy.ID+"/enabled", `{"enabled":1}`, http.StatusOK)
	assertPublication([]string{developer, store}, 1)

	require.NoError(t, db.Create(&schema.Endpoint{
		ID: "new-endpoint", Code: "new-endpoint", ModelID: "a", ProviderID: "provider", Enabled: 1, Protocol: "openai",
		URL: "https://other.example.test/v1",
	}).Error)
	require.NoError(t, redisSync.SyncModelByCode(ctx, "a"))
	assertPublication([]string{developer, store}, 2)

	redisServer.SetError("ERR simulated publish outage")
	data := request(ctx, http.MethodPut, "/"+storePolicy.ID+"/enabled", `{"enabled":0}`, http.StatusOK)
	var result schema.ModelMutationResult
	require.NoError(t, json.Unmarshal(data, &result))
	require.True(t, result.Saved)
	require.Equal(t, "failed", result.SyncStatus)
	require.NotEmpty(t, result.Warnings)
	require.NoError(t, db.First(&storeStatus, "id = ?", storePolicy.ID).Error)
	require.Equal(t, int64(3), storeStatus.Version, "saved change advances version even when publication fails")
	httpConfig, err := pull.GetGatewayConfig(ctx, "a")
	require.NoError(t, err)
	require.Equal(t, []string{developer}, httpConfig.Models["a"].Endpoints[0].DialectImplementations)

	redisServer.SetError("")
	require.NoError(t, models.Sync(ctx, "a"))
	assertPublication([]string{developer}, 2)
	request(ctx, http.MethodDelete, "/"+developerPolicy.ID, "", http.StatusOK)
	assertPublication(nil, 2)
	require.NotEqual(t, developerPolicy.ID, create("a", developer).ID)
	request(ctx, http.MethodPut, "/"+developerPolicy.ID+"/enabled", `{"enabled":1}`, http.StatusNotFound)
}
