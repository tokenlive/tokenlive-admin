package schema

import (
	"encoding/json"
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
