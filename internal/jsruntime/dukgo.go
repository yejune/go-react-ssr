//go:build use_dukgo

package jsruntime

import (
	"runtime"

	djs "github.com/rosbit/dukgo"
)

func init() {
	defaultRuntimeType = RuntimeDukgo
	defaultESTarget = "es5" // Duktape only supports ES5
}

// newRuntime creates the default runtime for this build
func newRuntime() JSRuntime {
	return NewDukgoRuntime()
}

// DukgoRuntime wraps Duktape via dukgo for pooled usage
type DukgoRuntime struct {
	context *djs.JsContext
}

// NewDukgoRuntime creates a new Duktape runtime
func NewDukgoRuntime() *DukgoRuntime {
	ctx, err := djs.NewContext()
	if err != nil {
		panic("failed to create dukgo context: " + err.Error())
	}
	return &DukgoRuntime{
		context: ctx,
	}
}

// Execute runs JavaScript code and returns the result
func (d *DukgoRuntime) Execute(code string) (string, error) {
	res, err := d.context.Eval(code, nil)
	if err != nil {
		return "", err
	}

	if res == nil {
		return "", nil
	}

	// Convert result to string
	switch v := res.(type) {
	case string:
		return v, nil
	default:
		// For other types, convert using fmt.Sprintf
		str, err := d.context.CallFunc("String", res)
		if err != nil {
			return "", err
		}
		if s, ok := str.(string); ok {
			return s, nil
		}
		return "", nil
	}
}

// Reset prepares the runtime for reuse
// Create a new context to clear any global state from previous executions
func (d *DukgoRuntime) Reset() {
	// Let GC handle the old context via finalizer
	d.context = nil
	runtime.GC() // Hint to GC to clean up the old context

	ctx, err := djs.NewContext()
	if err != nil {
		panic("failed to create dukgo context: " + err.Error())
	}
	d.context = ctx
}

// Destroy permanently destroys the runtime
func (d *DukgoRuntime) Destroy() {
	// dukgo uses finalizers for cleanup
	d.context = nil
}
