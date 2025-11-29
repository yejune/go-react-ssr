package go_ssr

import (
	"context"
	"log/slog"
	"os"

	"github.com/yejune/go-react-ssr/internal/cache"
	"github.com/yejune/go-react-ssr/internal/jsruntime"
	"github.com/yejune/go-react-ssr/internal/utils"
)

type Engine struct {
	Logger                  *slog.Logger
	Config                  *Config
	HotReload               *HotReload
	CacheManager            *cache.Manager
	RuntimePool             *jsruntime.Pool
	CachedLayoutCSSFilePath string
}

// New creates a new gossr Engine instance
func New(config Config) (*Engine, error) {
	engine := &Engine{
		Logger:       slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})),
		Config:       &config,
		CacheManager: cache.NewManager(),
	}

	if err := os.Setenv("APP_ENV", config.AppEnv); err != nil {
		engine.Logger.Error("Failed to set APP_ENV environment variable", "error", err)
	}

	// Validate config first to set defaults
	err := config.Validate()
	if err != nil {
		engine.Logger.Error("Failed to validate config", "error", err)
		return nil, err
	}

	// Initialize the JS runtime pool after validation (defaults are now set)
	engine.RuntimePool = jsruntime.NewPool(jsruntime.PoolConfig{
		RuntimeType: config.JSRuntime,
		PoolSize:    config.JSRuntimePoolSize,
	})
	engine.Logger.Debug("Initialized JS runtime pool",
		"runtime", string(config.JSRuntime),
		"pool_size", config.JSRuntimePoolSize)
	utils.CleanCacheDirectories()
	// If using a layout css file, build it and cache it
	if config.LayoutCSSFilePath != "" {
		if err = engine.BuildLayoutCSSFile(); err != nil {
			engine.Logger.Error("Failed to build layout css file", "error", err)
			return nil, err
		}
	}

	// Initialize dev tools (hot reload, type converter) - no-op in prod builds
	if err := engine.initDevTools(); err != nil {
		return nil, err
	}

	return engine, nil
}

// Shutdown gracefully shuts down the engine and releases all resources.
// It should be called when the server is shutting down.
// The context can be used to set a timeout for the shutdown.
func (engine *Engine) Shutdown(ctx context.Context) error {
	engine.Logger.Info("Shutting down go-react-ssr engine")

	// Close the runtime pool
	if engine.RuntimePool != nil {
		engine.RuntimePool.Close()
		engine.Logger.Debug("Runtime pool closed")
	}

	// Clear the cache
	if engine.CacheManager != nil {
		engine.CacheManager.Clear()
		engine.Logger.Debug("Cache cleared")
	}

	// Stop hot reload server (dev only)
	engine.stopHotReload()

	engine.Logger.Info("go-react-ssr engine shutdown complete")
	return nil
}
