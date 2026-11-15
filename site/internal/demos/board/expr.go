package board

import (
	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds"
	"github.com/zweiundeins/lith/site/internal/views"
)

// The Datastar expressions of the demo, built from the typed refs of its
// signals (S, which dsgen writes from Signals) and its typed commands:
// none names a signal or a route by hand. The drag, the keyboard, the
// pending move and its looks live in board.ts, which tsc checks against
// the same structs.
//
// Nothing here moves a card. A drag or the keyboard picks a place, and the
// page shows where the card would go; the move goes to the server, the card
// waits there, marked pending, and the board's next frame moves it.

// post sends the move board.ts dispatches: requestCancellation 'disabled',
// so that a second move to the same URL never cancels the first.
var post = MoveAction.On(demo.Post{RequestCancellation: "disabled"})

// hello tells the server this tab is on the page, for the presence row:
// once when the page loads, and again whenever the page's stream opens
// (after a deploy). It retries a refusal by the rate limit. It goes out
// from an element of its own, so that the stage's count of the move's
// requests in flight never sees it.
var hello = HelloAction.PostTab(demo.Post{Retry: "error", RetryMaxCount: 5})

var helloAgain = `evt.detail.el.id === ` + ds.Lit(views.StreamID) + ` && evt.detail.type === 'started' && ` + hello
