package gatewaycontract_test

import (
	"encoding/json"
	"testing"

	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
)

func TestKeyShapes(t *testing.T) {
	k := gatewaycontract.Keys
	cases := []struct{ got, want string }{
		{k.Config.ModelVersions(), "aigw:config:model_versions"},
		{k.Config.Endpoints("gpt-5"), "aigw:config:endpoints:gpt-5"},
		{k.Config.SmartRouting("smart"), "aigw:config:smart_routing:smart"},
		{k.Config.Alias("fast"), "aigw:config:alias:fast"},
		{k.Config.ModelAliases("gpt-5"), "aigw:config:model_aliases:gpt-5"},

		{k.Policies.User("u1"), "aigw:policies:user:u1"},
		{k.Policies.Tenant("acme"), "aigw:policies:tenant:acme"},
		{k.Policies.Model("gpt-5"), "aigw:policies:model:gpt-5"},
		// Gateway reads this; Admin never writes it.
		{k.Policies.Global, "aigw:policies:global"},

		{k.Tenant.Prefix(), "aigw:tenant:"},
		{k.Tenant.ModelsSuffix(), ":models"},
		{k.Tenant.EndpointsSuffix("gpt-5"), ":model:gpt-5:endpoints"},
		{k.Tenant.Models("acme"), "aigw:tenant:acme:models"},
		{k.Tenant.Endpoints("acme", "gpt-5"), "aigw:tenant:acme:model:gpt-5:endpoints"},
		{k.Tenant.Providers("acme", "gpt-5"), "aigw:tenant:acme:model:gpt-5:providers"},
		{k.Tenant.Temp(k.Tenant.Models("acme")), "aigw:tenant:acme:models:tmp"},

		// Gateway reads this; Admin never writes it.
		{k.User.Models("u1"), "aigw:user:u1:models"},

		{k.Status.Global(12, gatewaycontract.MetricSuccess), "aigw:status:global:12:s"},
		{k.Status.Global(12, gatewaycontract.MetricFailure), "aigw:status:global:12:f"},
		{k.Status.Model("gpt-5", 12, gatewaycontract.MetricTTFTSum), "aigw:status:model:gpt-5:12:ttft_sum"},
		{k.Status.Model("gpt-5", 12, gatewaycontract.MetricTTFTCount), "aigw:status:model:gpt-5:12:ttft_cnt"},
		{k.Status.Model("gpt-5", 12, gatewaycontract.MetricOutputTokens), "aigw:status:model:gpt-5:12:out"},
		{k.Status.Model("gpt-5", 12, gatewaycontract.MetricDurationMs), "aigw:status:model:gpt-5:12:dur_ms"},
		{k.Status.Endpoint("ep1", 12, gatewaycontract.MetricSuccess), "aigw:status:endpoint:ep1:12:s"},
		{k.Status.Provider("openai", 12, gatewaycontract.MetricFailure), "aigw:status:provider:openai:12:f"},
		{k.Status.Daily(gatewaycontract.DailyRequests, "2026-10-10"), "aigw:status:daily:req:2026-10-10"},
		{k.Status.Daily(gatewaycontract.DailyInputTokens, "2026-10-10"), "aigw:status:daily:input_tokens:2026-10-10"},
		{k.Status.Daily(gatewaycontract.DailyOutputTokens, "2026-10-10"), "aigw:status:daily:output_tokens:2026-10-10"},
		{k.Status.Daily(gatewaycontract.DailyCachedTokens, "2026-10-10"), "aigw:status:daily:cached_tokens:2026-10-10"},
		{k.Status.Daily(gatewaycontract.DailyCacheCreationTokens, "2026-10-10"), "aigw:status:daily:cache_creation_tokens:2026-10-10"},
		{k.Status.Daily(gatewaycontract.DailyCost, "2026-10-10"), "aigw:status:daily:cost:2026-10-10"},

		{k.Channel.PolicyUpdate(), "aigw:channel:policy_update"},
		{k.Channel.APIKeyUpdate(), "aigw:channel:apikey_update"},
		{k.Events.Policy, "aigw:events:policy"},
		// Admin reads these; neither side writes them.
		{k.Circuit.OpenEndpoints, "aigw:cb:open_endpoints"},
		{k.Circuit.OpenServices, "aigw:cb:open_services"},
		{k.APIKey.Hash("hash123"), "aigw:apikey_hash:hash123"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("key = %q, want %q", tc.got, tc.want)
		}
	}
	if gatewaycontract.FieldBilling != "*:billing" || gatewaycontract.FieldAll != "*" {
		t.Errorf("policy fields drifted: billing=%q all=%q", gatewaycontract.FieldBilling, gatewaycontract.FieldAll)
	}
	if gatewaycontract.BillingField("gpt-5") != "gpt-5:billing" {
		t.Errorf("BillingField = %q", gatewaycontract.BillingField("gpt-5"))
	}
	if gatewaycontract.PurgePayload != "purge" {
		t.Errorf("PurgePayload = %q", gatewaycontract.PurgePayload)
	}
}

func TestResolvedEndpointRoundTripKeepsEmptyRequiredFields(t *testing.T) {
	price := 1.5
	in := gatewaycontract.ResolvedEndpoint{
		RealModel:        "gpt-5",
		ProviderName:     "OpenAI",
		ProviderProtocol: "openai",
		APIKey:           "sk-live",
		URL:              "https://api.openai.com",
		Timeout:          60000,
		InputPrice:       &price,
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out gatewaycontract.ResolvedEndpoint
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.RealModel != in.RealModel || out.ProviderName != in.ProviderName || out.ProviderProtocol != in.ProviderProtocol ||
		out.APIKey != in.APIKey || out.URL != in.URL || out.Timeout != in.Timeout || out.InputPrice == nil || *out.InputPrice != *in.InputPrice {
		t.Fatalf("round trip = %+v, want %+v", out, in)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"real_model", "provider_name", "provider_protocol", "api_key", "url", "timeout", "max_retries", "priority", "weight"} {
		if _, ok := fields[required]; !ok {
			t.Errorf("required field %q omitted from %s", required, raw)
		}
	}
	for _, omitted := range []string{"id", "provider_code", "headers", "request_types", "context_length", "output_price"} {
		if _, ok := fields[omitted]; ok {
			t.Errorf("zero field %q present in %s", omitted, raw)
		}
	}
	if string(fields["input_price"]) != "1.5" {
		t.Errorf("input_price = %s", fields["input_price"])
	}
}

func TestRuntimeSmartRoutingRoundTrip(t *testing.T) {
	in := gatewaycontract.RuntimeSmartRouting{
		Version: 3, JudgeModel: "judge",
		Ranges: []gatewaycontract.RuntimeSmartRange{{Min: 0, Max: 1000, Model: "small"}},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out gatewaycontract.RuntimeSmartRouting
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.Version != in.Version || out.JudgeModel != in.JudgeModel || len(out.Ranges) != 1 || out.Ranges[0] != in.Ranges[0] {
		t.Fatalf("round trip = %+v, want %+v", out, in)
	}
}

func TestHashAPIKeyMatchesGatewayHMACSHA256(t *testing.T) {
	got := gatewaycontract.HashAPIKey("tl_live_example", "pepper")
	want := "06bfbed9282f1dcb96bd25c7bef96d9b49de0be5f3777b44f4f71cfcca8821b1"
	if got != want {
		t.Fatalf("HashAPIKey() = %q, want %q", got, want)
	}
}
