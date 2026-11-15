package board

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"

	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds/dscheck"
)

// TestSeam renders the board's page and its embed and checks each
// Datastar attribute: a known plugin in its form, valid JavaScript, and
// only signals that Signals and the shell declare.
func TestSeam(t *testing.T) {
	b := Board{Version: 3}
	b.Lanes[0] = []Card{{ID: 1, Title: "It’s a card", Tag: "ops", Version: 2}}
	b.Lanes[1] = []Card{{ID: 2, Title: "Another", Tag: "docs", Version: 1, MovedBy: "x", MovedAt: time.Now()}}
	people := []Person{{Avatar: AvatarOf("x"), Tabs: 1}}
	schema := dscheck.Schema{"_board": reflect.TypeFor[Signals]()}
	for name, c := range map[string]templ.Component{
		"stage": Stage(b, Numbers{}, people, "x", time.Now()),
		"embed": Embed(b, Numbers{}, people, time.Now()),
	} {
		var sb strings.Builder
		if err := c.Render(context.Background(), &sb); err != nil {
			t.Fatal(err)
		}
		for _, err := range dscheck.Check(sb.String(), schema, "board") {
			t.Errorf("%s: %v", name, err)
		}
		if name == "stage" {
			for _, err := range dscheck.CheckDOM(sb.String(), css, dsDOM) {
				t.Errorf("the names board.ts reads the page by: %v", err)
			}
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

// TestMeStyle checks the viewer's style: its CSS intact inside <style>,
// and nothing for a seed that is not one.
func TestMeStyle(t *testing.T) {
	for me, want := range map[string]string{
		"0123abcd":                   `<style>[data-demo="board"] [data-who="0123abcd"]{--bd-me:1;--bd-you:" (you)"}</style>`,
		`x"]{}*{color:red}[data-x="`: ``,
		"":                           ``,
	} {
		var sb strings.Builder
		if err := meStyle(me).Render(context.Background(), &sb); err != nil {
			t.Fatal(err)
		}
		if sb.String() != want {
			t.Errorf("%q: %s, want %s", me, sb.String(), want)
		}
	}
}

// TestDeclared checks that the stage declares the signals Signals has,
// with their starting values.
func TestDeclared(t *testing.T) {
	got := NewSignals(7).Decl()
	for _, want := range []string{`"_board":{`, `"v":7`, `"op":""`, `"synced":false`, `"mode":""`, `"say":""`} {
		if !strings.Contains(got, want) {
			t.Errorf("%s lacks %s", got, want)
		}
	}
	// The drag's own state is the controller's (board.ts), out of the
	// page's signals.
	for _, gone := range []string{`"drag"`, `"lane"`, `"pcard"`, `"inflight"`} {
		if strings.Contains(got, gone) {
			t.Errorf("%s declares %s", got, gone)
		}
	}
}

// TestPayload posts the move's payload with a key misspelt, missing,
// extra or null: each is a 400, never a move with a zero lane.
func TestPayload(t *testing.T) {
	if MoveAction.URL() != "/cmd/board/move" || MoveAction.Event() != "board-move" {
		t.Fatalf("%s %s", MoveAction.URL(), MoveAction.Event())
	}
	mux := http.NewServeMux()
	k := demo.NewKit(Slug, mux, demo.Deps{})
	var got []MoveReq
	MoveAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req MoveReq) {
		got = append(got, req)
		w.WriteHeader(http.StatusNoContent)
	})
	for body, want := range map[string]int{
		`{"tab":"t1","op":"o1","card":4,"lane":1,"index":0,"seen":0}`:          http.StatusNoContent,
		`{"tab":"t1","op":"o1","card":4,"lanee":1,"index":0,"seen":0}`:         http.StatusBadRequest,
		`{"tab":"t1","op":"o1","card":4,"index":0,"seen":0}`:                   http.StatusBadRequest,
		`{"tab":"t1","op":"o1","card":4,"lane":1,"index":0,"seen":0,"x":1}`:    http.StatusBadRequest,
		`{"tab":"t1","op":"o1","card":4,"lane":null,"index":0,"seen":0}`:       http.StatusBadRequest,
		`{"tab":"t1","op":"o1","card":"4","lane":1,"index":0,"seen":0}`:        http.StatusBadRequest,
		`{"Tab":"t1","Op":"o1","Card":4,"Lane":1,"Index":0,"Seen":0}`:          http.StatusBadRequest,
		`{"tab":"no tab!","op":"o1","card":4,"lane":1,"index":0,"seen":0}`:     http.StatusBadRequest,
		`{"tab":"t1","op":"o1","card":4,"lane":1,"index":0,"seen":0} trailing`: http.StatusBadRequest,
	} {
		r := httptest.NewRequest(http.MethodPost, MoveAction.URL(), strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Errorf("%s: %d, want %d", body, w.Code, want)
		}
	}
	if len(got) != 1 || got[0] != (MoveReq{Tab: "t1", Op: "o1", Card: 4, Lane: 1, Seen: 0}) {
		t.Errorf("the handler got %+v", got)
	}
}

// TestExpressions pins what the typed builders write for the page.
func TestExpressions(t *testing.T) {
	for got, want := range map[string]string{
		post["data-on:board-move"].(string):        `@post("/cmd/board/move", {payload: {...evt.detail, tab: $_tab}, "requestCancellation":"disabled"})`,
		helloPosts["data-on:board-hello"].(string): `@post("/cmd/board/hello", {payload: {...evt.detail, tab: $_tab}, "retry":"error","retryMaxCount":5})`,
		S.Mode.Eq(Pointer):                         `$_board.mode === "pointer"`,
	} {
		if got != want {
			t.Errorf("got  %s\nwant %s", got, want)
		}
	}
}
