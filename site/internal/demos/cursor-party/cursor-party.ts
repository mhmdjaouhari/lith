// The cursor party's controller, on the floor (data-cursor-party). It
// finds the anchors' boxes on this page, relative to the floor: those the
// page lacks or hides stand at their homes (Anchors, in AttrHomes). It
// hands them to CSS (--cp-<name>-x, -y, -w, -h on the floor, which the
// stream never patches), measures again when the floor or an anchor
// changes size, and implements the move the floor's pointer calls
// (views.templ): the anchor nearest to the pointer, and its place as
// fractions of that anchor's box.

import { actions, clamp, commands, controller, signals } from 'lith-ui'
import { type Actions, AttrAnchor, AttrHomes, type Commands, PropAnchor, type Roots } from './ds.gen'

/** A box on the floor: left, top, width, height, in px or, for a home, fractions of the floor. */
type Box = readonly [number, number, number, number]

controller('cursor-party', (el, scope) => {
	const s = signals<Roots>()._cursorParty
	// Moves go out from the floor, side by side: none cancels the one
	// before, none is retried, and the server keeps the one with the
	// highest number.
	const send = commands<Commands>(el)
	const homes = JSON.parse(el.getAttribute(AttrHomes) ?? '{}') as Record<string, Box>
	const names = Object.keys(homes)
	const find = (n: string): Element | null => document.querySelector(`[${AttrAnchor}="${CSS.escape(n)}"]`)
	let at: Record<string, Box> = {}

	const measure = (): void => {
		const f = el.getBoundingClientRect()
		const next: Record<string, Box> = {}
		for (const n of names) {
			const r = find(n)?.getBoundingClientRect()
			const h = homes[n] ?? [0, 0, 0, 0]
			const b: Box = r && r.width && r.height ? [r.left - f.left, r.top - f.top, r.width, r.height] : [h[0] * f.width, h[1] * f.height, h[2] * f.width, h[3] * f.height]
			next[n] = b
			;(['x', 'y', 'w', 'h'] as const).forEach((k, i) => el.style.setProperty(`${PropAnchor}${n}-${k}`, `${b[i]}px`))
		}
		at = next
	}
	measure()
	const ro = new ResizeObserver(measure)
	ro.observe(el)
	for (const n of names) {
		const a = find(n)
		if (a) ro.observe(a)
	}
	scope.defer(() => ro.disconnect())

	/** A fraction of an anchor's box, to four places, within -20 and 20. */
	const fraction = (v: number): number => clamp(Math.round(v * 1e4) / 1e4, -20, 20)

	actions<Actions>(el, scope, {
		// Where the pointer is, when it is over the floor.
		cursorPartyMove({ evt }) {
			if (!(evt instanceof PointerEvent) || !evt.isPrimary) return
			const f = el.getBoundingClientRect()
			const px = evt.clientX - f.left
			const py = evt.clientY - f.top
			if (px < 0 || py < 0 || px > f.width || py > f.height) return
			let best = ''
			let d = Infinity
			for (const [n, b] of Object.entries(at)) {
				const dx = Math.max(b[0] - px, 0, px - b[0] - b[2])
				const dy = Math.max(b[1] - py, 0, py - b[1] - b[3])
				if (dx * dx + dy * dy < d) {
					d = dx * dx + dy * dy
					best = n
				}
			}
			const b = at[best]
			if (!b) return
			s.seq++
			send('cursor-party-move', { a: best, x: fraction((px - b[0]) / b[2]), y: fraction((py - b[1]) / b[3]), seq: s.seq })
		},
	})
})
