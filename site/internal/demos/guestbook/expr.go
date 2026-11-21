package guestbook

import (
	"fmt"

	"github.com/zweiundeins/lith/site/internal/demo"
)

// The Datastar expressions of the demo, built from the typed refs of its
// signals (S) and its typed command: none names a signal or a route by
// hand. A form this small needs no controller; one whose logic grows past
// a line gets a <slug>.ts, as the board's and the checkboxes' do.

// op is the form's pending op: it ends on its ack (settle), a request that
// never got through (fetched), or a stream that connects again without its
// ack (lost), never on a clock.
var op = demo.PendingOf(S.PendingSignalsRefs)

// sign sends the form with a new op id unless one is pending. The payload
// is SignReq's, read from the signals: PostSignals panics, as the package
// loads, at a key it lacks or misspells. The tab's stream brings the op's
// outcome back as ack.
var sign = S.Op.Eq("") + " && " + trimmed(S.Name.String()) + " && " + trimmed(S.Message.String()) + ` && (
	` + S.Problem.Assign("") + `,
	` + S.Op.String() + ` = Date.now().toString(36) + Math.random().toString(36).slice(2, 7),
	` + SignAction.PostSignals(demo.Post{RequestCancellation: "disabled"},
	demo.From("op", S.Op),
	demo.From("name", S.Name),
	demo.From("message", S.Message),
) + `
)`

// pending is true from the click until the stream brings the outcome.
var pending = S.Op.Ne("")

// busy is pending for aria-busy: data-attr writes true as an empty value,
// which aria-busy reads as false.
var busy = pending + " ? 'true' : false"

// settle ends the pending state when the outcome arrives, and empties the
// message once it was accepted.
var settle = pending + " && " + S.Ack.String() + " === " + S.Op.String() + " && (" + S.Accepted.String() + " && (" + S.Message.Assign("") + "), " + S.Op.Assign("") + ")"

// fetched follows the form's requests (demo.Pending.Fetched).
var fetched = op.Fetched()

// The problem's line.
var problem = S.Problem.String()

// trimmed is a field's value without its spaces at the ends. The parens
// keep .trim() out of the signal's path, which dscheck reads up to the
// last dot.
func trimmed(v string) string { return "(" + v + ").trim()" }

func maxLen(n int) string { return fmt.Sprint(n) }
