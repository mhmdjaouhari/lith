package board

// The names of the board's markup that its templates and its controller
// (board.ts) share. dsgen exports them to ds.gen.ts, so board.ts reads
// them as values that tsc checks, and TestSeam fails when the rendered
// page lacks one: renaming a class or an attribute in one place breaks
// the build or a test, never the drag.
//
//ds:dom
const (
	AttrLane     = "data-lane"    // a lane's index, on its section
	AttrLaneName = "data-name"    // the lane's name, for what the page says
	AttrCard     = "data-card"    // a card's id, on its slot
	AttrVersion  = "data-version" // the card's version, which a move sends as seen
	AttrTitle    = "data-title"   // the card's title, for what the page says

	ClassLanes     = "bd-lanes"       // the lanes, the drag's bounds
	ClassLaneCards = "bd-lane__cards" // a lane's list, the marker's anchor when it is empty
	ClassSlot      = "bd-slot"        // a card's place in its list
	ClassCard      = "bd-card"        // the card, the ghost's copy
	ClassGrip      = "bd-card__grip"  // where a finger drags it
	ClassGhost     = "bd-ghost"       // the copy that follows the pointer
	ClassMarker    = "bd-marker"      // the bar where the card would go

	// The custom properties and animations board.css draws the page's
	// state with: the controller sets them on the cards, the lanes and
	// the marker, or reads them.
	PropFlow   = "--bd-flow"   // 1 where a lane is a row of cards that wraps
	PropLift   = "--bd-lift"   // the card the pointer holds
	PropPick   = "--bd-pick"   // the card the keyboard holds
	PropWait   = "--bd-wait"   // the card of the pending move
	PropBusy   = "--bd-busy"   // its pixel's animation
	PropNudge  = "--bd-nudge"  // a refused card's animation
	PropTarget = "--bd-target" // the lane the card would go to
	PropOn     = "--bd-on"     // the marker shows
	PropEdge   = "--bd-edge"   // the marker sits below its anchor, not above
	PropSlot   = "--bd-slot"   // the marker's anchor
	AnimBusy   = "bd-busy"
	AnimNudge  = "bd-nudge"
)
