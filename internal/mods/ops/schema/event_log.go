package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Event types
const (
	EventTypeCircuitBreak   = "circuit_break"
	EventTypeRateLimit      = "rate_limit"
	EventTypeInvocationFail = "invocation_fail"
	EventTypeLBSwitch       = "lb_switch"
)

// EventLog records policy execution events from the AI Gateway.
type EventLog struct {
	ID           string    `json:"id" gorm:"size:20;primaryKey;<-:create;comment:Unique ID;"`
	EventType    string    `json:"event_type" gorm:"size:32;not null;index:idx_el_event_type;comment:Event type;"`
	TenantCode   string    `json:"tenant_code" gorm:"size:64;not null;default:'';index:idx_el_tenant;comment:Tenant code;"`
	ModelCode    string    `json:"model_code" gorm:"size:64;not null;default:'';index:idx_el_model;comment:Model code;"`
	EndpointID   string    `json:"endpoint_id" gorm:"size:20;not null;default:'';index:idx_el_endpoint;comment:Endpoint ID;"`
	EndpointCode string    `json:"endpoint_code" gorm:"size:128;not null;default:'';index:idx_el_endpoint_code;comment:Endpoint code;"`
	ProviderName string    `json:"provider_name" gorm:"size:128;not null;default:'';comment:Provider name;"`
	PolicyID     string    `json:"policy_id" gorm:"size:20;not null;default:'';index:idx_el_policy;comment:Policy ID;"`
	PolicyName   string    `json:"policy_name" gorm:"size:128;not null;default:'';comment:Policy name;"`
	Threshold    *float64  `json:"threshold" gorm:"type:decimal(10,2);comment:Threshold value;"`
	CurrentValue *float64  `json:"current_value" gorm:"type:decimal(10,2);comment:Current value at trigger;"`
	RequestID    string    `json:"request_id" gorm:"size:64;not null;default:'';comment:Request ID;"`
	TraceID      string    `json:"trace_id" gorm:"size:64;not null;default:'';comment:Trace ID;"`
	Message      string    `json:"message" gorm:"size:10240;not null;default:'';comment:Human-readable message;"`
	EventTime    time.Time `json:"event_time" gorm:"type:datetime;not null;index:idx_el_time;comment:Event timestamp from gateway;"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime;comment:Ingest time;"`
}

func (EventLog) TableName() string {
	return config.C.FormatTableName("event_log")
}

// EventQueryParam defines query parameters for event list.
type EventQueryParam struct {
	util.PaginationParam
	EventType    string `form:"event_type"`    // Filter by event type
	TenantCode   string `form:"tenant_code"`   // Filter by tenant code
	ModelCode    string `form:"model_code"`    // Filter by model code
	ProviderName string `form:"provider_name"` // Filter by provider name
	EndpointID   string `form:"endpoint_id"`   // Filter by endpoint ID
	EndpointCode string `form:"endpoint_code"` // Filter by endpoint code
	PolicyID     string `form:"policy_id"`     // Filter by policy ID
	StartTime    string `form:"start_time"`    // Filter by start time (ISO 8601 / unix)
	EndTime      string `form:"end_time"`      // Filter by end time (ISO 8601 / unix)
}

// EventQueryOptions defines query options.
type EventQueryOptions struct {
	util.QueryOptions
}

// EventQueryResult defines query result.
type EventQueryResult struct {
	Data       EventLogs
	PageResult *util.PaginationResult
}

// EventLogs is a slice of EventLog.
type EventLogs []*EventLog

// EventTypeCount represents event count by type.
type EventTypeCount struct {
	EventType string `json:"event_type"`
	Count     int64  `json:"count"`
}

// TrendPoint represents a point in the trend chart.
type TrendPoint struct {
	Time           string `json:"time"`
	CircuitBreak   int64  `json:"circuit_break"`
	RateLimit      int64  `json:"rate_limit"`
	InvocationFail int64  `json:"invocation_fail"`
	LBSwitch       int64  `json:"lb_switch"`
}

// RankingItem represents an item in tenant/model ranking.
type RankingItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// EventStatistics represents the full statistics response.
type EventStatistics struct {
	TotalEvents         int64         `json:"total_events"`
	CircuitBreakCount   int64         `json:"circuit_break_count"`
	RateLimitCount      int64         `json:"rate_limit_count"`
	InvocationFailCount int64         `json:"invocation_fail_count"`
	LBSwitchCount       int64         `json:"lb_switch_count"`
	Trend               []TrendPoint  `json:"trend"`
	TenantRanking       []RankingItem `json:"tenant_ranking"`
	ModelRanking        []RankingItem `json:"model_ranking"`
}
