package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ModelCatalogI18n 模型目录多语言内容表，为 ModelCatalog 提供国际化描述、SEO 信息和展示标签。
type ModelCatalogI18n struct {
	ModelID          string    `json:"model_id" gorm:"type:varchar(191);primaryKey;comment:模型ID，关联 model_catalog.model_id;"`
	Locale           string    `json:"locale" gorm:"type:varchar(16);primaryKey;comment:语言区域，如 zh-CN, en-US;"`
	DisplayName      string    `json:"display_name" gorm:"type:varchar(255);not null;comment:模型展示名称;"`
	ShortDescription string    `json:"short_description" gorm:"type:varchar(512);not null;default:'';comment:短描述，用于列表展示;"`
	LongDescription  *string   `json:"long_description" gorm:"type:text;default:null;comment:长描述，支持 Markdown，用于详情页;"`
	SEOTitle         string    `json:"seo_title" gorm:"type:varchar(255);not null;default:'';comment:SEO 标题;"`
	SEODescription   string    `json:"seo_description" gorm:"type:varchar(512);not null;default:'';comment:SEO 描述;"`
	Tags             *string   `json:"tags,omitempty" gorm:"type:json;default:null;comment:展示标签，如 [\"最新\",\"高性价比\"];"`
	Creator          string    `json:"creator" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier         string    `json:"modifier" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt        time.Time `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
}

func (ModelCatalogI18n) TableName() string {
	return config.C.FormatTableName("model_catalog_i18n")
}

// ModelCatalogI18nQueryParam defines the query parameters for ModelCatalogI18n.
type ModelCatalogI18nQueryParam struct {
	util.PaginationParam
	ModelID string `form:"model_id"` // Filter by model ID
	Locale  string `form:"locale"`   // Filter by locale
}

// ModelCatalogI18nQueryOptions defines the query options for ModelCatalogI18n.
type ModelCatalogI18nQueryOptions struct {
	util.QueryOptions
}

// ModelCatalogI18nQueryResult defines the query result for ModelCatalogI18n.
type ModelCatalogI18nQueryResult struct {
	Data       ModelCatalogI18ns
	PageResult *util.PaginationResult
}

// ModelCatalogI18ns defines a slice of ModelCatalogI18n.
type ModelCatalogI18ns []*ModelCatalogI18n

// ModelCatalogI18nForm defines the form for creating/updating a ModelCatalogI18n.
type ModelCatalogI18nForm struct {
	ModelID          string  `json:"model_id" binding:"required,max=191"`     // Model ID
	Locale           string  `json:"locale" binding:"required,max=16"`        // Locale
	DisplayName      string  `json:"display_name" binding:"required,max=255"` // Display name
	ShortDescription string  `json:"short_description" binding:"max=512"`     // Short description
	LongDescription  *string `json:"long_description"`                        // Long description (Markdown)
	SEOTitle         string  `json:"seo_title" binding:"max=255"`             // SEO title
	SEODescription   string  `json:"seo_description" binding:"max=512"`       // SEO description
	Tags             *string `json:"tags"`                                    // Display tags
}

func (a *ModelCatalogI18nForm) Validate() error {
	return nil
}

func (a *ModelCatalogI18nForm) FillTo(i18n *ModelCatalogI18n) error {
	i18n.ModelID = a.ModelID
	i18n.Locale = a.Locale
	i18n.DisplayName = a.DisplayName
	i18n.ShortDescription = a.ShortDescription
	i18n.LongDescription = a.LongDescription
	i18n.SEOTitle = a.SEOTitle
	i18n.SEODescription = a.SEODescription
	i18n.Tags = a.Tags
	return nil
}

// ModelCatalogI18nBatchForm defines the batch form for upserting i18n entries for a model.
type ModelCatalogI18nBatchForm struct {
	ModelID string                 `json:"model_id" binding:"required,max=191"` // Model ID
	Entries []ModelCatalogI18nForm `json:"entries" binding:"required,dive"`     // I18n entries
}

func (a *ModelCatalogI18nBatchForm) Validate() error {
	return nil
}
