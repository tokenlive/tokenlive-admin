package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetModelCatalogI18nDB returns the database instance for ModelCatalogI18n.
func GetModelCatalogI18nDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.ModelCatalogI18n))
}

// ModelCatalogI18n data access layer
type ModelCatalogI18n struct {
	DB *gorm.DB
}

// Query model catalog i18n entries from the database.
func (m *ModelCatalogI18n) Query(ctx context.Context, params schema.ModelCatalogI18nQueryParam, opts ...schema.ModelCatalogI18nQueryOptions) (*schema.ModelCatalogI18nQueryResult, error) {
	var opt schema.ModelCatalogI18nQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetModelCatalogI18nDB(ctx, m.DB)
	if v := params.ModelID; len(v) > 0 {
		db = db.Where("model_id = ?", v)
	}
	if v := params.Locale; len(v) > 0 {
		db = db.Where("locale = ?", v)
	}

	var list schema.ModelCatalogI18ns
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &schema.ModelCatalogI18nQueryResult{
		PageResult: pageResult,
		Data:       list,
	}, nil
}

// Get gets a specific i18n entry by model_id and locale.
func (m *ModelCatalogI18n) Get(ctx context.Context, modelID, locale string) (*schema.ModelCatalogI18n, error) {
	item := new(schema.ModelCatalogI18n)
	ok, err := util.FindOne(ctx, GetModelCatalogI18nDB(ctx, m.DB).Where("model_id=? AND locale=?", modelID, locale), util.QueryOptions{}, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// QueryByModelID gets all i18n entries for a model.
func (m *ModelCatalogI18n) QueryByModelID(ctx context.Context, modelID string) (schema.ModelCatalogI18ns, error) {
	var list schema.ModelCatalogI18ns
	if err := GetModelCatalogI18nDB(ctx, m.DB).Where("model_id=?", modelID).Find(&list).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return list, nil
}

// Create a new i18n entry.
func (m *ModelCatalogI18n) Create(ctx context.Context, item *schema.ModelCatalogI18n) error {
	result := GetModelCatalogI18nDB(ctx, m.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified i18n entry.
func (m *ModelCatalogI18n) Update(ctx context.Context, item *schema.ModelCatalogI18n) error {
	result := GetModelCatalogI18nDB(ctx, m.DB).
		Where("model_id=? AND locale=?", item.ModelID, item.Locale).
		Select("display_name", "short_description", "long_description", "seo_title", "seo_description", "tags", "modifier", "updated_at").
		Updates(item)
	return errors.WithStack(result.Error)
}

// Upsert creates or updates an i18n entry.
func (m *ModelCatalogI18n) Upsert(ctx context.Context, item *schema.ModelCatalogI18n) error {
	existing, err := m.Get(ctx, item.ModelID, item.Locale)
	if err != nil {
		return err
	}
	if existing != nil {
		return m.Update(ctx, item)
	}
	return m.Create(ctx, item)
}

// DeleteByModelID deletes all i18n entries for a model.
func (m *ModelCatalogI18n) DeleteByModelID(ctx context.Context, modelID string) error {
	result := GetModelCatalogI18nDB(ctx, m.DB).Where("model_id=?", modelID).Delete(new(schema.ModelCatalogI18n))
	return errors.WithStack(result.Error)
}

// Delete deletes a specific i18n entry.
func (m *ModelCatalogI18n) Delete(ctx context.Context, modelID, locale string) error {
	result := GetModelCatalogI18nDB(ctx, m.DB).Where("model_id=? AND locale=?", modelID, locale).Delete(new(schema.ModelCatalogI18n))
	return errors.WithStack(result.Error)
}
