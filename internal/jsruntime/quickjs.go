//go:build use_quickjs

package jsruntime

import (
	"github.com/buke/quickjs-go"
)

func init() {
	defaultRuntimeType = RuntimeQuickJS
}

// newRuntime creates the default runtime for this build
func newRuntime() JSRuntime {
	return NewQuickJSRuntime()
}

// QuickJSRuntime wraps QuickJS for pooled usage
type QuickJSRuntime struct {
	runtime *quickjs.Runtime
	context *quickjs.Context
}

// NewQuickJSRuntime creates a new QuickJS runtime
func NewQuickJSRuntime() *QuickJSRuntime {
	rt := quickjs.NewRuntime()
	ctx := rt.NewContext()
	return &QuickJSRuntime{
		runtime: rt,
		context: ctx,
	}
}

// Execute runs JavaScript code and returns the result
func (q *QuickJSRuntime) Execute(code string) (string, error) {
	res := q.context.Eval(code)
	defer res.Free()

	if res.IsException() {
		return "", res.Error()
	}

	return res.String(), nil
}

// Preload runs JavaScript code without returning a result
// Used to load heavy bundles once per runtime
func (q *QuickJSRuntime) Preload(code string) error {
	res := q.context.Eval(code)
	defer res.Free()

	if res.IsException() {
		return res.Error()
	}
	return nil
}

// Reset prepares the runtime for reuse
// Instead of recreating context (expensive), we clear only per-request state
// Note: __ssrRender is the preloaded render function - DO NOT delete it
func (q *QuickJSRuntime) Reset() {
	// Clear only per-request globals, keep preloaded bundle
	q.context.Eval(`
		globalThis.__ssr_errors = [];
	`)
}

// Destroy permanently destroys the runtime
func (q *QuickJSRuntime) Destroy() {
	if q.context != nil {
		q.context.Close()
		q.context = nil
	}
	if q.runtime != nil {
		q.runtime.Close()
		q.runtime = nil
	}
}
