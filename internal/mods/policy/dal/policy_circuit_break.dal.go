package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Get policy circuit break storage instance (only active records)
func GetPolicyCircuitBreakDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.PolicyCircuitBreak)).Where("deleted = '0'")
}

// Circuit break policy management
type PolicyCircuitBreak struct {
	DB *gorm.DB
}

// Query policy circuit breaks from the database based on the provided parameters and options.
func (a *PolicyCircuitBreak) Query(ctx context.Context, params schema.PolicyCircuitBreakQueryParam, opts ...schema.PolicyCircuitBreakQueryOptions) (*schema.PolicyCircuitBreakQueryResult, error) {
	var opt schema.PolicyCircuitBreakQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetPolicyCircuitBreakDB(ctx, a.DB)
	if v := params.Name; v != "" {
		db = db.Where("name LIKE ?", "%"+v+"%")
	}

	var list schema.PolicyCircuitBreaks
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &schema.PolicyCircuitBreakQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get the specified policy circuit break from the database.
func (a *PolicyCircuitBreak) Get(ctx context.Context, id string, opts ...schema.PolicyCircuitBreakQueryOptions) (*schema.PolicyCircuitBreak, error) {
	var opt schema.PolicyCircuitBreakQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.PolicyCircuitBreak)
	ok, err := util.FindOne(ctx, GetPolicyCircuitBreakDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified policy circuit break exists in the database.
func (a *PolicyCircuitBreak) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyCircuitBreakDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// ExistsByUniqueKey checks whether a policy circuit break with the given unique key already exists.
func (a *PolicyCircuitBreak) ExistsByUniqueKey(ctx context.Context, name string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyCircuitBreakDB(ctx, a.DB).Where("name = ?", name))
	return ok, errors.WithStack(err)
}

// Create a new policy circuit break.
func (a *PolicyCircuitBreak) Create(ctx context.Context, item *schema.PolicyCircuitBreak) error {
	result := GetPolicyCircuitBreakDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified policy circuit break in the database.
func (a *PolicyCircuitBreak) Update(ctx context.Context, item *schema.PolicyCircuitBreak) error {
	result := GetPolicyCircuitBreakDB(ctx, a.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified policy circuit break from the database using logical deletion.
func (a *PolicyCircuitBreak) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetPolicyCircuitBreakDB(ctx, a.DB), id))
}
