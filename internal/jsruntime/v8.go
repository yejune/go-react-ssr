//go:build !use_quickjs && !use_moderncjs

package jsruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	v8 "github.com/tommie/v8go"
)

func init() {
	defaultRuntimeType = RuntimeV8
}

// newRuntime creates the default runtime for this build
func newRuntime() JSRuntime {
	return NewV8Runtime()
}

// V8Runtime wraps V8 for pooled usage
type V8Runtime struct {
	isolate       *v8.Isolate
	context       *v8.Context
	requestCount  int
	maxRequests   int // Context is recreated after this many requests to prevent memory buildup
	cachedScripts map[string]*v8.UnboundScript // hash -> compiled script
}

// NewV8Runtime creates a new V8 runtime
func NewV8Runtime() *V8Runtime {
	isolate := v8.NewIsolate()
	context := v8.NewContext(isolate)
	return &V8Runtime{
		isolate:       isolate,
		context:       context,
		maxRequests:   1000, // Recreate context every 1000 requests to prevent memory buildup
		cachedScripts: make(map[string]*v8.UnboundScript),
	}
}

// hashBundle creates a short hash of the bundle for cache lookup
func hashBundle(bundle string) string {
	h := sha256.Sum256([]byte(bundle))
	return hex.EncodeToString(h[:8]) // First 8 bytes = 16 hex chars
}

// Execute runs JavaScript code and returns the result
func (v *V8Runtime) Execute(code string) (string, error) {
	val, err := v.context.RunScript(code, "render.js")
	if err != nil {
		// Try to get more detailed error info
		if jsErr, ok := err.(*v8.JSError); ok {
			return "", fmt.Errorf("%s\n%s", jsErr.Message, jsErr.StackTrace)
		}
		return "", err
	}

	if val == nil {
		return "", nil
	}

	return val.String(), nil
}

// ExecuteWithProps runs a cached bundle with props injected
// The bundle is compiled once as UnboundScript and cached
// Only the small props script is compiled each request
func (v *V8Runtime) ExecuteWithProps(bundle, propsJSON string) (string, error) {
	// 1. Set props via small script (fast to compile)
	propsScript := fmt.Sprintf("var props = %s;", propsJSON)
	if _, err := v.context.RunScript(propsScript, "props.js"); err != nil {
		if jsErr, ok := err.(*v8.JSError); ok {
			return "", fmt.Errorf("props error: %s\n%s", jsErr.Message, jsErr.StackTrace)
		}
		return "", fmt.Errorf("props error: %w", err)
	}

	// 2. Get or compile cached bundle
	hash := hashBundle(bundle)
	script, ok := v.cachedScripts[hash]
	if !ok {
		var err error
		script, err = v.isolate.CompileUnboundScript(bundle, "bundle.js", v8.CompileOptions{
			Mode: v8.CompileModeEager, // Compile fully upfront
		})
		if err != nil {
			if jsErr, ok := err.(*v8.JSError); ok {
				return "", fmt.Errorf("compile error: %s\n%s", jsErr.Message, jsErr.StackTrace)
			}
			return "", fmt.Errorf("compile error: %w", err)
		}
		v.cachedScripts[hash] = script
	}

	// 3. Run cached script
	val, err := script.Run(v.context)
	if err != nil {
		if jsErr, ok := err.(*v8.JSError); ok {
			return "", fmt.Errorf("%s\n%s", jsErr.Message, jsErr.StackTrace)
		}
		return "", err
	}

	if val == nil {
		return "", nil
	}

	return val.String(), nil
}

// Reset prepares the runtime for reuse
// Context is reused to avoid expensive context creation/destruction
// Context is only recreated periodically to prevent memory buildup.
func (v *V8Runtime) Reset() {
	v.requestCount++

	// Periodically recreate context to prevent memory buildup
	if v.requestCount >= v.maxRequests {
		if v.context != nil {
			v.context.Close()
		}
		v.context = v8.NewContext(v.isolate)
		v.requestCount = 0
	}
	// Otherwise, reuse context as-is
	// JS bundle structure handles state reset:
	// - Banner: globalThis.__ssr_errors=[] (reset each execution)
	// - Code: var props = {...}; (new props each execution)
	// - Footer: calculates __ssr_result (new result each execution)
}

// Destroy permanently destroys the runtime
func (v *V8Runtime) Destroy() {
	if v.context != nil {
		v.context.Close()
		v.context = nil
	}
	if v.isolate != nil {
		v.isolate.Dispose()
		v.isolate = nil
	}
}
