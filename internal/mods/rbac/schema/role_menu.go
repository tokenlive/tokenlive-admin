package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Role permissions for RBAC
type RoleMenu struct {
	ID        string    `json:"id" gorm:"type:varchar(20);primaryKey;comment:ID;"`
	RoleID    string    `json:"role_id" gorm:"type:varchar(20);default:null;index:idx_role_menu_role_id;comment:角色ID;"`
	MenuID    string    `json:"menu_id" gorm:"type:varchar(20);default:null;index:idx_role_menu_menu_id;comment:菜单ID;"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime(3);default:null;autoCreateTime;comment:创建时间;"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime(3);default:null;autoUpdateTime;comment:更新时间;"`
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
