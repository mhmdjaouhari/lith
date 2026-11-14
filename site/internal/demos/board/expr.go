package board

import (
	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds"
	"github.com/zweiundeins/lith/site/internal/views"
)

// The Datastar expressions of the demo, built from the typed refs of its
// signals (S, which dsgen writes from Signals) and its typed commands:
// none names a signal or a route by hand. The drag, the keyboard and the
// pending move's styles live in board.ts, which tsc checks against the
// same structs.
//
// Nothing here moves a card. A drag or the keyboard picks a place, and the
// page shows where the card would go; the move goes to the server, the card
// waits there, marked pending, and the board's next frame moves it.

// moving is the pending move: it ends on its outcome (board.ts's settle),
// a request that never got through, or a stream that connects again
// without its ack, never on a clock.
var moving = func() demo.Pending {
	p := demo.PendingOf(S.PendingSignalsRefs)
	p.End = S.Op.Assign("") + ", " + S.PCard.Assign(0) + ", " + S.Drag.String() + " || (" + S.Lane.Assign(-1) + ")"
	p.Down = "That move did not reach the server. Try again in a moment."
	p.Lost = "That move did not go through. Try again."
	return p
}()

// post sends the move board.ts dispatches: requestCancellation 'disabled',
// so that a second move to the same URL never cancels the first.
var post = MoveAction.On(demo.Post{RequestCancellation: "disabled"})

// fetched says so while the move's request is retried (the op id makes a
// retry safe), and follows the stage's requests (demo.Pending.Fetched).
var fetched = `evt.detail.el === el && ` + S.Op.String() + ` && evt.detail.type === 'retrying' && (` + S.Problem.Assign("The server does not answer. Trying again.") + `);
` + moving.Fetched()

// hello tells the server this tab is on the page, for the presence row:
// once when the page loads, and again whenever the page's stream opens
// (after a deploy). It retries a refusal by the rate limit.
var hello = HelloAction.PostTab(demo.Post{Retry: "error", RetryMaxCount: 5})

var helloAgain = `evt.detail.el.id === ` + ds.Lit(views.StreamID) + ` && evt.detail.type === 'started' && ` + hello
