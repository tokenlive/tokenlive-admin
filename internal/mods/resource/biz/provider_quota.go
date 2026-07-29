package biz

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
)

const (
	codexUsageURL = "https://chatgpt.com/backend-api/wham/usage"
	xaiBillingWeeklyURL  = "https://cli-chat-proxy.grok.com/v1/billing?format=credits"
	xaiBillingMonthlyURL = "https://cli-chat-proxy.grok.com/v1/billing"
	xaiAPIMeURL          = "https://api.x.ai/v1/me"
	xaiAPIChatURL        = "https://api.x.ai/v1/chat/completions"
	xaiPaidHealthModel   = "grok-4.5"
	xaiGrokClientVersion = "0.2.91"
	xaiGrokUserAgent     = "grok-pager/0.2.91 grok-shell/0.2.91 (macos; aarch64)"
)

// GetQuota fetches live upstream usage/quota for an oauth provider (xAI / Codex).
func (p *Provider) GetQuota(ctx context.Context, providerID string) (*schema.ProviderQuotaResult, error) {
	provider, err := p.ProviderDAL.Get(ctx, providerID)
	if err != nil {
		return nil, err
	} else if provider == nil {
		return nil, errors.NotFound("", "Provider not found")
	}
	if provider.AuthType != "oauth_token" {
		return nil, errors.BadRequest("", "Quota is only supported for oauth_token providers")
	}

	// Refresh token if near expiry (same path as FetchModels / endpoint test).
	if cred := provider.GetOAuth(); cred != nil && cred.RefreshToken != "" {
		now := time.Now()
		if cred.ExpiresAt == nil || cred.ExpiresAt.Before(now.Add(5*time.Minute)) {
			refresher := NewTokenRefresher(p.ProviderDAL.DB, p.ConfigRedisSync)
			refresher.lockAndRefreshProvider(ctx, *provider)
			if refreshed, rErr := p.ProviderDAL.Get(ctx, providerID); rErr == nil && refreshed != nil {
				provider = refreshed
			}
		}
	}

	accessToken := ""
	if keys := provider.GetApiKeys(); len(keys) > 0 {
		accessToken = strings.TrimSpace(keys[0].Value)
	}
	if accessToken == "" {
		return nil, errors.BadRequest("", "Provider has no access token; please re-bind OAuth")
	}

	switch {
	case isCodexOAuthProvider(provider) || isCodexModelsBaseURL(provider.URL):
		return p.fetchCodexQuota(ctx, provider, accessToken)
	case isXAIOAuthProvider(provider):
		return p.fetchXAIQuota(ctx, provider, accessToken)
	default:
		return nil, errors.BadRequest("", "Quota is only supported for xAI or Codex OAuth providers")
	}
}

func isXAIOAuthProvider(provider *schema.Provider) bool {
	if provider == nil {
		return false
	}
	if strings.Contains(strings.ToLower(provider.URL), "x.ai") ||
		strings.Contains(strings.ToLower(provider.URL), "grok.com") {
		return true
	}
	cred := provider.GetOAuth()
	if cred == nil {
		return false
	}
	endpoint := strings.ToLower(cred.TokenEndpoint)
	return strings.Contains(endpoint, "x.ai") || strings.Contains(endpoint, "auth.x.ai")
}

func (p *Provider) fetchCodexQuota(ctx context.Context, provider *schema.Provider, accessToken string) (*schema.ProviderQuotaResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, codexUsageURL, nil)
	if err != nil {
		return nil, errors.BadRequest("", "Failed to create Codex usage request: %s", err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", codexModelsUserAgent)
	req.Header.Set("Originator", codexModelsOriginator)
	if cred := provider.GetOAuth(); cred != nil && strings.TrimSpace(cred.AccountID) != "" {
		req.Header.Set("Chatgpt-Account-Id", strings.TrimSpace(cred.AccountID))
	}

	body, status, err := doJSONRequest(req)
	if err != nil {
		return nil, errors.BadRequest("", "Failed to call Codex usage API: %s", err.Error())
	}
	if status < 200 || status >= 300 {
		return nil, errors.BadRequest("", "Codex usage API returned status %d: %s", status, truncateForError(body))
	}

	result, err := parseCodexUsageToQuota(body)
	if err != nil {
		return nil, errors.BadRequest("", "Failed to parse Codex usage: %s", err.Error())
	}
	// Prefer persisted snapshot from OAuth bind/refresh (id_token claim).
	if result.SubscriptionActiveUntil == "" {
		if cred := provider.GetOAuth(); cred != nil && strings.TrimSpace(cred.SubscriptionActiveUntil) != "" {
			result.SubscriptionActiveUntil = strings.TrimSpace(cred.SubscriptionActiveUntil)
		}
	}
	// Fallback: JWT claim on access_token (often missing; id_token is the reliable source).
	if result.SubscriptionActiveUntil == "" {
		if _, _, until := parseCodexTokenClaims(accessToken); until != "" {
			result.SubscriptionActiveUntil = until
		}
	}
	return result, nil
}

func (p *Provider) fetchXAIQuota(ctx context.Context, provider *schema.Provider, accessToken string) (*schema.ProviderQuotaResult, error) {
	headers := map[string]string{
		"Authorization":         "Bearer " + accessToken,
		"x-xai-token-auth":      "xai-grok-cli",
		"x-grok-client-version": xaiGrokClientVersion,
		"Accept":                "*/*",
		"User-Agent":            xaiGrokUserAgent,
	}
	// cli-chat-proxy billing often requires x-userid (subject/email from OAuth snapshot).
	if userID := resolveXAIUserID(provider, accessToken); userID != "" {
		headers["x-userid"] = userID
	}

	weeklyBody, weeklyStatus, weeklyErr := getWithHeaders(ctx, xaiBillingWeeklyURL, headers)
	monthlyBody, monthlyStatus, monthlyErr := getWithHeaders(ctx, xaiBillingMonthlyURL, headers)

	var weeklySummary, monthlySummary *xaiBillingSummary
	if weeklyErr == nil && weeklyStatus >= 200 && weeklyStatus < 300 {
		weeklySummary = buildXAIBillingSummary(weeklyBody)
	}
	if monthlyErr == nil && monthlyStatus >= 200 && monthlyStatus < 300 {
		monthlySummary = buildXAIBillingSummary(monthlyBody)
	}
	summary := mergeXAIBillingSummaries(weeklySummary, monthlySummary)
	if summary != nil {
		return xaiSummaryToQuotaResult(summary), nil
	}

	// Fallback: paid health check via api.x.ai
	if paid, err := fetchXAIPaidHealth(ctx, accessToken); err == nil && paid != nil {
		// Keep billing failure hints for debugging empty plan/monthly bars.
		if paid.Extras == nil {
			paid.Extras = map[string]any{}
		}
		paid.Extras["billing_weekly_status"] = weeklyStatus
		paid.Extras["billing_monthly_status"] = monthlyStatus
		if weeklyErr != nil {
			paid.Extras["billing_weekly_error"] = weeklyErr.Error()
		} else if weeklyStatus >= 300 {
			paid.Extras["billing_weekly_body"] = truncateForError(weeklyBody)
		}
		if monthlyErr != nil {
			paid.Extras["billing_monthly_error"] = monthlyErr.Error()
		} else if monthlyStatus >= 300 {
			paid.Extras["billing_monthly_body"] = truncateForError(monthlyBody)
		}
		return paid, nil
	}

	if weeklyErr != nil && monthlyErr != nil {
		return nil, errors.BadRequest("", "Failed to call xAI billing API: %s", weeklyErr.Error())
	}
	if weeklyStatus >= 300 || monthlyStatus >= 300 {
		status := weeklyStatus
		body := weeklyBody
		if status < 300 {
			status = monthlyStatus
			body = monthlyBody
		}
		return nil, errors.BadRequest("", "xAI billing API returned status %d: %s", status, truncateForError(body))
	}
	return nil, errors.BadRequest("", "xAI billing returned empty data")
}

func resolveXAIUserID(provider *schema.Provider, accessToken string) string {
	if provider != nil {
		if cred := provider.GetOAuth(); cred != nil {
			if s := strings.TrimSpace(cred.Email); s != "" {
				return s
			}
			if s := strings.TrimSpace(cred.AccountID); s != "" {
				return s
			}
		}
	}
	// Fallback: JWT sub/email from access token.
	if sub, email := parseXAITokenIdentity(accessToken); email != "" {
		return email
	} else if sub != "" {
		return sub
	}
	return ""
}

func parseXAITokenIdentity(token string) (sub, email string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", ""
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return "", ""
		}
	}
	var claims struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", ""
	}
	return strings.TrimSpace(claims.Sub), strings.TrimSpace(claims.Email)
}

func fetchXAIPaidHealth(ctx context.Context, accessToken string) (*schema.ProviderQuotaResult, error) {
	// Profile
	reqMe, err := http.NewRequestWithContext(ctx, http.MethodGet, xaiAPIMeURL, nil)
	if err != nil {
		return nil, err
	}
	reqMe.Header.Set("Authorization", "Bearer "+accessToken)
	reqMe.Header.Set("Accept", "application/json")
	meBody, meStatus, meErr := doJSONRequest(reqMe)

	// Tiny chat probe
	chatPayload := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"ping"}],"max_tokens":1,"stream":false}`, xaiPaidHealthModel))
	reqChat, err := http.NewRequestWithContext(ctx, http.MethodPost, xaiAPIChatURL, strings.NewReader(string(chatPayload)))
	if err != nil {
		return nil, err
	}
	reqChat.Header.Set("Authorization", "Bearer "+accessToken)
	reqChat.Header.Set("Accept", "application/json")
	reqChat.Header.Set("Content-Type", "application/json")
	_, chatStatus, chatErr := doJSONRequest(reqChat)
	if chatErr != nil {
		return nil, chatErr
	}
	if chatStatus < 200 || chatStatus >= 300 {
		return nil, fmt.Errorf("paid health chat status %d", chatStatus)
	}

	plan := "paid"
	extras := map[string]any{"mode": "paid-health", "health": "ok"}
	if meErr == nil && meStatus >= 200 && meStatus < 300 && len(meBody) > 0 {
		extras["profile"] = json.RawMessage(meBody)
	}
	return &schema.ProviderQuotaResult{
		Provider: oauthProviderXAI,
		Plan:     plan,
		Windows:  []schema.ProviderQuotaWindow{},
		Extras:   extras,
	}, nil
}

func getWithHeaders(ctx context.Context, url string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return doJSONRequest(req)
}

func doJSONRequest(req *http.Request) ([]byte, int, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func truncateForError(body []byte) string {
	s := strings.TrimSpace(string(body))
	if len(s) > 512 {
		return s[:512] + "..."
	}
	return s
}

// ---- Codex parse ----

func parseCodexUsageToQuota(body []byte) (*schema.ProviderQuotaResult, error) {
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}

	plan := firstString(root, "plan_type", "planType")
	result := &schema.ProviderQuotaResult{
		Provider:                oauthProviderCodex,
		Plan:                    strings.ToLower(strings.TrimSpace(plan)),
		SubscriptionActiveUntil: firstCodexSubscriptionUntil(root),
		Windows:                 []schema.ProviderQuotaWindow{},
		Extras:                  map[string]any{},
	}

	// rate_limit / code_review_rate_limit windows.
	// Labels are derived from limit_window_seconds (not hard-coded 5h/week),
	// matching CLIProxyAPI Management Center classification.
	appendLimitWindows := func(prefix, namePrefix string, limit any) {
		lim, ok := limit.(map[string]any)
		if !ok || lim == nil {
			return
		}
		primary := firstMap(lim, "primary_window", "primaryWindow")
		secondary := firstMap(lim, "secondary_window", "secondaryWindow")
		for _, item := range classifyCodexWindowList(prefix, namePrefix, primary, secondary) {
			if w := codexWindowFromMap(item.id, item.label, item.window); w != nil {
				result.Windows = append(result.Windows, *w)
			}
		}
	}

	rateLimit := firstMap(root, "rate_limit", "rateLimit")
	appendLimitWindows("code", "", rateLimit)

	codeReview := firstMap(root, "code_review_rate_limit", "codeReviewRateLimit")
	appendLimitWindows("code-review", "Code Review ", codeReview)

	// additional_rate_limits
	appendAdditional := func(arr []any) {
		for i, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name := firstString(m, "limit_name", "limitName", "metered_feature", "meteredFeature")
			if name == "" {
				name = fmt.Sprintf("additional-%d", i+1)
			}
			rl := firstMap(m, "rate_limit", "rateLimit")
			appendLimitWindows(fmt.Sprintf("add-%d", i), name+" ", rl)
		}
	}
	if arr, ok := root["additional_rate_limits"].([]any); ok {
		appendAdditional(arr)
	} else if arr, ok := root["additionalRateLimits"].([]any); ok {
		appendAdditional(arr)
	}

	if reset := firstMap(root, "rate_limit_reset_credits", "rateLimitResetCredits"); reset != nil {
		if n := firstFloat(reset, "available_count", "availableCount"); n != nil {
			result.Extras["rate_limit_reset_credits_available"] = *n
		}
	}

	if len(result.Windows) == 0 && result.Plan == "" {
		return nil, fmt.Errorf("empty usage payload")
	}
	return result, nil
}

const (
	codexFiveHourSeconds = 18000
	codexWeekSeconds     = 604800
	codexMonthMinSeconds = 28 * 24 * 3600
	codexMonthMaxSeconds = 31 * 24 * 3600
)

type classifiedCodexWindow struct {
	id     string
	label  string
	window map[string]any
}

// classifyCodexWindowList labels windows by limit_window_seconds.
// Falls back to primary/secondary names when duration is missing.
func classifyCodexWindowList(prefix, namePrefix string, primary, secondary map[string]any) []classifiedCodexWindow {
	type candidate struct {
		fallbackRole string // primary|secondary
		window       map[string]any
	}
	cands := []candidate{
		{fallbackRole: "primary", window: primary},
		{fallbackRole: "secondary", window: secondary},
	}

	out := make([]classifiedCodexWindow, 0, 2)
	used := map[string]bool{}
	for _, c := range cands {
		if c.window == nil {
			continue
		}
		kind := codexWindowKind(c.window, c.fallbackRole == "primary")
		idSuffix := kind
		if kind == "primary" || kind == "secondary" {
			idSuffix = c.fallbackRole
		}
		// Avoid duplicate ids if both windows somehow classify the same.
		id := prefix + "-" + idSuffix
		if used[id] {
			id = prefix + "-" + c.fallbackRole
		}
		used[id] = true
		out = append(out, classifiedCodexWindow{
			id:     id,
			label:  codexWindowLabel(namePrefix, kind),
			window: c.window,
		})
	}
	return out
}

func isCodexMonthWindowSeconds(secs float64) bool {
	return secs >= float64(codexMonthMinSeconds) && secs <= float64(codexMonthMaxSeconds)
}

func codexWindowSeconds(window map[string]any) *float64 {
	return firstFloat(window, "limit_window_seconds", "limitWindowSeconds")
}

func codexWindowKind(window map[string]any, isPrimaryFallback bool) string {
	if secs := codexWindowSeconds(window); secs != nil {
		switch {
		case int(*secs) == codexFiveHourSeconds:
			return "five-hour"
		case int(*secs) == codexWeekSeconds:
			return "weekly"
		case isCodexMonthWindowSeconds(*secs):
			return "monthly"
		}
	}
	if isPrimaryFallback {
		return "primary"
	}
	return "secondary"
}

func codexWindowLabel(namePrefix, kind string) string {
	base := "次窗口"
	switch kind {
	case "five-hour":
		base = "5 小时窗口"
	case "weekly":
		base = "周窗口"
	case "monthly":
		base = "月窗口"
	case "primary":
		base = "主窗口"
	case "secondary":
		base = "次窗口"
	}
	prefix := strings.TrimSpace(namePrefix)
	if prefix == "" {
		return base
	}
	switch kind {
	case "five-hour":
		return prefix + " 5 小时"
	case "weekly":
		return prefix + " 周窗口"
	case "monthly":
		return prefix + " 月窗口"
	case "primary":
		return prefix + " 主窗口"
	default:
		return prefix + " 次窗口"
	}
}

func firstCodexSubscriptionUntil(root map[string]any) string {
	// usage payload may embed subscription expiry under several shapes.
	candidates := []any{
		root["chatgpt_subscription_active_until"],
		root["chatgptSubscriptionActiveUntil"],
		root["subscription_active_until"],
		root["subscriptionActiveUntil"],
	}
	if sub := firstMap(root, "subscription"); sub != nil {
		candidates = append(candidates, sub["active_until"], sub["activeUntil"], sub["chatgpt_subscription_active_until"])
	}
	for _, c := range candidates {
		if s := normalizeAnyTimeToDisplay(c); s != "" {
			return s
		}
	}
	return ""
}

func normalizeAnyTimeToDisplay(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		b, _ := json.Marshal(t)
		return normalizeCodexTimeValue(b)
	case float64, float32, int, int64, json.Number:
		b, _ := json.Marshal(t)
		return normalizeCodexTimeValue(b)
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return normalizeCodexTimeValue(b)
	}
}

func codexWindowFromMap(id, label string, window map[string]any) *schema.ProviderQuotaWindow {
	if window == nil {
		return nil
	}
	used := firstFloat(window, "used_percent", "usedPercent")
	resetAt, resetLabel := resolveResetFields(window)
	w := &schema.ProviderQuotaWindow{
		ID:         id,
		Label:      label,
		ResetAt:    resetAt,
		ResetLabel: resetLabel,
	}
	if used != nil {
		u := clampPercent(*used)
		r := clampPercent(100 - u)
		w.UsedPercent = &u
		w.RemainingPercent = &r
	}
	return w
}

// resolveResetFields normalizes Codex/xAI reset metadata.
// Codex usage windows typically provide reset_at as unix seconds (e.g. 1785908987),
// or reset_after_seconds as a relative countdown.
func resolveResetFields(window map[string]any) (resetAtRFC3339 string, resetLabel string) {
	now := time.Now()

	// 1) Prefer absolute reset timestamp.
	if ts := parseTimeValue(window, "reset_at", "resetAt", "resets_at", "resetsAt", "reset_time", "resetTime"); ts != nil {
		resetAtRFC3339 = ts.UTC().Format(time.RFC3339)
		d := ts.Sub(now)
		if d <= 0 {
			return resetAtRFC3339, "已可重置"
		}
		return resetAtRFC3339, humanizeDuration(d)
	}

	// 2) Relative countdown seconds.
	if n := firstFloat(window, "reset_after_seconds", "resetAfterSeconds", "resets_in_seconds", "resetsInSeconds", "reset_after", "resetAfter"); n != nil && *n > 0 {
		target := now.Add(time.Duration(*n) * time.Second)
		return target.UTC().Format(time.RFC3339), humanizeDuration(time.Duration(*n) * time.Second)
	}

	// 3) Fallback bare string (already humanized or unknown).
	if s := strings.TrimSpace(firstString(window, "reset_after", "resetAfter")); s != "" && !isAllDigits(s) {
		return "", s
	}
	return "", "-"
}

func formatResetLabel(window map[string]any, resetAt string) string {
	// Keep helper for non-Codex callers (e.g. xAI amount windows).
	if resetAt != "" {
		if t, err := time.Parse(time.RFC3339, resetAt); err == nil {
			d := time.Until(t)
			if d <= 0 {
				return "已可重置"
			}
			return humanizeDuration(d)
		}
		// Unix seconds encoded as string.
		if isAllDigits(resetAt) {
			var sec int64
			if _, err := fmt.Sscanf(resetAt, "%d", &sec); err == nil && sec > 1_000_000_000 {
				t := time.Unix(sec, 0)
				d := time.Until(t)
				if d <= 0 {
					return "已可重置"
				}
				return humanizeDuration(d)
			}
		}
	}
	_, label := resolveResetFields(window)
	return label
}

func parseTimeValue(m map[string]any, keys ...string) *time.Time {
	if m == nil {
		return nil
	}
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			s := strings.TrimSpace(t)
			if s == "" {
				continue
			}
			if parsed, err := time.Parse(time.RFC3339, s); err == nil {
				return &parsed
			}
			// numeric string unix seconds/millis
			if isAllDigits(s) {
				var n int64
				if _, err := fmt.Sscanf(s, "%d", &n); err == nil {
					if ts := unixToTime(n); ts != nil {
						return ts
					}
				}
			}
		case float64:
			if ts := unixToTime(int64(t)); ts != nil {
				return ts
			}
		case float32:
			if ts := unixToTime(int64(t)); ts != nil {
				return ts
			}
		case int:
			if ts := unixToTime(int64(t)); ts != nil {
				return ts
			}
		case int64:
			if ts := unixToTime(t); ts != nil {
				return ts
			}
		case json.Number:
			if n, err := t.Int64(); err == nil {
				if ts := unixToTime(n); ts != nil {
					return ts
				}
			}
			if f, err := t.Float64(); err == nil {
				if ts := unixToTime(int64(f)); ts != nil {
					return ts
				}
			}
		}
	}
	return nil
}

func unixToTime(n int64) *time.Time {
	// Heuristic: ms vs sec.
	if n > 1_000_000_000_000 { // ms
		t := time.UnixMilli(n)
		return &t
	}
	if n > 1_000_000_000 { // sec
		t := time.Unix(n, 0)
		return &t
	}
	return nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func humanizeDuration(d time.Duration) string {
	if d < time.Minute {
		return "不到 1 分钟"
	}
	mins := int(math.Ceil(d.Minutes()))
	days := mins / (24 * 60)
	hours := (mins % (24 * 60)) / 60
	minutes := mins % 60
	if days > 0 {
		return fmt.Sprintf("%d 天 %d 小时", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%d 小时 %d 分钟", hours, minutes)
	}
	return fmt.Sprintf("%d 分钟", minutes)
}

// ---- xAI parse ----

type xaiBillingSummary struct {
	PeriodType         string
	UsagePercent       *float64
	PeriodEnd          string
	MonthlyLimitCents  *float64
	UsedCents          *float64
	IncludedUsedCents  *float64
	OnDemandCapCents   *float64
	OnDemandUsedCents  *float64
	OnDemandUsedPercent *float64
	UsedPercent        *float64
	BillingPeriodEnd   string
	ProductUsage       []struct {
		Product      string
		UsagePercent *float64
	}
}

func buildXAIBillingSummary(body []byte) *xaiBillingSummary {
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil
	}
	config := firstMap(root, "config")
	if config == nil {
		// Some payloads are the config itself.
		if _, ok := root["monthlyLimit"]; ok || root["monthly_limit"] != nil || root["creditUsagePercent"] != nil || root["credit_usage_percent"] != nil {
			config = root
		}
	}
	if config == nil {
		return nil
	}

	summary := &xaiBillingSummary{PeriodType: "unknown"}
	currentPeriod := firstMap(config, "currentPeriod", "current_period")
	periodType := firstString(currentPeriod, "type", "period_type", "periodType")
	if periodType == "" {
		periodType = "unknown"
	}
	creditUsage := firstFloat(config, "creditUsagePercent", "credit_usage_percent")
	periodEnd := firstString(currentPeriod, "end")
	if periodEnd == "" {
		periodEnd = firstString(config, "billingPeriodEnd", "billing_period_end")
	}

	// xAI amounts are often objects: {"val": 15000} (cents), not bare numbers.
	monthlyLimit := firstXAICent(config, "monthlyLimit", "monthly_limit")
	used := firstXAICent(config, "used")
	onDemandCap := firstXAICent(config, "onDemandCap", "on_demand_cap")
	onDemandUsed := firstXAICent(config, "onDemandUsed", "on_demand_used")
	billingPeriodEnd := firstString(config, "billingPeriodEnd", "billing_period_end")

	var includedUsed *float64
	if used != nil {
		if monthlyLimit != nil && *monthlyLimit > 0 {
			v := math.Min(*used, *monthlyLimit)
			includedUsed = &v
		} else {
			includedUsed = used
		}
	}
	if onDemandUsed == nil && used != nil && monthlyLimit != nil {
		v := math.Max(0, *used-*monthlyLimit)
		onDemandUsed = &v
	}
	var usedPercent *float64
	if monthlyLimit != nil && *monthlyLimit > 0 && includedUsed != nil {
		v := (*includedUsed / *monthlyLimit) * 100
		usedPercent = &v
	}
	var onDemandUsedPercent *float64
	if onDemandCap != nil && *onDemandCap > 0 && onDemandUsed != nil {
		v := (*onDemandUsed / *onDemandCap) * 100
		onDemandUsedPercent = &v
	}

	// product usage
	if arr, ok := config["productUsage"].([]any); ok {
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			product := firstString(m, "product", "name")
			if product == "" {
				continue
			}
			summary.ProductUsage = append(summary.ProductUsage, struct {
				Product      string
				UsagePercent *float64
			}{Product: product, UsagePercent: firstFloat(m, "usagePercent", "usage_percent")})
		}
	} else if arr, ok := config["product_usage"].([]any); ok {
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			product := firstString(m, "product", "name")
			if product == "" {
				continue
			}
			summary.ProductUsage = append(summary.ProductUsage, struct {
				Product      string
				UsagePercent *float64
			}{Product: product, UsagePercent: firstFloat(m, "usagePercent", "usage_percent")})
		}
	}

	hasWeekly := creditUsage != nil || periodType == "weekly" || len(summary.ProductUsage) > 0
	hasMonthly := monthlyLimit != nil || used != nil || (!hasWeekly && (onDemandCap != nil || billingPeriodEnd != ""))
	if !hasWeekly && !hasMonthly {
		return nil
	}

	if hasWeekly {
		if periodType == "unknown" {
			summary.PeriodType = "weekly"
		} else {
			summary.PeriodType = periodType
		}
		summary.UsagePercent = creditUsage
		summary.PeriodEnd = periodEnd
	} else {
		summary.PeriodType = "monthly"
		summary.UsagePercent = usedPercent
		summary.PeriodEnd = billingPeriodEnd
	}
	summary.MonthlyLimitCents = monthlyLimit
	summary.UsedCents = used
	summary.IncludedUsedCents = includedUsed
	summary.OnDemandCapCents = onDemandCap
	summary.OnDemandUsedCents = onDemandUsed
	summary.OnDemandUsedPercent = onDemandUsedPercent
	summary.UsedPercent = usedPercent
	summary.BillingPeriodEnd = billingPeriodEnd
	return summary
}

func mergeXAIBillingSummaries(primary, fallback *xaiBillingSummary) *xaiBillingSummary {
	if primary == nil {
		return fallback
	}
	if fallback == nil {
		return primary
	}
	out := *primary
	if out.UsagePercent == nil {
		out.UsagePercent = fallback.UsagePercent
	}
	if out.PeriodEnd == "" {
		out.PeriodEnd = fallback.PeriodEnd
	}
	if out.MonthlyLimitCents == nil {
		out.MonthlyLimitCents = fallback.MonthlyLimitCents
	}
	if out.UsedCents == nil {
		out.UsedCents = fallback.UsedCents
	}
	if out.IncludedUsedCents == nil {
		out.IncludedUsedCents = fallback.IncludedUsedCents
	}
	if out.OnDemandCapCents == nil {
		out.OnDemandCapCents = fallback.OnDemandCapCents
	}
	if out.OnDemandUsedCents == nil {
		out.OnDemandUsedCents = fallback.OnDemandUsedCents
	}
	if out.OnDemandUsedPercent == nil {
		out.OnDemandUsedPercent = fallback.OnDemandUsedPercent
	}
	if out.UsedPercent == nil {
		out.UsedPercent = fallback.UsedPercent
	}
	if out.BillingPeriodEnd == "" {
		out.BillingPeriodEnd = fallback.BillingPeriodEnd
	}
	if len(out.ProductUsage) == 0 {
		out.ProductUsage = fallback.ProductUsage
	}
	return &out
}

func xaiSummaryToQuotaResult(s *xaiBillingSummary) *schema.ProviderQuotaResult {
	result := &schema.ProviderQuotaResult{
		Provider: oauthProviderXAI,
		Windows:  []schema.ProviderQuotaWindow{},
		Extras:   map[string]any{"mode": "billing", "period_type": s.PeriodType},
	}
	// Plan heuristic from monthly limit cents
	if s.MonthlyLimitCents != nil {
		switch int(*s.MonthlyLimitCents) {
		case 15000:
			result.Plan = "supergrok"
		case 150000:
			result.Plan = "supergrok_heavy"
		}
	}

	addWindow := func(id, label string, usedPct *float64, resetAt, amount string) {
		w := schema.ProviderQuotaWindow{ID: id, Label: label, AmountLabel: amount}
		if usedPct != nil {
			u := clampPercent(*usedPct)
			r := clampPercent(100 - u)
			w.UsedPercent = &u
			w.RemainingPercent = &r
		}
		if resetAt != "" {
			// Accept RFC3339 or unix seconds string.
			labelText := formatResetLabel(map[string]any{"reset_at": resetAt}, resetAt)
			if ts := parseTimeValue(map[string]any{"reset_at": resetAt}, "reset_at"); ts != nil {
				w.ResetAt = ts.UTC().Format(time.RFC3339)
			} else {
				w.ResetAt = resetAt
			}
			w.ResetLabel = labelText
		}
		result.Windows = append(result.Windows, w)
	}

	if s.PeriodType == "weekly" || s.UsagePercent != nil {
		addWindow("weekly", "周限额", s.UsagePercent, s.PeriodEnd, "")
	}
	for i, p := range s.ProductUsage {
		addWindow(fmt.Sprintf("product-%d", i), p.Product+" 使用", p.UsagePercent, "", "")
	}
	if s.OnDemandCapCents != nil && *s.OnDemandCapCents > 0 {
		amount := formatUSDRange(s.OnDemandUsedCents, s.OnDemandCapCents)
		addWindow("on-demand", "按量付费", s.OnDemandUsedPercent, "", amount)
	}
	if s.MonthlyLimitCents != nil || s.UsedPercent != nil {
		amount := formatUSDRemaining(s.IncludedUsedCents, s.MonthlyLimitCents)
		reset := s.BillingPeriodEnd
		if reset == "" {
			reset = s.PeriodEnd
		}
		addWindow("monthly", "月额度", s.UsedPercent, reset, amount)
	}
	return result
}

// formatResetAtLabel formats an RFC3339/ISO timestamp or unix-seconds string.

func formatUSDFromCents(cents *float64) string {
	if cents == nil {
		return "--"
	}
	return fmt.Sprintf("$%.2f", *cents/100)
}

func formatUSDRange(used, cap *float64) string {
	if cap == nil {
		return formatUSDFromCents(used)
	}
	remaining := *cap
	if used != nil {
		remaining = math.Max(0, *cap-*used)
	}
	return fmt.Sprintf("%s / %s", formatUSDFromCents(&remaining), formatUSDFromCents(cap))
}

func formatUSDRemaining(includedUsed, monthlyLimit *float64) string {
	if monthlyLimit == nil {
		return formatUSDFromCents(includedUsed)
	}
	remaining := *monthlyLimit
	if includedUsed != nil {
		remaining = math.Max(0, *monthlyLimit-*includedUsed)
	}
	return fmt.Sprintf("%s / %s", formatUSDFromCents(&remaining), formatUSDFromCents(monthlyLimit))
}

// ---- helpers ----

func firstMap(m map[string]any, keys ...string) map[string]any {
	if m == nil {
		return nil
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if mm, ok := v.(map[string]any); ok {
				return mm
			}
		}
	}
	return nil
}

func firstString(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					return s
				}
			case float64:
				return fmt.Sprintf("%.0f", t)
			case json.Number:
				return t.String()
			}
		}
	}
	return ""
}

func firstFloat(m map[string]any, keys ...string) *float64 {
	if m == nil {
		return nil
	}
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			if f := anyToFloat(v); f != nil {
				return f
			}
		}
	}
	return nil
}

// firstXAICent reads xAI money fields that may be:
// - number: 15000
// - string: "15000"
// - object: {"val": 15000}
func firstXAICent(m map[string]any, keys ...string) *float64 {
	if m == nil {
		return nil
	}
	for _, k := range keys {
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		if f := anyToFloat(v); f != nil {
			return f
		}
		if obj, ok := v.(map[string]any); ok {
			if f := anyToFloat(obj["val"]); f != nil {
				return f
			}
			if f := anyToFloat(obj["value"]); f != nil {
				return f
			}
			if f := anyToFloat(obj["amount"]); f != nil {
				return f
			}
		}
	}
	return nil
}

func anyToFloat(v any) *float64 {
	switch t := v.(type) {
	case float64:
		return &t
	case float32:
		f := float64(t)
		return &f
	case int:
		f := float64(t)
		return &f
	case int64:
		f := float64(t)
		return &f
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return &f
		}
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil
		}
		var f float64
		if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
			return &f
		}
	}
	return nil
}

func clampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
