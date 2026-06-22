package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	// "github.com/tokenlive/tokenlive-admin/internal/mods/space/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Model defines the LLM model from user's perspective.
type Model struct {
	ID                 string          `json:"id" gorm:"type:char(20);primaryKey;comment:主键ID (XID);"`
	ModelName          string          `json:"model_name" gorm:"type:varchar(128);not null;uniqueIndex:uniq_model_name;comment:模型名称;"`
	ModelCode          string          `json:"model_code" gorm:"type:varchar(64);not null;uniqueIndex:uniq_model_code_deleted,priority:1;comment:模型唯一编码;"`
	SpaceCode          string          `json:"space_code" gorm:"type:varchar(255);not null;comment:模型空间编码;"`
	RequestTypes       string          `json:"request_types" gorm:"type:json;default:null;comment:模型支持的请求类型，如 [\"chat_completion\", \"embedding\"];"`
	ContextLength      int             `json:"context_length" gorm:"type:bigint;not null;default:128000;comment:最大上下文窗口（Tokens）;"`
	MaxOutputTokens    int             `json:"max_output_tokens" gorm:"type:bigint;not null;default:8192;comment:最大输出Token;"`
	Owner              string          `json:"owner,omitempty" gorm:"type:varchar(64);default:null;comment:模型所属企业/厂商，如 OpenAI, Google, DeepSeek;"`
	Abilities          string          `json:"abilities" gorm:"type:json;default:null;comment:能力列表,如:流式输出,工具调用,思维链,结构化输出等;"`
	Enabled            int             `json:"enabled" gorm:"type:int;not null;default:0;comment:启用状态: 0-未启用，1-启用;"`
	InputPrice         float64         `json:"input_price" gorm:"type:decimal(10,6);not null;default:0.002000;comment:输入价格（元/百万 Tokens）;"`
	OutputPrice        float64         `json:"output_price" gorm:"type:decimal(10,6);not null;default:0.002000;comment:输出价格（元/百万 Tokens）;"`
	CachedPrice        float64         `json:"cached_price" gorm:"type:decimal(10,6);not null;default:0.002000;comment:缓存命中价格（元/百万 Tokens）;"`
	CacheCreationPrice float64         `json:"cache_creation_price" gorm:"type:decimal(10,6);not null;default:0.002000;comment:缓存创建价格（元/百万 Tokens）;"`
	Description        string          `json:"description" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	Extra              *string         `json:"extra,omitempty" gorm:"type:json;default:null;comment:其他信息;"`
	Creator            string          `json:"creator" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier           string          `json:"modifier" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt          time.Time       `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt          time.Time       `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	StatusPoints       []StatusPoint   `json:"status_points" gorm:"-"`                                                                                // Recent status points
	Deleted            string          `json:"-" gorm:"type:varchar(20);not null;default:'0';uniqueIndex:uniq_model_code_deleted,priority:2;comment:逻辑删除标识;"`
	DeletedAt          *gorm.DeletedAt `json:"-" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
}

func (m *Model) TableName() string {
	return config.C.FormatTableName("model")
}

// ModelQueryParam defines the query parameters for Model.
type ModelQueryParam struct {
	util.PaginationParam
	LikeName  string `form:"model_name"` // Model name (like)
	ModelCode string `form:"model_code"` // Model code (exact)
	SpaceCode string `form:"space_code"` // Space code
}

// ModelQueryOptions defines the query options for Model.
type ModelQueryOptions struct {
	util.QueryOptions
}

// ModelQueryResult defines the query result for Model.
type ModelQueryResult struct {
	Data       Models
	PageResult *util.PaginationResult
}

// Models defines a slice of Model.
type Models []*Model

// ModelForm defines the form for creating/updating a Model.
type ModelForm struct {
	ModelName          string  `json:"model_name" binding:"required,max=128"` // Client-facing model name
	ModelCode          string  `json:"model_code" binding:"required,max=64"`  // Internal model code
	SpaceCode          string  `json:"space_code" binding:"required,max=255"` // Space code
	RequestTypes       string  `json:"request_types" binding:"required"`      // Model RequestTypes JSON
	ContextLength      int     `json:"context_length"`                        // Max context window
	MaxOutputTokens    int     `json:"max_output_tokens"`                     // Max output tokens
	Abilities          string  `json:"abilities"`                             // Model Abilities JSON
	Owner              string  `json:"owner"`                                 // Model owner
	Enabled            int     `json:"enabled"`                               // Enable status
	InputPrice         float64 `json:"input_price"`                           // Input price (CNY/M Tokens)
	OutputPrice        float64 `json:"output_price"`                          // Output price (CNY/M Tokens)
	CachedPrice        float64 `json:"cached_price"`                          // Cached price (CNY/M Tokens)
	CacheCreationPrice float64 `json:"cache_creation_price"`                  // Cache creation price (CNY/M Tokens)
	Extra              *string `json:"extra"`                                 // Extra info
	Description        string  `json:"description"`                           // Description
}

func (m *ModelForm) Validate() error {
	return nil
}

func (m *ModelForm) FillTo(model *Model) error {
	model.ModelName = m.ModelName
	model.ModelCode = m.ModelCode
	model.SpaceCode = m.SpaceCode
	model.RequestTypes = m.RequestTypes
	model.ContextLength = m.ContextLength
	model.MaxOutputTokens = m.MaxOutputTokens
	model.Abilities = m.Abilities
	model.Owner = m.Owner
	model.Enabled = m.Enabled
	model.InputPrice = m.InputPrice
	model.OutputPrice = m.OutputPrice
	model.CachedPrice = m.CachedPrice
	model.CacheCreationPrice = m.CacheCreationPrice
	model.Extra = toNilIfEmpty(m.Extra)
	model.Description = m.Description
	return nil
}

func toNilIfEmpty(s *string) *string {
	if s != nil && *s == "" {
		return nil
	}
	return s
}

// StatusPoint 最近状态时间点的成功失败数
type StatusPoint struct {
	SuccessCount int64  `json:"success_count"`
	FailCount    int64  `json:"fail_count"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
}
