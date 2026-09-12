package systemversion

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac"
	"github.com/tokenlive/tokenlive-admin/internal/updatecheck"
	"github.com/tokenlive/tokenlive-admin/internal/versionstatus"
	apierrors "github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/middleware"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"github.com/tokenlive/tokenlive-admin/pkg/versionregistry"
)

type apiSource struct{ calls atomic.Int32 }

func (s *apiSource) Latest(context.Context) (updatecheck.Candidate, error) {
	s.calls.Add(1)
	return updatecheck.Candidate{Version: "v2.0.0", ReleaseURL: "https://example.invalid/release"}, nil
}

type unavailableStore struct{ versionregistry.Store }

func (unavailableStore) List(context.Context) ([]versionregistry.Node, error) {
	return nil, errors.New("private-node-id redis://private-registry/key")
}

func newAPITest(t *testing.T, enabled bool, store versionregistry.Store) (*SystemVersion, *apiSource, *gin.Engine) {
	t.Helper()
	isolatedConfig(t)
	t.Setenv("GATEWAY_SYNC_TOKEN", "test-deployment-token")
	config.C.Middleware.Casbin.Disable = true
	if store == nil {
		store = versionregistry.NewMemoryStore("default", nil)
	}
	source := new(apiSource)
	checker, err := updatecheck.NewChecker(context.Background(), updatecheck.Options{
		Enabled: enabled, Now: func() time.Time { return time.Unix(1000, 0) },
	}, map[string]updatecheck.Source{"admin": source, "gateway": source})
	if err != nil {
		t.Fatal(err)
	}
	a := &SystemVersion{
		Service: versionstatus.New(testIdentity("professional", "release"), store, checker, "this_admin"),
		RBAC:    &rbac.RBAC{Casbinx: &rbac.Casbinx{}},
	}
	t.Cleanup(a.Service.Close)
	gin.SetMode(gin.TestMode)
	e := gin.New()
	e.Use(middleware.AuthWithConfig(middleware.AuthConfig{
		RootID: "root-id", RootUsername: "root",
		Skipper: func(c *gin.Context) bool {
			return c.Request.Method == "POST" && c.Request.URL.Path == "/api/v1/gateway/version"
		},
		ParseUserID: func(c *gin.Context) (string, error) {
			switch c.GetHeader("Authorization") {
			case "root-token":
				return "root-id", nil
			case "viewer-token":
				ctx := util.NewUserCache(c.Request.Context(), util.UserCache{Username: "root", RoleIDs: []string{"admin"}})
				c.Request = c.Request.WithContext(util.NewUsername(ctx, "root"))
				return "viewer-id", nil
			default:
				return "", apierrors.Unauthorized("", "Authentication required")
			}
		},
	}))
	if err := a.RegisterV1Routers(context.Background(), e.Group("/api/v1")); err != nil {
		t.Fatal(err)
	}
	return a, source, e
}

func apiRequest(e http.Handler, method, path, auth, syncToken, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", auth)
	req.Header.Set("X-Sync-Token", syncToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestAPIAuthenticationAndDisabledCasbin(t *testing.T) {
	_, source, e := newAPITest(t, true, nil)
	for _, tc := range []struct {
		method, path, auth string
		status             int
	}{
		{"GET", "/api/v1/current/version", "", 401},
		{"GET", "/api/v1/system/updates", "", 401},
		{"POST", "/api/v1/system/updates/check", "", 401},
		{"GET", "/api/v1/current/version", "viewer-token", 200},
		{"GET", "/api/v1/current/version", "root-token", 200},
		{"GET", "/api/v1/system/updates", "viewer-token", 403},
		{"POST", "/api/v1/system/updates/check", "viewer-token", 403},
		{"GET", "/api/v1/system/updates", "root-token", 200},
	} {
		rec := apiRequest(e, tc.method, tc.path, tc.auth, "", "")
		if rec.Code != tc.status {
			t.Errorf("%s %s %q: %d %s, want %d", tc.method, tc.path, tc.auth, rec.Code, rec.Body, tc.status)
		}
		if tc.path == "/api/v1/current/version" && rec.Code == 200 {
			var result struct{ Data versionstatus.Summary }
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Data.CanManageUpdates != (tc.auth == "root-token") {
				t.Errorf("capability trusts username/disabled Casbin: %s", rec.Body)
			}
		}
	}
	if source.calls.Load() != 0 {
		t.Fatal("read-only/unauthorized requests contacted sources")
	}
}

func TestAPISummarySanitizesRegistryOutages(t *testing.T) {
	_, _, e := newAPITest(t, true, unavailableStore{})
	for _, auth := range []string{"viewer-token", "root-token"} {
		rec := apiRequest(e, "GET", "/api/v1/current/version", auth, "", "")
		if rec.Code != 200 {
			t.Fatalf("safe partial summary was discarded: %d %s", rec.Code, rec.Body)
		}
		var result struct{ Data versionstatus.Summary }
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Data.Identity.Build.Version != "v1.0.0" || result.Data.Gateway.Status != "unavailable" {
			t.Fatalf("missing partial local identity: %s", rec.Body)
		}
		for _, secret := range []string{"private-node-id", "private-registry", "latest", "release_url", "instance_id", "token"} {
			if strings.Contains(rec.Body.String(), secret) {
				t.Errorf("summary leaks %q: %s", secret, rec.Body)
			}
		}
	}
}

func TestAPICheckDisabledAndCooldown(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "cooldown"}[enabled], func(t *testing.T) {
			_, source, e := newAPITest(t, enabled, nil)
			wantStatus, wantID, wantRetry := 409, "update_check_disabled", 0
			if enabled {
				first := apiRequest(e, "POST", "/api/v1/system/updates/check", "root-token", "", "")
				if first.Code != 200 || source.calls.Load() != 2 {
					t.Fatalf("first manual check: %d %s calls=%d", first.Code, first.Body, source.calls.Load())
				}
				wantStatus, wantID, wantRetry = 429, "update_check_cooldown", 60
			}
			before := source.calls.Load()
			rec := apiRequest(e, "POST", "/api/v1/system/updates/check", "root-token", "", "")
			var result struct {
				Success bool
				Error   *apierrors.Error
				Data    versionstatus.Updates
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if rec.Code != wantStatus || result.Success || result.Error == nil || result.Error.ID != wantID ||
				result.Data.Enabled != enabled || result.Data.RetryAfterSeconds != wantRetry {
				t.Fatalf("not an explicit failed check with Updates: %d %s", rec.Code, rec.Body)
			}
			if enabled && rec.Header().Get("Retry-After") != "60" {
				t.Fatalf("missing Retry-After: %v", rec.Header())
			}
			if !enabled && (len(result.Data.Components) != 2 || result.Data.Components[0].State != "disabled") {
				t.Fatalf("disabled result lost component state: %s", rec.Body)
			}
			if source.calls.Load() != before {
				t.Fatal("disabled/cooldown request contacted source")
			}
		})
	}
}

func TestGatewayReportRejectsBeforeMutation(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := versionregistry.NewRedisStore(client, "default")
	_, source, e := newAPITest(t, false, store)
	node := providerNode("default")
	bytes, err := json.Marshal(node)
	if err != nil {
		t.Fatal(err)
	}
	body := string(bytes)
	for _, tc := range []struct {
		name, token, body string
		status            int
	}{
		{"missing token", "", body, 401},
		{"bad token", "wrong", body, 401},
		{"empty token configuration", "", body, 401},
		{"oversize", "test-deployment-token", body + strings.Repeat(" ", 4096), 413},
		{"wrong namespace", "test-deployment-token", strings.Replace(body, `"default"`, `"another"`, 1), 400},
		{"unsupported schema", "test-deployment-token", strings.Replace(body, `"schema_version":1`, `"schema_version":2`, 1), 400},
		{"bad UUID", "test-deployment-token", strings.Replace(body, node.InstanceID, "not-a-uuid", 1), 400},
		{"invalid build kind", "test-deployment-token", strings.Replace(body, `"release"`, `"nightly"`, 1), 400},
		{"invalid JSON", "test-deployment-token", "{", 400},
		{"trailing JSON", "test-deployment-token", body + "{}", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "empty token configuration" {
				t.Setenv("GATEWAY_SYNC_TOKEN", "")
			}
			before := server.CommandCount()
			rec := apiRequest(e, "POST", "/api/v1/gateway/version", "", tc.token, tc.body)
			if rec.Code != tc.status {
				t.Fatalf("%d %s, want %d", rec.Code, rec.Body, tc.status)
			}
			if server.CommandCount() != before {
				t.Fatal("invalid report reached Redis")
			}
		})
	}
	rec := apiRequest(e, "POST", "/api/v1/gateway/version", "", "test-deployment-token", body)
	if rec.Code != 200 {
		t.Fatalf("valid deployment report: %d %s", rec.Code, rec.Body)
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), node.InstanceID) })
	nodes, err := store.List(context.Background())
	if err != nil || len(nodes) != 1 || nodes[0] != node {
		t.Fatalf("report not stored: %+v %v", nodes, err)
	}
	summary := apiRequest(e, "GET", "/api/v1/current/version", "viewer-token", "", "")
	if strings.Contains(summary.Body.String(), node.InstanceID) || !strings.Contains(summary.Body.String(), `"count":1`) {
		t.Fatalf("aggregate leaked identity or lost report: %s", summary.Body)
	}
	if source.calls.Load() != 0 {
		t.Fatal("disabled update checks prevented internal reporting")
	}
	server.FastForward(3 * time.Minute)
	nodes, err = store.List(context.Background())
	if err != nil || len(nodes) != 0 {
		t.Fatalf("report did not expire: %+v %v", nodes, err)
	}
}
