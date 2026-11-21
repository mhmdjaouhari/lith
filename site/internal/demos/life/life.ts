// The Game of Life's controller, on the stage (data-life). It follows the
// pointer and the keyboard's cursor over the board, sends a stroke's cells
// in batches as it is drawn, drops patterns, picks the colour, resets the
// board, and keeps what was sent pending, drawn over the board, until the
// tab's view confirms it. It reads the page by the names that dom.go
// shares with the templates and the frames.
//
// Nothing here changes the board. Every action goes to the server, and
// the next generation's frame, which the stream morphs into the board,
// shows it.

import { clamp, commands, controller, measured, must, num, orderedOpId, pending, signals } from 'lith-ui'
import { AttrColour, AttrReset, CellClasses, ClassCursor, ClassField, ClassMarks, type Commands, FlushMillis, H, IdBoard, IdRow, type Roots, StrokeBatch, W } from './ds.gen'

controller('life', (el, scope) => {
	const s = signals<Roots>()._life
	// Every action goes out from the stage, which pending counts.
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

	// A stroke starts where the pointer goes down: it clears where it
	// starts on a live cell, and draws where it starts on a dead one.
	scope.listen(field, 'pointerdown', (evt) => {
		if (s.tool !== 'draw' || !evt.isPrimary || evt.button !== 0) return
		point(evt)
		field.setPointerCapture(evt.pointerId)
		drawing = true
		s.alive = !lives()
		last = -1
		stroke = [cell()]
		s.draft = square()
		sent = Date.now()
	})

	// The cursor follows the pointer; a stroke adds each new cell, and
	// sends once it holds StrokeBatch cells or FlushMillis went by.
	scope.listen(field, 'pointermove', (evt) => {
		point(evt)
		if (!drawing) return
		const c = cell()
		if (!stroke.includes(c)) {
			stroke.push(c)
			s.draft += square()
		}
		if (stroke.length >= StrokeBatch || Date.now() - sent > FlushMillis) flush()
	})

	// The stroke ends: the rest of it goes.
	const end = (): void => {
		if (!drawing) return
		flush()
		drawing = false
	}
	scope.listen(field, 'pointerup', end)
	scope.listen(field, 'pointercancel', end)
	scope.listen(field, 'lostpointercapture', end)

	// The cursor goes when the pointer or the focus leaves the board.
	const away = (): void => {
		if (!drawing) s.x = -1
	}
	scope.listen(field, 'pointerleave', away)
	scope.listen(field, 'blur', away)

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

	// A click drops the pattern where the board was clicked.
	scope.listen(field, 'click', (evt) => {
		if (s.tool === 'draw') return
		point(evt)
		drop()
	})

	// The keyboard: the arrows move the cursor (the board wraps around,
	// and on its side they move along its rows and columns as they show),
	// Enter or Space draws or drops, R turns the pattern, Escape hides the
	// cursor.
	const arrows: Partial<Record<string, readonly [number, number]>> = { ArrowRight: [1, 0], ArrowLeft: [-1, 0], ArrowDown: [0, 1], ArrowUp: [0, -1] }
	const centre = (): void => {
		if (s.x >= 0) return
		s.x = W / 2
		s.y = H / 2
	}
	scope.listen(field, 'keydown', (evt) => {
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
			if (s.tool === 'draw') toggle()
			else drop()
		} else if (k === 'r' || k === 'R') {
			s.turn = s.turn + 1
		} else if (k === 'Escape') {
			s.x = -1
		}
	})

	// A swatch asks for its colour, which stays pending until the tab's view
	// brings it back as colour; the reset asks to clear the board.
	scope.listen(el, 'click', (evt) => {
		const t = evt.target instanceof Element ? evt.target : null
		const swatch = t?.closest(`[${AttrColour}]`)
		if (swatch) {
			const colour = num(swatch, AttrColour)
			s.pick = colour
			s.problem = ''
			send('life-colour', { colour })
		} else if (t?.closest(`[${AttrReset}]`) && !s.resetting) {
			const op = begin()
			s.resetting = true
			send('life-reset', { op })
		}
	})

	// The tab's view confirms the newest op, and with it the older ones (op
	// ids grow with time): what was pending goes, and the board's frame
	// shows it.
	scope.effect(() => {
		if (s.op === '' || !(s.ack >= s.op)) return
		s.op = ''
		s.pend = ''
		s.ptool = ''
		s.resetting = false
	})

	// What waits on the page: the newest op until the tab's view confirms
	// it, and a colour asked for until the session has it. It ends on
	// those, a request that never got through, or a stream that connects
	// again without them, never on a clock.
	pending(el, scope, {
		signals: s,
		root: '_life' satisfies keyof Roots,
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
