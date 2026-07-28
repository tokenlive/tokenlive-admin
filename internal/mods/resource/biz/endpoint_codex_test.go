package biz

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
)

func TestIsCodexEndpoint(t *testing.T) {
	p := &schema.Provider{
		AuthType: "oauth_token",
		URL:      "https://chatgpt.com/backend-api/codex",
	}
	cred, _ := json.Marshal(schema.OAuthCredential{
		TokenEndpoint: "https://auth.openai.com/oauth/token",
		AccountID:     "acct-1",
	})
	p.OAuth = cred

	if !isCodexEndpoint(p, p.URL, []string{"chat_completion"}) {
		t.Fatal("expected codex endpoint detection from provider oauth/url")
	}
	if !isCodexEndpoint(nil, "https://chatgpt.com/backend-api/codex", nil) {
		t.Fatal("expected codex detection from endpoint url")
	}
	if isCodexEndpoint(nil, "https://api.openai.com/v1", []string{"chat_completion"}) {
		t.Fatal("did not expect plain openai url to be codex")
	}
}

func TestEvaluateResponsesTestResultSSE(t *testing.T) {
	sse := []byte("data: {\"type\":\"response.created\",\"response\":{\"status\":\"in_progress\"}}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"pong\"}]}]}}\n\n")
	result, err := evaluateResponsesTestResult(200, 12, sse, string(sse), true)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if result == nil || !result.Success {
		t.Fatalf("expected success, got %#v", result)
	}
	if result.Detail != "pong" {
		t.Fatalf("detail=%q want pong", result.Detail)
	}

	errSSE := []byte("data: {\"type\":\"error\",\"error\":{\"message\":\"bad token\"}}\n\n")
	result, err = evaluateResponsesTestResult(200, 12, errSSE, string(errSSE), true)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if result == nil || result.Success {
		t.Fatalf("expected failure, got %#v", result)
	}
	if !strings.Contains(result.Message, "bad token") {
		t.Fatalf("message=%q", result.Message)
	}
}
