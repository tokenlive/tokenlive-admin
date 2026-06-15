package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetProviderDB returns the database instance for Provider (only active records).
func GetProviderDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.Provider)).Where("deleted = '0'")
}

// Provider data access layer
type Provider struct {
	DB *gorm.DB
}

// Query providers from the database.
func (p *Provider) Query(ctx context.Context, params schema.ProviderQueryParam, opts ...schema.ProviderQueryOptions) (*schema.ProviderQueryResult, error) {
	var opt schema.ProviderQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetProviderDB(ctx, p.DB)
	if v := params.LikeCode; len(v) > 0 {
		db = db.Where("code LIKE ?", "%"+v+"%")
	}
	if v := params.LikeName; len(v) > 0 {
		db = db.Where("name LIKE ?", "%"+v+"%")
	}

	var list schema.Providers
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &schema.ProviderQueryResult{
		PageResult: pageResult,
		Data:       list,
	}, nil
}

// Get the specified provider from the database.
func (p *Provider) Get(ctx context.Context, id string, opts ...schema.ProviderQueryOptions) (*schema.Provider, error) {
	var opt schema.ProviderQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.Provider)
	ok, err := util.FindOne(ctx, GetProviderDB(ctx, p.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified provider exists.
func (p *Provider) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetProviderDB(ctx, p.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// ExistsName checks if a provider with the given name exists.
func (p *Provider) ExistsName(ctx context.Context, name string) (bool, error) {
	ok, err := util.Exists(ctx, GetProviderDB(ctx, p.DB).Where("name=?", name))
	return ok, errors.WithStack(err)
}

// ExistsCode checks if a provider with the given code exists.
func (p *Provider) ExistsCode(ctx context.Context, code string) (bool, error) {
	ok, err := util.Exists(ctx, GetProviderDB(ctx, p.DB).Where("code=?", code))
	return ok, errors.WithStack(err)
}

// Create a new provider.
func (p *Provider) Create(ctx context.Context, item *schema.Provider) error {
	result := GetProviderDB(ctx, p.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified provider.
func (p *Provider) Update(ctx context.Context, item *schema.Provider) error {
	result := GetProviderDB(ctx, p.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified provider logically.
func (p *Provider) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetProviderDB(ctx, p.DB), id))
}
