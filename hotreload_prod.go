//go:build prod

package go_ssr

// HotReload is a stub for production builds
type HotReload struct{}

// newHotReload returns nil in production (hot reload disabled)
func newHotReload(engine *Engine) *HotReload {
	return nil
}

// Start is a no-op in production
func (hr *HotReload) Start() {}
