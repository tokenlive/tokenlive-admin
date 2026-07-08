package dal

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"gorm.io/gorm"
)

func TestModelDALLookupUsesConfiguredTableName(t *testing.T) {
	restoreModelTablePrefix(t, "ut_")
	db := newModelTestDB(t)
	require.NoError(t, db.AutoMigrate(&schema.Model{}))
	require.NoError(t, db.Create(&schema.Model{
		ID:           "model-1",
		ModelName:    "GLM 5",
		ModelCode:    "GLM-5",
		SpaceCode:    "default",
		RequestTypes: `["chat_completion"]`,
		Deleted:      "0",
	}).Error)

	modelDAL := &Model{DB: db}
	modelCode, err := modelDAL.GetModelCodeByID(context.Background(), "model-1")
	require.NoError(t, err)
	require.Equal(t, "GLM-5", modelCode)

	modelID, err := modelDAL.GetIDByModelCode(context.Background(), "GLM-5")
	require.NoError(t, err)
	require.Equal(t, "model-1", modelID)
}

func newModelTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func restoreModelTablePrefix(t *testing.T, tablePrefix string) {
	t.Helper()

	oldPrefix := config.C.Storage.DB.TablePrefix
	config.C.Storage.DB.TablePrefix = tablePrefix
	t.Cleanup(func() {
		config.C.Storage.DB.TablePrefix = oldPrefix
	})
}
