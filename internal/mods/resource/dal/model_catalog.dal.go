package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetModelCatalogDB returns the database instance for ModelCatalog.
func GetModelCatalogDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.ModelCatalog))
}

// ModelCatalog data access layer
type ModelCatalog struct {
	DB *gorm.DB
}

// Query model catalogs from the database.
func (m *ModelCatalog) Query(ctx context.Context, params schema.ModelCatalogQueryParam, opts ...schema.ModelCatalogQueryOptions) (*schema.ModelCatalogQueryResult, error) {
	var opt schema.ModelCatalogQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetModelCatalogDB(ctx, m.DB)
	if v := params.LikeSlug; len(v) > 0 {
		db = db.Where("slug LIKE ?", "%"+v+"%")
	}
	if v := params.Status; len(v) > 0 {
		db = db.Where("status = ?", v)
	}
	if v := params.Visibility; len(v) > 0 {
		db = db.Where("visibility = ?", v)
	}
	if v := params.Featured; v != nil {
		db = db.Where("featured = ?", *v)
	}
	if v := params.ModelCode; len(v) > 0 {
		db = db.Where("model_code = ?", v)
	}

	var list schema.ModelCatalogs
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &schema.ModelCatalogQueryResult{
		PageResult: pageResult,
		Data:       list,
	}, nil
}

// Get the specified model catalog from the database.
func (m *ModelCatalog) Get(ctx context.Context, modelID string, opts ...schema.ModelCatalogQueryOptions) (*schema.ModelCatalog, error) {
	var opt schema.ModelCatalogQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.ModelCatalog)
	ok, err := util.FindOne(ctx, GetModelCatalogDB(ctx, m.DB).Where("model_id=?", modelID), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// GetBySlug gets a model catalog by slug.
func (m *ModelCatalog) GetBySlug(ctx context.Context, slug string) (*schema.ModelCatalog, error) {
	item := new(schema.ModelCatalog)
	ok, err := util.FindOne(ctx, GetModelCatalogDB(ctx, m.DB).Where("slug=?", slug), util.QueryOptions{}, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified model catalog exists.
func (m *ModelCatalog) Exists(ctx context.Context, modelID string) (bool, error) {
	ok, err := util.Exists(ctx, GetModelCatalogDB(ctx, m.DB).Where("model_id=?", modelID))
	return ok, errors.WithStack(err)
}

// ExistsBySlug checks if a model catalog with the given slug exists.
func (m *ModelCatalog) ExistsBySlug(ctx context.Context, slug string, excludeModelID string) (bool, error) {
	db := GetModelCatalogDB(ctx, m.DB).Where("slug = ?", slug)
	if len(excludeModelID) > 0 {
		db = db.Where("model_id != ?", excludeModelID)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, errors.WithStack(err)
	}
	return count > 0, nil
}

// Create a new model catalog.
func (m *ModelCatalog) Create(ctx context.Context, item *schema.ModelCatalog) error {
	result := GetModelCatalogDB(ctx, m.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified model catalog.
func (m *ModelCatalog) Update(ctx context.Context, item *schema.ModelCatalog) error {
	result := GetModelCatalogDB(ctx, m.DB).Where("model_id=?", item.ModelID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified model catalog.
func (m *ModelCatalog) Delete(ctx context.Context, modelID string) error {
	result := GetModelCatalogDB(ctx, m.DB).Where("model_id=?", modelID).Delete(new(schema.ModelCatalog))
	return errors.WithStack(result.Error)
}

// QueryPublic queries public available model catalogs for portal display.
func (m *ModelCatalog) QueryPublic(ctx context.Context, limit int) (schema.ModelCatalogs, error) {
	var list schema.ModelCatalogs
	db := GetModelCatalogDB(ctx, m.DB).
		Where("visibility = ? AND status = ?", schema.ModelCatalogVisibilityPublic, schema.ModelCatalogStatusAvailable).
		Order("featured DESC, sort_weight DESC, published_at DESC")
	if limit > 0 {
		db = db.Limit(limit)
	}
	if err := db.Find(&list).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return list, nil
}
