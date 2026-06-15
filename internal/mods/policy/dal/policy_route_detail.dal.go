package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Get policy route detail storage instance (only active records)
func GetPolicyRouteDetailDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.PolicyRouteDetail)).Where("deleted = '0'")
}

// Route policy detail management
type PolicyRouteDetail struct {
	DB *gorm.DB
}

// Query policy route details from the database based on the provided parameters and options.
func (a *PolicyRouteDetail) Query(ctx context.Context, params schema.PolicyRouteDetailQueryParam, opts ...schema.PolicyRouteDetailQueryOptions) (*schema.PolicyRouteDetailQueryResult, error) {
	var opt schema.PolicyRouteDetailQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetPolicyRouteDetailDB(ctx, a.DB)
	if v := params.RouteId; v != "" {
		db = db.Where("route_id = ?", v)
	}

	var list schema.PolicyRouteDetails
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &schema.PolicyRouteDetailQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get the specified policy route detail from the database.
func (a *PolicyRouteDetail) Get(ctx context.Context, id string, opts ...schema.PolicyRouteDetailQueryOptions) (*schema.PolicyRouteDetail, error) {
	var opt schema.PolicyRouteDetailQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.PolicyRouteDetail)
	ok, err := util.FindOne(ctx, GetPolicyRouteDetailDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified policy route detail exists in the database.
func (a *PolicyRouteDetail) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyRouteDetailDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// Create a new policy route detail.
func (a *PolicyRouteDetail) Create(ctx context.Context, item *schema.PolicyRouteDetail) error {
	result := GetPolicyRouteDetailDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified policy route detail in the database.
func (a *PolicyRouteDetail) Update(ctx context.Context, item *schema.PolicyRouteDetail) error {
	result := GetPolicyRouteDetailDB(ctx, a.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified policy route detail from the database using logical deletion.
func (a *PolicyRouteDetail) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetPolicyRouteDetailDB(ctx, a.DB), id))
}
