package town

import (
	"github.com/a-h/templ"

	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds"
)

// The Datastar expressions of the demo, built from the typed refs of its
// signals (S), its actions (Act…, which town.ts implements) and its typed
// commands: none names a signal, an action or a route by hand. A change
// of a tile, the arrows and the slider's seeks take more than a line and
// live in town.ts; the rest is here, next to the markup that runs it.

// placing is the pending change of a tile: it ends on its ack (settle), a
// request that never got through, or a stream that connects again without
// its ack, never on a clock.
var placing = func() demo.Pending {
	p := demo.PendingOf(S.PendingSignalsRefs)
	p.End = string(ds.Do(S.Op.Assign(""), S.PTile.Assign(-1)))
	p.Down = "That change did not reach the server. Try again in a moment."
	return p
}()

// seeking is the slider's pending seek, by its number: the tab's view ends
// it with the seq of the newest seek it took, and so does a notice.
var seeking = func() demo.Pending {
	p := demo.PendingOf(S.PendingSignalsRefs).Within()
	p.Waiting = string(seekWaits)
	p.End = string(S.Mine.To(S.Seq))
	p.Inflight, p.Recheck = S.Seeks.String(), S.SeekRecheck.String()
	p.Down = "That position did not reach the server. Drag again."
	p.Lost = "That position did not reach the server. Drag again."
	return p
}()

// seekWaits is true while a seek waits for the server; pending is it for
// the map's aria-busy.
var (
	seekWaits = ds.Not(S.Mine.Is(S.Seq))
	pending   = ds.AriaBusy(seekWaits)
)

// post is how the changes and the seeks post: requestCancellation
// 'disabled', so that no change cancels the one before, and no seek the
// one in flight with its number.
var post = demo.Post{RequestCancellation: "disabled"}

// placePosts post the changes town.ts dispatches on the map's frame,
// whose requests placing counts; seekPosts the seeks it dispatches on the
// slider's row, whose requests seeking counts.
var (
	placePosts = PlaceAction.On(post)
	seekPosts  = SeekAction.On(post)
)

// drag seeks the slider's position once every Throttle at most,
// Datastar's throttle at its leading edge and at its trailing one, so that
// the position where the thumb comes to rest goes too: a position a tick.
var drag = templ.Attributes{"data-on:input__throttle." + ds.Ms(Throttle) + ".trailing": string(ds.Call(ActDrag))}

// past is true while the tab shows the town as it was: nothing to build.
var past = S.At.Ne(0)

// atEnd is true while the tab is live with the thumb at the end: the Now
// button has nothing to do.
var atEnd = ds.And(S.At.Eq(0), S.Want.AtLeast(S.Head))

// posLabel is the slider's position as the person drags it: their own
// input, shown at once.
var posLabel = ds.Cond(S.Want.AtLeast(S.Head), ds.Val("Now"), ds.Concat(ds.Val("Position "), ds.Grouped(S.Want)))

// settle ends the pending change once its outcome has come.
var settle = ds.If(ds.And(S.Op.Ne(""), S.Ack.Is(S.Op)), S.Op.Assign(""), S.PTile.Assign(-1))

// The requests' followers (demo.Pending.Fetched).
var (
	placeFetched = placing.Fetched()
	seekFetched  = seeking.Fetched()
)

// greet answers a stream's first frame: the server then watches the town
// at the tab's position on the new stream. A refusal is tried again.
var greet = ds.If(S.Hello, S.Hello.Assign(false), FollowAction.PostTab(demo.Post{Retry: "error", RetryMaxCount: 5}))

// badge has the stage's badge say "past" while the tab shows the past:
// its stream is open, but what the stage shows is not live.
var badge = demo.Badge.To(ds.Cond(past, ds.Val("past"), ds.Val("")))
