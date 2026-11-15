// The board's controller: the drag, the keyboard and the pending move's
// looks, on the stage (data-board). It reads the page: the lanes carry
// data-lane and data-name, the cards data-card, data-version and
// data-title. On a narrow board a lane is a row of cards that wraps, and
// board.css says so with --bd-flow.
//
// Nothing here moves a card. It moves the page's signals ($_board, typed
// by Signals from signals.go) and its own, and dispatches the move, typed
// by MoveReq, which the stage posts; the board's next frame moves the
// card.

import { announcer, commands, controller, insertionIndex, local, must, nearest, num, opId, pending, pointerDrag, signals } from 'lith-ui'
import type { Commands, Roots, Signals } from './ds.gen'

type Mode = Signals['mode']

controller('board', (el, scope) => {
	// The page's signals: what the page's attributes and the server read.
	const s = signals<Roots>()._board
	// The controller's own, reactive for its effects but no part of the
	// page's signals, which the server could patch: the card in hand (0:
	// none), its version when it was picked up, the lane and place it came
	// from; where it would go (lane -1: nowhere) and how many other cards
	// that lane has; the card of the pending move, and a refused one.
	const h = local({ drag: 0, seen: 0, from: -1, at: 0, lane: -1, index: 0, n: 0, pcard: 0, nudge: 0 })
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
		[...lane(n).querySelectorAll<HTMLElement>('[data-card]')].filter((c) => num(c, 'card') !== (h.drag || h.pcard))
	const name = (n: number): string => lane(n).dataset['name'] ?? ''
	const title = (c: HTMLElement): string => c.dataset['title'] ?? ''
	const place = (n: number, index: number, count: number): string => `${name(n)}, ${index + 1} of ${count + 1}`

	// aim marks where the card would go: lane -1 is nowhere.
	const aim = (n: number, index: number): void => {
		if (n < 0) {
			h.lane = -1
			return
		}
		const count = others(n).length
		h.index = Math.max(0, Math.min(index, count))
		h.n = count
		h.lane = n
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
		h.drag = id
		s.mode = mode
		h.seen = num(c, 'version')
		h.from = from
		h.at = indexOf(c)
		aim(from, h.at)
		if (mode === 'keys') say(`Picked up “${title(c)}”: ${place(from, h.at, h.n)}. Arrows choose a place, Space drops it, Escape puts it back.`)
	}

	const cancel = (tell: boolean): void => {
		const c = card(h.drag)
		h.drag = 0
		s.mode = ''
		aim(-1, 0)
		if (tell && c) say(`“${title(c)}” stays in ${name(laneOf(c))}.`)
	}

	// drop sends the move in hand, with the card's version from when it
	// was picked up; a card dropped where it was goes back.
	const drop = (): void => {
		const c = card(h.drag)
		const to = h.lane
		const index = h.index
		if (!c || to < 0 || (to === h.from && index === h.at)) {
			cancel(true)
			return
		}
		const move = { op: opId(), card: h.drag, lane: to, index, seen: h.seen }
		h.pcard = h.drag
		h.drag = 0
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
				const n = Math.max(0, Math.min(lanes().length - 1, h.lane + step))
				aim(n, step ? h.index : h.index + (k === 'ArrowUp' ? -1 : 1))
				const home = n === h.from && h.index === h.at
				say(place(n, h.index, h.n) + (home ? ', where it is' : '') + '.')
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

	// The pending move ends on its outcome (settle, below), a request that
	// never got through, or a stream that connects again without its ack,
	// never on a clock.
	const root = '_board' satisfies keyof Roots
	pending(el, scope, {
		signals: s,
		root,
		end: () => {
			s.op = ''
			h.pcard = 0
			if (!h.drag) aim(-1, 0)
		},
		down: 'That move did not reach the server. Try again in a moment.',
		failed: 'That did not work. Try again.',
		lost: 'That move did not go through. Try again.',
		retrying: 'The server does not answer. Trying again.',
	})

	// settle ends the pending move once its outcome has come and the board
	// shows the version that has it. A refused card shakes for a moment.
	scope.effect(() => {
		const op = s.op
		if (!op || s.ack !== op || s.v < s.ackv) return
		const c = h.pcard
		if (!s.ok) {
			h.nudge = c
			const t = setTimeout(() => {
				if (h.nudge === c) h.nudge = 0
			}, 900)
			scope.defer(() => clearTimeout(t))
		}
		s.op = ''
		h.pcard = 0
		s.problem = ''
		if (!h.drag) aim(-1, 0)
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
		if (h.drag) rules.push(`#board-card-${h.drag}{${s.mode === 'keys' ? '--bd-pick:1' : '--bd-lift:1'}}`)
		if (h.pcard) rules.push(`#board-card-${h.pcard}{--bd-wait:1;--bd-busy:bd-busy}`)
		if (h.nudge) rules.push(`#board-card-${h.nudge}{--bd-nudge:bd-nudge}`)
		if (h.lane >= 0) {
			const at = `[data-lane="${h.lane}"]`
			rules.push(`${at}{--bd-target:1}`)
			const home = h.drag && h.lane === h.from && h.index === h.at
			if (!home) {
				const last = h.index >= h.n
				if (h.n === 0) rules.push(`${at} .bd-lane__cards{anchor-name:--bd-slot}.bd-marker{--bd-on:1;--bd-edge:0}`)
				else
					rules.push(
						`${at} .bd-slot:nth-${last ? 'last-child(1' : `child(${h.index + 1}`} of :not(#board-card-${h.drag || h.pcard})){anchor-name:--bd-slot}` +
							`.bd-marker{--bd-on:1;--bd-edge:${last ? 1 : 0}}`,
					)
			}
		}
		style.textContent = rules.join('')
	})
})
