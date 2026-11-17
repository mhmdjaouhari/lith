package checkboxes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds/dscheck"
)

// stage renders the stage at window w, its tiles with a box checked and
// two tabs watching each, as the GET document draws it.
func stage(t *testing.T, w Window) string {
	t.Helper()
	var tiles []shown
	for range w.End() - w.Start {
		bits := make([]byte, ChunkSize/8)
		bits[0] = 1
		tiles = append(tiles, shown{bits: bits, watching: 2})
	}
	var sb strings.Builder
	if err := Stage(w, tiles, Stats{Checked: 3, Flips: 4, Rate: make([]int, Trend), Here: 2}).Render(context.Background(), &sb); err != nil {
		t.Fatal(err)
	}
	return sb.String()
}

// TestSeam renders the demo's page and checks each Datastar attribute: a
// known plugin in its form, valid JavaScript, and only signals that
// Signals and the shell declare; and that the page carries every name
// checkboxes.ts reads it by.
func TestSeam(t *testing.T) {
	page := stage(t, Window{Start: 0})
	schema := dscheck.Schema{"_checkboxes": reflect.TypeFor[Signals]()}
	for _, err := range dscheck.Check(page, schema, Slug) {
		t.Error(err)
	}
	for _, err := range dscheck.CheckRefs(page) {
		t.Error(err)
	}
	for _, err := range dscheck.CheckDOM(page, css, dsDOM) {
		t.Error(err)
	}
	errs, err := dscheck.Lint(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range errs {
		t.Error(err)
	}
	// A tile's frame, which writeTile writes by hand, reads only the
	// shared names too.
	frame := string(tileFrame(13, make([]byte, ChunkSize/8), 1))
	for _, want := range []string{`id="` + IdTile + `13"`, `class="` + ClassTile + `"`, AttrChunk + `="13"`, `class="` + ClassBoxes + `"`, "#" + IdSlot + "1\n"} {
		if !strings.Contains(frame, want) {
			t.Errorf("a tile's frame has no %s", want)
		}
	}
	if strings.Contains(frame, "data-init") || strings.Contains(frame, "$") {
		t.Errorf("a tile's frame carries an expression:\n%s", frame)
	}
}

// TestCSS checks checkboxes.css against the page it styles: every --cb-
// property it reads is set, and every class, id and data- attribute it
// styles is on the page, with the end of the grid's park.
func TestCSS(t *testing.T) {
	var pages []string
	for _, w := range []Window{{Start: 0}, {Start: Chunks - 1}} {
		pages = append(pages, `<div class="island" data-demo="`+Slug+`">`+stage(t, w)+`</div>`)
	}
	for _, err := range dscheck.CheckCSS(css, "cb", Slug, dsDOM, pages...) {
		t.Error(err)
	}
}

// TestDeclared checks that the stage declares the signals Signals has,
// with their starting values, and none of what the controller keeps.
func TestDeclared(t *testing.T) {
	got := NewSignals().Decl()
	for _, want := range []string{`"_checkboxes":{`, `"op":""`, `"ack":""`, `"synced":false`, `"cols":2`, `"pitch":20`, `"item":354`, `"jump":""`, `"hello":false`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s lacks %s", got, want)
		}
	}
	for _, gone := range []string{`"target"`, `"moved"`, `"inflight"`, `"recheck"`} {
		if strings.Contains(got, gone) {
			t.Errorf("%s declares %s", got, gone)
		}
	}
}

// TestPayload posts each action's payload with a key misspelt, missing,
// extra or null: each is a 400, never a click on box 0.
func TestPayload(t *testing.T) {
	if SetAction.URL() != "/cmd/checkboxes/set" || WindowAction.URL() != "/cmd/checkboxes/window" || SetAction.Event() != "checkboxes-set" {
		t.Fatalf("%s %s %s", SetAction.URL(), WindowAction.URL(), SetAction.Event())
	}
	mux := http.NewServeMux()
	k := demo.NewKit(Slug, mux, demo.Deps{})
	var sets []SetReq
	var windows []WindowReq
	SetAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req SetReq) {
		sets = append(sets, req)
		w.WriteHeader(http.StatusNoContent)
	})
	WindowAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req WindowReq) {
		windows = append(windows, req)
		w.WriteHeader(http.StatusNoContent)
	})
	for _, c := range []struct {
		url, body string
		want      int
	}{
		{SetAction.URL(), `{"tab":"t1","op":"o1","box":5,"checked":true}`, http.StatusNoContent},
		{SetAction.URL(), `{"tab":"t1","op":"o1","bx":5,"checked":true}`, http.StatusBadRequest},
		{SetAction.URL(), `{"tab":"t1","op":"o1","checked":true}`, http.StatusBadRequest},
		{SetAction.URL(), `{"tab":"t1","op":"o1","box":null,"checked":true}`, http.StatusBadRequest},
		{SetAction.URL(), `{"tab":"t1","op":"o1","box":5,"checked":true,"x":1}`, http.StatusBadRequest},
		{WindowAction.URL(), `{"tab":"t1","offset":40,"count":8,"cols":2}`, http.StatusNoContent},
		{WindowAction.URL(), `{"tab":"t1","offset":40,"count":8}`, http.StatusBadRequest},
		{WindowAction.URL(), `{"tab":"t1","offset":"40","count":8,"cols":2}`, http.StatusBadRequest},
	} {
		r := httptest.NewRequest(http.MethodPost, c.url, strings.NewReader(c.body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s %s: %d, want %d", c.url, c.body, w.Code, c.want)
		}
	}
	if len(sets) != 1 || sets[0] != (SetReq{Tab: "t1", Op: "o1", Box: 5, Checked: true}) {
		t.Errorf("the click handler got %+v", sets)
	}
	if len(windows) != 1 || windows[0] != (WindowReq{Tab: "t1", Offset: 40, Count: 8, Cols: 2}) {
		t.Errorf("the window handler got %+v", windows)
	}
}

// TestExpressions pins what the typed builders write for the page.
func TestExpressions(t *testing.T) {
	for got, want := range map[string]string{
		setPosts["data-on:checkboxes-set"].(string):       `@post("/cmd/checkboxes/set", {payload: {...evt.detail, tab: $_tab}, "requestCancellation":"disabled"})`,
		windowPosts["data-on:checkboxes-window"].(string): `@post("/cmd/checkboxes/window", {payload: {...evt.detail, tab: $_tab}})`,
		greetPosts["data-on:checkboxes-window"].(string):  `@post("/cmd/checkboxes/window", {payload: {...evt.detail, tab: $_tab}, "retry":"error","retryMaxCount":5})`,
		pitch: `$_checkboxes.pitch + 'px'`,
	} {
		if got != want {
			t.Errorf("got  %s\nwant %s", got, want)
		}
	}
}
