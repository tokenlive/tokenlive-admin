package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsDal "github.com/tokenlive/tokenlive-admin/internal/mods/ops/dal"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestProviderUpdateSyncsAssociatedEndpointApiKey(t *testing.T) {
	db := newProviderSyncTestDB(t)
	providerBiz := newProviderSyncTestBiz(db)
	ctx := newProviderSyncTestContext()

	initialKeys := []schema.ApiKeyItem{
		{ID: "key-1", Value: "sk-old-1", Description: "Key 1"},
		{ID: "key-2", Value: "sk-old-2", Description: "Key 2"},
	}
	initialKeysRaw, _ := json.Marshal(initialKeys)
	require.NoError(t, db.Create(&schema.Provider{
		ID:        "provider-1",
		Code:      "openai-prov",
		Name:      "OpenAI",
		Protocol:  "openai",
		AuthType:  "api_key",
		ApiKeys:   initialKeysRaw,
		CreatedAt: time.Now(),
		Deleted:   "0",
	}).Error)

	// ep-1: bound to key-1 (should cascade to sk-new-1 even when key order changes)
	// ep-2: bound to key-2 (unchanged)
	// ep-3: key inheritance (provider_key_id="", api_key="", unchanged)
	// ep-4: belongs to another provider (unchanged)
	// ep-5: custom override key not in provider (unchanged)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:            "ep-1",
		Code:          "ep-code-1",
		ModelID:       "model-1",
		ProviderID:    "provider-1",
		URL:           "https://api.openai.com/v1",
		ProviderKeyID: "key-1",
		ApiKey:        "sk-old-1",
		Enabled:       1,
		Deleted:       "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:            "ep-2",
		Code:          "ep-code-2",
		ModelID:       "model-1",
		ProviderID:    "provider-1",
		URL:           "https://api.openai.com/v1",
		ProviderKeyID: "key-2",
		ApiKey:        "sk-old-2",
		Enabled:       1,
		Deleted:       "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "ep-3",
		Code:       "ep-code-3",
		ModelID:    "model-1",
		ProviderID: "provider-1",
		URL:        "https://api.openai.com/v1",
		ApiKey:     "",
		Enabled:    1,
		Deleted:    "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "ep-4",
		Code:       "ep-code-4",
		ModelID:    "model-1",
		ProviderID: "provider-2",
		URL:        "https://other.com/v1",
		ApiKey:     "sk-old-1",
		Enabled:    1,
		Deleted:    "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "ep-5",
		Code:       "ep-code-5",
		ModelID:    "model-1",
		ProviderID: "provider-1",
		URL:        "https://api.openai.com/v1",
		ApiKey:     "sk-custom-override",
		Enabled:    1,
		Deleted:    "0",
	}).Error)

	// Reorder keys AND change key-1 value to sk-new-1.
	updateForm := &schema.ProviderForm{
		Code:     "openai-prov",
		Name:     "OpenAI Updated",
		Protocol: "openai",
		AuthType: "api_key",
		ApiKeys: []schema.ApiKeyItem{
			{ID: "key-2", Value: "sk-old-2", Description: "Key 2"},
			{ID: "key-1", Value: "sk-new-1", Description: "Key 1 Updated"},
		},
	}

	require.NoError(t, providerBiz.Update(ctx, "provider-1", updateForm))

	var ep1, ep2, ep3, ep4, ep5 schema.Endpoint
	require.NoError(t, db.First(&ep1, "id = ?", "ep-1").Error)
	require.Equal(t, "sk-new-1", ep1.ApiKey)
	require.Equal(t, "key-1", ep1.ProviderKeyID)
	require.Equal(t, "alice", ep1.Modifier)

	require.NoError(t, db.First(&ep2, "id = ?", "ep-2").Error)
	require.Equal(t, "sk-old-2", ep2.ApiKey)

	require.NoError(t, db.First(&ep3, "id = ?", "ep-3").Error)
	require.Equal(t, "", ep3.ApiKey)

	require.NoError(t, db.First(&ep4, "id = ?", "ep-4").Error)
	require.Equal(t, "sk-old-1", ep4.ApiKey)

	require.NoError(t, db.First(&ep5, "id = ?", "ep-5").Error)
	require.Equal(t, "sk-custom-override", ep5.ApiKey)

	var epAuditCount int64
	require.NoError(t, db.Model(&opsSchema.AuditLog{}).
		Where("resource_type = ? AND resource_id = ?", opsSchema.AuditResourceTypeEndpoint, "ep-1").
		Count(&epAuditCount).Error)
	require.Equal(t, int64(1), epAuditCount)
}

func TestProviderDeleteKeyDisablesBoundEndpointsAndReportsImpact(t *testing.T) {
	db := newProviderSyncTestDB(t)
	providerBiz := newProviderSyncTestBiz(db)
	ctx := newProviderSyncTestContext()

	initialKeysRaw, _ := json.Marshal([]schema.ApiKeyItem{
		{ID: "key-1", Value: "sk-1", Description: "Primary"},
		{ID: "key-2", Value: "sk-2", Description: "Backup"},
	})
	require.NoError(t, db.Create(&schema.Provider{
		ID:       "provider-del",
		Code:     "prov-del",
		Name:     "Prov Del",
		Protocol: "openai",
		AuthType: "api_key",
		ApiKeys:  initialKeysRaw,
		Deleted:  "0",
	}).Error)

	require.NoError(t, db.Create(&schema.Endpoint{
		ID:            "ep-enabled",
		Code:          "ep-enabled",
		ModelID:       "m-1",
		ProviderID:    "provider-del",
		URL:           "https://api.openai.com/v1",
		ProviderKeyID: "key-1",
		ApiKey:        "sk-1",
		Enabled:       1,
		Deleted:       "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:            "ep-already-disabled",
		Code:          "ep-already-disabled",
		ModelID:       "m-2",
		ProviderID:    "provider-del",
		URL:           "https://api.openai.com/v1",
		ProviderKeyID: "key-1",
		ApiKey:        "sk-1",
		Enabled:       0,
		Deleted:       "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:            "ep-kept",
		Code:          "ep-kept",
		ModelID:       "m-1",
		ProviderID:    "provider-del",
		URL:           "https://api.openai.com/v1",
		ProviderKeyID: "key-2",
		ApiKey:        "sk-2",
		Enabled:       1,
		Deleted:       "0",
	}).Error)

	// Delete key-1 and add a brand-new key-new (without key-1's ID).
	form := &schema.ProviderForm{
		Code:     "prov-del",
		Name:     "Prov Del",
		Protocol: "openai",
		AuthType: "api_key",
		ApiKeys: []schema.ApiKeyItem{
			{ID: "key-2", Value: "sk-2", Description: "Backup"},
			{Value: "sk-brand-new", Description: "New Key"},
		},
	}

	impact, err := providerBiz.PreviewKeyImpact(ctx, "provider-del", form)
	require.NoError(t, err)
	require.Equal(t, 1, impact.AffectedEndpointCount)
	require.Len(t, impact.AffectedEndpoints, 1)
	require.Equal(t, "ep-enabled", impact.AffectedEndpoints[0].ID)

	require.NoError(t, providerBiz.Update(ctx, "provider-del", form))

	var epEnabled, epDisabled, epKept schema.Endpoint
	require.NoError(t, db.First(&epEnabled, "id = ?", "ep-enabled").Error)
	require.Equal(t, 0, epEnabled.Enabled)
	require.Equal(t, "", epEnabled.ProviderKeyID)
	require.Equal(t, "sk-1", epEnabled.ApiKey) // kept old key to prevent accidental inherit on re-enable

	require.NoError(t, db.First(&epDisabled, "id = ?", "ep-already-disabled").Error)
	require.Equal(t, 0, epDisabled.Enabled)
	require.Equal(t, "", epDisabled.ProviderKeyID)

	require.NoError(t, db.First(&epKept, "id = ?", "ep-kept").Error)
	require.Equal(t, 1, epKept.Enabled)
	require.Equal(t, "key-2", epKept.ProviderKeyID)
	require.Equal(t, "sk-2", epKept.ApiKey)
}

func TestProviderUpdateRejectsDuplicateKeysAndRouteCollisions(t *testing.T) {
	db := newProviderSyncTestDB(t)
	providerBiz := newProviderSyncTestBiz(db)
	ctx := newProviderSyncTestContext()

	initialKeysRaw, _ := json.Marshal([]schema.ApiKeyItem{
		{ID: "key-1", Value: "sk-1"},
		{ID: "key-2", Value: "sk-2"},
	})
	require.NoError(t, db.Create(&schema.Provider{
		ID:       "provider-dup",
		Code:     "prov-dup",
		Name:     "Prov Dup",
		Protocol: "openai",
		AuthType: "api_key",
		ApiKeys:  initialKeysRaw,
		Deleted:  "0",
	}).Error)

	// ep-bound binds key-1; ep-override manually overrides with sk-custom.
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:            "ep-bound",
		Code:          "ep-bound",
		ModelID:       "m-1",
		ProviderID:    "provider-dup",
		URL:           "https://api.openai.com/v1",
		ProviderKeyID: "key-1",
		ApiKey:        "sk-1",
		Enabled:       1,
		Deleted:       "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "ep-override",
		Code:       "ep-override",
		ModelID:    "m-1",
		ProviderID: "provider-dup",
		URL:        "https://api.openai.com/v1",
		ApiKey:     "sk-custom",
		Enabled:    1,
		Deleted:    "0",
	}).Error)

	// 1. Duplicate key values inside the same provider are rejected.
	err := providerBiz.Update(ctx, "provider-dup", &schema.ProviderForm{
		Code:     "prov-dup",
		Name:     "Prov Dup",
		Protocol: "openai",
		AuthType: "api_key",
		ApiKeys: []schema.ApiKeyItem{
			{ID: "key-1", Value: "sk-same"},
			{ID: "key-2", Value: "sk-same"},
		},
	})
	require.Error(t, err)

	// 2. Updating key-1 to sk-custom collides with ep-override on the same model route; rolls back cleanly.
	err = providerBiz.Update(ctx, "provider-dup", &schema.ProviderForm{
		Code:     "prov-dup",
		Name:     "Prov Dup",
		Protocol: "openai",
		AuthType: "api_key",
		ApiKeys: []schema.ApiKeyItem{
			{ID: "key-1", Value: "sk-custom"},
			{ID: "key-2", Value: "sk-2"},
		},
	})
	require.Error(t, err)

	var epBound schema.Endpoint
	require.NoError(t, db.First(&epBound, "id = ?", "ep-bound").Error)
	require.Equal(t, "sk-1", epBound.ApiKey)
}

func TestProviderSwitchToOAuthDisablesBoundEndpoints(t *testing.T) {
	db := newProviderSyncTestDB(t)
	providerBiz := newProviderSyncTestBiz(db)
	ctx := newProviderSyncTestContext()

	initialKeysRaw, _ := json.Marshal([]schema.ApiKeyItem{
		{ID: "key-1", Value: "sk-1"},
	})
	require.NoError(t, db.Create(&schema.Provider{
		ID:       "provider-oauth",
		Code:     "prov-oauth",
		Name:     "Prov OAuth",
		Protocol: "openai",
		AuthType: "api_key",
		ApiKeys:  initialKeysRaw,
		Deleted:  "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:            "ep-bound",
		Code:          "ep-bound",
		ModelID:       "m-1",
		ProviderID:    "provider-oauth",
		URL:           "https://api.openai.com/v1",
		ProviderKeyID: "key-1",
		ApiKey:        "sk-1",
		Enabled:       1,
		Deleted:       "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "ep-inherit",
		Code:       "ep-inherit",
		ModelID:    "m-1",
		ProviderID: "provider-oauth",
		URL:        "https://api.openai.com/v1",
		ApiKey:     "",
		Enabled:    1,
		Deleted:    "0",
	}).Error)

	form := &schema.ProviderForm{
		Code:     "prov-oauth",
		Name:     "Prov OAuth",
		Protocol: "openai",
		AuthType: "oauth_token",
		ApiKeys:  []schema.ApiKeyItem{{Value: "oauth-access-token"}},
	}

	impact, err := providerBiz.PreviewKeyImpact(ctx, "provider-oauth", form)
	require.NoError(t, err)
	require.Equal(t, 1, impact.AffectedEndpointCount)

	require.NoError(t, providerBiz.Update(ctx, "provider-oauth", form))

	var epBound, epInherit schema.Endpoint
	require.NoError(t, db.First(&epBound, "id = ?", "ep-bound").Error)
	require.Equal(t, 0, epBound.Enabled)
	require.Equal(t, "", epBound.ProviderKeyID)
	require.Equal(t, "sk-1", epBound.ApiKey)

	require.NoError(t, db.First(&epInherit, "id = ?", "ep-inherit").Error)
	require.Equal(t, 1, epInherit.Enabled)
}

func TestBackfillProviderKeyBindingsAndLegacyFallbackClaim(t *testing.T) {
	db := newProviderSyncTestDB(t)
	providerBiz := newProviderSyncTestBiz(db)
	ctx := newProviderSyncTestContext()

	// Seed legacy provider without key IDs.
	raw, _ := json.Marshal([]schema.ApiKeyItem{{Value: "sk-legacy-1", Description: "Legacy"}})
	require.NoError(t, db.Create(&schema.Provider{
		ID:       "prov-legacy",
		Code:     "prov-legacy",
		Name:     "Legacy",
		Protocol: "openai",
		AuthType: "api_key",
		ApiKeys:  raw,
		Deleted:  "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "ep-legacy-match",
		Code:       "ep-legacy-match",
		ModelID:    "m-1",
		ProviderID: "prov-legacy",
		URL:        "https://api.openai.com/v1",
		ApiKey:     "sk-legacy-1",
		Enabled:    1,
		Deleted:    "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "ep-legacy-override",
		Code:       "ep-legacy-override",
		ModelID:    "m-1",
		ProviderID: "prov-legacy",
		URL:        "https://api.openai.com/v1",
		ApiKey:     "sk-custom",
		Enabled:    1,
		Deleted:    "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "ep-legacy-inherit",
		Code:       "ep-legacy-inherit",
		ModelID:    "m-1",
		ProviderID: "prov-legacy",
		URL:        "https://api.openai.com/v1",
		ApiKey:     "",
		Enabled:    1,
		Deleted:    "0",
	}).Error)

	require.NoError(t, BackfillProviderKeyBindings(ctx, db))
	// Second invocation is idempotent.
	require.NoError(t, BackfillProviderKeyBindings(ctx, db))

	var p schema.Provider
	require.NoError(t, db.First(&p, "id = ?", "prov-legacy").Error)
	keys := p.GetApiKeys()
	require.Len(t, keys, 1)
	require.NotEmpty(t, keys[0].ID)

	var epMatch, epOverride, epInherit schema.Endpoint
	require.NoError(t, db.First(&epMatch, "id = ?", "ep-legacy-match").Error)
	require.Equal(t, keys[0].ID, epMatch.ProviderKeyID)

	require.NoError(t, db.First(&epOverride, "id = ?", "ep-legacy-override").Error)
	require.Equal(t, "", epOverride.ProviderKeyID)

	require.NoError(t, db.First(&epInherit, "id = ?", "ep-legacy-inherit").Error)
	require.Equal(t, "", epInherit.ProviderKeyID)

	// Now update the provider key value and verify ep-legacy-match cascades.
	require.NoError(t, providerBiz.Update(ctx, "prov-legacy", &schema.ProviderForm{
		Code:     "prov-legacy",
		Name:     "Legacy",
		Protocol: "openai",
		AuthType: "api_key",
		ApiKeys:  []schema.ApiKeyItem{{ID: keys[0].ID, Value: "sk-legacy-rotated", Description: "Legacy"}},
	}))

	require.NoError(t, db.First(&epMatch, "id = ?", "ep-legacy-match").Error)
	require.Equal(t, "sk-legacy-rotated", epMatch.ApiKey)
}

func TestEndpointCreateAndUpdateWithProviderKeyBinding(t *testing.T) {
	db := newProviderSyncTestDB(t)
	trans := &util.Trans{DB: db}
	epBiz := &Endpoint{
		Trans:             trans,
		EndpointDAL:       &dal.Endpoint{DB: db},
		ModelDAL:          &dal.Model{DB: db},
		ProviderDAL:       &dal.Provider{DB: db},
		ConfigRedisSync:   &ConfigRedisSync{},
		DataPermissionBIZ: &DataPermission{Trans: trans, DataPermissionDAL: &dal.DataPermission{DB: db}},
		AuditLogBIZ:       &opsBiz.AuditLog{Trans: trans, AuditLogDAL: &opsDal.AuditLog{DB: db}},
	}
	ctx := newProviderSyncTestContext()

	require.NoError(t, db.Create(&schema.Model{
		ID: "model-1", ModelName: "GPT", ModelCode: "gpt", SpaceCode: "default", RequestTypes: `["chat_completion"]`, Enabled: 1, Deleted: "0",
	}).Error)
	raw, _ := json.Marshal([]schema.ApiKeyItem{
		{ID: "key-1", Value: "sk-live-1"},
	})
	require.NoError(t, db.Create(&schema.Provider{
		ID: "prov-1", Code: "prov-1", Name: "Prov 1", Protocol: "openai", AuthType: "api_key", ApiKeys: raw, Enabled: 1, Deleted: "0",
	}).Error)

	// Creating endpoint with provider_key_id mirrors api_key from provider even if client omits api_key.
	created, err := epBiz.Create(ctx, &schema.EndpointForm{
		Code:          "ep-bound-create",
		ProviderID:    "prov-1",
		ModelID:       "model-1",
		URL:           "https://api.openai.com/v1",
		AuthType:      "api_key",
		ProviderKeyID: "key-1",
		Enabled:       1,
	})
	require.NoError(t, err)
	require.Equal(t, "key-1", created.ProviderKeyID)
	require.Equal(t, "sk-live-1", created.ApiKey)

	// Invalid provider_key_id is rejected.
	_, err = epBiz.Create(ctx, &schema.EndpointForm{
		Code:          "ep-bound-bad",
		ProviderID:    "prov-1",
		ModelID:       "model-1",
		URL:           "https://api.openai.com/v1",
		AuthType:      "api_key",
		ProviderKeyID: "non-existent",
		Enabled:       1,
	})
	require.Error(t, err)

	// Updating to custom override clears provider_key_id.
	require.NoError(t, epBiz.Update(ctx, created.ID, &schema.EndpointForm{
		Code:       "ep-bound-create",
		ProviderID: "prov-1",
		ModelID:    "model-1",
		URL:        "https://api.openai.com/v1",
		AuthType:   "api_key",
		ApiKey:     "sk-manual-override",
		Enabled:    1,
	}))
	var after schema.Endpoint
	require.NoError(t, db.First(&after, "id = ?", created.ID).Error)
	require.Equal(t, "", after.ProviderKeyID)
	require.Equal(t, "sk-manual-override", after.ApiKey)
}

func newProviderSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&schema.Model{},
		&schema.Provider{},
		&schema.Endpoint{},
		&schema.DataPermission{},
		&opsSchema.AuditLog{},
	))
	return db
}

func newProviderSyncTestBiz(db *gorm.DB) *Provider {
	trans := &util.Trans{DB: db}
	return &Provider{
		Trans:             trans,
		ProviderDAL:       &dal.Provider{DB: db},
		EndpointDAL:       &dal.Endpoint{DB: db},
		ConfigRedisSync:   &ConfigRedisSync{},
		DataPermissionBIZ: &DataPermission{Trans: trans, DataPermissionDAL: &dal.DataPermission{DB: db}},
		AuditLogBIZ:       &opsBiz.AuditLog{Trans: trans, AuditLogDAL: &opsDal.AuditLog{DB: db}},
	}
}

func newProviderSyncTestContext() context.Context {
	ctx := context.Background()
	ctx = util.NewUsername(ctx, "alice")
	ctx = util.NewTenant(ctx, "tenant-a")
	return ctx
}
