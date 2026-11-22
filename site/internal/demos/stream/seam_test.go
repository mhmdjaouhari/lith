package stream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds"
	"github.com/zweiundeins/lith/site/internal/ds/dscheck"
)

// TestSeam renders the demo's page and its landing slot and checks each
// Datastar attribute: a known plugin in its form, valid JavaScript, and
// only signals that Signals and the shell declare.
func TestSeam(t *testing.T) {
	var sb strings.Builder
	st := TabState{}
	if err := Stage(st, payloadFor(st, 1)).Render(context.Background(), &sb); err != nil {
		t.Fatal(err)
	}
	if err := Embed().Render(context.Background(), &sb); err != nil {
		t.Fatal(err)
	}
	page := sb.String()
	schema := dscheck.Schema{"_stream": reflect.TypeFor[Signals]()}
	for _, err := range dscheck.Check(page, schema) {
		t.Error(err)
	}
	for _, err := range dscheck.CheckRefs(page) {
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

// TestDeclared checks that the page declares what demo.Pending reads, the
// reports' signals and the tab's choices.
func TestDeclared(t *testing.T) {
	got := NewSignals(TabState{Size: Small, Churn: Still}).Decl()
	for _, want := range []string{`"_stream":{`, `"n":0`, `"fwt":""`, `"budget":-1`, `"fps":"0"`, `"quiet":0`, `"op":""`, `"ack":""`, `"synced":false`, `"inflight":0`, `"recheck":false`, `"problem":""`, `"size":"small"`, `"churn":"still"`, `"want":""`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s lacks %s", got, want)
		}
	}
}

// TestPayloadKeys posts changes with a key misspelt, missing, extra or
// null: each is a 400, a size or a churn alone a 204.
func TestPayloadKeys(t *testing.T) {
	mux := http.NewServeMux()
	k := demo.NewKit(Slug, mux, demo.Deps{})
	var got []PayloadReq
	PayloadAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req PayloadReq) {
		got = append(got, req)
		w.WriteHeader(http.StatusNoContent)
	})
	for _, c := range []struct {
		body string
		want int
	}{
		{`{"tab":"t1","op":"o1","size":"large"}`, http.StatusNoContent},
		{`{"tab":"t1","op":"o1","churn":"storm"}`, http.StatusNoContent},
		{`{"tab":"t1","op":"o1","sise":"large"}`, http.StatusBadRequest},
		{`{"tab":"t1","op":"o1","size":null}`, http.StatusBadRequest},
		{`{"tab":"t1","size":"large"}`, http.StatusBadRequest},
		{`{"tab":"t1","op":"o1","size":"large","x":1}`, http.StatusBadRequest},
	} {
		r := httptest.NewRequest(http.MethodPost, PayloadAction.URL(), strings.NewReader(c.body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s: %d, want %d", c.body, w.Code, c.want)
		}
	}
	want := []PayloadReq{{Tab: "t1", Op: "o1", Size: Large}, {Tab: "t1", Op: "o1", Churn: Storm}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the handler got %+v", got)
	}
}

// TestExpressions pins what the typed builders write for the page.
func TestExpressions(t *testing.T) {
	large := choice("size", S.Size, Large, "Large", Small)
	for _, c := range []struct{ got, want ds.Expr }{
		{large.choose, `(($_stream.op === "") && ($_stream.size !== "large")) && ($_stream.problem = "", ` +
			`$_stream.op = Date.now().toString(36) + Math.random().toString(36).slice(2, 7), $_stream.want = "size:large", ` +
			`@post("/cmd/stream/payload", {payload: {"op": $_stream.op, "size": "large", tab: $_tab}, "requestCancellation":"disabled"}))`},
		{large.busy, `($_stream.want === "size:large") ? 'true' : false`},
		{push, `($_stream.n > 0) && (el.push && el.push($_stream.fw))`},
		{budgetValue, `Math.max(0, $_stream.budget)`},
	} {
		if c.got != c.want {
			t.Errorf("got  %s\nwant %s", c.got, c.want)
		}
	}
	if large.on {
		t.Error("the large button is pressed for a small table")
	}
}
