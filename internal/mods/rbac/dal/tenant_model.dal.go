package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Get tenant_model storage instance
func GetTenantModelDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.TenantModel))
}

// TenantModel management for RBAC
type TenantModel struct {
	DB *gorm.DB
}

// QueryByTenantCode queries all models associated with the specified tenant.
func (a *TenantModel) QueryByTenantCode(ctx context.Context, tenantCode string) ([]*schema.TenantModel, error) {
	db := GetTenantModelDB(ctx, a.DB).Where("tenant_code = ?", tenantCode)
	var list []*schema.TenantModel
	err := db.Find(&list).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return list, nil
}

// DeleteByTenantCode deletes all bindings of a tenant inside a transaction.
func (a *TenantModel) DeleteByTenantCode(ctx context.Context, tx *gorm.DB, tenantCode string) error {
	db := GetTenantModelDB(ctx, tx).Where("tenant_code = ?", tenantCode)
	err := db.Delete(&schema.TenantModel{}).Error
	return errors.WithStack(err)
}

// CreateInBatches inserts a list of bindings inside a transaction.
func (a *TenantModel) CreateInBatches(ctx context.Context, tx *gorm.DB, items []*schema.TenantModel) error {
	if len(items) == 0 {
		return nil
	}
	db := GetTenantModelDB(ctx, tx)
	err := db.CreateInBatches(items, 100).Error
	return errors.WithStack(err)
}
