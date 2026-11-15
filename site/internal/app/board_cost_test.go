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

// TestBoardStageCost measures what a tab that watches the board spends on
// the board's frames: another session moves a card 40 times over HTTP, and
// Chromium's own counters on the watching tab (script, style, layout and
// task time) are read before and after, with the mutations under the stage
// and Datastar's re-scans of it (datastar-scope-children). It compares
// ways of hosting the stage's controller; it asserts nothing.
//
//	LITH_BOARD_COST=1 LITH_CHROME_NO_SANDBOX=1 go test ./internal/app -run TestBoardStageCost -v
func TestBoardStageCost(t *testing.T) {
	if os.Getenv("LITH_BOARD_COST") == "" {
		t.Skip("a measurement: LITH_BOARD_COST=1")
	}
	_, base := apptest.Start(t)
	b := browser.Start(t)
	browser.SkipFirefox(t, "Chromium's Performance.getMetrics")
	watcher := b.NewPage(t, 1280, 1000)
	watcher.Navigate(t, base+"/demos/board")
	watcher.Wait(t, `document.querySelector('.live-badge.is-live')`, 10*time.Second)
	watcher.Eval(t, `(() => {
		window.__cost = {mutations: 0, rescans: 0, frames: 0}
		const stage = document.querySelector('.bd')
		new MutationObserver((rs) => { window.__cost.mutations += rs.length }).observe(stage, {subtree: true, childList: true, attributes: true, characterData: true})
		document.addEventListener('datastar-scope-children', () => window.__cost.rescans++, true)
		document.addEventListener('datastar-patch-elements', () => window.__cost.frames++, true)
		return true
	})()`, nil)
	watcher.Call(t, "Performance.enable", map[string]any{"timeDomain": "threadTicks"})
	metrics := func() map[string]float64 {
		var r struct {
			Metrics []struct {
				Name  string
				Value float64
			}
		}
		if err := json.Unmarshal(watcher.Call(t, "Performance.getMetrics", nil), &r); err != nil {
			t.Fatal(err)
		}
		m := map[string]float64{}
		for _, x := range r.Metrics {
			m[x.Name] = x.Value
		}
		return m
	}

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
	var counts struct{ Mutations, Rescans, Frames int }
	watcher.Eval(t, `window.__cost`, &counts)
	for _, k := range []string{"ScriptDuration", "RecalcStyleDuration", "LayoutDuration", "TaskDuration"} {
		t.Logf("%-20s %7.1f ms a move", k, (after[k]-before[k])*1000/moves)
	}
	t.Logf("frames %d, mutations under the stage %d, re-scans %d", counts.Frames, counts.Mutations, counts.Rescans)
	if errs := watcher.Errors(); len(errs) > 0 {
		t.Errorf("errors: %v", errs)
	}
}
