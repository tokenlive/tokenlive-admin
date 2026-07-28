package biz

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/cachex"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestParseOAuthCallbackURL(t *testing.T) {
	code, state, errParam, err := parseOAuthCallbackURL("http://localhost:1455/auth/callback?code=abc&state=xyz")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if code != "abc" || state != "xyz" || errParam != "" {
		t.Fatalf("got code=%q state=%q err=%q", code, state, errParam)
	}

	code, state, errParam, err = parseOAuthCallbackURL("code=abc&state=xyz&error=access_denied")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if code != "abc" || state != "xyz" || errParam != "access_denied" {
		t.Fatalf("got code=%q state=%q err=%q", code, state, errParam)
	}
}

func TestParseCodexIDTokenClaims(t *testing.T) {
	payload := map[string]any{
		"email": "user@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "acct-123",
		},
	}
	raw, _ := json.Marshal(payload)
	idToken := "hdr." + base64.RawURLEncoding.EncodeToString(raw) + ".sig"

	accountID, email := parseCodexIDTokenClaims(idToken)
	if accountID != "acct-123" || email != "user@example.com" {
		t.Fatalf("got accountID=%q email=%q", accountID, email)
	}
}

func TestMergeOAuthAccountHeader(t *testing.T) {
	provider := &schema.Provider{AuthType: "oauth_token"}
	cred, _ := json.Marshal(schema.OAuthCredential{AccountID: "acct-1"})
	provider.OAuth = cred

	headers := MergeOAuthAccountHeader(nil, provider, "oauth_token")
	if headers["Chatgpt-Account-Id"] != "acct-1" {
		t.Fatalf("expected injected header, got %#v", headers)
	}

	// Existing header should win.
	existing := map[string]string{"Chatgpt-Account-Id": "manual"}
	headers = MergeOAuthAccountHeader(existing, provider, "oauth_token")
	if headers["Chatgpt-Account-Id"] != "manual" {
		t.Fatalf("expected existing header preserved, got %#v", headers)
	}

	// Non-oauth auth should not inject.
	headers = MergeOAuthAccountHeader(nil, provider, "api_key")
	if len(headers) != 0 {
		t.Fatalf("expected no headers for api_key, got %#v", headers)
	}
}

func TestGeneratePKCECodes(t *testing.T) {
	codes, err := generatePKCECodes()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if codes.CodeVerifier == "" || codes.CodeChallenge == "" {
		t.Fatalf("empty pkce codes: %#v", codes)
	}
	if len(codes.CodeVerifier) < 43 {
		t.Fatalf("verifier too short: %d", len(codes.CodeVerifier))
	}
}

func TestParseCodexModelsResponse(t *testing.T) {
	raw := []byte(`{
		"models": [
			{"slug": "gpt-5.5", "display_name": "GPT-5.5"},
			{"slug": "gpt-5.4", "display_name": "GPT-5.4"},
			{"slug": "gpt-5.5", "display_name": "dup"}
		]
	}`)
	models, err := parseCodexModelsResponse(raw)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 unique models, got %d: %#v", len(models), models)
	}
	if models[0].ID != "gpt-5.5" || models[0].Object != "model" || models[0].OwnedBy != "openai" {
		t.Fatalf("unexpected first model: %#v", models[0])
	}
	if models[1].ID != "gpt-5.4" {
		t.Fatalf("unexpected second model: %#v", models[1])
	}
}

func TestBuildCodexModelsURL(t *testing.T) {
	got, err := buildCodexModelsURL("https://chatgpt.com/backend-api/codex/", "0.144.1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := "https://chatgpt.com/backend-api/codex/models?client_version=0.144.1"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestIsCodexOAuthProvider(t *testing.T) {
	p := &schema.Provider{
		AuthType: "oauth_token",
		URL:      "https://chatgpt.com/backend-api/codex",
	}
	cred, _ := json.Marshal(schema.OAuthCredential{
		TokenEndpoint: "https://auth.openai.com/oauth/token",
		AccountID:     "acct-1",
	})
	p.OAuth = cred
	if !isCodexOAuthProvider(p) {
		t.Fatal("expected codex oauth provider")
	}
	if !isCodexModelsBaseURL(p.URL) {
		t.Fatal("expected codex base url detection")
	}
}

func TestCompleteOAuthFlowRejectsForeignUser(t *testing.T) {
	cache := cachex.NewMemoryCache(cachex.MemoryConfig{CleanupInterval: time.Minute})
	p := &Provider{Cache: cache}

	ctxUserA := util.NewUserID(context.Background(), "user-a")
	ctxUserB := util.NewUserID(context.Background(), "user-b")

	state := "state-1"
	if err := p.saveOAuthSession(ctxUserA, state, providerOAuthSession{
		Provider:     oauthProviderCodex,
		UserID:       "user-a",
		CodeVerifier: "verifier",
	}); err != nil {
		t.Fatalf("save session: %v", err)
	}

	_, err := p.CompleteOAuthFlow(ctxUserB, &schema.OAuthCompleteForm{
		Provider:    oauthProviderCodex,
		State:       state,
		CallbackURL: "http://localhost:1455/auth/callback?code=abc&state=" + state,
	})
	if err == nil {
		t.Fatal("expected forbidden for mismatched user")
	}

	// Ownership failure must not consume the one-time session.
	session, err := p.getOAuthSession(ctxUserA, state)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if session == nil || session.UserID != "user-a" {
		t.Fatalf("expected session retained for owner, got %#v", session)
	}
}
