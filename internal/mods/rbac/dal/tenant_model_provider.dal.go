package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetTenantModelProviderDB 获取租户-模型-供应商白名单表 DB 实例
func GetTenantModelProviderDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.TenantModelProvider))
}

// TenantModelProvider RBAC 模块下的租户-模型-供应商白名单数据访问层
type TenantModelProvider struct {
	DB *gorm.DB
}

// QueryByTenantAndModel 获取指定租户及模型关联的供应商绑定记录
func (a *TenantModelProvider) QueryByTenantAndModel(ctx context.Context, tenantCode, modelID string) ([]*schema.TenantModelProvider, error) {
	db := GetTenantModelProviderDB(ctx, a.DB).Where("tenant_code = ? AND model_id = ?", tenantCode, modelID)
	var list []*schema.TenantModelProvider
	err := db.Find(&list).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return list, nil
}

// DeleteByTenantAndModel 在事务中删除指定租户及模型关联的全部供应商白名单
func (a *TenantModelProvider) DeleteByTenantAndModel(ctx context.Context, tx *gorm.DB, tenantCode, modelID string) error {
	db := GetTenantModelProviderDB(ctx, tx).Where("tenant_code = ? AND model_id = ?", tenantCode, modelID)
	err := db.Delete(&schema.TenantModelProvider{}).Error
	return errors.WithStack(err)
}

// CreateInBatches 在事务中批量插入供应商白名单关联
func (a *TenantModelProvider) CreateInBatches(ctx context.Context, tx *gorm.DB, items []*schema.TenantModelProvider) error {
	if len(items) == 0 {
		return nil
	}
	db := GetTenantModelProviderDB(ctx, tx)
	err := db.CreateInBatches(items, 100).Error
	return errors.WithStack(err)
}
