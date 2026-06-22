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
	ID          string          `json:"id" gorm:"type:char(20);primaryKey;<-:create;comment:主键ID (XID);"`
	TenantCode  string          `json:"tenant_code" gorm:"type:varchar(64);not null;default:'';uniqueIndex:uniq_dimensions_policy,priority:1;index:idx_pb_tenant,priority:1;comment:租户唯一英文编码，不限则为空字符串;"`
	UserID      string          `json:"user_id" gorm:"type:char(20);not null;default:'';uniqueIndex:uniq_dimensions_policy,priority:2;index:idx_pb_user,priority:1;comment:用户唯一ID (XID)，不限则为空字符串;"`
	ModelCode   string          `json:"model_code" gorm:"type:varchar(64);not null;default:'';uniqueIndex:uniq_dimensions_policy,priority:3;index:idx_pb_model,priority:1;comment:模型唯一编码，不限则为空字符串;"`
	PolicyType  string          `json:"policy_type" gorm:"type:varchar(64);not null;uniqueIndex:uniq_dimensions_policy,priority:4;comment:策略类型：tagging / loadbalance / invocation / limit / route / circuit_break;"`
	PolicyID    string          `json:"policy_id" gorm:"type:char(20);not null;uniqueIndex:uniq_dimensions_policy,priority:5;comment:关联的具体策略表主键 ID (XID);"`
	Priority    int             `json:"priority" gorm:"type:int;not null;default:0;comment:冲突合并时的优先级，数字越小越优先;"`
	Enabled     int             `json:"enabled" gorm:"type:int;not null;default:0;comment:启用状态: 0-未启用，1-启用;"`
	Description *string         `json:"description,omitempty" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	Creator     *string         `json:"creator,omitempty" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier    *string         `json:"modifier,omitempty" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt   time.Time       `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt   time.Time       `json:"updated_at,omitempty" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted     string          `json:"-" gorm:"type:varchar(20);not null;default:'0';uniqueIndex:uniq_dimensions_policy,priority:6;index:idx_pb_tenant,priority:2;index:idx_pb_user,priority:2;index:idx_pb_model,priority:2;comment:逻辑删除标识;"`
	DeletedAt   *gorm.DeletedAt `json:"-" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
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
