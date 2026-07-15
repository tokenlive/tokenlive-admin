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
		reqURL := baseURL + "/api/saas/models/v2/modelList"
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader("{}"))
		if err != nil {
			return nil, errors.BadRequest("", "Failed to create request: %s", err.Error())
		}
		req.Header.Set("ptKey", apiKey)
		req.Header.Set("loginType", "PIN_JD_CLOUD")
		req.Header.Set("x-ms-client-request-id", uuid.NewString())
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

		var joycodeResp struct {
			Code int `json:"code"`
			Data []struct {
				ChatApiModel string `json:"chatApiModel"`
				Label        string `json:"label"`
			} `json:"data"`
			Msg string `json:"msg"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&joycodeResp); err != nil {
			return nil, errors.BadRequest("", "Failed to parse upstream response: %s", err.Error())
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

type OAuthTokenResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (p *Provider) StartOAuthFlow(ctx context.Context, providerName string) (string, string, error) {
	if providerName != "xai" {
		return "", "", errors.BadRequest("", "Unsupported oauth provider: %s", providerName)
	}

	clientID := getEnvWithDefault("OAUTH_XAI_CLIENT_ID", "b1a00492-073a-47ea-816f-4c329264a828")
	redirectURI := getEnvWithDefault("OAUTH_REDIRECT_URI", "http://localhost:8040/api/v1/providers/oauth/callback")

	// Generate PKCE
	verifier := generateCodeVerifier()
	challenge := generatePKCEChallenge(verifier)

	// Encrypt state containing code_verifier
	state, err := EncryptState(verifier)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt state: %w", err)
	}

	// Build auth URL
	authURL := fmt.Sprintf("https://x.ai/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&state=%s&code_challenge=%s&code_challenge_method=S256",
		clientID, url.QueryEscape(redirectURI), url.QueryEscape(state), challenge)

	return authURL, state, nil
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

func (p *Provider) HandleOAuthCallback(ctx context.Context, code, state string) (string, error) {
	if code == "" || state == "" {
		return "", errors.BadRequest("", "code or state is empty")
	}

	// 1. Decrypt state and retrieve code_verifier
	verifier, err := DecryptState(state, 5*time.Minute)
	if err != nil {
		return "", errors.BadRequest("", "Invalid or expired state: %v", err)
	}

	clientID := getEnvWithDefault("OAUTH_XAI_CLIENT_ID", "b1a00492-073a-47ea-816f-4c329264a828")
	redirectURI := getEnvWithDefault("OAUTH_REDIRECT_URI", "http://localhost:8040/api/v1/providers/oauth/callback")

	// 2. Perform Code Exchange with x.ai
	tokenURL := "https://api.x.ai/v2/oauth2/token"
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read exchange response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("exchange token returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token exchange response: %w", err)
	}

	// 3. Cache token result
	res := &OAuthTokenResult{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
	}
	oauthResultCache.Set(state, res, cache.DefaultExpiration)

	// Return a user-friendly HTML success landing page
	htmlResponse := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Authentication Successful</title>
		<style>
			body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; background-color: #0f172a; color: #f8fafc; }
			.card { text-align: center; padding: 2.5rem; background: #1e293b; border-radius: 12px; box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1); border: 1px solid #334155; }
			h1 { color: #10b981; margin-top: 0; }
			p { color: #94a3b8; font-size: 0.95rem; line-height: 1.5; margin-bottom: 0; }
		</style>
	</head>
	<body>
		<div class="card">
			<h1>绑定成功</h1>
			<p>您已成功绑定 OAuth 凭证。</p>
			<p style="margin-top: 0.5rem;">此窗口现在可以安全关闭。</p>
		</div>
	</body>
	</html>
	`
	return htmlResponse, nil
}

func generateCodeVerifier() string {
	token := make([]byte, 32)
	_, _ = rand.Read(token)
	return base64.RawURLEncoding.EncodeToString(token)[:43]
}

func generatePKCEChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
