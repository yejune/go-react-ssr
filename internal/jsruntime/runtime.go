package jsruntime

import (
	"errors"
	"sync"
	"time"
)

// RuntimeType represents the type of JavaScript runtime
type RuntimeType string

const (
	RuntimeQuickJS RuntimeType = "quickjs"
	RuntimeV8      RuntimeType = "v8"
)

// defaultRuntimeType is set by init() in the build-specific files
var defaultRuntimeType RuntimeType

// JSRuntime is the interface for JavaScript execution
type JSRuntime interface {
	// Execute runs JavaScript code and returns the result as a string
	Execute(code string) (string, error)
	// Preload runs JavaScript code without returning a result (for loading bundles)
	Preload(code string) error
	// Close releases resources (called when returning to pool)
	Reset()
	// Destroy permanently destroys the runtime
	Destroy()
}

// Pool manages a pool of JS runtimes for reuse
type Pool struct {
	runtimeType RuntimeType
	pool        chan JSRuntime
	maxSize     int
	created     int
	closed      bool
	mu          sync.Mutex

	// Track all created runtimes for proper cleanup
	allRuntimes []JSRuntime
	runtimesMu  sync.Mutex

	// Preload code for overflow runtimes
	preloadCode string
	preloadMu   sync.RWMutex
}

// PoolConfig configures the runtime pool
type PoolConfig struct {
	RuntimeType RuntimeType
	PoolSize    int // Maximum number of runtimes to keep in pool
}

// DefaultRuntimeType returns the runtime type for this build
func DefaultRuntimeType() RuntimeType {
	return defaultRuntimeType
}

// NewPool creates a new runtime pool
func NewPool(config PoolConfig) *Pool {
	if config.PoolSize <= 0 {
		config.PoolSize = 10
	}
	// Use default runtime type if not specified
	if config.RuntimeType == "" {
		config.RuntimeType = defaultRuntimeType
	}

	p := &Pool{
		runtimeType: config.RuntimeType,
		maxSize:     config.PoolSize,
		pool:        make(chan JSRuntime, config.PoolSize),
		allRuntimes: make([]JSRuntime, 0, config.PoolSize),
	}

	// Pre-warm the pool
	for i := 0; i < config.PoolSize; i++ {
		rt := p.createRuntime()
		p.pool <- rt
	}

	return p
}

// createRuntime creates a new runtime and tracks it
// Also preloads bundle if preloadCode is set
func (p *Pool) createRuntime() JSRuntime {
	p.mu.Lock()
	p.created++
	p.mu.Unlock()

	rt := newRuntime()

	// Apply preload code if set (for overflow runtimes)
	p.preloadMu.RLock()
	code := p.preloadCode
	p.preloadMu.RUnlock()
	if code != "" {
		rt.Preload(code)
	}

	// Track for cleanup
	p.runtimesMu.Lock()
	p.allRuntimes = append(p.allRuntimes, rt)
	p.runtimesMu.Unlock()

	return rt
}

// PoolTimeout is the max time to wait for a runtime from pool
const PoolTimeout = 100 * time.Millisecond

// ErrPoolTimeout is returned when no runtime available within timeout
var ErrPoolTimeout = errors.New("pool timeout: no runtime available")

// Get retrieves a runtime from the pool with timeout
// If no runtime available within PoolTimeout, creates a temporary one
func (p *Pool) Get() JSRuntime {
	select {
	case rt := <-p.pool:
		return rt
	case <-time.After(PoolTimeout):
		// Pool exhausted, create overflow runtime (slower but prevents spike)
		return p.createRuntime()
	}
}

// GetBlocking retrieves a runtime, blocking indefinitely
func (p *Pool) GetBlocking() JSRuntime {
	return <-p.pool
}

// Put returns a runtime to the pool
// If pool is full (overflow runtime), destroys the runtime instead
func (p *Pool) Put(rt JSRuntime) {
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()

	if closed {
		rt.Destroy()
		return
	}

	rt.Reset()
	// Try to return to pool, but destroy if full (overflow runtime)
	select {
	case p.pool <- rt:
		// Successfully returned to pool
	default:
		// Pool is full (this was an overflow runtime), destroy it
		rt.Destroy()
	}
}

// Execute is a convenience method that gets a runtime, executes code, and returns it
func (p *Pool) Execute(code string) (string, error) {
	rt := p.Get()
	defer p.Put(rt)
	return rt.Execute(code)
}

// Preload runs preload code on all runtimes in the pool
// This is used to load heavy bundles once, so subsequent Execute calls are fast
// Also saves the code for overflow runtimes created later
func (p *Pool) Preload(code string) error {
	// Save for overflow runtimes
	p.preloadMu.Lock()
	p.preloadCode = code
	p.preloadMu.Unlock()

	// Drain all runtimes from pool
	runtimes := make([]JSRuntime, 0, p.maxSize)
	for i := 0; i < p.maxSize; i++ {
		rt := <-p.pool
		runtimes = append(runtimes, rt)
	}

	// Preload on each runtime
	var lastErr error
	for _, rt := range runtimes {
		if err := rt.Preload(code); err != nil {
			lastErr = err
		}
	}

	// Return all runtimes to pool
	for _, rt := range runtimes {
		p.pool <- rt
	}

	return lastErr
}

// Stats returns pool statistics
func (p *Pool) Stats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	return map[string]interface{}{
		"runtime_type":  p.runtimeType,
		"total_created": p.created,
		"max_pool_size": p.maxSize,
		"pool_size":     len(p.pool),
		"closed":        p.closed,
	}
}

// Close marks the pool as closed and destroys all runtimes
func (p *Pool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	// Drain the pool
	close(p.pool)
	for rt := range p.pool {
		rt.Destroy()
	}

	// Destroy any remaining tracked runtimes
	p.runtimesMu.Lock()
	for _, rt := range p.allRuntimes {
		rt.Destroy()
	}
	p.allRuntimes = nil
	p.runtimesMu.Unlock()
}
