package jsruntime

import (
	"fmt"
	"testing"
)

// Simple JS code for basic benchmark
const simpleJS = `
var result = 0;
for (var i = 0; i < 1000; i++) {
	result += i;
}
result.toString();
`

// More complex JS code simulating React-like rendering
const complexJS = `
var props = { count: 42, name: "Test" };

function createElement(tag, props, children) {
	return {
		tag: tag,
		props: props || {},
		children: children || []
	};
}

function renderToString(element) {
	if (typeof element === 'string') return element;
	if (typeof element === 'number') return element.toString();

	var tag = element.tag;
	var props = element.props;
	var children = element.children;

	var attrs = '';
	for (var key in props) {
		if (key !== 'children') {
			attrs += ' ' + key + '="' + props[key] + '"';
		}
	}

	var childStr = '';
	for (var i = 0; i < children.length; i++) {
		childStr += renderToString(children[i]);
	}

	return '<' + tag + attrs + '>' + childStr + '</' + tag + '>';
}

var app = createElement('div', { id: 'root', class: 'container' }, [
	createElement('h1', {}, ['Hello ' + props.name]),
	createElement('p', {}, ['Count: ' + props.count]),
	createElement('ul', {}, [
		createElement('li', {}, ['Item 1']),
		createElement('li', {}, ['Item 2']),
		createElement('li', {}, ['Item 3']),
	])
]);

renderToString(app);
`

func BenchmarkRuntime_Simple(b *testing.B) {
	pool := NewPool(PoolConfig{
		PoolSize: 10,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pool.Execute(simpleJS)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRuntime_Complex(b *testing.B) {
	pool := NewPool(PoolConfig{
		PoolSize: 10,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pool.Execute(complexJS)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRuntime_NoPool(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt := newRuntime()
		_, err := rt.Execute(simpleJS)
		if err != nil {
			b.Fatal(err)
		}
		rt.Destroy()
	}
}

func BenchmarkRuntime_Parallel(b *testing.B) {
	pool := NewPool(PoolConfig{
		PoolSize: 20,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := pool.Execute(complexJS)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestRuntimeOutput(t *testing.T) {
	rt := newRuntime()
	defer rt.Destroy()

	result, err := rt.Execute(complexJS)
	if err != nil {
		t.Fatalf("Runtime error: %v", err)
	}

	expected := `<div id="root" class="container"><h1>Hello Test</h1><p>Count: 42</p><ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul></div>`
	if result != expected {
		t.Errorf("Unexpected result:\nGot: %s\nExpected: %s", result, expected)
	}

	fmt.Printf("Runtime: %s\nOutput: %s\n", DefaultRuntimeType(), result)
}
