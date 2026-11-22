// The Game of Life's controller, on the stage (data-life). The markup says
// which element answers which event (views.templ); this module implements
// the actions it calls there that take more than a line: a stroke's cells
// sent in batches as it is drawn, a pattern dropped where the board was
// clicked, the keyboard's cursor. It also ends what is pending. It reads
// the page by the names that dom.go shares with the templates and the
// frames.
//
// Nothing here changes the board. Every action goes to the server, and
// the next generation's frame, which the stream morphs into the board,
// shows it.

import { actions, clamp, commands, controller, measured, must, orderedOpId, pending, signals } from 'lith-ui'
import { type Actions, CellClasses, ClassCursor, ClassField, ClassMarks, type Commands, FlushMillis, H, IdBoard, IdRow, type Roots, StrokeBatch, ToolDraw, W } from './ds.gen'

controller('life', (el, scope) => {
	const s = signals<Roots>()._life
	// The strokes and the drops go out from the stage, which pending counts.
	const send = commands<Commands>(el)
	const field = must(el, `.${ClassField}`)

	// The field's box, and whether it stands on its side: on a field taller
	// than wide CSS turns the board, so a column on the screen is a row of
	// the board. The board's frames, the marks and the outlines change ten
	// times a second and never move the field: only a scroll or a resize
	// measures it again, so a pointer move forces no layout.
	const own = [must(field, `#${IdBoard}`), must(field, `.${ClassMarks}`), ...field.querySelectorAll(`.${ClassCursor}`)]
	const box = measured(field, scope, own, () => ({ r: field.getBoundingClientRect(), side: field.clientHeight > field.clientWidth }))

	/** Puts the cursor on the cell under the pointer. */
	const point = (evt: MouseEvent): void => {
		const { r, side } = box()
		const across = (evt.clientX - r.left) / r.width
		const down = (evt.clientY - r.top) / r.height
		s.x = clamp(Math.floor((side ? down : across) * W), 0, W - 1)
		s.y = clamp(Math.floor((side ? across : down) * H), 0, H - 1)
	}
	const cell = (): number => s.y * W + s.x
	/** The cell under the cursor as an SVG path: a pending mark. */
	const square = (): string => `M${s.x} ${s.y}h1v1h-1z`
	/** Whether the cell under the cursor lives, as the board's last frame shows it. */
	const lives = (): boolean => CellClasses.indexOf(document.getElementById(IdRow + s.y)?.children[s.x]?.className || '_') > 0
	/** Starts an action: a new op, and no problem from the last one. */
	const begin = (): string => {
		s.problem = ''
		s.op = orderedOpId()
		return s.op
	}

	// A stroke under way: its cells not sent yet, the last one sent, which
	// the next batch starts from so that the server joins them by a line,
	// and when it last sent.
	let drawing = false
	let stroke: number[] = []
	let last = -1
	let sent = 0

	/** Sends the stroke's cells not sent yet, which stay pending. Every batch must arrive: requestCancellation is off. */
	const flush = (): void => {
		if (stroke.length === 0) return
		const op = begin()
		s.pend += s.draft
		s.draft = ''
		send('life-draw', { op, cells: last >= 0 ? [last, ...stroke] : stroke, alive: s.alive })
		last = stroke[stroke.length - 1] ?? -1
		stroke = []
		sent = Date.now()
	}

	/** Drops the pattern at the cursor, and shows its outline pending there. */
	const drop = (): void => {
		const op = begin()
		s.px = s.x
		s.py = s.y
		s.ptool = s.tool
		s.pturn = s.turn
		send('life-drop', { op, pattern: s.tool, cell: cell(), turn: s.turn % 4 })
	}

	/** Flips the cell at the cursor: the keyboard's draw. */
	const toggle = (): void => {
		const op = begin()
		s.alive = !lives()
		s.pend += square()
		send('life-draw', { op, cells: [cell()], alive: s.alive })
	}

	// The keyboard's arrows move the cursor; the board wraps around, and on
	// its side they move along its rows and columns as they show.
	const arrows: Partial<Record<string, readonly [number, number]>> = { ArrowRight: [1, 0], ArrowLeft: [-1, 0], ArrowDown: [0, 1], ArrowUp: [0, -1] }
	const centre = (): void => {
		if (s.x >= 0) return
		s.x = W / 2
		s.y = H / 2
	}

	actions<Actions>(el, scope, {
		// A stroke starts where the pointer goes down, with the draw tool: it
		// clears where it starts on a live cell, and draws where it starts on
		// a dead one.
		lifeBegin({ evt }) {
			if (!(evt instanceof PointerEvent) || !evt.isPrimary || evt.button !== 0) return
			point(evt)
			field.setPointerCapture(evt.pointerId)
			drawing = true
			s.alive = !lives()
			last = -1
			stroke = [cell()]
			s.draft = square()
			sent = Date.now()
		},

		// The cursor follows the pointer; a stroke adds each new cell, and
		// sends once it holds StrokeBatch cells or FlushMillis went by.
		lifeMove({ evt }) {
			if (!(evt instanceof PointerEvent)) return
			point(evt)
			if (!drawing) return
			const c = cell()
			if (!stroke.includes(c)) {
				stroke.push(c)
				s.draft += square()
			}
			if (stroke.length >= StrokeBatch || Date.now() - sent > FlushMillis) flush()
		},

		// The stroke ends: the rest of it goes.
		lifeEnd() {
			if (!drawing) return
			flush()
			drawing = false
		},

		// The cursor goes when the pointer or the focus leaves the board,
		// unless a stroke holds the pointer.
		lifeAway() {
			if (!drawing) s.x = -1
		},

		// A click drops the pattern, with a pattern's tool, where the board
		// was clicked.
		lifeDrop({ evt }) {
			if (!(evt instanceof MouseEvent)) return
			point(evt)
			drop()
		},

		// Arrows move the cursor, Enter or Space draws or drops, R turns the
		// pattern, Escape hides the cursor.
		lifeKey({ evt }) {
			if (!(evt instanceof KeyboardEvent)) return
			const k = evt.key
			const step = arrows[k]
			if (step) {
				evt.preventDefault()
				centre()
				const [a, b] = box().side ? [step[1], step[0]] : step
				s.x = (s.x + a + W) % W
				s.y = (s.y + b + H) % H
			} else if (k === 'Enter' || k === ' ') {
				evt.preventDefault()
				centre()
				if (s.tool === ToolDraw) toggle()
				else drop()
			} else if (k === 'r' || k === 'R') {
				s.turn = s.turn + 1
			} else if (k === 'Escape') {
				s.x = -1
			}
		},
	})

	// What waits on the page: the newest op until the tab's view confirms
	// it, and a colour asked for until the session has it. It ends on
	// those, a request that never got through, or a stream that connects
	// again without them, never on a clock. The swatches and the reset post
	// from their own buttons, inside the stage: within counts them.
	pending(el, scope, {
		signals: s,
		root: '_life' satisfies keyof Roots,
		within: true,
		waiting: () => (s.op !== '' && !(s.ack >= s.op)) || s.pick !== s.colour,
		end: () => {
			s.op = ''
			s.pend = ''
			s.draft = ''
			s.ptool = ''
			s.resetting = false
			s.pick = s.colour
		},
		down: 'The connection is down. Try again once it is back.',
		failed: 'That did not work. Try again.',
		lost: 'That did not go through. Try again.',
	})
})
