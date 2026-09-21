package adminapp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/bootstrap"
	"github.com/tokenlive/tokenlive-admin/internal/mods"
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/internal/wirex"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoadGatewaySnapshotCarriesSmartRoutingContract(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close(); biz.ClearGatewayConfigCache() })
	require.NoError(t, db.AutoMigrate(
		&schema.Model{}, &schema.Endpoint{}, &schema.Provider{}, &schema.ModelAlias{},
		&policySchema.PolicyLoadbalance{}, &policySchema.PolicyInvocation{}, &policySchema.PolicyLimit{},
		&policySchema.PolicyCircuitBreak{}, &policySchema.PolicyTagging{}, &policySchema.PolicyRoute{}, &policySchema.PolicyRouteDetail{},
	))
	for _, sql := range []string{
		"CREATE TABLE user (id TEXT, tenant TEXT)",
		"CREATE TABLE user_api_key (user_id TEXT, api_key TEXT, status INTEGER, credits INTEGER, expires_at DATETIME, deleted TEXT)",
		"CREATE TABLE tenant (code TEXT, api_key TEXT, status TEXT, deleted TEXT)",
	} {
		require.NoError(t, db.Exec(sql).Error)
	}
	for _, id := range []string{"judge", "strong"} {
		require.NoError(t, db.Create(&schema.Model{ID: id, ModelCode: id + "-code", ModelName: id, SpaceCode: "test", RequestTypes: `["chat_completion"]`, Enabled: 1}).Error)
	}
	require.NoError(t, db.Create(&schema.Model{
		ID: "composite", ModelCode: "composite", ModelName: "Composite", ModelType: "smart", Enabled: 1, SpaceCode: "test", RequestTypes: `["chat_completion"]`,
		SmartRouting: &schema.SmartRouting{Version: 3, JudgeModelID: "judge", JudgeTimeoutMS: 5000, JudgeMaxInputBytes: 65536, JudgeMaxOutputTokens: 256,
			Ranges: []schema.SmartRange{{Min: 0, Max: 50, ModelID: "judge"}, {Min: 50, Max: 100, ModelID: "strong"}}},
	}).Error)
	biz.ClearGatewayConfigCache()
	app := &App{rt: &bootstrap.Runtime{Injector: &wirex.Injector{M: &mods.Mods{Resource: &resource.Resource{
		GatewaySyncAPI: &api.GatewaySync{GatewaySyncBIZ: &biz.GatewaySync{DB: db}},
	}}}}}
	snapshot, err := app.LoadGatewaySnapshot(context.Background())
	require.NoError(t, err)
	var wire struct {
		Models map[string]json.RawMessage `json:"models"`
	}
	require.NoError(t, json.Unmarshal(snapshot.ConfigJSON, &wire))
	require.JSONEq(t, `{"model_type":"smart","request_types":["chat_completion"],"endpoints":[],"smart_routing":{"version":3,"judge_model":"judge-code","judge_timeout_ms":5000,"judge_max_input_bytes":65536,"judge_max_output_tokens":256,"ranges":[{"min":0,"max":50,"model":"judge-code"},{"min":50,"max":100,"model":"strong-code"}]}}`, string(wire.Models["composite"]))
}
