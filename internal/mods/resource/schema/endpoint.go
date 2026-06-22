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
	ID                 string          `json:"id" gorm:"type:char(20);primaryKey;comment:主键ID (XID);"`
	ModelID            string          `json:"model_id" gorm:"type:char(20);not null;index:idx_endpoint_route,priority:1;index:idx_model_id,priority:1;comment:关联 of model ID;"`
	ProviderID         string          `json:"provider_id" gorm:"type:char(20);not null;index:idx_endpoint_route,priority:2;index:idx_provider_id,priority:1;comment:关联 of provider ID;"`
	URL                string          `json:"url" gorm:"type:varchar(512);not null;comment:上游 API 地址;"`
	ApiKey             string          `json:"api_key,omitempty" gorm:"type:varchar(512);default:null;comment:可选，覆盖 provider 级别的 api_key;"`
	Protocol           string          `json:"protocol,omitempty" gorm:"type:varchar(64);default:null;comment:可选，覆盖 provider 级别的 protocol;"`
	RealModel          string          `json:"real_model,omitempty" gorm:"type:varchar(128);default:null;comment:可选，覆盖 model 级别的 real_model;"`
	Priority           int             `json:"priority" gorm:"type:int;not null;default:0;comment:故障转移顺序，数字越小越优先;"`
	Weight             int             `json:"weight" gorm:"type:int;not null;default:1;comment:负载均衡权重;"`
	Enabled            int             `json:"enabled" gorm:"type:int;not null;default:0;comment:启用状态: 0-未启用，1-启用;"`
	Headers            json.RawMessage `json:"headers,omitempty" gorm:"type:json;default:null;comment:自定义请求头，如 {\"X-Custom-Header\": \"value\"};"`
	Metadata           json.RawMessage `json:"metadata,omitempty" gorm:"type:json;default:null;comment:元数据，用于存储标签等额外信息;"`
	InputPrice         *float64        `json:"input_price" gorm:"type:decimal(10,6);default:null;comment:输入价格（元/百万 Tokens），NULL表示继承模型;"`
	OutputPrice        *float64        `json:"output_price" gorm:"type:decimal(10,6);default:null;comment:输出价格（元/百万 Tokens），NULL表示继承模型;"`
	CachedPrice        *float64        `json:"cached_price" gorm:"type:decimal(10,6);default:null;comment:缓存命中价格（元/百万 Tokens），NULL表示继承模型;"`
	CacheCreationPrice *float64        `json:"cache_creation_price" gorm:"type:decimal(10,6);default:null;comment:缓存创建价格（元/百万 Tokens），NULL表示继承模型;"`
	Description        string          `json:"description" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	Creator            string          `json:"creator" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier           string          `json:"modifier" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt          time.Time       `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt          time.Time       `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted            string          `json:"-" gorm:"type:varchar(20);not null;default:'0';index:idx_provider_id,priority:2;index:idx_model_id,priority:2;comment:逻辑删除标识;"`
	DeletedAt          *gorm.DeletedAt `json:"-" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
	StatusPoints       []StatusPoint   `json:"status_points" gorm:"-"`                                                                                   // Recent status points

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
