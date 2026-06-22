package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Route policy management
type PolicyRoute struct {
	ID          string               `json:"id" gorm:"type:char(20);primaryKey;<-:create;comment:主键ID (XID);"`
	Name        string               `json:"name" gorm:"type:varchar(128);not null;uniqueIndex:uniq_policy_route_name;comment:策略名称;"`
	Order       int                  `json:"order" gorm:"type:int;not null;default:0;comment:执行顺序，数字越小越优先;"`
	Version     int64                `json:"version" gorm:"type:bigint;not null;default:1;comment:配置版本号;"`
	Enabled     int                  `json:"enabled" gorm:"type:int;not null;default:0;comment:启用状态: 0-未启用，1-启用;"`
	Description *string              `json:"description,omitempty" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	Creator     *string              `json:"creator,omitempty" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier    *string              `json:"modifier,omitempty" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt   time.Time            `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt   time.Time            `json:"updated_at,omitempty" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted     string               `json:"-" gorm:"type:varchar(20);not null;default:'0';comment:逻辑删除标识;"`
	DeletedAt   *gorm.DeletedAt      `json:"-" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
	Details     *[]PolicyRouteDetail `json:"details,omitempty" gorm:"foreignKey:RouteId;references:ID"`
}

func (a PolicyRoute) TableName() string {
	return config.C.FormatTableName("policy_route")
}

// ConvertTo Convert `PolicyRoute` to `PolicyRouteForm` object.
func (a PolicyRoute) ConvertTo(route *PolicyRouteForm) error {
	route.ID = a.ID
	route.Name = a.Name
	route.Order = a.Order
	route.Version = a.Version
	route.Enabled = a.Enabled
	route.Description = a.Description
	route.Creator = a.Creator
	route.Modifier = a.Modifier
	route.CreatedAt = a.CreatedAt
	route.UpdatedAt = a.UpdatedAt
	if a.Details != nil {
		details := make([]PolicyRouteDetailForm, 0)
		for _, detail := range *a.Details {
			d := PolicyRouteDetailForm{}
			err := detail.ConvertTo(&d)
			if err != nil {
				return err
			}
			details = append(details, d)
		}
		route.Details = details
	}
	return nil
}

// Defining the query parameters for the `PolicyRoute` struct.
type PolicyRouteQueryParam struct {
	util.PaginationParam
	Name string `form:"name"` // Policy name (like)
}

// Defining the query options for the `PolicyRoute` struct.
type PolicyRouteQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `PolicyRoute` struct.
type PolicyRouteQueryResult struct {
	Data       PolicyRoutes
	PageResult *util.PaginationResult
}

// Defining the slice of `PolicyRoute` struct.
type PolicyRoutes []*PolicyRoute

// Defining the data structure for creating a `PolicyRoute` struct.
type PolicyRouteForm struct {
	ID          string                  `json:"id"`                              // Unique ID
	Name        string                  `json:"name" binding:"required,max=128"` // Policy name
	Order       int                     `json:"order"`                           // Sort order
	Version     int64                   `json:"version"`                         // Version
	Enabled     int                     `json:"enabled"`                         // Enabled
	Description *string                 `json:"description"`                     // Description
	Details     []PolicyRouteDetailForm `json:"details"`                         // RouteDetail
	Creator     *string                 `json:"creator,omitempty"`               // Creator
	Modifier    *string                 `json:"modifier,omitempty"`              // Modifier
	CreatedAt   time.Time               `json:"created_at"`                      // Create timestamp
	UpdatedAt   time.Time               `json:"updated_at,omitempty"`            // Update timestamp
}

// A validation function for the `PolicyRouteForm` struct.
func (a *PolicyRouteForm) Validate() error {
	if a.Name == "" {
		return errors.BadRequest("", "Name is required")
	}
	return nil
}

// Convert `PolicyRouteForm` to `PolicyRoute` object.
func (a *PolicyRouteForm) FillTo(policyRoute *PolicyRoute) error {
	policyRoute.Name = a.Name
	policyRoute.Order = a.Order
	policyRoute.Enabled = a.Enabled
	policyRoute.Description = a.Description
	policyRoute.Version = time.Now().UnixMilli()
	if util.IsNilOrEmpty(policyRoute.Creator) {
		policyRoute.Creator = a.Creator
	}
	policyRoute.Modifier = a.Modifier
	policyRoute.Details = func() *[]PolicyRouteDetail {
		var details []PolicyRouteDetail
		for _, detail := range a.Details {
			d := PolicyRouteDetail{}
			err := detail.FillTo(&d)
			if err != nil {
				return nil
			}
			if len(d.ID) == 0 {
				d.ID = util.NewXID()
			}
			details = append(details, d)
		}
		return &details
	}()
	return nil
}
