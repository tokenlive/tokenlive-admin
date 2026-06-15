package schema

import (
	"encoding/json"
	"strconv"
)

// TagCondition represents a condition with an operation type, type, key, and values.
type TagCondition struct {
	OpType OpType   `json:"op_type"`
	Type   string   `json:"type"`
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

// UnmarshalJSON implements custom JSON unmarshaling to handle opType (camelCase) compatibly.
func (c *TagCondition) UnmarshalJSON(data []byte) error {
	type Alias TagCondition
	aux := &struct {
		OpTypeCamel OpType `json:"opType"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.OpTypeCamel != "" {
		c.OpType = aux.OpTypeCamel
	}
	return nil
}

const (
	TagConditionTypeHeader = "HEADER"
	TagConditionTypeQuery  = "QUERY"
	TagConditionTypeCookie = "COOKIE"
	TagConditionTypeSystem = "SYSTEM"
	TagConditionTypeTag    = "TAG"
)

// OpType defines the operation types for a TagCondition.
type OpType string

const (
	OpType_EQUAL     OpType = "EQUAL"
	OpType_NOT_EQUAL OpType = "NOT_EQUAL"
	OpType_IN        OpType = "IN"
	OpType_NOT_IN    OpType = "NOT_IN"
	OpType_REGULAR   OpType = "REGULAR"
	OpType_PREFIX    OpType = "PREFIX"
)

// TagGroup represents a group of conditions with a relation type and order.
type TagGroup struct {
	RelationType RelationType   `json:"relation_type"`
	Conditions   []TagCondition `json:"conditions"`
	Order        int32          `json:"order"`
}

// UnmarshalJSON implements custom JSON unmarshaling to handle relationType (camelCase) compatibly.
func (g *TagGroup) UnmarshalJSON(data []byte) error {
	type Alias TagGroup
	aux := &struct {
		RelationTypeCamel RelationType `json:"relationType"`
		*Alias
	}{
		Alias: (*Alias)(g),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.RelationTypeCamel != "" {
		g.RelationType = aux.RelationTypeCamel
	}
	return nil
}

// RelationType defines the relation types for a TagGroup.
type RelationType string

const (
	RelationType_AND RelationType = "AND"
	RelationType_OR  RelationType = "OR"
)

// TagDestination represents a destination with weight, relation type, conditions, and order.
type TagDestination struct {
	Weight       int32          `json:"weight"`
	RelationType RelationType   `json:"relation_type"`
	Conditions   []TagCondition `json:"conditions"`
}

// UnmarshalJSON implements custom JSON unmarshaling to handle relationType (camelCase) compatibly.
func (d *TagDestination) UnmarshalJSON(data []byte) error {
	type Alias TagDestination
	aux := &struct {
		RelationTypeCamel RelationType `json:"relationType"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.RelationTypeCamel != "" {
		d.RelationType = aux.RelationTypeCamel
	}
	return nil
}

// TagRule represents a rule with destinations, relation type, conditions, and order.
type TagRule struct {
	TagGroup
	Destinations []TagDestination `json:"destinations"`
}

// Defining the SlidingWindow of `LimitForm` struct.
type SlidingWindow struct {
	Threshold      int      `json:"threshold"`
	TimeWindowInMs int64    `json:"time_window_in_ms"`
	BurstRatio     *float64 `json:"burst_ratio,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling to handle timeWindowInMs (camelCase) and burstRatio compatibly.
func (s *SlidingWindow) UnmarshalJSON(data []byte) error {
	type Alias SlidingWindow
	aux := &struct {
		TimeWindowInMsCamel int64    `json:"timeWindowInMs"`
		BurstRatioCamel     *float64 `json:"burstRatio"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.TimeWindowInMsCamel > 0 {
		s.TimeWindowInMs = aux.TimeWindowInMsCamel
	}
	if aux.BurstRatioCamel != nil {
		s.BurstRatio = aux.BurstRatioCamel
	}
	return nil
}

// ErrorParserPolicy represents an error parser policy with a parser, expression, statuses, and contentTypes.
type ErrorParserPolicy struct {
	Parser       string   `json:"parser"`
	Expression   string   `json:"expression"`
	Statuses     []string `json:"statuses"`
	ContentTypes []string `json:"content_types"`
}

// UnmarshalJSON implements custom JSON unmarshaling to handle contentTypes (camelCase) compatibly.
func (p *ErrorParserPolicy) UnmarshalJSON(data []byte) error {
	type Alias ErrorParserPolicy
	aux := &struct {
		ContentTypesCamel []string `json:"contentTypes"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.ContentTypesCamel) > 0 {
		p.ContentTypes = aux.ContentTypesCamel
	}
	return nil
}

// DegradeConfig represents a degrade configuration.
type DegradeConfig struct {
	ResponseCode int               `json:"response_code"`
	Attributes   map[string]string `json:"attributes"`
	ResponseBody string            `json:"response_body"`
}

// UnmarshalJSON implements custom JSON unmarshaling to handle response_code compatible with string/int and responseCode.
func (d *DegradeConfig) UnmarshalJSON(data []byte) error {
	type Alias DegradeConfig
	aux := &struct {
		ResponseCodeCamel json.RawMessage `json:"responseCode"`
		ResponseCodeSnake json.RawMessage `json:"response_code"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var rawCode json.RawMessage
	if len(aux.ResponseCodeSnake) > 0 {
		rawCode = aux.ResponseCodeSnake
	} else if len(aux.ResponseCodeCamel) > 0 {
		rawCode = aux.ResponseCodeCamel
	}

	if len(rawCode) > 0 {
		var s string
		if err := json.Unmarshal(rawCode, &s); err == nil {
			if val, err := strconv.Atoi(s); err == nil {
				d.ResponseCode = val
			}
		} else {
			var val int
			if err := json.Unmarshal(rawCode, &val); err == nil {
				d.ResponseCode = val
			}
		}
	}
	return nil
}

// RetryPolicy represents a retry policy configuration.
type RetryPolicy struct {
	Retry          int                `json:"retry"`             // 重试次数
	BackoffType    string             `json:"backoff_type"`      // 退避类型 (e.g. "fixed", "exponential")
	BaseMs         int                `json:"base_ms"`           // 退避间隔 (毫秒)
	ErrorCodes     []string           `json:"error_codes"`       // 需要重试的错误码/状态码列表
	ErrorMessages  []string           `json:"error_messages"`    // 需要重试的错误消息列表
	CodePolicy     *ErrorParserPolicy `json:"code_policy"`       // 错误码解析策略
	MessagePolicy  *ErrorParserPolicy `json:"message_policy"`    // 错误消息解析策略
	ConnectTimeout int                `json:"connect_timeout"`   // 建立连接超时 (毫秒)
	TtftTimeout    int                `json:"ttft_timeout"`      // 首字超时 (毫秒)
	TotalTimeout   int                `json:"total_timeout"`     // 请求总超时 (毫秒)
	IdleTimeout    int                `json:"idle_timeout"`      // 读空闲超时 (毫秒)
	Version        int64              `json:"version,omitempty"` // 版本标识 (保留)
}

// UnmarshalJSON implements custom JSON unmarshaling to handle string/integer conversion for ErrorCodes.
func (r *RetryPolicy) UnmarshalJSON(data []byte) error {
	type Alias RetryPolicy
	aux := &struct {
		ErrorCodesCamel     []json.RawMessage  `json:"errorCodes"`
		ErrorCodesSnake     []json.RawMessage  `json:"error_codes"`
		ErrorMessagesCamel  []string           `json:"errorMessages"`
		CodePolicyCamel     *ErrorParserPolicy `json:"codePolicy"`
		MessagePolicyCamel  *ErrorParserPolicy `json:"messagePolicy"`
		ConnectTimeoutCamel int                `json:"connectTimeout"`
		TtftTimeoutCamel    int                `json:"ttftTimeout"`
		TotalTimeoutCamel   int                `json:"totalTimeout"`
		IdleTimeoutCamel    int                `json:"idleTimeout"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 1. 处理 ErrorCodes (支持 errorCodes 和 error_codes，且兼容整型或字符型)
	var rawCodes []json.RawMessage
	if len(aux.ErrorCodesSnake) > 0 {
		rawCodes = aux.ErrorCodesSnake
	} else if len(aux.ErrorCodesCamel) > 0 {
		rawCodes = aux.ErrorCodesCamel
	}
	if len(rawCodes) > 0 {
		r.ErrorCodes = make([]string, len(rawCodes))
		for i, raw := range rawCodes {
			var s string
			if err := json.Unmarshal(raw, &s); err == nil {
				r.ErrorCodes[i] = s
			} else {
				var val float64
				if err := json.Unmarshal(raw, &val); err == nil {
					r.ErrorCodes[i] = strconv.FormatFloat(val, 'f', -1, 64)
				} else {
					r.ErrorCodes[i] = string(raw)
				}
			}
		}
	}

	// 2. 处理 ErrorMessages
	if len(aux.ErrorMessagesCamel) > 0 {
		r.ErrorMessages = aux.ErrorMessagesCamel
	}

	// 3. 处理 CodePolicy 和 MessagePolicy
	if aux.CodePolicyCamel != nil {
		r.CodePolicy = aux.CodePolicyCamel
	}
	if aux.MessagePolicyCamel != nil {
		r.MessagePolicy = aux.MessagePolicyCamel
	}

	// 4. 处理超时相关字段
	if aux.ConnectTimeoutCamel > 0 {
		r.ConnectTimeout = aux.ConnectTimeoutCamel
	}
	if aux.TtftTimeoutCamel > 0 {
		r.TtftTimeout = aux.TtftTimeoutCamel
	}
	if aux.TotalTimeoutCamel > 0 {
		r.TotalTimeout = aux.TotalTimeoutCamel
	}
	if aux.IdleTimeoutCamel > 0 {
		r.IdleTimeout = aux.IdleTimeoutCamel
	}

	return nil
}

// FallbackPolicy represents a fallback policy configuration.
type FallbackPolicy struct {
	Targets []string `json:"targets"` // 降级目标模型链条，如 ["gpt-4:free", "gpt-3.5-turbo"]
}
