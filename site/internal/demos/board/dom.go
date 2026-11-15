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
	AttrHello    = "data-hello"   // the element that says hello, outside the stage's count of the move's requests
	AttrWho      = "data-who"     // an avatar's seed, which marks the viewer's own (meStyle)

	// The ids of the regions the stream patches, and the prefixes of the ids
	// a frame matches its elements by. A frame finds its target by its id,
	// so a template that dropped one would leave that region stale.
	IdLanes    = "board-lanes"   // the cards (boardView), and the scope of their view transition
	IdHere     = "board-here"    // who is on the board (hereView)
	IdStats    = "board-stats"   // the numbers (statsView)
	IdOutcome  = "board-outcome" // the tab's last move (tabView)
	IdMini     = "board-mini"    // the landing page's miniature (miniView)
	IdCard     = "board-card-"   // a card's slot, and its view-transition name, by its id
	IdAvatar   = "board-av-"     // an avatar, by its seed
	IdMiniCard = "board-mini-"   // a card in the miniature, by its id and version

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
	PropMe     = "--bd-me"     // the viewer's own avatar (meStyle)
	PropYou    = "--bd-you"    // and what its name adds
	AnimBusy   = "bd-busy"
	AnimNudge  = "bd-nudge"
)
