package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
)

// TenantModelProvider 租户-模型-上游供应商白名单关联表
type TenantModelProvider struct {
	ID         string    `json:"id" gorm:"size:20;primarykey;comment:主键 ID;"`
	TenantCode string    `json:"tenant_code" gorm:"size:64;not null;uniqueIndex:uniq_tenant_model_provider,priority:1;index:idx_tmp_tenant_code;comment:租户唯一英文编码;"`
	ModelID    string    `json:"model_id" gorm:"size:20;not null;uniqueIndex:uniq_tenant_model_provider,priority:2;index:idx_tmp_model_id;comment:模型主键 ID;"`
	ProviderID string    `json:"provider_id" gorm:"size:20;not null;uniqueIndex:uniq_tenant_model_provider,priority:3;index:idx_tmp_provider_id;comment:供应商主键 ID;"`
	Creator    string    `json:"creator" gorm:"size:255;comment:创建人;"`
	CreatedAt  time.Time `json:"created_at" gorm:"index;comment:创建时间;"`
}

func (TenantModelProvider) TableName() string {
	return config.C.FormatTableName("tenant_model_provider")
}

// TenantModelProviderForm 保存租户模型供应商白名单表单
type TenantModelProviderForm struct {
	TenantCode  string   `json:"tenant_code" binding:"required"`
	ModelID     string   `json:"model_id" binding:"required"`
	ProviderIDs []string `json:"provider_ids"` // 允许的供应商 ID 列表，空表示不限制（全放通）
}
