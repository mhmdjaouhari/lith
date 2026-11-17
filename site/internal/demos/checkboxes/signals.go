package checkboxes

import "github.com/zweiundeins/lith/site/internal/demo"

//go:generate go run github.com/zweiundeins/lith/site/cmd/dsgen

// The contract between the grid's page, its controller (checkboxes.ts)
// and the server, in one place: dsgen writes S, the typed refs of Signals
// that the templates and the views read, and ds.gen.ts, the same structs
// and the grid's numbers (//ds:const) for tsc to check checkboxes.ts
// against.

// Signals are the demo's own on the page, declared on its stage, local
// to the browser: an action sends only its payload. They are what the
// page's attributes or the server read or write; the box asked for and
// the boxes that wait stay in checkboxes.ts.
//
//ds:signals _checkboxes S
type Signals struct {
	// op is the newest click, ack the newest the server took; op ids grow
	// with time, so an ack answers the older clicks too (checkboxes.ts).
	demo.OpSignals

	Cols  int    `json:"cols"`  // tiles to a row, as the page measured the grid
	Pitch int    `json:"pitch"` // px from box to box (--cb-pitch)
	Item  int    `json:"item"`  // a row's height in px: a tile and its head
	Jump  string `json:"jump"`  // the jump field
	Hello bool   `json:"hello"` // set by a stream's first frame, to ask for the window
}

// NewSignals are the signals of a page before it measures its grid.
func NewSignals() Signals {
	return Signals{Cols: defaultCols, Pitch: defaultPitch, Item: defaultItem}
}

// SetReq checks or clears a box: the state it should take, the opposite
// of what it showed, with an op id that the tab's clicks view acks.
//
//ds:command checkboxes set
type SetReq struct {
	Tab     string `json:"tab"`
	Op      string `json:"op"`
	Box     int    `json:"box"` // from 0
	Checked bool   `json:"checked"`
}

// WindowReq is what the list asks for as it scrolls (sb-window), and its
// tiles per row.
//
//ds:command checkboxes window
type WindowReq struct {
	Tab    string `json:"tab"`
	Offset int    `json:"offset"`
	Count  int    `json:"count"`
	Cols   int    `json:"cols"`
}

// The demo's actions.
var (
	SetAction    = demo.Command[SetReq]{Rate: demo.Rate{Every: setEvery, Burst: setBurst}}
	WindowAction = demo.Command[WindowReq]{Rate: demo.Rate{Every: windowEvery, Burst: windowBurst}}
)
