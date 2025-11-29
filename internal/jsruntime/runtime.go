package jsruntime

import "sync"

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
	// Close releases resources (called when returning to pool)
	Reset()
	// Destroy permanently destroys the runtime
	Destroy()
}

// Pool manages a pool of JS runtimes for reuse
type Pool struct {
	runtimeType RuntimeType
	pool        sync.Pool
	maxSize     int
	created     int
	closed      bool
	mu          sync.Mutex
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
	}

	p.pool = sync.Pool{
		New: func() interface{} {
			return p.createRuntime()
		},
	}

	// Pre-warm the pool
	runtimes := make([]JSRuntime, config.PoolSize)
	for i := 0; i < config.PoolSize; i++ {
		runtimes[i] = p.Get()
	}
	for _, rt := range runtimes {
		p.Put(rt)
	}

	return p
}

// createRuntime creates a new runtime (uses the only available runtime in this build)
func (p *Pool) createRuntime() JSRuntime {
	p.mu.Lock()
	p.created++
	p.mu.Unlock()

	return newRuntime()
}

// Get retrieves a runtime from the pool
func (p *Pool) Get() JSRuntime {
	return p.pool.Get().(JSRuntime)
}

// Put returns a runtime to the pool
func (p *Pool) Put(rt JSRuntime) {
	rt.Reset()
	p.pool.Put(rt)
}

// Execute is a convenience method that gets a runtime, executes code, and returns it
func (p *Pool) Execute(code string) (string, error) {
	rt := p.Get()
	defer p.Put(rt)
	return rt.Execute(code)
}

// Stats returns pool statistics
func (p *Pool) Stats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	return map[string]interface{}{
		"runtime_type":  p.runtimeType,
		"total_created": p.created,
		"max_pool_size": p.maxSize,
		"closed":        p.closed,
	}
}

// Close marks the pool as closed and prevents further use.
// Note: sync.Pool doesn't support iteration, so we can't destroy all runtimes.
// They will be garbage collected when no longer referenced.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
}
