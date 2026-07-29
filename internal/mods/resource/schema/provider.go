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
	URL         string          `json:"url" gorm:"type:varchar(1024);default:null;comment:供应商 API 基础地址;"`
	AuthType    string          `json:"auth_type" gorm:"type:varchar(64);default:'api_key';comment:认证类型: api_key, oauth_token;"`
	ApiKeys     json.RawMessage `json:"api_keys,omitempty" gorm:"type:json;default:null;comment:上游API认证密钥列表;"`
	OAuth       json.RawMessage `json:"oauth,omitempty" gorm:"type:json;default:null;comment:OAuth 凭证(refresh_token/token_endpoint/expires_at);"`
	LockOwner   string          `json:"-" gorm:"type:varchar(128);default:null;comment:OAuth 刷新分布式锁持有者实例ID;"`
	LockedUntil *time.Time      `json:"-" gorm:"type:datetime;default:null;comment:OAuth 刷新分布式锁过期时间;"`
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

// ApiKeyItem represents a single API key with optional description.
type ApiKeyItem struct {
	Value       string `json:"value"`
	Description string `json:"description"`
}

// OAuthCredential holds the OAuth secrets and refresh metadata for a provider,
	// persisted as a single JSON column. access_token itself lives in ApiKeys; the
	// upstream base URL lives in the Provider.URL column.
	type OAuthCredential struct {
		RefreshToken  string     `json:"refresh_token,omitempty"`
		TokenEndpoint string     `json:"token_endpoint,omitempty"`
		ExpiresAt     *time.Time `json:"expires_at,omitempty"`
		// AccountID is Codex/ChatGPT account id used as Chatgpt-Account-Id header.
		AccountID string `json:"account_id,omitempty"`
		// Email is the upstream account email snapshot (non-secret).
		Email string `json:"email,omitempty"`
		// SubscriptionActiveUntil is Codex plan renewal time display/snapshot
		// (e.g. "2026/8/28 01:21:08"), extracted from id_token claims.
		SubscriptionActiveUntil string `json:"subscription_active_until,omitempty"`
	}

	// OAuthCompleteForm is the request body for finishing authorization-code OAuth
	// flows (e.g. Codex) by submitting the browser callback URL.
	type OAuthCompleteForm struct {
		Provider    string `json:"provider" binding:"required"`
		State       string `json:"state" binding:"required"`
		CallbackURL string `json:"callback_url" binding:"required"`
	}

	func (f *OAuthCompleteForm) Validate() error {
		return nil
	}

// ProviderForm defines the form for creating/updating a Provider.
type ProviderForm struct {
	Code        string       `json:"code" binding:"required,max=128"`    // Provider unique code
	Name        string       `json:"name" binding:"required,max=128"`    // Provider display name
	Protocol    string       `json:"protocol" binding:"required,max=64"` // Protocol type: openai / anthropic / ...
	URL         string       `json:"url"`                                // Provider API base URL
	ApiKeys     []ApiKeyItem `json:"api_keys"`                           // Upstream API key list
	AuthType    string       `json:"auth_type"`                          // Auth type: api_key, oauth_token
	OAuth       *OAuthCredential `json:"oauth"`                          // OAuth credential (refresh_token/token_endpoint/expires_at)
	Enabled     int          `json:"enabled"`                            // Enable status: 0-disabled, 1-enabled
	Description string       `json:"description"`                        // Description
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
	if p.AuthType == "" {
		provider.AuthType = "api_key"
	} else {
		provider.AuthType = p.AuthType
	}
	if p.OAuth != nil {
		b, _ := json.Marshal(p.OAuth)
		provider.OAuth = json.RawMessage(b)
	} else {
		provider.OAuth = nil
	}
	provider.Enabled = p.Enabled
	provider.Description = p.Description
	return nil
}

// GetApiKeys deserializes the JSON api_keys field into an ApiKeyItem slice.
func (p *Provider) GetApiKeys() []ApiKeyItem {
	if len(p.ApiKeys) == 0 {
		return nil
	}
	var keys []ApiKeyItem
	_ = json.Unmarshal(p.ApiKeys, &keys)
	return keys
}

// GetOAuth deserializes the JSON oauth field into an OAuthCredential.
func (p *Provider) GetOAuth() *OAuthCredential {
	if len(p.OAuth) == 0 {
		return nil
	}
	var cred OAuthCredential
	if err := json.Unmarshal(p.OAuth, &cred); err != nil {
		return nil
	}
	return &cred
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
