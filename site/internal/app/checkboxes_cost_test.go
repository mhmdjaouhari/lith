package app_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/zweiundeins/lith/site/internal/app/apptest"
	"github.com/zweiundeins/lith/site/internal/browser"
)

// TestCheckboxesScrollCost measures what scrolling the grid back and forth
// costs the tab: three times 3,000 px down and back up in steps of 150 px
// every 50 ms. Besides Chromium's counters it counts the windows the page
// asks for, Rocket's re-scans of the list (datastar-scope-children, one
// per patch inside it), and the long tasks, their sum and the longest.
//
//	LITH_BOARD_COST=1 LITH_CHROME_NO_SANDBOX=1 go test ./internal/app -run TestCheckboxesScrollCost -v
func TestCheckboxesScrollCost(t *testing.T) {
	costGate(t)
	_, base := apptest.Start(t)
	b := browser.Start(t)
	browser.SkipFirefox(t, "Chromium's Performance.getMetrics")
	p := b.NewPage(t, 1280, 1000)
	p.Navigate(t, base+"/demos/checkboxes")
	p.Wait(t, `document.querySelector('.live-badge.is-live') && customElements.get('sb-virtual-scroll') && document.getElementById('checkboxes-grid').shadowRoot`, 10*time.Second)
	time.Sleep(time.Second)
	p.Eval(t, `(() => {
		window.__cost = {rescans: 0, windows: 0, long: 0, longest: 0, longs: 0}
		document.addEventListener('datastar-scope-children', () => __cost.rescans++, true)
		document.getElementById('checkboxes-grid').addEventListener('sb-window', () => __cost.windows++)
		new PerformanceObserver((l) => { for (const e of l.getEntries()) { __cost.longs++; __cost.long += e.duration; __cost.longest = Math.max(__cost.longest, e.duration) } }).observe({type: 'longtask'})
		return true
	})()`, nil)
	costCSS(t, p)
	metrics := costMetrics(t, p)
	before := metrics()
	const scroller = `document.getElementById('checkboxes-grid').shadowRoot.querySelector('.scroller')`
	steps := 0
	for range 3 {
		for _, d := range []int{150, -150} {
			for range 20 {
				p.Eval(t, scroller+`.scrollBy(0, `+strconv.Itoa(d)+`)`, nil)
				steps++
				time.Sleep(50 * time.Millisecond)
			}
		}
	}
	time.Sleep(time.Second)
	after := metrics()
	var c struct {
		Rescans, Windows, Longs int
		Long, Longest           float64
	}
	p.Eval(t, `window.__cost`, &c)
	costLog(t, before, after, float64(steps), "step")
	t.Logf("total: task %.0f ms, script %.0f ms, style %.0f ms, layout %.0f ms", (after["TaskDuration"]-before["TaskDuration"])*1000, (after["ScriptDuration"]-before["ScriptDuration"])*1000, (after["RecalcStyleDuration"]-before["RecalcStyleDuration"])*1000, (after["LayoutDuration"]-before["LayoutDuration"])*1000)
	t.Logf("sb-window %d, re-scans %d, long tasks %d (%.0f ms, longest %.0f ms)", c.Windows, c.Rescans, c.Longs, c.Long, c.Longest)
	if errs := p.Errors(); len(errs) > 0 {
		t.Logf("page errors (a measurement, so logged): %v", errs)
	}
}
