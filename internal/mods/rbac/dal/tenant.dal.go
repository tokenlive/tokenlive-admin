package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Get tenant storage instance
func GetTenantDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.Tenant))
}

// Tenant management for RBAC
type Tenant struct {
	DB *gorm.DB
}

// Query tenants from the database based on the provided parameters and options.
func (a *Tenant) Query(ctx context.Context, params schema.TenantQueryParam, opts ...schema.TenantQueryOptions) (*schema.TenantQueryResult, error) {
	var opt schema.TenantQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetTenantDB(ctx, a.DB)
	if v := params.LikeCode; len(v) > 0 {
		db = db.Where("code LIKE ?", "%"+v+"%")
	}
	if v := params.LikeName; len(v) > 0 {
		db = db.Where("name LIKE ?", "%"+v+"%")
	}
	if v := params.Status; len(v) > 0 {
		db = db.Where("status = ?", v)
	}

	var list schema.Tenants
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &schema.TenantQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get the specified tenant from the database.
func (a *Tenant) Get(ctx context.Context, id string, opts ...schema.TenantQueryOptions) (*schema.Tenant, error) {
	var opt schema.TenantQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.Tenant)
	ok, err := util.FindOne(ctx, GetTenantDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

func (a *Tenant) GetByCode(ctx context.Context, code string, opts ...schema.TenantQueryOptions) (*schema.Tenant, error) {
	var opt schema.TenantQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.Tenant)
	ok, err := util.FindOne(ctx, GetTenantDB(ctx, a.DB).Where("code=?", code), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exist checks if the specified tenant exists in the database.
func (a *Tenant) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetTenantDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

func (a *Tenant) ExistsCode(ctx context.Context, code string) (bool, error) {
	ok, err := util.Exists(ctx, GetTenantDB(ctx, a.DB).Where("code=?", code))
	return ok, errors.WithStack(err)
}

// Create a new tenant.
func (a *Tenant) Create(ctx context.Context, item *schema.Tenant) error {
	result := GetTenantDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified tenant in the database.
func (a *Tenant) Update(ctx context.Context, item *schema.Tenant) error {
	result := GetTenantDB(ctx, a.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified tenant from the database.
func (a *Tenant) Delete(ctx context.Context, id string) error {
	result := GetTenantDB(ctx, a.DB).Where("id=?", id).Delete(new(schema.Tenant))
	return errors.WithStack(result.Error)
}
