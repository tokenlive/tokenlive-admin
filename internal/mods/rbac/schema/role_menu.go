package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Role permissions for RBAC
type RoleMenu struct {
	ID        string    `json:"id" gorm:"size:20;primarykey"`                        // Unique ID
	RoleID    string    `json:"role_id" gorm:"size:20;index"`                        // From Role.ID
	MenuID    string    `json:"menu_group_id" gorm:"column:menu_group_id;size:20;index"` // From Menu.ID (column: menu_group_id)
	CreatedAt time.Time `json:"created_at" gorm:"type:timestamp;autoCreateTime;"` // Create time
	UpdatedAt time.Time `json:"updated_at" gorm:"type:timestamp;autoUpdateTime;"` // Update time
}

func (a *RoleMenu) TableName() string {
	return config.C.FormatTableName("role_menu")
}

// Defining the query parameters for the `RoleMenu` struct.
type RoleMenuQueryParam struct {
	util.PaginationParam
	RoleID string `form:"-"` // From Role.ID
}

// Defining the query options for the `RoleMenu` struct.
type RoleMenuQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `RoleMenu` struct.
type RoleMenuQueryResult struct {
	Data       RoleMenus
	PageResult *util.PaginationResult
}

// Defining the slice of `RoleMenu` struct.
type RoleMenus []*RoleMenu

// Defining the data structure for creating a `RoleMenu` struct.
type RoleMenuForm struct {
}

// A validation function for the `RoleMenuForm` struct.
func (a *RoleMenuForm) Validate() error {
	return nil
}

func (a *RoleMenuForm) FillTo(roleMenu *RoleMenu) error {
	return nil
}
