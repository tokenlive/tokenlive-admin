package schema

import (
	"encoding/json"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Provider defines the upstream LLM provider (e.g., OpenAI, Anthropic).
type Provider struct {
	ID          string          `json:"id" gorm:"type:char(20);primaryKey;comment:主键ID (XID);"`
	Code        string          `json:"code" gorm:"type:varchar(128);not null;uniqueIndex:uniq_provider_code,priority:1;comment:Provider唯一标识;"`
	Name        string          `json:"name" gorm:"type:varchar(128);not null;comment:Provider名称;"`
	Protocol    string          `json:"protocol" gorm:"type:varchar(64);not null;comment:协议类型，决定使用哪个 ProviderFactory;"`
	URL         string          `json:"url" gorm:"type:varchar(512);default:null;comment:供应商 API 基础地址;"`
	ApiKeys     json.RawMessage `json:"api_keys,omitempty" gorm:"type:json;default:null;comment:上游API认证密钥列表;"`
	Enabled     int             `json:"enabled" gorm:"type:int;not null;default:0;comment:启用状态: 0-未启用，1-启用;"`
	Description string          `json:"description" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	Creator     string          `json:"creator" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier    string          `json:"modifier" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt   time.Time       `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted     string          `json:"-" gorm:"type:varchar(20);not null;default:'0';uniqueIndex:uniq_provider_code,priority:2;comment:逻辑删除标识;"`
	DeletedAt   *gorm.DeletedAt `json:"-" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
}

func (p *Provider) TableName() string {
	return config.C.FormatTableName("provider")
}

// ProviderQueryParam defines the query parameters for Provider.
type ProviderQueryParam struct {
	util.PaginationParam
	LikeCode string `form:"code"` // Code (like)
	LikeName string `form:"name"` // Name (like)
}

// ProviderQueryOptions defines the query options for Provider.
type ProviderQueryOptions struct {
	util.QueryOptions
}

// ProviderQueryResult defines the query result for Provider.
type ProviderQueryResult struct {
	Data       Providers
	PageResult *util.PaginationResult
}

// Providers defines a slice of Provider.
type Providers []*Provider

// ProviderForm defines the form for creating/updating a Provider.
type ProviderForm struct {
	Code        string   `json:"code" binding:"required,max=128"`    // Provider unique code
	Name        string   `json:"name" binding:"required,max=128"`    // Provider display name
	Protocol    string   `json:"protocol" binding:"required,max=64"` // Protocol type: openai / anthropic / ...
	URL         string   `json:"url"`                                // Provider API base URL
	ApiKeys     []string `json:"api_keys"`                           // Upstream API key list
	Enabled     int      `json:"enabled"`                            // Enable status: 0-disabled, 1-enabled
	Description string   `json:"description"`                        // Description
}

func (p *ProviderForm) Validate() error {
	return nil
}

func (p *ProviderForm) FillTo(provider *Provider) error {
	provider.Code = p.Code
	provider.Name = p.Name
	provider.Protocol = p.Protocol
	provider.URL = p.URL
	if len(p.ApiKeys) > 0 {
		b, _ := json.Marshal(p.ApiKeys)
		provider.ApiKeys = json.RawMessage(b)
	} else {
		provider.ApiKeys = nil
	}
	provider.Enabled = p.Enabled
	provider.Description = p.Description
	return nil
}

// GetApiKeys deserializes the JSON api_keys field into a string slice.
func (p *Provider) GetApiKeys() []string {
	if len(p.ApiKeys) == 0 {
		return nil
	}
	var keys []string
	_ = json.Unmarshal(p.ApiKeys, &keys)
	return keys
}

// FetchModelsForm defines the request form for fetching models from upstream provider.
type FetchModelsForm struct {
	BaseURL string `json:"base_url"` // Upstream base URL, e.g., https://api.openai.com (optional if provider has url)
	APIKey  string `json:"api_key"`  // API key to use (optional, defaults to provider's first key)
}

func (f *FetchModelsForm) Validate() error {
	return nil
}

// UpstreamModel represents a single model returned by the upstream /v1/models API.
type UpstreamModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

// FetchModelsResult defines the response for the fetch models API.
type FetchModelsResult struct {
	Models []UpstreamModel `json:"models"`
}
