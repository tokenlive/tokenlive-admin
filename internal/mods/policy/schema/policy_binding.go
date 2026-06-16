package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// PolicyBinding 策略绑定表，管理策略与实体的多对多应用关系
type PolicyBinding struct {
	ID          string          `json:"id" gorm:"size:20;primaryKey;<-:create;comment:Unique ID;"`
	TenantCode  string          `json:"tenant_code" gorm:"size:64;not null;default:'';uniqueIndex:uniq_dimensions_policy,priority:1;index:idx_pb_tenant,priority:1;comment:Tenant code, empty for unlimited;"`
	UserID      string          `json:"user_id" gorm:"size:20;not null;default:'';uniqueIndex:uniq_dimensions_policy,priority:2;index:idx_pb_user,priority:1;comment:User ID, empty for unlimited;"`
	ModelCode   string          `json:"model_code" gorm:"size:64;not null;default:'';uniqueIndex:uniq_dimensions_policy,priority:3;index:idx_pb_model,priority:1;comment:Model code, empty for unlimited;"`
	PolicyType  string          `json:"policy_type" gorm:"size:64;not null;uniqueIndex:uniq_dimensions_policy,priority:4;comment:Policy type: tagging/loadbalance/invocation/limit/route/circuit_break;"`
	PolicyID    string          `json:"policy_id" gorm:"size:20;not null;uniqueIndex:uniq_dimensions_policy,priority:5;comment:Policy ID;"`
	Priority    int             `json:"priority" gorm:"not null;default:0;comment:Priority (smaller is higher priority);"`
	Enabled     int             `json:"enabled" gorm:"not null;default:0;comment:Enabled;"`
	Description *string         `json:"description,omitempty" gorm:"size:255;comment:Details;"`
	Creator     *string         `json:"creator,omitempty" gorm:"size:255;comment:Creator;"`
	Modifier    *string         `json:"modifier,omitempty" gorm:"size:255;comment:Modifier;"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime;comment:Create timestamp;"`
	UpdatedAt   time.Time       `json:"updated_at,omitempty" gorm:"autoUpdateTime;comment:Update timestamp;"`
	Deleted     string          `json:"-" gorm:"size:20;default:0;index:idx_pb_tenant,priority:2;index:idx_pb_user,priority:2;index:idx_pb_model,priority:2;comment:Delete flag;"`
	DeletedAt   *gorm.DeletedAt `json:"-" gorm:"type:datetime;comment:Delete timestamp;"`
}

func (a PolicyBinding) TableName() string {
	return config.C.FormatTableName("policy_binding")
}

// ConvertTo Convert `PolicyBinding` to `PolicyBindingForm` object.
func (a PolicyBinding) ConvertTo(form *PolicyBindingForm) error {
	form.ID = a.ID
	form.TenantCode = a.TenantCode
	form.UserID = a.UserID
	form.ModelCode = a.ModelCode
	form.PolicyType = a.PolicyType
	form.PolicyID = a.PolicyID
	form.Priority = a.Priority
	form.Enabled = a.Enabled
	form.Description = a.Description
	form.Creator = a.Creator
	form.Modifier = a.Modifier
	form.CreatedAt = a.CreatedAt
	form.UpdatedAt = a.UpdatedAt
	return nil
}

// Defining the query parameters for the `PolicyBinding` struct.
type PolicyBindingQueryParam struct {
	util.PaginationParam
	TenantCode string `form:"tenant_code"` // Tenant code
	UserID     string `form:"user_id"`     // User ID
	ModelCode  string `form:"model_code"`  // Model code
	PolicyType string `form:"policy_type"` // Policy type (tagging/loadbalance/invocation/limit/route/circuit_break)
	PolicyID   string `form:"policy_id"`   // Policy ID
	Enabled    *int   `form:"enabled"`     // Enabled status
}

// Defining the query options for the `PolicyBinding` struct.
type PolicyBindingQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `PolicyBinding` struct.
type PolicyBindingQueryResult struct {
	Data       PolicyBindings
	PageResult *util.PaginationResult
}

// Defining the slice of `PolicyBinding` struct.
type PolicyBindings []*PolicyBinding

// Defining the data structure for creating or updating a `PolicyBinding` struct.
type PolicyBindingForm struct {
	ID          string    `json:"id"`
	TenantCode  string    `json:"tenant_code" binding:"omitempty,max=64"`
	UserID      string    `json:"user_id" binding:"omitempty,max=20"`
	ModelCode   string    `json:"model_code" binding:"omitempty,max=64"`
	PolicyType  string    `json:"policy_type" binding:"required,oneof=tagging loadbalance invocation limit route circuit_break"`
	PolicyID    string    `json:"policy_id" binding:"required,max=20"`
	Priority    int       `json:"priority" binding:"min=0"`
	Enabled     int       `json:"enabled" binding:"oneof=0 1"`
	Description *string   `json:"description" binding:"omitempty,max=255"`
	Creator     *string   `json:"creator,omitempty"`
	Modifier    *string   `json:"modifier,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// A validation function for the `PolicyBindingForm` struct.
func (a *PolicyBindingForm) Validate() error {
	if a.PolicyType == "" {
		return errors.BadRequest("", "policy_type is required")
	}
	if a.PolicyID == "" {
		return errors.BadRequest("", "policy_id is required")
	}
	return nil
}

// Convert `PolicyBindingForm` to `PolicyBinding` object.
func (a *PolicyBindingForm) FillTo(binding *PolicyBinding) error {
	binding.TenantCode = a.TenantCode
	binding.UserID = a.UserID
	binding.ModelCode = a.ModelCode
	binding.PolicyType = a.PolicyType
	binding.PolicyID = a.PolicyID
	binding.Priority = a.Priority
	binding.Enabled = a.Enabled
	binding.Description = a.Description
	return nil
}
