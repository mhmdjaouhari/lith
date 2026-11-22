// The town's controller, on the stage (data-town). The markup says which
// element answers which event (views.templ); this module implements the
// actions it calls there that take more than a line: a change of the tile
// clicked, the arrows between tiles, and the slider's seeks, each with a
// number, so that the server drops one the network delayed past a newer
// one. It also keeps the slider's ends, and marks the tile whose change is
// on its way. It reads the page by the names that dom.go shares with the
// templates and the frames.
//
// Nothing here changes a tile. The change goes to the server, and the
// map's next frame, which the stream morphs, shows it.

import { actions, attr, commands, controller, must, num, opId, Sheet, signals } from 'lith-ui'
import { type Actions, AttrTile, AttrVersion, ClassBoard, ClassSlider, ClassTime, Cols, type Commands, IdMap, PropWait, type Roots, Tiles } from './ds.gen'

controller('town', (el, scope) => {
	const s = signals<Roots>()._town
	// The changes go out from the map's frame, the seeks from the slider's
	// row: each has its own pending op, which counts its element's
	// requests.
	const board = must(el, `.${ClassBoard}`)
	const place = commands<Commands>(board)
	const time = must(el, `.${ClassTime}`)
	const send = commands<Commands>(time)
	const slider = must<HTMLInputElement>(time, `.${ClassSlider}`)
	const tileOf = (t: EventTarget | null | undefined): HTMLElement | null => (t instanceof Element ? t.closest<HTMLElement>(`[${AttrTile}]`) : null)

	/** Seeks the position want: at the slider's end, now (0). */
	const seek = (want: number): void => {
		s.want = want
		s.mine++
		send('town-seek', { at: want >= s.head ? 0 : want, seq: s.mine })
	}

	// The slider's ends follow the head view's positions, and the thumb
	// stays at the end while the tab is live and no seek is under way: then
	// new events move it along. The ends go first, so that the value is not
	// clamped to the old end.
	scope.effect(() => {
		slider.min = String(s.first)
		slider.max = String(s.head)
		if (s.at === 0 && s.mine === s.seq) s.want = s.head
	})

	// The tile whose change is on its way waits, marked: a rule of a sheet
	// the controller owns, since the stream morphs the tiles.
	const sheet = new Sheet<typeof PropWait>(scope)
	scope.effect(() => {
		sheet.set(s.ptile < 0 ? [] : [[`#${IdMap} ${attr(AttrTile, s.ptile)}`, { [PropWait]: 1 }]])
	})

	actions<Actions>(el, scope, {
		// A click sends a change of the tile to the tool, with the tile's
		// version on the page, unless the tab shows the past or a change is
		// under way. The tile waits until the tab's view brings the change's
		// outcome back (settle).
		townPlace({ evt }) {
			const b = tileOf(evt?.target)
			if (!b || s.at !== 0 || s.op) return
			s.problem = ''
			s.ptile = num(b, AttrTile)
			s.op = opId()
			place('town-place', { op: s.op, tile: s.ptile, kind: s.tool, seen: num(b, AttrVersion) })
		},

		// The arrows move the focus from tile to tile.
		townKey({ evt }) {
			const b = tileOf(evt?.target)
			if (!(evt instanceof KeyboardEvent) || !b || !evt.key.startsWith('Arrow')) return
			const t = num(b, AttrTile)
			const c = t % Cols
			const k = evt.key
			const to = k === 'ArrowLeft' ? (c > 0 ? t - 1 : -1) : k === 'ArrowRight' ? (c < Cols - 1 ? t + 1 : -1) : k === 'ArrowUp' ? t - Cols : t + Cols
			if (to < 0 || to >= Tiles) return
			evt.preventDefault()
			el.querySelector<HTMLElement>(attr(AttrTile, to))?.focus()
		},

		// The slider moved: its own value, which data-bind may not have
		// handed to the signal yet.
		townDrag() {
			seek(Number(slider.value))
		},

		townNow() {
			seek(s.head)
		},
	})
})
