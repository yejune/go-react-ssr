package go_ssr

// Generator interface for custom code generation
// Implementations can generate routes, API clients, or any other code
// based on the SSR configuration
type Generator interface {
	// Generate is called during engine initialization (dev mode only)
	// config contains all SSR configuration including props paths
	Generate(config *Config) error
}
