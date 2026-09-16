package biz

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
	opsbiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	rbac "github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaykeys"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func metadataDB(t *testing.T) *gorm.DB {
	t.Helper()
	previous := config.C
	config.C = new(config.Config)
	config.C.Storage.DB.TablePrefix = "fixture_"
	config.C.General.Root.ID = "custom-root"
	config.C.General.Root.Name = "Root Name"
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "metadata.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&rbac.User{}, &rbac.UserAPIKey{}, &rbac.Tenant{}))
	t.Cleanup(func() {
		for _, model := range []any{&rbac.UserAPIKey{}, &rbac.User{}, &rbac.Tenant{}} {
			require.NoError(t, db.Unscoped().Where("id LIKE ?", "fixture-%").Delete(model).Error)
		}
		raw, err := db.DB()
		require.NoError(t, err)
		require.NoError(t, raw.Close())
		config.C = previous
	})
	return db
}

func TestMetadataIncludesDeletedDisabledAndMissingWithoutGuessing(t *testing.T) {
	db := metadataDB(t)
	require.NoError(t, db.Create(&rbac.User{ID: "fixture-user", Name: "Developer"}).Error)
	for _, key := range []rbac.UserAPIKey{
		{ID: "fixture-disabled", UserID: "fixture-user", Name: "Disabled App", APIKey: "fixture-disabled-secret", Status: 2, Deleted: "0"},
		{ID: "fixture-deleted", UserID: "fixture-user", Name: "Deleted App", APIKey: "fixture-deleted-secret", Status: 1, Deleted: "fixture-deleted"},
	} {
		require.NoError(t, db.Create(&key).Error)
	}
	require.NoError(t, db.Create(&rbac.Tenant{ID: "fixture-tenant", Code: "acme", Name: "Acme", APIKey: "fixture-tenant-secret", Status: rbac.TenantStatusActivated}).Error)
	resolver := NewIdentityResolver(db, nil, "pepper", time.Now)
	groups := []schema.Group{
		{Canonical: "disabled", Ref: schema.KeyRef{Hash: gatewaykeys.HashAPIKey("fixture-disabled-secret", "pepper")}},
		{Canonical: "deleted", Ref: schema.KeyRef{Hash: gatewaykeys.HashAPIKey("fixture-deleted-secret", "pepper")}},
		{Canonical: "missing", Ref: schema.KeyRef{Hash: "missing", UserID: "former-user"}},
		{Canonical: "tenant", Ref: schema.KeyRef{Hash: gatewaykeys.HashAPIKey("fixture-tenant-secret", "pepper")}},
		{Canonical: "rotated", Ref: schema.KeyRef{Hash: "old-tenant-hash", TenantID: "acme"}},
	}
	meta := resolver.Describe(context.Background(), groups)
	require.Equal(t, "disabled", meta["disabled"].KeyStatus)
	require.Equal(t, "Developer", meta["disabled"].Owner.Name)
	require.Equal(t, "deleted", meta["deleted"].KeyStatus)
	require.Equal(t, "unknown", meta["missing"].KeyStatus)
	require.Equal(t, "unavailable", meta["missing"].MetadataStatus)
	require.Equal(t, "tenant", meta["tenant"].Source)
	require.Equal(t, "Acme", meta["tenant"].Owner.Name)
	require.Equal(t, "acme", meta["rotated"].Owner.ID)
	require.Equal(t, "unknown", meta["rotated"].KeyStatus)
}

func TestIdentityAdminIDMustMatchUser(t *testing.T) {
	db := metadataDB(t)
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("check_usage_query", func(tx *gorm.DB) {
		if tx.Error != nil {
			t.Errorf("metadata query must use its own model: %v", tx.Error)
		}
	}))
	require.NoError(t, db.Create(&rbac.UserAPIKey{ID: "fixture-key", UserID: "fixture-user", Name: "App", APIKey: "fixture-secret", Status: 1}).Error)
	resolver := NewIdentityResolver(db, nil, "pepper", time.Now)
	good := schema.KeyRef{KeyID: "fixture-key", UserID: "fixture-user"}
	bad := schema.KeyRef{KeyID: "fixture-key", UserID: "another"}
	result := resolver.Resolve(context.Background(), []schema.Candidate{{Ref: good}, {Ref: bad}})
	require.Equal(t, "h:"+gatewaykeys.HashAPIKey("fixture-secret", "pepper"), result.Keys[good])
	require.Empty(t, result.Keys[bad])
}

func TestMetadataConflictingOwnersRemainExplicitlyUnknown(t *testing.T) {
	db := metadataDB(t)
	for _, key := range []rbac.UserAPIKey{
		{ID: "fixture-a", UserID: "owner-a", Name: "A", APIKey: "shared-fixture-key", Deleted: "fixture-a", Status: 1},
		{ID: "fixture-b", UserID: "owner-b", Name: "B", APIKey: "shared-fixture-key", Deleted: "0", Status: 1},
	} {
		require.NoError(t, db.Create(&key).Error)
	}
	resolver := NewIdentityResolver(db, nil, "pepper", time.Now)
	hash := gatewaykeys.HashAPIKey("shared-fixture-key", "pepper")
	result := resolver.Describe(context.Background(), []schema.Group{{Canonical: "a", Ref: schema.KeyRef{Hash: hash, UserID: "owner-a"}}})
	require.Empty(t, result["a"].Owner.ID)
	require.Equal(t, "partial", result["a"].MetadataStatus)
	require.Equal(t, "unknown", result["a"].KeyStatus)
}

func TestMetadataPortalExpiryCachingAndGlobalConcurrencyBound(t *testing.T) {
	now := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	expired := now.Add(-time.Second)
	var active, peak atomic.Int32
	portal := &fixturePortal{list: func(ctx context.Context, _ string) ([]opsbiz.PortalWorkspaceAPIKey, error) {
		n := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Millisecond):
		}
		return []opsbiz.PortalWorkspaceAPIKey{{ID: "id", Name: "App", Status: "enabled", ExpiresAt: &expired}}, nil
	}}
	resolver := NewIdentityResolver(nil, portal, "", func() time.Time { return now })
	groups := make([]schema.Group, 12)
	for i := range groups {
		ws := string(rune('a' + i))
		groups[i] = schema.Group{Canonical: ws, Ref: schema.KeyRef{WorkspaceID: ws, KeyID: "id", Hash: ws}}
	}
	meta := resolver.Describe(context.Background(), groups)
	require.Len(t, meta, 12)
	require.Equal(t, "expired", meta["a"].KeyStatus)
	require.LessOrEqual(t, peak.Load(), int32(4))
	resolver.Describe(context.Background(), groups)
	require.Equal(t, 12, portal.calls)
	now = now.Add(61 * time.Second)
	resolver.Describe(context.Background(), groups)
	require.Equal(t, 24, portal.calls)
}
