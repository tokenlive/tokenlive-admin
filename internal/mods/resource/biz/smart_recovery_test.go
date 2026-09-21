package biz

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSmartLateDisablePublicationKeepsLatestEnable(t *testing.T) {
	db, b, ctx := smartFixture(t)
	sync, server := smartRedisFixture(t, db)
	b.ConfigRedisSync = sync
	before := createSmartFixture(t, b, ctx)
	disabled := *before
	disabled.Enabled = 0
	require.NoError(t, b.ModelDAL.UpdateEnabled(ctx, before.ID, 0, "alice"))
	require.NoError(t, b.ToggleEnabled(ctx, before.ID, &schema.ModelEnabledForm{Enabled: 1}))
	require.NoError(t, b.publishModelChange(ctx, &disabled, before, nil))
	current, err := b.ModelDAL.Get(ctx, before.ID)
	require.NoError(t, err)
	require.Equal(t, 1, current.Enabled)
	require.True(t, server.Exists(RedisKeySmartRoutingPrefix+"smart"))
	require.NotEmpty(t, server.HGet(RedisKeyConfigModelVersions, "smart"))
}

func TestSmartSyncRetryRepairsFailedAvailability(t *testing.T) {
	db, b, ctx := smartFixture(t)
	sync, server := smartRedisFixture(t, db)
	b.ConfigRedisSync = sync
	seedSmartEndpoints(t, db)
	createSmartFixture(t, b, ctx)
	require.NoError(t, b.Sync(ctx, "a"))
	server.SetError("ERR simulated outage")
	form := &schema.ModelForm{ModelCode: "a", ModelName: "Updated", SpaceCode: "default", Enabled: 1, RequestTypes: `["chat_completion"]`}
	require.NoError(t, b.Update(ctx, "a", form))
	require.Equal(t, "failed", form.Result.SyncStatus)
	server.SetError("")
	require.NoError(t, b.Sync(ctx, "a"))
	raw, err := server.Get(RedisKeySmartRoutingPrefix + "smart")
	require.NoError(t, err)
	require.Contains(t, raw, `"judge_model":"a"`)
	require.Contains(t, raw, `"version":1`)
	require.True(t, server.Exists("aigw:config:endpoints:a"))
}

func TestSmartDeletedCodeReuseDoesNotInheritFormerOwnerAccess(t *testing.T) {
	db, b, ctx := smartFixture(t)
	sync, server := smartRedisFixture(t, db)
	b.ConfigRedisSync = sync
	seedSmartEndpoints(t, db)
	require.NoError(t, b.Sync(ctx, "a"))
	require.NoError(t, db.Exec("DELETE FROM endpoint WHERE model_id = 'a'").Error)
	require.NoError(t, b.Delete(ctx, "a"))
	require.NoError(t, db.Create(&schema.Model{ID: "reused", ModelCode: "a", ModelName: "Reused", SpaceCode: "default", Enabled: 1, RequestTypes: `["chat_completion"]`}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{ID: "ep-reused", Code: "ep-reused", ModelID: "reused", ProviderID: "provider", Enabled: 1, Protocol: "openai", URL: "https://reused.example.test/v1"}).Error)
	require.NoError(t, sync.SyncModelByCode(ctx, "a"))
	reusedJSON, err := server.Get("aigw:config:endpoints:a")
	require.NoError(t, err)
	require.Contains(t, reusedJSON, "ep-reused")
	require.NoError(t, b.Sync(ctx, "b"))
	after, err := server.Get("aigw:config:endpoints:a")
	require.NoError(t, err)
	require.JSONEq(t, reusedJSON, after, "reused code belongs to a different model and must not be deleted")
}

func TestSmartExportUsesOneDependencySnapshot(t *testing.T) {
	source, original, ctx := smartFixture(t)
	createSmartFixture(t, original, ctx)
	seedSmartEndpoints(t, source)
	// A real WAL-backed database lets the injected writer commit while an
	// export read transaction holds its earlier consistent snapshot.
	path := filepath.Join(t.TempDir(), "smart-export.db")
	require.NoError(t, source.Exec("VACUUM INTO ?", path).Error)
	db, err := gorm.Open(sqlite.Open(path+"?_journal_mode=WAL&_busy_timeout=1000"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	b := newModelDeleteTestBiz(db)
	updated := false
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register("smart:disable-between-export-queries", func(tx *gorm.DB) {
		if updated || tx.Statement.Table != "endpoint" {
			return
		}
		updated = true
		require.NoError(t, b.ToggleEnabled(ctx, "a", &schema.ModelEnabledForm{Enabled: 0}))
	}))
	t.Cleanup(func() { _ = db.Callback().Query().Remove("smart:disable-between-export-queries") })
	ClearGatewayConfigCache()
	got, err := (&GatewaySync{DB: db}).GetGatewayConfig(ctx, "smart")
	require.NoError(t, err)
	require.True(t, updated)
	judge := got.Models["smart"].SmartRouting.JudgeModel
	require.Contains(t, got.Models, judge, "an enabled dependency must accompany its smart definition from the same read snapshot")
	next, err := (&GatewaySync{DB: db}).GetGatewayConfig(ctx, "smart")
	require.NoError(t, err)
	require.Equal(t, "a", next.Models["smart"].SmartRouting.JudgeModel)
	require.NotContains(t, next.Models, "a", "an in-flight old snapshot must not refill the cache after the disable invalidated it")
}

func TestSmartAvailabilityFailureDoesNotClaimCompletePublication(t *testing.T) {
	for _, state := range []struct {
		name    string
		enabled int
	}{{"disable", 0}, {"enable", 1}} {
		t.Run(state.name, func(t *testing.T) {
			db, b, ctx := smartFixture(t)
			sync, server := smartRedisFixture(t, db)
			b.ConfigRedisSync = sync
			model := createSmartFixture(t, b, ctx)
			require.NoError(t, db.Exec("CREATE TABLE tenant_endpoint (tenant_code TEXT, endpoint_id TEXT)").Error)
			require.NoError(t, db.Exec("INSERT INTO tenant_model (tenant_code, model_id) VALUES (?, ?)", "tenant-a", model.ID).Error)
			require.NoError(t, b.ModelDAL.UpdateEnabled(ctx, model.ID, 1-state.enabled, "alice"))
			server.Set("aigw:tenant:tenant-a:models", "wrong-type")
			form := &schema.ModelEnabledForm{Enabled: state.enabled}
			require.NoError(t, b.ToggleEnabled(ctx, model.ID, form))
			require.Equal(t, "failed", form.Result.SyncStatus, "routing alone is not complete publication when tenant binding publication failed")
		})
	}
}

func TestSmartReusedCodeRecoveryRebuildsTenantAccessFromCurrentOwner(t *testing.T) {
	db, b, ctx := smartFixture(t)
	sync, server := smartRedisFixture(t, db)
	b.ConfigRedisSync = sync
	seedSmartEndpoints(t, db)
	require.NoError(t, db.Exec("CREATE TABLE tenant_endpoint (tenant_code TEXT, endpoint_id TEXT)").Error)
	for _, tenant := range []string{"old-only", "shared"} {
		require.NoError(t, db.Exec("INSERT INTO tenant_model (tenant_code, model_id) VALUES (?, 'a')", tenant).Error)
		require.NoError(t, db.Exec("INSERT INTO tenant_endpoint (tenant_code, endpoint_id) VALUES (?, 'ep-a')", tenant).Error)
	}
	require.NoError(t, b.Sync(ctx, "a"))
	require.NoError(t, sync.RedisClient.SAdd(ctx, "aigw:tenant:old-only:models", "unrelated").Err())
	require.NoError(t, db.Exec("DELETE FROM tenant_model WHERE model_id = 'a'").Error)
	require.NoError(t, db.Exec("DELETE FROM tenant_endpoint WHERE endpoint_id = 'ep-a'").Error)
	require.NoError(t, db.Exec("DELETE FROM endpoint WHERE model_id = 'a'").Error)
	require.NoError(t, b.Delete(ctx, "a"))

	require.NoError(t, db.Create(&schema.Model{ID: "reused", ModelCode: "a", ModelName: "Reused", SpaceCode: "default", Enabled: 1, RequestTypes: `["chat_completion"]`}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{ID: "ep-reused", Code: "ep-reused", ModelID: "reused", ProviderID: "provider", Enabled: 1, Protocol: "openai", URL: "https://reused.example.test/v1"}).Error)
	for _, tenant := range []string{"new-only", "shared", "unrestricted"} {
		require.NoError(t, db.Exec("INSERT INTO tenant_model (tenant_code, model_id) VALUES (?, 'reused')", tenant).Error)
	}
	for _, tenant := range []string{"new-only", "shared"} {
		require.NoError(t, db.Exec("INSERT INTO tenant_endpoint (tenant_code, endpoint_id) VALUES (?, 'ep-reused')", tenant).Error)
	}
	require.NoError(t, sync.SyncModelByCode(ctx, "a"))
	routing, err := server.Get("aigw:config:endpoints:a")
	require.NoError(t, err)
	for _, tenant := range []string{"new-only", "unrestricted", "orphan"} {
		require.NoError(t, sync.RedisClient.SAdd(ctx, "aigw:tenant:"+tenant+":models", "a", "unrelated").Err())
		endpoint := "ep-a"
		if tenant == "new-only" {
			endpoint = "ep-reused"
		}
		require.NoError(t, sync.RedisClient.SAdd(ctx, "aigw:tenant:"+tenant+":model:a:endpoints", endpoint).Err())
	}

	require.NoError(t, sync.syncModelAvailability(ctx, "a", "a"))
	after, err := server.Get("aigw:config:endpoints:a")
	require.NoError(t, err)
	require.JSONEq(t, routing, after, "preserve the current owner's routing")
	for _, tc := range []struct {
		tenant    string
		allowed   []string
		endpoints []string
	}{
		{"old-only", []string{"unrelated"}, nil},
		{"new-only", []string{"a", "unrelated"}, []string{"ep-reused"}},
		{"shared", []string{"a"}, []string{"ep-reused"}},
		{"unrestricted", []string{"a", "unrelated"}, nil},
		{"orphan", []string{"unrelated"}, nil},
	} {
		t.Run(tc.tenant, func(t *testing.T) {
			members, err := sync.RedisClient.SMembers(ctx, "aigw:tenant:"+tc.tenant+":models").Result()
			require.NoError(t, err)
			require.ElementsMatch(t, tc.allowed, members)
			endpoints, err := sync.RedisClient.SMembers(ctx, "aigw:tenant:"+tc.tenant+":model:a:endpoints").Result()
			require.NoError(t, err)
			require.ElementsMatch(t, tc.endpoints, endpoints)
		})
	}
}
