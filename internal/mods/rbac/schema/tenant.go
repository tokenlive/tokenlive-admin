package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

const (
	TenantStatusActivated = "activated"
	TenantStatusFreezed   = "freezed"
)

// Tenant management for multi-tenancy
type Tenant struct {
	ID          string     `json:"id" gorm:"type:char(20);primaryKey;comment:主键ID (XID);"`
	Code        string     `json:"code" gorm:"type:varchar(64);not null;uniqueIndex:uniq_tenant_code_deleted,priority:1;comment:租户唯一英文编码，如 default、company-a;"`
	Name        string     `json:"name" gorm:"type:varchar(128);not null;comment:租户名称，如 默认租户、演示组织;"`
	Status      string     `json:"status" gorm:"type:varchar(20);not null;default:activated;comment:状态: activated-启用, freezed-冻结;"`
	Description string     `json:"description" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	APIKey      string     `json:"api_key" gorm:"type:varchar(128);default:null;uniqueIndex:uniq_tenant_api_key;comment:租户 API Key (toB场景专属);"`
	Creator     string     `json:"creator" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier    string     `json:"modifier" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt   time.Time  `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted     string     `json:"-" gorm:"type:varchar(20);not null;default:'0';uniqueIndex:uniq_tenant_code_deleted,priority:2;comment:逻辑删除标识;"`
	DeletedAt   *time.Time `json:"-" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
}

func (a *Tenant) TableName() string {
	return config.C.FormatTableName("tenant")
}

// TenantQueryParam defines the query parameters for Tenant.
type TenantQueryParam struct {
	util.PaginationParam
	LikeCode string `form:"code"`                                        // Code (like)
	LikeName string `form:"name"`                                        // Name (like)
	Status   string `form:"status" binding:"oneof=activated freezed ''"` // Status
}

// TenantQueryOptions defines the query options for Tenant.
type TenantQueryOptions struct {
	util.QueryOptions
}

// TenantQueryResult defines the query result for Tenant.
type TenantQueryResult struct {
	Data       Tenants
	PageResult *util.PaginationResult
}

// Tenants defines a slice of Tenant.
type Tenants []*Tenant

func (a Tenants) ToIDs() []string {
	var ids []string
	for _, item := range a {
		ids = append(ids, item.ID)
	}
	return ids
}

// TenantForm defines the data structure for creating/updating a Tenant.
type TenantForm struct {
	Code        string `json:"code" binding:"required,max=64"`                    // Code
	Name        string `json:"name" binding:"required,max=128"`                   // Name
	Status      string `json:"status" binding:"required,oneof=activated freezed"` // Status (activated, freezed)
	Description string `json:"description" binding:"max=255"`                     // Description
	APIKey      string `json:"api_key" binding:"max=128"`                         // Tenant API Key
}

func (a *TenantForm) Validate() error {
	return nil
}

func (a *TenantForm) FillTo(tenant *Tenant) error {
	tenant.Code = a.Code
	tenant.Name = a.Name
	tenant.Status = a.Status
	tenant.Description = a.Description
	tenant.APIKey = a.APIKey
	return nil
}
