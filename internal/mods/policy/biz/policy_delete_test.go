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

func TestPolicyDelete(t *testing.T) {
	db := newPolicyDeleteTestDB(t)
	biz := newPolicyLoadbalanceDeleteTestBiz(db)
	createModelWithPermission(t, db, "model-1", "model-code", "alice", "tenant-a", 0b111)
	require.NoError(t, db.Create(&schema.PolicyLoadbalance{
		ID:        "policy-1",
		ModelID:   "model-1",
		Name:      "Loadbalance One",
		Type:      "ROUND_ROBIN",
		Deleted:   "0",
		CreatedAt: time.Now(),
	}).Error)

	err := biz.Delete(newPolicyDeleteTestContext(), "policy-1")

	require.NoError(t, err)

	var policy schema.PolicyLoadbalance
	require.NoError(t, db.Unscoped().First(&policy, "id = ?", "policy-1").Error)
	require.NotEqual(t, "0", policy.Deleted)
}

func newPolicyDeleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&schema.PolicyLoadbalance{},
		&resourceSchema.Model{},
		&resourceSchema.DataPermission{},
		&opsSchema.AuditLog{},
	))
	return db
}

func newPolicyLoadbalanceDeleteTestBiz(db *gorm.DB) *PolicyLoadbalance {
	trans := &util.Trans{DB: db}
	return &PolicyLoadbalance{
		Trans:                trans,
		PolicyLoadbalanceDAL: &dal.PolicyLoadbalance{DB: db},
		PolicyRedisSync:      &PolicyRedisSync{},
		ModelDAL:             &resourceDal.Model{DB: db},
		DataPermissionDAL:    &resourceDal.DataPermission{DB: db},
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
