package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetModelDB returns the database instance for Model (only active records).
func GetModelDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	tableName := config.C.FormatTableName("model")
	return util.GetDB(ctx, defDB).Model(new(schema.Model)).Where(tableName + ".deleted = '0'")
}

// Model data access layer
type Model struct {
	DB *gorm.DB
}

// Query models from the database.
func (m *Model) Query(ctx context.Context, params schema.ModelQueryParam, opts ...schema.ModelQueryOptions) (*schema.ModelQueryResult, error) {
	var opt schema.ModelQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	tableName := config.C.FormatTableName("model")
	db := GetModelDB(ctx, m.DB)

	if v := params.LikeName; len(v) > 0 {
		db = db.Where(tableName+".model_name LIKE ?", "%"+v+"%")
	}
	if v := params.ModelCode; len(v) > 0 {
		db = db.Where(tableName+".model_code = ?", v)
	}
	if v := params.SpaceCode; len(v) > 0 {
		db = db.Where(tableName+".space_code = ?", v)
	}

	if !util.FromIsRootUser(ctx) {
		user := util.FromUsername(ctx)
		tenant := util.FromTenant(ctx)
		permTable := config.C.FormatTableName("data_permission")
		db = db.Where(tableName+".id IN (SELECT data_id FROM "+permTable+" WHERE type = ? AND user = ? AND tenant = ? AND permission & 1 = 1)",
			schema.DataPermissionTypeModel, user, tenant)
	}

	var list schema.Models
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &schema.ModelQueryResult{
		PageResult: pageResult,
		Data:       list,
	}, nil
}

// Get the specified model from the database.
func (m *Model) Get(ctx context.Context, id string, opts ...schema.ModelQueryOptions) (*schema.Model, error) {
	var opt schema.ModelQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.Model)
	ok, err := util.FindOne(ctx, GetModelDB(ctx, m.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified model exists.
func (m *Model) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetModelDB(ctx, m.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// ExistsByModelCode checks if a model with the given model_code exists.
func (m *Model) ExistsByModelCode(ctx context.Context, modelCode string) (bool, error) {
	ok, err := util.Exists(ctx, GetModelDB(ctx, m.DB).Where("model_code=?", modelCode))
	return ok, errors.WithStack(err)
}

// ExistsByModelName checks if a model with the given model_name exists.
func (m *Model) ExistsByModelName(ctx context.Context, modelName string) (bool, error) {
	ok, err := util.Exists(ctx, GetModelDB(ctx, m.DB).Where("model_name=?", modelName))
	return ok, errors.WithStack(err)
}

// Create a new model.
func (m *Model) Create(ctx context.Context, item *schema.Model) error {
	result := GetModelDB(ctx, m.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified model.
func (m *Model) Update(ctx context.Context, item *schema.Model) error {
	result := GetModelDB(ctx, m.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified model logically.
func (m *Model) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetModelDB(ctx, m.DB), id))
}
