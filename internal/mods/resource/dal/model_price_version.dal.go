package dal

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetModelPriceVersionDB returns the database instance for ModelPriceVersion (only active records).
func GetModelPriceVersionDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.ModelPriceVersion))
}

// ModelPriceVersion data access layer
type ModelPriceVersion struct {
	DB *gorm.DB
}

// Query model price versions from the database.
func (m *ModelPriceVersion) Query(ctx context.Context, params schema.ModelPriceVersionQueryParam, opts ...schema.ModelPriceVersionQueryOptions) (*schema.ModelPriceVersionQueryResult, error) {
	var opt schema.ModelPriceVersionQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetModelPriceVersionDB(ctx, m.DB)
	if v := params.ModelID; len(v) > 0 {
		db = db.Where("model_id = ?", v)
	}
	if v := params.Status; len(v) > 0 {
		db = db.Where("status = ?", v)
	}
	if v := params.Currency; len(v) > 0 {
		db = db.Where("currency = ?", v)
	}

	var list schema.ModelPriceVersions
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &schema.ModelPriceVersionQueryResult{
		PageResult: pageResult,
		Data:       list,
	}, nil
}

// Get the specified model price version from the database.
func (m *ModelPriceVersion) Get(ctx context.Context, id string, opts ...schema.ModelPriceVersionQueryOptions) (*schema.ModelPriceVersion, error) {
	var opt schema.ModelPriceVersionQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.ModelPriceVersion)
	ok, err := util.FindOne(ctx, GetModelPriceVersionDB(ctx, m.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// GetCurrentPrice gets the currently effective price version for a model.
func (m *ModelPriceVersion) GetCurrentPrice(ctx context.Context, modelID, currency string) (*schema.ModelPriceVersion, error) {
	if currency == "" {
		currency = "CNY"
	}
	now := time.Now()

	item := new(schema.ModelPriceVersion)
	db := GetModelPriceVersionDB(ctx, m.DB).
		Where("model_id = ? AND currency = ? AND status = ?", modelID, currency, schema.ModelPriceStatusActive).
		Where("effective_from <= ?", now).
		Where("effective_until IS NULL OR effective_until > ?", now).
		Order("effective_from DESC")

	ok, err := util.FindOne(ctx, db, util.QueryOptions{}, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Create a new model price version.
func (m *ModelPriceVersion) Create(ctx context.Context, item *schema.ModelPriceVersion) error {
	result := GetModelPriceVersionDB(ctx, m.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified model price version.
func (m *ModelPriceVersion) Update(ctx context.Context, item *schema.ModelPriceVersion) error {
	result := GetModelPriceVersionDB(ctx, m.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified model price version.
func (m *ModelPriceVersion) Delete(ctx context.Context, id string) error {
	result := GetModelPriceVersionDB(ctx, m.DB).Where("id=?", id).Delete(new(schema.ModelPriceVersion))
	return errors.WithStack(result.Error)
}

// QueryByModelID queries all price versions for a model.
func (m *ModelPriceVersion) QueryByModelID(ctx context.Context, modelID string) (schema.ModelPriceVersions, error) {
	var list schema.ModelPriceVersions
	if err := GetModelPriceVersionDB(ctx, m.DB).Where("model_id=?", modelID).Order("effective_from DESC").Find(&list).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return list, nil
}
