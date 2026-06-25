package schema

import (
	"strings"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/encoding/json"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

const (
	TaggingActionTypeTag       = "TAG"
	TaggingActionTypeReqHeader = "REQ_HEADER"
	TaggingActionTypeRspHeader = "RSP_HEADER"
	TaggingActionTypeReqCookie = "REQ_COOKIE"
	TaggingActionTypeRspCookie = "RSP_COOKIE"
	TaggingActionTypeReqBody   = "REQ_BODY"
)

// TaggingAction 染色打标动作
type TaggingAction struct {
	Type  string `json:"type"`                     // 动作类型/操作范围，如 TAG, REQ_HEADER, RSP_HEADER 等
	Key   string `json:"key" binding:"required"`   // 标签名/键名
	Value string `json:"value" binding:"required"` // 标签值，支持变量插值
}

// PolicyTagging 流量染色策略表
type PolicyTagging struct {
	ID          string          `json:"id" gorm:"type:char(20);primaryKey;<-:create;comment:主键ID (XID);"`
	Name        string          `json:"name" gorm:"type:varchar(128);not null;uniqueIndex:uniq_policy_tagging_name;comment:策略名称;"`
	Order       int             `json:"order" gorm:"type:int;not null;default:0;comment:执行顺序，数字越小越优先;"`
	Relation    string          `json:"relation" gorm:"type:varchar(16);not null;default:'AND';comment:多条件之间的逻辑关系：AND / OR;"`
	Conditions  *string         `json:"conditions,omitempty" gorm:"type:json;default:null;comment:匹配条件列表，嵌套 Condition 数组;"`
	Actions     *string         `json:"actions,omitempty" gorm:"type:json;default:null;comment:染色动作列表，嵌套 TaggingAction 数组;"`
	Version     int64           `json:"version" gorm:"type:bigint;not null;default:1;comment:配置版本号;"`
	Enabled     int             `json:"enabled" gorm:"type:int;not null;default:0;comment:启用状态: 0-未启用，1-启用;"`
	Description *string         `json:"description,omitempty" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	Creator     *string         `json:"creator,omitempty" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier    *string         `json:"modifier,omitempty" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt   time.Time       `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt   time.Time       `json:"updated_at,omitempty" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted     string          `json:"-" gorm:"type:varchar(20);not null;default:'0';comment:逻辑删除标识;"`
	DeletedAt   *gorm.DeletedAt `json:"-" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
}

func (a PolicyTagging) TableName() string {
	return config.C.FormatTableName("policy_tagging")
}

// ConvertTo Convert `PolicyTagging` to `PolicyTaggingForm` object.
func (a PolicyTagging) ConvertTo(form *PolicyTaggingForm) error {
	form.ID = a.ID
	form.Name = a.Name
	form.Order = a.Order
	form.Relation = a.Relation
	if !util.IsNilOrEmpty(a.Conditions) {
		conditions := make([]TagCondition, 0)
		json.UnMarshalToObject(*a.Conditions, &conditions)
		form.Conditions = &conditions
	}
	if !util.IsNilOrEmpty(a.Actions) {
		actions := make([]TaggingAction, 0)
		json.UnMarshalToObject(*a.Actions, &actions)
		form.Actions = &actions
	}
	form.Version = a.Version
	form.Enabled = a.Enabled
	form.Description = a.Description
	form.Creator = a.Creator
	form.Modifier = a.Modifier
	form.CreatedAt = a.CreatedAt
	form.UpdatedAt = a.UpdatedAt
	return nil
}

// Defining the query parameters for the `PolicyTagging` struct.
type PolicyTaggingQueryParam struct {
	util.PaginationParam
	Name string `form:"name"` // Policy name (like)
}

// Defining the query options for the `PolicyTagging` struct.
type PolicyTaggingQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `PolicyTagging` struct.
type PolicyTaggingQueryResult struct {
	Data       PolicyTaggings
	PageResult *util.PaginationResult
}

// Defining the slice of `PolicyTagging` struct.
type PolicyTaggings []*PolicyTagging

// Defining the data structure for creating or updating a `PolicyTagging` struct.
type PolicyTaggingForm struct {
	ID          string           `json:"id"`
	Name        string           `json:"name" binding:"required,max=128"`          // Policy name
	Order       int              `json:"order"`                                    // Execution order
	Relation    string           `json:"relation" binding:"required,oneof=AND OR"` // Relation type
	Conditions  *[]TagCondition  `json:"conditions"`                               // Match conditions
	Actions     *[]TaggingAction `json:"actions"`                                  // Tagging actions
	Version     int64            `json:"version"`                                  // Version
	Enabled     int              `json:"enabled" binding:"oneof=0 1"`              // Enabled
	Description *string          `json:"description"`                              // Details
	Creator     *string          `json:"creator,omitempty"`                        // Creator
	Modifier    *string          `json:"modifier,omitempty"`                       // Modifier
	CreatedAt   time.Time        `json:"created_at"`                               // Create timestamp
	UpdatedAt   time.Time        `json:"updated_at,omitempty"`                     // Update timestamp
}

// A validation function for the `PolicyTaggingForm` struct.
func (a *PolicyTaggingForm) Validate() error {
	if a.Name == "" {
		return errors.BadRequest("", "Name is required")
	}
	if a.Relation == "" {
		return errors.BadRequest("", "Relation is required")
	}
	if a.Actions != nil {
		for _, action := range *a.Actions {
			if action.Key == "" {
				return errors.BadRequest("", "Action key is required")
			}
			if action.Value == "" {
				return errors.BadRequest("", "Action value is required")
			}
			if action.Type != "" {
				t := strings.ToUpper(action.Type)
				if t != TaggingActionTypeTag &&
					t != TaggingActionTypeReqHeader &&
					t != TaggingActionTypeRspHeader &&
					t != TaggingActionTypeReqCookie &&
					t != TaggingActionTypeRspCookie &&
					t != TaggingActionTypeReqBody {
					return errors.BadRequest("", "Invalid action type: %s", action.Type)
				}
			}
		}
	}
	return nil
}

// Convert `PolicyTaggingForm` to `PolicyTagging` object.
func (a *PolicyTaggingForm) FillTo(policyTagging *PolicyTagging) error {
	policyTagging.Name = a.Name
	policyTagging.Order = a.Order
	policyTagging.Relation = a.Relation
	policyTagging.Conditions = func() *string {
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
	policyTagging.Actions = func() *string {
		if a.Actions == nil {
			return nil
		}
		normalizedActions := make([]TaggingAction, len(*a.Actions))
		for i, act := range *a.Actions {
			t := strings.ToUpper(act.Type)
			if t == "" {
				t = TaggingActionTypeTag
			}
			normalizedActions[i] = TaggingAction{
				Type:  t,
				Key:   act.Key,
				Value: act.Value,
			}
		}
		return json.MarshalToString(&normalizedActions)
	}()
	policyTagging.Enabled = a.Enabled
	policyTagging.Description = a.Description
	policyTagging.Version = time.Now().UnixMilli()
	return nil
}
