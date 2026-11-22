package life

// The names of the board's markup that its templates, its frames
// (boardCells) and its controller (life.ts) share. dsgen exports them to
// ds.gen.ts, so life.ts reads them as values that tsc checks, and TestSeam
// fails when the rendered page or life.css lacks one.
//
//ds:dom
const (
	// The stage, by the colour this tab draws in: life.css gives it --me.
	AttrColour = "data-colour"

	// The board, which every frame morphs, and its rows by their number:
	// a row's id lets the morph pair it with the row it replaces.
	IdBoard = "life-board"
	IdRow   = "life-r-"

	ClassField  = "life__field"  // the board, the marks and the outlines: what the pointer and the keys act on
	ClassMarks  = "life__marks"  // the pending cells, drawn from the signals
	ClassCursor = "life__cursor" // a pattern's outline: under the pointer, and the one dropped last

	// The actions the field's events call (views.templ), which life.ts
	// implements: tsc fails on one it lacks, TestSeam on one the page
	// does not call.
	ActBegin = "lifeBegin" // the pointer goes down with the draw tool: a stroke starts
	ActMove  = "lifeMove"  // the pointer moves: the cursor follows, a stroke grows
	ActEnd   = "lifeEnd"   // the pointer goes up or is lost: the stroke's rest goes
	ActAway  = "lifeAway"  // the pointer or the focus leaves: the cursor goes
	ActDrop  = "lifeDrop"  // a click with a pattern: it drops there
	ActKey   = "lifeKey"   // a key on the board
)

// The numbers and names of a stroke, which life.ts reads from ds.gen.ts.
//
//ds:const
const (
	// MaxStroke is the most points one draw command takes; lines join
	// them.
	MaxStroke = 64
	// StrokeBatch and FlushMillis are when the page sends a stroke's
	// points: once it holds StrokeBatch of them, or FlushMillis after it
	// last sent, ten times a second at most (the draw's rate). With the
	// last point sent before them, a batch stays under MaxStroke.
	StrokeBatch = 48
	FlushMillis = 100
	// CellClasses are the classes of the board's cells, by colour: _ for a
	// dead cell, a to h for the people's colours, w for the soup's. A cell
	// with another class (a dying one, a trail) is dead too.
	CellClasses = "_abcdefghw"
	// ToolDraw is the tool that draws cells; every other tool is a
	// pattern to drop.
	ToolDraw = "draw"
)
