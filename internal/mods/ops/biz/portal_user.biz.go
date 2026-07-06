package biz

import (
	"bytes"
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
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	KeyPrefix   string     `json:"key_prefix"`
	SecretLast4 string     `json:"secret_last4"`
	Status      string     `json:"status"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

type PortalWorkspaceAPIKeysResponse struct {
	APIKeys []PortalWorkspaceAPIKey `json:"api_keys"`
}

type PortalWorkspaceRuntimeAccess struct {
	WorkspaceID string     `json:"workspace_id"`
	ScopeType   string     `json:"scope_type"`
	ScopeCode   string     `json:"scope_code"`
	Status      string     `json:"status"`
	ActivatedAt *time.Time `json:"activated_at"`
	ActivatedBy string     `json:"activated_by"`
	DisabledAt  *time.Time `json:"disabled_at"`
	DisabledBy  string     `json:"disabled_by"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

type PortalWorkspaceRuntimeAccessResponse struct {
	RuntimeAccess *PortalWorkspaceRuntimeAccess `json:"runtime_access"`
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

func (a *PortalUser) GetWorkspaceRuntimeAccess(ctx context.Context, workspaceID string) (*PortalWorkspaceRuntimeAccess, error) {
	req, err := newPortalInternalRequest(
		ctx,
		http.MethodGet,
		"/internal/v1/workspaces/"+url.PathEscape(workspaceID)+"/runtime-access",
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

	var res PortalWorkspaceRuntimeAccessResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.RuntimeAccess, nil
}

func (a *PortalUser) ActivateWorkspaceRuntimeAccess(ctx context.Context, workspaceID string, scopeType string, scopeCode string) error {
	body, err := json.Marshal(map[string]string{"scope_type": scopeType, "scope_code": scopeCode})
	if err != nil {
		return err
	}
	req, err := newPortalInternalRequestWithBody(
		ctx,
		http.MethodPut,
		"/internal/v1/workspaces/"+url.PathEscape(workspaceID)+"/runtime-access",
		body,
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

func (a *PortalUser) DisableWorkspaceRuntimeAccess(ctx context.Context, workspaceID string) error {
	req, err := newPortalInternalRequest(
		ctx,
		http.MethodPost,
		"/internal/v1/workspaces/"+url.PathEscape(workspaceID)+"/runtime-access/disable",
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
	return newPortalInternalRequestWithBody(ctx, method, path, nil)
}

func newPortalInternalRequestWithBody(ctx context.Context, method string, path string, body []byte) (*http.Request, error) {
	portalCfg := config.C.Portal
	if portalCfg.BaseURL == "" {
		return nil, fmt.Errorf("portal base URL not configured")
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(portalCfg.BaseURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if portalCfg.InternalAPIToken != "" {
		req.Header.Set("Authorization", "Bearer "+portalCfg.InternalAPIToken)
	}
	return req, nil
}
