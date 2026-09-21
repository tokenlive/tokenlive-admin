package biz

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

func smartRedisFixture(t *testing.T, db *gorm.DB) (*ConfigRedisSync, *miniredis.Miniredis) {
	t.Helper()
	old := config.C.Sync
	config.C.Sync.Endpoints, config.C.Sync.Policies = true, true
	t.Cleanup(func() { config.C.Sync = old })
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })
	return &ConfigRedisSync{RedisClient: client, ModelDAL: &dal.Model{DB: db}, EndpointDAL: &dal.Endpoint{DB: db}}, server
}

func seedSmartEndpoints(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Create(&schema.Provider{ID: "provider", Name: "Provider", Code: "provider", Enabled: 1, Protocol: "openai"}).Error)
	for _, id := range []string{"a", "b"} {
		require.NoError(t, db.Create(&schema.Endpoint{ID: "ep-" + id, Code: "ep-" + id, ModelID: id, ProviderID: "provider", Enabled: 1, Protocol: "openai", URL: "https://example.test/v1"}).Error)
	}
}

func TestSmartGatewayExportIncludesDependenciesAndEndpointFreeModel(t *testing.T) {
	db, b, ctx := smartFixture(t)
	model := createSmartFixture(t, b, ctx)
	seedSmartEndpoints(t, db)
	sync := &GatewaySync{DB: db}
	for _, code := range []string{"", "smart"} {
		ClearGatewayConfigCache()
		got, err := sync.GetGatewayConfig(ctx, code)
		require.NoError(t, err)
		require.Len(t, got.Models, 3)
		raw, err := json.Marshal(got)
		require.NoError(t, err)
		var wire struct {
			Models map[string]struct {
				ModelType    string           `json:"model_type"`
				Endpoints    []EndpointConfig `json:"endpoints"`
				SmartRouting map[string]any   `json:"smart_routing"`
			} `json:"models"`
		}
		require.NoError(t, json.Unmarshal(raw, &wire))
		require.Equal(t, "smart", wire.Models["smart"].ModelType)
		require.Empty(t, wire.Models["smart"].Endpoints)
		require.Equal(t, "a", wire.Models["smart"].SmartRouting["judge_model"])
		require.Equal(t, float64(1), wire.Models["smart"].SmartRouting["version"])
		require.Len(t, wire.Models["a"].Endpoints, 1)
	}
	require.NoError(t, b.ToggleEnabled(ctx, "a", &schema.ModelEnabledForm{Enabled: 0}))
	ClearGatewayConfigCache()
	got, err := sync.GetGatewayConfig(ctx, model.ModelCode)
	require.NoError(t, err)
	require.Contains(t, got.Models, "smart", "dependency outage must not suppress the composite")
	require.NotContains(t, got.Models, "a")
}

func TestSmartRedisPublicationDisableDeleteAndConversion(t *testing.T) {
	db, b, ctx := smartFixture(t)
	sync, server := smartRedisFixture(t, db)
	b.ConfigRedisSync = sync
	seedSmartEndpoints(t, db)
	model := createSmartFixture(t, b, ctx)
	raw, err := server.Get("aigw:config:smart_routing:smart")
	require.NoError(t, err)
	require.JSONEq(t, `{"version":1,"judge_model":"a","judge_timeout_ms":5000,"judge_max_input_bytes":65536,"judge_max_output_tokens":256,"ranges":[{"min":0,"max":40,"model":"a"},{"min":40,"max":100,"model":"b"}]}`, raw)
	require.NotEmpty(t, server.HGet(RedisKeyConfigModelVersions, "smart"))
	require.False(t, server.Exists("aigw:config:endpoints:smart"))

	disable := &schema.ModelEnabledForm{Enabled: 0}
	require.NoError(t, b.ToggleEnabled(ctx, model.ID, disable))
	require.False(t, server.Exists("aigw:config:smart_routing:smart"))
	require.Empty(t, server.HGet(RedisKeyConfigModelVersions, "smart"))
	require.NoError(t, b.ToggleEnabled(ctx, model.ID, &schema.ModelEnabledForm{Enabled: 1}))
	require.True(t, server.Exists("aigw:config:smart_routing:smart"))
	normal := &schema.ModelForm{ModelCode: "smart", ModelName: "Smart", SpaceCode: "default", Enabled: 1, RequestTypes: `["chat_completion"]`}
	version := int64(1)
	normal.SmartRoutingVersion = &version
	require.NoError(t, b.Update(ctx, model.ID, normal))
	require.False(t, server.Exists("aigw:config:smart_routing:smart"))
	f := smartForm()
	require.NoError(t, b.Update(ctx, model.ID, f))
	require.True(t, server.Exists("aigw:config:smart_routing:smart"))
	require.NoError(t, b.Delete(ctx, model.ID))
	require.False(t, server.Exists("aigw:config:smart_routing:smart"))
	require.Empty(t, server.HGet(RedisKeyConfigModelVersions, "smart"))
}

func TestSmartSavedPublicationFailureIsObservableAndRetryable(t *testing.T) {
	db, b, ctx := smartFixture(t)
	sync, server := smartRedisFixture(t, db)
	b.ConfigRedisSync = sync
	server.SetError("ERR simulated publish outage")
	result, err := b.Create(ctx, smartForm())
	require.NoError(t, err)
	require.True(t, result.Saved)
	require.Equal(t, "failed", result.SyncStatus)
	require.NotEmpty(t, result.Warnings)
	require.NotNil(t, result.Model)
	server.SetError("")
	require.NoError(t, b.Sync(ctx, result.ID))
	require.True(t, server.Exists("aigw:config:smart_routing:smart"))
	raw, err := server.Get("aigw:config:smart_routing:smart")
	require.NoError(t, err)
	require.Contains(t, raw, `"version":1`)
}

func TestSmartPublishNotifiesOnlyAfterCompleteRuntimeVersion(t *testing.T) {
	db, b, ctx := smartFixture(t)
	sync, server := smartRedisFixture(t, db)
	b.ConfigRedisSync = sync
	util.ResetConfigChangeListeners()
	t.Cleanup(util.ResetConfigChangeListeners)
	sawPublished := false
	util.OnConfigChanged(func(_ context.Context, _ util.ConfigChangeKind, codes ...string) {
		for _, code := range codes {
			if code == "smart" {
				sawPublished = true
				require.True(t, server.Exists("aigw:config:smart_routing:smart"))
				require.NotEmpty(t, server.HGet(RedisKeyConfigModelVersions, "smart"))
			}
		}
	})
	createSmartFixture(t, b, ctx)
	require.True(t, sawPublished)
}

func TestSmartRedisRejectedGenerationDoesNotWriteHalfConfiguration(t *testing.T) {
	db, b, ctx := smartFixture(t)
	sync, server := smartRedisFixture(t, db)
	model := createSmartFixture(t, b, ctx)
	server.Set(RedisKeyConfigModelVersions, "wrong-type")
	require.Error(t, sync.SyncModelByCode(ctx, model.ModelCode))
	require.False(t, server.Exists("aigw:config:smart_routing:smart"))
}

func TestSmartDeletePublicationFailureIsReturnedAfterDatabaseCommit(t *testing.T) {
	db, b, ctx := smartFixture(t)
	sync, server := smartRedisFixture(t, db)
	b.ConfigRedisSync = sync
	model := createSmartFixture(t, b, ctx)
	server.SetError("ERR simulated outage")
	require.Error(t, b.Delete(ctx, model.ID), "deletion must not claim runtime revocation succeeded")
	server.SetError("")
	got, err := b.ModelDAL.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Nil(t, got, "database deletion already committed")
	// A deleted model must remain synchronizable by its recorded code.
	require.NoError(t, sync.SyncModelByCode(ctx, model.ModelCode))
	require.False(t, server.Exists("aigw:config:smart_routing:smart"))
}

func TestSmartLatePublicationCannotOverwriteNewerRuntime(t *testing.T) {
	db, b, ctx := smartFixture(t)
	sync, server := smartRedisFixture(t, db)
	b.ConfigRedisSync = sync
	old := createSmartFixture(t, b, ctx)
	form := smartForm()
	form.SmartRouting.Version = 1
	form.SmartRouting.Ranges[0].Max, form.SmartRouting.Ranges[1].Min = 70, 70
	require.NoError(t, b.Update(ctx, old.ID, form))
	require.Error(t, sync.publishSmartRouting(ctx, old), "an older concurrent save must not supersede the newer published version")
	raw, err := server.Get("aigw:config:smart_routing:smart")
	require.NoError(t, err)
	require.Contains(t, raw, `"version":2`)
	require.Contains(t, raw, `"max":70`)
}

func TestSmartLatePublicationCannotReactivateDisabledModel(t *testing.T) {
	db, b, ctx := smartFixture(t)
	sync, server := smartRedisFixture(t, db)
	b.ConfigRedisSync = sync
	old := createSmartFixture(t, b, ctx)
	require.NoError(t, b.ToggleEnabled(ctx, old.ID, &schema.ModelEnabledForm{Enabled: 0}))
	require.Error(t, sync.publishSmartRouting(ctx, old))
	require.False(t, server.Exists("aigw:config:smart_routing:smart"))
	require.Empty(t, server.HGet(RedisKeyConfigModelVersions, "smart"))
}
