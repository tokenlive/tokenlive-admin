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
	policyDal "github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	resourceDal "github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	resourceSchema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

func TestPolicyLimitCreateOwnsModel(t *testing.T) {
	db := newModelOwnedPolicyTestDB(t)
	biz := newModelOwnedPolicyLimitBiz(db)
	ctx := newModelOwnedPolicyTestContext("alice", "tenant-a")
	createModelWithPermission(t, db, "model-1", "gpt-4o", "alice", "tenant-a", 0b111)

	policy, err := biz.Create(ctx, &policySchema.PolicyLimitForm{
		ModelID:      "model-1",
		Name:         "default limit",
		Type:         "request",
		RelationType: "AND",
		Enabled:      1,
	})

	require.NoError(t, err)
	require.Equal(t, "model-1", policy.ModelID)
}

func TestPolicyLimitCreateTemplate(t *testing.T) {
	db := newModelOwnedPolicyTestDB(t)
	biz := newModelOwnedPolicyLimitBiz(db)
	ctx := newModelOwnedPolicyTestContext("alice", "tenant-a")

	policy, err := biz.Create(ctx, &policySchema.PolicyLimitForm{
		Name:         "template limit",
		Type:         "request",
		RelationType: "AND",
		Enabled:      1,
	})

	require.NoError(t, err)
	require.Empty(t, policy.ModelID)
}

func TestPolicyLimitCopyTemplateToModelCreatesInstance(t *testing.T) {
	db := newModelOwnedPolicyTestDB(t)
	biz := newModelOwnedPolicyLimitBiz(db)
	ctx := newModelOwnedPolicyTestContext("alice", "tenant-a")
	createModelWithPermission(t, db, "model-1", "gpt-4o", "alice", "tenant-a", 0b111)

	template, err := biz.Create(ctx, &policySchema.PolicyLimitForm{
		Name:         "default limit",
		Type:         "request",
		RelationType: "AND",
		Enabled:      1,
	})
	require.NoError(t, err)

	instance, err := biz.CopyTemplateToModel(ctx, template.ID, &policySchema.PolicyCopyToModelForm{ModelID: "model-1"})

	require.NoError(t, err)
	require.NotEqual(t, template.ID, instance.ID)
	require.Equal(t, "model-1", instance.ModelID)
	require.Equal(t, template.Name, instance.Name)
	require.Equal(t, template.Type, instance.Type)
}

func TestPolicyLimitCopyTemplateToModelRenamesConflictingInstance(t *testing.T) {
	db := newModelOwnedPolicyTestDB(t)
	biz := newModelOwnedPolicyLimitBiz(db)
	ctx := newModelOwnedPolicyTestContext("alice", "tenant-a")
	createModelWithPermission(t, db, "model-1", "gpt-4o", "alice", "tenant-a", 0b111)

	template, err := biz.Create(ctx, &policySchema.PolicyLimitForm{
		Name:         "default limit",
		Type:         "request",
		RelationType: "AND",
		Enabled:      1,
	})
	require.NoError(t, err)
	_, err = biz.CopyTemplateToModel(ctx, template.ID, &policySchema.PolicyCopyToModelForm{ModelID: "model-1"})
	require.NoError(t, err)

	instance, err := biz.CopyTemplateToModel(ctx, template.ID, &policySchema.PolicyCopyToModelForm{ModelID: "model-1"})

	require.NoError(t, err)
	require.Equal(t, "default limit-2", instance.Name)
}

func TestPolicyLimitCreateRejectsUserWithoutModelWritePermission(t *testing.T) {
	db := newModelOwnedPolicyTestDB(t)
	biz := newModelOwnedPolicyLimitBiz(db)
	ctx := newModelOwnedPolicyTestContext("bob", "tenant-a")
	createModelWithPermission(t, db, "model-1", "gpt-4o", "bob", "tenant-a", 0b001)

	_, err := biz.Create(ctx, &policySchema.PolicyLimitForm{
		ModelID:      "model-1",
		Name:         "default limit",
		Type:         "request",
		RelationType: "AND",
		Enabled:      1,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "Model not found")
}

func TestPolicyLimitQueryFiltersByReadableModels(t *testing.T) {
	db := newModelOwnedPolicyTestDB(t)
	biz := newModelOwnedPolicyLimitBiz(db)
	ctx := newModelOwnedPolicyTestContext("alice", "tenant-a")
	createModelWithPermission(t, db, "model-1", "gpt-4o", "alice", "tenant-a", 0b001)
	createModelWithPermission(t, db, "model-2", "claude", "bob", "tenant-b", 0b111)
	require.NoError(t, db.Create(&policySchema.PolicyLimit{
		ID:           "policy-1",
		ModelID:      "model-1",
		Name:         "visible",
		Type:         "request",
		RelationType: "AND",
		Deleted:      "0",
		CreatedAt:    time.Now(),
	}).Error)
	require.NoError(t, db.Create(&policySchema.PolicyLimit{
		ID:           "policy-2",
		ModelID:      "model-2",
		Name:         "hidden",
		Type:         "request",
		RelationType: "AND",
		Deleted:      "0",
		CreatedAt:    time.Now(),
	}).Error)

	result, err := biz.Query(ctx, policySchema.PolicyLimitQueryParam{ModelID: "model-1"})

	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	require.Equal(t, "visible", result.Data[0].Name)
}

func TestPolicyLimitQueryDefaultsToTemplates(t *testing.T) {
	db := newModelOwnedPolicyTestDB(t)
	biz := newModelOwnedPolicyLimitBiz(db)
	ctx := newModelOwnedPolicyTestContext("alice", "tenant-a")
	createModelWithPermission(t, db, "model-1", "gpt-4o", "alice", "tenant-a", 0b111)
	require.NoError(t, db.Create(&policySchema.PolicyLimit{
		ID:           "template-1",
		Name:         "template",
		Type:         "request",
		RelationType: "AND",
		Deleted:      "0",
		CreatedAt:    time.Now(),
	}).Error)
	require.NoError(t, db.Create(&policySchema.PolicyLimit{
		ID:           "policy-1",
		ModelID:      "model-1",
		Name:         "model policy",
		Type:         "request",
		RelationType: "AND",
		Deleted:      "0",
		CreatedAt:    time.Now(),
	}).Error)

	result, err := biz.Query(ctx, policySchema.PolicyLimitQueryParam{})

	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	require.Equal(t, "template", result.Data[0].Name)
}

func newModelOwnedPolicyTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&policySchema.PolicyLimit{},
		&resourceSchema.Model{},
		&resourceSchema.DataPermission{},
		&opsSchema.AuditLog{},
	))
	return db
}

func newModelOwnedPolicyLimitBiz(db *gorm.DB) *PolicyLimit {
	trans := &util.Trans{DB: db}
	return &PolicyLimit{
		Trans:             trans,
		PolicyLimitDAL:    &policyDal.PolicyLimit{DB: db},
		PolicyRedisSync:   &PolicyRedisSync{},
		ModelDAL:          &resourceDal.Model{DB: db},
		DataPermissionDAL: &resourceDal.DataPermission{DB: db},
		AuditLogBIZ: &opsBiz.AuditLog{
			Trans:       trans,
			AuditLogDAL: &opsDal.AuditLog{DB: db},
		},
	}
}

func newModelOwnedPolicyTestContext(username, tenant string) context.Context {
	ctx := context.Background()
	ctx = util.NewUsername(ctx, username)
	ctx = util.NewTenant(ctx, tenant)
	return ctx
}

func createModelWithPermission(t *testing.T, db *gorm.DB, modelID, modelCode, username, tenant string, permission uint) {
	t.Helper()
	require.NoError(t, db.Create(&resourceSchema.Model{
		ID:           modelID,
		ModelName:    modelCode,
		ModelCode:    modelCode,
		SpaceCode:    "default",
		RequestTypes: "[]",
		Abilities:    "[]",
		Deleted:      "0",
		CreatedAt:    time.Now(),
	}).Error)
	require.NoError(t, db.Create(&resourceSchema.DataPermission{
		ID:         util.NewXID(),
		Type:       resourceSchema.DataPermissionTypeModel,
		DataId:     modelID,
		User:       username,
		Tenant:     tenant,
		Role:       "owner",
		Permission: permission,
		Creator:    username,
		CreatedAt:  time.Now(),
	}).Error)
}
