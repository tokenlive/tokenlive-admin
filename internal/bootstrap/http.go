package bootstrap

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/utility/prom"
	"github.com/tokenlive/tokenlive-admin/internal/wirex"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"github.com/tokenlive/tokenlive-admin/pkg/middleware"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"go.uber.org/zap"
)

// EngineOptions controls optional HTTP surfaces when building/registering the admin engine.
type EngineOptions struct {
	// DisableHealth skips GET /health (host may register its own).
	DisableHealth bool
	// DisableSwagger skips openapi/swagger routes.
	DisableSwagger bool
	// DisableStatic skips SPA static middleware even if Static.Dir is set.
	DisableStatic bool
	// DisableNoRoute skips NoMethod/NoRoute handlers (useful when sharing a host gin.Engine).
	DisableNoRoute bool
}

// BuildEngine creates a new Gin engine with admin middlewares, routes, optional swagger/SPA.
func BuildEngine(ctx context.Context, injector *wirex.Injector, opts *EngineOptions) (*gin.Engine, error) {
	if config.C.IsDebug() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	e := gin.New()
	if err := RegisterTo(ctx, e, injector, opts); err != nil {
		return nil, err
	}
	return e, nil
}

// RegisterTo mounts admin middlewares and /api/v1 routes onto an existing gin.Engine.
// Safe for embed hosts that already own the engine (e.g. tokenlive-standalone).
func RegisterTo(ctx context.Context, e *gin.Engine, injector *wirex.Injector, opts *EngineOptions) error {
	if e == nil {
		return fmt.Errorf("bootstrap: gin engine is nil")
	}
	if injector == nil {
		return fmt.Errorf("bootstrap: injector is nil")
	}
	if opts == nil {
		opts = &EngineOptions{}
	}

	if !opts.DisableHealth {
		e.GET("/health", func(c *gin.Context) {
			util.ResOK(c)
		})
	}

	e.Use(middleware.RecoveryWithConfig(middleware.RecoveryConfig{
		Skip: config.C.Middleware.Recovery.Skip,
	}))

	if !opts.DisableNoRoute {
		e.NoMethod(func(c *gin.Context) {
			util.ResError(c, errors.MethodNotAllowed("", "Method Not Allowed"))
		})
		e.NoRoute(func(c *gin.Context) {
			util.ResError(c, errors.NotFound("", "Not Found"))
		})
	}

	allowedPrefixes := injector.M.RouterPrefixes()

	if err := useHTTPMiddlewares(ctx, e, injector, allowedPrefixes); err != nil {
		return err
	}

	if err := injector.M.RegisterRouters(ctx, e); err != nil {
		return err
	}

	if !opts.DisableSwagger && !config.C.General.DisableSwagger {
		e.StaticFile("/openapi.json", filepath.Join(config.C.General.WorkDir, "openapi.json"))
		e.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	if !opts.DisableStatic {
		if dir := config.C.Middleware.Static.Dir; dir != "" {
			e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
				Root:                dir,
				SkippedPathPrefixes: allowedPrefixes,
			}))
		}
	}

	return nil
}

// ListenAndServe starts HTTP on config.C.General.HTTP.Addr using the given handler.
// Returns a shutdown function.
func ListenAndServe(ctx context.Context, handler http.Handler) (func(), error) {
	if handler == nil {
		return nil, fmt.Errorf("bootstrap: handler is nil")
	}

	addr := config.C.General.HTTP.Addr
	logging.Context(ctx).Info(fmt.Sprintf("HTTP server is listening on %s", addr))
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  time.Second * time.Duration(config.C.General.HTTP.ReadTimeout),
		WriteTimeout: time.Second * time.Duration(config.C.General.HTTP.WriteTimeout),
		IdleTimeout:  time.Second * time.Duration(config.C.General.HTTP.IdleTimeout),
	}

	go func() {
		var err error
		if config.C.General.HTTP.CertFile != "" && config.C.General.HTTP.KeyFile != "" {
			srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
			err = srv.ListenAndServeTLS(config.C.General.HTTP.CertFile, config.C.General.HTTP.KeyFile)
		} else {
			err = srv.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			logging.Context(ctx).Error("Failed to listen http server", zap.Error(err))
		}
	}()

	return func() {
		ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(config.C.General.HTTP.ShutdownTimeout))
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			logging.Context(ctx).Error("Failed to shutdown http server", zap.Error(err))
		}
	}, nil
}

func startHTTPServer(ctx context.Context, injector *wirex.Injector) (func(), error) {
	e, err := BuildEngine(ctx, injector, nil)
	if err != nil {
		return nil, err
	}
	return ListenAndServe(ctx, e)
}

func useHTTPMiddlewares(_ context.Context, e *gin.Engine, injector *wirex.Injector, allowedPrefixes []string) error {
	// Bypass auth and casbin for the internal gateway config sync endpoints
	config.C.Middleware.Auth.SkippedPathPrefixes = append(config.C.Middleware.Auth.SkippedPathPrefixes,
		"/api/v1/gateway/config", "/api/v1/gateway/policies", "/api/v1/gateway/apikeys", "/api/v1/gateway/metrics", "/api/v1/gateway/events")
	config.C.Middleware.Casbin.SkippedPathPrefixes = append(config.C.Middleware.Casbin.SkippedPathPrefixes,
		"/api/v1/gateway/config", "/api/v1/gateway/policies", "/api/v1/gateway/apikeys", "/api/v1/gateway/metrics", "/api/v1/gateway/events")

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		Enable:                 config.C.Middleware.CORS.Enable,
		AllowAllOrigins:        config.C.Middleware.CORS.AllowAllOrigins,
		AllowOrigins:           config.C.Middleware.CORS.AllowOrigins,
		AllowMethods:           config.C.Middleware.CORS.AllowMethods,
		AllowHeaders:           config.C.Middleware.CORS.AllowHeaders,
		AllowCredentials:       config.C.Middleware.CORS.AllowCredentials,
		ExposeHeaders:          config.C.Middleware.CORS.ExposeHeaders,
		MaxAge:                 config.C.Middleware.CORS.MaxAge,
		AllowWildcard:          config.C.Middleware.CORS.AllowWildcard,
		AllowBrowserExtensions: config.C.Middleware.CORS.AllowBrowserExtensions,
		AllowWebSockets:        config.C.Middleware.CORS.AllowWebSockets,
		AllowFiles:             config.C.Middleware.CORS.AllowFiles,
	}))

	e.Use(middleware.TraceWithConfig(middleware.TraceConfig{
		AllowedPathPrefixes: allowedPrefixes,
		SkippedPathPrefixes: config.C.Middleware.Trace.SkippedPathPrefixes,
		RequestHeaderKey:    config.C.Middleware.Trace.RequestHeaderKey,
		ResponseTraceKey:    config.C.Middleware.Trace.ResponseTraceKey,
	}))

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		AllowedPathPrefixes:      allowedPrefixes,
		SkippedPathPrefixes:      config.C.Middleware.Logger.SkippedPathPrefixes,
		MaxOutputRequestBodyLen:  config.C.Middleware.Logger.MaxOutputRequestBodyLen,
		MaxOutputResponseBodyLen: config.C.Middleware.Logger.MaxOutputResponseBodyLen,
	}))

	e.Use(middleware.CopyBodyWithConfig(middleware.CopyBodyConfig{
		AllowedPathPrefixes: allowedPrefixes,
		SkippedPathPrefixes: config.C.Middleware.CopyBody.SkippedPathPrefixes,
		MaxContentLen:       config.C.Middleware.CopyBody.MaxContentLen,
	}))

	e.Use(middleware.AuthWithConfig(middleware.AuthConfig{
		AllowedPathPrefixes: allowedPrefixes,
		SkippedPathPrefixes: config.C.Middleware.Auth.SkippedPathPrefixes,
		ParseUserID:         injector.M.RBAC.LoginAPI.LoginBIZ.ParseUserID,
		RootID:              config.C.General.Root.ID,
		RootUsername:        config.C.General.Root.Username,
	}))

	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Enable:              config.C.Middleware.RateLimiter.Enable,
		AllowedPathPrefixes: allowedPrefixes,
		SkippedPathPrefixes: config.C.Middleware.RateLimiter.SkippedPathPrefixes,
		Period:              config.C.Middleware.RateLimiter.Period,
		MaxRequestsPerIP:    config.C.Middleware.RateLimiter.MaxRequestsPerIP,
		MaxRequestsPerUser:  config.C.Middleware.RateLimiter.MaxRequestsPerUser,
		StoreType:           config.C.Middleware.RateLimiter.Store.Type,
		MemoryStoreConfig: middleware.RateLimiterMemoryConfig{
			Expiration:      time.Second * time.Duration(config.C.Middleware.RateLimiter.Store.Memory.Expiration),
			CleanupInterval: time.Second * time.Duration(config.C.Middleware.RateLimiter.Store.Memory.CleanupInterval),
		},
		RedisStoreConfig: middleware.RateLimiterRedisConfig{
			Addr:     config.C.Middleware.RateLimiter.Store.Redis.Addr,
			Password: config.C.Middleware.RateLimiter.Store.Redis.Password,
			DB:       config.C.Middleware.RateLimiter.Store.Redis.DB,
			Username: config.C.Middleware.RateLimiter.Store.Redis.Username,
		},
	}))

	e.Use(middleware.CasbinWithConfig(middleware.CasbinConfig{
		AllowedPathPrefixes: allowedPrefixes,
		SkippedPathPrefixes: config.C.Middleware.Casbin.SkippedPathPrefixes,
		Skipper: func(c *gin.Context) bool {
			if config.C.Middleware.Casbin.Disable ||
				util.FromIsRootUser(c.Request.Context()) {
				return true
			}
			return false
		},
		GetEnforcer: func(c *gin.Context) *casbin.Enforcer {
			return injector.M.RBAC.Casbinx.GetEnforcer()
		},
		GetSubjects: func(c *gin.Context) []string {
			return util.FromUserCache(c.Request.Context()).RoleIDs
		},
	}))

	if config.C.Util.Prometheus.Enable {
		e.Use(prom.GinMiddleware)
	}

	return nil
}
