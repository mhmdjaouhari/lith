// The board's controller: the drag, the keyboard and the pending move's
// looks, on the stage (data-board). It reads the page: the lanes carry
// data-lane and data-name, the cards data-card, data-version and
// data-title. On a narrow board a lane is a row of cards that wraps, and
// board.css says so with --bd-flow.
//
// Nothing here moves a card. It moves the local signals ($_board, typed
// by Signals from signals.go) and dispatches the move, typed by MoveReq,
// which the stage posts; the board's next frame moves the card.

import { announcer, commands, controller, insertionIndex, must, nearest, num, opId, pointerDrag, signals } from 'lith-ui'
import type { Commands, Roots, Signals } from './ds.gen'

type Mode = Signals['mode']

controller('board', (el, scope) => {
	const s = signals<Roots>()._board
	const say = announcer(() => s.say, (t) => (s.say = t))
	const send = commands<Commands>(el)

	const lane = (n: number): HTMLElement => must(el, `[data-lane="${n}"]`)
	const lanes = (): HTMLElement[] => [...el.querySelectorAll<HTMLElement>('[data-lane]')]
	const card = (id: number): HTMLElement | null => document.getElementById('board-card-' + id)
	const laneOf = (c: HTMLElement): number => {
		const l = c.closest<HTMLElement>('[data-lane]')
		if (!l) throw new Error('board: a card outside a lane')
		return num(l, 'lane')
	}
	const indexOf = (c: HTMLElement): number => (c.parentElement ? [...c.parentElement.children].indexOf(c) : 0)
	const others = (n: number): HTMLElement[] =>
		[...lane(n).querySelectorAll<HTMLElement>('[data-card]')].filter((c) => num(c, 'card') !== (s.drag || s.pcard))
	const name = (n: number): string => lane(n).dataset['name'] ?? ''
	const title = (c: HTMLElement): string => c.dataset['title'] ?? ''
	const place = (n: number, index: number, count: number): string => `${name(n)}, ${index + 1} of ${count + 1}`

	// aim marks where the card would go: lane -1 is nowhere.
	const aim = (n: number, index: number): void => {
		if (n < 0) {
			s.lane = -1
			return
		}
		const count = others(n).length
		s.index = Math.max(0, Math.min(index, count))
		s.n = count
		s.lane = n
	}

	const aimAt = (x: number, y: number): void => {
		const box = must(el, '.bd-lanes').getBoundingClientRect()
		if (x < box.left - 48 || x > box.right + 48 || y < box.top - 64 || y > box.bottom + 64) {
			aim(-1, 0)
			return
		}
		const near = nearest(lanes(), x, y)
		if (!near) return
		const n = num(near, 'lane')
		const wraps = getComputedStyle(near).getPropertyValue('--bd-flow').trim() === '1'
		aim(n, insertionIndex(others(n), x, y, wraps))
	}

	const lift = (id: number, mode: Mode): void => {
		const c = card(id)
		if (!c) return
		const from = laneOf(c)
		s.drag = id
		s.mode = mode
		s.seen = num(c, 'version')
		s.from = from
		s.at = indexOf(c)
		aim(from, s.at)
		if (mode === 'keys') say(`Picked up “${title(c)}”: ${place(from, s.at, s.n)}. Arrows choose a place, Space drops it, Escape puts it back.`)
	}

	const cancel = (tell: boolean): void => {
		const c = card(s.drag)
		s.drag = 0
		s.mode = ''
		aim(-1, 0)
		if (tell && c) say(`“${title(c)}” stays in ${name(laneOf(c))}.`)
	}

	// drop sends the move in hand, with the card's version from when it
	// was picked up; a card dropped where it was goes back.
	const drop = (): void => {
		const c = card(s.drag)
		const to = s.lane
		const index = s.index
		if (!c || to < 0 || (to === s.from && index === s.at)) {
			cancel(true)
			return
		}
		const move = { op: opId(), card: s.drag, lane: to, index, seen: s.seen }
		s.pcard = s.drag
		s.drag = 0
		s.mode = ''
		s.problem = ''
		s.op = move.op
		say(`Moving “${title(c)}” to ${name(to)}.`)
		send('board-move', move)
	}

	// walk moves the focus from card to card with the arrows.
	const walk = (c: HTMLElement, key: string): void => {
		let n = laneOf(c)
		const i = indexOf(c)
		let next: Element | null = null
		if (key === 'ArrowUp') next = c.previousElementSibling
		if (key === 'ArrowDown') next = c.nextElementSibling
		if (key === 'ArrowLeft' || key === 'ArrowRight') {
			const step = key === 'ArrowLeft' ? -1 : 1
			for (n += step; !next && n >= 0 && el.querySelector(`[data-lane="${n}"]`); n += step) {
				const cards = lane(n).querySelectorAll('[data-card]')
				next = cards[Math.min(i, cards.length - 1)] ?? null
			}
		}
		next?.querySelector('button')?.focus()
	}

	// The pointer: the main button of a mouse or pen anywhere on a card, a
	// finger on its grip (the rest of the card scrolls the page).
	const drag = pointerDrag(el, scope, {
		items: '[data-card]',
		handle: '.bd-card__grip',
		ghost: must(el, '.bd-ghost'),
		copy: (c) => must(c, '.bd-card').cloneNode(true),
		enabled: () => !s.op,
		pressed: () => {
			if (s.mode === 'keys') cancel(false)
		},
		lift: (c) => lift(num(c, 'card'), 'pointer'),
		aim: aimAt,
		drop,
		cancel: () => cancel(false),
	})

	// The keyboard: on a card, Space or Enter picks it up and the arrows go
	// from card to card; picked up, the arrows choose a place, Space or
	// Enter drops it there and Escape or Tab puts it back.
	scope.listen(el, 'keydown', (evt) => {
		const k = evt.key
		const arrow = k.startsWith('Arrow')
		if (s.mode === 'pointer') {
			if (k === 'Escape') {
				evt.preventDefault()
				drag.abort()
			}
			return
		}
		if (s.mode === 'keys') {
			if (arrow) {
				evt.preventDefault()
				const step = k === 'ArrowLeft' ? -1 : k === 'ArrowRight' ? 1 : 0
				const n = Math.max(0, Math.min(lanes().length - 1, s.lane + step))
				aim(n, step ? s.index : s.index + (k === 'ArrowUp' ? -1 : 1))
				const home = n === s.from && s.index === s.at
				say(place(n, s.index, s.n) + (home ? ', where it is' : '') + '.')
			} else if (k === ' ' || k === 'Enter') {
				evt.preventDefault()
				drop()
			} else if (k === 'Escape' || k === 'Tab') {
				if (k === 'Escape') evt.preventDefault()
				cancel(true)
			}
			return
		}
		const c = evt.target instanceof Element ? evt.target.closest<HTMLElement>('[data-card]') : null
		if (!c || evt.altKey || evt.ctrlKey || evt.metaKey) return
		if (k === ' ' || k === 'Enter') {
			evt.preventDefault()
			if (s.op) say('One move at a time: the last one is on its way.')
			else lift(num(c, 'card'), 'keys')
		} else if (arrow) {
			evt.preventDefault()
			walk(c, k)
		}
	})

	// A card picked up from the keyboard goes back when the focus goes
	// elsewhere: a click outside the board, another window.
	scope.listen(el, 'focusout', (evt) => {
		const to = evt.relatedTarget
		if (s.mode === 'keys' && !(to instanceof Node && el.contains(to))) cancel(true)
	})

	// settle ends the pending move once its outcome has come and the board
	// shows the version that has it. A refused card shakes for a moment.
	scope.effect(() => {
		const op = s.op
		if (!op || s.ack !== op || s.v < s.ackv) return
		const c = s.pcard
		if (!s.ok) {
			s.nudge = c
			const t = setTimeout(() => {
				if (s.nudge === c) s.nudge = 0
			}, 900)
			scope.defer(() => clearTimeout(t))
		}
		s.op = ''
		s.pcard = 0
		s.problem = ''
		if (!s.drag) aim(-1, 0)
	})

	// looks is the style of the card in hand (--bd-lift under the pointer,
	// which leaves a hole behind its ghost; --bd-pick from the keyboard,
	// which rises), of a card pending or refused, and of the place the card
	// would go. It lives outside the regions the stream morphs, so no frame
	// takes it away. It sets custom properties and the marker's anchor;
	// board.css draws them. The marker sits above the card that would
	// follow the moved one, below the lane's last card, or at the top of an
	// empty lane, and stays hidden while the place is the card's own.
	const style = must<HTMLStyleElement>(el, 'style[data-bd-looks]')
	scope.effect(() => {
		const rules: string[] = []
		if (s.drag) rules.push(`#board-card-${s.drag}{${s.mode === 'keys' ? '--bd-pick:1' : '--bd-lift:1'}}`)
		if (s.pcard) rules.push(`#board-card-${s.pcard}{--bd-wait:1;--bd-busy:bd-busy}`)
		if (s.nudge) rules.push(`#board-card-${s.nudge}{--bd-nudge:bd-nudge}`)
		if (s.lane >= 0) {
			const at = `[data-lane="${s.lane}"]`
			rules.push(`${at}{--bd-target:1}`)
			const home = s.drag && s.lane === s.from && s.index === s.at
			if (!home) {
				const last = s.index >= s.n
				if (s.n === 0) rules.push(`${at} .bd-lane__cards{anchor-name:--bd-slot}.bd-marker{--bd-on:1;--bd-edge:0}`)
				else
					rules.push(
						`${at} .bd-slot:nth-${last ? 'last-child(1' : `child(${s.index + 1}`} of :not(#board-card-${s.drag || s.pcard})){anchor-name:--bd-slot}` +
							`.bd-marker{--bd-on:1;--bd-edge:${last ? 1 : 0}}`,
					)
			}
		}
		style.textContent = rules.join('')
	})
})
