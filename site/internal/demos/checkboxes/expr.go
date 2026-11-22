package checkboxes

import (
	"github.com/a-h/templ"

	"github.com/zweiundeins/lith/site/internal/demo"
)

// The Datastar expressions of the demo, built from the typed refs of its
// signals (S) and its typed commands: none names a signal or a route by
// hand. The clicks, the keyboard, the jump, the pending boxes and the
// grid's measure live in checkboxes.ts, which tsc checks against the same
// structs.

// The grid's size before the page measures it: two tiles to a row, boxes
// 20px apart. chrome is a tile's head and the space under it, in px
// (--cb-chrome in checkboxes.css).
const (
	defaultCols  = 2
	defaultPitch = 20
	chrome       = 34
	defaultItem  = 16*defaultPitch + chrome
)

// setPosts posts the clicks checkboxes.ts dispatches on the stage, whose
// requests lith-ui's pending counts: requestCancellation 'disabled', so
// that a second click never cancels the first.
var setPosts = SetAction.On(demo.Post{RequestCancellation: "disabled"})

// windowPosts posts the windows the list asks for as it scrolls, from an
// element of their own: a newer window cancels the one in flight, as
// Datastar does for an element's requests to the same URL.
var windowPosts = WindowAction.On(demo.Post{})

// greetPosts posts the window that answers a stream's first frame, so
// that the new stream watches its chunks; a refusal (the rate, a restart)
// is tried again.
var greetPosts = WindowAction.On(demo.Post{Retry: "error", RetryMaxCount: 5})

// pitch is the distance from box to box that the tiles take: the signal,
// a number of px, as --cb-pitch, which checkboxes.css multiplies by 1px.
var pitch = templ.Attributes{"data-style:" + PropPitch: S.Pitch.String()}
