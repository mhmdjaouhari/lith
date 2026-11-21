package guestbook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds/dscheck"
)

// TestSeam renders the demo's page and its landing slot and checks each
// Datastar attribute: a known plugin in its form, valid JavaScript, and
// only signals that Signals and the shell declare.
func TestSeam(t *testing.T) {
	now := time.Now()
	list := []Entry{{ID: 1, Name: "Ada", Message: "Hello", At: now}}
	var sb strings.Builder
	if err := Stage(list, 1, now).Render(context.Background(), &sb); err != nil {
		t.Fatal(err)
	}
	if err := Embed(list, 1, now).Render(context.Background(), &sb); err != nil {
		t.Fatal(err)
	}
	page := sb.String()
	schema := dscheck.Schema{"_guestbook": reflect.TypeFor[Signals]()}
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

// TestDeclared checks that the stage declares what demo.Pending reads,
// and the form's fields, empty.
func TestDeclared(t *testing.T) {
	got := Signals{}.Decl()
	for _, want := range []string{`"_guestbook":{`, `"op":""`, `"ack":""`, `"synced":false`, `"problem":""`, `"inflight":0`, `"recheck":false`, `"name":""`, `"message":""`, `"accepted":false`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s lacks %s", got, want)
		}
	}
}

// TestPayload posts signatures with a key misspelt, missing, extra or
// null: each is a 400, never an entry without a name.
func TestPayload(t *testing.T) {
	mux := http.NewServeMux()
	k := demo.NewKit(Slug, mux, demo.Deps{})
	var got []SignReq
	SignAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req SignReq) {
		got = append(got, req)
		w.WriteHeader(http.StatusNoContent)
	})
	for _, c := range []struct {
		body string
		want int
	}{
		{`{"tab":"t1","op":"o1","name":"Ada","message":"Hello"}`, http.StatusNoContent},
		{`{"tab":"t1","op":"o1","nam":"Ada","message":"Hello"}`, http.StatusBadRequest},
		{`{"tab":"t1","op":"o1","message":"Hello"}`, http.StatusBadRequest},
		{`{"tab":"t1","op":"o1","name":null,"message":"Hello"}`, http.StatusBadRequest},
		{`{"tab":"t1","op":"o1","name":"Ada","message":"Hello","x":1}`, http.StatusBadRequest},
	} {
		r := httptest.NewRequest(http.MethodPost, SignAction.URL(), strings.NewReader(c.body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s: %d, want %d", c.body, w.Code, c.want)
		}
	}
	if len(got) != 1 || got[0] != (SignReq{Tab: "t1", Op: "o1", Name: "Ada", Message: "Hello"}) {
		t.Errorf("the handler got %+v", got)
	}
}

// TestExpressions pins what the typed builders write for the form.
func TestExpressions(t *testing.T) {
	want := `$_guestbook.op === "" && ($_guestbook.name).trim() && ($_guestbook.message).trim() && (
	$_guestbook.problem = "",
	$_guestbook.op = Date.now().toString(36) + Math.random().toString(36).slice(2, 7),
	@post("/cmd/guestbook/sign", {payload: {"op": $_guestbook.op, "name": $_guestbook.name, "message": $_guestbook.message, tab: $_tab}, "requestCancellation":"disabled"})
)`
	if sign != want {
		t.Errorf("sign:\n%s\nwant\n%s", sign, want)
	}
	if want := `$_guestbook.op !== "" && $_guestbook.ack === $_guestbook.op && ($_guestbook.accepted && ($_guestbook.message = ""), $_guestbook.op = "")`; settle != want {
		t.Errorf("settle:\n%s\nwant\n%s", settle, want)
	}
}
