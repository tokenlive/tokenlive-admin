package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/encoding/json"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Invocation policy management
type PolicyInvocation struct {
	ID             string          `json:"id" gorm:"size:20;primaryKey;<-:create;comment:Unique ID;"`                                  // Unique ID
	Name           string          `json:"name" gorm:"size:128;not null;uniqueIndex:uniq_policy_invocation_name;comment:Policy name;"` // Policy name
	Type           string          `json:"type" gorm:"size:64;not null;default:failover;comment:Invocation type (failfast | failover);"`                // Invocation type (failfast | failover)
	RetryPolicy    *string         `json:"retry_policy,omitempty" gorm:"type:json;comment:Retry policy (JSON);"`                       // Retry policy (JSON)
	FallbackPolicy *string         `json:"fallback_policy,omitempty" gorm:"type:json;comment:Fallback policy (JSON);"`                 // Fallback policy (JSON)
	Version        int64           `json:"version" gorm:"not null;default:1;comment:Version;"`                                         // Version
	Enabled        int             `json:"enabled" gorm:"not null;default:0;comment:Enabled;"`                                         // Enabled
	Description    *string         `json:"description,omitempty" gorm:"size:255;comment:Details;"`                                     // Details
	Creator        *string         `json:"creator,omitempty" gorm:"size:255;comment:Creator;"`                                         // Creator
	Modifier       *string         `json:"modifier,omitempty" gorm:"size:255;comment:Modifier;"`                                       // Modifier
	CreatedAt      time.Time       `json:"created_at" gorm:"autoCreateTime;comment:Create timestamp;"`                                 // Create timestamp
	UpdatedAt      time.Time       `json:"updated_at,omitempty" gorm:"autoUpdateTime;comment:Update timestamp;"`                       // Update timestamp
	Deleted        string          `json:"-" gorm:"size:20;default:0;comment:Delete flag;"`    // Delete flag
	DeletedAt      *gorm.DeletedAt `json:"-" gorm:"type:datetime;comment:Delete timestamp;"`   // Delete timestamp
}

func (a PolicyInvocation) TableName() string {
	return config.C.FormatTableName("policy_invocation")
}

// ConvertTo Convert `PolicyInvocation` to `PolicyInvocationForm` object.
func (a PolicyInvocation) ConvertTo(form *PolicyInvocationForm) error {
	form.ID = a.ID
	form.Name = a.Name
	form.Type = a.Type
	if !util.IsNilOrEmpty(a.RetryPolicy) {
		rp := new(RetryPolicy)
		json.UnMarshalToObject(*a.RetryPolicy, rp)
		form.RetryPolicy = rp
	}
	if !util.IsNilOrEmpty(a.FallbackPolicy) {
		fp := new(FallbackPolicy)
		json.UnMarshalToObject(*a.FallbackPolicy, fp)
		form.FallbackPolicy = fp
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

// Defining the query parameters for the `PolicyInvocation` struct.
type PolicyInvocationQueryParam struct {
	util.PaginationParam
	Name string `form:"name"` // Policy name (like)
}

// Defining the query options for the `PolicyInvocation` struct.
type PolicyInvocationQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `PolicyInvocation` struct.
type PolicyInvocationQueryResult struct {
	Data       PolicyInvocations
	PageResult *util.PaginationResult
}

// Defining the slice of `PolicyInvocation` struct.
type PolicyInvocations []*PolicyInvocation

// Defining the data structure for creating a `PolicyInvocation` struct.
type PolicyInvocationForm struct {
	ID             string          `json:"id"`
	Name           string          `json:"name" binding:"required,max=128"` // Policy name
	Type           string          `json:"type" binding:"required,max=64"`  // Invocation type (failfast | failover)
	RetryPolicy    *RetryPolicy    `json:"retry_policy"`                    // Retry policy
	FallbackPolicy *FallbackPolicy `json:"fallback_policy"`                 // Fallback policy
	Version        int64           `json:"version"`                         // Version
	Enabled        int             `json:"enabled"`                         // Enabled
	Description    *string         `json:"description"`                     // Details
	Creator        *string         `json:"creator,omitempty"`               // Creator
	Modifier       *string         `json:"modifier,omitempty"`              // Modifier
	CreatedAt      time.Time       `json:"created_at"`                      // Create timestamp
	UpdatedAt      time.Time       `json:"updated_at,omitempty"`            // Update timestamp
}

// A validation function for the `PolicyInvocationForm` struct.
func (a *PolicyInvocationForm) Validate() error {
	if a.Name == "" {
		return errors.BadRequest("", "Name is required")
	}
	if a.Type == "" {
		return errors.BadRequest("", "Type is required")
	}
	return nil
}

// Convert `PolicyInvocationForm` to `PolicyInvocation` object.
func (a *PolicyInvocationForm) FillTo(policyInvocation *PolicyInvocation) error {
	policyInvocation.Name = a.Name
	policyInvocation.Type = a.Type
	if a.RetryPolicy != nil {
		a.RetryPolicy.Version = time.Now().UnixMilli()
	}
	policyInvocation.RetryPolicy = func() *string { return json.MarshalToString(a.RetryPolicy) }()
	policyInvocation.FallbackPolicy = func() *string { return json.MarshalToString(a.FallbackPolicy) }()
	policyInvocation.Enabled = a.Enabled
	policyInvocation.Description = a.Description
	policyInvocation.Version = time.Now().UnixMilli()
	return nil
}
