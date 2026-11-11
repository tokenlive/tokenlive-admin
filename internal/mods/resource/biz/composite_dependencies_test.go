package biz

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	policyBiz "github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	policyDal "github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/compositerouting"
	"gorm.io/gorm"
)

func TestCompositeBackupExportAndDeletionProtection(t *testing.T) {
	db, b, ctx := smartFixture(t)
	old := config.C.Gateway.CompositeRoutingV2
	config.C.Gateway.CompositeRoutingV2 = true
	t.Cleanup(func() { config.C.Gateway.CompositeRoutingV2 = old })
	f := smartForm()
	f.SmartRouting = &schema.SmartRouting{SchemaVersion: 2, Mode: compositerouting.Priority, PrimaryModelID: "a"}
	created, err := b.Create(ctx, f)
	require.NoError(t, err)
	inv := &policyBiz.PolicyInvocation{AuditLogBIZ: b.AuditLogBIZ, Trans: b.Trans, PolicyInvocationDAL: &policyDal.PolicyInvocation{DB: db}, ModelDAL: b.ModelDAL, DataPermissionDAL: b.DataPermissionBIZ.DataPermissionDAL}
	p, err := inv.Create(ctx, &policySchema.PolicyInvocationForm{ModelID: created.ID, Name: "tenant backup", Type: "failover", ScopeType: "tenant", ScopeCode: "tenant-a", Enabled: 1, FallbackPolicy: &policySchema.FallbackPolicy{Targets: []string{"b"}}})
	require.NoError(t, err)
	seedSmartEndpoints(t, db)
	sync := &GatewaySync{DB: db}
	got, err := sync.GetGatewayConfig(ctx, "smart")
	require.NoError(t, err)
	require.Contains(t, got.Models, "b", "a per-model pull must contain fallback endpoints from every active scope")
	require.Len(t, got.Models["b"].Endpoints, 1)
	refs, err := smartReferences(ctx, db, "b")
	require.NoError(t, err)
	require.Len(t, refs, 1)
	err = b.Delete(ctx, "b")
	require.ErrorContains(t, err, "引用")
	// Disabling preserves deletion protection, but removes runtime export dependency.
	require.NoError(t, db.Model(&policySchema.PolicyInvocation{}).Where("id = ?", p.ID).Update("enabled", 0).Error)
	ClearGatewayConfigCache()
	got, err = sync.GetGatewayConfig(ctx, "smart")
	require.NoError(t, err)
	require.NotContains(t, got.Models, "b")
	refs, err = smartReferences(ctx, db, "b")
	require.NoError(t, err)
	require.Len(t, refs, 1)
	require.NoError(t, inv.Delete(ctx, p.ID))
	refs, err = smartReferences(ctx, db, "b")
	require.NoError(t, err)
	require.Empty(t, refs)
	require.NoError(t, b.Delete(ctx, created.ID))
}

// The same SQL transaction must protect target validation and the enabled write.
// This catches splitting validation and lifecycle.toggle into two transactions.
func TestCompositePolicyEnableValidatesAndWritesInSameTransaction(t *testing.T) {
	db, b, ctx := smartFixture(t)
	f := smartForm()
	f.Enabled = 0
	f.SmartRouting = &schema.SmartRouting{SchemaVersion: 2, Mode: compositerouting.Priority, PrimaryModelID: "a"}
	created, err := b.Create(ctx, f)
	require.NoError(t, err)
	inv := &policyBiz.PolicyInvocation{AuditLogBIZ: b.AuditLogBIZ, Trans: b.Trans, PolicyInvocationDAL: &policyDal.PolicyInvocation{DB: db}, ModelDAL: b.ModelDAL, DataPermissionDAL: b.DataPermissionBIZ.DataPermissionDAL}
	p, err := inv.Create(ctx, &policySchema.PolicyInvocationForm{ModelID: created.ID, Name: "enable atomically", Type: "failover", ScopeType: "global", Enabled: 0, FallbackPolicy: &policySchema.FallbackPolicy{Targets: []string{"b"}}})
	require.NoError(t, err)
	var validatedConnection gorm.ConnPool
	writes := 0
	observing := true
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("test_composite_enable_target_transaction", func(tx *gorm.DB) {
		if tx.Statement.Table == "model" && strings.Contains(tx.Statement.SQL.String(), "model_code =") {
			validatedConnection = tx.Statement.ConnPool
		}
	}))
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("test_composite_enable_write_transaction", func(tx *gorm.DB) {
		if observing && tx.Statement.Table == "policy_invocation" {
			writes++
			require.NotNil(t, validatedConnection)
			require.Same(t, validatedConnection, tx.Statement.ConnPool, "validation locks must remain held through enable")
		}
	}))
	t.Cleanup(func() {
		_ = db.Callback().Query().Remove("test_composite_enable_target_transaction")
		_ = db.Callback().Update().Remove("test_composite_enable_write_transaction")
	})
	require.NoError(t, inv.ToggleEnabled(ctx, p.ID, &policySchema.PolicyEnabledForm{Enabled: 1}))
	require.Equal(t, 1, writes)
	stored, err := inv.PolicyInvocationDAL.Get(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, 1, stored.Enabled)
	observing = false
	require.NoError(t, inv.Delete(ctx, p.ID))
	require.NoError(t, b.Delete(ctx, created.ID))
}

func TestCompositeBackupPolicyRejectsInvalidTargets(t *testing.T) {
	db, b, ctx := smartFixture(t)
	f := smartForm()
	f.Enabled = 0
	f.SmartRouting = &schema.SmartRouting{SchemaVersion: 2, Mode: compositerouting.Priority, PrimaryModelID: "a"}
	created, err := b.Create(ctx, f)
	require.NoError(t, err)
	inv := &policyBiz.PolicyInvocation{AuditLogBIZ: b.AuditLogBIZ, Trans: b.Trans, PolicyInvocationDAL: &policyDal.PolicyInvocation{DB: db}, ModelDAL: b.ModelDAL, DataPermissionDAL: b.DataPermissionBIZ.DataPermissionDAL}
	for _, targets := range [][]string{{"missing"}, {"smart"}, {"b", "b"}, {""}} {
		_, err = inv.Create(ctx, &policySchema.PolicyInvocationForm{ModelID: created.ID, Name: "invalid", Type: "failover", ScopeType: "global", Enabled: 1, FallbackPolicy: &policySchema.FallbackPolicy{Targets: targets}})
		require.Error(t, err)
	}
	require.NoError(t, db.Model(&schema.Model{}).Where("id = ?", "b").Update("space_code", "other").Error)
	_, err = inv.Create(ctx, &policySchema.PolicyInvocationForm{ModelID: created.ID, Name: "cross space", Type: "failover", ScopeType: "global", Enabled: 1, FallbackPolicy: &policySchema.FallbackPolicy{Targets: []string{"b"}}})
	require.Error(t, err)
	var count int64
	require.NoError(t, db.Model(&policySchema.PolicyInvocation{}).Where("model_id = ?", created.ID).Count(&count).Error)
	require.Zero(t, count, "rejected policies must not leave stored records")
	require.NoError(t, db.Model(&schema.Model{}).Where("id = ?", "b").Update("space_code", "default").Error)
	policy, err := inv.Create(ctx, &policySchema.PolicyInvocationForm{ModelID: created.ID, Name: "disabled backup", Type: "failover", ScopeType: "global", Enabled: 0, FallbackPolicy: &policySchema.FallbackPolicy{Targets: []string{"b"}}})
	require.NoError(t, err)
	require.NoError(t, db.Model(&schema.Model{}).Where("id = ?", "b").Update("enabled", 0).Error)
	require.Error(t, inv.ToggleEnabled(ctx, policy.ID, &policySchema.PolicyEnabledForm{Enabled: 1}))
	stored, err := inv.PolicyInvocationDAL.Get(ctx, policy.ID)
	require.NoError(t, err)
	require.Zero(t, stored.Enabled, "failed re-enabling must not change stored state")
	require.NoError(t, inv.Delete(ctx, policy.ID))

	require.NoError(t, b.Delete(ctx, created.ID))
}
