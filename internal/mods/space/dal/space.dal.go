package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/space/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetSpaceDB returns the database instance for Space (only active records).
func GetSpaceDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.Space)).Where("deleted = '0'")
}

// Space data access layer
type Space struct {
	DB *gorm.DB
}

// Query spaces from the database based on the provided parameters and options.
func (a *Space) Query(ctx context.Context, params schema.SpaceQueryParam, opts ...schema.SpaceQueryOptions) (*schema.SpaceQueryResult, error) {
	var opt schema.SpaceQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetSpaceDB(ctx, a.DB)
	if v := params.LikeName; len(v) > 0 {
		db = db.Where("name LIKE ?", "%"+v+"%")
	}
	if v := params.LikeCode; len(v) > 0 {
		db = db.Where("code LIKE ?", "%"+v+"%")
	}
	if v := params.Tenant; len(v) > 0 {
		db = db.Where("tenant = ?", v)
	}

	var list schema.Spaces
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &schema.SpaceQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get the specified space from the database.
func (a *Space) Get(ctx context.Context, id string, opts ...schema.SpaceQueryOptions) (*schema.Space, error) {
	var opt schema.SpaceQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.Space)
	ok, err := util.FindOne(ctx, GetSpaceDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified space exists in the database.
func (a *Space) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetSpaceDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// ExistsCode checks if a space with the given code exists.
func (a *Space) ExistsCode(ctx context.Context, code string) (bool, error) {
	ok, err := util.Exists(ctx, GetSpaceDB(ctx, a.DB).Where("code=?", code))
	return ok, errors.WithStack(err)
}

// Create a new space.
func (a *Space) Create(ctx context.Context, item *schema.Space) error {
	result := GetSpaceDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified space in the database.
func (a *Space) Update(ctx context.Context, item *schema.Space) error {
	result := GetSpaceDB(ctx, a.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified space logically (soft delete).
func (a *Space) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetSpaceDB(ctx, a.DB), id))
}

// ExistsModels checks if there are any active models associated with the space code.
func (a *Space) ExistsModels(ctx context.Context, spaceCode string) (bool, error) {
	tableName := config.C.FormatTableName("model")
	var count int64
	err := a.DB.Table(tableName).Where("space_code = ? AND deleted = '0'", spaceCode).Count(&count).Error
	if err != nil {
		return false, errors.WithStack(err)
	}
	return count > 0, nil
}
