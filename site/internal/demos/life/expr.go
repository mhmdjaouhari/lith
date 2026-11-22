package life

import (
	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds"
)

// The Datastar expressions of the demo, built from the typed refs of its
// signals (S), its actions (Act…, which life.ts implements) and its typed
// commands: none names a signal, an action or a route by hand. A stroke,
// a drop and the keyboard take more than a line and live in life.ts; the
// rest is here, next to the markup that runs it (views.templ).

// post is how every action posts: requestCancellation 'disabled', so that
// every batch of a stroke arrives and no action cancels the one before.
// lith-ui's pending counts the requests of the stage and of the buttons
// in it.
var post = demo.Post{RequestCancellation: "disabled"}

// posts post the strokes and the drops life.ts dispatches on the stage.
var posts = demo.Attrs(DrawAction.On(post), DropAction.On(post))

// useTool picks a tool, and toolPressed is its button's aria-pressed.
func useTool(name string) ds.Expr     { return S.Tool.Assign(name) }
func toolPressed(name string) ds.Expr { return ds.Aria(S.Tool.Eq(name)) }

// drawing is true while the tool draws: the field's crosshair, and the
// turn button, which has nothing to turn.
var drawing = S.Tool.Eq(ToolDraw)

// The field's events: the draw tool strokes, a pattern drops on a click.
var (
	begin   = ds.If(drawing, ds.Call(ActBegin))
	dropped = ds.If(ds.Not(drawing), ds.Call(ActDrop))
)

// pick asks for the colour c, which stays pending until the tab's view
// brings it back as colour; picked and picking are its swatch's
// aria-pressed and aria-busy.
func pick(c int) ds.Expr {
	return ds.Do(S.Pick.Assign(c), S.Problem.Assign(""), ColourAction.PostSignals(post, demo.From("colour", S.Pick)))
}
func picked(c int) ds.Expr  { return ds.Aria(S.Colour.Eq(c)) }
func picking(c int) ds.Expr { return ds.AriaBusy(ds.And(S.Pick.Eq(c), S.Colour.Ne(c))) }

// reset asks to clear the board, one reset at a time, as a new op;
// resetting is its button's aria-busy.
var (
	reset = ds.If(ds.Not(S.Resetting),
		S.Problem.Assign(""),
		S.Op.To(ds.OrderedOpID),
		S.Resetting.Assign(true),
		ResetAction.PostSignals(post, demo.From("op", S.Op)),
	)
	resetting = ds.AriaBusy(S.Resetting)
)

// settle ends what was pending once the tab's view confirms the newest op,
// and with it the older ones (op ids grow with time): the board's frame
// shows it.
var settle = ds.If(ds.And(S.Op.Ne(""), S.Ack.AtLeast(S.Op)),
	S.Op.Assign(""),
	S.Pend.Assign(""),
	S.PTool.Assign(""),
	S.Resetting.Assign(false),
)

// marks are the stroke's cells, sent and not, as one SVG path; clearing
// marks a stroke that clears.
var (
	marks    = ds.Concat(S.Pend, S.Draft)
	clearing = ds.Not(S.Alive)
)

// The outlines show while they have a cell: the pattern under the pointer
// or the cursor, and the one dropped last until it is confirmed.
var (
	cursorOn = S.X.Ge(0)
	stampOn  = S.PTool.Ne("")
)
