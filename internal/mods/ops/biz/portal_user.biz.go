package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/config"
)

type PortalUser struct{}

type PortalUserResult struct {
	ID           string  `json:"id"`
	DisplayName  string  `json:"display_name"`
	PrimaryEmail *string `json:"primary_email"`
}

type PortalSearchResponse struct {
	Users []PortalUserResult `json:"users"`
}

type PortalWorkspaceAPIKey struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	KeyPrefix   string    `json:"key_prefix"`
	SecretLast4 string    `json:"secret_last4"`
	Status      string    `json:"status"`
	ExpiresAt   time.Time `json:"expires_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PortalWorkspaceAPIKeysResponse struct {
	APIKeys []PortalWorkspaceAPIKey `json:"api_keys"`
}

func (a *PortalUser) Search(ctx context.Context, keyword string, limit int) ([]PortalUserResult, error) {
	portalCfg := config.C.Portal
	if portalCfg.BaseURL == "" {
		return nil, fmt.Errorf("portal base URL not configured")
	}

	searchURL := fmt.Sprintf("%s/internal/v1/users/search?keyword=%s&limit=%d",
		portalCfg.BaseURL, url.QueryEscape(keyword), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}

	if portalCfg.InternalAPIToken != "" {
		req.Header.Set("Authorization", "Bearer "+portalCfg.InternalAPIToken)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("portal returned unexpected status: %d", resp.StatusCode)
	}

	var res PortalSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	return res.Users, nil
}

func (a *PortalUser) ListWorkspaceAPIKeys(ctx context.Context, workspaceID string) ([]PortalWorkspaceAPIKey, error) {
	req, err := newPortalInternalRequest(
		ctx,
		http.MethodGet,
		"/internal/v1/workspaces/"+url.PathEscape(workspaceID)+"/api-keys",
	)
	if err != nil {
		return nil, err
	}

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("portal returned unexpected status: %d", resp.StatusCode)
	}

	var res PortalWorkspaceAPIKeysResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.APIKeys, nil
}

func (a *PortalUser) SyncWorkspaceRuntime(ctx context.Context, workspaceID string) error {
	req, err := newPortalInternalRequest(
		ctx,
		http.MethodPost,
		"/internal/v1/workspaces/"+url.PathEscape(workspaceID)+"/runtime-sync",
	)
	if err != nil {
		return err
	}

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("portal returned unexpected status: %d", resp.StatusCode)
	}
	return nil
}

func newPortalInternalRequest(ctx context.Context, method string, path string) (*http.Request, error) {
	portalCfg := config.C.Portal
	if portalCfg.BaseURL == "" {
		return nil, fmt.Errorf("portal base URL not configured")
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(portalCfg.BaseURL, "/")+path, nil)
	if err != nil {
		return nil, err
	}
	if portalCfg.InternalAPIToken != "" {
		req.Header.Set("Authorization", "Bearer "+portalCfg.InternalAPIToken)
	}
	return req, nil
}
