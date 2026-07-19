package biz

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Provider business logic layer
type Provider struct {
	Trans             *util.Trans
	ProviderDAL       *dal.Provider
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

	if err := formItem.FillTo(provider); err != nil {
		return err
	}
	provider.Modifier = util.FromUsername(ctx)
	provider.UpdatedAt = time.Now()

	err = p.Trans.Exec(ctx, func(ctx context.Context) error {
		return p.ProviderDAL.Update(ctx, provider)
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
	if baseURL == "" {
		return nil, errors.BadRequest("", "Base URL is required, please provide it or set provider URL")
	}
	var upstreamModels []schema.UpstreamModel

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

var oauthResultCache = cache.New(5*time.Minute, 1*time.Minute)

const (
	// xaiOAuthClientID is xAI's public Grok CLI OAuth client. It only permits the
	// device-authorization grant (RFC 8628); a browser authorization-code redirect
	// to localhost is rejected with HTTP 403.
	xaiOAuthClientID   = "b1a00492-073a-47ea-816f-4c329264a828"
	xaiOAuthScope      = "openid profile email offline_access grok-cli:access api:access"
	xaiDeviceCodeGrant = "urn:ietf:params:oauth:grant-type:device_code"
	xaiDiscoveryURL    = "https://auth.x.ai/.well-known/openid-configuration"
	// xaiDefaultBaseURL is the official xAI API base URL used to call the model.
	xaiDefaultBaseURL = "https://api.x.ai/v1"
)

type OAuthTokenResult struct {
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	ExpiresIn     int    `json:"expires_in"`
	TokenEndpoint string `json:"token_endpoint"`
	BaseURL       string `json:"base_url"`
}

// OAuthStartResult carries the device-code details returned to the frontend so it
// can open the verification page and poll for completion.
type OAuthStartResult struct {
	URL      string `json:"url"`
	UserCode string `json:"user_code"`
	State    string `json:"state"`
}

type xaiDiscovery struct {
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	TokenEndpoint               string `json:"token_endpoint"`
}

// StartOAuthFlow launches xAI's OAuth device-code flow: it requests a device code,
// returns the verification URL for the user to open, and starts a background
// goroutine that polls the token endpoint until the user authorizes.
func (p *Provider) StartOAuthFlow(ctx context.Context, providerName string) (*OAuthStartResult, error) {
	if providerName != "xai" {
		return nil, errors.BadRequest("", "Unsupported oauth provider: %s", providerName)
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

	// Poll the token endpoint in the background; the result lands in oauthResultCache
	// keyed by state, where PollOAuthStatus picks it up.
	go p.pollDeviceToken(clientID, disc.TokenEndpoint, dc.DeviceCode, state, dc.Interval, dc.ExpiresIn)

	return &OAuthStartResult{
		URL:      openURL,
		UserCode: dc.UserCode,
		State:    state,
	}, nil
}

// pollDeviceToken exchanges the device code for tokens once the user authorizes,
// then stores the result keyed by state. It runs detached from the request context.
func (p *Provider) pollDeviceToken(clientID, tokenEndpoint, deviceCode, state string, interval, expiresIn int) {
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
			oauthResultCache.Set(state, &OAuthTokenResult{
				AccessToken:   payload.AccessToken,
				RefreshToken:  payload.RefreshToken,
				ExpiresIn:     payload.ExpiresIn,
				TokenEndpoint: tokenEndpoint,
				BaseURL:       xaiDefaultBaseURL,
			}, cache.DefaultExpiration)
			return
		}
	}
}

func (p *Provider) PollOAuthStatus(ctx context.Context, state string) (*OAuthTokenResult, error) {
	val, found := oauthResultCache.Get(state)
	if !found {
		return nil, nil // Return nil, nil to indicate still waiting
	}
	result, ok := val.(*OAuthTokenResult)
	if !ok {
		return nil, errors.InternalServerError("", "Invalid cached oauth token result")
	}
	oauthResultCache.Delete(state)
	return result, nil
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
