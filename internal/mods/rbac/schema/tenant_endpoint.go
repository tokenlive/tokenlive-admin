package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
)

// TenantEndpoint 租户-端点关联表
type TenantEndpoint struct {
	ID         string    `json:"id" gorm:"type:char(20);primaryKey;comment:主键ID (XID);"`
	TenantCode string    `json:"tenant_code" gorm:"type:varchar(64);not null;uniqueIndex:uniq_tenant_endpoint,priority:1;index:idx_te_tenant_code;comment:租户唯一英文编码，关联 tenant.code;"`
	EndpointID string    `json:"endpoint_id" gorm:"type:char(20);not null;uniqueIndex:uniq_tenant_endpoint,priority:2;index:idx_te_endpoint_id;comment:端点ID，关联 endpoint.id;"`
	Creator    string    `json:"creator" gorm:"type:varchar(255);default:null;comment:创建者;"`
	CreatedAt  time.Time `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
}

func (TenantEndpoint) TableName() string {
	return config.C.FormatTableName("tenant_endpoint")
}

// TenantEndpointForm 保存租户端点白名单表单
type TenantEndpointForm struct {
	TenantCode  string   `json:"tenant_code" binding:"required"`
	ModelID     string   `json:"model_id" binding:"required"`
	EndpointIDs []string `json:"endpoint_ids"` // 允许的端点 ID 列表，空表示不限制（全放通）
}
