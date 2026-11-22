package stream

import (
	"time"

	"github.com/zweiundeins/lith/site/internal/demo"
)

//go:generate go run github.com/zweiundeins/lith/site/cmd/dsgen

// The contract between the demo's page and the server, in one place:
// dsgen writes S, the typed refs of Signals that the templates and the
// views read. The demo has no controller: its page is expressions, built
// from S (expr.go), and its pending change ends with demo.Pending.

// Signals are the demo's own on the page, declared on its stage and in
// its landing slot, local to the browser: the reports fill them in
// (PageReport, EmbedReport), the tab's view the choices, and an action
// sends only its payload.
//
//ds:signals _stream S
type Signals struct {
	PageReport
	// op is the change on its way, ack the one the tab's choices answer;
	// inflight and recheck what demo.Pending keeps.
	demo.PendingSignals

	Size  Size   `json:"size"`  // the tab's choices, as the server last said
	Churn Churn  `json:"churn"` //
	Want  string `json:"want"`  // the button whose change is on its way: "size:large"
}

// NewSignals are the signals of a page whose tab has the choices of st,
// before its first report.
func NewSignals(st TabState) Signals {
	s := Signals{Size: st.size(), Churn: st.churn()}
	s.Budget = -1
	s.FPS = "0"
	return s
}

// PayloadReq changes the tab's payload: its size or its churn, as the op
// that the tab's choices answer.
//
//ds:command stream payload
type PayloadReq struct {
	Tab   string `json:"tab"`
	Op    string `json:"op"`
	Size  Size   `json:"size,omitempty"`  // one of Sizes
	Churn Churn  `json:"churn,omitempty"` // one of Churns
}

// A session changes its payload four times a second at most, after eight
// changes at once.
const (
	changeEvery = 250 * time.Millisecond
	changeBurst = 8
)

// PayloadAction is the demo's one action.
var PayloadAction = demo.Command[PayloadReq]{Rate: demo.Rate{Every: changeEvery, Burst: changeBurst}}
