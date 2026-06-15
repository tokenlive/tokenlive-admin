package schema

import (
	"encoding/json"
	"testing"
)

func TestRetryPolicy_UnmarshalJSON(t *testing.T) {
	// Test int array for errorCodes
	rawInt := `{
		"retry": 3,
		"interval": 200,
		"errorCodes": [429, 502],
		"errorMessages": ["timeout", "limit exceeded"],
		"exceptions": ["ConnectionException", "TimeoutException"],
		"codePolicy": {
			"parser": "JsonPath",
			"expression": "$.error.code",
			"statuses": ["200"],
			"contentTypes": ["application/json"]
		}
	}`

	var r1 RetryPolicy
	if err := json.Unmarshal([]byte(rawInt), &r1); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(r1.ErrorCodes) != 2 || r1.ErrorCodes[0] != "429" || r1.ErrorCodes[1] != "502" {
		t.Errorf("unexpected errorCodes: %v", r1.ErrorCodes)
	}
	if len(r1.ErrorMessages) != 2 || r1.ErrorMessages[0] != "timeout" {
		t.Errorf("unexpected errorMessages: %v", r1.ErrorMessages)
	}
	if r1.CodePolicy == nil || r1.CodePolicy.Parser != "JsonPath" || r1.CodePolicy.Statuses[0] != "200" {
		t.Errorf("unexpected codePolicy: %+v", r1.CodePolicy)
	}

	// Test string array for errorCodes
	rawStr := `{
		"retry": 3,
		"interval": 200,
		"errorCodes": ["429", "rate_limit_exceeded"]
	}`

	var r2 RetryPolicy
	if err := json.Unmarshal([]byte(rawStr), &r2); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(r2.ErrorCodes) != 2 || r2.ErrorCodes[0] != "429" || r2.ErrorCodes[1] != "rate_limit_exceeded" {
		t.Errorf("unexpected errorCodes: %v", r2.ErrorCodes)
	}
}

func TestPolicy_MarshalJSON_IgnoreKeys(t *testing.T) {
	desc := "test description"
	creator := "test creator"
	modifier := "test modifier"

	p := &Policy{
		LoadBalancePolicy: &PolicyLoadbalanceForm{
			ID:          "lb-id",
			Name:        "lb-name",
			Type:        "round_robin",
			Enabled:     1,
			Description: &desc,
			Creator:     &creator,
			Modifier:    &modifier,
		},
		InvocationPolicy: &PolicyInvocationForm{
			ID:          "inv-id",
			Name:        "inv-name",
			Type:        "failover",
			Enabled:     1,
			Description: &desc,
			Creator:     &creator,
			Modifier:    &modifier,
			RetryPolicy: &RetryPolicy{
				Retry: 3,
			},
		},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("failed to marshal policy: %v", err)
	}

	// 反序列化为 map 校验这些字段是否被清除
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// 检查 load_balance_policy 内部
	lb, ok := m["load_balance_policy"].(map[string]interface{})
	if !ok {
		t.Fatal("expected load_balance_policy map")
	}
	for _, key := range []string{"enabled", "description", "creator", "modifier", "created_at", "updated_at"} {
		if _, exists := lb[key]; exists {
			t.Errorf("expected key '%s' to be ignored in load_balance_policy, but it exists", key)
		}
	}
	if lb["id"] != "lb-id" {
		t.Errorf("expected id 'lb-id', got %v", lb["id"])
	}
	if lb["type"] != "round_robin" {
		t.Errorf("expected type 'round_robin', got %v", lb["type"])
	}

	// 检查 invocation_policy 内部
	inv, ok := m["invocation_policy"].(map[string]interface{})
	if !ok {
		t.Fatal("expected invocation_policy map")
	}
	for _, key := range []string{"enabled", "description", "creator", "modifier", "created_at", "updated_at"} {
		if _, exists := inv[key]; exists {
			t.Errorf("expected key '%s' to be ignored in invocation_policy, but it exists", key)
		}
	}
	if inv["id"] != "inv-id" {
		t.Errorf("expected id 'inv-id', got %v", inv["id"])
	}
	if inv["type"] != "failover" {
		t.Errorf("expected type 'failover', got %v", inv["type"])
	}
}
