package stream

import (
	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds"
)

// The Datastar expressions of the demo, built from the typed refs of its
// signals (S) and its typed command: none names a signal or a route by
// hand. The reports fill the signals in; the page only shows them.

// op is the pending change: it ends on its ack (settle), a request that
// never got through (fetched), or a stream that connects again without its
// ack (lost), never on a clock. The buttons inside the controls send it.
var op = func() demo.Pending {
	p := demo.PendingOf(S.PendingSignalsRefs).Within()
	p.End = string(ds.Do(S.Op.Assign(""), S.Want.Assign("")))
	return p
}()

// tick counts the seconds since the last report, which sets it to 0;
// silent is true from the second one.
var (
	tick   = S.Quiet.Inc()
	silent = S.Quiet.Ge(2)
)

// option is a button of a choice: its label, whether the server pressed
// it, whether its change is on its way, and the change it sends.
type option struct {
	label                 string
	pressed, busy, choose ds.Expr
	on                    bool // the tab's choice as the page was drawn
}

// choice is the button of a value of the choice field, which the ref r
// holds: it sends a change with a new op id, unless one is pending or the
// tab has that choice already. The tab's stream brings the op back as
// ack, with the choice. busy reads Want, set to the field and the value,
// since both fields share one pending op.
func choice[T ~string](field string, r ds.Ref[T], v T, label string, current T) option {
	want := field + ":" + string(v)
	return option{
		label:   label,
		pressed: ds.Aria(r.Eq(v)),
		busy:    ds.AriaBusy(S.Want.Eq(want)),
		choose: ds.If(ds.And(S.Op.Eq(""), r.Ne(v)),
			S.Problem.Assign(""),
			S.Op.To(ds.OpID),
			S.Want.Assign(want),
			PayloadAction.PostSignals(demo.Post{RequestCancellation: "disabled"}, demo.From("op", S.Op), demo.Value(field, v)),
		),
		on: v == current,
	}
}

// sizeOptions and churnOptions are the buttons of the tab's choices st.
func sizeOptions(st TabState) []option {
	c := st.size()
	return []option{choice("size", S.Size, Small, "Small", c), choice("size", S.Size, Medium, "Medium", c), choice("size", S.Size, Large, "Large", c)}
}

func churnOptions(st TabState) []option {
	c := st.churn()
	return []option{choice("churn", S.Churn, Still, "Still", c), choice("churn", S.Churn, Calm, "Calm", c), choice("churn", S.Churn, Busy, "Busy", c), choice("churn", S.Churn, Storm, "Storm", c)}
}

// settle ends the pending state when the tab's stream brings the op back.
var settle = ds.If(ds.And(S.Op.Ne(""), S.Ack.Is(S.Op)), S.Op.Assign(""), S.Want.Assign(""))

// fetched follows the controls' requests (demo.Pending.Fetched).
var fetched = op.Fetched()

// reported is true once the first report came; push adds the last
// frame's bytes on the wire to the sparkline with every report, a repeat
// of the same number too.
var (
	reported = S.N.Gt(0)
	push     = ds.If(reported, ds.Invoke("push", S.FW))
)

// or is the signal, or what the page shows before a report fills it.
func or(r ds.Ref[string], before string) ds.Expr { return ds.Or(r, ds.Val(before)) }

// The numbers' words: the ratio's, the unit of the last frame's bytes,
// and the streams'.
var (
	ratioWords  = ds.Concat(S.Ratio, ds.Val(" to 1"))
	kilobytes   = S.FW.Ge(10_000)
	frameUnit   = ds.Cond(kilobytes, ds.Val("KB"), ds.Val("B"))
	streamsText = ds.Cond(S.Streams.Eq(1), ds.Val("stream open on this site"), ds.Val("streams open on this site"))
	lastFrame   = ds.Cond(reported, S.FWT, ds.Val("waiting"))
)

// The budget: the meter's value, and whether there is a limit.
var (
	budgetValue = ds.Max(ds.Val(0), S.Budget)
	limited     = S.Budget.Ge(0)
)
