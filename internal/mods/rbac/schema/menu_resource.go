package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Menu resource management for RBAC
type MenuResource struct {
	ID        string    `json:"id" gorm:"type:varchar(20);primaryKey;comment:ID;"`
	MenuID    string    `json:"menu_id" gorm:"type:varchar(20);default:null;index:idx_menu_resource_menu_id;comment:菜单ID;"`
	Method    string    `json:"method" gorm:"type:varchar(20);default:null;comment:请求方法;"`
	Path      string    `json:"path" gorm:"type:varchar(255);default:null;comment:请求路径;"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime(3);default:null;autoCreateTime;comment:创建时间;"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime(3);default:null;autoUpdateTime;comment:更新时间;"`
}

func (a *MenuResource) TableName() string {
	return config.C.FormatTableName("menu_resource")
}

// Defining the query parameters for the `MenuResource` struct.
type MenuResourceQueryParam struct {
	util.PaginationParam
	MenuID  string   `form:"-"` // From Menu.ID
	MenuIDs []string `form:"-"` // From Menu.ID
}

// Defining the query options for the `MenuResource` struct.
type MenuResourceQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `MenuResource` struct.
type MenuResourceQueryResult struct {
	Data       MenuResources
	PageResult *util.PaginationResult
}

// Defining the slice of `MenuResource` struct.
type MenuResources []*MenuResource

// Defining the data structure for creating a `MenuResource` struct.
type MenuResourceForm struct {
}

// A validation function for the `MenuResourceForm` struct.
func (a *MenuResourceForm) Validate() error {
	return nil
}

func (a *MenuResourceForm) FillTo(menuResource *MenuResource) error {
	return nil
}
