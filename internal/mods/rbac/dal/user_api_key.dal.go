package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetUserAPIKeyDB 获取 API Key 表的数据库实例并关联 Model
func GetUserAPIKeyDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.UserAPIKey))
}

// UserAPIKey 用户 API Key 的数据持久层结构体
type UserAPIKey struct {
	DB *gorm.DB
}

// Query 分页查询用户 API Key
func (a *UserAPIKey) Query(ctx context.Context, params schema.UserAPIKeyQueryParam, opts ...schema.UserAPIKeyQueryOptions) (*schema.UserAPIKeyQueryResult, error) {
	var opt schema.UserAPIKeyQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetUserAPIKeyDB(ctx, a.DB)
	if v := params.UserID; len(v) > 0 {
		db = db.Where("user_id = ?", v)
	}
	if v := params.LikeName; len(v) > 0 {
		db = db.Where("name LIKE ?", "%"+v+"%")
	}
	if params.Status > 0 {
		db = db.Where("status = ?", params.Status)
	}

	var list schema.UserAPIKeys
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &schema.UserAPIKeyQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get 根据 ID 获取单条 API Key
func (a *UserAPIKey) Get(ctx context.Context, id string, opts ...schema.UserAPIKeyQueryOptions) (*schema.UserAPIKey, error) {
	var opt schema.UserAPIKeyQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.UserAPIKey)
	ok, err := util.FindOne(ctx, GetUserAPIKeyDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// GetByAPIKey 根据实际的 API Key 字符串获取对应的信息（供精确匹配或校验使用）
func (a *UserAPIKey) GetByAPIKey(ctx context.Context, apiKey string, opts ...schema.UserAPIKeyQueryOptions) (*schema.UserAPIKey, error) {
	var opt schema.UserAPIKeyQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.UserAPIKey)
	ok, err := util.FindOne(ctx, GetUserAPIKeyDB(ctx, a.DB).Where("api_key=?", apiKey), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists 检查主键 ID 是否存在
func (a *UserAPIKey) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetUserAPIKeyDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// ExistsAPIKey 检查 API Key 字符串是否已存在（避免生成重复的 Key）
func (a *UserAPIKey) ExistsAPIKey(ctx context.Context, apiKey string) (bool, error) {
	ok, err := util.Exists(ctx, GetUserAPIKeyDB(ctx, a.DB).Where("api_key=?", apiKey))
	return ok, errors.WithStack(err)
}

// Create 新增一条记录
func (a *UserAPIKey) Create(ctx context.Context, item *schema.UserAPIKey) error {
	result := GetUserAPIKeyDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update 更新记录
func (a *UserAPIKey) Update(ctx context.Context, item *schema.UserAPIKey, selectFields ...string) error {
	db := GetUserAPIKeyDB(ctx, a.DB).Where("id=?", item.ID)
	if len(selectFields) > 0 {
		db = db.Select(selectFields)
	} else {
		db = db.Select("*").Omit("created_at", "api_key") // 创建时间与 Key 字符串一旦生成，就不允许二次修改
	}
	result := db.Updates(item)
	return errors.WithStack(result.Error)
}

// Delete 逻辑删除记录
func (a *UserAPIKey) Delete(ctx context.Context, id string) error {
	result := GetUserAPIKeyDB(ctx, a.DB).Where("id=?", id).Delete(new(schema.UserAPIKey))
	return errors.WithStack(result.Error)
}
