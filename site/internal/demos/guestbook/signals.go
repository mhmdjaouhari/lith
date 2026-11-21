package guestbook

import "github.com/zweiundeins/lith/site/internal/demo"

//go:generate go run github.com/zweiundeins/lith/site/cmd/dsgen

// The contract between the guestbook's page and the server, in one place:
// dsgen writes S, the typed refs of Signals that the templates and the
// views read. The guestbook has no controller: its form is a few
// expressions, built from S (expr.go), and its pending op ends with
// demo.Pending, so ds.gen.ts only types the structs for a controller to
// come.

// Signals are the demo's own on the page, declared on its stage, local
// to the browser: an action sends only its payload.
//
//ds:signals _guestbook S
type Signals struct {
	// op is the signature on its way, ack the one the tab's outcome
	// answers; inflight and recheck what demo.Pending keeps.
	demo.PendingSignals

	Name     string `json:"name"`     // the form's fields, bound to its inputs
	Message  string `json:"message"`  //
	Accepted bool   `json:"accepted"` // the tab's last signature added an entry: the form empties its message
}

// SignReq is a signature: the name and the message, with the op id that
// the tab's outcome answers.
//
//ds:command guestbook sign
type SignReq struct {
	Tab     string `json:"tab"`
	Op      string `json:"op"`
	Name    string `json:"name"`    // 1 to MaxName characters
	Message string `json:"message"` // 1 to MaxMessage characters
}

// SignAction is the guestbook's one action: a session signs once in three
// seconds, after three at once.
var SignAction = demo.Command[SignReq]{Rate: demo.Rate{Every: signEvery, Burst: signBurst}}
