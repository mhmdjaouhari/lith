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
	"errors"
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

// markup is what a page's elements carry: attribute names, classes, ids,
// and the id references of its id-reference attributes.
type markup struct {
	attrs, classes, ids map[string]bool
	refs                []string // "attr=id"
}

// idRefs are the attributes whose value names ids on the page.
var idRefs = map[string]bool{"aria-labelledby": true, "aria-describedby": true, "aria-controls": true, "aria-owns": true, "aria-activedescendant": true, "aria-errormessage": true, "for": true, "popovertarget": true, "commandfor": true, "list": true, "form": true}

func parse(page string) (markup, error) {
	m := markup{attrs: map[string]bool{}, classes: map[string]bool{}, ids: map[string]bool{}}
	nodes, err := html.ParseFragment(strings.NewReader(page), &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body})
	if err != nil {
		return m, err
	}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		for _, a := range n.Attr {
			m.attrs[a.Key] = true
			switch {
			case a.Key == "class":
				for _, c := range strings.Fields(a.Val) {
					m.classes[c] = true
				}
			case a.Key == "id":
				m.ids[a.Val] = true
			case idRefs[a.Key]:
				for _, id := range strings.Fields(a.Val) {
					m.refs = append(m.refs, a.Key+"="+id)
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
	return m, nil
}

// CheckRefs reports the id references of page (aria-labelledby, for and
// the like) that name no element of page: a label that labels nothing.
func CheckRefs(page string) []error {
	m, err := parse(page)
	if err != nil {
		return []error{err}
	}
	var errs []error
	for _, r := range m.refs {
		attr, id, _ := strings.Cut(r, "=")
		if !m.ids[id] {
			errs = append(errs, fmt.Errorf("%s names %q, which no element of the page is", attr, id))
		}
	}
	return errs
}

// CheckDOM reports the //ds:dom names (dsgen's dsDOM, by Go name) that
// the page or its stylesheet lacks, by the name's kind: an Attr… is an
// attribute of an element of page, a Class… a class of one, an Id… the id
// of one (an id prefix when it ends in -, which some id begins with), a
// Prop… a custom property and an Anim… an animation that css names. The
// controller reads the page by these names and the stream finds its
// regions by their ids; one the templates or the CSS dropped is a drag
// that finds nothing or a region no frame reaches.
func CheckDOM(page, css string, names map[string]string) []error {
	m, err := parse(page)
	if err != nil {
		return []error{err}
	}
	attrs, classes := m.attrs, m.classes
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
		case strings.HasPrefix(name, "Id") && strings.HasSuffix(v, "-"):
			for id := range m.ids {
				ok = ok || strings.HasPrefix(id, v)
			}
		case strings.HasPrefix(name, "Id"):
			ok = m.ids[v]
		case strings.HasPrefix(name, "Prop"):
			ok = regexp.MustCompile(regexp.QuoteMeta(v) + `\b`).MatchString(css)
		case strings.HasPrefix(name, "Anim"):
			ok = regexp.MustCompile(`@keyframes\s+` + regexp.QuoteMeta(v) + `\b`).MatchString(css)
		default:
			errs = append(errs, fmt.Errorf("%s: name a //ds:dom constant Attr…, Class…, Id…, Prop… or Anim…", name))
			continue
		}
		if !ok {
			errs = append(errs, fmt.Errorf("%s = %q is not on the page or in its CSS", name, v))
		}
	}
	return errs
}

var (
	cssComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssRead    = regexp.MustCompile(`var\(\s*(--[a-zA-Z0-9_-]+)`)
	cssSet     = regexp.MustCompile(`(--[a-zA-Z0-9_-]+)\s*:`)
	cssClass   = regexp.MustCompile(`\.([a-zA-Z_][a-zA-Z0-9_-]*)`)
	cssID      = regexp.MustCompile(`#([a-zA-Z][a-zA-Z0-9_-]*)`)
	cssAttr    = regexp.MustCompile(`\[\s*(data-[a-z0-9-]+)`)
)

// CheckCSS reports what a demo's stylesheet names of its own that nothing
// gives it, for the names that begin with prefix ("bd": the custom
// properties --bd-…, the classes bd, bd-… and bd__…) and with slug + "-"
// (the ids board-…): a custom property it reads that neither css, a
// //ds:dom Prop… (names, which a controller or the server sets) nor an
// inline style of pages sets; a class, an id or a data- attribute it styles
// that no element of pages has. Each is a rule that never applies or a
// value that falls back to its default without an error. pages should show
// every state the stylesheet styles.
func CheckCSS(css, prefix, slug string, names map[string]string, pages ...string) []error {
	css = cssComment.ReplaceAllString(css, "")
	var page markup
	for i, p := range pages {
		m, err := parse(p)
		if err != nil {
			return []error{err}
		}
		if i == 0 {
			page = m
			continue
		}
		for k := range m.attrs {
			page.attrs[k] = true
		}
		for k := range m.classes {
			page.classes[k] = true
		}
		for k := range m.ids {
			page.ids[k] = true
		}
	}
	set := map[string]bool{}
	for _, m := range cssSet.FindAllStringSubmatch(css, -1) {
		set[m[1]] = true
	}
	for name, v := range names {
		if strings.HasPrefix(name, "Prop") {
			set[v] = true
		}
	}
	for _, p := range pages {
		for _, m := range cssSet.FindAllStringSubmatch(styleAttrs(p), -1) {
			set[m[1]] = true
		}
	}
	own := func(name string) bool {
		return name == prefix || strings.HasPrefix(name, prefix+"-") || strings.HasPrefix(name, prefix+"__")
	}
	var errs []error
	seen := map[string]bool{}
	report := func(key, msg string) {
		if !seen[key] {
			seen[key] = true
			errs = append(errs, errors.New(msg))
		}
	}
	for _, m := range cssRead.FindAllStringSubmatch(css, -1) {
		if v := m[1]; strings.HasPrefix(v, "--"+prefix+"-") && !set[v] {
			report(v, fmt.Sprintf("the CSS reads %s, which nothing sets", v))
		}
	}
	// Selectors only: the text before each block's brace.
	for _, sel := range selectors(css) {
		for _, m := range cssClass.FindAllStringSubmatch(sel, -1) {
			if c := m[1]; own(c) && !page.classes[c] {
				report("."+c, fmt.Sprintf("the CSS styles .%s, which no element has", c))
			}
		}
		for _, m := range cssID.FindAllStringSubmatch(sel, -1) {
			if id := m[1]; strings.HasPrefix(id, slug+"-") && !page.ids[id] {
				report("#"+id, fmt.Sprintf("the CSS styles #%s, which no element is", id))
			}
		}
		for _, m := range cssAttr.FindAllStringSubmatch(sel, -1) {
			if a := m[1]; !page.attrs[a] {
				report("["+a, fmt.Sprintf("the CSS styles [%s], which no element has", a))
			}
		}
	}
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errs
}

// selectors returns the selectors of css: the text before each {, but the
// at-rules' preludes (@container, @media, @keyframes...) and keyframe
// selectors (from, 50%).
func selectors(css string) []string {
	var out []string
	for _, part := range strings.Split(css, "{") {
		if i := strings.LastIndexAny(part, ";}"); i >= 0 {
			part = part[i+1:]
		}
		part = strings.TrimSpace(part)
		if part == "" || strings.HasPrefix(part, "@") || part == "from" || part == "to" || strings.HasSuffix(part, "%") {
			continue
		}
		out = append(out, part)
	}
	return out
}

var styleAttr = regexp.MustCompile(`style="([^"]*)"`)

// styleAttrs returns the inline styles of page, and its <style> elements'
// text, joined.
func styleAttrs(page string) string {
	var b strings.Builder
	for _, m := range styleAttr.FindAllStringSubmatch(page, -1) {
		b.WriteString(m[1] + ";")
	}
	for _, s := range regexp.MustCompile(`(?s)<style[^>]*>(.*?)</style>`).FindAllStringSubmatch(page, -1) {
		b.WriteString(s[1])
	}
	return b.String()
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
