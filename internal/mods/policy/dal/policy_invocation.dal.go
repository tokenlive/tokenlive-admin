package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Get policy invocation storage instance (only active records)
func GetPolicyInvocationDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.PolicyInvocation)).Where("deleted = '0'")
}

// Invocation policy management
type PolicyInvocation struct {
	DB *gorm.DB
}

// Query policy invocations from the database based on the provided parameters and options.
func (a *PolicyInvocation) Query(ctx context.Context, params schema.PolicyInvocationQueryParam, opts ...schema.PolicyInvocationQueryOptions) (*schema.PolicyInvocationQueryResult, error) {
	var opt schema.PolicyInvocationQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetPolicyInvocationDB(ctx, a.DB)
	if v := params.ModelID; v != "" {
		db = db.Where("model_id = ?", v)
	} else {
		db = db.Where("model_id = '' OR model_id IS NULL")
	}
	if v := params.Name; v != "" {
		db = db.Where("name LIKE ?", "%"+v+"%")
	}
	if params.ModelID != "" && !util.FromIsRootUser(ctx) {
		permTable := config.C.FormatTableName("data_permission")
		db = db.Where("model_id IN (SELECT data_id FROM "+permTable+" WHERE type = ? AND user = ? AND tenant = ? AND permission & 1 = 1)",
			"model", util.FromUsername(ctx), util.FromTenant(ctx))
	}

	var list schema.PolicyInvocations
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &schema.PolicyInvocationQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get the specified policy invocation from the database.
func (a *PolicyInvocation) Get(ctx context.Context, id string, opts ...schema.PolicyInvocationQueryOptions) (*schema.PolicyInvocation, error) {
	var opt schema.PolicyInvocationQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.PolicyInvocation)
	ok, err := util.FindOne(ctx, GetPolicyInvocationDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified policy invocation exists in the database.
func (a *PolicyInvocation) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyInvocationDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// Create a new policy invocation.
func (a *PolicyInvocation) Create(ctx context.Context, item *schema.PolicyInvocation) error {
	result := GetPolicyInvocationDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified policy invocation in the database.
func (a *PolicyInvocation) Update(ctx context.Context, item *schema.PolicyInvocation) error {
	result := GetPolicyInvocationDB(ctx, a.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// UpdateEnabled updates only the enabled status (and modifier) of the specified policy.
func (a *PolicyInvocation) UpdateEnabled(ctx context.Context, id string, enabled int, modifier string) error {
	result := GetPolicyInvocationDB(ctx, a.DB).Where("id=?", id).Updates(map[string]interface{}{
		"enabled":  enabled,
		"modifier": modifier,
	})
	return errors.WithStack(result.Error)
}

// ExistsByUniqueKey checks whether a policy invocation with the given unique key already exists.
func (a *PolicyInvocation) ExistsByUniqueKey(ctx context.Context, scopeType, scopeCode, modelID, name string) (bool, error) {
	db := GetPolicyInvocationDB(ctx, a.DB).Where("name = ?", name)
	if modelID == "" {
		db = db.Where("model_id = '' OR model_id IS NULL")
	} else {
		db = db.Where("model_id = ?", modelID)
	}
	ok, err := util.Exists(ctx, db)
	return ok, errors.WithStack(err)
}

// Delete the specified policy invocation from the database using logical deletion.
func (a *PolicyInvocation) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetPolicyInvocationDB(ctx, a.DB), id))
}
