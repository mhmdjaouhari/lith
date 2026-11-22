package town

// The names of the town's markup that its templates, its frames
// (writeBoard) and its controller (town.ts) share. dsgen exports them to
// ds.gen.ts, so town.ts reads them as values that tsc checks, and TestSeam
// fails when the rendered page or town.css lacks one.
//
//ds:dom
const (
	AttrTile    = "data-t" // a tile's number, on its button
	AttrVersion = "data-v" // the tile's version, which a change sends as seen

	IdMap = "town-map" // the tiles, which the stream morphs

	ClassBoard  = "town-board"  // the map's frame: it posts the changes and answers the clicks and keys
	ClassTime   = "town-time"   // the slider's row: it posts the seeks
	ClassSlider = "town-slider" // the slider

	PropWait = "--town-wait" // the tile of the change on its way

	// The actions the markup's events call (views.templ), which town.ts
	// implements: tsc fails on one it lacks, TestSeam on one the page does
	// not call.
	ActPlace = "townPlace" // a click on a tile: it changes to the tool
	ActKey   = "townKey"   // a key on a tile: the arrows move the focus
	ActDrag  = "townDrag"  // the slider moved: it seeks its position
	ActNow   = "townNow"   // the Now button: the slider goes to its end
)
