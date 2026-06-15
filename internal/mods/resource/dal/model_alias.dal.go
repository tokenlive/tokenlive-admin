package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetModelAliasDB returns the database instance for ModelAlias (only active records).
func GetModelAliasDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.ModelAlias)).Where("deleted = '0'")
}

// ModelAlias data access layer
type ModelAlias struct {
	DB *gorm.DB
}

// Query model aliases from the database.
func (m *ModelAlias) Query(ctx context.Context, params schema.ModelAliasQueryParam, opts ...schema.ModelAliasQueryOptions) (*schema.ModelAliasQueryResult, error) {
	var opt schema.ModelAliasQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetModelAliasDB(ctx, m.DB).Preload("Model")
	if v := params.SpaceCode; v != "" {
		db = db.Where("space_code = ?", v)
	}
	if v := params.Alias; v != "" {
		db = db.Where("alias = ?", v)
	}
	if v := params.ModelId; v != "" {
		db = db.Where("model_id = ?", v)
	}

	var list schema.ModelAliases
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &schema.ModelAliasQueryResult{
		PageResult: pageResult,
		Data:       list,
	}, nil
}

// Get the specified model alias from the database.
func (m *ModelAlias) Get(ctx context.Context, id string, opts ...schema.ModelAliasQueryOptions) (*schema.ModelAlias, error) {
	var opt schema.ModelAliasQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.ModelAlias)
	ok, err := util.FindOne(ctx, GetModelAliasDB(ctx, m.DB).Preload("Model").Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified model alias exists.
func (m *ModelAlias) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetModelAliasDB(ctx, m.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// ExistsByUniqueKey checks if a model alias with the given space_code and alias exists.
func (m *ModelAlias) ExistsByUniqueKey(ctx context.Context, spaceCode, alias string) (bool, error) {
	ok, err := util.Exists(ctx, GetModelAliasDB(ctx, m.DB).
		Where("space_code = ? AND alias = ?", spaceCode, alias))
	return ok, errors.WithStack(err)
}

// Create a new model alias.
func (m *ModelAlias) Create(ctx context.Context, item *schema.ModelAlias) error {
	result := GetModelAliasDB(ctx, m.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified model alias.
func (m *ModelAlias) Update(ctx context.Context, item *schema.ModelAlias) error {
	result := GetModelAliasDB(ctx, m.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified model alias logically.
func (m *ModelAlias) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetModelAliasDB(ctx, m.DB), id))
}

// CountByModelId counts active aliases for a given model ID.
func (m *ModelAlias) CountByModelId(ctx context.Context, modelId string) (int64, error) {
	var count int64
	err := GetModelAliasDB(ctx, m.DB).Where("model_id = ?", modelId).Count(&count).Error
	return count, errors.WithStack(err)
}

// ListByModelId returns all active aliases for a given model ID.
func (m *ModelAlias) ListByModelId(ctx context.Context, modelId string) (schema.ModelAliases, error) {
	var list schema.ModelAliases
	err := GetModelAliasDB(ctx, m.DB).Where("model_id = ?", modelId).Find(&list).Error
	return list, errors.WithStack(err)
}

// ExistsByAlias checks if a model alias with the given alias name exists (global scope).
func (m *ModelAlias) ExistsByAlias(ctx context.Context, alias string) (bool, error) {
	ok, err := util.Exists(ctx, GetModelAliasDB(ctx, m.DB).Where("alias = ?", alias))
	return ok, errors.WithStack(err)
}
