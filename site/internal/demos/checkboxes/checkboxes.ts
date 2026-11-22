// The million checkboxes' controller, on the stage (data-checkboxes). The
// markup says which element answers which event (views.templ); this
// module implements the actions it calls there that take more than a
// line: a click sent as the state the box should take, the box kept
// waiting, indeterminate, until the frame that flips it or the ack of its
// op; the grid's measure, the windows the list asks for, the jump to a
// box and the arrows between boxes. It also answers a stream's first
// frame with the window, and ends what is pending. It reads the page by
// the names that dom.go shares with the templates and the tiles.
//
// Nothing here checks a box. The click goes to the server, and the tile's
// next frame, which the stream morphs into its slot, flips it.

import { actions, clamp, commands, controller, grouped, must, orderedOpId, pending, signals } from 'lith-ui'
import { type Actions, AttrChunk, AttrGreet, AttrWindow, Boxes, ChunkSize, ClassBoxes, ClassFrame, type Commands, IdBoard, IdGrid, IdTile, PropChrome, type Roots } from './ds.gen'

/** The sb-virtual-scroll host: its scrollToIndex, defined once it upgrades. */
interface VirtualScroll extends HTMLElement {
	scrollToIndex?: (index: number) => void
}

/** The window sb-virtual-scroll asks for as it scrolls: count items from offset. */
type WindowEvent = CustomEvent<{ offset: number; count: number }>

/** A box's click on its way: its op, and how many clicks the box sent. */
interface Sent {
	op: string
	n: number
}

controller('checkboxes', (el, scope) => {
	const s = signals<Roots>()._checkboxes
	// The clicks go out from the stage, which pending counts; the windows
	// from elements of their own.
	const send = commands<Commands>(el)
	const sendWindow = commands<Commands>(must(el, `[${AttrWindow}]`))
	const greet = commands<Commands>(must(el, `[${AttrGreet}]`))
	const board = must(el, `#${IdBoard}`)
	// The board view morphs the list in place, by its id.
	const grid = (): VirtualScroll => must<VirtualScroll>(board, `#${IdGrid}`)
	const boxes = `.${ClassBoxes} > input`
	const isBox = (t: EventTarget | null | undefined): t is HTMLInputElement => t instanceof HTMLInputElement && t.matches(boxes)
	const waiting = (): NodeListOf<HTMLInputElement> => el.querySelectorAll<HTMLInputElement>(`${boxes}:indeterminate`)
	const sent = new WeakMap<HTMLInputElement, Sent>()
	/** The list's window as it shows it: its offset and how many slots it holds. */
	const shown = (): { offset: number; count: number } => {
		const g = grid()
		return { offset: Number(g.getAttribute('offset')), count: g.children.length }
	}

	/** The box's number from 0: its tile's chunk and its place in the tile. */
	const numberOf = (b: HTMLInputElement): number => {
		const tile = b.closest(`[${AttrChunk}]`)
		if (!tile || !b.parentElement) throw new Error('checkboxes: a box outside a tile')
		return Number(tile.getAttribute(AttrChunk)) * ChunkSize + Array.prototype.indexOf.call(b.parentElement.children, b)
	}

	// The grid fits the width it has: two tiles to a row from 600px, boxes
	// 16 to 24px apart. The tiles take the pitch from --cb-pitch, the list
	// its row height and tiles per row from attributes that follow these
	// signals.
	const frame = must(el, `.${ClassFrame}`)
	const measure = (): void => {
		const w = frame.clientWidth
		const cols = w >= 600 ? 2 : 1
		const chrome = parseFloat(getComputedStyle(el).getPropertyValue(PropChrome)) || 34
		s.pitch = clamp(Math.floor((w / cols - 12) / 16), 16, 24)
		s.cols = cols
		s.item = 16 * s.pitch + chrome
	}
	measure()

	// The jump's box, until it has the focus: focusTarget gives it the
	// focus once its tile sits in its place, as tiles and windows arrive.
	let target = 0
	const focusTarget = (): void => {
		const n = target - 1
		if (n < 0) return
		const c = Math.floor(n / ChunkSize)
		const g = grid()
		const tile = document.getElementById(IdTile + c)
		const slot = tile?.parentElement
		if (!tile || !slot || slot.parentElement !== g) return
		// The list has not laid out its new window yet: no focus yet.
		if (shown().offset + Array.prototype.indexOf.call(g.children, slot) !== c) return
		const box = tile.querySelector(`.${ClassBoxes}`)?.children[n % ChunkSize]
		if (box instanceof HTMLElement) requestAnimationFrame(() => box.focus({ focusVisible: true, preventScroll: true }))
		target = 0
	}
	const arrived = new MutationObserver(focusTarget)
	arrived.observe(board, { childList: true, subtree: true, attributes: true, attributeFilter: ['offset'] })
	scope.defer(() => arrived.disconnect())

	// The arrows move the focus between the boxes: left and right along
	// them, up and down a row of 16, and on into the next tile of the
	// window, the one beside or the one above or below.
	const steps: Partial<Record<string, number>> = { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -16, ArrowDown: 16 }

	actions<Actions>(el, scope, {
		// A click sends the state the box should take, the opposite of what
		// it shows, with an op id, and marks the box pending until the frame
		// that flips it or the ack of its op. The browser has toggled the box
		// by now: preventDefault puts it back, so it keeps showing what the
		// server said.
		checkboxesClick({ evt }) {
			const b = evt?.target
			if (!evt || !isBox(b)) return
			evt.preventDefault()
			const want = b.checked
			const op = orderedOpId()
			const n = (sent.get(b)?.n ?? 0) + 1
			sent.set(b, { op, n })
			s.problem = ''
			s.op = op
			// Pending: indeterminate, unless the frame came first.
			setTimeout(() => {
				if (b.checked !== want && sent.get(b)?.n === n) b.indeterminate = true
			})
			send('checkboxes-set', { op, box: numberOf(b), checked: want })
		},

		// A frame that flips a box ends its wait. The morph never touches
		// indeterminate, so a frame that changes another box of the tile
		// leaves the wait alone.
		checkboxesFlipped({ evt }) {
			const b = evt?.target
			if (isBox(b)) b.indeterminate = false
		},

		checkboxesKey({ evt }) {
			const b = evt?.target
			if (!(evt instanceof KeyboardEvent) || !isBox(b) || !b.parentElement) return
			const d = steps[evt.key]
			if (!d) return
			const tiles = [...grid().querySelectorAll(`.${ClassBoxes}`)]
			const box = b.parentElement
			const i = Array.prototype.indexOf.call(box.children, b)
			const k = tiles.indexOf(box)
			let to = box.children[i + d]
			if (Math.abs(d) === 1 && !to) {
				const t = tiles[k + d]
				to = (d > 0 ? t?.firstElementChild : t?.lastElementChild) ?? undefined
			} else if (Math.abs(d) === 16 && (i + d < 0 || i + d >= box.children.length)) {
				const t = tiles[k + Math.sign(d) * s.cols]
				to = t?.children[Math.min((i % 16) + (d > 0 ? 0 : t.children.length - 16), t.children.length - 1)]
			}
			if (!(to instanceof HTMLElement)) return
			evt.preventDefault()
			to.focus()
		},

		// The list asks for a window as it scrolls: the server stores it and
		// moves the tab's stream to its chunks; the tab's view sends the list
		// at its new offset.
		checkboxesWindow({ evt }) {
			const { offset, count } = (evt as WindowEvent).detail
			sendWindow('checkboxes-window', { offset, count, cols: s.cols })
		},

		checkboxesMeasure: measure,

		// The jump scrolls the list to the tile of the box asked for.
		checkboxesJump() {
			const n = Math.round(Number(s.jump))
			if (!(n >= 1 && n <= Boxes)) {
				s.problem = `Pick a box from 1 to ${grouped(Boxes)}.`
				return
			}
			s.problem = ''
			target = n
			grid().scrollToIndex?.(Math.floor((n - 1) / ChunkSize))
			focusTarget()
		},
	})

	// The ack ends the wait of the boxes whose clicks the server took: a
	// frame that changes nothing (someone set the box back in the same
	// tick) ends it too. The stage's settle ends the op itself.
	scope.effect(() => {
		const ack = s.ack
		if (!ack) return
		for (const b of waiting()) if ((sent.get(b)?.op ?? '') <= ack) b.indeterminate = false
	})

	// A stream's first frame sets hello: the page answers with the window
	// the list shows, so that the new stream watches its chunks.
	scope.effect(() => {
		if (!s.hello) return
		s.hello = false
		greet('checkboxes-window', { ...shown(), cols: s.cols })
	})

	// The clicks end on their ack, a request that never got through, or a
	// stream that connects again without the ack, never on a clock.
	pending(el, scope, {
		signals: s,
		root: '_checkboxes' satisfies keyof Roots,
		waiting: () => s.op !== '' && !(s.ack >= s.op),
		end: () => {
			for (const b of waiting()) b.indeterminate = false
			s.op = ''
		},
		down: 'That click did not reach the server. Try again once the connection is back.',
		failed: 'That did not work. Try again.',
		lost: 'That click did not go through. Try again.',
	})
})
