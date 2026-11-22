package canvas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds/dscheck"
)

// TestSeam renders the demo's page and checks each Datastar attribute: a
// known plugin in its form, valid JavaScript, and only signals that
// Signals and the shell declare; and that the page and canvas.css carry
// every name canvas.ts reads it by.
func TestSeam(t *testing.T) {
	var sb strings.Builder
	p := Page{Cells: strings.Repeat("0", Size*Size), Color: DefaultColor, Me: Tag("tab1")}
	if err := Stage(p).Render(context.Background(), &sb); err != nil {
		t.Fatal(err)
	}
	page := sb.String()
	schema := dscheck.Schema{"_canvas": reflect.TypeFor[Signals]()}
	for _, err := range dscheck.Check(page, schema, Slug) {
		t.Error(err)
	}
	for _, err := range dscheck.CheckRefs(page) {
		t.Error(err)
	}
	for _, err := range dscheck.CheckDOM(page, css, dsDOM) {
		t.Error(err)
	}
	for i := range Colours {
		if prop := PropColour + strconv.Itoa(i) + ":"; !strings.Contains(css, prop) {
			t.Errorf("canvas.css sets no %s", prop)
		}
	}
	errs, err := dscheck.Lint(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range errs {
		t.Error(err)
	}
}

// TestPayloads posts what sb-pixel-board's sb-paint and the palette send,
// and the same with a key misspelt, missing or extra: each of those is a
// 400.
func TestPayloads(t *testing.T) {
	mux := http.NewServeMux()
	k := demo.NewKit(Slug, mux, demo.Deps{})
	var got []any
	PaintAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req PaintReq) {
		got = append(got, req)
		w.WriteHeader(http.StatusNoContent)
	})
	PickAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req PickReq) {
		got = append(got, req)
		w.WriteHeader(http.StatusNoContent)
	})
	for _, c := range []struct {
		url, body string
		want      int
	}{
		{PaintAction.URL(), `{"color":3,"cells":[1,2],"tab":"t1"}`, http.StatusNoContent},
		{PaintAction.URL(), `{"colour":3,"cells":[1,2],"tab":"t1"}`, http.StatusBadRequest},
		{PaintAction.URL(), `{"color":3,"tab":"t1"}`, http.StatusBadRequest},
		{PickAction.URL(), `{"color":3,"tab":"t1"}`, http.StatusNoContent},
		{PickAction.URL(), `{"color":3,"tab":"t1","op":"o1"}`, http.StatusBadRequest},
	} {
		r := httptest.NewRequest(http.MethodPost, c.url, strings.NewReader(c.body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s %s: %d, want %d", c.url, c.body, w.Code, c.want)
		}
	}
	want := []any{PaintReq{Tab: "t1", Color: 3, Cells: []int{1, 2}}, PickReq{Tab: "t1", Color: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the handlers got %+v", got)
	}
}

// TestExpressions pins what the typed builders write for the page.
func TestExpressions(t *testing.T) {
	for _, c := range []struct{ got, want string }{
		{paint["data-on:sb-paint"].(string), `@post("/cmd/canvas/paint", {payload: {...evt.detail, tab: $_tab}, "requestCancellation":"disabled"})`},
		{string(pick), `@post("/cmd/canvas/pick", {payload: {"color": $_canvas.color, tab: $_tab}})`},
		{string(choose(4)), `$_canvas.color = 4, $_canvas.problem = ""`},
		{string(refused), `(evt.detail.el === el && evt.detail.type === 'retries-failed') && ($_canvas.problem = "Some pixels did not reach the canvas: the connection is down. They fade back; paint them again once it is back.")`},
	} {
		if c.got != c.want {
			t.Errorf("got  %s\nwant %s", c.got, c.want)
		}
	}
}
