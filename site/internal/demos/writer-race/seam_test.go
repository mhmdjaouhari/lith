package writerrace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds"
	"github.com/zweiundeins/lith/site/internal/ds/dscheck"
)

// TestSeam renders the demo's page and its landing slot, a race of the
// tab's on the line and one in the results, and checks each Datastar
// attribute: a known plugin in its form, valid JavaScript, and only
// signals that Signals and the shell declare.
func TestSeam(t *testing.T) {
	now := time.Now()
	race := Race{ID: 1, Settings: Settings{200, "NORMAL"}, State: "done"}
	var sb strings.Builder
	if err := Stage(track{}, []Race{race}, []Race{race}, now).Render(context.Background(), &sb); err != nil {
		t.Fatal(err)
	}
	if err := Embed(track{}).Render(context.Background(), &sb); err != nil {
		t.Fatal(err)
	}
	page := sb.String()
	schema := dscheck.Schema{"_writerRace": reflect.TypeFor[Signals]()}
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

// TestDeclared checks that the page declares what demo.Pending reads, and
// the slot's choices.
func TestDeclared(t *testing.T) {
	got := NewSignals().Decl()
	for _, want := range []string{`"_writerRace":{`, `"op":""`, `"ack":""`, `"synced":false`, `"problem":""`, `"inflight":0`, `"recheck":false`, `"producers":200`, `"sync":"NORMAL"`, `"mine":0`, `"active":false`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s lacks %s", got, want)
		}
	}
}

// TestPayload posts starts with a key misspelt, missing, extra, null or of
// the wrong type: each is a 400, never a race of zero producers.
func TestPayload(t *testing.T) {
	mux := http.NewServeMux()
	k := demo.NewKit(Slug, mux, demo.Deps{})
	var got []StartReq
	StartAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req StartReq) {
		got = append(got, req)
		w.WriteHeader(http.StatusNoContent)
	})
	for _, c := range []struct {
		body string
		want int
	}{
		{`{"tab":"t1","op":"o1","producers":50,"sync":"FULL"}`, http.StatusNoContent},
		{`{"tab":"t1","op":"o1","producer":50,"sync":"FULL"}`, http.StatusBadRequest},
		{`{"tab":"t1","op":"o1","sync":"FULL"}`, http.StatusBadRequest},
		{`{"tab":"t1","op":"o1","producers":null,"sync":"FULL"}`, http.StatusBadRequest},
		{`{"tab":"t1","op":"o1","producers":"50","sync":"FULL"}`, http.StatusBadRequest},
		{`{"tab":"t1","op":"o1","producers":50,"sync":"FULL","x":1}`, http.StatusBadRequest},
	} {
		r := httptest.NewRequest(http.MethodPost, StartAction.URL(), strings.NewReader(c.body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s: %d, want %d", c.body, w.Code, c.want)
		}
	}
	if len(got) != 1 || got[0] != (StartReq{Tab: "t1", Op: "o1", Producers: 50, Sync: "FULL"}) {
		t.Errorf("the handler got %+v", got)
	}
}

// TestExpressions pins what the typed builders write for the form.
func TestExpressions(t *testing.T) {
	want := ds.Expr(`(($_writerRace.op === "") && !$_writerRace.active) && (` +
		`$_writerRace.problem = "", ` +
		`$_writerRace.op = Date.now().toString(36) + Math.random().toString(36).slice(2, 7), ` +
		`@post("/cmd/writer-race/start", {payload: {"op": $_writerRace.op, "producers": $_writerRace.producers, "sync": $_writerRace.sync, tab: $_tab}, "requestCancellation":"disabled"}))`)
	if start != want {
		t.Errorf("start:\n%s\nwant\n%s", start, want)
	}
	if want := ds.Expr(`$_writerRace.mine === 3`); mine(3) != want {
		t.Errorf("mine: %s, want %s", mine(3), want)
	}
}
