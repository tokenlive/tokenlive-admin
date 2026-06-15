package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetTenantEndpointDB 获取租户-端点关联表 DB 实例
func GetTenantEndpointDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.TenantEndpoint))
}

// TenantEndpoint RBAC 模块下的租户-端点关联数据访问层
type TenantEndpoint struct {
	DB *gorm.DB
}

// QueryEndpointIDsByTenantAndModel 查询指定租户及模型下已授权的端点 ID 列表
func (a *TenantEndpoint) QueryEndpointIDsByTenantAndModel(ctx context.Context, tenantCode, modelID string) ([]string, error) {
	db := util.GetDB(ctx, a.DB)
	teTable := config.C.FormatTableName("tenant_endpoint")
	epTable := config.C.FormatTableName("endpoint")

	var endpointIDs []string
	err := db.Table(teTable+" AS te").
		Select("te.endpoint_id").
		Joins("JOIN "+epTable+" AS ep ON te.endpoint_id = ep.id AND ep.deleted = '0'").
		Where("te.tenant_code = ? AND ep.model_id = ?", tenantCode, modelID).
		Pluck("te.endpoint_id", &endpointIDs).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return endpointIDs, nil
}

// DeleteByTenantAndModel 在事务中删除指定租户及模型关联的全部端点白名单（通过子查询按 model_id 范围删除）
func (a *TenantEndpoint) DeleteByTenantAndModel(ctx context.Context, tx *gorm.DB, tenantCode, modelID string) error {
	db := util.GetDB(ctx, tx)
	teTable := config.C.FormatTableName("tenant_endpoint")
	epTable := config.C.FormatTableName("endpoint")

	// 使用子查询按 model_id 范围删除
	subQuery := db.Table(epTable).Select("id").Where("model_id = ? AND deleted = '0'", modelID)
	err := db.Table(teTable).Where("tenant_code = ? AND endpoint_id IN (?)", tenantCode, subQuery).
		Delete(nil).Error
	return errors.WithStack(err)
}

// CreateInBatches 在事务中批量插入端点白名单关联
func (a *TenantEndpoint) CreateInBatches(ctx context.Context, tx *gorm.DB, items []*schema.TenantEndpoint) error {
	if len(items) == 0 {
		return nil
	}
	db := GetTenantEndpointDB(ctx, tx)
	err := db.CreateInBatches(items, 100).Error
	return errors.WithStack(err)
}
