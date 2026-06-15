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
	ID                          string          `json:"id" gorm:"size:20;primaryKey;<-:create;comment:Unique ID;"`                                              // Unique ID
	Name                        string          `json:"name" gorm:"size:128;not null;uniqueIndex:uniq_policy_circuit_break_name;comment:Policy name;"`          // Policy name
	Level                       string          `json:"level" gorm:"size:64;not null;default:SERVICE;comment:Policy level;"`                                    // Policy level
	SlidingWindowType           string          `json:"sliding_window_type" gorm:"size:16;not null;default:count;comment:Sliding window type;"`                 // Sliding window type
	SlidingWindowSize           int             `json:"sliding_window_size" gorm:"not null;default:20;comment:Sliding window size;"`                            // Sliding window size
	MinCallsThreshold           int             `json:"min_calls_threshold" gorm:"not null;default:5;comment:Min calls threshold;"`                             // Min calls threshold
	FailureRateThreshold        float64         `json:"failure_rate_threshold" gorm:"type:decimal(5,2);not null;default:50.00;comment:Failure rate threshold;"` // Failure rate threshold
	SlowCallRateThreshold       float64         `json:"slow_call_rate_threshold" gorm:"type:decimal(5,2);comment:Slow call rate threshold;"`                    // Slow call rate threshold
	SlowCallDurationThreshold   int             `json:"slow_call_duration_threshold" gorm:"comment:Slow call duration threshold;"`                              // Slow call duration threshold
	SlowCallMetric              string          `json:"slow_call_metric" gorm:"size:32;comment:Slow call metric;"`                                              // Slow call metric
	WaitDurationInOpenState     int             `json:"wait_duration_in_open_state" gorm:"not null;default:10000;comment:Wait duration in open state;"`         // Wait duration in open state
	AllowedCallsInHalfOpenState int             `json:"allowed_calls_in_half_open_state" gorm:"not null;default:3;comment:Allowed calls in half open state;"`   // Allowed calls in half open state
	ForceOpen                   int             `json:"force_open" gorm:"not null;default:0;comment:Force open;"`                                               // Force open
	OutlierMaxPercent           int             `json:"outlier_max_percent" gorm:"not null;default:10;comment:Outlier max percent;"`                            // Outlier max percent
	CodePolicy                  *string         `json:"code_policy,omitempty" gorm:"type:json;comment:Code policy (JSON);"`                                     // Code policy (JSON)
	MessagePolicy               *string         `json:"message_policy,omitempty" gorm:"type:json;comment:Message policy (JSON);"`                               // Message policy (JSON)
	DegradeConfig               *string         `json:"degrade_config,omitempty" gorm:"type:json;comment:Degrade config (JSON);"`                               // Degrade config (JSON)
	ErrorCodes                  *string         `json:"error_codes,omitempty" gorm:"type:json;comment:Error codes (JSON);"`                                     // Error codes (JSON)
	ErrorMessages               *string         `json:"error_messages,omitempty" gorm:"type:json;comment:Error messages (JSON);"`                               // Error messages (JSON)
	Version                     int64           `json:"version" gorm:"not null;default:1;comment:Version;"`                                                     // Version
	Enabled                     int             `json:"enabled" gorm:"not null;default:0;comment:Enabled;"`                                                     // Enabled
	Description                 *string         `json:"description,omitempty" gorm:"size:255;comment:Details;"`                                                 // Details
	Creator                     *string         `json:"creator,omitempty" gorm:"size:255;comment:Creator;"`                                                     // Creator
	Modifier                    *string         `json:"modifier,omitempty" gorm:"size:255;comment:Modifier;"`                                                   // Modifier
	CreatedAt                   time.Time       `json:"created_at" gorm:"autoCreateTime;comment:Create timestamp;"`                                             // Create timestamp
	UpdatedAt                   time.Time       `json:"updated_at,omitempty" gorm:"autoUpdateTime;comment:Update timestamp;"`                                   // Update timestamp
	Deleted                     string          `json:"-" gorm:"uniqueIndex:uniq_policy_circuit_break_name;size:20;default:0;comment:Delete flag;"`             // Delete flag
	DeletedAt                   *gorm.DeletedAt `json:"-" gorm:"comment:Delete timestamp;"`                                                                     // Delete timestamp
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
