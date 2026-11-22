package demo_test

import (
	"strings"
	"testing"

	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds"
)

// noteReq is a payload as dsgen would see it, its DSCommand by hand.
type noteReq struct {
	Tab  string `json:"tab"`
	Op   string `json:"op"`
	Text string `json:"text"`
	Pin  bool   `json:"pin"`
	Tag  string `json:"tag,omitempty"`
}

func (noteReq) DSCommand() (slug, name string) { return "notes", "add" }

var (
	noteOp   = ds.NewRef[string]("_notes.op")
	noteText = ds.NewRef[string]("_notes.text")
	notePin  = ds.NewRef[bool]("_notes.pin")
	noteN    = ds.NewRef[int]("_notes.n")
)

// TestPostSignals checks the post a form sends from its signals, and that
// a payload the server would refuse panics as it is built.
func TestPostSignals(t *testing.T) {
	c := demo.Command[noteReq]{}
	got := c.PostSignals(demo.Post{RequestCancellation: "disabled"}, demo.From("op", noteOp), demo.From("text", noteText), demo.From("pin", notePin))
	want := ds.Expr(`@post("/cmd/notes/add", {payload: {"op": $_notes.op, "text": $_notes.text, "pin": $_notes.pin, tab: $_tab}, "requestCancellation":"disabled"})`)
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	for name, c := range map[string]struct {
		args []demo.Arg
		says string
	}{
		"missing":  {[]demo.Arg{demo.From("op", noteOp), demo.From("text", noteText)}, `needs "pin"`},
		"unknown":  {[]demo.Arg{demo.From("op", noteOp), demo.From("text", noteText), demo.From("pin", notePin), demo.From("colour", noteText)}, `no key "colour"`},
		"twice":    {[]demo.Arg{demo.From("op", noteOp), demo.From("text", noteText), demo.From("text", noteText), demo.From("pin", notePin)}, `no key "text", or it is filled twice`},
		"type":     {[]demo.Arg{demo.From("op", noteOp), demo.From("text", noteN), demo.From("pin", notePin)}, `"text" is a string, not the int`},
		"value":    {[]demo.Arg{demo.From("op", noteOp), demo.From("text", noteText), demo.Value("pin", 1)}, `"pin" is a bool, not the int`},
		"tab sent": {[]demo.Arg{demo.From("tab", noteOp), demo.From("op", noteOp), demo.From("text", noteText), demo.From("pin", notePin)}, `no key "tab"`},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				r := recover()
				if s, _ := r.(string); !strings.Contains(s, c.says) {
					t.Errorf("panic %v, want one that says %s", r, c.says)
				}
			}()
			demo.Command[noteReq]{}.PostSignals(demo.Post{}, c.args...)
		})
	}
}

// TestPostValue checks a post with a key fixed in Go, and an optional key
// left out.
func TestPostValue(t *testing.T) {
	got := demo.Command[noteReq]{}.PostSignals(demo.Post{}, demo.From("op", noteOp), demo.Value("text", "it's"), demo.Value("pin", true))
	want := ds.Expr(`@post("/cmd/notes/add", {payload: {"op": $_notes.op, "text": "it's", "pin": true, tab: $_tab}})`)
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
