package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Get policy binding storage instance (only active records)
func GetPolicyBindingDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.PolicyBinding)).Where("deleted = '0'")
}

// PolicyBinding 策略绑定数据访问层
type PolicyBinding struct {
	DB *gorm.DB
}

// Query policy bindings from the database based on the provided parameters and options.
func (a *PolicyBinding) Query(ctx context.Context, params schema.PolicyBindingQueryParam, opts ...schema.PolicyBindingQueryOptions) (*schema.PolicyBindingQueryResult, error) {
	var opt schema.PolicyBindingQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetPolicyBindingDB(ctx, a.DB)
	if params.TenantCode != "" {
		db = db.Where("tenant_code = ?", params.TenantCode)
	}
	if params.UserID != "" {
		db = db.Where("user_id = ?", params.UserID)
	}
	if params.ModelCode != "" {
		db = db.Where("model_code = ?", params.ModelCode)
	}
	if params.PolicyType != "" {
		db = db.Where("policy_type = ?", params.PolicyType)
	}
	if params.PolicyID != "" {
		db = db.Where("policy_id = ?", params.PolicyID)
	}
	if params.Enabled != nil {
		db = db.Where("enabled = ?", *params.Enabled)
	}

	var list schema.PolicyBindings
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &schema.PolicyBindingQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get the specified policy binding from the database.
func (a *PolicyBinding) Get(ctx context.Context, id string, opts ...schema.PolicyBindingQueryOptions) (*schema.PolicyBinding, error) {
	var opt schema.PolicyBindingQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.PolicyBinding)
	ok, err := util.FindOne(ctx, GetPolicyBindingDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified policy binding exists in the database.
func (a *PolicyBinding) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyBindingDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// ExistsByDimensions checks if a binding of the same type already exists for the given dimensions (used for exclusive policy validation).
func (a *PolicyBinding) ExistsByDimensions(ctx context.Context, tenantCode, userID, modelCode, policyType string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyBindingDB(ctx, a.DB).
		Where("tenant_code = ? AND user_id = ? AND model_code = ? AND policy_type = ?", tenantCode, userID, modelCode, policyType))
	return ok, errors.WithStack(err)
}

// ExistsByUniqueKey checks whether a policy binding with the given unique dimensions and policy_id already exists.
func (a *PolicyBinding) ExistsByUniqueKey(ctx context.Context, tenantCode, userID, modelCode, policyType, policyID string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyBindingDB(ctx, a.DB).
		Where("tenant_code = ? AND user_id = ? AND model_code = ? AND policy_type = ? AND policy_id = ?", tenantCode, userID, modelCode, policyType, policyID))
	return ok, errors.WithStack(err)
}

// ExistsByPolicyID checks whether any binding exists for the given policy_id and policy_type.
func (a *PolicyBinding) ExistsByPolicyID(ctx context.Context, policyType, policyID string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyBindingDB(ctx, a.DB).
		Where("policy_type = ? AND policy_id = ?", policyType, policyID))
	return ok, errors.WithStack(err)
}

// CleanDeletedConflict checks if a logically deleted record exists with the same unique key
// (tenant_code, user_id, model_code, policy_type, policy_id) and deleted != '0'.
// If found, it physically deletes it to prevent 1062 duplicate entry DB errors when inserting or updating a record.
func (a *PolicyBinding) CleanDeletedConflict(ctx context.Context, tenantCode, userID, modelCode, policyType, policyID string) error {
	err := util.GetDB(ctx, a.DB).Unscoped().
		Where("tenant_code = ? AND user_id = ? AND model_code = ? AND policy_type = ? AND policy_id = ? AND deleted != '0'",
			tenantCode, userID, modelCode, policyType, policyID).
		Delete(new(schema.PolicyBinding)).Error
	return errors.WithStack(err)
}

// Create a new policy binding.
func (a *PolicyBinding) Create(ctx context.Context, item *schema.PolicyBinding) error {
	result := GetPolicyBindingDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified policy binding in the database.
func (a *PolicyBinding) Update(ctx context.Context, item *schema.PolicyBinding) error {
	result := GetPolicyBindingDB(ctx, a.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified policy binding from the database using logical deletion.
func (a *PolicyBinding) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetPolicyBindingDB(ctx, a.DB), id))
}
