package life

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds/dscheck"
)

// stage renders the stage as the GET document draws it, for a session
// that draws in colour 3, with a few live cells of every colour and a
// trail.
func stage(t *testing.T) string {
	t.Helper()
	b := Board{Cells: make([]byte, Cells), Trail: make([]byte, Cells), History: "[1,2,3]"}
	for c := byte(1); c <= Wild; c++ {
		b.Cells[int(c)*W+int(c)] = c
		b.Trail[int(c)*W+int(c)+1] = 1<<4 | c
	}
	var sb strings.Builder
	if err := Stage(b, 2, 3, time.Now()).Render(context.Background(), &sb); err != nil {
		t.Fatal(err)
	}
	return sb.String()
}

// TestSeam renders the demo's page and checks each Datastar attribute: a
// known plugin in its form, valid JavaScript, and only signals that
// Signals and the shell declare; and that the page carries every name
// life.ts reads it by.
func TestSeam(t *testing.T) {
	page := stage(t)
	schema := dscheck.Schema{"_life": reflect.TypeFor[Signals]()}
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
	// A frame, which boardCells writes by hand, carries the rows by the
	// shared prefix, and live cells by their classes.
	var sb strings.Builder
	if err := boardCells(make([]byte, Cells), nil).Render(context.Background(), &sb); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`id="` + IdBoard + `"`, `id="` + IdRow + `0"`, `id="` + IdRow + `49"`} {
		if !strings.Contains(sb.String(), want) {
			t.Errorf("the board's frame has no %s", want)
		}
	}
	if cellClass[0] != '_' || len(CellClasses) != Colours+2 {
		t.Errorf("CellClasses %q is not a dead cell, the colours and the soup's", CellClasses)
	}
}

// TestCSS checks life.css against the page it styles: every --life-
// property it reads is set, and every class, id and data- attribute it
// styles is on the page, with a refusal, and an outline named by its tool
// as data-attr sets it.
func TestCSS(t *testing.T) {
	var refused strings.Builder
	if err := Outcome(TabState{Refusal: "Someone reset the board a moment ago."}).Render(context.Background(), &refused); err != nil {
		t.Fatal(err)
	}
	page := stage(t) + refused.String() + `<div class="` + ClassCursor + `" data-tool="glider"></div>`
	page = `<div class="island" data-demo="` + Slug + `">` + page + `</div>`
	for _, err := range dscheck.CheckCSS(css, "life", Slug, dsDOM, page) {
		t.Error(err)
	}
}

// TestDeclared checks that the stage declares the signals Signals has,
// with their starting values, and none of what the controller keeps.
func TestDeclared(t *testing.T) {
	got := NewSignals(3).Decl()
	for _, want := range []string{`"_life":{`, `"op":""`, `"ack":""`, `"synced":false`, `"tool":"glider"`, `"x":-1`, `"y":-1`, `"alive":true`, `"px":-1`, `"colour":3`, `"pick":3`, `"resetting":false`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s lacks %s", got, want)
		}
	}
	for _, gone := range []string{`"drawing"`, `"stroke"`, `"last"`, `"sent"`, `"inflight"`, `"recheck"`} {
		if strings.Contains(got, gone) {
			t.Errorf("%s declares %s", got, gone)
		}
	}
}

// TestPayload posts each action's payload with a key misspelt, missing,
// extra or null: each is a 400, never a stroke at cell 0.
func TestPayload(t *testing.T) {
	mux := http.NewServeMux()
	k := demo.NewKit(Slug, mux, demo.Deps{})
	var got []any
	ok := func(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }
	DrawAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req DrawReq) {
		got = append(got, req)
		ok(w)
	})
	DropAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req DropReq) {
		got = append(got, req)
		ok(w)
	})
	ResetAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req ResetReq) {
		got = append(got, req)
		ok(w)
	})
	ColourAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req ColourReq) {
		got = append(got, req)
		ok(w)
	})
	for _, c := range []struct {
		url, body string
		want      int
	}{
		{DrawAction.URL(), `{"tab":"t1","op":"o1","cells":[5,6],"alive":true}`, http.StatusNoContent},
		{DrawAction.URL(), `{"tab":"t1","op":"o1","cells":"5,6","alive":true}`, http.StatusBadRequest},
		{DrawAction.URL(), `{"tab":"t1","op":"o1","cells":null,"alive":true}`, http.StatusBadRequest},
		{DrawAction.URL(), `{"tab":"t1","op":"o1","cells":[5]}`, http.StatusBadRequest},
		{DropAction.URL(), `{"tab":"t1","op":"o1","pattern":"glider","cell":7,"turn":1}`, http.StatusNoContent},
		{DropAction.URL(), `{"tab":"t1","op":"o1","pattern":"glider","cell":7,"turn":1,"x":1}`, http.StatusBadRequest},
		{DropAction.URL(), `{"tab":"t1","op":"o1","pattern":"glider","cel":7,"turn":1}`, http.StatusBadRequest},
		{ResetAction.URL(), `{"tab":"t1","op":"o1"}`, http.StatusNoContent},
		{ResetAction.URL(), `{"tab":"t1"}`, http.StatusBadRequest},
		{ColourAction.URL(), `{"tab":"t1","colour":4}`, http.StatusNoContent},
		{ColourAction.URL(), `{"tab":"t1","color":4}`, http.StatusBadRequest},
	} {
		r := httptest.NewRequest(http.MethodPost, c.url, strings.NewReader(c.body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s %s: %d, want %d", c.url, c.body, w.Code, c.want)
		}
	}
	want := []any{
		DrawReq{Tab: "t1", Op: "o1", Cells: []int{5, 6}, Alive: true},
		DropReq{Tab: "t1", Op: "o1", Pattern: "glider", Cell: 7, Turn: 1},
		ResetReq{Tab: "t1", Op: "o1"},
		ColourReq{Tab: "t1", Colour: 4},
	}
	if !slices.EqualFunc(got, want, func(a, b any) bool { return reflect.DeepEqual(a, b) }) {
		t.Errorf("the handlers got %+v", got)
	}
}

// TestExpressions pins what the typed builders write for the page.
func TestExpressions(t *testing.T) {
	for _, c := range []struct{ got, want string }{
		{posts["data-on:life-draw"].(string), `@post("/cmd/life/draw", {payload: {...evt.detail, tab: $_tab}, "requestCancellation":"disabled"})`},
		{string(toolPressed("pulsar")), `($_life.tool === "pulsar") ? 'true' : 'false'`},
		{string(begin), `($_life.tool === "draw") && @lifeBegin()`},
		{string(picking(2)), `(($_life.pick === 2) && ($_life.colour !== 2)) ? 'true' : false`},
		{string(pick(2)), `$_life.pick = 2, $_life.problem = "", @post("/cmd/life/colour", {payload: {"colour": $_life.pick, tab: $_tab}, "requestCancellation":"disabled"})`},
		{string(reset), `!$_life.resetting && ($_life.problem = "", $_life.op = @orderedOpId(), $_life.resetting = true, @post("/cmd/life/reset", {payload: {"op": $_life.op, tab: $_tab}, "requestCancellation":"disabled"}))`},
		{string(settle), `(($_life.op !== "") && ($_life.ack >= $_life.op)) && ($_life.op = "", $_life.pend = "", $_life.ptool = "", $_life.resetting = false)`},
	} {
		if c.got != c.want {
			t.Errorf("got  %s\nwant %s", c.got, c.want)
		}
	}
	if len(posts) != 2 {
		t.Errorf("the stage posts %d actions, want 2", len(posts))
	}
}
