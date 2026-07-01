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
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPolicyDeleteRejectsBoundPolicy(t *testing.T) {
	db := newPolicyDeleteTestDB(t)
	biz := newPolicyLoadbalanceDeleteTestBiz(db)
	require.NoError(t, db.Create(&schema.PolicyLoadbalance{
		ID:        "policy-1",
		Name:      "Loadbalance One",
		Type:      "ROUND_ROBIN",
		Deleted:   "0",
		CreatedAt: time.Now(),
	}).Error)
	require.NoError(t, db.Create(&schema.PolicyBinding{
		ID:         "binding-1",
		ModelCode:  "model-code",
		PolicyType: "loadbalance",
		PolicyID:   "policy-1",
		Deleted:    "0",
		CreatedAt:  time.Now(),
	}).Error)

	err := biz.Delete(newPolicyDeleteTestContext(), "policy-1")

	require.Error(t, err)
	require.Contains(t, err.Error(), "对应模型下解绑")
	require.Contains(t, err.Error(), "删除")

	var policy schema.PolicyLoadbalance
	require.NoError(t, db.First(&policy, "id = ?", "policy-1").Error)
	require.Equal(t, "0", policy.Deleted)
}

func newPolicyDeleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&schema.PolicyLoadbalance{},
		&schema.PolicyBinding{},
		&opsSchema.AuditLog{},
	))
	return db
}

func newPolicyLoadbalanceDeleteTestBiz(db *gorm.DB) *PolicyLoadbalance {
	trans := &util.Trans{DB: db}
	return &PolicyLoadbalance{
		Trans:                trans,
		PolicyLoadbalanceDAL: &dal.PolicyLoadbalance{DB: db},
		PolicyBindingDAL:     &dal.PolicyBinding{DB: db},
		PolicyRedisSync:      &PolicyRedisSync{},
		AuditLogBIZ: &opsBiz.AuditLog{
			Trans:       trans,
			AuditLogDAL: &opsDal.AuditLog{DB: db},
		},
	}
}

func newPolicyDeleteTestContext() context.Context {
	ctx := context.Background()
	ctx = util.NewUsername(ctx, "alice")
	ctx = util.NewTenant(ctx, "tenant-a")
	return ctx
}
