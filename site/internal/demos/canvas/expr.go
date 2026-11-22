package canvas

import (
	"time"

	"github.com/a-h/templ"

	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds"
)

// The Datastar expressions of the canvas, built from the typed refs of
// its signals (S) and its typed commands: none names a signal or a route
// by hand. Reading the palette from the site's tokens takes more than a
// line and lives in canvas.ts.

// paint posts a stroke's batch, sb-pixel-board's sb-paint. A stroke sends
// one every 80 ms, and Datastar would cancel a request still under way
// when the next one goes to the same URL: the pixels of a slow request
// would be lost.
var paint = PaintAction.OnEvent("sb-paint", demo.Post{RequestCancellation: "disabled"})

// refused says why pixels stay faded when their batch never reached the
// server: only the browser knows, so the page says so in its own words.
// A batch over the pixel rate is a refusal the tab's view shows, and a
// failed write a notice. sb-pixel-board takes the pixels back after three
// seconds, since no frame confirms them: the component's own fade, with
// no op of the page's waiting.
var refused = demo.RetriesFailed(S.Problem.Assign("Some pixels did not reach the canvas: the connection is down. They fade back; paint them again once it is back."))

// paletteAttr is the board's palette attribute: none until canvas.ts has
// read the colours (false removes the attribute).
var paletteAttr = ds.Or(S.Palette, ds.Val(false))

// choose makes the swatch of colour c the brush, at once.
func choose(c int) ds.Expr { return ds.Do(S.Color.Assign(c), S.Problem.Assign("")) }

// pick tells the server the brush: as the page starts, which adds the tab
// to the painters, and a moment after the last change, so that arrowing
// through the swatches sends one pick.
var (
	pick      = PickAction.PostSignals(demo.Post{}, demo.From("color", S.Color))
	pickLater = templ.Attributes{"data-on:change__debounce." + ds.Ms(pickQuiet): string(pick)}
)

// pickQuiet is how long the brush stays put before the pick goes.
const pickQuiet = 300 * time.Millisecond

// pickFailed: the brush changed on this page, but the list of painters
// still shows the old colour until the next paint. It says so when the
// pick never reached the server; a failed write comes as a notice.
var pickFailed = demo.RetriesFailed(S.Problem.Assign("Your new colour did not reach the server; the painters see it with your next stroke."))
