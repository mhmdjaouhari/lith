package board

import "github.com/zweiundeins/lith/site/internal/demo"

//go:generate go run github.com/zweiundeins/lith/site/cmd/dsgen

// The contract between the board's page, its controller (board.ts) and
// the server, in one place: dsgen writes S, the typed refs of Signals that
// the templates and expressions read, and ds.gen.d.ts, the same structs
// for tsc to check board.ts against. A misspelt or renamed signal or
// payload key fails the Go build or tsc, never a page at runtime.

// Signals are the demo's own, declared on its stage, local to the
// browser: a move sends only its payload.
//
//ds:signals _board S
type Signals struct {
	demo.PendingSignals // op is the move waiting for its outcome

	V     int64  `json:"v"`     // the board's version on the page
	Drag  int64  `json:"drag"`  // the card in hand, 0: none
	Mode  Mode   `json:"mode"`  // how it is held
	Seen  int64  `json:"seen"`  // the card's version when it was picked up
	From  int    `json:"from"`  // the lane it was picked up from
	At    int    `json:"at"`    // and its place there
	Lane  int    `json:"lane"`  // where it would go, -1: nowhere
	Index int    `json:"index"` // its place there
	N     int    `json:"n"`     // how many other cards that lane has
	PCard int64  `json:"pcard"` // the card of the pending move
	AckV  int64  `json:"ackv"`  // the board version that has the tab's last outcome
	OK    bool   `json:"ok"`    // whether that outcome moved the card
	Nudge int64  `json:"nudge"` // a refused card, for a moment
	Say   string `json:"say"`   // the live region's text
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
	return Signals{V: version, From: -1, Lane: -1}
}

// MoveReq moves a card: the move in hand, with the op id that ends its
// pending state and the card's version from when it was picked up.
//
//ds:command board move
type MoveReq struct {
	Tab   string `json:"tab"`
	Op    string `json:"op"`
	Card  int64  `json:"card"`
	Lane  int    `json:"lane"`
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
