package biz

import (
	"context"
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

func TestProviderDeleteRejectsAssociatedEndpoint(t *testing.T) {
	db := newProviderDeleteTestDB(t)
	biz := newProviderDeleteTestBiz(db)
	seedProviderDeleteProvider(t, db)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID:         "endpoint-1",
		Code:       "endpoint-code",
		ModelID:    "model-1",
		ProviderID: "provider-1",
		URL:        "https://example.test",
		Deleted:    "0",
	}).Error)

	err := biz.Delete(newProviderDeleteTestContext(), "provider-1")

	require.Error(t, err)
	require.Contains(t, err.Error(), "关联端点")
	require.Contains(t, err.Error(), "请先清理")

	var provider schema.Provider
	require.NoError(t, db.First(&provider, "id = ?", "provider-1").Error)
	require.Equal(t, "0", provider.Deleted)
}

func TestProviderDeleteRemovesDataPermissionWhenNoAssociatedEndpoint(t *testing.T) {
	db := newProviderDeleteTestDB(t)
	biz := newProviderDeleteTestBiz(db)
	seedProviderDeleteProvider(t, db)
	require.NoError(t, db.Create(&schema.DataPermission{
		ID:         "permission-1",
		Type:       schema.DataPermissionTypeProvider,
		DataId:     "provider-1",
		User:       "alice",
		Tenant:     "tenant-a",
		Role:       "owner",
		Permission: 7,
		Deleted:    "0",
	}).Error)

	err := biz.Delete(newProviderDeleteTestContext(), "provider-1")

	require.NoError(t, err)

	var activeProviders int64
	require.NoError(t, db.Model(&schema.Provider{}).Where("id = ? AND deleted = '0'", "provider-1").Count(&activeProviders).Error)
	require.Zero(t, activeProviders)

	var permissions int64
	require.NoError(t, db.Model(&schema.DataPermission{}).Where("type = ? AND data_id = ?", schema.DataPermissionTypeProvider, "provider-1").Count(&permissions).Error)
	require.Zero(t, permissions)
}

func newProviderDeleteTestDB(t *testing.T) *gorm.DB {
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

func newProviderDeleteTestBiz(db *gorm.DB) *Provider {
	trans := &util.Trans{DB: db}
	return &Provider{
		Trans:             trans,
		ProviderDAL:       &dal.Provider{DB: db},
		ConfigRedisSync:   &ConfigRedisSync{},
		DataPermissionBIZ: &DataPermission{Trans: trans, DataPermissionDAL: &dal.DataPermission{DB: db}},
		AuditLogBIZ:       &opsBiz.AuditLog{Trans: trans, AuditLogDAL: &opsDal.AuditLog{DB: db}},
	}
}

func seedProviderDeleteProvider(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Create(&schema.Provider{
		ID:        "provider-1",
		Code:      "provider-code",
		Name:      "Provider One",
		Protocol:  "openai",
		CreatedAt: time.Now(),
		Deleted:   "0",
	}).Error)
}

func newProviderDeleteTestContext() context.Context {
	ctx := context.Background()
	ctx = util.NewUsername(ctx, "alice")
	ctx = util.NewTenant(ctx, "tenant-a")
	return ctx
}
