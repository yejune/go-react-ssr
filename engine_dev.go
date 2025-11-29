//go:build !prod

package go_ssr

import (
	"os"

	"github.com/yejune/gotossr/internal/typeconverter"
)

// initDevTools initializes development tools (hot reload, type converter)
func (engine *Engine) initDevTools() error {
	// If running in production mode (APP_ENV), skip dev tools
	if os.Getenv("APP_ENV") == "production" {
		engine.Logger.Info("Running go-ssr in production mode")
		return nil
	}

	engine.Logger.Info("Running go-ssr in development mode")
	engine.Logger.Debug("Starting type converter")

	// Start the type converter to convert Go types to Typescript types
	if err := typeconverter.Start(engine.Config.PropsStructsPath, engine.Config.GeneratedTypesPath); err != nil {
		engine.Logger.Error("Failed to init type converter", "error", err)
		return err
	}

	engine.Logger.Debug("Starting hot reload server")
	engine.HotReload = newHotReload(engine)
	engine.HotReload.Start()

	return nil
}

// stopHotReload stops the hot reload server (dev only)
func (engine *Engine) stopHotReload() {
	if engine.HotReload != nil {
		engine.Logger.Debug("Hot reload server stopping")
		// Hot reload server runs in a goroutine; it will be cleaned up on process exit
	}
}
