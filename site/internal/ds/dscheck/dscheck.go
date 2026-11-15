// Package dscheck checks what the compiler cannot see of a page's
// Datastar: it reads rendered HTML and reports every Datastar attribute
// whose plugin is unknown or misspelt, whose expression is no valid
// JavaScript, or that reads a signal its page does not declare. A wrong
// attribute in Datastar is ignored without an error, and a wrong
// expression fails only in a browser; here both fail a test that names
// the attribute.
//
//	errs := dscheck.Check(html, dscheck.Schema{"_board": reflect.TypeFor[board.Signals]()})
//
// Templates (Lint) must not write expressions inline: an attribute value
// with $ or @ is built in Go, from ds refs, where the compiler checks it.
package dscheck

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/zweiundeins/lith/site/internal/ds"
)

// Plugins are the attributes the site's Datastar build reads (v1.0.5
// with Rocket: its 17 plugins, and the attributes its morph and plugins
// read), and lith's newest-frame.js, by name, with whether the value is
// an expression to check.
var Plugins = map[string]bool{
	"attr": true, "bind": false, "class": true, "computed": true, "effect": true,
	"indicator": false, "init": true, "json-signals": false, "on": true,
	"on-intersect": true, "on-interval": true, "on-signal-patch": true,
	"ref": false, "show": true, "signals": true, "style": true, "text": true,
	"on-signal-patch-filter": true, "ignore": false, "ignore-morph": false,
	"preserve-attr": false, "newest-frame": false,
}

// keyedPlugins take a key after a colon: data-on:click, never data-on-click.
var keyedPlugins = []string{"attr", "bind", "class", "computed", "indicator", "on", "ref", "signals", "style"}

// Schema is a page's signals: each top-level signal's Go type, a struct
// whose JSON fields are the signals under it, or a basic type.
type Schema map[string]reflect.Type

// Page are the signals the site's shell declares on every streaming page
// (views.pageSignals) and the indicator of its stream.
var Page = Schema{
	"_tab": reflect.TypeFor[string](), "_slow": reflect.TypeFor[bool](), "_badge": reflect.TypeFor[string](),
	"_build": reflect.TypeFor[string](), "_assets": reflect.TypeFor[string](), "_live": reflect.TypeFor[bool](),
}

var (
	signalRef = regexp.MustCompile(`\$([_a-zA-Z][_a-zA-Z0-9]*(?:\.[_a-zA-Z][_a-zA-Z0-9]*)*)`)
	actionRef = regexp.MustCompile(`@([a-zA-Z]+)\(`)
	pluginKey = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)
)

// Check returns what is wrong with the Datastar attributes of the HTML
// fragment page; controllers are the names of attribute plugins the
// page's modules register (data-board), which take no value.
func Check(page string, schema Schema, controllers ...string) []error {
	all := Schema{}
	for k, v := range Page {
		all[k] = v
	}
	for k, v := range schema {
		all[k] = v
	}
	nodes, err := html.ParseFragment(strings.NewReader(page), &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body})
	if err != nil {
		return []error{err}
	}
	var errs []error
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, a := range n.Attr {
				if err := checkAttr(n, a, all, controllers); err != nil {
					errs = append(errs, err)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	return errs
}

func checkAttr(n *html.Node, a html.Attribute, schema Schema, controllers []string) error {
	name, ok := strings.CutPrefix(a.Key, "data-")
	if !ok {
		return nil
	}
	where := fmt.Sprintf("<%s %s>", n.Data, a.Key)
	plugin, _, _ := strings.Cut(name, "__")
	plugin, key, keyed := strings.Cut(plugin, ":")
	for _, c := range controllers {
		if plugin == c {
			if a.Val != "" || keyed {
				return fmt.Errorf("%s: a controller takes no key and no value", where)
			}
			return nil
		}
	}
	expr, known := Plugins[plugin]
	if !known {
		// A data attribute of the page's own (data-card), unless it is a
		// keyed plugin written with a hyphen, which Datastar ignores.
		for _, p := range keyedPlugins {
			if strings.HasPrefix(plugin, p+"-") {
				return fmt.Errorf("%s: no Datastar plugin; its key follows a colon: data-%s:%s", where, p, strings.TrimPrefix(plugin, p+"-"))
			}
		}
		if signalRef.MatchString(a.Val) {
			return fmt.Errorf("%s: reads signals, but is no Datastar attribute", where)
		}
		return nil
	}
	if keyed && !pluginKey.MatchString(key) {
		return fmt.Errorf("%s: key %q: lowercase, digits and -", where, key)
	}
	if !expr {
		return nil
	}
	if strings.TrimSpace(a.Val) == "" {
		return fmt.Errorf("%s: no expression", where)
	}
	if err := Syntax(a.Val, returnsValue[plugin]); err != nil {
		return fmt.Errorf("%s: %v\n\t%s", where, err, a.Val)
	}
	for _, m := range signalRef.FindAllStringSubmatch(a.Val, -1) {
		if err := schema.Has(m[1]); err != nil {
			return fmt.Errorf("%s: $%s: %v", where, m[1], err)
		}
	}
	return nil
}

// returnsValue are the plugins whose expression Datastar evaluates for
// its value: one that is a single expression, such as data-signals' object
// literal, is valid there though it is no statement.
var returnsValue = map[string]bool{"attr": true, "class": true, "computed": true, "show": true, "signals": true, "style": true, "text": true}

// Syntax reports whether expr is a valid Datastar expression: statements
// with $signals and @actions, run with el, evt and patch, or, for a
// plugin that takes its value, an expression too.
func Syntax(expr string, value bool) error {
	js := signalRef.ReplaceAllString(expr, "__s.$1")
	js = actionRef.ReplaceAllString(js, "__a.$1(")
	parse := func(body string) error {
		r := api.Transform("(function (el, evt, patch, __s, __a) {\n"+body+"\n})", api.TransformOptions{Loader: api.LoaderJS})
		if len(r.Errors) > 0 {
			return fmt.Errorf("not JavaScript: %s", r.Errors[0].Text)
		}
		return nil
	}
	err := parse(js)
	if err != nil && value && parse("return (\n"+js+"\n)") == nil {
		return nil
	}
	return err
}

// Has reports whether the schema declares the signal at path.
func (s Schema) Has(path string) error {
	keys := strings.Split(path, ".")
	t, ok := s[keys[0]]
	if !ok {
		return fmt.Errorf("the page declares no $%s", keys[0])
	}
	for _, k := range keys[1:] {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return fmt.Errorf("%s is a %s, without .%s", t, t.Kind(), k)
		}
		fs, err := ds.Fields(t)
		if err != nil {
			return err
		}
		next := reflect.Type(nil)
		var names []string
		for _, f := range fs {
			names = append(names, f.Key)
			if f.Key == k {
				next = f.Type
			}
		}
		if next == nil {
			sort.Strings(names)
			return fmt.Errorf("no signal %q in %s (%s)", k, t, strings.Join(names, ", "))
		}
		t = next
	}
	return nil
}

// CheckDOM reports the //ds:dom names (dsgen's dsDOM, by Go name) that
// the page or its stylesheet lacks, by the name's kind: an Attr… is an
// attribute of an element of page, a Class… a class of one, a Prop… a
// custom property and an Anim… an animation that css names. The
// controller reads the page by these names; one the templates or the CSS
// dropped is a drag that finds nothing.
func CheckDOM(page, css string, names map[string]string) []error {
	nodes, err := html.ParseFragment(strings.NewReader(page), &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body})
	if err != nil {
		return []error{err}
	}
	attrs, classes := map[string]bool{}, map[string]bool{}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		for _, a := range n.Attr {
			attrs[a.Key] = true
			if a.Key == "class" {
				for _, c := range strings.Fields(a.Val) {
					classes[c] = true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	keys := make([]string, 0, len(names))
	for k := range names {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var errs []error
	for _, name := range keys {
		v := names[name]
		var ok bool
		switch {
		case strings.HasPrefix(name, "Attr"):
			ok = attrs[v]
		case strings.HasPrefix(name, "Class"):
			ok = classes[v]
		case strings.HasPrefix(name, "Prop"):
			ok = regexp.MustCompile(regexp.QuoteMeta(v) + `\b`).MatchString(css)
		case strings.HasPrefix(name, "Anim"):
			ok = regexp.MustCompile(`@keyframes\s+` + regexp.QuoteMeta(v) + `\b`).MatchString(css)
		default:
			errs = append(errs, fmt.Errorf("%s: name a //ds:dom constant Attr…, Class…, Prop… or Anim…", name))
			continue
		}
		if !ok {
			errs = append(errs, fmt.Errorf("%s = %q is not on the page or in its CSS", name, v))
		}
	}
	return errs
}

// inline is a Datastar attribute of a template with a literal value that
// reads a signal or calls an action.
var inline = regexp.MustCompile(`\bdata-([a-z][a-z0-9-]*)(?::[a-z0-9-]+)?(?:__[a-z0-9.-]+)*="([^"]*[$@][^"]*)"`)

// Lint reports the templates in dir that write a Datastar expression
// inline: an expression is built in Go from typed refs, so the compiler
// sees every signal it reads.
func Lint(dir string) ([]error, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.templ"))
	if err != nil {
		return nil, err
	}
	var errs []error
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		for i, line := range bytes.Split(src, []byte("\n")) {
			for _, m := range inline.FindAllSubmatch(line, -1) {
				if _, ok := Plugins[string(m[1])]; ok {
					errs = append(errs, fmt.Errorf("%s:%d: data-%s=%q: build the expression in Go from ds refs, as { expr }", filepath.Base(f), i+1, m[1], m[2]))
				}
			}
		}
	}
	return errs, nil
}
