package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// User roles for RBAC
type UserRole struct {
	ID        string    `json:"id" gorm:"type:varchar(20);primaryKey;comment:ID;"`
	UserID    string    `json:"user_id" gorm:"type:varchar(20);default:null;index:idx_user_role_user_id;comment:用户ID;"`
	RoleID    string    `json:"role_id" gorm:"type:varchar(20);default:null;index:idx_user_role_role_id;comment:角色ID;"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime(3);default:null;autoCreateTime;comment:创建时间;"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime(3);default:null;autoUpdateTime;comment:更新时间;"`
	RoleName  string    `json:"role_name" gorm:"->;-:migration"`                  // From Role.Name
}

func (a *UserRole) TableName() string {
	return config.C.FormatTableName("user_role")
}

// Defining the query parameters for the `UserRole` struct.
type UserRoleQueryParam struct {
	util.PaginationParam
	InUserIDs []string `form:"-"` // From User.ID
	UserID    string   `form:"-"` // From User.ID
	RoleID    string   `form:"-"` // From Role.ID
}

// Defining the query options for the `UserRole` struct.
type UserRoleQueryOptions struct {
	util.QueryOptions
	JoinRole bool // Join role table
}

// Defining the query result for the `UserRole` struct.
type UserRoleQueryResult struct {
	Data       UserRoles
	PageResult *util.PaginationResult
}

// Defining the slice of `UserRole` struct.
type UserRoles []*UserRole

func (a UserRoles) ToUserIDMap() map[string]UserRoles {
	m := make(map[string]UserRoles)
	for _, userRole := range a {
		m[userRole.UserID] = append(m[userRole.UserID], userRole)
	}
	return m
}

func (a UserRoles) ToRoleIDs() []string {
	var ids []string
	for _, item := range a {
		ids = append(ids, item.RoleID)
	}
	return ids
}

// Defining the data structure for creating a `UserRole` struct.
type UserRoleForm struct {
}

// A validation function for the `UserRoleForm` struct.
func (a *UserRoleForm) Validate() error {
	return nil
}

func (a *UserRoleForm) FillTo(userRole *UserRole) error {
	return nil
}
