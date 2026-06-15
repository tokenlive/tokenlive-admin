package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Get policy limit storage instance (only active records)
func GetPolicyLimitDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.PolicyLimit)).Where("deleted = '0'")
}

// Limit policy management
type PolicyLimit struct {
	DB *gorm.DB
}

// Query policy limits from the database based on the provided parameters and options.
func (a *PolicyLimit) Query(ctx context.Context, params schema.PolicyLimitQueryParam, opts ...schema.PolicyLimitQueryOptions) (*schema.PolicyLimitQueryResult, error) {
	var opt schema.PolicyLimitQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetPolicyLimitDB(ctx, a.DB)
	if v := params.Name; v != "" {
		db = db.Where("name LIKE ?", "%"+v+"%")
	}

	var list schema.PolicyLimits
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &schema.PolicyLimitQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get the specified policy limit from the database.
func (a *PolicyLimit) Get(ctx context.Context, id string, opts ...schema.PolicyLimitQueryOptions) (*schema.PolicyLimit, error) {
	var opt schema.PolicyLimitQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.PolicyLimit)
	ok, err := util.FindOne(ctx, GetPolicyLimitDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified policy limit exists in the database.
func (a *PolicyLimit) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyLimitDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// ExistsByUniqueKey checks whether a policy limit with the given unique key already exists.
func (a *PolicyLimit) ExistsByUniqueKey(ctx context.Context, name string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyLimitDB(ctx, a.DB).
		Where("name = ?", name))
	return ok, errors.WithStack(err)
}

// Create a new policy limit.
func (a *PolicyLimit) Create(ctx context.Context, item *schema.PolicyLimit) error {
	result := GetPolicyLimitDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified policy limit in the database.
func (a *PolicyLimit) Update(ctx context.Context, item *schema.PolicyLimit) error {
	result := GetPolicyLimitDB(ctx, a.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified policy limit from the database using logical deletion.
func (a *PolicyLimit) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetPolicyLimitDB(ctx, a.DB), id))
}
