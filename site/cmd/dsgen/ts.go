package main

import (
	"bytes"
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"sort"
	"strings"
)

// tsSource writes the structs as TypeScript: an interface per signals
// struct and per payload; Roots, the page's signals by the names they are
// declared under; Commands, each command's payload by the DOM event that
// sends it (<slug>-<name>); the //ds:dom names and the //ds:const
// constants as values. A named Go
// type of a basic kind is a named type here too: an enum (a type with
// constants) the union of their values, with a guard that tells one.
func tsSource(sigs []signals, cmds []command, doms []domConst, vals []valueConst) ([]byte, error) {
	g := &tsGen{aliases: map[string]string{}}
	var body bytes.Buffer
	var roots []signals
	for _, s := range sigs {
		st, err := g.structType(s.st, "")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", s.name, err)
		}
		where := ""
		if s.root != "" {
			where = " The page declares them under $" + s.root + "."
			roots = append(roots, s)
		}
		fmt.Fprintf(&body, "\n/** The signals of the Go struct %s.%s */\nexport interface %s %s\n", s.name, where, s.name, st)
	}
	if len(roots) > 0 {
		body.WriteString("\n/** The page's signals that this package declares, by name: lith-ui's signals<Roots>(). */\nexport interface Roots {\n")
		for _, s := range roots {
			fmt.Fprintf(&body, "\t%s: %s\n", s.root, s.name)
		}
		body.WriteString("}\n")
	}
	for _, c := range cmds {
		st, err := g.structType(c.st, "")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.name, err)
		}
		fmt.Fprintf(&body, "\n/** The payload of POST /cmd/%s/%s (Go: %s). */\nexport interface %s %s\n", c.slug, c.verb, c.name, c.name, st)
	}
	if len(cmds) > 0 {
		body.WriteString("\n/** The commands by the DOM event that sends each: its detail is the payload but the tab id, which the page adds. lith-ui's commands<Commands>(). */\nexport interface Commands {\n")
		for _, c := range cmds {
			fmt.Fprintf(&body, "\t%q: Omit<%s, \"tab\">\n", c.slug+"-"+c.verb, c.name)
		}
		body.WriteString("}\n")
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "// %s\n", header)
	if len(doms) > 0 {
		b.WriteString("\n// The names of the page's markup that the templates and the controller share (//ds:dom).\n")
		var acts []string
		for _, d := range doms {
			fmt.Fprintf(&b, "export const %s = %q\n", d.name, d.value)
			if strings.HasPrefix(d.name, "Act") {
				acts = append(acts, fmt.Sprintf("%q", d.value))
			}
		}
		// The controller implements every action the markup calls: lith-ui's
		// actions<Actions>() takes one function for each, and tsc fails on
		// one missing.
		if len(acts) > 0 {
			b.WriteString("\n/** The actions the page's markup calls (@name()), which the controller implements: lith-ui's actions<Actions>(). */\n")
			fmt.Fprintf(&b, "export type Actions = %s\n", strings.Join(acts, " | "))
		}
	}
	if len(vals) > 0 {
		b.WriteString("\n// The constants that the Go code and the controller share (//ds:const).\n")
		for _, v := range vals {
			fmt.Fprintf(&b, "export const %s = %s\n", v.name, v.lit)
		}
	}
	names := make([]string, 0, len(g.aliases))
	for n := range g.aliases {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		b.WriteString(g.aliases[n])
	}
	b.Write(body.Bytes())
	return b.Bytes(), nil
}

type tsGen struct {
	aliases map[string]string // a named type's declaration, by name
}

func (g *tsGen) structType(st *types.Struct, indent string) (string, error) {
	fields, err := jsonFields(st)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("{\n")
	for _, f := range fields {
		t, err := g.typ(f.typ, indent+"\t")
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

func (g *tsGen) typ(t types.Type, indent string) (string, error) {
	if named, ok := types.Unalias(t).(*types.Named); ok && named.Obj().Pkg() != nil {
		if _, basic := named.Underlying().(*types.Basic); basic {
			return g.named(named)
		}
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		return basicType(u)
	case *types.Struct:
		return g.structType(u, indent)
	case *types.Slice:
		if b, ok := u.Elem().Underlying().(*types.Basic); ok && b.Kind() == types.Byte {
			return "string", nil // base64, as encoding/json writes it
		}
		e, err := g.typ(u.Elem(), indent)
		return "(" + e + ")[]", err
	case *types.Array:
		e, err := g.typ(u.Elem(), indent)
		return "(" + e + ")[]", err
	case *types.Pointer:
		e, err := g.typ(u.Elem(), indent)
		return e + " | null", err
	case *types.Map:
		if k, ok := u.Key().Underlying().(*types.Basic); ok && k.Info()&types.IsString != 0 {
			e, err := g.typ(u.Elem(), indent)
			return "Record<string, " + e + ">", err
		}
	case *types.Interface:
		return "unknown", nil
	}
	return "", fmt.Errorf("%s has no JSON form a page reads", t)
}

func basicType(u *types.Basic) (string, error) {
	switch {
	case u.Info()&types.IsString != 0:
		return "string", nil
	case u.Info()&types.IsBoolean != 0:
		return "boolean", nil
	case u.Info()&types.IsNumeric != 0 && u.Info()&types.IsComplex == 0:
		return "number", nil
	}
	return "", fmt.Errorf("%s has no JSON form a page reads", u)
}

// named declares a named Go type of a basic kind under its name: an enum
// as the union of its constants' values and a guard, is<Name>, that tells
// a value of it (a lane from the page, say); another as its basic type.
func (g *tsGen) named(n *types.Named) (string, error) {
	name := n.Obj().Name()
	if _, done := g.aliases[name]; done {
		return name, nil
	}
	base, err := basicType(n.Underlying().(*types.Basic))
	if err != nil {
		return "", err
	}
	doc := fmt.Sprintf("\n/** Go: %s.%s. */\n", n.Obj().Pkg().Name(), name)
	vals := enumValues(n)
	if len(vals) == 0 {
		g.aliases[name] = fmt.Sprintf("%sexport type %s = %s\n", doc, name, base)
		return name, nil
	}
	list := strings.Join(vals, ", ")
	g.aliases[name] = fmt.Sprintf("%sexport type %s = %s\n\n/** Whether v is a %s. */\nexport const is%s = (v: unknown): v is %s => ([%s] as unknown[]).includes(v)\n",
		doc, name, strings.Join(vals, " | "), name, name, name, list)
	return name, nil
}

// enumValues returns the values of the constants of a named type in its
// package, in order: a Go enum stays one in TypeScript.
func enumValues(named *types.Named) []string {
	if b, ok := named.Underlying().(*types.Basic); !ok || b.Info()&(types.IsString|types.IsInteger) == 0 {
		return nil
	}
	scope := named.Obj().Pkg().Scope()
	type val struct {
		lit string
		v   constant.Value
	}
	var vals []val
	for _, name := range scope.Names() {
		c, ok := scope.Lookup(name).(*types.Const)
		if !ok || !types.Identical(c.Type(), named) {
			continue
		}
		lit := c.Val().ExactString()
		if c.Val().Kind() == constant.String {
			lit = fmt.Sprintf("%q", constant.StringVal(c.Val()))
		}
		vals = append(vals, val{lit, c.Val()})
	}
	sort.Slice(vals, func(i, j int) bool {
		if vals[i].v.Kind() == constant.Int && vals[j].v.Kind() == constant.Int {
			return constant.Compare(vals[i].v, token.LSS, vals[j].v)
		}
		return vals[i].lit < vals[j].lit
	})
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = v.lit
	}
	return out
}
