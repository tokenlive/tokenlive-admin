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
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestModelDeleteRejectsAssociatedResources(t *testing.T) {
	tests := []struct {
		name        string
		seed        func(t *testing.T, db *gorm.DB)
		wantMessage string
	}{
		{
			name: "endpoint",
			seed: func(t *testing.T, db *gorm.DB) {
				require.NoError(t, db.Create(&schema.Endpoint{
					ID:         "endpoint-1",
					Code:       "endpoint-code",
					ModelID:    "model-1",
					ProviderID: "provider-1",
					URL:        "https://example.test",
					Deleted:    "0",
				}).Error)
			},
			wantMessage: "关联端点",
		},
		{
			name: "alias",
			seed: func(t *testing.T, db *gorm.DB) {
				require.NoError(t, db.Create(&schema.ModelAlias{
					ID:        "alias-1",
					SpaceCode: "default",
					Alias:     "model-alias",
					ModelId:   "model-1",
					Deleted:   "0",
				}).Error)
			},
			wantMessage: "关联别名",
		},
		{
			name: "policy binding",
			seed: func(t *testing.T, db *gorm.DB) {
				require.NoError(t, db.Create(&policySchema.PolicyBinding{
					ID:         "binding-1",
					ModelCode:  "model-code",
					PolicyType: "loadbalance",
					PolicyID:   "policy-1",
					Deleted:    "0",
				}).Error)
			},
			wantMessage: "关联策略",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newModelDeleteTestDB(t)
			biz := newModelDeleteTestBiz(db)
			seedModelDeleteModel(t, db)
			tt.seed(t, db)

			err := biz.Delete(newModelDeleteTestContext(), "model-1")

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantMessage)
			require.Contains(t, err.Error(), "请先清理")

			var model schema.Model
			require.NoError(t, db.First(&model, "id = ?", "model-1").Error)
			require.Equal(t, "0", model.Deleted)
		})
	}
}

func TestModelDeleteRemovesDataPermissionWhenNoAssociations(t *testing.T) {
	db := newModelDeleteTestDB(t)
	biz := newModelDeleteTestBiz(db)
	seedModelDeleteModel(t, db)
	require.NoError(t, db.Create(&schema.DataPermission{
		ID:         "permission-1",
		Type:       schema.DataPermissionTypeModel,
		DataId:     "model-1",
		User:       "alice",
		Tenant:     "tenant-a",
		Role:       "owner",
		Permission: 7,
		Deleted:    "0",
	}).Error)

	err := biz.Delete(newModelDeleteTestContext(), "model-1")

	require.NoError(t, err)

	var activeModels int64
	require.NoError(t, db.Model(&schema.Model{}).Where("id = ? AND deleted = '0'", "model-1").Count(&activeModels).Error)
	require.Zero(t, activeModels)

	var permissions int64
	require.NoError(t, db.Model(&schema.DataPermission{}).Where("type = ? AND data_id = ?", schema.DataPermissionTypeModel, "model-1").Count(&permissions).Error)
	require.Zero(t, permissions)
}

func newModelDeleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&schema.Model{},
		&schema.Endpoint{},
		&schema.ModelAlias{},
		&schema.DataPermission{},
		&policySchema.PolicyBinding{},
		&policySchema.PolicyLoadbalance{},
		&opsSchema.AuditLog{},
	))
	require.NoError(t, db.Exec("CREATE TABLE IF NOT EXISTS tenant_model (tenant_code varchar(64), model_id varchar(20))").Error)
	return db
}

func newModelDeleteTestBiz(db *gorm.DB) *Model {
	trans := &util.Trans{DB: db}
	return &Model{
		Trans:           trans,
		ModelDAL:        &dal.Model{DB: db},
		ConfigRedisSync: &ConfigRedisSync{},
		DataPermissionBIZ: &DataPermission{
			Trans:             trans,
			DataPermissionDAL: &dal.DataPermission{DB: db},
		},
		AuditLogBIZ: &opsBiz.AuditLog{
			Trans:       trans,
			AuditLogDAL: &opsDal.AuditLog{DB: db},
		},
	}
}

func seedModelDeleteModel(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Create(&schema.Model{
		ID:              "model-1",
		ModelName:       "Model One",
		ModelCode:       "model-code",
		SpaceCode:       "default",
		RequestTypes:    `["chat_completion"]`,
		ContextLength:   128000,
		MaxOutputTokens: 8192,
		CreatedAt:       time.Now(),
		Deleted:         "0",
	}).Error)
}

func newModelDeleteTestContext() context.Context {
	ctx := context.Background()
	ctx = util.NewUsername(ctx, "alice")
	ctx = util.NewTenant(ctx, "tenant-a")
	return ctx
}
