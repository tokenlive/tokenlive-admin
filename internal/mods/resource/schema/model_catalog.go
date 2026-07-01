package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ModelCatalog 面向用户的模型目录，与内部 Model 表形成互补。
// Model 表面向运维管理（内部编码、端点关联、价格），ModelCatalog 面向终端用户展示（国际化描述、SEO、能力标签、排序权重）。
type ModelCatalog struct {
	ModelID          string     `json:"model_id" gorm:"type:varchar(191);primaryKey;comment:模型ID，关联 model.model_code 或自定义标识;"`
	ModelCode        string     `json:"model_code" gorm:"type:varchar(64);default:null;index:idx_mc_model_code;comment:关联 admin model.model_code，桥接内部模型;"`
	Slug             string     `json:"slug" gorm:"type:varchar(191);not null;uniqueIndex:uniq_mc_slug;comment:URL 友好标识，如 gpt-4o, claude-sonnet-4;"`
	Status           string     `json:"status" gorm:"type:varchar(32);not null;default:available;index:idx_mc_public_list,priority:2;comment:状态: available, paused;"`
	Visibility       string     `json:"visibility" gorm:"type:varchar(32);not null;default:public;index:idx_mc_public_list,priority:1;comment:可见性: public, private;"`
	LogoURL          string     `json:"logo_url" gorm:"type:varchar(1024);not null;default:'';comment:模型 Logo URL;"`
	ContextLength    *int64     `json:"context_length" gorm:"type:bigint;default:null;comment:最大上下文窗口（Tokens）;"`
	KnowledgeCutoff  *time.Time `json:"knowledge_cutoff" gorm:"type:date;default:null;comment:知识截止日期;"`
	InputModalities  *string    `json:"input_modalities,omitempty" gorm:"type:json;default:null;comment:支持的输入模态，如 [\"text\",\"image\"];"`
	OutputModalities *string    `json:"output_modalities,omitempty" gorm:"type:json;default:null;comment:支持的输出模态，如 [\"text\"];"`
	Capabilities     *string    `json:"capabilities,omitempty" gorm:"type:json;default:null;comment:模型能力标签，如 [\"streaming\",\"tool_use\",\"reasoning\"];"`
	Featured         bool       `json:"featured" gorm:"not null;default:false;index:idx_mc_public_list,priority:3;comment:是否精选推荐;"`
	SortWeight       int64      `json:"sort_weight" gorm:"type:bigint;not null;default:0;index:idx_mc_public_list,priority:4;comment:排序权重，数值越大越靠前;"`
	PublishedAt      *time.Time `json:"published_at" gorm:"type:datetime;default:null;index:idx_mc_public_list,priority:5;comment:首次发布时间;"`
	Creator          string     `json:"creator" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier         string     `json:"modifier" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt        time.Time  `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt        time.Time  `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
}

func (ModelCatalog) TableName() string {
	return config.C.FormatTableName("model_catalog")
}

const (
	ModelCatalogStatusAvailable = "available"
	ModelCatalogStatusPaused    = "paused"

	ModelCatalogVisibilityPublic  = "public"
	ModelCatalogVisibilityPrivate = "private"
)

// ModelCatalogQueryParam defines the query parameters for ModelCatalog.
type ModelCatalogQueryParam struct {
	util.PaginationParam
	LikeSlug   string `form:"slug"`       // Slug (like)
	Status     string `form:"status"`     // Filter by status
	Visibility string `form:"visibility"` // Filter by visibility
	Featured   *bool  `form:"featured"`   // Filter by featured
	ModelCode  string `form:"model_code"` // Filter by model_code
}

// ModelCatalogQueryOptions defines the query options for ModelCatalog.
type ModelCatalogQueryOptions struct {
	util.QueryOptions
}

// ModelCatalogQueryResult defines the query result for ModelCatalog.
type ModelCatalogQueryResult struct {
	Data       ModelCatalogs
	PageResult *util.PaginationResult
}

// ModelCatalogs defines a slice of ModelCatalog.
type ModelCatalogs []*ModelCatalog

// ModelCatalogForm defines the form for creating/updating a ModelCatalog.
type ModelCatalogForm struct {
	ModelID          string     `json:"model_id" binding:"required,max=191"` // Model ID
	ModelCode        string     `json:"model_code" binding:"max=64"`         // Link to admin model
	Slug             string     `json:"slug" binding:"required,max=191"`     // URL slug
	Status           string     `json:"status" binding:"required,oneof=available paused"`
	Visibility       string     `json:"visibility" binding:"required,oneof=public private"`
	LogoURL          string     `json:"logo_url" binding:"max=1024"` // Logo URL
	ContextLength    *int64     `json:"context_length"`              // Context window
	KnowledgeCutoff  *time.Time `json:"knowledge_cutoff"`            // Knowledge cutoff
	InputModalities  *string    `json:"input_modalities"`            // Input modalities
	OutputModalities *string    `json:"output_modalities"`           // Output modalities
	Capabilities     *string    `json:"capabilities"`                // Capabilities
	Featured         *bool      `json:"featured"`                    // Featured
	SortWeight       *int64     `json:"sort_weight"`                 // Sort weight
}

func (a *ModelCatalogForm) Validate() error {
	if a.ModelID == "" {
		return errors.BadRequest("", "ModelID is required")
	}
	if a.Slug == "" {
		return errors.BadRequest("", "Slug is required")
	}
	return nil
}

func (a *ModelCatalogForm) FillTo(catalog *ModelCatalog) error {
	catalog.ModelID = a.ModelID
	catalog.ModelCode = a.ModelCode
	catalog.Slug = a.Slug
	catalog.Status = a.Status
	catalog.Visibility = a.Visibility
	catalog.LogoURL = a.LogoURL
	catalog.ContextLength = a.ContextLength
	catalog.KnowledgeCutoff = a.KnowledgeCutoff
	catalog.InputModalities = a.InputModalities
	catalog.OutputModalities = a.OutputModalities
	catalog.Capabilities = a.Capabilities
	if a.Featured != nil {
		catalog.Featured = *a.Featured
	}
	if a.SortWeight != nil {
		catalog.SortWeight = *a.SortWeight
	}
	return nil
}

// ModelCatalogPublishForm defines the form for publishing a model catalog.
type ModelCatalogPublishForm struct {
	Visibility  string     `json:"visibility" binding:"required,oneof=public private"` // Visibility
	PublishedAt *time.Time `json:"published_at"`                                       // Publish time (defaults to now)
}

func (a *ModelCatalogPublishForm) Validate() error {
	return nil
}

// ModelCatalogMetric represents model service metrics from Prometheus
type ModelCatalogMetric struct {
	Window        string  `json:"window"`         // Time window: "1h", "24h", "7d"
	Availability  float64 `json:"availability"`   // Availability rate (0-1)
	SuccessRate   float64 `json:"success_rate"`   // Success rate (0-1)
	TtftP50Ms     float64 `json:"ttft_p50_ms"`    // TTFT p50 in milliseconds
	TtftP95Ms     float64 `json:"ttft_p95_ms"`    // TTFT p95 in milliseconds
	ResponseSpeed float64 `json:"response_speed"` // Response speed (tokens per second)
	SampleCount   int64   `json:"sample_count"`   // Number of samples/requests
	UpdatedAt     string  `json:"updated_at"`     // Last update time
}

// ModelCatalogMetricResponse is the response for metrics API
type ModelCatalogMetricResponse []ModelCatalogMetric
