package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/encoding/json"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// TaggingAction 染色打标动作
type TaggingAction struct {
	Key   string `json:"key" binding:"required"`   // 标签名
	Value string `json:"value" binding:"required"` // 标签值，支持变量插值
}

// PolicyTagging 流量染色策略表
type PolicyTagging struct {
	ID          string          `json:"id" gorm:"size:20;primaryKey;<-:create;comment:Unique ID;"`
	Name        string          `json:"name" gorm:"size:128;not null;uniqueIndex:uniq_policy_tagging_name;comment:Policy name;"`
	Order       int             `json:"order" gorm:"not null;default:0;comment:Execution order (smaller is higher priority);"`
	Relation    string          `json:"relation" gorm:"size:16;not null;default:AND;comment:Relation type: AND/OR;"`
	Conditions  *string         `json:"conditions,omitempty" gorm:"type:json;comment:Match conditions (JSON);"`
	Actions     *string         `json:"actions,omitempty" gorm:"type:json;comment:Tagging actions (JSON);"`
	Version     int64           `json:"version" gorm:"not null;default:1;comment:Version;"`
	Enabled     int             `json:"enabled" gorm:"not null;default:0;comment:Enabled;"`
	Description *string         `json:"description,omitempty" gorm:"size:255;comment:Details;"`
	Creator     *string         `json:"creator,omitempty" gorm:"size:255;comment:Creator;"`
	Modifier    *string         `json:"modifier,omitempty" gorm:"size:255;comment:Modifier;"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime;comment:Create timestamp;"`
	UpdatedAt   time.Time       `json:"updated_at,omitempty" gorm:"autoUpdateTime;comment:Update timestamp;"`
	Deleted     string          `json:"-" gorm:"uniqueIndex:uniq_policy_tagging_name;size:20;default:0;comment:Delete flag;"`
	DeletedAt   *gorm.DeletedAt `json:"-" gorm:"comment:Delete timestamp;"`
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
	policyTagging.Actions = func() *string { return json.MarshalToString(a.Actions) }()
	policyTagging.Enabled = a.Enabled
	policyTagging.Description = a.Description
	policyTagging.Version = time.Now().UnixMilli()
	return nil
}
