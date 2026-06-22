package schema

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

const (
	MenuStatusDisabled = "disabled"
	MenuStatusEnabled  = "enabled"
)

var (
	MenusOrderParams = []util.OrderByParam{
		{Field: "sequence", Direction: util.DESC},
		{Field: "created_at", Direction: util.DESC},
	}
)

// Menu management for RBAC
type Menu struct {
	ID          string        `json:"id" gorm:"type:varchar(20);primaryKey;comment:ID;"`
	Code        string        `json:"code" gorm:"type:varchar(32);default:null;index:idx_menu_code;comment:菜单编码;"`
	Name        string        `json:"name" gorm:"type:varchar(128);default:null;index:idx_menu_name;comment:菜单名称;"`
	Description string        `json:"description" gorm:"type:varchar(1024);default:null;comment:描述;"`
	Sequence    int           `json:"sequence" gorm:"type:bigint;default:null;index:idx_menu_sequence;comment:序列;"`
	Type        string        `json:"type" gorm:"type:varchar(20);default:null;index:idx_menu_type;comment:类型: page, button;"`
	Path        string        `json:"path" gorm:"type:varchar(255);default:null;comment:路径;"`
	Properties  string        `json:"properties" gorm:"type:text;comment:属性;"`
	Status      string        `json:"status" gorm:"type:varchar(20);default:null;index:idx_menu_status;comment:状态;"`
	ParentID    string        `json:"parent_id" gorm:"type:varchar(20);default:null;index:idx_menu_parent_id;comment:父ID;"`
	ParentPath  string        `json:"parent_path" gorm:"type:varchar(255);default:null;index:idx_menu_parent_path;comment:父路径;"`
	Children    *Menus        `json:"children" gorm:"-"`                                 // Child menus
	CreatedAt   time.Time     `json:"created_at" gorm:"type:datetime(3);default:null;autoCreateTime;comment:创建时间;"`
	UpdatedAt   time.Time     `json:"updated_at" gorm:"type:datetime(3);default:null;autoUpdateTime;comment:更新时间;"`
	Resources   MenuResources `json:"resources" gorm:"-"`                                // Resources of menu
}

func (a *Menu) TableName() string {
	return config.C.FormatTableName("menu")
}

// Defining the query parameters for the `Menu` struct.
type MenuQueryParam struct {
	util.PaginationParam
	CodePath         string   `form:"code"`             // Code path (like xxx.xxx.xxx)
	LikeName         string   `form:"name"`             // Display name of menu
	IncludeResources bool     `form:"includeResources"` // Include resources
	InIDs            []string `form:"-"`                // Include menu IDs
	Status           string   `form:"-"`                // Status of menu (disabled, enabled)
	ParentID         string   `form:"-"`                // Parent ID (From Menu.ID)
	ParentPathPrefix string   `form:"-"`                // Parent path (split by .)
	UserID           string   `form:"-"`                // User ID
	RoleID           string   `form:"-"`                // Role ID
}

// Defining the query options for the `Menu` struct.
type MenuQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `Menu` struct.
type MenuQueryResult struct {
	Data       Menus
	PageResult *util.PaginationResult
}

// Defining the slice of `Menu` struct.
type Menus []*Menu

func (a Menus) Len() int {
	return len(a)
}

func (a Menus) Less(i, j int) bool {
	if a[i].Sequence == a[j].Sequence {
		return a[i].CreatedAt.Unix() > a[j].CreatedAt.Unix()
	}
	return a[i].Sequence > a[j].Sequence
}

func (a Menus) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a Menus) ToMap() map[string]*Menu {
	m := make(map[string]*Menu)
	for _, item := range a {
		m[item.ID] = item
	}
	return m
}

func (a Menus) SplitParentIDs() []string {
	parentIDs := make([]string, 0, len(a))
	idMapper := make(map[string]struct{})
	for _, item := range a {
		if _, ok := idMapper[item.ID]; ok {
			continue
		}
		idMapper[item.ID] = struct{}{}
		if pp := item.ParentPath; pp != "" {
			for _, pid := range strings.Split(pp, util.TreePathDelimiter) {
				if pid == "" {
					continue
				}
				if _, ok := idMapper[pid]; ok {
					continue
				}
				parentIDs = append(parentIDs, pid)
				idMapper[pid] = struct{}{}
			}
		}
	}
	return parentIDs
}

func (a Menus) ToTree() Menus {
	var list Menus
	m := a.ToMap()
	for _, item := range a {
		if item.ParentID == "" {
			list = append(list, item)
			continue
		}
		if parent, ok := m[item.ParentID]; ok {
			if parent.Children == nil {
				children := Menus{item}
				parent.Children = &children
				continue
			}
			*parent.Children = append(*parent.Children, item)
		}
	}
	return list
}

// Defining the data structure for creating a `Menu` struct.
type MenuForm struct {
	Code        string        `json:"code" binding:"required,max=32"`                   // Code of menu (unique for each level)
	Name        string        `json:"name" binding:"required,max=128"`                  // Display name of menu
	Description string        `json:"description"`                                      // Details about menu
	Sequence    int           `json:"sequence"`                                         // Sequence for sorting (Order by desc)
	Type        string        `json:"type" binding:"required,oneof=page button"`        // Type of menu (page, button)
	Path        string        `json:"path"`                                             // Access path of menu
	Properties  string        `json:"properties"`                                       // Properties of menu (JSON)
	Status      string        `json:"status" binding:"required,oneof=disabled enabled"` // Status of menu (enabled, disabled)
	ParentID    string        `json:"parent_id"`                                        // Parent ID (From Menu.ID)
	Resources   MenuResources `json:"resources"`                                        // Resources of menu
}

// A validation function for the `MenuForm` struct.
func (a *MenuForm) Validate() error {
	if v := a.Properties; v != "" {
		if !json.Valid([]byte(v)) {
			return errors.BadRequest("", "invalid properties")
		}
	}
	return nil
}

func (a *MenuForm) FillTo(menu *Menu) error {
	menu.Code = a.Code
	menu.Name = a.Name
	menu.Description = a.Description
	menu.Sequence = a.Sequence
	menu.Type = a.Type
	menu.Path = a.Path
	menu.Properties = a.Properties
	menu.Status = a.Status
	menu.ParentID = a.ParentID
	return nil
}
