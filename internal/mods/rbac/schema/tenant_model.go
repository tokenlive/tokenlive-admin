package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
)

// TenantModel represents the multi-to-multi association between Tenant and Model
type TenantModel struct {
	ID         string    `json:"id" gorm:"size:20;primaryKey;comment:Unique ID;"`
	TenantCode string    `json:"tenant_code" gorm:"size:64;not null;uniqueIndex:uniq_tenant_model;index:idx_tenant_model_tenant_code;comment:Tenant unique code;"`
	ModelID    string    `json:"model_id" gorm:"size:20;not null;uniqueIndex:uniq_tenant_model;index:idx_tenant_model_model_id;comment:Model primary key ID;"`
	Creator    string    `json:"creator" gorm:"size:255;comment:Creator username;"`
	CreatedAt  time.Time `json:"created_at" gorm:"type:timestamp;autoCreateTime;comment:Create time;"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"type:timestamp;autoUpdateTime;comment:Update time;"`
}

func (TenantModel) TableName() string {
	return config.C.FormatTableName("tenant_model")
}

// TenantModelForm defines the data structure for batch binding tenant models.
type TenantModelForm struct {
	TenantCode string   `json:"tenant_code" binding:"required"`
	ModelIDs   []string `json:"model_ids"`
}
