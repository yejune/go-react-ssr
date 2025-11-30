//go:build prod

package go_ssr

// initDevTools is a no-op in production builds
func (engine *Engine) initDevTools() error {
	engine.Logger.Info("Running go-ssr in production mode")
	return nil
}

// stopHotReload is a no-op in production builds
func (engine *Engine) stopHotReload() {
	// No hot reload in production
}
