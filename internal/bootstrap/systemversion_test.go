package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creasty/defaults"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/api"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/systemversion"
	"github.com/tokenlive/tokenlive-admin/internal/updatecheck"
	"github.com/tokenlive/tokenlive-admin/internal/versionstatus"
	"github.com/tokenlive/tokenlive-admin/internal/wirex"
	"github.com/tokenlive/tokenlive-admin/pkg/cachex"
	"github.com/tokenlive/tokenlive-admin/pkg/jwtx"
	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"github.com/tokenlive/tokenlive-admin/pkg/versionregistry"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Signal the existing policy loader's first tick before cleanup stops it. This
// makes its asynchronous ticker initialization deterministic without sleeping.
type versionPolicyCache struct {
	cachex.Cacher
	ticked chan struct{}
	once   sync.Once
}

type versionCountingReader struct {
	io.Reader
	bytes int
}

func (r *versionCountingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.bytes += n
	return n, err
}

func (c *versionPolicyCache) Get(ctx context.Context, ns, key string) (string, bool, error) {
	if ns == config.CacheNSForRole && key == config.CacheKeyForSyncToCasbin {
		c.once.Do(func() { close(c.ticked) })
	}
	return c.Cacher.Get(ctx, ns, key)
}

func TestVersionRoutesThroughApplicationMiddleware(t *testing.T) {
	previous := config.C
	config.C = new(config.Config)
	if err := defaults.Set(config.C); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { config.C = previous })
	config.C.General.Root.ID = "trusted-root-id"
	config.C.General.Root.Username = "root"
	config.C.General.WorkDir = "../.."
	config.C.Middleware.Casbin.ModelFile = "configs/rbac_model.conf"
	config.C.Middleware.Casbin.LoadThread = 1
	config.C.Middleware.Casbin.AutoLoadInterval = 1
	config.C.Middleware.Auth.SkippedPathPrefixes = nil
	config.C.Middleware.Casbin.SkippedPathPrefixes = nil
	config.C.Middleware.Logger.SkippedPathPrefixes = []string{"/api/"}
	config.C.Middleware.RateLimiter.Enable = false
	config.C.Util.Prometheus.Enable = false
	config.C.Middleware.CopyBody.MaxContentLen = 32 << 20
	t.Setenv("GATEWAY_SYNC_TOKEN", "test-deployment-token")
	t.Setenv("GATEWAY_VERSION_NAMESPACE", "default")
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "permissions.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&schema.Role{}, &schema.Menu{}, &schema.MenuResource{}, &schema.RoleMenu{}); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"delegated-role-id", "read-only-role-id"} {
		if err := db.Create(&schema.Role{ID: role, Status: "enabled"}).Error; err != nil {
			t.Fatal(err)
		}
		menu := &schema.Menu{ID: role, Code: role, Type: "button", Status: "enabled"}
		if err := db.Create(menu).Error; err != nil {
			t.Fatal(err)
		}
		path, method := "/api/v1/system/updates/check", "POST"
		if role == "read-only-role-id" {
			path, method = "/api/v1/system/updates", "GET"
		}
		if err := db.Create(&schema.MenuResource{ID: role, MenuID: role, Path: path, Method: method}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&schema.RoleMenu{ID: role, RoleID: role, MenuID: role}).Error; err != nil {
			t.Fatal(err)
		}
	}
	cache := &versionPolicyCache{Cacher: cachex.NewMemoryCache(cachex.MemoryConfig{}), ticked: make(chan struct{})}
	t.Cleanup(func() { _ = cache.Close(context.Background()) })
	casbinx := &rbac.Casbinx{
		DB: db, Cache: cache, MenuDAL: &dal.Menu{DB: db},
		MenuResourceDAL: &dal.MenuResource{DB: db}, RoleDAL: &dal.Role{DB: db},
	}
	if err := casbinx.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		select {
		case <-cache.ticked:
			_ = casbinx.Release(context.Background())
		case <-time.After(3 * time.Second):
			t.Error("policy loader did not initialize")
		}
	})
	auth := jwtx.New(jwtx.NewStoreWithCache(cache), jwtx.SetSigningKey("isolated-version-test-signing-key", ""))
	login := &biz.Login{Auth: auth, Cache: cache}
	rbacModule := &rbac.RBAC{LoginAPI: &api.Login{LoginBIZ: login}, Casbinx: casbinx}
	checker, err := updatecheck.NewChecker(context.Background(), updatecheck.Options{Enabled: false}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(checker.Close)
	svc := versionstatus.New(productversion.Identity{Edition: "professional", Build: productversion.Build{Version: "v1.0.0", Kind: "release"}},
		versionregistry.NewMemoryStore("default", nil), checker, "this_admin")
	module := &systemversion.SystemVersion{Service: svc, RBAC: rbacModule}
	injector := &wirex.Injector{M: &mods.Mods{RBAC: rbacModule, SystemVersion: module}}
	gin.SetMode(gin.TestMode)
	e := gin.New()
	if err := useHTTPMiddlewares(context.Background(), e, injector, []string{"/api/"}); err != nil {
		t.Fatal(err)
	}
	if err := module.RegisterV1Routers(context.Background(), e.Group("/api/v1")); err != nil {
		t.Fatal(err)
	}
	// Probes ensure the new exceptions do not bypass adjacent/future endpoints
	// or grant another HTTP method access through a prefix match.
	for _, path := range []string{"/gateway/version-extra", "/gateway/version/nodes", "/current/version-extra"} {
		e.POST("/api/v1"+path, func(c *gin.Context) { util.ResOK(c) })
		e.GET("/api/v1"+path, func(c *gin.Context) { util.ResOK(c) })
	}
	e.GET("/api/v1/gateway/version", func(c *gin.Context) { util.ResOK(c) })
	e.POST("/api/v1/current/version", func(c *gin.Context) { util.ResOK(c) })
	tokens := map[string]string{"anonymous": ""}
	for _, user := range []string{"trusted-root-id", "delegated", "viewer", "spoofed-root-name", "read-only"} {
		info, err := auth.GenerateToken(context.Background(), user)
		if err != nil {
			t.Fatal(err)
		}
		tokens[user] = info.GetAccessToken()
		roles := []string{"viewer-role-id"}
		username := user
		if user == "delegated" {
			roles = []string{"delegated-role-id"}
		} else if user == "read-only" {
			roles = []string{"read-only-role-id"}
		} else if user == "spoofed-root-name" {
			username, roles = "root", []string{"admin"}
		}
		if err := cache.Set(context.Background(), config.CacheNSForUser, user, (util.UserCache{Username: username, RoleIDs: roles}).String()); err != nil {
			t.Fatal(err)
		}
	}
	request := func(method, path, user, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+tokens[user])
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Sync-Token", "test-deployment-token")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}
	for _, user := range []string{"anonymous", "trusted-root-id", "delegated", "viewer", "spoofed-root-name", "read-only"} {
		allowed := user == "trusted-root-id" || user == "delegated"
		for _, endpoint := range []struct{ method, path string }{
			{"GET", "/api/v1/current/version"},
			{"GET", "/api/v1/system/updates"},
			{"POST", "/api/v1/system/updates/check"},
		} {
			want := 403
			if user == "anonymous" {
				want = 401
			} else if endpoint.path == "/api/v1/current/version" || allowed {
				want = 200
				if endpoint.method == "POST" {
					want = 409
				}
			}
			rec := request(endpoint.method, endpoint.path, user, "")
			if rec.Code != want {
				t.Errorf("%s %s %s = %d %s, want %d", user, endpoint.method, endpoint.path, rec.Code, rec.Body, want)
			}
			if endpoint.path == "/api/v1/current/version" && want == 200 {
				var result struct{ Data versionstatus.Summary }
				if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Data.CanManageUpdates != allowed {
					t.Errorf("wrong capability for %s: %s", user, rec.Body)
				}
			}
		}
	}
	for _, endpoint := range []struct{ method, path string }{
		{"GET", "/api/v1/gateway/version"}, {"POST", "/api/v1/gateway/version-extra"},
		{"POST", "/api/v1/gateway/version/nodes"}, {"GET", "/api/v1/current/version-extra"},
	} {
		if rec := request(endpoint.method, endpoint.path, "anonymous", ""); rec.Code != 401 {
			t.Errorf("auth exception was not exact: %s %s = %d", endpoint.method, endpoint.path, rec.Code)
		}
	}
	for _, endpoint := range []struct{ method, path string }{
		{"GET", "/api/v1/current/version-extra"}, {"POST", "/api/v1/current/version"},
	} {
		if rec := request(endpoint.method, endpoint.path, "viewer", ""); rec.Code != 403 {
			t.Errorf("Casbin exception was not exact: %s %s = %d", endpoint.method, endpoint.path, rec.Code)
		}
	}
	body := `{"schema_version":1,"namespace":"default","instance_id":"c356f2b8-9021-43c4-b4be-64c43643a17e","version":"v1.0.0","build_kind":"release"}`
	if rec := request("POST", "/api/v1/gateway/version", "anonymous", body); rec.Code != 200 {
		t.Errorf("deployment report incorrectly requires login: %d %s", rec.Code, rec.Body)
	}
	if rec := request("POST", "/api/v1/gateway/version", "anonymous", body+strings.Repeat(" ", 4096)); rec.Code != 413 {
		t.Errorf("report limit not enforced through middleware: %d %s", rec.Code, rec.Body)
	}
	for _, token := range []string{"invalid-token", "test-deployment-token"} {
		body := &versionCountingReader{Reader: strings.NewReader(strings.Repeat(" ", 10000))}
		req := httptest.NewRequest("POST", "/api/v1/gateway/version", body)
		req.Header.Set("X-Sync-Token", token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		wantStatus, maxRead := 401, 0
		if token == "test-deployment-token" {
			wantStatus, maxRead = 413, 4097
		}
		if rec.Code != wantStatus || body.bytes > maxRead {
			t.Errorf("gateway body was buffered before token/size boundary: status=%d bytes=%d, want %d <=%d", rec.Code, body.bytes, wantStatus, maxRead)
		}
	}
	config.C.Middleware.Casbin.Disable = true
	// A disabled Load never initializes its atomic pointer.
	rbacModule.Casbinx = &rbac.Casbinx{}
	if rec := request("GET", "/api/v1/system/updates", "delegated", ""); rec.Code != 403 {
		t.Errorf("disabled Casbin granted permission: %d %s", rec.Code, rec.Body)
	}
	if rec := request("GET", "/api/v1/system/updates", "trusted-root-id", ""); rec.Code != 200 {
		t.Errorf("disabled Casbin removed trusted Root authority: %d %s", rec.Code, rec.Body)
	}
}
