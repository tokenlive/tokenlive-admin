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

// QueryEndpointIDsByModel 查询指定模型下所有未删除的端点 ID 列表（单表查询）。
// 单表 SELECT 对 Vitess 分片环境友好，供按模型范围清理/过滤场景复用。
func (a *TenantEndpoint) QueryEndpointIDsByModel(ctx context.Context, tx *gorm.DB, modelID string) ([]string, error) {
	db := util.GetDB(ctx, tx)
	epTable := config.C.FormatTableName("endpoint")

	var endpointIDs []string
	err := db.Table(epTable).
		Where("model_id = ? AND deleted = '0'", modelID).
		Pluck("id", &endpointIDs).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return endpointIDs, nil
}

// QueryEndpointIDsByTenantAndModel 查询指定租户及模型下已授权的端点 ID 列表。
// 采用两步查询替代跨表 JOIN：先按模型查出端点 ID，再在租户关联表中按字面量 IN 过滤，
// 避免 Vitess 分片环境下的跨表 JOIN scatter 风险。
func (a *TenantEndpoint) QueryEndpointIDsByTenantAndModel(ctx context.Context, tenantCode, modelID string) ([]string, error) {
	db := util.GetDB(ctx, a.DB)

	endpointIDs, err := a.QueryEndpointIDsByModel(ctx, db, modelID)
	if err != nil {
		return nil, err
	}
	if len(endpointIDs) == 0 {
		return nil, nil
	}

	teTable := config.C.FormatTableName("tenant_endpoint")
	var result []string
	err = db.Table(teTable).
		Where("tenant_code = ? AND endpoint_id IN ?", tenantCode, endpointIDs).
		Pluck("endpoint_id", &result).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return result, nil
}

// DeleteByTenantAndModel 在事务中删除指定租户及模型关联的全部端点白名单。
// 采用两步执行（先查端点 ID，再用字面量 IN 列表删除）替代 DML 内子查询，
// 规避 Vitess "subqueries in sharded DML" 限制（Error 1235）。
func (a *TenantEndpoint) DeleteByTenantAndModel(ctx context.Context, tx *gorm.DB, tenantCode, modelID string) error {
	db := util.GetDB(ctx, tx)

	endpointIDs, err := a.QueryEndpointIDsByModel(ctx, db, modelID)
	if err != nil {
		return err
	}
	if len(endpointIDs) == 0 {
		return nil // 该模型无端点，无需删除
	}

	teTable := config.C.FormatTableName("tenant_endpoint")
	err = db.Table(teTable).
		Where("tenant_code = ? AND endpoint_id IN ?", tenantCode, endpointIDs).
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
