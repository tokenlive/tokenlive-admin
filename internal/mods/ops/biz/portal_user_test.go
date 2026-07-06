package biz

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
)

func TestPortalUserListWorkspaceAPIKeysUsesInternalAPI(t *testing.T) {
	var gotAuth string
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"api_keys": [{
				"id": "key-1",
				"name": "Default",
				"key_prefix": "tl_live",
				"secret_last4": "abcd",
				"status": "active",
				"expires_at": "2026-07-01T00:00:00Z",
				"last_used_at": "2026-07-02T00:00:00Z",
				"created_at": "2026-06-30T00:00:00Z",
				"updated_at": "2026-06-30T01:00:00Z"
			}]
		}`))
	}))
	defer server.Close()
	restorePortalConfig(t, server.URL, "internal-token")

	keys, err := (&PortalUser{}).ListWorkspaceAPIKeys(context.Background(), "workspace-1")

	require.NoError(t, err)
	require.Equal(t, "Bearer internal-token", gotAuth)
	require.Equal(t, "/internal/v1/workspaces/workspace-1/api-keys", gotPath)
	require.Len(t, keys, 1)
	require.Equal(t, "key-1", keys[0].ID)
	require.Equal(t, "tl_live", keys[0].KeyPrefix)
	require.Equal(t, "abcd", keys[0].SecretLast4)
	require.Equal(t, "active", keys[0].Status)
	require.NotNil(t, keys[0].ExpiresAt)
	require.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), *keys[0].ExpiresAt)

	encoded, err := json.Marshal(keys[0])
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "key_hash")
	require.NotContains(t, string(encoded), "plaintext")
}

func TestPortalUserListWorkspaceAPIKeysKeepsNullableTimes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"api_keys": [{
				"id": "key-1",
				"status": "enabled",
				"expires_at": null,
				"last_used_at": null,
				"created_at": "2026-06-30T00:00:00Z",
				"updated_at": "2026-06-30T01:00:00Z"
			}]
		}`))
	}))
	defer server.Close()
	restorePortalConfig(t, server.URL, "internal-token")

	keys, err := (&PortalUser{}).ListWorkspaceAPIKeys(context.Background(), "workspace-1")

	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Nil(t, keys[0].ExpiresAt)
	require.Nil(t, keys[0].LastUsedAt)

	encoded, err := json.Marshal(keys[0])
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"expires_at":null`)
	require.Contains(t, string(encoded), `"last_used_at":null`)
}

func TestPortalUserSyncWorkspaceRuntimePostsInternalAPI(t *testing.T) {
	var gotAuth string
	var gotMethod string
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.RequestURI()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	restorePortalConfig(t, server.URL, "internal-token")

	err := (&PortalUser{}).SyncWorkspaceRuntime(context.Background(), "workspace-1")

	require.NoError(t, err)
	require.Equal(t, "Bearer internal-token", gotAuth)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "/internal/v1/workspaces/workspace-1/runtime-sync", gotPath)
}

func TestPortalUserGetWorkspaceRuntimeAccessUsesInternalAPI(t *testing.T) {
	var gotAuth string
	var gotMethod string
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"runtime_access":{"workspace_id":"workspace-1","scope_type":"tenant","scope_code":"tenant-a","status":"active"}}`))
	}))
	defer server.Close()
	restorePortalConfig(t, server.URL, "internal-token")

	access, err := (&PortalUser{}).GetWorkspaceRuntimeAccess(context.Background(), "workspace-1")

	require.NoError(t, err)
	require.NotNil(t, access)
	require.Equal(t, "Bearer internal-token", gotAuth)
	require.Equal(t, http.MethodGet, gotMethod)
	require.Equal(t, "/internal/v1/workspaces/workspace-1/runtime-access", gotPath)
	require.Equal(t, "tenant", access.ScopeType)
	require.Equal(t, "tenant-a", access.ScopeCode)
	require.Equal(t, "active", access.Status)
}

func TestPortalUserActivateWorkspaceRuntimeAccessPutsInternalAPI(t *testing.T) {
	var gotAuth string
	var gotMethod string
	var gotPath string
	var gotBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.RequestURI()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		gotBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	restorePortalConfig(t, server.URL, "internal-token")

	err := (&PortalUser{}).ActivateWorkspaceRuntimeAccess(context.Background(), "workspace-1", "tenant", "tenant-a")

	require.NoError(t, err)
	require.Equal(t, "Bearer internal-token", gotAuth)
	require.Equal(t, http.MethodPut, gotMethod)
	require.Equal(t, "/internal/v1/workspaces/workspace-1/runtime-access", gotPath)
	require.JSONEq(t, `{"scope_type":"tenant","scope_code":"tenant-a"}`, gotBody)
}

func TestPortalUserDisableWorkspaceRuntimeAccessPostsInternalAPI(t *testing.T) {
	var gotMethod string
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.RequestURI()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	restorePortalConfig(t, server.URL, "internal-token")

	err := (&PortalUser{}).DisableWorkspaceRuntimeAccess(context.Background(), "workspace-1")

	require.NoError(t, err)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "/internal/v1/workspaces/workspace-1/runtime-access/disable", gotPath)
}

func restorePortalConfig(t *testing.T, baseURL string, token string) {
	t.Helper()

	old := config.C.Portal
	t.Cleanup(func() { config.C.Portal = old })
	config.C.Portal.BaseURL = baseURL
	config.C.Portal.InternalAPIToken = token
}
