package writerrace

import (
	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds"
)

// The Datastar expressions of the demo, built from the typed refs of its
// signals (S) and its typed command: none names a signal or a route by
// hand.

// op is the start's pending op: it ends on its ack (settle), a request that
// never got through (fetched), or a stream that connects again without its
// ack (lost), never on a clock.
var op = demo.PendingOf(S.PendingSignalsRefs)

// start sends the choices with a new op id unless one is pending or the
// tab's race still waits or runs. The payload is StartReq's, read from
// the signals: PostSignals panics, as the package loads, at a key it
// lacks or misspells. The tab's stream brings the op's outcome back as
// ack.
var start = ds.If(ds.And(S.Op.Eq(""), ds.Not(S.Active)),
	S.Problem.Assign(""),
	S.Op.To(ds.OpID),
	StartAction.PostSignals(demo.Post{RequestCancellation: "disabled"},
		demo.From("op", S.Op),
		demo.From("producers", S.Producers),
		demo.From("sync", S.Sync),
	),
)

// busy is aria-busy from the click until the stream brings the outcome:
// data-attr writes true as an empty value, which aria-busy reads as
// false.
var busy = ds.AriaBusy(S.Op.Ne(""))

// settle ends the pending state when the outcome arrives.
var settle = ds.If(ds.And(S.Op.Ne(""), S.Ack.Is(S.Op)), S.Op.Assign(""))

// fetched follows the start's requests (demo.Pending.Fetched).
var fetched = op.Fetched()

// mine marks the tab's own race in the shared views.
func mine(id int64) ds.Expr { return S.Mine.Eq(id) }
