package life

import (
	"github.com/a-h/templ"

	"github.com/zweiundeins/lith/site/internal/demo"
)

// The Datastar expressions of the demo, built from the typed refs of its
// signals (S) and its typed commands: none names a signal or a route by
// hand. The strokes, the drops, the keyboard, the colours, the reset and
// the pending state live in life.ts, which tsc checks against the same
// structs.

// posts post the actions life.ts dispatches on the stage, whose requests
// lith-ui's pending counts: requestCancellation 'disabled', so that every
// batch of a stroke arrives and no action cancels the one before.
var posts = merge(
	DrawAction.On(demo.Post{RequestCancellation: "disabled"}),
	DropAction.On(demo.Post{RequestCancellation: "disabled"}),
	ResetAction.On(demo.Post{RequestCancellation: "disabled"}),
	ColourAction.On(demo.Post{RequestCancellation: "disabled"}),
)

// useTool picks a tool, and toolPressed is its button's aria-pressed.
func useTool(name string) string     { return S.Tool.Assign(name) }
func toolPressed(name string) string { return S.Tool.Eq(name) + " ? 'true' : 'false'" }

// drawing is true while the tool draws: the field's crosshair, and the
// turn button, which has nothing to turn.
var drawing = S.Tool.Eq("draw")

// turn turns the pattern a quarter clockwise.
var turn = S.Turn.String() + " = " + S.Turn.String() + " + 1"

// picked and picking are a swatch's aria-pressed and aria-busy: data-attr
// writes true as an empty value, which aria-busy reads as false.
func picked(c int) string { return S.Colour.Eq(c) + " ? 'true' : 'false'" }
func picking(c int) string {
	return S.Pick.Eq(c) + " && " + S.Colour.Ne(c) + " ? 'true' : false"
}

// resetting is the reset button's aria-busy.
var resetting = S.Resetting.String() + " ? 'true' : false"

// marks are the stroke's cells, sent and not, as one SVG path; clearing
// marks a stroke that clears.
var (
	marks    = S.Pend.String() + " + " + S.Draft.String()
	clearing = "!" + S.Alive.String()
)

// The outlines: the pattern under the pointer or the cursor, and the one
// dropped last until it is confirmed, each shown, named and placed by its
// signals.
var (
	cursorOn  = S.X.String() + " >= 0"
	stampOn   = S.PTool.Ne("")
	cursor    = outlineAt(S.X.String(), S.Y.String(), S.Turn.String())
	stamp     = outlineAt(S.PX.String(), S.PY.String(), S.PTurn.String())
	cursorFor = S.Tool.String()
	stampFor  = S.PTool.String()
)

func outlineAt(x, y, turn string) string {
	return `'--x:' + ` + x + ` + ';--y:' + ` + y + ` + ';--turn:' + ` + turn
}

// me is the stage's --me, the colour this tab draws in, for the outlines
// and the pending marks.
var me = `'--me: var(--life-' + ' abcdefgh'[` + S.Colour.String() + `] + ')'`

// The problem's line.
var problem = S.Problem.String()

func merge(as ...templ.Attributes) templ.Attributes {
	out := templ.Attributes{}
	for _, a := range as {
		for k, v := range a {
			out[k] = v
		}
	}
	return out
}
