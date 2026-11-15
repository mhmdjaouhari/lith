package app_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zweiundeins/lith/site/internal/app/apptest"
	"github.com/zweiundeins/lith/site/internal/browser"
)

// TestBoardPendingAcrossAReconnect does to a board move what
// TestPendingAcrossAReconnect does to a signature, for the board's
// controller, which ends its op in TypeScript (lith-ui's pending): a 500
// ends nothing, a reconnect without the ack does, and a bare 400 at once.
func TestBoardPendingAcrossAReconnect(t *testing.T) {
	a, base := apptest.Start(t)
	p := browser.Start(t).NewPage(t, 1280, 1000)
	p.Navigate(t, base+"/demos/board")
	p.Wait(t, `document.querySelector('.live-badge.is-live')`, 10*time.Second)
	p.Eval(t, `(document.addEventListener('datastar-fetch', e => { if (e.detail.type === 'error') window.lastStatus = e.detail.argsRaw.status }), true)`, nil)
	visibility := func(hidden bool) {
		t.Helper()
		h := fmt.Sprint(hidden)
		p.Eval(t, `(() => {
			Object.defineProperty(document, 'hidden', {get: () => `+h+`, configurable: true})
			Object.defineProperty(document, 'visibilityState', {get: () => `+h+` ? 'hidden' : 'visible', configurable: true})
			document.dispatchEvent(new Event('visibilitychange', {bubbles: true}))
			return true
		})()`, nil)
	}
	move := func(card int) {
		t.Helper()
		p.Eval(t, `(window.lastStatus = undefined, true)`, nil)
		boardFocus(t, p, card)
		boardPress(t, p, " ")
		boardPress(t, p, "ArrowRight")
		boardPress(t, p, " ")
	}
	const waiting = `document.querySelector('style[data-bd-looks]').textContent.includes('--bd-wait')`
	ended := func(problem string) string {
		return `!(` + waiting + `) && document.querySelector('.bd .is-problem').textContent === '` + problem + `'`
	}

	p.Eval(t, `(() => {
		const fetch = window.fetch
		window.fetch = (url, init) => String(url).includes('/cmd/board/move') && !window.passMove ? Promise.resolve(new Response(null, {status: 500})) : fetch(url, init)
		return true
	})()`, nil)
	move(4)
	p.Wait(t, `window.lastStatus === '500'`, 5*time.Second)
	time.Sleep(300 * time.Millisecond)
	var busy bool
	p.Eval(t, waiting, &busy)
	if !busy {
		t.Fatal("the 500 ended the pending move")
	}
	visibility(true) // Datastar closes the stream
	waitFor(t, "the stream to close", func() bool { return a.Hub().Count("board:board") == 0 })
	visibility(false) // and opens it again, without the move's ack
	p.Wait(t, ended("That move did not go through. Try again."), 10*time.Second)

	p.Eval(t, `(window.passMove = true)`, nil)
	visibility(true)
	waitFor(t, "the stream to close", func() bool { return a.Hub().Count("board:board") == 0 })
	move(5) // no open stream: a bare 400
	p.Wait(t, `window.lastStatus === '400'`, 5*time.Second)
	p.Wait(t, ended("That did not work. Try again."), 5*time.Second)
	for _, e := range p.Errors() {
		if !strings.HasPrefix(e, "error network:") {
			t.Errorf("page error: %s", e)
		}
	}
}
