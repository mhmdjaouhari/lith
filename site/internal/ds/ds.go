// Package ds types the seam between Go and Datastar: the signals a page
// declares, the expressions that read them and the patches the server
// sends into them all come from one Go struct.
//
// A demo declares its signals as a struct with a //ds:signals directive
// and runs cmd/dsgen, which writes a var of typed references, one per
// field (S.Op is a Ref[string] that reads "$_board.op"), and the same
// struct as a TypeScript interface for the demo's controller. Renaming a
// field breaks the Go build at every reference, and tsc at every use in
// the controller, instead of a page at runtime.
//
//	//ds:signals _board S
//	type Signals struct {
//		V  int64  `json:"v"`
//		Op string `json:"op"`
//	}
//
// The declaration is the struct's JSON (Decl), a server patch is a list
// of typed assignments (Set), and an expression is built from refs and
// Go values (Ref.Eq, Lit), never typed out by hand.
package ds

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Ref is a signal at a path, such as "_board.op", whose value is a T.
type Ref[T any] struct{ path string }

// NewRef returns the ref of the signal at path. Generated code calls it;
// a demo uses the refs dsgen writes.
func NewRef[T any](path string) Ref[T] {
	if path == "" || strings.ContainsAny(path, " $'\"") {
		panic(fmt.Sprintf("ds: bad signal path %q", path))
	}
	return Ref[T]{path}
}

// String is the signal in an expression: "$_board.op".
func (r Ref[T]) String() string { return "$" + r.path }

// Path is the signal's path without the $: "_board.op", as a signal
// patch filter and Datastar's getPath name it.
func (r Ref[T]) Path() string { return r.path }

// Set is the assignment of v to the signal, for a patch (Patch).
func (r Ref[T]) Set(v T) Assign { return Assign{path: r.path, value: v} }

// Assign is the JS statement that sets the signal to v.
func (r Ref[T]) Assign(v T) string { return r.String() + " = " + Lit(v) }

// Eq is the expression that the signal equals v.
func (r Ref[T]) Eq(v T) string { return r.String() + " === " + Lit(v) }

// Ne is the expression that the signal differs from v.
func (r Ref[T]) Ne(v T) string { return r.String() + " !== " + Lit(v) }

// Lit is v as a JavaScript literal. JSON is one, quotes and line breaks
// escaped, so copy with an apostrophe can never end an expression early.
func Lit(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("ds: %T is no literal: %v", v, err))
	}
	return string(b)
}

// Assign is a typed value for a signal, built with Ref.Set.
type Assign struct {
	path  string
	value any
}

// Tree nests assignments by their paths: Set("_board.ack") and
// Set("_board.ok") become {"_board": {"ack": ..., "ok": ...}}, the shape a
// signal patch merges. Two values for one path, or a value where another
// assignment needs an object, panic: no page could mean that.
func Tree(as ...Assign) map[string]any {
	root := map[string]any{}
	for _, a := range as {
		keys := strings.Split(a.path, ".")
		m := root
		for _, k := range keys[:len(keys)-1] {
			next, ok := m[k]
			if !ok {
				next = map[string]any{}
				m[k] = next
			}
			sub, ok := next.(map[string]any)
			if !ok {
				panic(fmt.Sprintf("ds: %s sets a value and an object", a.path))
			}
			m = sub
		}
		last := keys[len(keys)-1]
		if _, dup := m[last]; dup {
			panic(fmt.Sprintf("ds: %s is set twice", a.path))
		}
		m[last] = a.value
	}
	return root
}

// Decl is the data-signals value that declares v under root, as
// {"_board": {...}}: v's JSON, so every field of the struct is declared
// with its Go value, and none can be left out or misspelt.
func Decl(root string, v any) string {
	return Lit(map[string]any{root: v})
}
