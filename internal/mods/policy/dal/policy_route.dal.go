package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Get policy route storage instance (only active records)
func GetPolicyRouteDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.PolicyRoute)).Where("deleted = '0'")
}

// Route policy management
type PolicyRoute struct {
	DB *gorm.DB
}

// Query policy routes from the database based on the provided parameters and options.
func (a *PolicyRoute) Query(ctx context.Context, params schema.PolicyRouteQueryParam, opts ...schema.PolicyRouteQueryOptions) (*schema.PolicyRouteQueryResult, error) {
	var opt schema.PolicyRouteQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetPolicyRouteDB(ctx, a.DB)
	if v := params.Name; v != "" {
		db = db.Where("name LIKE ?", "%"+v+"%")
	}

	var list schema.PolicyRoutes
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &schema.PolicyRouteQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get the specified policy route from the database.
func (a *PolicyRoute) Get(ctx context.Context, id string, opts ...schema.PolicyRouteQueryOptions) (*schema.PolicyRoute, error) {
	var opt schema.PolicyRouteQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.PolicyRoute)
	ok, err := util.FindOne(ctx, GetPolicyRouteDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified policy route exists in the database.
func (a *PolicyRoute) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyRouteDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// ExistsByUniqueKey checks whether a policy route with the given unique key already exists.
func (a *PolicyRoute) ExistsByUniqueKey(ctx context.Context, name string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyRouteDB(ctx, a.DB).
		Where("name = ?", name))
	return ok, errors.WithStack(err)
}

// Create a new policy route.
func (a *PolicyRoute) Create(ctx context.Context, item *schema.PolicyRoute) error {
	result := GetPolicyRouteDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified policy route in the database.
func (a *PolicyRoute) Update(ctx context.Context, item *schema.PolicyRoute) error {
	result := GetPolicyRouteDB(ctx, a.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified policy route from the database using logical deletion.
func (a *PolicyRoute) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetPolicyRouteDB(ctx, a.DB), id))
}
