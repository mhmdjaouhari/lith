package town

import "github.com/zweiundeins/lith/site/internal/demo"

//go:generate go run github.com/zweiundeins/lith/site/cmd/dsgen

// The contract between the town's page, its controller (town.ts) and the
// server, in one place: dsgen writes S, the typed refs of Signals that
// the templates and the views read, and ds.gen.ts, the same structs, the
// markup's names (dom.go) and the town's size for tsc to check town.ts
// against.

// Signals are the demo's own on the page, declared on its stage, local to
// the browser: an action sends only its payload.
//
//ds:signals _town S
type Signals struct {
	// op is the change of a tile on its way, ack the one the tab's last
	// outcome answers; inflight and recheck what its demo.Pending keeps.
	demo.PendingSignals

	Tool  string `json:"tool"`  // the palette's choice, one of Tools
	PTile int    `json:"ptile"` // the tile of the change on its way, -1 for none
	OK    bool   `json:"ok"`    // the tab's last change went through

	// The slider: its value, the number of the page's last seek, the
	// position the server has the tab at (0: now) and the number of the
	// seek that put it there (the tab's view), and its ends (the head
	// view). The seeks' demo.Pending keeps their own count in flight.
	Want        int64 `json:"want"`
	Mine        int64 `json:"mine"`
	At          int64 `json:"at"`
	Seq         int64 `json:"seq"`
	First       int64 `json:"first"`
	Head        int64 `json:"head"`
	Seeks       int   `json:"seeks"`
	SeekRecheck bool  `json:"seekRecheck"`

	Hello bool `json:"hello"` // a new stream's first frame: the page says hello (Follow)
}

// NewSignals are the signals of a page whose slider has the ends of b,
// its thumb at the end: a house to build, no change on its way.
func NewSignals(b Bounds) Signals {
	return Signals{Tool: House, PTile: -1, Want: b.Head, First: b.First, Head: b.Head}
}

// PlaceReq changes the tile to kind, from the version the page showed.
//
//ds:command town place
type PlaceReq struct {
	Tab  string `json:"tab"`
	Op   string `json:"op"`
	Tile int    `json:"tile"` // 0 to Tiles-1
	Kind string `json:"kind"` // one of Tools
	Seen int64  `json:"seen"`
}

// SeekReq moves the tab to the position at, 0 for now, as its seek seq:
// the server drops a seek older than the tab's last.
//
//ds:command town seek
type SeekReq struct {
	Tab string `json:"tab"`
	At  int64  `json:"at"`
	Seq int64  `json:"seq"`
}

// FollowReq is a new stream's hello: the server watches the town at the
// tab's position on it.
//
//ds:command town follow
type FollowReq struct {
	Tab string `json:"tab"`
}

// The demo's actions. The slider sends a position each tick while it is
// dragged: Separate keeps a drag out of the site's limit for all actions,
// which it would use up for every other demo, and each new position
// renders the tab's views afresh, so an address gets one drag's worth
// (Sessions: 1), however many sessions it makes up.
var (
	PlaceAction  = demo.Command[PlaceReq]{Rate: demo.Rate{Every: placeEvery, Burst: placeBurst}}
	SeekAction   = demo.Command[SeekReq]{Rate: demo.Rate{Every: seekEvery, Burst: seekBurst, Separate: true, Sessions: 1}}
	FollowAction = demo.Command[FollowReq]{Rate: demo.Rate{Every: followEvery, Burst: followBurst}}
)
