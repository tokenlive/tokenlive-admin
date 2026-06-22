package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// UserAPIKey 用户 API Key 结构体
type UserAPIKey struct {
	ID          string     `json:"id" gorm:"type:char(20);primaryKey;comment:主键ID (XID);"`
	UserID      string     `json:"user_id" gorm:"type:char(20);not null;index:idx_user_id,priority:1;comment:关联的用户 ID;"`
	Name        string     `json:"name" gorm:"type:varchar(64);not null;comment:API Key 友好名称;"`
	APIKey      string     `json:"api_key" gorm:"type:varchar(128);not null;uniqueIndex:uniq_api_key_deleted,priority:1;comment:实际的 API Key 字符串;"`
	Status      int        `json:"status" gorm:"type:int;not null;default:1;comment:状态: 1-启用, 2-禁用;"`
	Quota       int64      `json:"quota" gorm:"type:bigint;not null;default:-1;comment:剩余配额: -1表示无限制;"`
	ExpiresAt   *time.Time `json:"expires_at" gorm:"type:datetime;default:null;comment:过期时间: NULL表示永不过期;"`
	Description string     `json:"description" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	Creator     string     `json:"creator" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier    string     `json:"modifier" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt   time.Time  `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted     string     `json:"deleted" gorm:"type:varchar(20);not null;default:'0';uniqueIndex:uniq_api_key_deleted,priority:2;index:idx_user_id,priority:2;comment:逻辑删除标识;"`
	DeletedAt   *time.Time `json:"deleted_at" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
}

// TableName 返回对应的数据库表名
func (a *UserAPIKey) TableName() string {
	return config.C.FormatTableName("user_api_key")
}

// MaskAPIKey 掩码 API Key，脱敏安全保护
func MaskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "****" + apiKey[len(apiKey)-4:]
}

// Mask 对当前的 UserAPIKey 进行掩码处理
func (a *UserAPIKey) Mask() {
	a.APIKey = MaskAPIKey(a.APIKey)
}

// UserAPIKeyQueryParam 定义 API Key 查询参数
type UserAPIKeyQueryParam struct {
	util.PaginationParam
	UserID   string `form:"user_id"` // 按用户ID过滤
	LikeName string `form:"name"`    // 按友好名称模糊搜索
	Status   int    `form:"status"`  // 按启用/禁用状态过滤
}

// UserAPIKeyQueryOptions 定义查询可选项
type UserAPIKeyQueryOptions struct {
	util.QueryOptions
}

// UserAPIKeyQueryResult 定义查询结果结构体
type UserAPIKeyQueryResult struct {
	Data       UserAPIKeys
	PageResult *util.PaginationResult
}

// UserAPIKeys 定义 UserAPIKey 的切片类型
type UserAPIKeys []*UserAPIKey

// Mask 对整个 API Key 切片进行掩码脱敏
func (a UserAPIKeys) Mask() {
	for _, item := range a {
		item.Mask()
	}
}

// UserAPIKeyForm 定义创建与修改 API Key 的表单数据结构
type UserAPIKeyForm struct {
	UserID      string     `json:"user_id" binding:"required,max=20"`   // 关联的用户 ID
	Name        string     `json:"name" binding:"required,max=64"`      // API Key 友好名称
	Status      int        `json:"status" binding:"required,oneof=1 2"` // 状态: 1-启用, 2-禁用
	Quota       int64      `json:"quota"`                               // 剩余配额: -1表示无限制
	ExpiresAt   *time.Time `json:"expires_at"`                          // 过期时间
	Description string     `json:"description" binding:"max=255"`       // 描述信息
}

// Validate 基础校验方法
func (a *UserAPIKeyForm) Validate() error {
	return nil
}

// FillTo 将表单数据填充到 GORM 模型中
func (a *UserAPIKeyForm) FillTo(apiKey *UserAPIKey) {
	apiKey.UserID = a.UserID
	apiKey.Name = a.Name
	apiKey.Status = a.Status
	apiKey.Quota = a.Quota
	apiKey.ExpiresAt = a.ExpiresAt
	apiKey.Description = a.Description
}
