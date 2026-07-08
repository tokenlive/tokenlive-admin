package schema

import (
	"encoding/json"

	"github.com/tokenlive/tokenlive-admin/pkg/errors"
)

// Policy 网关策略大聚合传输结构体，对应网关侧的 policy.Policy。
// 内部直接引用各个具体策略的 Form 结构体，它们已剔除纯数据库维度的 Deleted 审计字段。
type Policy struct {
	LoadBalancePolicy    *PolicyLoadbalanceForm    `json:"load_balance_policy,omitempty"`
	InvocationPolicy     *PolicyInvocationForm     `json:"invocation_policy,omitempty"`
	LimitPolicies        []*PolicyLimitForm        `json:"limit_policies,omitempty"`
	RoutePolicies        []*PolicyRouteForm        `json:"route_policies,omitempty"`
	CircuitBreakPolicies []*PolicyCircuitBreakForm `json:"circuit_break_policies,omitempty"`
	TaggingPolicies      []*PolicyTaggingForm      `json:"tagging_policies,omitempty"`
}

type PolicyCopyToModelForm struct {
	ModelID   string  `json:"model_id" binding:"required,max=20"`
	Name      string  `json:"name" binding:"max=128"`
	ScopeType *string `json:"scope_type" binding:"omitempty,max=32"`
	ScopeCode *string `json:"scope_code" binding:"omitempty,max=128"`
	Priority  *int    `json:"priority" binding:"omitempty,min=0"`
}

func (a *PolicyCopyToModelForm) Validate() error {
	if a.ModelID == "" {
		return errors.BadRequest("", "model_id is required")
	}
	if a.ScopeType != nil || a.ScopeCode != nil {
		var scopeType, scopeCode string
		if a.ScopeType != nil {
			scopeType = *a.ScopeType
		}
		if a.ScopeCode != nil {
			scopeCode = *a.ScopeCode
		}
		if err := validatePolicyScope(scopeType, scopeCode, false); err != nil {
			return err
		}
	}
	return nil
}

func validatePolicyScope(scopeType, scopeCode string, allowEmptyScopeCode bool) error {
	switch scopeType {
	case "", "global":
		if scopeCode != "" {
			return errors.BadRequest("", "scope_code must be empty when scope_type is global")
		}
	case "tenant", "user":
		if scopeCode == "" && !allowEmptyScopeCode {
			return errors.BadRequest("", "scope_code is required when scope_type is %s", scopeType)
		}
	default:
		return errors.BadRequest("", "scope_type must be one of global, tenant, user")
	}
	return nil
}

// MarshalJSON 自定义序列化，在序列化为写入 Redis 的 JSON 串时，递归过滤清除与网关无关的数据库元数据/审计字段以精简数据体积
func (p *Policy) MarshalJSON() ([]byte, error) {
	type Alias Policy
	raw, err := json.Marshal((*Alias)(p))
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}

	cleanJSONMap(m)

	return json.Marshal(m)
}

func cleanJSONMap(m map[string]interface{}) {
	ignoreKeys := map[string]bool{
		"creator":     true,
		"modifier":    true,
		"created_at":  true,
		"updated_at":  true,
		"model_id":    true,
		"scope_type":  true,
		"scope_code":  true,
		"enabled":     true,
		"description": true,
	}

	for k, v := range m {
		if ignoreKeys[k] {
			delete(m, k)
			continue
		}

		if subMap, ok := v.(map[string]interface{}); ok {
			cleanJSONMap(subMap)
		} else if slice, ok := v.([]interface{}); ok {
			for _, item := range slice {
				if subItemMap, ok := item.(map[string]interface{}); ok {
					cleanJSONMap(subItemMap)
				}
			}
		}
	}
}
