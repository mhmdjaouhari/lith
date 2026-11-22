package town

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/demo/demotest"
	"github.com/zweiundeins/lith/site/internal/ds"
	"github.com/zweiundeins/lith/site/internal/ds/dscheck"
)

// TestSeam renders the demo's page on the seeded town and checks each
// Datastar attribute: a known plugin in its form, valid JavaScript, and
// only signals that Signals and the shell declare; and that the page and
// town.css carry every name town.ts reads it by.
func TestSeam(t *testing.T) {
	e := demotest.Open(t, Demo)
	e.Do(t, Seed(time.Now()))
	var sb strings.Builder
	e.View(t, func(ctx context.Context, tx *sql.Tx) error {
		page, err := renderPage(ctx, tx, ada)
		if err != nil {
			return err
		}
		return page.Render(ctx, &sb)
	})
	page := sb.String()
	schema := dscheck.Schema{"_town": reflect.TypeFor[Signals]()}
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
}

// TestDeclared checks that the stage declares what both demo.Pending
// read, and the slider at its end.
func TestDeclared(t *testing.T) {
	got := NewSignals(Bounds{First: 2, Head: 9}).Decl()
	for _, want := range []string{`"_town":{`, `"op":""`, `"ack":""`, `"synced":false`, `"problem":""`, `"inflight":0`, `"recheck":false`, `"tool":"house"`, `"ptile":-1`, `"want":9`, `"first":2`, `"head":9`, `"seeks":0`, `"seekRecheck":false`, `"hello":false`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s lacks %s", got, want)
		}
	}
}

// TestPayloads posts each action with a key misspelt, missing or extra:
// each is a 400.
func TestPayloads(t *testing.T) {
	mux := http.NewServeMux()
	k := demo.NewKit(Slug, mux, demo.Deps{})
	var got []any
	PlaceAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req PlaceReq) {
		got = append(got, req)
		w.WriteHeader(http.StatusNoContent)
	})
	SeekAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req SeekReq) {
		got = append(got, req)
		w.WriteHeader(http.StatusNoContent)
	})
	FollowAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req FollowReq) {
		got = append(got, req)
		w.WriteHeader(http.StatusNoContent)
	})
	for _, c := range []struct {
		url, body string
		want      int
	}{
		{PlaceAction.URL(), `{"tab":"t1","op":"o1","tile":3,"kind":"tree","seen":2}`, http.StatusNoContent},
		{PlaceAction.URL(), `{"tab":"t1","op":"o1","tile":3,"kind":"tree"}`, http.StatusBadRequest},
		{SeekAction.URL(), `{"tab":"t1","at":0,"seq":4}`, http.StatusNoContent},
		{SeekAction.URL(), `{"tab":"t1","at":0,"sequence":4}`, http.StatusBadRequest},
		{FollowAction.URL(), `{"tab":"t1"}`, http.StatusNoContent},
		{FollowAction.URL(), `{"tab":"t1","op":"o1"}`, http.StatusBadRequest},
	} {
		r := httptest.NewRequest(http.MethodPost, c.url, strings.NewReader(c.body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s %s: %d, want %d", c.url, c.body, w.Code, c.want)
		}
	}
	want := []any{PlaceReq{Tab: "t1", Op: "o1", Tile: 3, Kind: "tree", Seen: 2}, SeekReq{Tab: "t1", Seq: 4}, FollowReq{Tab: "t1"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the handlers got %+v", got)
	}
}

// TestExpressions pins what the typed builders write for the page.
func TestExpressions(t *testing.T) {
	for _, c := range []struct{ got, want ds.Expr }{
		{greet, `$_town.hello && ($_town.hello = false, @post("/cmd/town/follow", {payload: {tab: $_tab}, "retry":"error","retryMaxCount":5}))`},
		{badge, `$_badge = ($_town.at !== 0) ? "past" : ""`},
		{posLabel, `($_town.want >= $_town.head) ? "Now" : ("Position " + (($_town.want).toLocaleString('en')))`},
	} {
		if c.got != c.want {
			t.Errorf("got  %s\nwant %s", c.got, c.want)
		}
	}
}
