package main

import (
	"bytes"
	"fmt"
	"go/constant"
	"go/types"
	"sort"
	"strings"
)

// tsSource writes the structs as TypeScript: an interface per signals
// struct and per payload; Roots, the page's signals by the names they are
// declared under; and Commands, each command's payload by the DOM event
// that sends it (<slug>-<name>).
func tsSource(sigs []signals, cmds []command, doms []domConst) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "// %s\n", header)
	if len(doms) > 0 {
		b.WriteString("\n// The names of the page's markup that the templates and the controller share (//ds:dom).\n")
		for _, d := range doms {
			fmt.Fprintf(&b, "export const %s = %q\n", d.name, d.value)
		}
	}
	var roots []signals
	for _, s := range sigs {
		body, err := tsStruct(s.st, "")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.name, err)
		}
		where := ""
		if s.root != "" {
			where = " The page declares them under $" + s.root + "."
			roots = append(roots, s)
		}
		fmt.Fprintf(&b, "\n/** The signals of the Go struct %s.%s */\nexport interface %s %s\n", s.name, where, s.name, body)
	}
	if len(roots) > 0 {
		b.WriteString("\n/** The page's signals that this package declares, by name: lith-ui's signals<Roots>(). */\nexport interface Roots {\n")
		for _, s := range roots {
			fmt.Fprintf(&b, "\t%s: %s\n", s.root, s.name)
		}
		b.WriteString("}\n")
	}
	for _, c := range cmds {
		body, err := tsStruct(c.st, "")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.name, err)
		}
		fmt.Fprintf(&b, "\n/** The payload of POST /cmd/%s/%s (Go: %s). */\nexport interface %s %s\n", c.slug, c.verb, c.name, c.name, body)
	}
	if len(cmds) > 0 {
		b.WriteString("\n/** The commands by the DOM event that sends each: its detail is the payload but the tab id, which the page adds. lith-ui's command<Commands>(). */\nexport interface Commands {\n")
		for _, c := range cmds {
			fmt.Fprintf(&b, "\t%q: Omit<%s, \"tab\">\n", c.slug+"-"+c.verb, c.name)
		}
		b.WriteString("}\n")
	}
	return b.Bytes(), nil
}

func tsStruct(st *types.Struct, indent string) (string, error) {
	fields, err := jsonFields(st)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("{\n")
	for _, f := range fields {
		t, err := tsType(f.typ, indent+"\t")
		if err != nil {
			return "", fmt.Errorf("%s: %w", f.key, err)
		}
		opt := ""
		if f.optional {
			opt = "?"
		}
		fmt.Fprintf(&b, "%s\t%s%s: %s\n", indent, f.key, opt, t)
	}
	b.WriteString(indent + "}")
	return b.String(), nil
}

func tsType(t types.Type, indent string) (string, error) {
	if u := enum(t); u != "" {
		return u, nil
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch {
		case u.Info()&types.IsString != 0:
			return "string", nil
		case u.Info()&types.IsBoolean != 0:
			return "boolean", nil
		case u.Info()&types.IsNumeric != 0 && u.Info()&types.IsComplex == 0:
			return "number", nil
		}
	case *types.Struct:
		return tsStruct(u, indent)
	case *types.Slice:
		if b, ok := u.Elem().Underlying().(*types.Basic); ok && b.Kind() == types.Byte {
			return "string", nil // base64, as encoding/json writes it
		}
		e, err := tsType(u.Elem(), indent)
		return "(" + e + ")[]", err
	case *types.Array:
		e, err := tsType(u.Elem(), indent)
		return "(" + e + ")[]", err
	case *types.Pointer:
		e, err := tsType(u.Elem(), indent)
		return e + " | null", err
	case *types.Map:
		if k, ok := u.Key().Underlying().(*types.Basic); ok && k.Info()&types.IsString != 0 {
			e, err := tsType(u.Elem(), indent)
			return "Record<string, " + e + ">", err
		}
	case *types.Interface:
		return "unknown", nil
	}
	return "", fmt.Errorf("%s has no JSON form a page reads", t)
}

// enum returns a named string or integer type with constants in its
// package as the union of their values: a Go enum stays one in
// TypeScript.
func enum(t types.Type) string {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return ""
	}
	if b, ok := named.Underlying().(*types.Basic); !ok || b.Info()&(types.IsString|types.IsInteger) == 0 {
		return ""
	}
	scope := named.Obj().Pkg().Scope()
	var vals []string
	for _, name := range scope.Names() {
		c, ok := scope.Lookup(name).(*types.Const)
		if !ok || !types.Identical(c.Type(), named) {
			continue
		}
		v := c.Val().ExactString()
		if c.Val().Kind() == constant.String {
			v = fmt.Sprintf("%q", constant.StringVal(c.Val()))
		}
		vals = append(vals, v)
	}
	if len(vals) == 0 {
		return ""
	}
	sort.Strings(vals)
	return strings.Join(vals, " | ")
}
