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
	InstanceID string
}

func NewTokenRefresher(db *gorm.DB) *TokenRefresher {
	// Generate a unique instance ID for current process (e.g. hostname + pid or env)
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	pid := os.Getpid()
	instanceID := fmt.Sprintf("%s-%d", hostname, pid)

	return &TokenRefresher{
		DB:         db,
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
	// Query providers with oauth_token auth_type that are expiring in 30 minutes, or already expired
	var providers []schema.Provider
	err := r.DB.Table(new(schema.Provider).TableName()).
		Where("auth_type = ? AND deleted = '0'", "oauth_token").
		Where("expires_at IS NULL OR expires_at < ?", time.Now().Add(30*time.Minute)).
		Find(&providers).Error

	if err != nil {
		logging.Context(ctx).Error("failed to query expiring providers", zap.Error(err))
		return
	}

	for _, provider := range providers {
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
	newAccessToken, newRefreshToken, expiresAt, err := r.refreshOAuthToken(ctx, provider)
	if err != nil {
		logging.Context(ctx).Error("failed to refresh OAuth token for provider", zap.String("provider_id", provider.ID), zap.Error(err))
		// Release lock immediately on failure
		r.DB.Table(tableName).Where("id = ? AND lock_owner = ?", provider.ID, r.InstanceID).Updates(map[string]interface{}{
			"locked_until": nil,
		})
		return
	}

	// 3. Update Provider credentials, save final token list to ApiKeys, and release lock in a transaction
	err = r.DB.Transaction(func(tx *gorm.DB) error {
		// Prepare ApiKeys json containing the new access token
		apiKeyItems := []schema.ApiKeyItem{
			{
				Value:       newAccessToken,
				Description: "OAuth Token (x.ai)",
			},
		}
		apiKeysJSON, _ := json.Marshal(apiKeyItems)

		// Update provider
		if err := tx.Table(tableName).
			Where("id = ? AND lock_owner = ?", provider.ID, r.InstanceID).
			Updates(map[string]interface{}{
				"api_keys":            json.RawMessage(apiKeysJSON),
				"oauth_refresh_token": newRefreshToken,
				"expires_at":          expiresAt,
				"lock_owner":          nil,
				"locked_until":        nil,
			}).Error; err != nil {
			return err
		}

		// Cascade update endpoints
		endpointTableName := new(schema.Endpoint).TableName()
		if err := tx.Table(endpointTableName).
			Where("provider_id = ? AND deleted = '0'", provider.ID).
			Updates(map[string]interface{}{
				"api_key": newAccessToken,
			}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		logging.Context(ctx).Error("failed to save refreshed tokens and cascade update endpoints", zap.String("provider_id", provider.ID), zap.Error(err))
		return
	}

	logging.Context(ctx).Info("Successfully refreshed and saved OAuth token for provider and cascade updated endpoints", zap.String("provider_id", provider.ID))
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (r *TokenRefresher) refreshOAuthToken(ctx context.Context, provider schema.Provider) (string, string, time.Time, error) {
	if provider.OAuthRefreshToken == "" {
		return "", "", time.Time{}, errors.New("refresh_token is empty")
	}

	var clientID string
	var tokenURL string

	protocol := strings.ToLower(provider.Protocol)
	switch protocol {
	case "xai":
		clientID = getEnvWithDefault("OAUTH_XAI_CLIENT_ID", "b1a00492-073a-47ea-816f-4c329264a828")
		tokenURL = getEnvWithDefault("OAUTH_XAI_TOKEN_URL", "https://api.x.ai/v2/oauth2/token")
	case "anthropic":
		clientID = getEnvWithDefault("ANTHROPIC_CLIENT_ID", "claude-cli")
		tokenURL = getEnvWithDefault("ANTHROPIC_TOKEN_URL", "https://api.anthropic.com/v1/oauth/token")
	case "openai", "openai-compatible":
		clientID = getEnvWithDefault("OPENAI_CLIENT_ID", "openai-cli")
		tokenURL = getEnvWithDefault("OPENAI_TOKEN_URL", "https://api.openai.com/v1/oauth/token")
	default:
		clientID = getEnvWithDefault("OAUTH_CLIENT_ID", "client-id")
		tokenURL = provider.URL // Fallback to provider URL
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", clientID)
	form.Set("refresh_token", provider.OAuthRefreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", time.Time{}, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", time.Time{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", time.Time{}, fmt.Errorf("oauth token refresh returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var tokenResp oauthTokenResponse
	if err = json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", "", time.Time{}, errors.New("access_token is empty in response")
	}

	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600 // Default 1 hour fallback
	}
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	finalRefreshToken := tokenResp.RefreshToken
	if finalRefreshToken == "" {
		finalRefreshToken = provider.OAuthRefreshToken
	}

	return tokenResp.AccessToken, finalRefreshToken, expiresAt, nil
}

func getEnvWithDefault(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}
