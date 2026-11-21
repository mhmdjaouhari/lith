package life

import (
	"time"

	"github.com/zweiundeins/lith/site/internal/demo"
)

//go:generate go run github.com/zweiundeins/lith/site/cmd/dsgen

// The contract between the board's page, its controller (life.ts) and the
// server, in one place: dsgen writes S, the typed refs of Signals that the
// templates and the views read, and ds.gen.ts, the same structs, the
// markup's names (dom.go) and the board's numbers (//ds:const in board.go)
// for tsc to check life.ts against.

// Signals are the demo's own on the page, declared on its stage, local to
// the browser: an action sends only its payload. They are what the page's
// attributes or the server read or write; a stroke under way (its points
// not sent yet, the last one sent, when it last sent) stays in life.ts.
//
//ds:signals _life S
type Signals struct {
	// op is the newest action, ack the newest the tab's view confirmed;
	// op ids grow with time (orderedOpId), so an ack answers the older
	// actions too.
	demo.OpSignals

	Tool  string `json:"tool"`  // draw, or a pattern's name (tools)
	Turn  int    `json:"turn"`  // the pattern's quarter turns; they only grow, so the outline always turns clockwise
	X     int    `json:"x"`     // the cell under the pointer or the keyboard's cursor; -1: none
	Y     int    `json:"y"`     //
	Alive bool   `json:"alive"` // the stroke draws (true) or clears
	Draft string `json:"draft"` // the stroke's cells not sent yet, as an SVG path
	Pend  string `json:"pend"`  // the cells sent and not confirmed, as an SVG path
	PX    int    `json:"px"`    // the pattern dropped last, while it is not confirmed
	PY    int    `json:"py"`    //
	PTool string `json:"ptool"` // "" once confirmed
	PTurn int    `json:"pturn"` //

	Colour    int  `json:"colour"`    // the colour the server says this session draws in
	Pick      int  `json:"pick"`      // the colour asked for, until the server has it
	Resetting bool `json:"resetting"` // a reset is on its way
}

// NewSignals are the signals of a page whose session draws in colour: a
// glider to drop, and no cursor.
func NewSignals(colour int) Signals {
	return Signals{Tool: "glider", X: -1, Y: -1, Alive: true, PX: -1, PY: -1, Colour: colour, Pick: colour}
}

// DrawReq is a stroke's batch of cells: drawn (alive) or cleared, the
// first one joined by a line to the last of the batch before.
//
//ds:command life draw
type DrawReq struct {
	Tab   string `json:"tab"`
	Op    string `json:"op"`
	Cells []int  `json:"cells"` // y*W + x, 1 to MaxStroke of them
	Alive bool   `json:"alive"`
}

// DropReq drops a pattern with its top left at cell, turned turn
// quarters clockwise.
//
//ds:command life drop
type DropReq struct {
	Tab     string `json:"tab"`
	Op      string `json:"op"`
	Pattern string `json:"pattern"`
	Cell    int    `json:"cell"`
	Turn    int    `json:"turn"` // 0 to 3
}

// ResetReq clears the board, unless someone did a moment ago.
//
//ds:command life reset
type ResetReq struct {
	Tab string `json:"tab"`
	Op  string `json:"op"`
}

// ColourReq picks the colour the session draws in.
//
//ds:command life colour
type ColourReq struct {
	Tab    string `json:"tab"`
	Colour int    `json:"colour"` // 1 to Colours
}

// The demo's actions, at their rates per session; an address gets four
// times as much. A stroke sends ten requests a second while it lasts: it
// stays out of the site's limit for all actions, which it would use up.
var (
	DrawAction   = demo.Command[DrawReq]{Rate: demo.Rate{Every: FlushMillis * time.Millisecond, Burst: 20, Separate: true}}
	DropAction   = demo.Command[DropReq]{Rate: demo.Rate{Every: 500 * time.Millisecond, Burst: 6}}
	ResetAction  = demo.Command[ResetReq]{Rate: demo.Rate{Every: ResetEvery, Burst: 2}}
	ColourAction = demo.Command[ColourReq]{Rate: demo.Rate{Every: time.Second, Burst: 5}}
)
