package policy

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gormx"
)

func TestPolicyDialectMigrationPreservesExistingRecords(t *testing.T) {
	db, err := gormx.New(gormx.Config{
		DBType: "sqlite3", DSN: filepath.Join(t.TempDir(), "admin.db"), MaxOpenConns: 1, MaxIdleConns: 1,
	})
	require.NoError(t, err)
	conn, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	require.NoError(t, db.Exec(`CREATE TABLE policy_dialect (
		id CHAR(20) PRIMARY KEY, model_id CHAR(20) NOT NULL, implementation VARCHAR(64) NOT NULL,
		enabled INT NOT NULL DEFAULT 0, creator VARCHAR(255), modifier VARCHAR(255),
		created_at DATETIME, updated_at DATETIME, deleted VARCHAR(20) NOT NULL DEFAULT '0', deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX uniq_policy_dialect_model_impl_deleted
		ON policy_dialect (model_id, implementation, deleted)`).Error)
	createdAt := time.Date(2026, time.November, 25, 1, 2, 3, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	require.NoError(t, db.Exec(`INSERT INTO policy_dialect
		(id, model_id, implementation, enabled, creator, created_at, updated_at, deleted, deleted_at)
		VALUES (?, 'model', ?, 1, 'alice', ?, ?, '0', NULL),
		       (?, 'model', ?, 0, NULL, NULL, ?, ?, ?)`,
		"current", schema.DialectStoreUnsupported, createdAt, updatedAt,
		"archived", schema.DialectStoreUnsupported, updatedAt, "archived", updatedAt).Error)

	module := &Policy{DB: db}
	require.NoError(t, module.AutoMigrate(t.Context()))
	require.NoError(t, module.AutoMigrate(t.Context()))
	require.True(t, db.Migrator().HasIndex(&schema.PolicyDialect{}, "idx_policy_dialect_model"))
	require.True(t, db.Migrator().HasIndex(&schema.PolicyDialect{}, "idx_policy_dialect_scope"))
	require.True(t, db.Migrator().HasIndex(&schema.PolicyDialect{}, "uniq_policy_dialect_model_impl_deleted"))

	var items []schema.PolicyDialect
	require.NoError(t, db.Unscoped().Order("id").Find(&items).Error)
	require.Len(t, items, 2)
	for _, item := range items {
		require.Equal(t, schema.DialectStoreUnsupported, item.Name)
		require.Equal(t, schema.DialectStoreUnsupported, item.Implementation)
		require.Equal(t, "global", item.ScopeType)
		require.Empty(t, item.ScopeCode)
		require.Equal(t, int64(1), item.Version)
		require.True(t, updatedAt.Equal(item.UpdatedAt))
		require.Nil(t, item.Description)
	}
	require.True(t, updatedAt.Equal(items[0].CreatedAt))
	require.Equal(t, "archived", items[0].Deleted)
	require.NotNil(t, items[0].DeletedAt)
	require.Nil(t, items[0].Creator)
	require.True(t, createdAt.Equal(items[1].CreatedAt))
	require.Equal(t, "0", items[1].Deleted)
	require.Equal(t, 1, items[1].Enabled)
	require.NotNil(t, items[1].Creator)
	require.Equal(t, "alice", *items[1].Creator)
	require.Error(t, db.Create(&schema.PolicyDialect{
		ID: "duplicate", ModelID: "model", Name: schema.DialectStoreUnsupported, Implementation: schema.DialectStoreUnsupported,
	}).Error)
}
