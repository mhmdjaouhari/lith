package checkboxes

// The names of the grid's markup that its templates, its tiles (writeTile)
// and its controller (checkboxes.ts) share. dsgen exports them to
// ds.gen.ts, so checkboxes.ts reads them as values that tsc checks, and
// TestSeam fails when the rendered page lacks one.
//
//ds:dom
const (
	AttrChunk  = "data-chunk"  // a tile's chunk, from which a box counts its number
	AttrWindow = "data-window" // the element that posts the windows the list asks for as it scrolls
	AttrGreet  = "data-greet"  // the element that answers a stream's first frame with the window

	// The ids the stream finds its regions by, and the prefixes of the
	// ids a frame matches its elements by.
	IdBoard = "checkboxes-board" // the list at its window (boardView)
	IdGrid  = "checkboxes-grid"  // the sb-virtual-scroll
	IdStats = "checkboxes-stats" // the numbers (statsView)
	IdSlot  = "checkboxes-slot-" // a slot, by its number: a chunk's frame goes into it
	IdTile  = "checkboxes-tile-" // a tile, by its chunk

	ClassFrame = "cb__frame" // the box the grid measures its width by
	ClassJump  = "cb-jump"   // the jump form
	ClassSlot  = "cb-slot"   // a slot
	ClassTile  = "cb-tile"   // a tile
	ClassBoxes = "cb-boxes"  // a tile's boxes, the checkboxes and nothing else

	PropPitch  = "--cb-pitch"  // px from box to box, a number, from the controller's measure
	PropChrome = "--cb-chrome" // a tile's head and the space under it

	// The actions the stage's events call (views.templ), which
	// checkboxes.ts implements: tsc fails on one it lacks, TestSeam on one
	// the page does not call.
	ActClick   = "checkboxesClick"   // a click on a box: it sends the state the box should take
	ActFlipped = "checkboxesFlipped" // a frame set a box (datastar-prop-change): its wait ends
	ActKey     = "checkboxesKey"     // a key on a box: the arrows move the focus
	ActWindow  = "checkboxesWindow"  // the list asks for a window as it scrolls (sb-window)
	ActMeasure = "checkboxesMeasure" // the window was resized: the grid fits its width again
	ActJump    = "checkboxesJump"    // the jump form: the list scrolls to the box
)

// The grid's numbers, which checkboxes.ts reads from ds.gen.ts.
//
//ds:const
const (
	// Boxes is how many boxes there are, numbered 1 to Boxes on the page
	// and 0 to Boxes-1 inside.
	Boxes = 1_000_000
	// ChunkSize is how many boxes a chunk holds: a row of
	// checkboxes_chunks, a tile on the page, a view.
	ChunkSize = 256
	// Chunks is how many chunks there are; the last one holds 64 boxes.
	Chunks = (Boxes + ChunkSize - 1) / ChunkSize
	// Slots is how many chunks a tab watches at once: the window around
	// what it shows. The page has as many slots for tiles, and chunk n's
	// tile goes into slot n % Slots, so its frame finds its place in every
	// tab that shows it.
	Slots = 12
	// MaxCols is the most tiles in a row of the grid.
	MaxCols = 4
)
