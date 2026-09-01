package bootstrap

import (
	"context"
	"net/http"
	_ "net/http/pprof" //nolint:gosec
	"os"
	"strings"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	_ "github.com/tokenlive/tokenlive-admin/internal/swagger"
	"github.com/tokenlive/tokenlive-admin/internal/utility/prom"
	"github.com/tokenlive/tokenlive-admin/internal/wirex"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"go.uber.org/zap"
)

// RunConfig defines the config for run command.
type RunConfig struct {
	WorkDir   string // Working directory
	Configs   string // Directory or files (multiple separated by commas)
	StaticDir string // Static files directory
	Version   string // Optional version override
}

// Runtime holds an initialized admin stack without listening.
// Used by adminapp embed facade and by Run (CLI).
type Runtime struct {
	Injector       *wirex.Injector
	CleanInjector  func()
	CleanLogger    func()
	Ctx            context.Context
}

// Init loads config, logger, wire injector, mods Init, and prometheus.
// It does not start HTTP. Caller must invoke Release.
func Init(ctx context.Context, runCfg RunConfig) (*Runtime, error) {
	workDir := runCfg.WorkDir
	staticDir := runCfg.StaticDir
	config.MustLoad(workDir, strings.Split(runCfg.Configs, ",")...)
	if runCfg.Version != "" {
		config.C.General.Version = runCfg.Version
	}
	config.C.General.WorkDir = workDir
	config.C.Middleware.Static.Dir = staticDir
	config.C.Print()
	config.C.PreLoad()
	initCaptcha()

	cleanLoggerFn, err := logging.InitWithConfig(ctx, &config.C.Logger, initLoggerHook)
	if err != nil {
		return nil, err
	}
	ctx = logging.NewTag(ctx, logging.TagKeyMain)

	logging.Context(ctx).Info("starting service ...",
		zap.String("version", config.C.General.Version),
		zap.Int("pid", os.Getpid()),
		zap.String("workdir", workDir),
		zap.String("config", runCfg.Configs),
		zap.String("static", staticDir),
	)

	// Start pprof server.
	if addr := config.C.General.PprofAddr; addr != "" {
		logging.Context(ctx).Info("pprof server is listening on " + addr)
		go func() {
			err := http.ListenAndServe(addr, nil)
			if err != nil {
				logging.Context(ctx).Error("failed to listen pprof server", zap.Error(err))
			}
		}()
	}

	injector, cleanInjectorFn, err := wirex.BuildInjector(ctx)
	if err != nil {
		if cleanLoggerFn != nil {
			cleanLoggerFn()
		}
		return nil, err
	}

	if err := injector.M.Init(ctx); err != nil {
		if cleanInjectorFn != nil {
			cleanInjectorFn()
		}
		if cleanLoggerFn != nil {
			cleanLoggerFn()
		}
		return nil, err
	}

	prom.Init()

	return &Runtime{
		Injector:      injector,
		CleanInjector: cleanInjectorFn,
		CleanLogger:   cleanLoggerFn,
		Ctx:           ctx,
	}, nil
}

// Release stops mods and cleans injector/logger.
func (rt *Runtime) Release(ctx context.Context) {
	if rt == nil {
		return
	}
	if rt.Injector != nil {
		if err := rt.Injector.M.Release(ctx); err != nil {
			logging.Context(ctx).Error("failed to release injector", zap.Error(err))
		}
	}
	if rt.CleanInjector != nil {
		rt.CleanInjector()
	}
	if rt.CleanLogger != nil {
		rt.CleanLogger()
	}
}

// The Run function initializes and starts a service with configuration and logging, and handles
// cleanup upon exit. CLI path uses util.Run which may os.Exit.
func Run(ctx context.Context, runCfg RunConfig) error {
	defer func() {
		_ = zap.L().Sync()
	}()

	rt, err := Init(ctx, runCfg)
	if err != nil {
		return err
	}

	return util.Run(rt.Ctx, func(ctx context.Context) (func(), error) {
		cleanHTTPServerFn, err := startHTTPServer(ctx, rt.Injector)
		if err != nil {
			return func() { rt.Release(ctx) }, err
		}

		return func() {
			if cleanHTTPServerFn != nil {
				cleanHTTPServerFn()
			}
			rt.Release(ctx)
		}, nil
	})
}
