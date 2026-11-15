package board

import "github.com/zweiundeins/lith/site/internal/demo"

//go:generate go run github.com/zweiundeins/lith/site/cmd/dsgen

// The contract between the board's page, its controller (board.ts) and
// the server, in one place: dsgen writes S, the typed refs of Signals that
// the templates and expressions read, and ds.gen.d.ts, the same structs
// for tsc to check board.ts against. A misspelt or renamed signal or
// payload key fails the Go build or tsc, never a page at runtime.

// Signals are the demo's own on the page, declared on its stage, local
// to the browser: a move sends only its payload. They are what the page's
// attributes or the server read or write; the drag's own state stays in
// board.ts (lith-ui's local), where no patch can reach it.
//
//ds:signals _board S
type Signals struct {
	demo.OpSignals // op is the move waiting for its outcome; board.ts ends it (lith-ui's pending)

	V    int64  `json:"v"`    // the board's version on the page (the server's board frame)
	AckV int64  `json:"ackv"` // the board version that has the tab's last outcome (its tab frame)
	OK   bool   `json:"ok"`   // whether that outcome moved the card
	Mode Mode   `json:"mode"` // how a card is held (the stage's class, the ghost)
	Say  string `json:"say"`  // the live region's text
}

// Mode is how a card is held.
type Mode string

const (
	Free    Mode = ""        // no card in hand
	Pointer Mode = "pointer" // by a mouse, pen or finger
	Keys    Mode = "keys"    // from the keyboard
)

// NewSignals are the signals of a page that shows the board at version.
func NewSignals(version int64) Signals {
	return Signals{V: version}
}

// LaneID is a lane of the board, by its index: a union of these three in
// TypeScript, with isLaneID to tell one.
type LaneID int

const (
	Todo LaneID = iota
	Doing
	Done
)

// CardID is a card's id.
type CardID int64

// MoveReq moves a card: the move in hand, with the op id that ends its
// pending state and the card's version from when it was picked up.
//
//ds:command board move
type MoveReq struct {
	Tab   string `json:"tab"`
	Op    string `json:"op"`
	Card  CardID `json:"card"`
	Lane  LaneID `json:"lane"`
	Index int    `json:"index"`
	Seen  int64  `json:"seen"`
}

// HelloReq tells the server the tab is on the page.
//
//ds:command board hello
type HelloReq struct {
	Tab string `json:"tab"`
}

// The board's actions.
var (
	MoveAction  = demo.Command[MoveReq]{Rate: demo.Rate{Every: moveEvery, Burst: moveBurst}}
	HelloAction = demo.Command[HelloReq]{Rate: demo.Rate{Every: helloEvery, Burst: helloBurst}}
)
