package biz

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsDal "github.com/tokenlive/tokenlive-admin/internal/mods/ops/dal"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	resourceDal "github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	resourceSchema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

func TestPolicyToggleEnabledUpdatesOnlyEnabledAndSyncsDimension(t *testing.T) {
	db := newPolicyToggleEnabledTestDB(t)
	syncer := &recordingPolicySyncer{}
	ctx := newPolicyToggleEnabledTestContext()
	createModelWithPermission(t, db, "model-1", "model-code", "alice", "tenant-a", 0b111)
	seedToggleEnabledPolicies(t, db)

	tests := []struct {
		name         string
		policyType   string
		toggle       func(context.Context, string, *schema.PolicyEnabledForm) error
		assertStored func(*testing.T, *gorm.DB)
	}{
		{
			name:       "loadbalance",
			policyType: "loadbalance",
			toggle:     newPolicyToggleEnabledLoadbalanceBiz(db, syncer).ToggleEnabled,
			assertStored: func(t *testing.T, db *gorm.DB) {
				var item schema.PolicyLoadbalance
				require.NoError(t, db.First(&item, "id = ?", "loadbalance-1").Error)
				require.Equal(t, 1, item.Enabled)
				require.Equal(t, "loadbalance policy", item.Name)
			},
		},
		{
			name:       "route",
			policyType: "route",
			toggle:     newPolicyToggleEnabledRouteBiz(db, syncer).ToggleEnabled,
			assertStored: func(t *testing.T, db *gorm.DB) {
				var item schema.PolicyRoute
				require.NoError(t, db.First(&item, "id = ?", "route-1").Error)
				require.Equal(t, 1, item.Enabled)
				require.Equal(t, "route policy", item.Name)
			},
		},
		{
			name:       "limit",
			policyType: "limit",
			toggle:     newPolicyToggleEnabledLimitBiz(db, syncer).ToggleEnabled,
			assertStored: func(t *testing.T, db *gorm.DB) {
				var item schema.PolicyLimit
				require.NoError(t, db.First(&item, "id = ?", "limit-1").Error)
				require.Equal(t, 1, item.Enabled)
				require.Equal(t, "limit policy", item.Name)
			},
		},
		{
			name:       "circuit_break",
			policyType: "circuit_break",
			toggle:     newPolicyToggleEnabledCircuitBreakBiz(db, syncer).ToggleEnabled,
			assertStored: func(t *testing.T, db *gorm.DB) {
				var item schema.PolicyCircuitBreak
				require.NoError(t, db.First(&item, "id = ?", "circuit_break-1").Error)
				require.Equal(t, 1, item.Enabled)
				require.Equal(t, "circuit break policy", item.Name)
			},
		},
		{
			name:       "invocation",
			policyType: "invocation",
			toggle:     newPolicyToggleEnabledInvocationBiz(db, syncer).ToggleEnabled,
			assertStored: func(t *testing.T, db *gorm.DB) {
				var item schema.PolicyInvocation
				require.NoError(t, db.First(&item, "id = ?", "invocation-1").Error)
				require.Equal(t, 1, item.Enabled)
				require.Equal(t, "invocation policy", item.Name)
			},
		},
		{
			name:       "tagging",
			policyType: "tagging",
			toggle:     newPolicyToggleEnabledTaggingBiz(db, syncer).ToggleEnabled,
			assertStored: func(t *testing.T, db *gorm.DB) {
				var item schema.PolicyTagging
				require.NoError(t, db.First(&item, "id = ?", "tagging-1").Error)
				require.Equal(t, 1, item.Enabled)
				require.Equal(t, "tagging policy", item.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncer.calls = nil
			err := tt.toggle(ctx, tt.name+"-1", &schema.PolicyEnabledForm{Enabled: 1})

			require.NoError(t, err)
			tt.assertStored(t, db)
			require.Equal(t, []routeDetailSyncCall{{
				scopeType: "tenant",
				scopeCode: "tenant-a",
				modelID:   "model-1",
			}}, syncer.calls)
		})
	}
}

func newPolicyToggleEnabledTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&schema.PolicyLoadbalance{},
		&schema.PolicyRoute{},
		&schema.PolicyLimit{},
		&schema.PolicyCircuitBreak{},
		&schema.PolicyInvocation{},
		&schema.PolicyTagging{},
		&resourceSchema.Model{},
		&resourceSchema.DataPermission{},
		&opsSchema.AuditLog{},
	))
	return db
}

func newPolicyToggleEnabledTestContext() context.Context {
	ctx := context.Background()
	ctx = util.NewUsername(ctx, "alice")
	ctx = util.NewTenant(ctx, "tenant-a")
	return ctx
}

func seedToggleEnabledPolicies(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Create(&schema.PolicyLoadbalance{
		ID: "loadbalance-1", ModelID: "model-1", ScopeType: "tenant", ScopeCode: "tenant-a",
		Name: "loadbalance policy", Type: "round_robin", Enabled: 0, Deleted: "0", CreatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&schema.PolicyRoute{
		ID: "route-1", ModelID: "model-1", ScopeType: "tenant", ScopeCode: "tenant-a",
		Name: "route policy", Enabled: 0, Deleted: "0", CreatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&schema.PolicyLimit{
		ID: "limit-1", ModelID: "model-1", ScopeType: "tenant", ScopeCode: "tenant-a",
		Name: "limit policy", Type: "request", RelationType: "AND", Enabled: 0, Deleted: "0", CreatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&schema.PolicyCircuitBreak{
		ID: "circuit_break-1", ModelID: "model-1", ScopeType: "tenant", ScopeCode: "tenant-a",
		Name: "circuit break policy", Level: "INSTANCE", SlidingWindowType: "time", Enabled: 0, Deleted: "0", CreatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&schema.PolicyInvocation{
		ID: "invocation-1", ModelID: "model-1", ScopeType: "tenant", ScopeCode: "tenant-a",
		Name: "invocation policy", Type: "retry", Enabled: 0, Deleted: "0", CreatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&schema.PolicyTagging{
		ID: "tagging-1", ModelID: "model-1", ScopeType: "tenant", ScopeCode: "tenant-a",
		Name: "tagging policy", Relation: "AND", Enabled: 0, Deleted: "0", CreatedAt: now,
	}).Error)
}

func newPolicyToggleEnabledAuditLogBiz(db *gorm.DB) *opsBiz.AuditLog {
	trans := &util.Trans{DB: db}
	return &opsBiz.AuditLog{
		Trans:       trans,
		AuditLogDAL: &opsDal.AuditLog{DB: db},
	}
}

func newPolicyToggleEnabledLoadbalanceBiz(db *gorm.DB, syncer PolicyChangeSyncer) *PolicyLoadbalance {
	return &PolicyLoadbalance{
		Trans:                &util.Trans{DB: db},
		PolicyLoadbalanceDAL: &dal.PolicyLoadbalance{DB: db},
		PolicyRedisSync:      syncer,
		ModelDAL:             &resourceDal.Model{DB: db},
		DataPermissionDAL:    &resourceDal.DataPermission{DB: db},
		AuditLogBIZ:          newPolicyToggleEnabledAuditLogBiz(db),
	}
}

func newPolicyToggleEnabledRouteBiz(db *gorm.DB, syncer PolicyChangeSyncer) *PolicyRoute {
	return &PolicyRoute{
		Trans:           &util.Trans{DB: db},
		PolicyRouteDAL:  &dal.PolicyRoute{DB: db},
		PolicyRedisSync: syncer,
		ModelDAL:        &resourceDal.Model{DB: db},
		DataPermissionDAL: &resourceDal.DataPermission{
			DB: db,
		},
		AuditLogBIZ: newPolicyToggleEnabledAuditLogBiz(db),
	}
}

func newPolicyToggleEnabledLimitBiz(db *gorm.DB, syncer PolicyChangeSyncer) *PolicyLimit {
	return &PolicyLimit{
		Trans:           &util.Trans{DB: db},
		PolicyLimitDAL:  &dal.PolicyLimit{DB: db},
		PolicyRedisSync: syncer,
		ModelDAL:        &resourceDal.Model{DB: db},
		DataPermissionDAL: &resourceDal.DataPermission{
			DB: db,
		},
		AuditLogBIZ: newPolicyToggleEnabledAuditLogBiz(db),
	}
}

func newPolicyToggleEnabledCircuitBreakBiz(db *gorm.DB, syncer PolicyChangeSyncer) *PolicyCircuitBreak {
	return &PolicyCircuitBreak{
		Trans:                 &util.Trans{DB: db},
		PolicyCircuitBreakDAL: &dal.PolicyCircuitBreak{DB: db},
		PolicyRedisSync:       syncer,
		ModelDAL:              &resourceDal.Model{DB: db},
		DataPermissionDAL:     &resourceDal.DataPermission{DB: db},
		AuditLogBIZ:           newPolicyToggleEnabledAuditLogBiz(db),
	}
}

func newPolicyToggleEnabledInvocationBiz(db *gorm.DB, syncer PolicyChangeSyncer) *PolicyInvocation {
	return &PolicyInvocation{
		Trans:               &util.Trans{DB: db},
		PolicyInvocationDAL: &dal.PolicyInvocation{DB: db},
		PolicyRedisSync:     syncer,
		ModelDAL:            &resourceDal.Model{DB: db},
		DataPermissionDAL:   &resourceDal.DataPermission{DB: db},
		AuditLogBIZ:         newPolicyToggleEnabledAuditLogBiz(db),
	}
}

func newPolicyToggleEnabledTaggingBiz(db *gorm.DB, syncer PolicyChangeSyncer) *PolicyTagging {
	return &PolicyTagging{
		Trans:            &util.Trans{DB: db},
		PolicyTaggingDAL: &dal.PolicyTagging{DB: db},
		PolicyRedisSync:  syncer,
		ModelDAL:         &resourceDal.Model{DB: db},
		DataPermissionDAL: &resourceDal.DataPermission{
			DB: db,
		},
		AuditLogBIZ: newPolicyToggleEnabledAuditLogBiz(db),
	}
}
