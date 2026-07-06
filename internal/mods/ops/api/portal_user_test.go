package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
)

func TestPortalUserAPIListWorkspaceAPIKeys(t *testing.T) {
	var gotPath string
	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"api_keys":[{"id":"key-1","name":"Default","key_prefix":"tl_live","secret_last4":"abcd","status":"active"}]}`))
	}))
	defer portal.Close()
	restorePortalConfig(t, portal.URL, "internal-token")

	router := newPortalUserTestRouter()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/workspaces/workspace-1/api-keys", nil)

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "/internal/v1/workspaces/workspace-1/api-keys", gotPath)

	var body struct {
		Success bool                        `json:"success"`
		Data    []biz.PortalWorkspaceAPIKey `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.Len(t, body.Data, 1)
	require.Equal(t, "key-1", body.Data[0].ID)
}

func TestPortalUserAPISyncWorkspaceRuntime(t *testing.T) {
	var gotMethod string
	var gotPath string
	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.RequestURI()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer portal.Close()
	restorePortalConfig(t, portal.URL, "internal-token")

	router := newPortalUserTestRouter()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/workspaces/workspace-1/runtime-sync", nil)

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "/internal/v1/workspaces/workspace-1/runtime-sync", gotPath)
}

func TestPortalUserAPIGetWorkspaceRuntimeAccess(t *testing.T) {
	var gotMethod string
	var gotPath string
	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"runtime_access":{"workspace_id":"workspace-1","scope_type":"tenant","scope_code":"tenant-a","status":"active"}}`))
	}))
	defer portal.Close()
	restorePortalConfig(t, portal.URL, "internal-token")

	router := newPortalUserTestRouter()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/workspaces/workspace-1/runtime-access", nil)

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, http.MethodGet, gotMethod)
	require.Equal(t, "/internal/v1/workspaces/workspace-1/runtime-access", gotPath)
}

func TestPortalUserAPIActivateWorkspaceRuntimeAccess(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotBody string
	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.RequestURI()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		gotBody = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer portal.Close()
	restorePortalConfig(t, portal.URL, "internal-token")

	router := newPortalUserTestRouter()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/workspaces/workspace-1/runtime-access", strings.NewReader(`{"scope_type":"tenant","scope_code":"tenant-a"}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, http.MethodPut, gotMethod)
	require.Equal(t, "/internal/v1/workspaces/workspace-1/runtime-access", gotPath)
	require.JSONEq(t, `{"scope_type":"tenant","scope_code":"tenant-a"}`, gotBody)
}

func TestPortalUserAPIDisableWorkspaceRuntimeAccess(t *testing.T) {
	var gotMethod string
	var gotPath string
	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.RequestURI()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer portal.Close()
	restorePortalConfig(t, portal.URL, "internal-token")

	router := newPortalUserTestRouter()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/workspaces/workspace-1/runtime-access/disable", nil)

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "/internal/v1/workspaces/workspace-1/runtime-access/disable", gotPath)
}

func newPortalUserTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := &PortalUserAPI{PortalUserBIZ: &biz.PortalUser{}}
	router.GET("/workspaces/:workspace_id/api-keys", api.ListWorkspaceAPIKeys)
	router.POST("/workspaces/:workspace_id/runtime-sync", api.SyncWorkspaceRuntime)
	router.GET("/workspaces/:workspace_id/runtime-access", api.GetWorkspaceRuntimeAccess)
	router.PUT("/workspaces/:workspace_id/runtime-access", api.ActivateWorkspaceRuntimeAccess)
	router.POST("/workspaces/:workspace_id/runtime-access/disable", api.DisableWorkspaceRuntimeAccess)
	return router
}

func restorePortalConfig(t *testing.T, baseURL string, token string) {
	t.Helper()

	old := config.C.Portal
	t.Cleanup(func() { config.C.Portal = old })
	config.C.Portal.BaseURL = baseURL
	config.C.Portal.InternalAPIToken = token
}
