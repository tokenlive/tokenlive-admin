package biz

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsDal "github.com/tokenlive/tokenlive-admin/internal/mods/ops/dal"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	policyBiz "github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	policyDal "github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

const createSeedJSON = `{
  "policy_invocation": [
    {"id": "seed-invocation-1", "name": "first invocation"},
    {"id": "seed-invocation-2", "name": "second invocation"}
  ],
  "policy_circuit_break": [
    {"id": "seed-circuit-break-1", "name": "first circuit break"}
  ]
}`

func TestModelCreateAppliesRecommendedSeeds(t *testing.T) {
	db := newModelCreateSeedTestDB(t)
	biz := newModelCreateSeedTestBiz(db)
	seedCreatePolicyTemplates(t, db)
	workDir := writeCreateSeedFile(t, createSeedJSON)
	setCreateSeedConfig(t, workDir)

	result, err := biz.Create(newModelCreateSeedTestContext(), newCreateSeedModelForm(t, true, true))
	require.NoError(t, err)
	require.NotNil(t, result.Model)
	require.ElementsMatch(t, []string{
		policyBiz.PolicySeedTableInvocation,
		policyBiz.PolicySeedTableCircuitBreak,
	}, result.AppliedSeeds)
	require.Empty(t, result.SkippedSeeds)

	var invocations []policySchema.PolicyInvocation
	require.NoError(t, db.Where("model_id = ?", result.ID).Find(&invocations).Error)
	require.Len(t, invocations, 1)
	require.Equal(t, "first invocation", invocations[0].Name)
	require.Equal(t, "global", invocations[0].ScopeType)
	require.Equal(t, 1, invocations[0].Enabled)

	var circuitBreaks []policySchema.PolicyCircuitBreak
	require.NoError(t, db.Where("model_id = ?", result.ID).Find(&circuitBreaks).Error)
	require.Len(t, circuitBreaks, 1)
	require.Equal(t, "first circuit break", circuitBreaks[0].Name)
	require.Equal(t, "global", circuitBreaks[0].ScopeType)
}

func TestModelCreateSkipsUncheckedSeeds(t *testing.T) {
	db := newModelCreateSeedTestDB(t)
	biz := newModelCreateSeedTestBiz(db)
	seedCreatePolicyTemplates(t, db)
	workDir := writeCreateSeedFile(t, createSeedJSON)
	setCreateSeedConfig(t, workDir)

	result, err := biz.Create(newModelCreateSeedTestContext(), newCreateSeedModelForm(t, false, false))
	require.NoError(t, err)
	require.Empty(t, result.AppliedSeeds)
	require.Empty(t, result.SkippedSeeds)

	var invocations int64
	require.NoError(t, db.Model(&policySchema.PolicyInvocation{}).Where("model_id = ?", result.ID).Count(&invocations).Error)
	require.Zero(t, invocations)
}

func TestModelCreateDoesNotRollbackWhenSeedMissing(t *testing.T) {
	db := newModelCreateSeedTestDB(t)
	biz := newModelCreateSeedTestBiz(db)
	workDir := writeCreateSeedFile(t, createSeedJSON)
	setCreateSeedConfig(t, workDir)

	result, err := biz.Create(newModelCreateSeedTestContext(), newCreateSeedModelForm(t, true, true))
	require.NoError(t, err)
	require.NotEmpty(t, result.ID)
	require.Empty(t, result.AppliedSeeds)
	require.ElementsMatch(t, []string{
		policyBiz.PolicySeedTableInvocation,
		policyBiz.PolicySeedTableCircuitBreak,
	}, result.SkippedSeeds)

	var model schema.Model
	require.NoError(t, db.First(&model, "id = ?", result.ID).Error)
	require.Equal(t, "0", model.Deleted)
}

func newModelCreateSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&schema.Model{},
		&schema.DataPermission{},
		&policySchema.PolicyCircuitBreak{},
		&policySchema.PolicyInvocation{},
		&opsSchema.AuditLog{},
	))
	return db
}

func newModelCreateSeedTestContext() context.Context {
	ctx := context.Background()
	ctx = util.NewUsername(ctx, "alice")
	ctx = util.NewTenant(ctx, "tenant-a")
	return ctx
}

func newModelCreateSeedTestBiz(db *gorm.DB) *Model {
	trans := &util.Trans{DB: db}
	audit := &opsBiz.AuditLog{
		Trans:       trans,
		AuditLogDAL: &opsDal.AuditLog{DB: db},
	}
	biz := &Model{
		Trans:    trans,
		ModelDAL: &dal.Model{DB: db},
		DataPermissionBIZ: &DataPermission{
			Trans:             trans,
			DataPermissionDAL: &dal.DataPermission{DB: db},
		},
		ConfigRedisSync: &ConfigRedisSync{},
		AuditLogBIZ:     audit,
	}
	biz.PolicyInvocationBIZ = &policyBiz.PolicyInvocation{
		Trans:               trans,
		PolicyInvocationDAL: &policyDal.PolicyInvocation{DB: db},
		ModelDAL:            &dal.Model{DB: db},
		DataPermissionDAL:   &dal.DataPermission{DB: db},
		AuditLogBIZ:         audit,
	}
	biz.PolicyCircuitBreakBIZ = &policyBiz.PolicyCircuitBreak{
		Trans:                 trans,
		PolicyCircuitBreakDAL: &policyDal.PolicyCircuitBreak{DB: db},
		ModelDAL:              &dal.Model{DB: db},
		DataPermissionDAL:     &dal.DataPermission{DB: db},
		AuditLogBIZ:           audit,
	}
	return biz
}

func newCreateSeedModelForm(t *testing.T, applyInvocation, applyCircuitBreak bool) *schema.ModelForm {
	t.Helper()
	return &schema.ModelForm{
		ModelName:             "Seeded Model " + t.Name(),
		ModelCode:             "seed-code-" + t.Name(),
		SpaceCode:             "default",
		RequestTypes:          `["chat_completion"]`,
		ContextLength:         128000,
		MaxOutputTokens:       8192,
		Enabled:               1,
		ApplyInvocationSeed:   applyInvocation,
		ApplyCircuitBreakSeed: applyCircuitBreak,
	}
}

func seedCreatePolicyTemplates(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Create(&policySchema.PolicyInvocation{
		ID:        "seed-invocation-1",
		Name:      "first invocation",
		Type:      "failover",
		Enabled:   1,
		ScopeType: "global",
		Deleted:   "0",
		CreatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&policySchema.PolicyInvocation{
		ID:        "seed-invocation-2",
		Name:      "second invocation",
		Type:      "failover",
		Enabled:   1,
		ScopeType: "global",
		Deleted:   "0",
		CreatedAt: now,
	}).Error)
	require.NoError(t, db.Create(&policySchema.PolicyCircuitBreak{
		ID:                "seed-circuit-break-1",
		Name:              "first circuit break",
		Level:             "INSTANCE",
		SlidingWindowType: "time",
		Enabled:           1,
		ScopeType:         "global",
		Deleted:           "0",
		CreatedAt:         now,
	}).Error)
}

func writeCreateSeedFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "policy_seed.json"), []byte(content), 0o644))
	return dir
}

func setCreateSeedConfig(t *testing.T, workDir string) {
	t.Helper()
	prevWorkDir := config.C.General.WorkDir
	prevSeedFile := config.C.General.PolicySeedFile
	config.C.General.WorkDir = workDir
	config.C.General.PolicySeedFile = "policy_seed.json"
	t.Cleanup(func() {
		config.C.General.WorkDir = prevWorkDir
		config.C.General.PolicySeedFile = prevSeedFile
	})
}
