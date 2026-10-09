package biz

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSelectEndpointProbeKindKeepsJoyCodeOffGenericResponses(t *testing.T) {
	apis := []string{"chat_completion", "responses", "messages"}
	if got := selectEndpointProbeKind("joycode", apis, false); got != endpointProbeJoyCode {
		t.Fatalf("joycode probe kind = %v, want joycode", got)
	}
	if got := selectEndpointProbeKind("openai", []string{"responses"}, false); got != endpointProbeResponses {
		t.Fatalf("openai responses probe kind = %v, want responses", got)
	}
	if got := selectEndpointProbeKind("openai", []string{"chat_completion", "responses"}, false); got != endpointProbeChat {
		t.Fatalf("openai chat probe kind = %v, want chat", got)
	}
}

func TestBuildJoyCodeProbeUsesRuntimeFunctionID(t *testing.T) {
	t.Setenv("JOYCODE_APPID", "joycode_ide")
	t.Setenv("JOYCODE_SIGN_KEY", "test-sign-key")

	cases := []struct {
		name      string
		model     string
		apis      []string
		function  string
		wantInput bool
	}{
		{"chat", "Kimi-K2.6", []string{"chat_completion"}, "chat_completions", false},
		{"responses only", "Kimi-K2.6", []string{"responses"}, "responses_completions", true},
		{"responses with chat", "Kimi-K2.6", []string{"chat_completion", "responses"}, "chat_completions", false},
		{"messages only", "GLM-5.1", []string{"messages"}, "chat_completions", false},
		{"claude messages", "claude-sonnet-4", []string{"messages"}, "anthropic_completions", false},
		{"claude responses", "Claude-Opus-4", []string{"responses"}, "anthropic_completions", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reqURL, body, err := buildJoyCodeProbe("https://api-ai.jd.com", tc.model, tc.apis)
			if err != nil {
				t.Fatalf("build probe: %v", err)
			}
			if !strings.Contains(reqURL, "functionId="+tc.function) {
				t.Fatalf("url %s does not use functionId=%s", reqURL, tc.function)
			}
			var payload map[string]interface{}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			_, hasInput := payload["input"]
			if hasInput != tc.wantInput {
				t.Fatalf("input present = %v, want %v", hasInput, tc.wantInput)
			}
		})
	}
}

func TestEvaluateJoyCodeTestResultRejectsLostLogin(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"business code", `{"code":401,"msg":"未登录"}`},
		{"http ok login text", `{"code":0,"message":"登录已失效，请重新登录"}`},
		{"empty success", `{"code":0,"data":null}`},
		{"responses envelope without output", `{"id":"resp_1","object":"response"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := evaluateJoyCodeTestResult(200, 8, []byte(tc.body), tc.body, "Kimi-K2.6")
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if result == nil || result.Success {
				t.Fatalf("expected login/probe failure, got %#v", result)
			}
		})
	}

	okBody := `{"choices":[{"message":{"role":"assistant","content":"pong"}}]}`
	result, err := evaluateJoyCodeTestResult(200, 8, []byte(okBody), okBody, "Kimi-K2.6")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if result == nil || !result.Success || result.Detail != "pong" {
		t.Fatalf("expected chat success, got %#v", result)
	}

	responsesBody := `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"pong"}]}]}`
	result, err = evaluateJoyCodeTestResult(200, 8, []byte(responsesBody), responsesBody, "Kimi-K2.6")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if result == nil || !result.Success || result.Detail != "pong" {
		t.Fatalf("expected responses success, got %#v", result)
	}
}
