package biz

import (
	"encoding/json"
	"testing"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
)

func TestParseCodexUsageToQuota(t *testing.T) {
	raw := []byte(`{
		"plan_type": "pro",
		"chatgpt_subscription_active_until": 1787858068,
		"rate_limit": {
			"primary_window": {"used_percent": 40, "limit_window_seconds": 18000, "reset_after": 3600},
			"secondary_window": {"used_percent": 10, "limit_window_seconds": 604800, "reset_after": 86400}
		},
		"rate_limit_reset_credits": {"available_count": 2}
	}`)
	result, err := parseCodexUsageToQuota(raw)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if result.Provider != "codex" || result.Plan != "pro" {
		t.Fatalf("provider/plan = %s/%s", result.Provider, result.Plan)
	}
	if result.SubscriptionActiveUntil == "" || result.SubscriptionActiveUntil == "1787856068" {
		t.Fatalf("subscription_active_until not formatted: %q", result.SubscriptionActiveUntil)
	}
	if len(result.Windows) < 2 {
		t.Fatalf("windows = %#v", result.Windows)
	}
	if result.Windows[0].Label != "5 小时窗口" {
		t.Fatalf("primary label = %q", result.Windows[0].Label)
	}
	if result.Windows[1].Label != "周窗口" {
		t.Fatalf("secondary label = %q", result.Windows[1].Label)
	}
	if result.Windows[0].UsedPercent == nil || *result.Windows[0].UsedPercent != 40 {
		t.Fatalf("used percent = %#v", result.Windows[0].UsedPercent)
	}
	if result.Windows[0].RemainingPercent == nil || *result.Windows[0].RemainingPercent != 60 {
		t.Fatalf("remaining percent = %#v", result.Windows[0].RemainingPercent)
	}
	if result.Extras["rate_limit_reset_credits_available"] == nil {
		t.Fatalf("expected reset credits in extras: %#v", result.Extras)
	}
}

func TestParseCodexUsageToQuota_MonthlyAndNoFalseFiveHour(t *testing.T) {
	// No 5h window: weekly + monthly should keep both labels from seconds.
	raw := []byte(`{
		"plan_type": "team",
		"rate_limit": {
			"primary_window": {"used_percent": 20, "limit_window_seconds": 604800},
			"secondary_window": {"used_percent": 5, "limit_window_seconds": 2592000}
		}
	}`)
	result, err := parseCodexUsageToQuota(raw)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(result.Windows) != 2 {
		t.Fatalf("windows=%#v", result.Windows)
	}
	if result.Windows[0].Label != "周窗口" || result.Windows[1].Label != "月窗口" {
		t.Fatalf("labels=%q, %q", result.Windows[0].Label, result.Windows[1].Label)
	}
	for _, w := range result.Windows {
		if w.Label == "5 小时窗口" {
			t.Fatalf("unexpected 5h label for non-5h payload: %#v", result.Windows)
		}
	}

	// True 5h + monthly.
	raw2 := []byte(`{
		"rate_limit": {
			"primary_window": {"used_percent": 1, "limit_window_seconds": 18000},
			"secondary_window": {"used_percent": 2, "limit_window_seconds": 2592000}
		}
	}`)
	result2, err := parseCodexUsageToQuota(raw2)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(result2.Windows) != 2 {
		t.Fatalf("windows=%#v", result2.Windows)
	}
	if result2.Windows[0].Label != "5 小时窗口" || result2.Windows[1].Label != "月窗口" {
		t.Fatalf("labels=%q, %q", result2.Windows[0].Label, result2.Windows[1].Label)
	}
}

func TestResolveResetFieldsUnixSeconds(t *testing.T) {
	// 1785908987 is a far-future unix timestamp; label must not be the raw number.
	window := map[string]any{
		"reset_at": float64(1785908987),
	}
	resetAt, label := resolveResetFields(window)
	if resetAt == "" {
		t.Fatal("expected rfc3339 reset_at")
	}
	if label == "" || label == "1785908987" || label == "-" {
		t.Fatalf("unexpected label: %q", label)
	}
	// Should look like a humanized duration (days/hours), not a bare integer.
	if !(containsAny(label, "天", "小时", "分钟", "已可重置")) {
		t.Fatalf("label not humanized: %q", label)
	}
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if len(p) > 0 && (len(s) >= len(p)) {
			for i := 0; i+len(p) <= len(s); i++ {
				if s[i:i+len(p)] == p {
					return true
				}
			}
		}
	}
	return false
}

func TestBuildAndMergeXAIBillingSummary(t *testing.T) {
	weekly := []byte(`{
		"config": {
			"creditUsagePercent": 25,
			"currentPeriod": {"type": "weekly", "end": "2026-08-01T00:00:00Z"},
			"productUsage": [{"product": "chat", "usagePercent": 30}]
		}
	}`)
	monthly := []byte(`{
		"config": {
			"monthlyLimit": 15000,
			"used": 3000,
			"billingPeriodEnd": "2026-08-31T00:00:00Z",
			"onDemandCap": 5000,
			"onDemandUsed": 500
		}
	}`)
	w := buildXAIBillingSummary(weekly)
	m := buildXAIBillingSummary(monthly)
	if w == nil || m == nil {
		t.Fatalf("expected summaries, got weekly=%v monthly=%v", w, m)
	}
	merged := mergeXAIBillingSummaries(w, m)
	if merged == nil {
		t.Fatal("merged nil")
	}
	result := xaiSummaryToQuotaResult(merged)
	if result.Provider != "xai" {
		t.Fatalf("provider=%s", result.Provider)
	}
	if result.Plan != "supergrok" {
		t.Fatalf("plan=%s want supergrok", result.Plan)
	}
	if len(result.Windows) == 0 {
		t.Fatal("expected windows")
	}
	// Ensure JSON shape is stable
	b, _ := json.Marshal(result)
	if len(b) < 10 {
		t.Fatalf("marshal failed: %s", string(b))
	}
}

func TestBuildXAIBillingSummary_CentObject(t *testing.T) {
	// Real xAI payloads often nest cents under {"val": N}.
	monthly := []byte(`{
		"config": {
			"monthlyLimit": {"val": 15000},
			"used": {"val": 4500},
			"billingPeriodEnd": "2026-08-31T00:00:00Z",
			"onDemandCap": {"val": 0},
			"onDemandUsed": {"val": 0}
		}
	}`)
	m := buildXAIBillingSummary(monthly)
	if m == nil {
		t.Fatal("summary nil")
	}
	if m.MonthlyLimitCents == nil || *m.MonthlyLimitCents != 15000 {
		t.Fatalf("monthlyLimit=%v", m.MonthlyLimitCents)
	}
	if m.UsedCents == nil || *m.UsedCents != 4500 {
		t.Fatalf("used=%v", m.UsedCents)
	}
	result := xaiSummaryToQuotaResult(m)
	if result.Plan != "supergrok" {
		t.Fatalf("plan=%s", result.Plan)
	}
	foundMonthly := false
	for _, w := range result.Windows {
		if w.ID == "monthly" {
			foundMonthly = true
			if w.Label != "月额度" {
				t.Fatalf("label=%q", w.Label)
			}
			if w.UsedPercent == nil || *w.UsedPercent != 30 {
				t.Fatalf("used percent=%v", w.UsedPercent)
			}
			if w.AmountLabel == "" {
				t.Fatal("expected amount label")
			}
		}
	}
	if !foundMonthly {
		t.Fatalf("missing monthly window: %#v", result.Windows)
	}
}

func TestIsXAIOAuthProvider(t *testing.T) {
	p := &schema.Provider{
		AuthType: "oauth_token",
		URL:      "https://api.x.ai/v1",
	}
	cred, _ := json.Marshal(schema.OAuthCredential{TokenEndpoint: "https://auth.x.ai/oauth/token"})
	p.OAuth = cred
	if !isXAIOAuthProvider(p) {
		t.Fatal("expected xai provider")
	}
}
