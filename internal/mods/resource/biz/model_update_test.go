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

func newModelUpdateTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&schema.Model{},
		&schema.Endpoint{},
		&schema.ModelAlias{},
		&opsSchema.AuditLog{},
	))
	require.NoError(t, db.Exec("CREATE TABLE IF NOT EXISTS tenant_model (tenant_code varchar(64), model_id varchar(20))").Error)
	return db
}

func newModelUpdateTestBiz(db *gorm.DB) *Model {
	trans := &util.Trans{DB: db}
	return &Model{
		Trans:           trans,
		ModelDAL:        &dal.Model{DB: db},
		ConfigRedisSync: &ConfigRedisSync{},
		AuditLogBIZ: &opsBiz.AuditLog{
			Trans:       trans,
			AuditLogDAL: &opsDal.AuditLog{DB: db},
		},
	}
}

func newModelUpdateTestContext() context.Context {
	ctx := context.Background()
	ctx = util.NewUsername(ctx, "alice")
	ctx = util.NewTenant(ctx, "tenant-a")
	return ctx
}

func TestModelUpdate_Success(t *testing.T) {
	db := newModelUpdateTestDB(t)
	biz := newModelUpdateTestBiz(db)
	ctx := newModelUpdateTestContext()

	require.NoError(t, db.Create(&schema.Model{
		ID:              "model-1",
		ModelName:       "Model Old",
		ModelCode:       "old-model-code",
		SpaceCode:       "default",
		RequestTypes:    `["chat_completion"]`,
		ContextLength:   128000,
		MaxOutputTokens: 8192,
		CreatedAt:       time.Now(),
		Deleted:         "0",
	}).Error)

	updateForm := &schema.ModelForm{
		ModelName:       "Model New",
		ModelCode:       "new-model-code",
		SpaceCode:       "default",
		RequestTypes:    `["chat_completion"]`,
		ContextLength:   128000,
		MaxOutputTokens: 8192,
		Enabled:         1,
	}

	err := biz.Update(ctx, "model-1", updateForm)
	require.NoError(t, err)

	updatedModel, err := biz.ModelDAL.Get(ctx, "model-1")
	require.NoError(t, err)
	require.NotNil(t, updatedModel)
	require.Equal(t, "new-model-code", updatedModel.ModelCode)
	require.Equal(t, "Model New", updatedModel.ModelName)
	require.Equal(t, "alice", updatedModel.Modifier)
}

func TestModelUpdate_DuplicateModelCode(t *testing.T) {
	db := newModelUpdateTestDB(t)
	biz := newModelUpdateTestBiz(db)
	ctx := newModelUpdateTestContext()

	require.NoError(t, db.Create(&schema.Model{
		ID:        "model-1",
		ModelName: "Model One",
		ModelCode: "code-1",
		SpaceCode: "default",
		CreatedAt: time.Now(),
		Deleted:   "0",
	}).Error)

	require.NoError(t, db.Create(&schema.Model{
		ID:        "model-2",
		ModelName: "Model Two",
		ModelCode: "code-2",
		SpaceCode: "default",
		CreatedAt: time.Now(),
		Deleted:   "0",
	}).Error)

	updateForm := &schema.ModelForm{
		ModelName: "Model Two",
		ModelCode: "code-1", // duplicate with model-1
		SpaceCode: "default",
	}

	err := biz.Update(ctx, "model-2", updateForm)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Model code already exists")
}

func TestModelUpdate_DuplicateModelName(t *testing.T) {
	db := newModelUpdateTestDB(t)
	biz := newModelUpdateTestBiz(db)
	ctx := newModelUpdateTestContext()

	require.NoError(t, db.Create(&schema.Model{
		ID:        "model-1",
		ModelName: "Model One",
		ModelCode: "code-1",
		SpaceCode: "default",
		CreatedAt: time.Now(),
		Deleted:   "0",
	}).Error)

	require.NoError(t, db.Create(&schema.Model{
		ID:        "model-2",
		ModelName: "Model Two",
		ModelCode: "code-2",
		SpaceCode: "default",
		CreatedAt: time.Now(),
		Deleted:   "0",
	}).Error)

	updateForm := &schema.ModelForm{
		ModelName: "Model One", // duplicate with model-1
		ModelCode: "code-2",
		SpaceCode: "default",
	}

	err := biz.Update(ctx, "model-2", updateForm)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Model name already exists")
}
