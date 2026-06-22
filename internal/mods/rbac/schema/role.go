package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

const (
	RoleStatusEnabled  = "enabled"  // Enabled
	RoleStatusDisabled = "disabled" // Disabled

	RoleResultTypeSelect = "select" // Select
)

// Role management for RBAC
type Role struct {
	ID          string    `json:"id" gorm:"type:varchar(20);primaryKey;comment:ID;"`
	Code        string    `json:"code" gorm:"type:varchar(32);default:null;index:idx_role_code;comment:角色编码;"`
	Name        string    `json:"name" gorm:"type:varchar(128);default:null;index:idx_role_name;comment:角色名称;"`
	Description string    `json:"description" gorm:"type:varchar(1024);default:null;comment:角色描述;"`
	Sequence    int       `json:"sequence" gorm:"type:bigint;default:null;index:idx_role_sequence;comment:角色序列;"`
	Tenant      string    `json:"tenant" gorm:"type:varchar(255);default:null;comment:租户信息;"`
	Status      string    `json:"status" gorm:"type:varchar(20);default:null;index:idx_role_status;comment:状态;"`
	CreatedAt   time.Time `json:"created_at" gorm:"type:datetime(3);default:null;autoCreateTime;comment:创建时间;"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"type:datetime(3);default:null;autoUpdateTime;comment:更新时间;"`
	Menus       RoleMenus `json:"menus" gorm:"-"`                                   // Role menu list
}

func (a *Role) TableName() string {
	return config.C.FormatTableName("role")
}

// Defining the query parameters for the `Role` struct.
type RoleQueryParam struct {
	util.PaginationParam
	LikeName    string     `form:"name"`                                       // Display name of role
	Status      string     `form:"status" binding:"oneof=disabled enabled ''"` // Status of role (disabled, enabled)
	ResultType  string     `form:"resultType"`                                 // Result type (options: select)
	InIDs       []string   `form:"-"`                                          // ID list
	GtUpdatedAt *time.Time `form:"-"`                                          // Update time is greater than
}

// Defining the query options for the `Role` struct.
type RoleQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `Role` struct.
type RoleQueryResult struct {
	Data       Roles
	PageResult *util.PaginationResult
}

// Defining the slice of `Role` struct.
type Roles []*Role

// Defining the data structure for creating a `Role` struct.
type RoleForm struct {
	Code        string    `json:"code" binding:"required,max=32"`                   // Code of role (unique)
	Name        string    `json:"name" binding:"required,max=128"`                  // Display name of role
	Description string    `json:"description"`                                      // Details about role
	Sequence    int       `json:"sequence"`                                         // Sequence for sorting
	Status      string    `json:"status" binding:"required,oneof=disabled enabled"` // Status of role (enabled, disabled)
	Menus       RoleMenus `json:"menus"`                                            // Role menu list
}

// A validation function for the `RoleForm` struct.
func (a *RoleForm) Validate() error {
	return nil
}

func (a *RoleForm) FillTo(role *Role) error {
	role.Code = a.Code
	role.Name = a.Name
	role.Description = a.Description
	role.Sequence = a.Sequence
	role.Status = a.Status
	return nil
}
