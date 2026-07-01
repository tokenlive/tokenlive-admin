package biz

import (
	"context"
	"encoding/json"
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
	require.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), keys[0].ExpiresAt)

	encoded, err := json.Marshal(keys[0])
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "key_hash")
	require.NotContains(t, string(encoded), "plaintext")
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

func restorePortalConfig(t *testing.T, baseURL string, token string) {
	t.Helper()

	old := config.C.Portal
	t.Cleanup(func() { config.C.Portal = old })
	config.C.Portal.BaseURL = baseURL
	config.C.Portal.InternalAPIToken = token
}
