package cursorparty

// The names of the party's markup that its templates and its controller
// (cursor-party.ts) share. dsgen exports them to ds.gen.ts, so
// cursor-party.ts reads them as values that tsc checks, and TestSeam fails
// when the rendered page lacks one.
//
//ds:dom
const (
	AttrHomes = "data-cp-homes" // the floor: the anchors' homes, as JSON (Anchors)

	// The action the floor's pointer calls (views.templ), which
	// cursor-party.ts implements: tsc fails if it lacks it, TestSeam if
	// the page does not call it.
	ActMove = "cursorPartyMove"
)

// The names the controller reads the landing page by, and hands its
// boxes to CSS by. The anchors are the landing page's, whose template
// (views/landing.templ) writes their attribute.
//
//ds:const
const (
	// AttrAnchor marks an anchor, by its name.
	AttrAnchor = "data-cp-anchor"
	// PropAnchor starts the custom properties of an anchor's box on the
	// floor: --cp-<name>-x, -y, -w and -h.
	PropAnchor = "--cp-"
)
