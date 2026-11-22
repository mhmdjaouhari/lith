package writerrace

import "github.com/zweiundeins/lith/site/internal/demo"

//go:generate go run github.com/zweiundeins/lith/site/cmd/dsgen

// The contract between the race's page and the server, in one place:
// dsgen writes S, the typed refs of Signals that the templates and the
// views read. The race has no controller: its form is a few expressions,
// built from S (expr.go), and its pending op ends with demo.Pending.

// Signals are the demo's own on the page, declared on its stage and in
// its landing slot, local to the browser: an action sends only its
// payload.
//
//ds:signals _writerRace S
type Signals struct {
	// op is the start on its way, ack the one the tab's outcome answers;
	// inflight and recheck what demo.Pending keeps.
	demo.PendingSignals

	Producers int    `json:"producers"` // the choices, bound to the radios: data-bind keeps a number a number
	Sync      string `json:"sync"`      //
	Mine      int64  `json:"mine"`      // the tab's race, 0 for none
	Active    bool   `json:"active"`    // the tab's race waits in line or runs: no second start
}

// NewSignals are the signals of a page: 200 producers with NORMAL, the
// slot's race.
func NewSignals() Signals {
	return Signals{Producers: 200, Sync: "NORMAL"}
}

// StartReq starts a race with the choices, as the op that the tab's
// outcome answers.
//
//ds:command writer-race start
type StartReq struct {
	Tab       string `json:"tab"`
	Op        string `json:"op"`
	Producers int    `json:"producers"` // one of Producers
	Sync      string `json:"sync"`      // one of Syncs
}

// StartAction is the race's one action.
var StartAction = demo.Command[StartReq]{Rate: demo.Rate{Every: startEvery, Burst: startBurst}}
