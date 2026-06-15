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
	ID        string `json:"id" gorm:"size:20;primarykey;"`                                  // Unique ID (XID)
	ModelName string `json:"model_name" gorm:"size:128;uniqueIndex:uniq_model_name;"`        // Client-facing model name, e.g., gpt-4, claude-sonnet
	ModelCode string `json:"model_code" gorm:"size:64;uniqueIndex:uniq_model_code_deleted;"` // Internal model code for association and billing
	SpaceCode string `json:"space_code" gorm:"size:255;not null;"`                           // Model space code
	// Space         *schema.Space   `json:"space,omitempty" gorm:"foreignKey:SpaceCode;references:Code"`    // Space association
	RequestTypes       string          `json:"request_types" gorm:"column:request_types;type:json;not null;"`                                         // Model RequestTypes JSON, e.g., ["chat_completion", "embedding"]
	ContextLength      int             `json:"context_length" gorm:"not null;default:0;"`                                                             // Max context window (Tokens)
	MaxOutputTokens    int             `json:"max_output_tokens" gorm:"not null;default:8192;"`                                                       // Max output tokens
	Abilities          string          `json:"abilities" gorm:"column:abilities;type:json;"`                                                          // Model Abilities JSON, e.g., ["stream", "tool_call"]
	Owner              string          `json:"owner,omitempty" gorm:"size:64;"`                                                                       // Model owner/enterprise, e.g., OpenAI, Google, DeepSeek
	Enabled            int             `json:"enabled" gorm:"not null;default:0;"`                                                                    // Enable status: 0-disabled, 1-enabled
	InputPrice         float64         `json:"input_price" gorm:"column:input_price;type:decimal(10,6);not null;default:0.002000;"`                   // Input price (CNY/M Tokens)
	OutputPrice        float64         `json:"output_price" gorm:"column:output_price;type:decimal(10,6);not null;default:0.002000;"`                 // Output price (CNY/M Tokens)
	CachedPrice        float64         `json:"cached_price" gorm:"column:cached_price;type:decimal(10,6);not null;default:0.002000;"`                 // Cached price (CNY/M Tokens)
	CacheCreationPrice float64         `json:"cache_creation_price" gorm:"column:cache_creation_price;type:decimal(10,6);not null;default:0.002000;"` // Cache creation price (CNY/M Tokens)
	Description        string          `json:"description" gorm:"size:255;"`                                                                          // Description
	Extra              *string         `json:"extra,omitempty" gorm:"type:json"`                                                                      // Extra info
	Creator            string          `json:"creator" gorm:"size:255;"`                                                                              // Creator
	Modifier           string          `json:"modifier" gorm:"size:255;"`                                                                             // Modifier
	CreatedAt          time.Time       `json:"created_at" gorm:"index;"`                                                                              // Create time
	UpdatedAt          time.Time       `json:"updated_at" gorm:"index;"`                                                                              // Update time
	StatusPoints       []StatusPoint   `json:"status_points" gorm:"-"`                                                                                // Recent status points
	Deleted            string          `json:"-" gorm:"size:20;uniqueIndex:uniq_model_code_deleted;default:0"`                                        // Logical delete flag
	DeletedAt          *gorm.DeletedAt `json:"-" gorm:"comment:Delete time;"`                                                                         // Delete time
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
