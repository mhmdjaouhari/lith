package cursorparty

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

// TestSeam renders the party's page and its hero and checks each
// Datastar attribute: a known plugin in its form, valid JavaScript, and
// only signals that Signals and the shell declare; and that the page
// carries every name cursor-party.ts reads it by.
func TestSeam(t *testing.T) {
	var sb strings.Builder
	history := make([]int, History)
	if err := Stage("", nil, 1, history).Render(context.Background(), &sb); err != nil {
		t.Fatal(err)
	}
	if err := Hero("", nil, 1, history).Render(context.Background(), &sb); err != nil {
		t.Fatal(err)
	}
	page := sb.String()
	schema := dscheck.Schema{"_cursorParty": reflect.TypeFor[Signals]()}
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

// TestPayload posts moves as the controller sends them, without an
// anchor as the floor's own, and with a key misspelt, missing or extra:
// each of those is a 400. A tab without a stream moves too.
func TestPayload(t *testing.T) {
	mux := http.NewServeMux()
	k := demo.NewKit(Slug, mux, demo.Deps{})
	var got []MoveReq
	MoveAction.MountOpen(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req MoveReq) {
		got = append(got, req)
		w.WriteHeader(http.StatusNoContent)
	})
	for _, c := range []struct {
		body string
		want int
	}{
		{`{"a":"L","x":0.5,"y":-1.25,"seq":3,"tab":"t1"}`, http.StatusNoContent},
		{`{"x":0.5,"y":0.25,"seq":4,"tab":"t1"}`, http.StatusNoContent},
		{`{"a":"L","x":0.5,"y":0.25,"sequence":3,"tab":"t1"}`, http.StatusBadRequest},
		{`{"a":"L","x":0.5,"seq":3,"tab":"t1"}`, http.StatusBadRequest},
		{`{"a":"L","x":0.5,"y":0.25,"seq":3,"tab":"t1","z":1}`, http.StatusBadRequest},
	} {
		r := httptest.NewRequest(http.MethodPost, MoveAction.URL(), strings.NewReader(c.body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s: %d, want %d", c.body, w.Code, c.want)
		}
	}
	want := []MoveReq{{Tab: "t1", A: "L", X: 0.5, Y: -1.25, Seq: 3}, {Tab: "t1", X: 0.5, Y: 0.25, Seq: 4}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the handler got %+v", got)
	}
}

// TestExpressions pins the floor's post: side by side, never retried, and
// a failure dropped.
func TestExpressions(t *testing.T) {
	got, _ := floor()["data-on:cursor-party-move"].(string)
	if want := `@post("/cmd/cursor-party/move", {payload: {...evt.detail, tab: $_tab}, "requestCancellation":"disabled","retry":"never"}).catch(() => {})`; got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
