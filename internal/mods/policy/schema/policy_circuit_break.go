package schema

import (
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/encoding/json"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Circuit break policy management
type PolicyCircuitBreak struct {
	ID                          string          `json:"id" gorm:"type:char(20);primaryKey;<-:create;comment:主键ID (XID);"`
	Name                        string          `json:"name" gorm:"type:varchar(128);not null;uniqueIndex:uniq_policy_circuit_break_name;comment:策略名称;"`
	Level                       string          `json:"level" gorm:"type:varchar(64);not null;default:'INSTANCE';comment:熔断隔离级别：SERVICE / INSTANCE;"`
	SlidingWindowType           string          `json:"sliding_window_type" gorm:"type:varchar(16);not null;default:'time';comment:滑动窗口类型：time / count;"`
	SlidingWindowSize           int             `json:"sliding_window_size" gorm:"type:int;not null;default:20;comment:滑动窗口大小（次数或秒数）;"`
	MinCallsThreshold           int             `json:"min_calls_threshold" gorm:"type:int;not null;default:5;comment:熔断计算的最小调用次数;"`
	FailureRateThreshold        float64         `json:"failure_rate_threshold" gorm:"type:decimal(5,2);not null;default:50.00;comment:失败率阈值百分比;"`
	SlowCallRateThreshold       float64         `json:"slow_call_rate_threshold" gorm:"type:decimal(5,2);default:null;comment:慢调用率阈值百分比;"`
	SlowCallDurationThreshold   int             `json:"slow_call_duration_threshold" gorm:"type:int;default:null;comment:慢调用时长阈值（毫秒）;"`
	SlowCallMetric              string          `json:"slow_call_metric" gorm:"type:varchar(32);default:null;comment:慢调用衡量指标：TTFT 等;"`
	WaitDurationInOpenState     int             `json:"wait_duration_in_open_state" gorm:"type:int;not null;default:10000;comment:熔断器开启状态持续时间（毫秒）;"`
	AllowedCallsInHalfOpenState int             `json:"allowed_calls_in_half_open_state" gorm:"type:int;not null;default:3;comment:半开状态下允许的试探调用次数;"`
	ForceOpen                   int             `json:"force_open" gorm:"type:tinyint(1);not null;default:0;comment:是否强制开启熔断：0-否，1-是;"`
	OutlierMaxPercent           int             `json:"outlier_max_percent" gorm:"type:int;not null;default:10;comment:最大熔断实例比例百分比(对 INSTANCE 级有效);"`
	CodePolicy                  *string         `json:"code_policy,omitempty" gorm:"type:json;default:null;comment:响应状态码提取解析策略;"`
	MessagePolicy               *string         `json:"message_policy,omitempty" gorm:"type:json;default:null;comment:错误消息提取解析策略;"`
	DegradeConfig               *string         `json:"degrade_config,omitempty" gorm:"type:json;default:null;comment:熔断降级响应配置，嵌套 DegradeConfig 结构;"`
	ErrorCodes                  *string         `json:"error_codes,omitempty" gorm:"type:json;default:null;comment:触发熔断的异常状态码列表;"`
	ErrorMessages               *string         `json:"error_messages,omitempty" gorm:"type:json;default:null;comment:错误消息列表;"`
	Version                     int64           `json:"version" gorm:"type:bigint;not null;default:1;comment:策略版本;"`
	Enabled                     int             `json:"enabled" gorm:"type:int;not null;default:0;comment:启用状态: 0-未启用，1-启用;"`
	Description                 *string         `json:"description,omitempty" gorm:"type:varchar(255);default:null;comment:备注描述;"`
	Creator                     *string         `json:"creator,omitempty" gorm:"type:varchar(255);default:null;comment:创建者;"`
	Modifier                    *string         `json:"modifier,omitempty" gorm:"type:varchar(255);default:null;comment:修改者;"`
	CreatedAt                   time.Time       `json:"created_at" gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;autoCreateTime;comment:创建时间;"`
	UpdatedAt                   time.Time       `json:"updated_at,omitempty" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime;comment:更新时间;"`
	Deleted                     string          `json:"-" gorm:"type:varchar(20);not null;default:'0';comment:逻辑删除标识;"`
	DeletedAt                   *gorm.DeletedAt `json:"-" gorm:"type:datetime;default:null;comment:逻辑删除时间;"`
}

func (a PolicyCircuitBreak) TableName() string {
	return config.C.FormatTableName("policy_circuit_break")
}

// ConvertTo Convert `PolicyCircuitBreak` to `PolicyCircuitBreakForm` object.
func (a PolicyCircuitBreak) ConvertTo(form *PolicyCircuitBreakForm) error {
	form.ID = a.ID
	form.Name = a.Name
	form.Level = a.Level
	form.SlidingWindowType = a.SlidingWindowType
	form.SlidingWindowSize = a.SlidingWindowSize
	form.MinCallsThreshold = a.MinCallsThreshold
	form.OutlierMaxPercent = a.OutlierMaxPercent
	if !util.IsNilOrEmpty(a.CodePolicy) {
		cp := new(ErrorParserPolicy)
		json.UnMarshalToObject(*a.CodePolicy, cp)
		form.CodePolicy = cp
	}
	if !util.IsNilOrEmpty(a.MessagePolicy) {
		mp := new(ErrorParserPolicy)
		json.UnMarshalToObject(*a.MessagePolicy, mp)
		form.MessagePolicy = mp
	}
	if !util.IsNilOrEmpty(a.ErrorCodes) {
		ec := make([]string, 0)
		json.UnMarshalToObject(*a.ErrorCodes, &ec)
		form.ErrorCodes = ec
	}
	if !util.IsNilOrEmpty(a.ErrorMessages) {
		em := make([]string, 0)
		json.UnMarshalToObject(*a.ErrorMessages, &em)
		form.ErrorMessages = em
	}
	form.FailureRateThreshold = a.FailureRateThreshold
	form.SlowCallRateThreshold = a.SlowCallRateThreshold
	form.SlowCallDurationThreshold = a.SlowCallDurationThreshold
	if a.SlowCallDurationThreshold > 0 || a.SlowCallRateThreshold > 0 {
		if a.SlowCallMetric != "" {
			form.SlowCallMetric = a.SlowCallMetric
		} else {
			form.SlowCallMetric = "TTFT"
		}
	} else {
		form.SlowCallMetric = ""
	}
	form.WaitDurationInOpenState = a.WaitDurationInOpenState
	form.AllowedCallsInHalfOpenState = a.AllowedCallsInHalfOpenState
	form.ForceOpen = a.ForceOpen
	if !util.IsNilOrEmpty(a.DegradeConfig) {
		dc := new(DegradeConfig)
		json.UnMarshalToObject(*a.DegradeConfig, dc)
		form.DegradeConfig = dc
	}
	form.Version = a.Version
	form.Enabled = a.Enabled
	form.Description = a.Description
	form.Creator = a.Creator
	form.Modifier = a.Modifier
	form.CreatedAt = a.CreatedAt
	form.UpdatedAt = a.UpdatedAt
	return nil
}

// Defining the query parameters for the `PolicyCircuitBreak` struct.
type PolicyCircuitBreakQueryParam struct {
	util.PaginationParam
	Name string `form:"name"` // Policy name (like)
}

// Defining the query options for the `PolicyCircuitBreak` struct.
type PolicyCircuitBreakQueryOptions struct {
	util.QueryOptions
}

// Defining the query result for the `PolicyCircuitBreak` struct.
type PolicyCircuitBreakQueryResult struct {
	Data       PolicyCircuitBreaks
	PageResult *util.PaginationResult
}

// Defining the slice of `PolicyCircuitBreak` struct.
type PolicyCircuitBreaks []*PolicyCircuitBreak

// Defining the data structure for creating a `PolicyCircuitBreak` struct.
type PolicyCircuitBreakForm struct {
	ID                          string             `json:"id"`
	Name                        string             `json:"name" binding:"required,max=128"`               // Policy name
	Level                       string             `json:"level" binding:"required,max=64"`               // Policy level
	SlidingWindowType           string             `json:"sliding_window_type" binding:"required,max=16"` // Sliding window type
	SlidingWindowSize           int                `json:"sliding_window_size"`                           // Sliding window size
	MinCallsThreshold           int                `json:"min_calls_threshold"`                           // Min calls threshold
	CodePolicy                  *ErrorParserPolicy `json:"code_policy"`                                   // Code policy
	MessagePolicy               *ErrorParserPolicy `json:"message_policy"`                                // Message policy
	ErrorCodes                  []string           `json:"error_codes"`                                   // Error codes
	ErrorMessages               []string           `json:"error_messages"`                                // Error messages
	FailureRateThreshold        float64            `json:"failure_rate_threshold"`                        // Failure rate threshold
	SlowCallRateThreshold       float64            `json:"slow_call_rate_threshold"`                      // Slow call rate threshold
	SlowCallDurationThreshold   int                `json:"slow_call_duration_threshold"`                  // Slow call duration threshold
	SlowCallMetric              string             `json:"slow_call_metric"`                              // Slow call metric
	WaitDurationInOpenState     int                `json:"wait_duration_in_open_state"`                   // Wait duration in open state
	AllowedCallsInHalfOpenState int                `json:"allowed_calls_in_half_open_state"`              // Allowed calls in half open state
	ForceOpen                   int                `json:"force_open"`                                    // Force open
	OutlierMaxPercent           int                `json:"outlier_max_percent"`                           // Outlier max percent
	DegradeConfig               *DegradeConfig     `json:"degrade_config"`                                // Degrade config
	Version                     int64              `json:"version"`                                       // Version
	Enabled                     int                `json:"enabled"`                                       // Enabled
	Description                 *string            `json:"description"`                                   // Details
	Creator                     *string            `json:"creator,omitempty"`                             // Creator
	Modifier                    *string            `json:"modifier,omitempty"`                            // Modifier
	CreatedAt                   time.Time          `json:"created_at"`                                    // Create timestamp
	UpdatedAt                   time.Time          `json:"updated_at,omitempty"`                          // Update timestamp
}

// A validation function for the `PolicyCircuitBreakForm` struct.
func (a *PolicyCircuitBreakForm) Validate() error {
	if a.Name == "" {
		return errors.BadRequest("", "Name is required")
	}
	if a.Level == "" {
		return errors.BadRequest("", "Level is required")
	}
	if a.SlidingWindowType == "" {
		return errors.BadRequest("", "SlidingWindowType is required")
	}
	if a.Level != "SERVICE" && a.Level != "INSTANCE" {
		return errors.BadRequest("", "Level must be either SERVICE or INSTANCE")
	}
	return nil
}

// Convert `PolicyCircuitBreakForm` to `PolicyCircuitBreak` object.
func (a *PolicyCircuitBreakForm) FillTo(policyCircuitBreak *PolicyCircuitBreak) error {
	policyCircuitBreak.Name = a.Name
	policyCircuitBreak.Level = a.Level
	policyCircuitBreak.SlidingWindowType = a.SlidingWindowType
	policyCircuitBreak.SlidingWindowSize = a.SlidingWindowSize
	policyCircuitBreak.MinCallsThreshold = a.MinCallsThreshold
	policyCircuitBreak.OutlierMaxPercent = a.OutlierMaxPercent
	policyCircuitBreak.CodePolicy = func() *string { return json.MarshalToString(a.CodePolicy) }()
	policyCircuitBreak.MessagePolicy = func() *string { return json.MarshalToString(a.MessagePolicy) }()
	policyCircuitBreak.ErrorCodes = func() *string { return json.MarshalToString(a.ErrorCodes) }()
	policyCircuitBreak.ErrorMessages = func() *string { return json.MarshalToString(a.ErrorMessages) }()
	policyCircuitBreak.FailureRateThreshold = a.FailureRateThreshold
	policyCircuitBreak.SlowCallRateThreshold = a.SlowCallRateThreshold
	policyCircuitBreak.SlowCallDurationThreshold = a.SlowCallDurationThreshold
	if a.SlowCallDurationThreshold > 0 || a.SlowCallRateThreshold > 0 {
		if a.SlowCallMetric != "" {
			policyCircuitBreak.SlowCallMetric = a.SlowCallMetric
		} else {
			policyCircuitBreak.SlowCallMetric = "TTFT"
		}
	} else {
		policyCircuitBreak.SlowCallMetric = ""
	}
	policyCircuitBreak.WaitDurationInOpenState = a.WaitDurationInOpenState
	policyCircuitBreak.AllowedCallsInHalfOpenState = a.AllowedCallsInHalfOpenState
	policyCircuitBreak.ForceOpen = a.ForceOpen
	policyCircuitBreak.DegradeConfig = func() *string { return json.MarshalToString(a.DegradeConfig) }()
	policyCircuitBreak.Enabled = a.Enabled
	policyCircuitBreak.Description = a.Description
	policyCircuitBreak.Version = time.Now().UnixMilli()
	return nil
}
