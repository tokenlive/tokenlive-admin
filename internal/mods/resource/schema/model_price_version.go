package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ModelPriceVersion 模型价格版本表，支持价格版本化管理和生效区间控制。
// 与 Model/Endpoint 上的直接价格字段互补：Model 价格用于网关实时计算，PriceVersion 用于 Portal 展示和历史追溯。
type ModelPriceVersion struct {
	ID                           string     `json:"id" gorm:"type:char(20);primaryKey;<-:create;comment:主键ID (XID);"`
	ModelID                      string     `json:"model_id" gorm:"type:varchar(191);not null;uniqueIndex:uniq_mpv_model_effective,priority:1;index:idx_mpv_current,priority:1;comment:模型ID，关联 model_catalog.model_id;"`
	Currency                     string     `json:"currency" gorm:"type:varchar(8);not null;default:CNY;comment:计价货币，如 CNY, USD;"`
	InputMicroCNYPer1MTokens     int64      `json:"input_micro_cny_per_1m_tokens" gorm:"type:bigint;not null;comment:每百万输入 Token 价格（微分）;"`
	OutputMicroCNYPer1MTokens    int64      `json:"output_micro_cny_per_1m_tokens" gorm:"type:bigint;not null;comment:每百万输出 Token 价格（微分）;"`
	CacheReadMicroCNYPer1MTokens *int64     `json:"cache_read_micro_cny_per_1m_tokens" gorm:"type:bigint;default:null;comment:每百万缓存读取 Token 价格（微分），NULL 表示不支持缓存;"`
	EffectiveFrom                time.Time  `json:"effective_from" gorm:"type:datetime(3);not null;uniqueIndex:uniq_mpv_model_effective,priority:2;index:idx_mpv_current,priority:3;comment:生效开始时间;"`
	EffectiveUntil               *time.Time `json:"effective_until" gorm:"type:datetime(3);default:null;index:idx_mpv_current,priority:4;comment:生效结束时间，NULL 表示永久有效;"`
	Status                       string     `json:"status" gorm:"type:varchar(32);not null;default:active;index:idx_mpv_current,priority:2;comment:价格状态: active, inactive;"`
	PublishedByUser              string     `json:"published_by_user" gorm:"type:varchar(255);default:null;comment:发布人;"`
	PublishedAt                  time.Time  `json:"published_at" gorm:"type:datetime(3);not null;comment:发布时间;"`
	Creator                      string     `json:"creator" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier                     string     `json:"modifier" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt                    time.Time  `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt                    time.Time  `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
}

func (ModelPriceVersion) TableName() string {
	return config.C.FormatTableName("model_price_version")
}

const (
	ModelPriceStatusActive   = "active"
	ModelPriceStatusInactive = "inactive"
)

// ModelPriceVersionQueryParam defines the query parameters for ModelPriceVersion.
type ModelPriceVersionQueryParam struct {
	util.PaginationParam
	ModelID  string `form:"model_id"`  // Filter by model ID
	Status   string `form:"status"`    // Filter by status
	Currency string `form:"currency"`  // Filter by currency
}

// ModelPriceVersionQueryOptions defines the query options for ModelPriceVersion.
type ModelPriceVersionQueryOptions struct {
	util.QueryOptions
}

// ModelPriceVersionQueryResult defines the query result for ModelPriceVersion.
type ModelPriceVersionQueryResult struct {
	Data       ModelPriceVersions
	PageResult *util.PaginationResult
}

// ModelPriceVersions defines a slice of ModelPriceVersion.
type ModelPriceVersions []*ModelPriceVersion

// ModelPriceVersionForm defines the form for creating/updating a ModelPriceVersion.
type ModelPriceVersionForm struct {
	ModelID                      string     `json:"model_id" binding:"required,max=191"`       // Model ID
	Currency                     string     `json:"currency" binding:"required,max=8"`          // Currency
	InputMicroCNYPer1MTokens     int64      `json:"input_micro_cny_per_1m_tokens" binding:"min=0"`  // Input price
	OutputMicroCNYPer1MTokens    int64      `json:"output_micro_cny_per_1m_tokens" binding:"min=0"` // Output price
	CacheReadMicroCNYPer1MTokens *int64     `json:"cache_read_micro_cny_per_1m_tokens"`              // Cache read price
	EffectiveFrom                time.Time  `json:"effective_from" binding:"required"`           // Effective from
	EffectiveUntil               *time.Time `json:"effective_until"`                              // Effective until
}

func (a *ModelPriceVersionForm) Validate() error {
	if a.ModelID == "" {
		return errors.BadRequest("", "ModelID is required")
	}
	if a.InputMicroCNYPer1MTokens < 0 {
		return errors.BadRequest("", "Input price must be non-negative")
	}
	if a.OutputMicroCNYPer1MTokens < 0 {
		return errors.BadRequest("", "Output price must be non-negative")
	}
	if a.CacheReadMicroCNYPer1MTokens != nil && *a.CacheReadMicroCNYPer1MTokens < 0 {
		return errors.BadRequest("", "Cache read price must be non-negative")
	}
	return nil
}

func (a *ModelPriceVersionForm) FillTo(version *ModelPriceVersion) error {
	version.ModelID = a.ModelID
	version.Currency = a.Currency
	version.InputMicroCNYPer1MTokens = a.InputMicroCNYPer1MTokens
	version.OutputMicroCNYPer1MTokens = a.OutputMicroCNYPer1MTokens
	version.CacheReadMicroCNYPer1MTokens = a.CacheReadMicroCNYPer1MTokens
	version.EffectiveFrom = a.EffectiveFrom
	version.EffectiveUntil = a.EffectiveUntil
	version.Status = ModelPriceStatusActive
	version.PublishedAt = time.Now()
	return nil
}

// GetCurrentPriceQuery 查询当前生效的价格版本
type GetCurrentPriceQuery struct {
	ModelID  string `form:"model_id" binding:"required"` // Model ID
	Currency string `form:"currency"`                     // Currency (default CNY)
}
