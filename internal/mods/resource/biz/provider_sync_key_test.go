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
	biz := newProviderSyncTestBiz(db)
	ctx := newProviderSyncTestContext()

	// Seed Provider with 2 ApiKeys
	initialKeys := []schema.ApiKeyItem{
		{Value: "sk-old-1", Description: "Key 1"},
		{Value: "sk-old-2", Description: "Key 2"},
	}
	initialKeysRaw, _ := json.Marshal(initialKeys)
	require.NoError(t, db.Create(&schema.Provider{
		ID:        "provider-1",
		Code:      "openai-prov",
		Name:      "OpenAI",
		Protocol:  "openai",
		ApiKeys:   initialKeysRaw,
		CreatedAt: time.Now(),
		Deleted:   "0",
	}).Error)

	// Seed Endpoints
	// Endpoint 1: Override with sk-old-1 (should sync)
	// Endpoint 2: Override with sk-old-2 (unchanged)
	// Endpoint 3: Inherits provider key (api_key = "", unchanged)
	// Endpoint 4: Belongs to another provider (should not change even if api_key = sk-old-1)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "ep-1",
		Code:       "ep-code-1",
		ModelID:    "model-1",
		ProviderID: "provider-1",
		URL:        "https://api.openai.com/v1",
		ApiKey:     "sk-old-1",
		Deleted:    "0",
	}).Error)

	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "ep-2",
		Code:       "ep-code-2",
		ModelID:    "model-1",
		ProviderID: "provider-1",
		URL:        "https://api.openai.com/v1",
		ApiKey:     "sk-old-2",
		Deleted:    "0",
	}).Error)

	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "ep-3",
		Code:       "ep-code-3",
		ModelID:    "model-1",
		ProviderID: "provider-1",
		URL:        "https://api.openai.com/v1",
		ApiKey:     "",
		Deleted:    "0",
	}).Error)

	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "ep-4",
		Code:       "ep-code-4",
		ModelID:    "model-1",
		ProviderID: "provider-2",
		URL:        "https://other.com/v1",
		ApiKey:     "sk-old-1",
		Deleted:    "0",
	}).Error)

	// Update Provider: change Key 1 from sk-old-1 -> sk-new-1, keep Key 2 as sk-old-2
	updateForm := &schema.ProviderForm{
		Code:     "openai-prov",
		Name:     "OpenAI Updated",
		Protocol: "openai",
		ApiKeys: []schema.ApiKeyItem{
			{Value: "sk-new-1", Description: "Key 1 Updated"},
			{Value: "sk-old-2", Description: "Key 2"},
		},
	}

	err := biz.Update(ctx, "provider-1", updateForm)
	require.NoError(t, err)

	// Verify ep-1 has synced api_key to sk-new-1
	var ep1 schema.Endpoint
	require.NoError(t, db.First(&ep1, "id = ?", "ep-1").Error)
	require.Equal(t, "sk-new-1", ep1.ApiKey)

	// Verify ep-2 is still sk-old-2
	var ep2 schema.Endpoint
	require.NoError(t, db.First(&ep2, "id = ?", "ep-2").Error)
	require.Equal(t, "sk-old-2", ep2.ApiKey)

	// Verify ep-3 is still ""
	var ep3 schema.Endpoint
	require.NoError(t, db.First(&ep3, "id = ?", "ep-3").Error)
	require.Equal(t, "", ep3.ApiKey)

	// Verify ep-4 (different provider) is still sk-old-1
	var ep4 schema.Endpoint
	require.NoError(t, db.First(&ep4, "id = ?", "ep-4").Error)
	require.Equal(t, "sk-old-1", ep4.ApiKey)
}

func newProviderSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
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
