package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/config"
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
func (a *PolicyRoute) ExistsByUniqueKey(ctx context.Context, scopeType, scopeCode, modelID, name string) (bool, error) {
	db := GetPolicyRouteDB(ctx, a.DB).Where("name = ?", name)
	if modelID == "" {
		db = db.Where("model_id = '' OR model_id IS NULL")
	} else {
		db = db.Where("model_id = ?", modelID)
	}
	ok, err := util.Exists(ctx, db)
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

// UpdateEnabled updates only the enabled status (and modifier) of the specified policy.
func (a *PolicyRoute) UpdateEnabled(ctx context.Context, id string, enabled int, modifier string) error {
	result := GetPolicyRouteDB(ctx, a.DB).Where("id=?", id).Updates(map[string]interface{}{
		"enabled":  enabled,
		"modifier": modifier,
	})
	return errors.WithStack(result.Error)
}

// Delete the specified policy route from the database using logical deletion.
func (a *PolicyRoute) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetPolicyRouteDB(ctx, a.DB), id))
}
