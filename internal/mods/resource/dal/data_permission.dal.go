package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Get data permission storage instance
func GetDataPermissionDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.DataPermission))
}

// Data permission management
type DataPermission struct {
	DB *gorm.DB
}

// Query data permissions from the database based on the provided parameters and options.
func (a *DataPermission) Query(ctx context.Context, params schema.DataPermissionQueryParam, opts ...schema.DataPermissionQueryOptions) (*schema.DataPermissionQueryResult, error) {
	var opt schema.DataPermissionQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetDataPermissionDB(ctx, a.DB)

	if v := params.Type; len(v) > 0 {
		db = db.Where("type = ?", v)
	}
	if v := params.DataId; len(v) > 0 {
		db = db.Where("data_id = ?", v)
	}
	if v := params.User; len(v) > 0 {
		db = db.Where("user LIKE ?", "%"+v+"%")
	}

	var list schema.DataPermissions
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &schema.DataPermissionQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get the specified data permission from the database.
func (a *DataPermission) Get(ctx context.Context, id string, opts ...schema.DataPermissionQueryOptions) (*schema.DataPermission, error) {
	var opt schema.DataPermissionQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.DataPermission)
	ok, err := util.FindOne(ctx, GetDataPermissionDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified data permission exists in the database.
func (a *DataPermission) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetDataPermissionDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// Create a new data permission.
func (a *DataPermission) Create(ctx context.Context, item *schema.DataPermission) error {
	result := GetDataPermissionDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified data permission in the database.
func (a *DataPermission) Update(ctx context.Context, item *schema.DataPermission) error {
	result := GetDataPermissionDB(ctx, a.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified data permission from the database.
func (a *DataPermission) Delete(ctx context.Context, id string) error {
	result := GetDataPermissionDB(ctx, a.DB).Where("id=?", id).Delete(new(schema.DataPermission))
	return errors.WithStack(result.Error)
}

// DeleteByTypeAndDataId deletes data permissions by type and data ID.
func (a *DataPermission) DeleteByTypeAndDataId(ctx context.Context, dataType, dataId string) error {
	result := GetDataPermissionDB(ctx, a.DB).Where("type=? AND data_id=?", dataType, dataId).Delete(new(schema.DataPermission))
	return errors.WithStack(result.Error)
}

// HasReadPermission checks if a user has read permission for a specific data item.
func (a *DataPermission) HasReadPermission(ctx context.Context, dataType, dataId, user, tenant string) (bool, error) {
	ok, err := util.Exists(ctx, GetDataPermissionDB(ctx, a.DB).
		Where("type=? AND data_id=? AND user=? AND tenant=? AND permission & 1 = 1", dataType, dataId, user, tenant))
	return ok, errors.WithStack(err)
}
