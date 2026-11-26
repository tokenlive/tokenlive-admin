package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

const (
	DialectDeveloperToSystem = "chat_developer_to_system"
	DialectStoreUnsupported  = "chat_store_unsupported"
)

// PolicyDialect belongs to one ordinary model.
type PolicyDialect struct {
	ID             string          `json:"id" gorm:"type:char(20);primaryKey;<-:create;comment:主键ID (XID);"`
	ModelID        string          `json:"model_id" gorm:"type:char(20);not null;uniqueIndex:uniq_policy_dialect_model_impl_deleted,priority:1;index:idx_policy_dialect_model,priority:1;comment:所属普通模型ID;"`
	ScopeType      string          `json:"scope_type" gorm:"type:varchar(32);not null;default:'global';index:idx_policy_dialect_scope,priority:1;comment:作用域类型，global 表示对所属模型全局生效;"`
	ScopeCode      string          `json:"scope_code" gorm:"type:varchar(128);not null;default:'';index:idx_policy_dialect_scope,priority:2;comment:作用域编码，global 作用域为空;"`
	Name           string          `json:"name" gorm:"type:varchar(128);not null;default:'';comment:策略名称，创建时使用适配实现ID;"`
	Implementation string          `json:"implementation" gorm:"type:varchar(64);not null;uniqueIndex:uniq_policy_dialect_model_impl_deleted,priority:2;comment:内置方言适配实现ID;"`
	Version        int64           `json:"version" gorm:"type:bigint;not null;default:1;comment:配置版本号;"`
	Enabled        int             `json:"enabled" gorm:"type:int;size:32;not null;default:0;index:idx_policy_dialect_model,priority:3;comment:启用状态: 0-未启用，1-启用;"`
	Description    *string         `json:"description,omitempty" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	Creator        *string         `json:"creator,omitempty" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier       *string         `json:"modifier,omitempty" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt      time.Time       `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt      time.Time       `json:"updated_at,omitempty" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted        string          `json:"-" gorm:"type:varchar(20);not null;default:'0';uniqueIndex:uniq_policy_dialect_model_impl_deleted,priority:3;index:idx_policy_dialect_model,priority:2;index:idx_policy_dialect_scope,priority:3;comment:逻辑删除标识;"`
	DeletedAt      *gorm.DeletedAt `json:"-" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
}

func (PolicyDialect) TableName() string {
	return config.C.FormatTableName("policy_dialect")
}

type PolicyDialectQuery struct {
	util.PaginationParam
	ModelID string `form:"model_id" binding:"required,max=20"`
}

type PolicyDialectForm struct {
	ModelID        string `json:"model_id" binding:"required,max=20"`
	Implementation string `json:"implementation" binding:"required,max=64"`
}

type PolicyDialectEnabledForm struct {
	Enabled *int `json:"enabled" binding:"required,oneof=0 1"`
}
