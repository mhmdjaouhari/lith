package cursorparty

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"github.com/a-h/templ"

	"github.com/zweiundeins/lith/site/internal/demo"
	"github.com/zweiundeins/lith/site/internal/ds"
)

// The Datastar expressions of the party, built from its actions
// (ActMove, which cursor-party.ts implements) and its typed command: none
// names an action or a route by hand. Measuring the anchors and finding
// the nearest take more than a line and live in cursor-party.ts.

// floor are the floor's attributes: its controller, the anchors' homes,
// the post of the moves the controller dispatches, and the pointer's
// listeners. The pointer is followed on the window, so that the hero's
// links and buttons, which sit above the floor, take nothing away: a move
// every Throttle at most (Datastar's throttle), and one more where the
// pointer comes to rest (its debounce), since the throttle drops the moves
// between two of its own. A tap or a click sends its place too. Moves go
// out side by side: none cancels the one before (requestCancellation),
// none is retried, and a failed one is dropped (Quiet): the next one
// replaces it.
func floor() templ.Attributes {
	move := string(ds.Call(ActMove))
	return demo.Attrs(
		MoveAction.On(demo.Post{RequestCancellation: "disabled", Retry: "never", Quiet: true}),
		templ.Attributes{
			"data-" + Slug: true,
			AttrHomes:      anchorHomes(),
			"data-on:pointermove__window__throttle." + ds.Ms(Throttle): move,
			"data-on:pointermove__window__debounce." + ds.Ms(Rest):     move,
			"data-on:pointerdown": move,
		},
	)
}

// countStyle hands the counter's numbers to CSS, which rolls them from
// one value to the next (@property --cursor-party-here and -rate).
func countStyle(here, rate int) templ.SafeCSS {
	return templ.SafeCSS(fmt.Sprintf("--cursor-party-here:%d;--cursor-party-rate:%d", here, rate))
}

// rate is the counter's moves a second: those of the last whole second.
func rate(h []int) int { return h[len(h)-1] }

// spark is the sparkline's data, a JSON array.
func spark(h []int) string {
	b, _ := json.Marshal(h)
	return string(b)
}

// sparkTop is the top of the sparkline's scale: its highest point, and at
// least the 15 moves a second of one pointer, so a quiet party stays low.
func sparkTop(h []int) string { return strconv.Itoa(max(15, slices.Max(h))) }

// countText is the counter for those who do not see it.
func countText(here, rate int) string {
	people := "people"
	if here == 1 {
		people = "person"
	}
	return fmt.Sprintf("%d %s here, %d moves a second", here, people, rate)
}
