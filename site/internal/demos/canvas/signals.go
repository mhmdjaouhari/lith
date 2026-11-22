package canvas

import "github.com/zweiundeins/lith/site/internal/demo"

//go:generate go run github.com/zweiundeins/lith/site/cmd/dsgen

// The contract between the canvas's page, its controller (canvas.ts) and
// the server, in one place: dsgen writes S, the typed refs of Signals that
// the templates read, and ds.gen.ts, the same structs and the markup's
// names (dom.go) for tsc to check canvas.ts against.

// Signals are the tab's on the page, declared on its stage, in the
// browser: an action sends only its payload. The canvas has no pending
// op: sb-pixel-board's own fade takes back the pixels no frame confirms.
//
//ds:signals _canvas S
type Signals struct {
	Color   int    `json:"color"`   // the brush, 0 to Colours-1
	Palette string `json:"palette"` // the palette in hex, as JSON, which canvas.ts reads from the site's tokens: "" until then
	Problem string `json:"problem"` // what went wrong with the last action, in the page's words
}

// PaintReq is a stroke's batch: the cells painted in the brush's colour.
// sb-pixel-board's sb-paint carries both as its detail.
//
//ds:command canvas paint
type PaintReq struct {
	Tab   string `json:"tab"`
	Color int    `json:"color"`
	Cells []int  `json:"cells"`
}

// PickReq tells the server the tab's brush, which the painters show.
//
//ds:command canvas pick
type PickReq struct {
	Tab   string `json:"tab"`
	Color int    `json:"color"`
}

// The canvas's actions. A stroke sends a batch every 80 ms: Separate
// keeps it out of the site's limit for all actions, which it would use
// up; the pixel rate is the canvas's own rule (mount).
var (
	PaintAction = demo.Command[PaintReq]{Rate: demo.Rate{Every: paintEvery, Burst: paintBurst, Separate: true}}
	PickAction  = demo.Command[PickReq]{Rate: demo.Rate{Every: pickEvery, Burst: pickBurst}}
)
