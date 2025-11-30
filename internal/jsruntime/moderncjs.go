//go:build use_moderncjs

package jsruntime

import (
	"fmt"

	"modernc.org/quickjs"
)

func init() {
	defaultRuntimeType = RuntimeModerncJS
}

// newRuntime creates the default runtime for this build
func newRuntime() JSRuntime {
	return NewModerncJSRuntime()
}

// ModerncJSRuntime wraps modernc.org/quickjs (pure Go port) for pooled usage
type ModerncJSRuntime struct {
	vm *quickjs.VM
}

// NewModerncJSRuntime creates a new pure Go QuickJS runtime
func NewModerncJSRuntime() *ModerncJSRuntime {
	vm, err := quickjs.NewVM()
	if err != nil {
		panic("failed to create modernc quickjs VM: " + err.Error())
	}
	// Configure VM settings
	vm.SetMemoryLimit(256 * 1024 * 1024) // 256MB limit
	vm.SetGCThreshold(0)                  // Disable automatic GC (0 = disabled)
	return &ModerncJSRuntime{
		vm: vm,
	}
}

// Execute runs JavaScript code and returns the result
func (m *ModerncJSRuntime) Execute(code string) (string, error) {
	res, err := m.vm.Eval(code, quickjs.EvalGlobal)
	if err != nil {
		return "", fmt.Errorf("JS execution error: %w", err)
	}

	if res == nil {
		return "", nil
	}

	// Convert result to string
	switch v := res.(type) {
	case string:
		return v, nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	case float64:
		return fmt.Sprintf("%f", v), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case quickjs.Undefined:
		return "", nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// ExecuteWithProps runs bundle with props (no bytecode caching, just concatenate)
func (m *ModerncJSRuntime) ExecuteWithProps(bundle, propsJSON string) (string, error) {
	code := "var props = " + propsJSON + "; " + bundle
	return m.Execute(code)
}

// Reset prepares the runtime for reuse
// Pure Go implementation - just close and create new VM
func (m *ModerncJSRuntime) Reset() {
	if m.vm != nil {
		m.vm.Close()
	}
	vm, err := quickjs.NewVM()
	if err != nil {
		panic("failed to create modernc quickjs VM: " + err.Error())
	}
	vm.SetMemoryLimit(256 * 1024 * 1024)
	vm.SetGCThreshold(0)
	m.vm = vm
}

// Destroy permanently destroys the runtime
func (m *ModerncJSRuntime) Destroy() {
	if m.vm != nil {
		m.vm.Close()
		m.vm = nil
	}
}
