package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
)

// TenantModel represents the multi-to-multi association between Tenant and Model
type TenantModel struct {
	ID         string    `json:"id" gorm:"type:char(20);primaryKey;comment:主键ID (XID);"`
	TenantCode string    `json:"tenant_code" gorm:"type:varchar(64);not null;uniqueIndex:uniq_tenant_model,priority:1;index:idx_tenant_model_tenant_code;comment:租户唯一英文编码，关联 tenant.code;"`
	ModelID    string    `json:"model_id" gorm:"type:char(20);not null;uniqueIndex:uniq_tenant_model,priority:2;index:idx_tenant_model_model_id;comment:模型ID，关联 model.id;"`
	Creator    string    `json:"creator" gorm:"type:varchar(255);default:null;comment:创建者;"`
	CreatedAt  time.Time `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
}

func (TenantModel) TableName() string {
	return config.C.FormatTableName("tenant_model")
}

// TenantModelForm defines the data structure for batch binding tenant models.
type TenantModelForm struct {
	TenantCode string   `json:"tenant_code" binding:"required"`
	ModelIDs   []string `json:"model_ids"`
}
