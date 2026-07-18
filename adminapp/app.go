// Package adminapp is the public embed facade for tokenlive-admin.
// External hosts (tokenlive-standalone) can Init + Register onto a shared Gin engine
// without starting the CLI server or calling os.Exit.
package adminapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/bootstrap"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Options configures admin embed startup.
type Options struct {
	WorkDir   string // default "configs"
	Configs   string // relative to WorkDir, default "dev"
	StaticDir string // SPA directory; empty skips static middleware

	// Engine, if non-nil, is used for Register (host-owned). New does not listen.
	Engine *gin.Engine

	// When Engine is set, these defaults avoid clobbering host routes.
	// DisableHealth defaults true if Engine != nil.
	// DisableNoRoute defaults true if Engine != nil.
	DisableHealth  *bool
	DisableNoRoute *bool
	DisableSwagger bool
	DisableStatic  bool

	// OnConfigChanged is invoked after admin mutations that affect gateway runtime.
	// kind: endpoints | policies | apikeys | all
	OnConfigChanged util.ConfigChangeListener
}

// App is an initialized admin runtime (DB, mods, optional HTTP).
type App struct {
	rt   *bootstrap.Runtime
	opts Options

	mu       sync.Mutex
	engine   *gin.Engine
	stopHTTP func()
	closed   bool
}

// New loads config, wire, mods Init. Does not listen and does not block.
func New(ctx context.Context, opt Options) (*App, error) {
	if opt.WorkDir == "" {
		opt.WorkDir = "configs"
	}
	if opt.Configs == "" {
		opt.Configs = "dev"
	}

	if opt.OnConfigChanged != nil {
		util.OnConfigChanged(opt.OnConfigChanged)
	}

	rt, err := bootstrap.Init(ctx, bootstrap.RunConfig{
		WorkDir:   opt.WorkDir,
		Configs:   opt.Configs,
		StaticDir: opt.StaticDir,
	})
	if err != nil {
		return nil, err
	}

	app := &App{rt: rt, opts: opt}

	if opt.Engine != nil {
		if err := app.Register(ctx, opt.Engine); err != nil {
			app.Shutdown(ctx)
			return nil, err
		}
	}

	return app, nil
}

// Register mounts admin API (and optional swagger/SPA) on e.
func (a *App) Register(ctx context.Context, e *gin.Engine) error {
	if a == nil || a.rt == nil {
		return fmt.Errorf("adminapp: not initialized")
	}
	if e == nil {
		return fmt.Errorf("adminapp: engine is nil")
	}

	engOpts := a.engineOptions(true)
	if err := bootstrap.RegisterTo(ctx, e, a.rt.Injector, engOpts); err != nil {
		return err
	}

	a.mu.Lock()
	a.engine = e
	a.mu.Unlock()
	return nil
}

// Handler returns an http.Handler with a dedicated Gin engine (builds once).
func (a *App) Handler() http.Handler {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.engine != nil {
		return a.engine
	}
	e, err := bootstrap.BuildEngine(a.rt.Ctx, a.rt.Injector, a.engineOptions(false))
	if err != nil {
		// Return a handler that always 500 — Register/New should have failed first.
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		})
	}
	a.engine = e
	return e
}

// Start listens on admin HTTP addr from config (CLI-compatible path without util.Run/os.Exit).
// Host should still call Shutdown. For full CLI signal loop, use bootstrap.Run.
func (a *App) Start(ctx context.Context) error {
	if a == nil || a.rt == nil {
		return fmt.Errorf("adminapp: not initialized")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stopHTTP != nil {
		return nil
	}
	h := a.engine
	if h == nil {
		e, err := bootstrap.BuildEngine(ctx, a.rt.Injector, a.engineOptions(false))
		if err != nil {
			return err
		}
		a.engine = e
		h = e
	}
	stop, err := bootstrap.ListenAndServe(ctx, h)
	if err != nil {
		return err
	}
	a.stopHTTP = stop
	return nil
}

// Shutdown stops HTTP (if Start was used), releases mods, and cleans DI/logger.
// Does not call os.Exit.
func (a *App) Shutdown(ctx context.Context) error {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return nil
	}
	a.closed = true
	stop := a.stopHTTP
	a.stopHTTP = nil
	a.mu.Unlock()

	if stop != nil {
		stop()
	}
	if a.rt != nil {
		a.rt.Release(ctx)
	}
	return nil
}

func (a *App) engineOptions(hostOwned bool) *bootstrap.EngineOptions {
	opts := &bootstrap.EngineOptions{
		DisableSwagger: a.opts.DisableSwagger,
		DisableStatic:  a.opts.DisableStatic,
	}
	// Defaults when attaching to host engine.
	disableHealth := hostOwned
	disableNoRoute := hostOwned
	if a.opts.DisableHealth != nil {
		disableHealth = *a.opts.DisableHealth
	}
	if a.opts.DisableNoRoute != nil {
		disableNoRoute = *a.opts.DisableNoRoute
	}
	opts.DisableHealth = disableHealth
	opts.DisableNoRoute = disableNoRoute
	return opts
}

// GatewaySnapshot is JSON-encoded admin gateway export (same shape as /api/v1/gateway/*).
// Hosts unmarshal into tokenlive-gateway config types without importing admin internal packages.
type GatewaySnapshot struct {
	ConfigJSON   []byte
	PoliciesJSON []byte
	APIKeysJSON  []byte
}

// LoadGatewaySnapshot loads routing config, policies, and API keys from the admin DB
// (same data as Gateway Sync HTTP APIs, without network or sync token).
func (a *App) LoadGatewaySnapshot(ctx context.Context) (*GatewaySnapshot, error) {
	if a == nil || a.rt == nil || a.rt.Injector == nil || a.rt.Injector.M == nil {
		return nil, fmt.Errorf("adminapp: not initialized")
	}
	res := a.rt.Injector.M.Resource
	if res == nil || res.GatewaySyncAPI == nil || res.GatewaySyncAPI.GatewaySyncBIZ == nil {
		return nil, fmt.Errorf("adminapp: gateway sync biz not available")
	}
	biz := res.GatewaySyncAPI.GatewaySyncBIZ

	cfg, err := biz.GetGatewayConfig(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("adminapp: snapshot config: %w", err)
	}
	policies, err := biz.GetGatewayPolicies(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("adminapp: snapshot policies: %w", err)
	}
	keys, err := biz.GetGatewayApiKeys(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("adminapp: snapshot apikeys: %w", err)
	}

	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	polJSON, err := json.Marshal(policies)
	if err != nil {
		return nil, err
	}
	keyJSON, err := json.Marshal(keys)
	if err != nil {
		return nil, err
	}
	return &GatewaySnapshot{
		ConfigJSON:   cfgJSON,
		PoliciesJSON: polJSON,
		APIKeysJSON:  keyJSON,
	}, nil
}
