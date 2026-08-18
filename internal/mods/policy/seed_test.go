package policy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"gorm.io/gorm"
)

const seedTestJSON = `{
  "policy_invocation": [
    {
      "id": "seed-invocation-1",
      "name": "seed invocation policy",
      "type": "failover",
      "retry_policy": {"retry": 3, "error_codes": ["500", "503"]},
      "version": 1787066763412,
      "enabled": 1,
      "description": "",
      "model_id": "",
      "scope_type": "global",
      "scope_code": "",
      "priority": 0
    }
  ],
  "policy_circuit_break": [
    {
      "id": "seed-circuit-break-1",
      "name": "seed circuit break policy",
      "level": "INSTANCE",
      "sliding_window_type": "time",
      "sliding_window_size": 60,
      "min_calls_threshold": 2,
      "failure_rate_threshold": 50.0,
      "wait_duration_in_open_state": 60000,
      "allowed_calls_in_half_open_state": 2,
      "outlier_max_percent": 100,
      "error_codes": ["500", "429"],
      "error_messages": ["Internal server error"],
      "version": 1787067091180,
      "enabled": 1,
      "description": "通用熔断策略",
      "model_id": "",
      "scope_type": "global",
      "scope_code": "",
      "priority": 0
    }
  ]
}`

func TestInitPolicySeedsCreatesMissingTemplates(t *testing.T) {
	db, seedFile := newSeedTestDB(t, seedTestJSON)
	ctx := context.Background()

	require.NoError(t, initPolicySeedsFromFile(ctx, db, seedFile))

	var invocation schema.PolicyInvocation
	require.NoError(t, db.First(&invocation, "id = ?", "seed-invocation-1").Error)
	require.Equal(t, "seed invocation policy", invocation.Name)
	require.Equal(t, "global", invocation.ScopeType)
	require.Equal(t, 1, invocation.Enabled)
	require.NotNil(t, invocation.RetryPolicy)
	require.Contains(t, *invocation.RetryPolicy, `"retry": 3`)
	require.Equal(t, int64(1787066763412), invocation.Version)
	require.NotNil(t, invocation.Creator)
	require.Equal(t, seedCreator, *invocation.Creator)
	require.False(t, invocation.CreatedAt.IsZero())

	var circuitBreak schema.PolicyCircuitBreak
	require.NoError(t, db.First(&circuitBreak, "id = ?", "seed-circuit-break-1").Error)
	require.Equal(t, "INSTANCE", circuitBreak.Level)
	require.NotNil(t, circuitBreak.ErrorCodes)
	require.Contains(t, *circuitBreak.ErrorCodes, "500")
	require.Equal(t, 100, circuitBreak.OutlierMaxPercent)
}

func TestInitPolicySeedsSkipsExistingByID(t *testing.T) {
	db, seedFile := newSeedTestDB(t, seedTestJSON)
	ctx := context.Background()
	require.NoError(t, initPolicySeedsFromFile(ctx, db, seedFile))

	// Operator renames the seeded template.
	require.NoError(t, db.Model(&schema.PolicyInvocation{}).
		Where("id = ?", "seed-invocation-1").
		Update("name", "renamed by operator").Error)

	require.NoError(t, initPolicySeedsFromFile(ctx, db, seedFile))

	var invocation schema.PolicyInvocation
	require.NoError(t, db.First(&invocation, "id = ?", "seed-invocation-1").Error)
	require.Equal(t, "renamed by operator", invocation.Name)
}

func TestInitPolicySeedsDoesNotReviveSoftDeleted(t *testing.T) {
	db, seedFile := newSeedTestDB(t, seedTestJSON)
	ctx := context.Background()
	require.NoError(t, initPolicySeedsFromFile(ctx, db, seedFile))

	// Operator soft-deletes the seeded template.
	require.NoError(t, db.Model(&schema.PolicyInvocation{}).
		Where("id = ?", "seed-invocation-1").
		Updates(map[string]any{"deleted": "seed-invocation-1", "deleted_at": time.Now()}).Error)

	require.NoError(t, initPolicySeedsFromFile(ctx, db, seedFile))

	var count int64
	require.NoError(t, db.Unscoped().Model(&schema.PolicyInvocation{}).
		Where("id = ?", "seed-invocation-1").Count(&count).Error)
	require.Equal(t, int64(1), count)

	var liveCount int64
	require.NoError(t, db.Model(&schema.PolicyInvocation{}).
		Where("id = ?", "seed-invocation-1").Count(&liveCount).Error)
	require.Equal(t, int64(0), liveCount)
}

func TestInitPolicySeedsMissingFileSkips(t *testing.T) {
	db, _ := newSeedTestDB(t, seedTestJSON)
	ctx := context.Background()

	err := initPolicySeedsFromFile(ctx, db, "/nonexistent/policy_seed.json")
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&schema.PolicyInvocation{}).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

func TestInitPolicySeedsUnsupportedTableFails(t *testing.T) {
	db, seedFile := newSeedTestDB(t, `{"policy_limit": [{"id": "x"}]}`)
	ctx := context.Background()

	err := initPolicySeedsFromFile(ctx, db, seedFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "policy_limit")
}

func TestInitPolicySeedsRealSeedFileIsValid(t *testing.T) {
	db, _ := newSeedTestDB(t, seedTestJSON)
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	seedFile := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "configs", "policy_seed.json")

	require.NoError(t, initPolicySeedsFromFile(context.Background(), db, seedFile))

	var invocations int64
	require.NoError(t, db.Model(&schema.PolicyInvocation{}).Count(&invocations).Error)
	require.Positive(t, invocations)
	var circuitBreaks int64
	require.NoError(t, db.Model(&schema.PolicyCircuitBreak{}).Count(&circuitBreaks).Error)
	require.Positive(t, circuitBreaks)
}

func newSeedTestDB(t *testing.T, seedContent string) (*gorm.DB, string) {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&schema.PolicyCircuitBreak{},
		&schema.PolicyInvocation{},
	))

	seedFile := filepath.Join(t.TempDir(), "policy_seed.json")
	require.NoError(t, os.WriteFile(seedFile, []byte(seedContent), 0o644))
	return db, seedFile
}
