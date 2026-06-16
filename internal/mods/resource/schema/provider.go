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
	ID          string          `json:"id" gorm:"size:20;primarykey;"`                                                       // Unique ID (XID)
	Code        string          `json:"code" gorm:"size:128;uniqueIndex:uniq_provider_code,priority:1;"`                     // Provider unique code, e.g., openai-official
	Name        string          `json:"name" gorm:"size:128;not null;"`                                                      // Provider display name
	Protocol    string          `json:"protocol" gorm:"size:64;not null;"`                                                   // Protocol type: openai / anthropic / ...
	URL         string          `json:"url" gorm:"size:512;"`                                                                // Provider API base URL, e.g., https://api.openai.com
	ApiKeys     json.RawMessage `json:"api_keys,omitempty" gorm:"type:json;"`                                                // Upstream API key list
	Enabled     int             `json:"enabled" gorm:"not null;default:0;"`                                                  // Enable status: 0-disabled, 1-enabled
	Description string          `json:"description" gorm:"size:255;"`                                                        // Description
	Creator     string          `json:"creator" gorm:"size:255;"`                                                            // Creator
	Modifier    string          `json:"modifier" gorm:"size:255;"`                                                           // Modifier
	CreatedAt   time.Time       `json:"created_at" gorm:"type:timestamp;autoCreateTime;"`                                                             // Create time
	UpdatedAt   time.Time       `json:"updated_at" gorm:"type:timestamp;autoUpdateTime;"`                                                             // Update time
	Deleted     string          `json:"-" gorm:"size:20;uniqueIndex:uniq_provider_code,priority:2;default:0"`                // Logical delete flag
	DeletedAt   *gorm.DeletedAt `json:"-" gorm:"type:datetime;comment:Delete time;"`                                         // Delete time
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
