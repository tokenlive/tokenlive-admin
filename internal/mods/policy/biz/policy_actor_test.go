package biz

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsDal "github.com/tokenlive/tokenlive-admin/internal/mods/ops/dal"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	policyDal "github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	resourceDal "github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	resourceSchema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

func TestBlankActorCreateKeepsKindDifference(t *testing.T) {
	db := newPolicyActorTestDB(t)
	ctx := context.Background()

	tagging, err := newPolicyActorTaggingBiz(db).Create(ctx, &policySchema.PolicyTaggingForm{
		Name: "tag", Relation: "AND",
	})
	require.NoError(t, err)
	require.NotNil(t, tagging.Creator)
	require.Empty(t, *tagging.Creator)

	limit, err := newPolicyActorLimitBiz(db).Create(ctx, &policySchema.PolicyLimitForm{
		Name: "limit", Type: "request", RelationType: "AND",
	})
	require.NoError(t, err)
	require.Nil(t, limit.Creator)
}

func newPolicyActorTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&policySchema.PolicyTagging{},
		&policySchema.PolicyLimit{},
		&resourceSchema.Model{},
		&resourceSchema.DataPermission{},
		&opsSchema.AuditLog{},
	))
	return db
}

func newPolicyActorTaggingBiz(db *gorm.DB) *PolicyTagging {
	trans := &util.Trans{DB: db}
	return &PolicyTagging{
		Trans:             trans,
		PolicyTaggingDAL:  &policyDal.PolicyTagging{DB: db},
		PolicyRedisSync:   &PolicyRedisSync{},
		ModelDAL:          &resourceDal.Model{DB: db},
		DataPermissionDAL: &resourceDal.DataPermission{DB: db},
		AuditLogBIZ:       &opsBiz.AuditLog{Trans: trans, AuditLogDAL: &opsDal.AuditLog{DB: db}},
	}
}

func newPolicyActorLimitBiz(db *gorm.DB) *PolicyLimit {
	trans := &util.Trans{DB: db}
	return &PolicyLimit{
		Trans:             trans,
		PolicyLimitDAL:    &policyDal.PolicyLimit{DB: db},
		PolicyRedisSync:   &PolicyRedisSync{},
		ModelDAL:          &resourceDal.Model{DB: db},
		DataPermissionDAL: &resourceDal.DataPermission{DB: db},
		AuditLogBIZ:       &opsBiz.AuditLog{Trans: trans, AuditLogDAL: &opsDal.AuditLog{DB: db}},
	}
}
