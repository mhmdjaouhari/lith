package app_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zweiundeins/lith/site/internal/app/apptest"
	"github.com/zweiundeins/lith/site/internal/browser"
)

// TestViewTransitionSkipped starts a view transition on a scope while
// another runs there, which skips the first: its update still runs, and
// the page reports no error for the animation it dropped. The board's
// frames do that when moves come faster than a glide (640 ms). An update
// that throws is still an error on the page.
func TestViewTransitionSkipped(t *testing.T) {
	_, base := apptest.Start(t)
	b := browser.Start(t)
	browser.SkipFirefox(t, "element-scoped view transitions")
	p := b.NewPage(t, 1280, 900)
	p.Navigate(t, base+"/demos/board")
	p.Wait(t, `document.querySelector('.live-badge.is-live')`, 10*time.Second)

	p.Eval(t, `(() => {
		const s = document.querySelector('#board-lanes')
		window.__ran = 0
		s.startViewTransition(() => window.__ran++)
		s.startViewTransition(() => window.__ran++)
		return true
	})()`, nil)
	p.Wait(t, `window.__ran === 2`, 2*time.Second)

	// The board's own frames, a move every 150 ms from another session.
	c := client(t)
	get(t, c, base+"/demos/board")
	listen(t, c, base+"/demos/board/stream?tab=mover")
	time.Sleep(300 * time.Millisecond)
	for i := range 6 {
		body := fmt.Sprintf(`{"tab":"mover","op":"skip%d","card":4,"lane":%d,"index":0,"seen":%d}`, i, (i+1)%2, i)
		if code := post(t, c, base+"/cmd/board/move", body); code != http.StatusNoContent {
			t.Fatalf("move %d: %d", i, code)
		}
		time.Sleep(150 * time.Millisecond)
	}
	p.Wait(t, `document.querySelector('#board-card-4')?.dataset.version === '6'`, 5*time.Second)
	time.Sleep(time.Second)
	if errs := p.Errors(); len(errs) > 0 {
		t.Errorf("skipped transitions reported errors: %v", errs)
	}

	reported := func(what, text string) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for !slices.ContainsFunc(p.Errors(), func(e string) bool { return strings.Contains(e, text) }) {
			if time.Now().After(deadline) {
				t.Fatalf("%s is not reported: %v", what, p.Errors())
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	p.Eval(t, `(document.querySelector('#board-lanes').startViewTransition(() => { throw new Error('the update failed') }), true)`, nil)
	reported("an update that throws", "the update failed")

	// Two elements with one name: the transition cannot run, which is a
	// mistake in the page, not a skip.
	p.Eval(t, `(() => {
		const [a, b] = document.querySelectorAll('#board-lanes [data-card]')
		a.style.viewTransitionName = b.style.viewTransitionName = 'twice'
		document.querySelector('#board-lanes').startViewTransition(() => {})
		return true
	})()`, nil)
	time.Sleep(500 * time.Millisecond)
	reported("a duplicate view-transition-name", "InvalidStateError")
}
