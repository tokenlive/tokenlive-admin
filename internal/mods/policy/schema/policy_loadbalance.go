package schema

import (
	"encoding/json"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// PolicyLoadbalance 负载均衡策略表
type PolicyLoadbalance struct {
	ID          string           `json:"id" gorm:"type:char(20);primaryKey;<-:create;comment:主键ID (XID);"`
	Name        string           `json:"name" gorm:"type:varchar(128);not null;uniqueIndex:uniq_policy_loadbalance_name;comment:策略名称;"`
	Type        string           `json:"type" gorm:"type:varchar(64);not null;comment:负载均衡算法类型，如 ROUND_ROBIN / WEIGHTED / STICKY;"`
	Version     int64            `json:"version" gorm:"type:bigint;not null;default:1;comment:配置版本号;"`
	Enabled     int              `json:"enabled" gorm:"type:int;not null;default:0;comment:启用状态: 0-未启用，1-启用;"`
	Params      *json.RawMessage `json:"params,omitempty" gorm:"type:json;default:null;comment:算法额外参数，如权重等;"`
	Description *string          `json:"description,omitempty" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	Creator     *string          `json:"creator,omitempty" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier    *string          `json:"modifier,omitempty" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt   time.Time        `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt   time.Time        `json:"updated_at,omitempty" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted     string           `json:"-" gorm:"type:varchar(20);not null;default:'0';comment:逻辑删除标识;"`
	DeletedAt   *gorm.DeletedAt  `json:"-" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
}

func (a PolicyLoadbalance) TableName() string {
	return config.C.FormatTableName("policy_loadbalance")
}

// ConvertTo Convert `PolicyLoadbalance` to `PolicyLoadbalanceForm` object.
func (a PolicyLoadbalance) ConvertTo(form *PolicyLoadbalanceForm) error {
	form.ID = a.ID
	form.Name = a.Name
	form.Type = a.Type
	form.Version = a.Version
	form.Enabled = a.Enabled
	form.Params = a.Params
	form.Description = a.Description
	form.Creator = a.Creator
	form.Modifier = a.Modifier
	form.CreatedAt = a.CreatedAt
	form.UpdatedAt = a.UpdatedAt
	return nil
}

// Defining the query parameters for the `PolicyLoadbalance` struct.
type PolicyLoadbalanceQueryParam struct {
	util.PaginationParam
	Name string `form:"name"` // Policy name (like)
	Type string `form:"type"` // Loadbalance type
}

// Defining the query options for the `PolicyLoadbalance` struct.
type PolicyLoadbalanceQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `PolicyLoadbalance` struct.
type PolicyLoadbalanceQueryResult struct {
	Data       PolicyLoadbalances
	PageResult *util.PaginationResult
}

// Defining the slice of `PolicyLoadbalance` struct.
type PolicyLoadbalances []*PolicyLoadbalance

// Defining the data structure for creating a `PolicyLoadbalance` struct.
type PolicyLoadbalanceForm struct {
	ID          string           `json:"id"`
	Name        string           `json:"name" binding:"required,max=128"` // Policy name
	Type        string           `json:"type" binding:"required,max=64"`  // Loadbalance policy type
	Version     int64            `json:"version"`                         // Version
	Enabled     int              `json:"enabled"`                         // Enabled
	Params      *json.RawMessage `json:"params,omitempty"`                // Extra params (JSON)
	Description *string          `json:"description"`                     // Details
	Creator     *string          `json:"creator,omitempty"`               // Creator
	Modifier    *string          `json:"modifier,omitempty"`              // Modifier
	CreatedAt   time.Time        `json:"created_at"`                      // Create timestamp
	UpdatedAt   time.Time        `json:"updated_at,omitempty"`            // Update timestamp
}

// A validation function for the `PolicyLoadbalanceForm` struct.
func (a *PolicyLoadbalanceForm) Validate() error {
	if a.Name == "" {
		return errors.BadRequest("", "Name is required")
	}
	if a.Type == "" {
		return errors.BadRequest("", "Type is required")
	}
	return nil
}

// Convert `PolicyLoadbalanceForm` to `PolicyLoadbalance` object.
func (a *PolicyLoadbalanceForm) FillTo(policyLoadbalance *PolicyLoadbalance) error {
	policyLoadbalance.Name = a.Name
	policyLoadbalance.Type = a.Type
	policyLoadbalance.Enabled = a.Enabled
	policyLoadbalance.Params = a.Params
	policyLoadbalance.Description = a.Description
	policyLoadbalance.Version = time.Now().UnixMilli()
	return nil
}
