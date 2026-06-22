package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/encoding/json"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Route policy detail management
type PolicyRouteDetail struct {
	ID           string          `json:"id" gorm:"type:varchar(20);primaryKey;<-:create;comment:ID;"`
	RouteId      string          `json:"route_id" gorm:"type:varchar(20);not null;index:idx_routeid,priority:1;comment:路由ID;"`
	RelationType string          `json:"relation_type" gorm:"type:varchar(20);not null;comment:关系类型;"`
	Conditions   *string         `json:"conditions,omitempty" gorm:"type:json;default:null;comment:匹配条件;"`
	Destinations *string         `json:"destinations,omitempty" gorm:"type:json;default:null;comment:目的规则;"`
	Order        int             `json:"order" gorm:"type:int;not null;default:0;comment:排序值;"`
	Enabled      int             `json:"enabled" gorm:"type:int;not null;default:0;comment:启用;"`
	Description  *string         `json:"description,omitempty" gorm:"type:varchar(255);default:null;comment:备注;"`
	CreatedAt    time.Time       `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt    time.Time       `json:"updated_at,omitempty" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted      string          `json:"-" gorm:"type:varchar(20);not null;default:'0';index:idx_routeid,priority:2;comment:逻辑删除标识;"`
	DeletedAt    *gorm.DeletedAt `json:"-" gorm:"type:datetime;default:null;comment:删除时间;"`
}

func (a PolicyRouteDetail) TableName() string {
	return config.C.FormatTableName("policy_route_detail")
}

// ConvertTo Convert `PolicyRouteDetail` to `PolicyRouteDetailForm` object.
func (a PolicyRouteDetail) ConvertTo(detail *PolicyRouteDetailForm) error {
	if len(a.ID) > 0 {
		detail.ID = a.ID
	}
	detail.RouteId = a.RouteId
	detail.RelationType = a.RelationType
	conditions := make([]TagCondition, 0)
	if !util.IsNilOrEmpty(a.Conditions) {
		json.UnMarshalToObject(*a.Conditions, &conditions)
	}
	detail.Conditions = &conditions
	destinations := make([]TagDestination, 0)
	if !util.IsNilOrEmpty(a.Destinations) {
		json.UnMarshalToObject(*a.Destinations, &destinations)
	}
	detail.Destinations = &destinations
	detail.Order = a.Order
	detail.Enabled = a.Enabled
	detail.Description = a.Description
	detail.CreatedAt = a.CreatedAt
	detail.UpdatedAt = a.UpdatedAt
	return nil
}

// Defining the query parameters for the `PolicyRouteDetail` struct.
type PolicyRouteDetailQueryParam struct {
	util.PaginationParam
	RouteId  string   `form:"route_id"`  // Route ID
	RouteIds []string `form:"route_ids"` // Multi route id
}

// Defining the query options for the `PolicyRouteDetail` struct.
type PolicyRouteDetailQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `PolicyRouteDetail` struct.
type PolicyRouteDetailQueryResult struct {
	Data       PolicyRouteDetails
	PageResult *util.PaginationResult
}

// Defining the slice of `PolicyRouteDetail` struct.
type PolicyRouteDetails []*PolicyRouteDetail

// Defining the data structure for creating a `PolicyRouteDetail` struct.
type PolicyRouteDetailForm struct {
	ID           string            `json:"id,omitempty"`
	RouteId      string            `json:"route_id" binding:"required,max=20"`      // Route ID
	RelationType string            `json:"relation_type" binding:"required,max=20"` // Relation type
	Conditions   *[]TagCondition   `json:"conditions"`                              // Match conditions
	Destinations *[]TagDestination `json:"destinations"`                            // Destination rules
	Order        int               `json:"order"`                                   // Sort order
	Enabled      int               `json:"enabled"`                                 // Enabled
	Description  *string           `json:"description"`                             // Details
	CreatedAt    time.Time         `json:"created_at"`                              // Create timestamp
	UpdatedAt    time.Time         `json:"updated_at,omitempty"`                    // Update timestamp
}

// A validation function for the `PolicyRouteDetailForm` struct.
func (a *PolicyRouteDetailForm) Validate() error {
	if a.RouteId == "" {
		return errors.BadRequest("", "RouteId is required")
	}
	if a.RelationType == "" {
		return errors.BadRequest("", "RelationType is required")
	}
	return nil
}

// FillTo Convert `PolicyRouteDetailForm` to `PolicyRouteDetail` object.
func (a *PolicyRouteDetailForm) FillTo(policyRouteDetail *PolicyRouteDetail) error {
	if len(a.ID) > 0 {
		policyRouteDetail.ID = a.ID
	}
	policyRouteDetail.RouteId = a.RouteId
	policyRouteDetail.RelationType = a.RelationType
	policyRouteDetail.Conditions = func() *string {
		if a.Conditions == nil {
			return nil
		}
		var validConds []TagCondition
		for _, cond := range *a.Conditions {
			if cond.Type != "" {
				validConds = append(validConds, cond)
			}
		}
		if len(validConds) == 0 {
			return nil
		}
		return json.MarshalToString(&validConds)
	}()
	policyRouteDetail.Destinations = func() *string { return json.MarshalToString(a.Destinations) }()
	policyRouteDetail.Order = a.Order
	policyRouteDetail.Enabled = a.Enabled
	policyRouteDetail.Description = a.Description
	return nil
}
