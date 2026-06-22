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
	ID             string          `json:"id" gorm:"type:char(20);primaryKey;<-:create;comment:主键ID (XID);"`
	Name           string          `json:"name" gorm:"type:varchar(128);not null;uniqueIndex:uniq_policy_invocation_name;comment:策略名称;"`
	Type           string          `json:"type" gorm:"type:varchar(64);not null;default:'failover';comment:调用类型：failover,failfast;"`
	RetryPolicy    *string         `json:"retry_policy,omitempty" gorm:"type:json;default:null;comment:重试策略;"`
	FallbackPolicy *string         `json:"fallback_policy,omitempty" gorm:"type:json;default:null;comment:降级策略;"`
	Version        int64           `json:"version" gorm:"type:bigint;not null;default:1;comment:配置版本号;"`
	Enabled        int             `json:"enabled" gorm:"type:int;not null;default:0;comment:启用状态: 0-未启用，1-启用;"`
	Description    *string         `json:"description,omitempty" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	Creator        *string         `json:"creator,omitempty" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier       *string         `json:"modifier,omitempty" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt      time.Time       `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt      time.Time       `json:"updated_at,omitempty" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted        string          `json:"-" gorm:"type:varchar(20);not null;default:'0';comment:逻辑删除标识;"`
	DeletedAt      *gorm.DeletedAt `json:"-" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
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
