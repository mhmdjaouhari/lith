package app_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/zweiundeins/lith/site/internal/app/apptest"
	"github.com/zweiundeins/lith/site/internal/browser"
)

// The cost tests measure what the site costs a tab's main thread, from
// Chromium's own counters (Performance.getMetrics), and assert nothing.
// They run with LITH_BOARD_COST=1. LITH_COST_CSS is a stylesheet added to
// the measured tab first, to switch a suspect off (`*{animation:none
// !important}`) and compare.
//
// On a busy machine the durations swing by a third from run to run: run
// the variants interleaved, five times or more, and compare medians. The
// counts (RecalcStyleCount, LayoutCount) hardly move with the machine's
// load, and say whether a change saves work or only time.

// costMetrics returns a function that reads p's counters.
func costMetrics(t *testing.T, p *browser.Page) func() map[string]float64 {
	t.Helper()
	p.Call(t, "Performance.enable", map[string]any{"timeDomain": "threadTicks"})
	return func() map[string]float64 {
		var r struct {
			Metrics []struct {
				Name  string
				Value float64
			}
		}
		if err := json.Unmarshal(p.Call(t, "Performance.getMetrics", nil), &r); err != nil {
			t.Fatal(err)
		}
		m := map[string]float64{}
		for _, x := range r.Metrics {
			m[x.Name] = x.Value
		}
		return m
	}
}

// costCSS adds LITH_COST_CSS to p.
func costCSS(t *testing.T, p *browser.Page) {
	t.Helper()
	css := os.Getenv("LITH_COST_CSS")
	if css == "" {
		return
	}
	b, _ := json.Marshal(css)
	p.Eval(t, `(() => { const s = document.createElement('style'); s.textContent = `+string(b)+`; document.head.append(s); return true })()`, nil)
}

// costLog logs the counters' change between before and after over n units.
func costLog(t *testing.T, before, after map[string]float64, n float64, unit string) {
	t.Helper()
	for _, k := range []string{"TaskDuration", "RecalcStyleDuration", "LayoutDuration", "ScriptDuration"} {
		t.Logf("%-20s %7.2f ms a %s", k, (after[k]-before[k])*1000/n, unit)
	}
	for _, k := range []string{"RecalcStyleCount", "LayoutCount"} {
		t.Logf("%-20s %7.2f a %s", k, (after[k]-before[k])/n, unit)
	}
}

func costGate(t *testing.T) {
	if os.Getenv("LITH_BOARD_COST") == "" {
		t.Skip("a measurement: LITH_BOARD_COST=1")
	}
}

// TestBoardDragCost measures what a drag costs the dragging tab: the
// mouse drags a card across the lanes in 60 steps.
//
//	LITH_BOARD_COST=1 LITH_CHROME_NO_SANDBOX=1 go test ./internal/app -run TestBoardDragCost -v
func TestBoardDragCost(t *testing.T) {
	costGate(t)
	_, base := apptest.Start(t)
	b := browser.Start(t)
	browser.SkipFirefox(t, "Chromium's Performance.getMetrics")
	p := b.NewPage(t, 1280, 1000)
	p.Navigate(t, base+"/demos/board")
	p.Wait(t, `document.querySelector('.live-badge.is-live')`, 10*time.Second)
	p.Eval(t, `document.querySelector('.bd__board').scrollIntoView({block: 'center', behavior: 'instant'})`, nil)
	costCSS(t, p)
	metrics := costMetrics(t, p)
	s := boardRect(t, p, "#board-card-1")
	d := boardRect(t, p, `[data-lane="2"]`)
	sx, sy, tx, ty := s.X+s.W/2, s.Y+s.H/2, d.X+d.W/2, d.Y+d.H*0.9
	mouse := func(typ string, x, y float64) {
		p.Call(t, "Input.dispatchMouseEvent", map[string]any{"type": typ, "x": x, "y": y, "button": "left", "buttons": 1, "clickCount": 1})
	}
	mouse("mousePressed", sx, sy)
	before := metrics()
	const steps = 60
	for i := 1; i <= steps; i++ {
		f := float64(i) / steps
		mouse("mouseMoved", sx+(tx-sx)*f, sy+(ty-sy)*f)
		time.Sleep(8 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	after := metrics()
	mouse("mouseReleased", tx, ty)
	p.Wait(t, `document.querySelector('[data-lane="2"] #board-card-1')`, 5*time.Second)
	costLog(t, before, after, steps, "pointer move")
	if errs := p.Errors(); len(errs) > 0 {
		t.Errorf("errors: %v", errs)
	}
}

// TestBoardStageCost measures what a tab that watches the board spends on
// the board's frames: another session moves a card 40 times over HTTP, a
// move every 300 ms. The mutations under the stage and Datastar's re-scans
// of it (datastar-scope-children) are counted besides.
//
//	LITH_BOARD_COST=1 LITH_CHROME_NO_SANDBOX=1 go test ./internal/app -run TestBoardStageCost -v
func TestBoardStageCost(t *testing.T) {
	costGate(t)
	_, base := apptest.Start(t)
	b := browser.Start(t)
	browser.SkipFirefox(t, "Chromium's Performance.getMetrics")
	watcher := b.NewPage(t, 1280, 1000)
	watcher.Navigate(t, base+"/demos/board")
	watcher.Wait(t, `document.querySelector('.live-badge.is-live')`, 10*time.Second)
	watcher.Eval(t, `(() => {
		window.__cost = {mutations: 0, rescans: 0}
		const stage = document.querySelector('.bd')
		new MutationObserver((rs) => { window.__cost.mutations += rs.length }).observe(stage, {subtree: true, childList: true, attributes: true, characterData: true})
		document.addEventListener('datastar-scope-children', () => window.__cost.rescans++, true)
		return true
	})()`, nil)
	costCSS(t, watcher)
	metrics := costMetrics(t, watcher)

	c := client(t)
	get(t, c, base+"/demos/board")
	listen(t, c, base+"/demos/board/stream?tab=mover")
	time.Sleep(300 * time.Millisecond)
	before := metrics()
	const moves = 40
	for i := range moves {
		body := fmt.Sprintf(`{"tab":"mover","op":"cost%d","card":4,"lane":%d,"index":0,"seen":%d}`, i, (i+1)%2, i)
		if code := post(t, c, base+"/cmd/board/move", body); code != http.StatusNoContent {
			t.Fatalf("move %d: %d", i, code)
		}
		time.Sleep(300 * time.Millisecond)
	}
	watcher.Wait(t, fmt.Sprintf(`document.querySelector('#board-card-4')?.dataset.version === '%d'`, moves), 5*time.Second)
	time.Sleep(time.Second)
	after := metrics()
	var counts struct{ Mutations, Rescans int }
	watcher.Eval(t, `window.__cost`, &counts)
	costLog(t, before, after, moves, "move")
	t.Logf("mutations under the stage %d, re-scans %d", counts.Mutations, counts.Rescans)
	if errs := watcher.Errors(); len(errs) > 0 {
		t.Logf("page errors (a measurement, so logged): %v", errs)
	}
}

// TestIdleCost measures what a page costs a tab while nothing happens on
// it, over 12 seconds: its animations, its streams' quiet frames. The
// page is LITH_COST_PAGE, /demos/board by default.
//
//	LITH_BOARD_COST=1 LITH_COST_PAGE=/ LITH_CHROME_NO_SANDBOX=1 go test ./internal/app -run TestIdleCost -v
func TestIdleCost(t *testing.T) {
	costGate(t)
	_, base := apptest.Start(t)
	b := browser.Start(t)
	browser.SkipFirefox(t, "Chromium's Performance.getMetrics")
	page := os.Getenv("LITH_COST_PAGE")
	if page == "" {
		page = "/demos/board"
	}
	p := b.NewPage(t, 1280, 1000)
	p.Navigate(t, base+page)
	p.Wait(t, `document.readyState === 'complete'`, 10*time.Second)
	time.Sleep(time.Second)
	costCSS(t, p)
	metrics := costMetrics(t, p)
	before := metrics()
	const seconds = 12
	time.Sleep(seconds * time.Second)
	costLog(t, before, metrics(), seconds, "second")
	if errs := p.Errors(); len(errs) > 0 {
		t.Logf("page errors (a measurement, so logged): %v", errs)
	}
}
