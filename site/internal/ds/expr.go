package ds

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Expr is a Datastar expression built in Go: from refs (S.Op.Eq(""),
// S.Turn.Inc()), the builders of this file (If, And, Call) and the typed
// commands of package demo (PostSignals). A template takes one as an
// attribute's value, data-on:click={ ds.Call(ActDrop) }, and dscheck.Lint
// fails on an expression typed out in a template instead.
type Expr string

// Value is what an expression is made of: an Expr or a Ref. Only this
// package implements it.
type Value interface{ js() string }

func (e Expr) js() string { return string(e) }

// atom is an expression that needs no parentheses inside another: a
// signal, a literal, a call without nested parentheses, or one of those
// negated.
var atom = regexp.MustCompile(`^!*(?:\$[_a-zA-Z][_a-zA-Z0-9.]*|-?[0-9]+(?:\.[0-9]+)?|true|false|null|"(?:[^"\\]|\\.)*"|'[^'\\]*'|@[a-zA-Z][a-zA-Z0-9]*\([^()]*\))$`)

// operand is v inside another expression: in parentheses unless it is an
// atom, so that no operator of v binds to its neighbours.
func operand(v Value) string {
	s := v.js()
	if atom.MatchString(s) {
		return s
	}
	return "(" + s + ")"
}

func join(vs []Value, sep string) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = operand(v)
	}
	return strings.Join(parts, sep)
}

// Val is v as a literal expression (Lit).
func Val(v any) Expr { return Expr(Lit(v)) }

// And is true when every one of vs is, Or when any is: JavaScript's &&
// and ||, which yield the operand that decided.
func And(vs ...Value) Expr { return Expr(join(vs, " && ")) }
func Or(vs ...Value) Expr  { return Expr(join(vs, " || ")) }

// Not is the negation of v.
func Not(v Value) Expr { return Expr("!" + operand(v)) }

// Do runs steps in order: one statement for a data-on or a data-effect.
// The comma binds looser than any operator, so a step needs no
// parentheses, and a step that is a Do itself only adds to the list.
func Do(steps ...Value) Expr {
	parts := make([]string, len(steps))
	for i, s := range steps {
		parts[i] = s.js()
	}
	return Expr(strings.Join(parts, ", "))
}

// Cond is a while cond holds, else b: cond ? a : b.
func Cond(cond, a, b Value) Expr {
	return Expr(operand(cond) + " ? " + operand(a) + " : " + operand(b))
}

// Grouped is v, a number, as people read it: 12,345. The parens keep
// .toLocaleString out of the signal's path, which dscheck reads up to the
// last dot.
func Grouped(v Value) Expr { return Expr("(" + v.js() + ").toLocaleString('en')") }

// Max is the largest of vs, numbers: Math.max.
func Max(vs ...Value) Expr { return Expr("Math.max(" + join(vs, ", ") + ")") }

// If runs steps while cond holds: cond && (steps...).
func If(cond Value, steps ...Value) Expr {
	if len(steps) == 0 {
		panic("ds: If without a step")
	}
	return Expr(operand(cond) + " && " + operand(Do(steps...)))
}

// Concat is the strings vs one after another.
func Concat(vs ...Value) Expr { return Expr(join(vs, " + ")) }

// actionName is what Datastar calls an action by: no dots.
var actionName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)

// Call calls the action name with args: Call(ActDrop) is @lifeDrop(). A
// demo's actions are its //ds:dom Act… names, which its controller
// implements (lith-ui's actions): tsc fails on one it lacks, and
// dscheck.CheckDOM on a call of an action nobody registers.
func Call(name string, args ...Value) Expr {
	if !actionName.MatchString(name) {
		panic(fmt.Sprintf("ds: bad action name %q", name))
	}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = a.js()
	}
	return Expr("@" + name + "(" + strings.Join(parts, ", ") + ")")
}

// Invoke calls the element's own method name with args, when the element
// has one: a component's, such as sb-sparkline's push, which an
// expression reaches as el.
func Invoke(name string, args ...Value) Expr {
	if !actionName.MatchString(name) {
		panic(fmt.Sprintf("ds: bad method name %q", name))
	}
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = a.js()
	}
	return Expr("el." + name + " && el." + name + "(" + strings.Join(parts, ", ") + ")")
}

// OpID is a new op id, random: for a page without a controller (the
// guestbook's), which loads no lith-ui. OrderedOpID is one that sorts
// after every older one, lith-ui's orderedOpId: for a page whose op ids
// are compared (ack >= op), whose controller loads lith-ui.
var (
	OpID        = Expr("Date.now().toString(36) + Math.random().toString(36).slice(2, 7)")
	OrderedOpID = Call("orderedOpId")
)

// Aria is an ARIA state that is true or false, aria-pressed say:
// 'true' while cond holds, else 'false'.
func Aria(cond Value) Expr { return Expr(operand(cond) + " ? 'true' : 'false'") }

// AriaBusy is aria-busy while cond holds. data-attr writes true as an
// empty value, which aria-busy reads as false, and false removes the
// attribute: so 'true', or false.
func AriaBusy(cond Value) Expr { return Expr(operand(cond) + " ? 'true' : false") }

// Trimmed is v, a string, without its spaces at the ends. The parens keep
// .trim() out of the signal's path, which dscheck reads up to the last
// dot.
func Trimmed(v Value) Expr { return Expr("(" + v.js() + ").trim()") }

// Ms is d as a Datastar modifier's duration: data-on:input__throttle.
// + Ms(100*time.Millisecond) is data-on:input__throttle.100ms.
func Ms(d time.Duration) string { return strconv.FormatInt(d.Milliseconds(), 10) + "ms" }
