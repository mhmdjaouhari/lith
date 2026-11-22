package cursorparty

import "github.com/zweiundeins/lith/site/internal/demo"

//go:generate go run github.com/zweiundeins/lith/site/cmd/dsgen

// The contract between the party's page, its controller (cursor-party.ts)
// and the server, in one place: dsgen writes S, the typed refs of Signals,
// and ds.gen.ts, the same structs and the markup's names (dom.go) for tsc
// to check cursor-party.ts against.

// Signals are the page's own, declared on the party, local to the
// browser: a move sends only its payload.
//
//ds:signals _cursorParty S
type Signals struct {
	Seq int64 `json:"seq"` // the count of the page's moves, which orders them
}

// MoveReq is where the pointer is: the anchor nearest to it ("" or floor
// for the floor itself), and its place as fractions of that anchor's box,
// numbered so that the server keeps the newest.
//
//ds:command cursor-party move
type MoveReq struct {
	Tab string  `json:"tab"`
	A   string  `json:"a,omitempty"`
	X   float64 `json:"x"`
	Y   float64 `json:"y"`
	Seq int64   `json:"seq"`
}

// MoveAction is the party's one action: a move every Throttle while the
// pointer moves. Separate keeps the moves out of the site's limit for all
// actions, which they would use up for every other demo.
var MoveAction = demo.Command[MoveReq]{Rate: demo.Rate{Every: moveEvery, Burst: moveBurst, Separate: true}}
