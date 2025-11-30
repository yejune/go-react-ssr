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

// NewQuickJSRuntime creates a new QuickJS runtime with optimized GC settings
func NewQuickJSRuntime() *QuickJSRuntime {
	// Disable automatic GC to prevent mid-request spikes
	// GC will be triggered manually during Reset()
	rt := quickjs.NewRuntime(
		quickjs.WithGCThreshold(-1),        // Disable automatic GC
		quickjs.WithMemoryLimit(256*1024*1024), // 256MB limit per runtime
		quickjs.WithMaxStackSize(1024*1024),    // 1MB stack
	)
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

// ExecuteWithProps runs bundle with props (QuickJS doesn't have UnboundScript, so just concatenate)
func (q *QuickJSRuntime) ExecuteWithProps(bundle, propsJSON string) (string, error) {
	code := "var props = " + propsJSON + "; " + bundle
	return q.Execute(code)
}

// Reset prepares the runtime for reuse
// QuickJS contexts can accumulate state, so we recreate the context
func (q *QuickJSRuntime) Reset() {
	// Close old context - this frees most resources via reference counting
	// QuickJS uses reference counting, so explicit GC is not needed here
	if q.context != nil {
		q.context.Close()
	}
	// Create new context for next request
	// Note: GC is disabled (-1 threshold), memory is managed via refcount
	q.context = q.runtime.NewContext()
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
