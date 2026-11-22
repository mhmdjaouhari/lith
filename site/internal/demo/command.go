package demo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/a-h/templ"
	"github.com/zweiundeins/lith/sse"

	"github.com/zweiundeins/lith/site/internal/ds"
)

//go:generate go run github.com/zweiundeins/lith/site/cmd/dsgen

// The page's own signals, which the shell declares (views.pageSignals).
var (
	// Tab is the page's tab id, which every action sends.
	Tab = ds.NewRef[string]("_tab")
	// Live is true while the page's stream is open.
	Live = ds.NewRef[bool]("_live")
	// Badge is the word the stage's badge says instead of live, such as
	// past for a stage that shows the past; "" is live again.
	Badge = ds.NewRef[string]("_badge")
)

// AckOf is the Ack of typed assignments to the demo's own signals:
// k.AckOf(S.Ack.Set(op), S.OK.Set(false)). It panics on a signal of
// another demo's, which Notify would put under this one's.
func (k *Kit) AckOf(as ...ds.Assign) Ack {
	tree := ds.Tree(as...)
	own := signalName(k.slug)
	for root := range tree {
		if root != own {
			panic(fmt.Sprintf("demo %s: an ack sets $%s, not the demo's $%s", k.slug, root, own))
		}
	}
	fields, _ := tree[own].(map[string]any)
	return fields
}

// Payload is the request of an action: a struct with a //ds:command
// directive, for which dsgen writes DSCommand, and a `json:"tab"` field.
type Payload interface {
	DSCommand() (slug, name string)
}

// Command is the action whose payload is Req, POST /cmd/<slug>/<name>
// as Req's //ds:command names it. One type holds the action's contract
// for both sides: the server decodes exactly Req (Mount), the page posts
// what the controller sends in the DOM event Event, whose detail tsc
// checks against the TypeScript interface dsgen writes from Req.
type Command[Req Payload] struct {
	Rate Rate
}

func (Command[Req]) names() (slug, name string) {
	var r Req
	return r.DSCommand()
}

// URL is the action's path: /cmd/<slug>/<name>.
func (c Command[Req]) URL() string {
	slug, name := c.names()
	return "/cmd/" + slug + "/" + name
}

// Event is the DOM event a page's controller dispatches to send the
// action: <slug>-<name>.
func (c Command[Req]) Event() string {
	slug, name := c.names()
	return slug + "-" + name
}

// Post are the options of Datastar's @post that an action uses.
type Post struct {
	// RequestCancellation 'disabled' keeps a second request to the same
	// URL from cancelling the first: an op's requests must all arrive.
	RequestCancellation string `json:"requestCancellation,omitempty"`
	Retry               string `json:"retry,omitempty"`
	RetryMaxCount       int    `json:"retryMaxCount,omitempty"`
	// Quiet drops what a failed request rejects with: for an action whose
	// person learns nothing from a failure (a cursor's move), so that the
	// console stays empty.
	Quiet bool `json:"-"`
}

func (c Command[Req]) post(payload string, o Post) ds.Expr {
	opts := strings.TrimPrefix(ds.Lit(o), "{")
	if opts != "}" {
		opts = ", " + opts
	}
	post := "@post(" + ds.Lit(c.URL()) + ", {payload: " + payload + opts + ")"
	if o.Quiet {
		post += ".catch(() => {})"
	}
	return ds.Expr(post)
}

// On is the data-on attribute that posts the action whenever the
// element hears its Event: the event's detail, which the controller
// typed, and the tab id. Spread it on the element that listens:
// <div { cmd.On(post)... }>.
func (c Command[Req]) On(o Post) templ.Attributes {
	return c.OnEvent(c.Event(), o)
}

// OnEvent is On for the event a component dispatches, such as
// sb-pixel-board's sb-paint, whose detail carries Req's keys but the
// tab. tsc does not see a component's detail: the payload's test posts
// what the component sends.
func (c Command[Req]) OnEvent(event string, o Post) templ.Attributes {
	return templ.Attributes{"data-on:" + event: string(c.post("{...evt.detail, tab: "+Tab.String()+"}", o))}
}

// PostTab is the expression that posts the action with the tab id
// alone. It panics when Req carries more than the tab.
func (c Command[Req]) PostTab(o Post) ds.Expr {
	fs := mustFields[Req]()
	if len(fs) != 1 {
		panic(fmt.Sprintf("demo: %s carries more than the tab id: post it with On", c.URL()))
	}
	return c.post("{tab: "+Tab.String()+"}", o)
}

// Arg is a key of an action's payload and the signal that fills it, for
// PostSignals.
type Arg struct {
	key, expr string
	typ       reflect.Type
}

// From fills the payload's key with the signal r.
func From[T any](key string, r ds.Ref[T]) Arg {
	return Arg{key: key, expr: r.String(), typ: reflect.TypeFor[T]()}
}

// Value fills the payload's key with v, the same for every post: the
// stream demo's size button sends its size.
func Value[T any](key string, v T) Arg {
	return Arg{key: key, expr: ds.Lit(v), typ: reflect.TypeFor[T]()}
}

// PostSignals is the expression that posts the action with its payload
// read from the page's signals (From) or fixed (Value), and the tab id:
// what a form without a controller sends (the guestbook's name and
// message). It panics unless args fill every key of Req but the tab and
// those tagged omitempty, each once and with a value of the field's type,
// and no other key: called as a package's expressions are built, so a
// payload the server would refuse fails the package's tests, not a
// person's click.
func (c Command[Req]) PostSignals(o Post, args ...Arg) ds.Expr {
	want := map[string]reflect.Type{}
	optional := map[string]bool{}
	for _, f := range mustFields[Req]() {
		if f.Key != "tab" {
			want[f.Key] = f.Type
			optional[f.Key] = f.Optional
		}
	}
	parts := make([]string, 0, len(args)+1)
	for _, a := range args {
		t, ok := want[a.key]
		switch {
		case !ok:
			panic(fmt.Sprintf("demo: %s has no key %q, or it is filled twice", c.URL(), a.key))
		case t != a.typ:
			panic(fmt.Sprintf("demo: %s's %q is a %s, not the %s of %s", c.URL(), a.key, t, a.typ, a.expr))
		}
		delete(want, a.key)
		parts = append(parts, ds.Lit(a.key)+": "+a.expr)
	}
	for key := range want {
		if !optional[key] {
			panic(fmt.Sprintf("demo: %s needs %q", c.URL(), key))
		}
	}
	return c.post("{"+strings.Join(append(parts, "tab: "+Tab.String()), ", ")+"}", o)
}

// Mount mounts the action on the kit (Kit.Action) with a handler that
// gets the decoded payload and its viewer. The payload must carry every
// field of Req but those tagged omitempty, none other, and no null: a
// misspelt key in a page is a 400, never a zero value that a command
// takes for an answer (lane 0 is a lane). A request whose tab is not the
// session's open tab is a 400 too (Who).
func (c Command[Req]) Mount(k *Kit, h func(w http.ResponseWriter, r *http.Request, v Viewer, req Req)) {
	c.mount(k, true, h)
}

// MountOpen is Mount for an action whose tab need not have its stream
// open: the cursor party's moves, which a page sends from its first
// pointer move, and which tell their person nothing. The handler checks
// the viewer itself.
func (c Command[Req]) MountOpen(k *Kit, h func(w http.ResponseWriter, r *http.Request, v Viewer, req Req)) {
	c.mount(k, false, h)
}

func (c Command[Req]) mount(k *Kit, open bool, h func(w http.ResponseWriter, r *http.Request, v Viewer, req Req)) {
	slug, name := c.names()
	if slug != k.slug {
		panic(fmt.Sprintf("demo %s: command %s is the payload of demo %s", k.slug, name, slug))
	}
	fields := mustFields[Req]()
	k.Action(name, c.Rate, func(w http.ResponseWriter, r *http.Request) {
		req, ok := DecodeStrict[Req](w, r, fields)
		if !ok {
			return
		}
		tab := reflect.ValueOf(req).FieldByIndex(tabIndex[Req]()).String()
		v, ok := Who(r, tab)
		if !ok && open {
			w.WriteHeader(http.StatusBadRequest) // no page of ours sends it
			return
		}
		h(w, r, v, req)
	})
}

// DecodeStrict reads an action's payload into a Req: a JSON object of at
// most 8 KiB with every field in fields that is not optional, no other
// key, and no null. When it cannot, it answers a bare 400 itself.
func DecodeStrict[Req any](w http.ResponseWriter, r *http.Request, fields []ds.Field) (Req, bool) {
	var req Req
	var body json.RawMessage
	if r.ContentLength == 0 || sse.ReadSignalsLimit(w, r, &body, 8<<10) != nil || checkKeys(body, fields) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return req, false
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(&req) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return req, false
	}
	return req, true
}

// checkKeys reports a key of body that fields lack, a field body lacks
// that is not optional, or a null.
func checkKeys(body []byte, fields []ds.Field) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return err
	}
	known := map[string]bool{}
	for _, f := range fields {
		known[f.Key] = true
		v, ok := m[f.Key]
		switch {
		case !ok && !f.Optional:
			return fmt.Errorf("no %q", f.Key)
		case ok && string(bytes.TrimSpace(v)) == "null":
			return fmt.Errorf("%q is null", f.Key)
		}
	}
	for k := range m {
		if !known[k] {
			return fmt.Errorf("unknown key %q", k)
		}
	}
	return nil
}

func mustFields[Req any]() []ds.Field {
	fs, err := ds.Fields(reflect.TypeFor[Req]())
	if err != nil {
		panic(err)
	}
	return fs
}

// tabIndex is the index of Req's tab field, which dsgen checked.
func tabIndex[Req any]() []int {
	t := reflect.TypeFor[Req]()
	var find func(t reflect.Type) []int
	find = func(t reflect.Type) []int {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "tab" && f.Type.Kind() == reflect.String {
				return []int{i}
			}
			if f.Anonymous && name == "" && f.Type.Kind() == reflect.Struct {
				if sub := find(f.Type); sub != nil {
					return append([]int{i}, sub...)
				}
			}
		}
		return nil
	}
	if idx := find(t); idx != nil {
		return idx
	}
	panic(fmt.Sprintf("demo: %s has no `json:\"tab\"` string", t))
}
