package schema

import (
	"encoding/json"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Endpoint defines an upstream endpoint belonging to a provider.
type Endpoint struct {
	ID                 string          `json:"id" gorm:"size:20;primarykey;"`                                                                                            // Unique ID (XID)
	ProviderID         string          `json:"provider_id" gorm:"size:20;not null;index:idx_provider_id,priority:1;index:idx_endpoint_route,priority:2"`                 // Associated provider ID
	ModelID            string          `json:"model_id" gorm:"size:20;not null;index:idx_model_id,priority:1;index:idx_endpoint_route,priority:1;"`                      // Associated Model ID
	URL                string          `json:"url" gorm:"size:512;not null;"`                                                                                            // Upstream API address
	ApiKey             string          `json:"api_key,omitempty" gorm:"size:512;"`                                                                                       // Optional, overrides provider-level api_key
	Protocol           string          `json:"protocol,omitempty" gorm:"size:64;"`                                                                                       // Optional, overrides provider-level protocol
	RealModel          string          `json:"real_model,omitempty" gorm:"size:128;"`                                                                                    // Optional, overrides model-level real_model
	Priority           int             `json:"priority" gorm:"not null;default:0;"`                                                                                      // Failover priority (higher = preferred)
	Weight             int             `json:"weight" gorm:"not null;default:1;"`                                                                                        // Load balancing weight
	Enabled            int             `json:"enabled" gorm:"not null;default:0;"`                                                                                       // Enable status: 0-disabled, 1-enabled
	Headers            json.RawMessage `json:"headers,omitempty" gorm:"type:json;"`                                                                                      // Custom HTTP headers
	Metadata           json.RawMessage `json:"metadata,omitempty" gorm:"type:json;"`                                                                                     // Metadata for tags etc.
	InputPrice         *float64        `json:"input_price" gorm:"column:input_price;type:decimal(10,6);default:null;"`                                                   // Input price (CNY/M Tokens), NULL means inherit model
	OutputPrice        *float64        `json:"output_price" gorm:"column:output_price;type:decimal(10,6);default:null;"`                                                 // Output price (CNY/M Tokens), NULL means inherit model
	CachedPrice        *float64        `json:"cached_price" gorm:"column:cached_price;type:decimal(10,6);default:null;"`                                                 // Cached price (CNY/M Tokens), NULL means inherit model
	CacheCreationPrice *float64        `json:"cache_creation_price" gorm:"column:cache_creation_price;type:decimal(10,6);default:null;"`                                 // Cache creation price (CNY/M Tokens), NULL means inherit model
	Description        string          `json:"description" gorm:"size:255;"`                                                                                             // Description
	Creator            string          `json:"creator" gorm:"size:255;"`                                                                                                 // Creator
	Modifier           string          `json:"modifier" gorm:"size:255;"`                                                                                                // Modifier
	CreatedAt          time.Time       `json:"created_at" gorm:"type:timestamp;autoCreateTime;"`                                                                              // Create time
	UpdatedAt          time.Time       `json:"updated_at" gorm:"type:timestamp;autoUpdateTime;"`                                                                              // Update time
	Deleted            string          `json:"-" gorm:"size:20;index:idx_provider_id,priority:2;index:idx_model_id,priority:2;default:0"`                               // Logical delete flag
	DeletedAt          *gorm.DeletedAt `json:"-" gorm:"type:datetime;comment:Delete time;"`                                                                              // Delete time
	StatusPoints       []StatusPoint   `json:"status_points" gorm:"-"`                                                                                                   // Recent status points

	// 关联查询
	Model    *Model    `json:"model,omitempty" gorm:"foreignKey:ModelID;references:ID"`
	Provider *Provider `json:"provider,omitempty" gorm:"foreignKey:ProviderID;references:ID"`
}

func (e *Endpoint) TableName() string {
	return config.C.FormatTableName("endpoint")
}

// EndpointQueryParam defines the query parameters for Endpoint.
type EndpointQueryParam struct {
	util.PaginationParam
	ProviderID string `form:"provider_id"` // Filter by provider ID
	ModelID    string `form:"model_id"`    // Filter by model ID
	LikeURL    string `form:"url"`         // URL (like)
	Priority   int    `form:"priority"`    // Filter by priority
}

// EndpointQueryOptions defines the query options for Endpoint.
type EndpointQueryOptions struct {
	util.QueryOptions
}

// EndpointQueryResult defines the query result for Endpoint.
type EndpointQueryResult struct {
	Data       Endpoints
	PageResult *util.PaginationResult
}

// Endpoints defines a slice of Endpoint.
type Endpoints []*Endpoint

// EndpointForm defines the form for creating/updating an Endpoint.
type EndpointForm struct {
	ProviderID         string          `json:"provider_id" binding:"required,max=20"` // Associated provider ID
	ModelID            string          `json:"model_id" binding:"required,max=20"`    // Associated Model ID
	URL                string          `json:"url" binding:"required,max=512"`        // Upstream API address
	ApiKey             string          `json:"api_key"`                               // Optional, overrides provider-level api_key
	Protocol           string          `json:"protocol"`                              // Optional, overrides provider-level protocol
	RealModel          string          `json:"real_model"`                            // Optional, overrides model-level real_model
	Priority           int             `json:"priority"`                              // Failover priority
	Weight             int             `json:"weight"`                                // Load balancing weight
	Enabled            int             `json:"enabled"`                               // Enable status: 0-disabled, 1-enabled
	Headers            json.RawMessage `json:"headers"`                               // Custom HTTP headers
	Metadata           json.RawMessage `json:"metadata"`                              // Metadata
	InputPrice         *float64        `json:"input_price"`                           // Input price (CNY/M Tokens)
	OutputPrice        *float64        `json:"output_price"`                          // Output price (CNY/M Tokens)
	CachedPrice        *float64        `json:"cached_price"`                          // Cached price (CNY/M Tokens)
	CacheCreationPrice *float64        `json:"cache_creation_price"`                  // Cache creation price (CNY/M Tokens)
	Description        string          `json:"description"`                           // Description
}

func (e *EndpointForm) Validate() error {
	return nil
}

func (e *EndpointForm) FillTo(endpoint *Endpoint) error {
	endpoint.ProviderID = e.ProviderID
	endpoint.ModelID = e.ModelID
	endpoint.URL = e.URL
	endpoint.ApiKey = e.ApiKey
	endpoint.Protocol = e.Protocol
	endpoint.RealModel = e.RealModel
	endpoint.Priority = e.Priority
	endpoint.Weight = e.Weight
	endpoint.Enabled = e.Enabled
	if len(e.Headers) == 0 || string(e.Headers) == "null" {
		endpoint.Headers = nil
	} else {
		endpoint.Headers = e.Headers
	}
	if len(e.Metadata) == 0 || string(e.Metadata) == "null" {
		endpoint.Metadata = nil
	} else {
		endpoint.Metadata = e.Metadata
	}
	endpoint.InputPrice = e.InputPrice
	endpoint.OutputPrice = e.OutputPrice
	endpoint.CachedPrice = e.CachedPrice
	endpoint.CacheCreationPrice = e.CacheCreationPrice
	endpoint.Description = e.Description
	return nil
}

// EndpointTestResult defines the result for the endpoint connectivity test.
type EndpointTestResult struct {
	Success   bool   `json:"success"`
	LatencyMs int64  `json:"latency_ms"`
	Message   string `json:"message"`
	Detail    string `json:"detail"`
}
