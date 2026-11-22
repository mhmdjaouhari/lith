package canvas

// The names of the canvas's markup that its templates, its CSS and its
// controller (canvas.ts) share. dsgen exports them to ds.gen.ts, so
// canvas.ts reads them as values that tsc checks, and TestSeam fails when
// the rendered page or canvas.css lacks one.
//
//ds:dom
const (
	ClassProbe = "cv__probe" // the element whose colour resolves a token of the palette
)

// The palette, which canvas.ts reads from ds.gen.ts.
//
//ds:const
const (
	// Colours is how many colours sb-pixel-board paints with, one hex
	// digit a cell.
	Colours = 16
	// PropColour names a colour of the palette with its number, the site's
	// token canvas.css gives it: --canvas-c0 to --canvas-c15 (TestSeam).
	PropColour = "--canvas-c"
)
