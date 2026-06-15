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
	ID             string          `json:"id" gorm:"size:20;primaryKey;<-:create;comment:Unique ID;"`                             // Unique ID
	Name           string          `json:"name" gorm:"size:128;not null;uniqueIndex:uniq_policy_limit_name;comment:Policy name;"` // Policy name
	Version        int64           `json:"version" gorm:"not null;default:1;comment:Version;"`                                    // Version
	Type           string          `json:"type" gorm:"size:64;not null;comment:Limit dimension: request / token / cost;"`         // Limit dimension
	MaxWaitMs      int             `json:"max_wait_ms" gorm:"not null;default:0;comment:Max queue wait time (ms);"`               // Max queue wait time
	RelationType   string          `json:"relation_type" gorm:"size:16;not null;default:AND;comment:Relation type: AND / OR;"`    // Relation type
	SlidingWindows *string         `json:"sliding_windows,omitempty" gorm:"type:json;comment:Sliding windows (JSON);"`            // Sliding windows (JSON)
	Conditions     *string         `json:"conditions,omitempty" gorm:"type:json;comment:Match conditions (JSON);"`                // Match conditions (JSON)
	Estimator      *string         `json:"estimator,omitempty" gorm:"type:json;comment:Estimator config (JSON);"`                 // Estimator config (JSON)
	Enabled        int             `json:"enabled" gorm:"not null;default:0;comment:Enabled;"`                                    // Enabled
	Description    *string         `json:"description,omitempty" gorm:"size:255;comment:Details;"`                                // Details
	Creator        *string         `json:"creator,omitempty" gorm:"size:255;comment:Creator;"`                                    // Creator
	Modifier       *string         `json:"modifier,omitempty" gorm:"size:255;comment:Modifier;"`                                  // Modifier
	CreatedAt      time.Time       `json:"created_at" gorm:"autoCreateTime;comment:Create timestamp;"`                            // Create timestamp
	UpdatedAt      time.Time       `json:"updated_at,omitempty" gorm:"autoUpdateTime;comment:Update timestamp;"`                  // Update timestamp
	Deleted        string          `json:"-" gorm:"uniqueIndex:uniq_policy_limit_name;size:20;default:0;comment:Delete flag;"`    // Delete flag
	DeletedAt      *gorm.DeletedAt `json:"-" gorm:"comment:Delete timestamp;"`                                                    // Delete timestamp
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
