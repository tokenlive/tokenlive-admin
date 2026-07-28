package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TokenRefresher struct {
	DB         *gorm.DB
	RedisSync  *ConfigRedisSync
	InstanceID string
}

func NewTokenRefresher(db *gorm.DB, redisSync *ConfigRedisSync) *TokenRefresher {
	// Generate a unique instance ID for current process (e.g. hostname + pid or env)
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	pid := os.Getpid()
	instanceID := fmt.Sprintf("%s-%d", hostname, pid)

	return &TokenRefresher{
		DB:         db,
		RedisSync:  redisSync,
		InstanceID: instanceID,
	}
}

// StartCronLoop starts the background cron loop to refresh expiring OAuth endpoints
func (r *TokenRefresher) StartCronLoop(ctx context.Context) {
	logging.Context(ctx).Info("Starting OAuth Token Refresher background loop", zap.String("instance_id", r.InstanceID))

	// Run initially after startup
	go r.scanAndRefresh(ctx)

	// Run every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logging.Context(ctx).Info("OAuth Token Refresher loop stopped")
			return
		case <-ticker.C:
			r.scanAndRefresh(ctx)
		}
	}
}

func (r *TokenRefresher) scanAndRefresh(ctx context.Context) {
	// Query all oauth_token providers, then filter in memory by the expires_at
	// stored inside the oauth JSON column.
	var providers []schema.Provider
	err := r.DB.Table(new(schema.Provider).TableName()).
		Where("auth_type = ? AND deleted = '0'", "oauth_token").
		Find(&providers).Error

	if err != nil {
		logging.Context(ctx).Error("failed to query oauth providers", zap.Error(err))
		return
	}

	threshold := time.Now().Add(30 * time.Minute)
	for _, provider := range providers {
		cred := provider.GetOAuth()
		// Refresh when there is no known expiry, or it is within the threshold.
		if cred != nil && cred.ExpiresAt != nil && cred.ExpiresAt.After(threshold) {
			continue
		}
		go r.lockAndRefreshProvider(ctx, provider)
	}
}

func (r *TokenRefresher) lockAndRefreshProvider(ctx context.Context, provider schema.Provider) {
	now := time.Now()
	lockUntil := now.Add(2 * time.Minute) // lock for max 2 minutes

	tableName := new(schema.Provider).TableName()
	res := r.DB.Table(tableName).
		Where("id = ? AND (locked_until IS NULL OR locked_until < ?)", provider.ID, now).
		Updates(map[string]interface{}{
			"lock_owner":   r.InstanceID,
			"locked_until": lockUntil,
		})

	if res.Error != nil {
		logging.Context(ctx).Error("failed to lock provider row", zap.String("provider_id", provider.ID), zap.Error(res.Error))
		return
	}

	if res.RowsAffected == 0 {
		// Locked by another node, skip
		return
	}

	logging.Context(ctx).Info("Locked provider row for OAuth token refresh", zap.String("provider_id", provider.ID), zap.String("instance_id", r.InstanceID))

	// 2. Perform refresh
	newAccessToken, newRefreshToken, expiresAt, meta, err := r.refreshOAuthToken(ctx, provider)
	if err != nil {
		logging.Context(ctx).Error("failed to refresh OAuth token for provider", zap.String("provider_id", provider.ID), zap.Error(err))
		// Release lock immediately on failure
		r.DB.Table(tableName).Where("id = ? AND lock_owner = ?", provider.ID, r.InstanceID).Updates(map[string]interface{}{
			"lock_owner":   gorm.Expr("NULL"),
			"locked_until": gorm.Expr("NULL"),
		})
		return
	}

	// 3. Update Provider credentials, save final token list to ApiKeys, and release lock in a transaction
	err = r.DB.Transaction(func(tx *gorm.DB) error {
		cred := provider.GetOAuth()
		tokenEndpoint := ""
		accountID := ""
		email := ""
		if cred != nil {
			tokenEndpoint = cred.TokenEndpoint
			accountID = cred.AccountID
			email = cred.Email
		}
		if meta.AccountID != "" {
			accountID = meta.AccountID
		}
		if meta.Email != "" {
			email = meta.Email
		}

		// Prepare ApiKeys json containing the new access token
		apiKeyItems := []schema.ApiKeyItem{
			{
				Value:       newAccessToken,
				Description: oauthTokenDescription(tokenEndpoint, accountID),
			},
		}
		apiKeysJSON, _ := json.Marshal(apiKeyItems)

		// Preserve token_endpoint and non-secret identity fields; refresh token/expiry.
		expiresAtCopy := expiresAt
		oauthJSON, _ := json.Marshal(schema.OAuthCredential{
			RefreshToken:  newRefreshToken,
			TokenEndpoint: tokenEndpoint,
			ExpiresAt:     &expiresAtCopy,
			AccountID:     accountID,
			Email:         email,
		})

		// Update provider. access_token is stored provider-level only; endpoints
		// inherit it at sync time (oauth_token endpoints always read the provider key).
		if err := tx.Table(tableName).
			Where("id = ? AND lock_owner = ?", provider.ID, r.InstanceID).
			Updates(map[string]interface{}{
				"api_keys":     string(apiKeysJSON),
				"o_auth":       string(oauthJSON),
				"lock_owner":   gorm.Expr("NULL"),
				"locked_until": gorm.Expr("NULL"),
			}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		logging.Context(ctx).Error("failed to save refreshed OAuth token for provider", zap.String("provider_id", provider.ID), zap.Error(err))
		return
	}

	logging.Context(ctx).Info("Successfully refreshed and saved OAuth token for provider", zap.String("provider_id", provider.ID))

	// Push the new token to Redis so the gateway picks it up. The gateway only
	// reacts to per-model version bumps, so every model referencing this provider
	// must be re-synced (SyncModelByCode also increments the version).
	if r.RedisSync != nil {
		modelCodes, mErr := r.RedisSync.GetModelCodesByProvider(ctx, provider.ID)
		if mErr != nil {
			logging.Context(ctx).Error("failed to list models for provider after refresh", zap.String("provider_id", provider.ID), zap.Error(mErr))
			return
		}
		for _, modelCode := range modelCodes {
			if err := r.RedisSync.SyncModelByCode(ctx, modelCode); err != nil {
				logging.Context(ctx).Error("failed to sync model to Redis after token refresh", zap.String("provider_id", provider.ID), zap.String("model_code", modelCode), zap.Error(err))
			}
		}
	}
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type oauthRefreshMeta struct {
	AccountID string
	Email     string
}

func (r *TokenRefresher) refreshOAuthToken(ctx context.Context, provider schema.Provider) (string, string, time.Time, oauthRefreshMeta, error) {
	var meta oauthRefreshMeta
	cred := provider.GetOAuth()
	if cred == nil || cred.RefreshToken == "" {
		return "", "", time.Time{}, meta, errors.New("refresh_token is empty")
	}

	if cred.TokenEndpoint == "" {
		return "", "", time.Time{}, meta, errors.New("token_endpoint is empty")
	}

	tokenURL := cred.TokenEndpoint
	var clientID string
	isCodex := false

	// 根据 TokenEndpoint 的域名特征选择对应的 Client ID
	tokenURLLower := strings.ToLower(tokenURL)
	if strings.Contains(tokenURLLower, "x.ai") {
		clientID = getEnvWithDefault("OAUTH_XAI_CLIENT_ID", xaiOAuthClientID)
	} else if strings.Contains(tokenURLLower, "anthropic") {
		clientID = getEnvWithDefault("ANTHROPIC_CLIENT_ID", "claude-cli")
	} else if strings.Contains(tokenURLLower, "auth.openai.com") || strings.Contains(tokenURLLower, "openai.com") {
		// Codex / ChatGPT CLI public client
		clientID = getEnvWithDefault("OAUTH_CODEX_CLIENT_ID", codexOAuthClientID)
		isCodex = true
	} else {
		clientID = getEnvWithDefault("OAUTH_CLIENT_ID", "client-id")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", clientID)
	form.Set("refresh_token", cred.RefreshToken)
	if isCodex {
		// Match CLIProxyAPI Codex refresh scope.
		form.Set("scope", "openid profile email")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", time.Time{}, meta, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", time.Time{}, meta, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", time.Time{}, meta, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", time.Time{}, meta, fmt.Errorf("oauth token refresh returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp oauthTokenResponse
	if err = json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", "", time.Time{}, meta, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", "", time.Time{}, meta, errors.New("access_token is empty in response")
	}

	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600 // Default 1 hour fallback
	}
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	finalRefreshToken := tokenResp.RefreshToken
	if finalRefreshToken == "" {
		finalRefreshToken = cred.RefreshToken
	}

	if isCodex {
		meta.AccountID, meta.Email = parseCodexIDTokenClaims(tokenResp.IDToken)
	}

	return tokenResp.AccessToken, finalRefreshToken, expiresAt, meta, nil
}

func oauthTokenDescription(tokenEndpoint, _ string) string {
	endpoint := strings.ToLower(tokenEndpoint)
	switch {
	case strings.Contains(endpoint, "x.ai"):
		return "OAuth Token (x.ai)"
	case strings.Contains(endpoint, "auth.openai.com"), strings.Contains(endpoint, "openai.com"):
		return "OAuth Token (codex)"
	default:
		return "OAuth Token"
	}
}

func getEnvWithDefault(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}
