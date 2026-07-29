package biz

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/cachex"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Provider business logic layer
type Provider struct {
	Trans             *util.Trans
	Cache             cachex.Cacher
	ProviderDAL       *dal.Provider
	EndpointDAL       *dal.Endpoint
	DataPermissionBIZ *DataPermission
	ConfigRedisSync   *ConfigRedisSync
	AuditLogBIZ       *opsBiz.AuditLog
}

func (p *Provider) Query(ctx context.Context, params schema.ProviderQueryParam) (*schema.ProviderQueryResult, error) {
	params.Pagination = true

	result, err := p.ProviderDAL.Query(ctx, params, schema.ProviderQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Get the specified provider.
func (p *Provider) Get(ctx context.Context, id string) (*schema.Provider, error) {
	provider, err := p.ProviderDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if provider == nil {
		return nil, errors.NotFound("", "Provider not found")
	}

	if !util.FromIsRootUser(ctx) {
		ok, err := p.DataPermissionBIZ.HasReadPermission(ctx, schema.DataPermissionTypeProvider, id)
		if err != nil {
			return nil, err
		} else if !ok {
			return nil, errors.NotFound("", "Provider not found")
		}
	}

	return provider, nil
}

// Create a new provider.
func (p *Provider) Create(ctx context.Context, formItem *schema.ProviderForm) (*schema.Provider, error) {
	if exists, err := p.ProviderDAL.ExistsCode(ctx, formItem.Code); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Provider code already exists")
	}

	provider := &schema.Provider{
		ID:        util.NewXID(),
		Creator:   util.FromUsername(ctx),
		CreatedAt: time.Now(),
	}
	if err := formItem.FillTo(provider); err != nil {
		return nil, err
	}

	err := p.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := p.ProviderDAL.Create(ctx, provider); err != nil {
			return err
		}
		return p.DataPermissionBIZ.CreateByOwner(ctx, schema.DataPermissionTypeProvider, provider.ID, util.FromTenant(ctx))
	})
	if err != nil {
		return nil, err
	}
	p.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypeProvider, provider.ID, provider.Name, nil, provider)
	return provider, nil
}

// Update the specified provider.
func (p *Provider) Update(ctx context.Context, id string, formItem *schema.ProviderForm) error {
	provider, err := p.ProviderDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if provider == nil {
		return errors.NotFound("", "Provider not found")
	} else if provider.Code != formItem.Code {
		if exists, err := p.ProviderDAL.ExistsCode(ctx, formItem.Code); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Provider code already exists")
		}
	}

	beforeProvider := *provider

	// Calculate APIKey changes (index-aligned)
	oldKeys := beforeProvider.GetApiKeys()
	newKeys := formItem.ApiKeys
	changeMap := make(map[string]string)

	minLen := len(oldKeys)
	if len(newKeys) < minLen {
		minLen = len(newKeys)
	}

	for i := 0; i < minLen; i++ {
		oldVal := strings.TrimSpace(oldKeys[i].Value)
		newVal := strings.TrimSpace(newKeys[i].Value)
		if oldVal != "" && newVal != "" && oldVal != newVal {
			changeMap[oldVal] = newVal
		}
	}

	if err := formItem.FillTo(provider); err != nil {
		return err
	}
	provider.Modifier = util.FromUsername(ctx)
	provider.UpdatedAt = time.Now()

	err = p.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := p.ProviderDAL.Update(ctx, provider); err != nil {
			return err
		}
		if p.EndpointDAL != nil && len(changeMap) > 0 {
			for oldKey, newKey := range changeMap {
				if err := p.EndpointDAL.UpdateApiKeyByProviderAndOldKey(ctx, provider.ID, oldKey, newKey); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err == nil {
		_ = p.ConfigRedisSync.SyncProviderID(ctx, provider.ID)
		p.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypeProvider, provider.ID, provider.Name, beforeProvider, provider)
	}
	return err
}

// Delete the specified provider.
func (p *Provider) Delete(ctx context.Context, id string) error {
	provider, err := p.ProviderDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if provider == nil {
		return errors.NotFound("", "Provider not found")
	}

	if err := p.ensureProviderCanDelete(ctx, id); err != nil {
		return err
	}

	err = p.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := p.ProviderDAL.Delete(ctx, id); err != nil {
			return err
		}
		return p.DataPermissionBIZ.DeleteByTypeAndDataId(ctx, schema.DataPermissionTypeProvider, id)
	})
	if err == nil {
		p.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypeProvider, provider.ID, provider.Name, provider, nil)
	}
	return err
}

func (p *Provider) ensureProviderCanDelete(ctx context.Context, id string) error {
	var count int64
	endpointTable := config.C.FormatTableName("endpoint")
	err := util.GetDB(ctx, p.ProviderDAL.DB).
		Table(endpointTable).
		Where("provider_id = ? AND deleted = '0'", id).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.BadRequest("", "供应商存在关联端点，请先清理后再执行删除操作")
	}
	return nil
}

// FetchModels calls the upstream provider's /v1/models API and returns the model list.
func (p *Provider) FetchModels(ctx context.Context, providerID string, formItem *schema.FetchModelsForm) (*schema.FetchModelsResult, error) {
	provider, err := p.ProviderDAL.Get(ctx, providerID)
	if err != nil {
		return nil, err
	} else if provider == nil {
		return nil, errors.NotFound("", "Provider not found")
	}

	// If the provider uses OAuth and the token is expiring (or expired), refresh it first!
	if provider.AuthType == "oauth_token" {
		cred := provider.GetOAuth()
		if cred != nil && cred.RefreshToken != "" {
			now := time.Now()
			if cred.ExpiresAt == nil || cred.ExpiresAt.Before(now.Add(5*time.Minute)) {
				// Trigger a synchronous refresh using the DB-locked TokenRefresher
				refresher := NewTokenRefresher(p.ProviderDAL.DB, p.ConfigRedisSync)
				refresher.lockAndRefreshProvider(ctx, *provider)

				// Reload the provider from DB to obtain the newly refreshed access token
				refreshedProvider, err := p.ProviderDAL.Get(ctx, providerID)
				if err == nil && refreshedProvider != nil {
					provider = refreshedProvider
				}
			}
		}
	}

	apiKey := formItem.APIKey
	if apiKey == "" {
		keys := provider.GetApiKeys()
		if len(keys) > 0 {
			apiKey = keys[0].Value
		}
	}

	// Build the upstream URL
	baseURL := strings.TrimRight(formItem.BaseURL, "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(provider.URL, "/")
	}
	// Codex OAuth defaults to the ChatGPT backend base when URL is empty.
	if baseURL == "" && isCodexOAuthProvider(provider) {
		baseURL = codexDefaultBaseURL
	}
	if baseURL == "" {
		return nil, errors.BadRequest("", "Base URL is required, please provide it or set provider URL")
	}
	var upstreamModels []schema.UpstreamModel

	// Codex ChatGPT backend uses a non-OpenAI models catalog response.
	if isCodexOAuthProvider(provider) || isCodexModelsBaseURL(baseURL) {
		models, err := p.fetchCodexModels(ctx, provider, baseURL, apiKey)
		if err != nil {
			return nil, err
		}
		return &schema.FetchModelsResult{Models: models}, nil
	}

	if provider.Protocol == "joycode" {
		var reqURL string
		if strings.HasPrefix(baseURL, "https://") {
			var err error
			reqURL, err = signJoyCodeGatewayURL(baseURL, "joycode_modelList")
			if err != nil {
				return nil, errors.BadRequest("", "JoyCode 签名失败: %v", err)
			}
		} else {
			reqURL = baseURL + "/api/saas/models/v2/modelList"
		}

		reqBody := injectJoyCodePayload([]byte("{}"))
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(string(reqBody)))
		if err != nil {
			return nil, errors.BadRequest("", "Failed to create request: %s", err.Error())
		}
		req.Header.Set("ptKey", apiKey)
		req.Header.Set("loginType", getLoginTypeForPtKey(apiKey))
		req.Header.Set("x-ms-client-request-id", uuid.NewString())
		req.Header.Set("client", "JoyCodeIDE")
		req.Header.Set("clientVersion", "3.8.61")
		req.Header.Set("Content-Type", "application/json; charset=UTF-8")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, errors.BadRequest("", "Failed to call upstream: %s", err.Error())
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, errors.BadRequest("", "Upstream returned status %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.BadRequest("", "Failed to read upstream response body: %s", err.Error())
		}

		var joycodeResp struct {
			Code interface{} `json:"code"`
			Data []struct {
				ChatApiModel string `json:"chatApiModel"`
				Label        string `json:"label"`
			} `json:"data"`
			Msg     string `json:"msg"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &joycodeResp); err != nil {
			return nil, errors.BadRequest("", "Failed to parse upstream response: %s (Raw body: %s)", err.Error(), string(body))
		}

		isSuccess := false
		if joycodeResp.Code != nil {
			switch val := joycodeResp.Code.(type) {
			case float64:
				if val == 0 {
					isSuccess = true
				}
			case string:
				if val == "0" || val == "200" || val == "" {
					isSuccess = true
				}
			}
		} else {
			isSuccess = true
		}

		if !isSuccess {
			errMsg := joycodeResp.Msg
			if errMsg == "" {
				errMsg = joycodeResp.Message
			}
			if errMsg == "" {
				errMsg = string(body)
			}

			// 如果是接口不存在的报错，说明上游网关未注册 modelList API，进行优雅降级，返回内置的默认模型列表
			if strings.Contains(strings.ToLower(errMsg), "the current api does not exist") ||
				strings.Contains(strings.ToLower(errMsg), "apidoesnotexist") {
				defaultModels := []string{
					"joyai-code",
					"kimi-k2",
					"deepseek-v3.1",
					"doubao-seed",
				}
				for _, mID := range defaultModels {
					upstreamModels = append(upstreamModels, schema.UpstreamModel{
						ID:      mID,
						Object:  "model",
						OwnedBy: "jd",
					})
				}
				return &schema.FetchModelsResult{Models: upstreamModels}, nil
			}

			return nil, errors.BadRequest("", "JoyCode 上游业务报错 (%v): %s", joycodeResp.Code, errMsg)
		}

		for _, m := range joycodeResp.Data {
			modelID := m.ChatApiModel
			if modelID == "" {
				modelID = m.Label
			}
			if modelID != "" {
				upstreamModels = append(upstreamModels, schema.UpstreamModel{
					ID:      modelID,
					Object:  "model",
					OwnedBy: "jd",
				})
			}
		}
	} else {
		reqURL := baseURL + "/models"
		if provider.Protocol == "xai" || strings.Contains(strings.ToLower(baseURL), "api.x.ai") {
			if !strings.HasSuffix(baseURL, "/v1") {
				reqURL = baseURL + "/v1/models"
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, errors.BadRequest("", "Failed to create request: %s", err.Error())
		}
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
		req.Header.Set("Accept", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, errors.BadRequest("", "Failed to call upstream: %s", err.Error())
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, errors.BadRequest("", "Upstream returned status %d: %s", resp.StatusCode, string(body))
		}

		var modelsResp struct {
			Data []schema.UpstreamModel `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
			return nil, errors.BadRequest("", "Failed to parse upstream response: %s", err.Error())
		}
		upstreamModels = modelsResp.Data
	}

	return &schema.FetchModelsResult{Models: upstreamModels}, nil
}

const (
	// Match CLIProxyAPI cmd/fetch_codex_models defaults.
	codexModelsClientVersion = "0.144.1"
	codexModelsUserAgent     = "codex_cli_rs/0.144.1 (Mac OS 26.3.1; arm64) iTerm.app/3.6.9"
	codexModelsOriginator    = "codex_cli_rs"
)

func isCodexOAuthProvider(provider *schema.Provider) bool {
	if provider == nil {
		return false
	}
	if isCodexModelsBaseURL(provider.URL) {
		return true
	}
	if provider.AuthType != "oauth_token" {
		return false
	}
	cred := provider.GetOAuth()
	if cred == nil {
		return false
	}
	endpoint := strings.ToLower(cred.TokenEndpoint)
	if strings.Contains(endpoint, "auth.openai.com") || strings.Contains(endpoint, "openai.com") {
		return true
	}
	return strings.TrimSpace(cred.AccountID) != "" && strings.Contains(strings.ToLower(provider.URL), "chatgpt.com")
}

func isCodexModelsBaseURL(baseURL string) bool {
	u := strings.ToLower(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	return strings.Contains(u, "chatgpt.com/backend-api/codex")
}

// fetchCodexModels calls ChatGPT Codex backend:
//
//	GET {base}/models?client_version=...
//
// with Bearer token + Chatgpt-Account-Id, then maps {models:[{slug}]} to UpstreamModel.
func (p *Provider) fetchCodexModels(ctx context.Context, provider *schema.Provider, baseURL, apiKey string) ([]schema.UpstreamModel, error) {
	reqURL, err := buildCodexModelsURL(baseURL, codexModelsClientVersion)
	if err != nil {
		return nil, errors.BadRequest("", "Invalid Codex base URL: %s", err.Error())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.BadRequest("", "Failed to create request: %s", err.Error())
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", codexModelsUserAgent)
	req.Header.Set("Originator", codexModelsOriginator)
	if provider != nil {
		if cred := provider.GetOAuth(); cred != nil && strings.TrimSpace(cred.AccountID) != "" {
			req.Header.Set("Chatgpt-Account-Id", strings.TrimSpace(cred.AccountID))
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.BadRequest("", "Failed to call Codex models API: %s", err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.BadRequest("", "Failed to read Codex models response: %s", err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.BadRequest("", "Codex models API returned status %d: %s", resp.StatusCode, string(body))
	}

	models, err := parseCodexModelsResponse(body)
	if err != nil {
		return nil, errors.BadRequest("", "Failed to parse Codex models response: %s", err.Error())
	}
	if len(models) == 0 {
		return nil, errors.BadRequest("", "Codex models API returned an empty model list")
	}
	return models, nil
}

func buildCodexModelsURL(baseURL, clientVersion string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = codexDefaultBaseURL
	}
	u, err := url.Parse(base + "/models")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(clientVersion) != "" {
		q := u.Query()
		q.Set("client_version", strings.TrimSpace(clientVersion))
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func parseCodexModelsResponse(body []byte) ([]schema.UpstreamModel, error) {
	// Primary shape from chatgpt.com/backend-api/codex/models
	var payload struct {
		Models []struct {
			Slug        string `json:"slug"`
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"models"`
		// Fallback: OpenAI-compatible envelope, just in case.
		Data []schema.UpstreamModel `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	out := make([]schema.UpstreamModel, 0, len(payload.Models)+len(payload.Data))
	seen := make(map[string]struct{})
	appendModel := func(id, ownedBy string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		if ownedBy == "" {
			ownedBy = "openai"
		}
		out = append(out, schema.UpstreamModel{
			ID:      id,
			Object:  "model",
			OwnedBy: ownedBy,
		})
	}

	for _, m := range payload.Models {
		id := m.Slug
		if id == "" {
			id = m.ID
		}
		appendModel(id, "openai")
	}
	for _, m := range payload.Data {
		appendModel(m.ID, m.OwnedBy)
	}
	return out, nil
}

const (
	providerOAuthSessionNS = "provider_oauth_session"
	providerOAuthResultNS  = "provider_oauth_result"
	providerOAuthTTL       = 10 * time.Minute

	// xaiOAuthClientID is xAI's public Grok CLI OAuth client. It only permits the
	// device-authorization grant (RFC 8628); a browser authorization-code redirect
	// to localhost is rejected with HTTP 403.
	xaiOAuthClientID   = "b1a00492-073a-47ea-816f-4c329264a828"
	xaiOAuthScope      = "openid profile email offline_access grok-cli:access api:access"
	xaiDeviceCodeGrant = "urn:ietf:params:oauth:grant-type:device_code"
	xaiDiscoveryURL    = "https://auth.x.ai/.well-known/openid-configuration"
	// xaiDefaultBaseURL is the official xAI API base URL used to call the model.
	xaiDefaultBaseURL = "https://api.x.ai/v1"

	// Codex / OpenAI ChatGPT CLI OAuth (authorization-code + PKCE).
	// Redirect URI is fixed by the public CLI client and cannot be changed.
	codexOAuthAuthURL     = "https://auth.openai.com/oauth/authorize"
	codexOAuthTokenURL    = "https://auth.openai.com/oauth/token"
	codexOAuthClientID    = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexOAuthRedirectURI = "http://localhost:1455/auth/callback"
	codexOAuthScope       = "openid email profile offline_access"
	codexDefaultBaseURL   = "https://chatgpt.com/backend-api/codex"

	oauthProviderXAI   = "xai"
	oauthProviderCodex = "codex"
)

type OAuthTokenResult struct {
	AccessToken             string `json:"access_token"`
	RefreshToken            string `json:"refresh_token"`
	ExpiresIn               int    `json:"expires_in"`
	TokenEndpoint           string `json:"token_endpoint"`
	BaseURL                 string `json:"base_url"`
	AccountID               string `json:"account_id,omitempty"`
	Email                   string `json:"email,omitempty"`
	SubscriptionActiveUntil string `json:"subscription_active_until,omitempty"`
	Provider                string `json:"provider,omitempty"`
}

// OAuthStartResult carries the details returned to the frontend so it can open
// the authorization page and finish the flow (poll for xAI, paste callback for Codex).
type OAuthStartResult struct {
	URL      string `json:"url"`
	UserCode string `json:"user_code,omitempty"`
	State    string `json:"state"`
	// Flow is "device_code" (xAI) or "authorization_code" (Codex).
	Flow string `json:"flow,omitempty"`
}

type xaiDiscovery struct {
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	TokenEndpoint               string `json:"token_endpoint"`
}

type providerOAuthSession struct {
	Provider     string `json:"provider"`
	UserID       string `json:"user_id"`
	CodeVerifier string `json:"code_verifier,omitempty"`
}

// StartOAuthFlow starts an upstream provider OAuth binding flow.
// xAI uses device-code; Codex uses authorization-code + PKCE with manual callback paste.
func (p *Provider) StartOAuthFlow(ctx context.Context, providerName string) (*OAuthStartResult, error) {
	providerName = strings.ToLower(strings.TrimSpace(providerName))
	switch providerName {
	case oauthProviderXAI:
		return p.startXAIOAuthFlow(ctx)
	case oauthProviderCodex:
		return p.startCodexOAuthFlow(ctx)
	default:
		return nil, errors.BadRequest("", "Unsupported oauth provider: %s", providerName)
	}
}

func (p *Provider) startXAIOAuthFlow(ctx context.Context) (*OAuthStartResult, error) {
	userID := util.FromUserID(ctx)
	if userID == "" {
		return nil, errors.Unauthorized("", "Login required")
	}
	if p.Cache == nil {
		return nil, errors.InternalServerError("", "OAuth session cache is not configured")
	}

	clientID := getEnvWithDefault("OAUTH_XAI_CLIENT_ID", xaiOAuthClientID)

	disc, err := discoverXAIEndpoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("xai oauth discovery failed: %w", err)
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", xaiOAuthScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, disc.DeviceAuthorizationEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create device code request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device code request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read device code response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var dc struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}
	if err := json.Unmarshal(respBody, &dc); err != nil {
		return nil, fmt.Errorf("failed to parse device code response: %w", err)
	}
	if dc.DeviceCode == "" {
		return nil, errors.InternalServerError("", "device code response missing device_code")
	}

	openURL := dc.VerificationURIComplete
	if openURL == "" {
		openURL = dc.VerificationURI
	}

	state := generateStateKey()
	if err := p.saveOAuthSession(ctx, state, providerOAuthSession{
		Provider: oauthProviderXAI,
		UserID:   userID,
	}); err != nil {
		return nil, err
	}

	// Poll the token endpoint in the background; the result lands in cache
	// keyed by state, where PollOAuthStatus picks it up.
	go p.pollDeviceToken(clientID, disc.TokenEndpoint, dc.DeviceCode, state, userID, dc.Interval, dc.ExpiresIn)

	return &OAuthStartResult{
		URL:      openURL,
		UserCode: dc.UserCode,
		State:    state,
		Flow:     "device_code",
	}, nil
}

func (p *Provider) startCodexOAuthFlow(ctx context.Context) (*OAuthStartResult, error) {
	userID := util.FromUserID(ctx)
	if userID == "" {
		return nil, errors.Unauthorized("", "Login required")
	}
	if p.Cache == nil {
		return nil, errors.InternalServerError("", "OAuth session cache is not configured")
	}

	pkce, err := generatePKCECodes()
	if err != nil {
		return nil, errors.InternalServerError("", "failed to generate PKCE codes")
	}
	state := generateStateKey()
	if err := p.saveOAuthSession(ctx, state, providerOAuthSession{
		Provider:     oauthProviderCodex,
		UserID:       userID,
		CodeVerifier: pkce.CodeVerifier,
	}); err != nil {
		return nil, err
	}

	clientID := getEnvWithDefault("OAUTH_CODEX_CLIENT_ID", codexOAuthClientID)
	params := url.Values{
		"client_id":                  {clientID},
		"response_type":              {"code"},
		"redirect_uri":               {codexOAuthRedirectURI},
		"scope":                      {codexOAuthScope},
		"state":                      {state},
		"code_challenge":             {pkce.CodeChallenge},
		"code_challenge_method":      {"S256"},
		"prompt":                     {"login"},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
	}
	authURL := codexOAuthAuthURL + "?" + params.Encode()

	return &OAuthStartResult{
		URL:   authURL,
		State: state,
		Flow:  "authorization_code",
	}, nil
}

// pollDeviceToken exchanges the device code for tokens once the user authorizes,
// then stores the result keyed by state. It runs detached from the request context.
func (p *Provider) pollDeviceToken(clientID, tokenEndpoint, deviceCode, state, userID string, interval, expiresIn int) {
	ctx := context.Background()

	if interval <= 0 {
		interval = 5
	}
	if expiresIn <= 0 {
		expiresIn = 600
	}
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)
	wait := time.Duration(interval) * time.Second

	client := &http.Client{Timeout: 30 * time.Second}

	for time.Now().Before(deadline) {
		time.Sleep(wait)

		// Bail out early if the session was consumed/expired.
		if _, ok, _ := p.Cache.Get(ctx, providerOAuthSessionNS, state); !ok {
			return
		}

		form := url.Values{}
		form.Set("grant_type", xaiDeviceCodeGrant)
		form.Set("device_code", deviceCode)
		form.Set("client_id", clientID)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var payload struct {
			Error        string `json:"error"`
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int    `json:"expires_in"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			continue
		}

		if payload.Error != "" {
			switch payload.Error {
			case "authorization_pending":
				continue
			case "slow_down":
				wait += 5 * time.Second
				continue
			default:
				// expired_token, access_denied, or any other terminal error
				return
			}
		}

		if payload.AccessToken != "" {
			_ = p.saveOAuthResult(ctx, state, &OAuthTokenResult{
				AccessToken:   payload.AccessToken,
				RefreshToken:  payload.RefreshToken,
				ExpiresIn:     payload.ExpiresIn,
				TokenEndpoint: tokenEndpoint,
				BaseURL:       xaiDefaultBaseURL,
				Provider:      oauthProviderXAI,
			})
			return
		}
	}
}

func (p *Provider) PollOAuthStatus(ctx context.Context, state string) (*OAuthTokenResult, error) {
	if strings.TrimSpace(state) == "" {
		return nil, errors.BadRequest("", "state is required")
	}
	userID := util.FromUserID(ctx)
	if userID == "" {
		return nil, errors.Unauthorized("", "Login required")
	}
	if p.Cache == nil {
		return nil, errors.InternalServerError("", "OAuth session cache is not configured")
	}

	// Ensure the state belongs to the current user before consuming the result.
	session, err := p.getOAuthSession(ctx, state)
	if err != nil {
		return nil, err
	}
	if session == nil {
		// Session may already be consumed after a previous success; still try result.
	} else if session.UserID != userID {
		return nil, errors.Forbidden("", "OAuth state does not belong to current user")
	} else if session.Provider != oauthProviderXAI {
		return nil, errors.BadRequest("", "OAuth status polling is only supported for device-code providers")
	}

	result, err := p.getAndDeleteOAuthResult(ctx, state)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil // still waiting
	}

	// Consume the session once the token is delivered.
	_, _, _ = p.Cache.GetAndDelete(ctx, providerOAuthSessionNS, state)
	return result, nil
}

// CompleteOAuthFlow finishes authorization-code OAuth (Codex) using the pasted callback URL.
func (p *Provider) CompleteOAuthFlow(ctx context.Context, formItem *schema.OAuthCompleteForm) (*OAuthTokenResult, error) {
	if formItem == nil {
		return nil, errors.BadRequest("", "request body is required")
	}
	providerName := strings.ToLower(strings.TrimSpace(formItem.Provider))
	state := strings.TrimSpace(formItem.State)
	callbackURL := strings.TrimSpace(formItem.CallbackURL)
	if providerName == "" || state == "" || callbackURL == "" {
		return nil, errors.BadRequest("", "provider, state and callback_url are required")
	}
	if providerName != oauthProviderCodex {
		return nil, errors.BadRequest("", "Complete OAuth is only supported for provider: %s", oauthProviderCodex)
	}

	userID := util.FromUserID(ctx)
	if userID == "" {
		return nil, errors.Unauthorized("", "Login required")
	}
	if p.Cache == nil {
		return nil, errors.InternalServerError("", "OAuth session cache is not configured")
	}

	// Validate ownership before consuming the one-time session.
	session, err := p.getOAuthSession(ctx, state)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, errors.BadRequest("", "Invalid or expired OAuth state")
	}
	if session.UserID != userID {
		return nil, errors.Forbidden("", "OAuth state does not belong to current user")
	}
	if session.Provider != providerName {
		return nil, errors.BadRequest("", "OAuth provider does not match state")
	}
	if session.CodeVerifier == "" {
		return nil, errors.BadRequest("", "OAuth session missing PKCE verifier")
	}

	code, callbackState, callbackErr, err := parseOAuthCallbackURL(callbackURL)
	if err != nil {
		return nil, errors.BadRequest("", "invalid callback_url: %s", err.Error())
	}
	if callbackErr != "" {
		return nil, errors.BadRequest("", "OAuth authorization failed: %s", callbackErr)
	}
	if code == "" {
		return nil, errors.BadRequest("", "callback_url missing authorization code")
	}
	if callbackState != "" && callbackState != state {
		return nil, errors.BadRequest("", "OAuth state mismatch")
	}

	// Consume session only after request validation succeeds.
	if _, err := p.getAndDeleteOAuthSession(ctx, state); err != nil {
		return nil, err
	}

	return p.exchangeCodexCode(ctx, code, session.CodeVerifier)
}

func (p *Provider) exchangeCodexCode(ctx context.Context, code, codeVerifier string) (*OAuthTokenResult, error) {
	clientID := getEnvWithDefault("OAUTH_CODEX_CLIENT_ID", codexOAuthClientID)
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {clientID},
		"code":          {code},
		"redirect_uri":  {codexOAuthRedirectURI},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, codexOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return nil, errors.InternalServerError("", "token exchange returned empty access_token")
	}

	accountID, email, until := parseCodexTokenClaims(tokenResp.IDToken)
	// Some deployments only put subscription claims on access_token.
	if until == "" {
		if _, _, u2 := parseCodexTokenClaims(tokenResp.AccessToken); u2 != "" {
			until = u2
		}
	}
	if accountID == "" || email == "" {
		a2, e2, _ := parseCodexTokenClaims(tokenResp.AccessToken)
		if accountID == "" {
			accountID = a2
		}
		if email == "" {
			email = e2
		}
	}
	return &OAuthTokenResult{
		AccessToken:             tokenResp.AccessToken,
		RefreshToken:            tokenResp.RefreshToken,
		ExpiresIn:               tokenResp.ExpiresIn,
		TokenEndpoint:           codexOAuthTokenURL,
		BaseURL:                 codexDefaultBaseURL,
		AccountID:               accountID,
		Email:                   email,
		SubscriptionActiveUntil: until,
		Provider:                oauthProviderCodex,
	}, nil
}

func parseOAuthCallbackURL(raw string) (code, state, errParam string, err error) {
	// Users may paste either a full URL or a relative callback path with query.
	parsed, parseErr := url.Parse(strings.TrimSpace(raw))
	if parseErr != nil {
		return "", "", "", fmt.Errorf("invalid callback_url")
	}
	if parsed.RawQuery == "" && strings.Contains(raw, "code=") {
		// Handle bare query strings without leading '?'
		values, qErr := url.ParseQuery(strings.TrimPrefix(raw, "?"))
		if qErr != nil {
			return "", "", "", fmt.Errorf("invalid callback_url")
		}
		return values.Get("code"), values.Get("state"), values.Get("error"), nil
	}
	q := parsed.Query()
	return q.Get("code"), q.Get("state"), q.Get("error"), nil
}

func parseCodexIDTokenClaims(idToken string) (accountID, email string) {
	accountID, email, _ = parseCodexTokenClaims(idToken)
	return accountID, email
}

// parseCodexTokenClaims extracts account/email/subscription expiry from a Codex JWT
// (id_token or access_token). subscriptionActiveUntil is normalized to RFC3339 when possible.
func parseCodexTokenClaims(token string) (accountID, email, subscriptionActiveUntil string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", "", ""
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", "", ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Some tokens may include standard base64 padding.
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return "", "", ""
		}
	}
	var claims struct {
		Email string `json:"email"`
		Auth  struct {
			ChatgptAccountID               string          `json:"chatgpt_account_id"`
			ChatgptSubscriptionActiveUntil json.RawMessage `json:"chatgpt_subscription_active_until"`
		} `json:"https://api.openai.com/auth"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", "", ""
	}
	return strings.TrimSpace(claims.Auth.ChatgptAccountID),
		strings.TrimSpace(claims.Email),
		normalizeCodexTimeValue(claims.Auth.ChatgptSubscriptionActiveUntil)
}

func normalizeCodexTimeValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	// string RFC3339 / date
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			return ""
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t.Local().Format("2006/1/2 15:04:05")
		}
		// unix seconds as string
		if isAllDigitsLocal(s) {
			var sec int64
			if _, err := fmt.Sscanf(s, "%d", &sec); err == nil {
				if sec > 1_000_000_000_000 {
					return time.UnixMilli(sec).Local().Format("2006/1/2 15:04:05")
				}
				if sec > 1_000_000_000 {
					return time.Unix(sec, 0).Local().Format("2006/1/2 15:04:05")
				}
			}
		}
		return s
	}
	// numeric unix seconds/millis
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		n := int64(f)
		if n > 1_000_000_000_000 {
			return time.UnixMilli(n).Local().Format("2006/1/2 15:04:05")
		}
		if n > 1_000_000_000 {
			return time.Unix(n, 0).Local().Format("2006/1/2 15:04:05")
		}
	}
	return ""
}

func isAllDigitsLocal(s string) bool {
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

type pkceCodes struct {
	CodeVerifier  string
	CodeChallenge string
}

func generatePKCECodes() (*pkceCodes, error) {
	// 96 random bytes -> 128 URL-safe base64 chars (within RFC 7636 43-128 range).
	buf := make([]byte, 96)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return &pkceCodes{CodeVerifier: verifier, CodeChallenge: challenge}, nil
}

func (p *Provider) saveOAuthSession(ctx context.Context, state string, session providerOAuthSession) error {
	buf, err := json.Marshal(session)
	if err != nil {
		return errors.WithStack(err)
	}
	if err := p.Cache.Set(ctx, providerOAuthSessionNS, state, string(buf), providerOAuthTTL); err != nil {
		return err
	}
	return nil
}

func (p *Provider) getOAuthSession(ctx context.Context, state string) (*providerOAuthSession, error) {
	val, ok, err := p.Cache.Get(ctx, providerOAuthSessionNS, state)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	var session providerOAuthSession
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return nil, errors.BadRequest("", "Invalid OAuth state")
	}
	return &session, nil
}

func (p *Provider) getAndDeleteOAuthSession(ctx context.Context, state string) (*providerOAuthSession, error) {
	val, ok, err := p.Cache.GetAndDelete(ctx, providerOAuthSessionNS, state)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	var session providerOAuthSession
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return nil, errors.BadRequest("", "Invalid OAuth state")
	}
	return &session, nil
}

func (p *Provider) saveOAuthResult(ctx context.Context, state string, result *OAuthTokenResult) error {
	buf, err := json.Marshal(result)
	if err != nil {
		return errors.WithStack(err)
	}
	return p.Cache.Set(ctx, providerOAuthResultNS, state, string(buf), providerOAuthTTL)
}

func (p *Provider) getAndDeleteOAuthResult(ctx context.Context, state string) (*OAuthTokenResult, error) {
	val, ok, err := p.Cache.GetAndDelete(ctx, providerOAuthResultNS, state)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	var result OAuthTokenResult
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, errors.InternalServerError("", "Invalid cached oauth token result")
	}
	return &result, nil
}

// discoverXAIEndpoints resolves xAI's OAuth endpoints via OIDC discovery.
func discoverXAIEndpoints(ctx context.Context) (*xaiDiscovery, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, xaiDiscoveryURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery returned status %d: %s", resp.StatusCode, string(body))
	}

	var disc xaiDiscovery
	if err := json.Unmarshal(body, &disc); err != nil {
		return nil, err
	}
	if disc.DeviceAuthorizationEndpoint == "" || disc.TokenEndpoint == "" {
		return nil, errors.InternalServerError("", "discovery response missing required endpoints")
	}
	return &disc, nil
}

func generateStateKey() string {
	token := make([]byte, 32)
	_, _ = rand.Read(token)
	return base64.RawURLEncoding.EncodeToString(token)
}

// MergeOAuthAccountHeader injects Chatgpt-Account-Id for oauth_token endpoints when
// the provider OAuth credential carries an account_id. Existing header values win.
func MergeOAuthAccountHeader(headers map[string]string, provider *schema.Provider, authType string) map[string]string {
	if provider == nil {
		return headers
	}
	if authType == "" {
		authType = provider.AuthType
	}
	if authType != "oauth_token" {
		return headers
	}
	cred := provider.GetOAuth()
	if cred == nil || strings.TrimSpace(cred.AccountID) == "" {
		return headers
	}
	if headers == nil {
		headers = make(map[string]string)
	}
	if _, exists := headers["Chatgpt-Account-Id"]; exists {
		return headers
	}
	// Case-insensitive check for an already-configured account header.
	for k := range headers {
		if strings.EqualFold(k, "Chatgpt-Account-Id") {
			return headers
		}
	}
	headers["Chatgpt-Account-Id"] = strings.TrimSpace(cred.AccountID)
	return headers
}
