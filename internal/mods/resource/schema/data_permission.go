package schema

import (
	"fmt"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Data permission type constants
const (
	DataPermissionTypeProvider = "provider"
	DataPermissionTypeModel    = "model"
)

var dataPermissionTypes = map[string]bool{
	DataPermissionTypeProvider: true,
	DataPermissionTypeModel:    true,
}

// Data permission management
type DataPermission struct {
	ID         string          `json:"id" gorm:"type:varchar(20);primaryKey;<-:create;comment:ID;"`
	Type       string          `json:"type" gorm:"type:varchar(50);not null;uniqueIndex:uniq_data_permission;index:idx_type;comment:数据类型(表名);"`
	DataId     string          `json:"data_id" gorm:"type:varchar(20);not null;uniqueIndex:uniq_data_permission;index:idx_data_id;comment:数据ID;"`
	User       string          `json:"user" gorm:"type:varchar(50);not null;uniqueIndex:uniq_data_permission;index:idx_user;comment:用户;"`
	Tenant     string          `json:"tenant" gorm:"type:varchar(50);not null;uniqueIndex:uniq_data_permission;comment:租户;"`
	Role       string          `json:"role" gorm:"type:varchar(20);not null;uniqueIndex:uniq_data_permission;comment:角色编码;"`
	Permission uint            `json:"permission" gorm:"type:int unsigned;not null;default:0;comment:数据权限位 - 格式(read,write,delete);"`
	Creator    string          `json:"creator" gorm:"type:varchar(255);default:null;comment:创建者;"`
	CreatedAt  time.Time       `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt  time.Time       `json:"updated_at,omitempty" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted    string          `json:"-" gorm:"type:varchar(20);not null;default:'0';uniqueIndex:uniq_data_permission;comment:逻辑删除标识;"`
	DeletedAt  *gorm.DeletedAt `json:"-" gorm:"type:datetime;default:null;comment:删除时间;"`
}

func (a DataPermission) TableName() string {
	return config.C.FormatTableName("data_permission")
}

// Defining the query parameters for the `DataPermission` struct.
type DataPermissionQueryParam struct {
	util.PaginationParam
	Type   string `form:"type"`    // Data type (application/service)
	DataId string `form:"data_id"` // Data ID
	User   string `form:"user"`    // User
}

// Defining the query options for the `DataPermission` struct.
type DataPermissionQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `DataPermission` struct.
type DataPermissionQueryResult struct {
	Data       DataPermissions
	PageResult *util.PaginationResult
}

// Defining the slice of `DataPermission` struct.
type DataPermissions []*DataPermission

// Defining the data structure for creating a `DataPermission` struct.
type DataPermissionForm struct {
	Type       string `json:"type" binding:"required,max=50"`    // Data type (table name)
	DataId     string `json:"data_id" binding:"required,max=20"` // Data ID
	User       string `json:"user" binding:"required,max=50"`    // User
	Tenant     string `json:"tenant" binding:"required,max=50"`  // Tenant
	Role       string `json:"role" binding:"required,max=20"`    // Role code
	Permission uint   `json:"permission"`                        // Data permission bits - format(read,write,delete)
}

// A validation function for the `DataPermissionForm` struct.
func (a *DataPermissionForm) Validate() error {
	if !dataPermissionTypes[a.Type] {
		return fmt.Errorf("invalid data permission type: %s, must be one of: application, service", a.Type)
	}
	return nil
}

// Convert `DataPermissionForm` to `DataPermission` object.
func (a *DataPermissionForm) FillTo(dataPermission *DataPermission) error {
	dataPermission.Type = a.Type
	dataPermission.DataId = a.DataId
	dataPermission.User = a.User
	dataPermission.Tenant = a.Tenant
	dataPermission.Role = a.Role
	dataPermission.Permission = a.Permission
	return nil
}
