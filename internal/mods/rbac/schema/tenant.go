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
	ID          string     `json:"id" gorm:"size:20;primarykey;"` // Unique ID
	Code        string     `json:"code" gorm:"size:64;index"`     // Unique identifier code (e.g. company-a)
	Name        string     `json:"name" gorm:"size:128;index"`    // Name of tenant
	Status      string     `json:"status" gorm:"size:20;index"`   // Status of tenant (activated, freezed)
	Description string     `json:"description" gorm:"size:255;"`  // Description
	APIKey      string     `json:"api_key" gorm:"size:128;index"` // Tenant API Key (toB scenario)
	Creator     string     `json:"creator" gorm:"size:255"`       // Creator
	Modifier    string     `json:"modifier" gorm:"size:255"`      // Modifier
	CreatedAt   time.Time  `json:"created_at" gorm:"index;"`      // Create time
	UpdatedAt   time.Time  `json:"updated_at" gorm:"index;"`      // Update time
	Deleted     string     `json:"-" gorm:"size:20;default:0"`    // Logical delete flag
	DeletedAt   *time.Time `json:"-" gorm:"index;"`               // Delete time
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
