package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/encoding/json"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Estimator config for Token limit
type Estimator struct {
	Type  string  `json:"type"`  // length_ratio, tiktoken
	Ratio float64 `json:"ratio"` // Character to token ratio
}

// Limit policy management
type PolicyLimit struct {
	ID             string          `json:"id" gorm:"type:char(20);primaryKey;<-:create;comment:主键ID (XID);"`
	Name           string          `json:"name" gorm:"type:varchar(128);not null;uniqueIndex:uniq_policy_limit_name;comment:策略名称;"`
	Version        int64           `json:"version" gorm:"type:bigint;not null;default:1;comment:配置版本号;"`
	Type           string          `json:"type" gorm:"type:varchar(64);not null;comment:限流维度：request / token / cost;"`
	MaxWaitMs      int             `json:"max_wait_ms" gorm:"type:int;not null;default:0;comment:排队等待最大时间（毫秒）;"`
	RelationType   string          `json:"relation_type" gorm:"type:varchar(16);not null;default:'AND';comment:多条件之间的逻辑关系：AND / OR;"`
	SlidingWindows *string         `json:"sliding_windows,omitempty" gorm:"type:json;default:null;comment:滑动窗口配额配置列表，嵌套 SlidingWindow 数组;"`
	Conditions     *string         `json:"conditions,omitempty" gorm:"type:json;default:null;comment:匹配条件列表，嵌套 Condition 数组;"`
	Estimator      *string         `json:"estimator,omitempty" gorm:"type:json;default:null;comment:估算器配置，包含 type 和 ratio;"`
	Enabled        int             `json:"enabled" gorm:"type:int;not null;default:0;comment:启用状态: 0-未启用，1-启用;"`
	Description    *string         `json:"description,omitempty" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	Creator        *string         `json:"creator,omitempty" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier       *string         `json:"modifier,omitempty" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt      time.Time       `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt      time.Time       `json:"updated_at,omitempty" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted        string          `json:"-" gorm:"type:varchar(20);not null;default:'0';comment:逻辑删除标识;"`
	DeletedAt      *gorm.DeletedAt `json:"-" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
}

func (a PolicyLimit) TableName() string {
	return config.C.FormatTableName("policy_limit")
}

// ConvertTo Convert `PolicyLimit` to `PolicyLimitForm` object.
func (a PolicyLimit) ConvertTo(limit *PolicyLimitForm) error {
	limit.ID = a.ID
	limit.Name = a.Name
	limit.Version = a.Version
	limit.Type = a.Type
	limit.MaxWaitMs = a.MaxWaitMs
	limit.RelationType = a.RelationType
	if !util.IsNilOrEmpty(a.SlidingWindows) {
		sw := make([]SlidingWindow, 0)
		json.UnMarshalToObject(*a.SlidingWindows, &sw)
		limit.SlidingWindows = &sw
	}
	if !util.IsNilOrEmpty(a.Conditions) {
		conditions := make([]TagCondition, 0)
		json.UnMarshalToObject(*a.Conditions, &conditions)
		limit.Conditions = &conditions
	}
	if !util.IsNilOrEmpty(a.Estimator) {
		est := new(Estimator)
		json.UnMarshalToObject(*a.Estimator, est)
		limit.Estimator = est
	}
	limit.Enabled = a.Enabled
	limit.Description = a.Description
	limit.Creator = a.Creator
	limit.Modifier = a.Modifier
	limit.CreatedAt = a.CreatedAt
	limit.UpdatedAt = a.UpdatedAt
	return nil
}

// Defining the query parameters for the `PolicyLimit` struct.
type PolicyLimitQueryParam struct {
	util.PaginationParam
	Name string `form:"name"` // Policy name (like)
}

// Defining the query options for the `PolicyLimit` struct.
type PolicyLimitQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `PolicyLimit` struct.
type PolicyLimitQueryResult struct {
	Data       PolicyLimits
	PageResult *util.PaginationResult
}

// Defining the slice of `PolicyLimit` struct.
type PolicyLimits []*PolicyLimit

// Defining the data structure for creating a `PolicyLimit` struct.
type PolicyLimitForm struct {
	ID             string           `json:"id"`
	Name           string           `json:"name" binding:"required,max=128"`         // Policy name
	Version        int64            `json:"version"`                                 // Version
	Type           string           `json:"type" binding:"required,max=64"`          // Limit dimension: request / token / cost
	MaxWaitMs      int              `json:"max_wait_ms"`                             // Max queue wait time (ms)
	RelationType   string           `json:"relation_type" binding:"required,max=16"` // Relation type: AND / OR
	SlidingWindows *[]SlidingWindow `json:"sliding_windows"`                         // Sliding windows (JSON)
	Conditions     *[]TagCondition  `json:"conditions"`                              // Match conditions (JSON)
	Estimator      *Estimator       `json:"estimator"`                               // Estimator config (JSON)
	Enabled        int              `json:"enabled"`                                 // Enabled
	Description    *string          `json:"description"`                             // Details
	Creator        *string          `json:"creator,omitempty"`                       // Creator
	Modifier       *string          `json:"modifier,omitempty"`                      // Modifier
	CreatedAt      time.Time        `json:"created_at"`                              // Create timestamp
	UpdatedAt      time.Time        `json:"updated_at,omitempty"`                    // Update timestamp
}

// A validation function for the `PolicyLimitForm` struct.
func (a *PolicyLimitForm) Validate() error {
	if a.Name == "" {
		return errors.BadRequest("", "Name is required")
	}
	if a.Type == "" {
		return errors.BadRequest("", "Type is required")
	}
	if a.RelationType == "" {
		return errors.BadRequest("", "RelationType is required")
	}
	return nil
}

// Convert `PolicyLimitForm` to `PolicyLimit` object.
func (a *PolicyLimitForm) FillTo(policyLimit *PolicyLimit) error {
	policyLimit.Name = a.Name
	policyLimit.Type = a.Type
	policyLimit.MaxWaitMs = a.MaxWaitMs
	policyLimit.RelationType = a.RelationType
	policyLimit.SlidingWindows = func() *string { return json.MarshalToString(a.SlidingWindows) }()
	policyLimit.Conditions = func() *string {
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
	policyLimit.Estimator = func() *string { return json.MarshalToString(a.Estimator) }()
	policyLimit.Enabled = a.Enabled
	policyLimit.Description = a.Description
	policyLimit.Version = time.Now().UnixMilli()
	return nil
}
