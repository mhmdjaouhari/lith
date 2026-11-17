// Package checkboxes is a million checkboxes that everyone shares. The
// boxes live in SQLite, 256 bits to a row. The page shows them in tiles of
// 256 inside Starbase's sb-virtual-scroll, which holds only the tiles near
// the view. Each tile is a view of its own, shared by every tab that shows
// it, and a tab watches the tiles of its window and no others: a tick is
// enough for a flip to reach everyone who has the box in view.
package checkboxes

import (
	"embed"
	"io/fs"
	"strconv"
	"time"

	"github.com/zweiundeins/lith/site/internal/demo"
)

// The grid's numbers (Boxes, ChunkSize, Chunks, Slots, MaxCols) are in
// dom.go, which shares them with checkboxes.ts.
const (
	Slug = "checkboxes"
	// Trend is how many seconds the rate's chart shows.
	Trend = 30
	// KeptSeconds is how long the per-second counts are kept.
	KeptSeconds = time.Hour

	setEvery, setBurst       = 100 * time.Millisecond, 30 // a session's clicks: 10 a second, after 30 at once
	windowEvery, windowBurst = 100 * time.Millisecond, 30 // a session's windows, as it scrolls
)

// StatsKey is the key of the numbers' view, which every tab watches.
const StatsKey = Slug + ":stats"

// ChunkTopic is touched by every flip in chunk n; it is also the key of
// the chunk's view.
func ChunkTopic(n int) string { return Slug + ":chunk:" + strconv.Itoa(n) }

var (
	//go:embed server.md
	server string
	//go:embed how.md
	how string
	//go:embed checkboxes.css
	css string
	// script is the controller and the seam it is checked against.
	//go:embed checkboxes.ts ds.gen.ts
	script embed.FS
)

// Demo is the million checkboxes as the site lists it.
var Demo = demo.Demo{
	Slug:       Slug,
	Title:      "A million checkboxes",
	Summary:    "One grid of a million boxes for everyone: tick one, and every tab that has it in view sees it flip within a tick.",
	Art:        art,
	Migrations: mustSub("migrations"),
	Server:     server,
	How:        how,
	CSS:        css,
	Script:     script,
	Components: []string{"virtual-scroll", "sparkline"},
	Mount:      mount,
	Page:       demo.Part{Render: renderPage, Views: pageViews},
	Sweep:      sweep,
}

//go:embed migrations
var migrationFiles embed.FS

func mustSub(dir string) fs.FS {
	f, err := fs.Sub(migrationFiles, dir)
	if err != nil {
		panic(err)
	}
	return f
}
