// lith-ui: what the site's controllers share. A controller is the code of
// a demo's stage that a Datastar expression would hold badly: a drag, the
// keyboard, an effect that draws from many signals. It lives in a module
// that tsc checks, typed by the interfaces dsgen writes from the demo's Go
// structs, and reaches the page through Datastar's own API: its signals,
// its effects, and an attribute plugin that mounts it on its element.
// The server stays the only one that changes state: a controller only
// moves local signals and dispatches commands, which the page posts.

import { attribute, beginBatch, effect, endBatch, root } from 'datastar'

/** The page's signals, typed by the Roots that dsgen writes: signals<Roots>()._board. */
export function signals<R extends object>(): R {
	return root as unknown as R
}

/** Runs fn with the effects of its writes held until it returns, as Datastar runs an expression. */
export function batch<T>(fn: () => T): T {
	beginBatch()
	try {
		return fn()
	} finally {
		endBatch()
	}
}

/** What a controller set up, undone when its element goes: listeners and effects. */
export class Scope {
	#undo: (() => void)[] = []

	/** Adds a listener, run in a batch, removed with the scope. */
	listen<K extends keyof HTMLElementEventMap>(target: HTMLElement, type: K, fn: (evt: HTMLElementEventMap[K]) => void, opts?: AddEventListenerOptions): void
	listen<K extends keyof WindowEventMap>(target: Window, type: K, fn: (evt: WindowEventMap[K]) => void, opts?: AddEventListenerOptions): void
	listen(target: EventTarget, type: string, fn: (evt: Event) => void, opts?: AddEventListenerOptions): void {
		const run = (evt: Event): void => batch(() => fn(evt))
		target.addEventListener(type, run, opts)
		this.#undo.push(() => target.removeEventListener(type, run, opts))
	}

	/** Runs fn now and whenever a signal it read changes, until the scope ends. */
	effect(fn: () => void): void {
		this.#undo.push(effect(fn))
	}

	/** Adds what to undo when the scope ends. */
	defer(fn: () => void): void {
		this.#undo.push(fn)
	}

	dispose(): void {
		for (const fn of this.#undo.splice(0).reverse()) fn()
	}
}

/**
 * Mounts setup on every element with data-<name>, once the element's
 * other attributes, its data-signals first, have applied. What setup
 * adds to its scope goes with the element.
 */
export function controller(name: string, setup: (el: HTMLElement, scope: Scope) => void): void {
	attribute({
		name,
		requirement: { key: 'denied', value: 'denied' },
		apply({ el }) {
			const scope = new Scope()
			setup(el, scope)
			return () => scope.dispose()
		},
	})
}

/**
 * Returns send, which dispatches a command with its payload on el, for
 * the element's data-on:<name> to post (demo.Command.On) with the tab id.
 * C is dsgen's Commands, so the name and the payload, every key and none
 * other, are checked against the Go payload: commands<Commands>(el).
 */
export function commands<C extends object>(el: HTMLElement): <K extends keyof C & string, D extends C[K]>(name: K, detail: Exact<D, C[K]>) => void {
	return (name, detail) => {
		el.dispatchEvent(new CustomEvent(name, { detail }))
	}
}

/** D when it has no key that T lacks: a payload with a misspelt key fails, a literal or not. */
export type Exact<D, T> = D & Record<Exclude<keyof D, keyof T>, never>

/**
 * Returns say, which puts text in a live region's signal. The same text
 * twice gets a no-break space, so the screen reader reads it again.
 */
export function announcer(get: () => string, set: (text: string) => void): (text: string) => void {
	return (text) => set(get() === text ? text + ' ' : text)
}

/** A new op id: unique enough for one tab, and what demo.ValidID accepts. */
export function opId(): string {
	return Date.now().toString(36) + Math.random().toString(36).slice(2, 7)
}

/** The value of a data- attribute as a number; a missing one is an error in the page. */
export function num(el: HTMLElement, key: string): number {
	const v = el.dataset[key]
	if (v === undefined || v === '' || Number.isNaN(+v)) throw new Error(`lith-ui: data-${key} is no number`)
	return +v
}

/** The element of a selector inside root; a missing one is an error in the page. */
export function must<T extends Element = HTMLElement>(root: ParentNode, selector: string): T {
	const e = root.querySelector<T>(selector)
	if (!e) throw new Error(`lith-ui: no ${selector}`)
	return e
}

/** Whether the point is inside the box. */
export function within(r: DOMRect, x: number, y: number): boolean {
	return x >= r.left && x < r.right && y >= r.top && y < r.bottom
}

/** The square of the distance from the point to the box, 0 inside it. */
export function distance2(r: DOMRect, x: number, y: number): number {
	const dx = x < r.left ? r.left - x : x > r.right ? x - r.right : 0
	const dy = y < r.top ? r.top - y : y > r.bottom ? y - r.bottom : 0
	return dx * dx + dy * dy
}

/** The element whose box is nearest the point. */
export function nearest<T extends Element>(els: Iterable<T>, x: number, y: number): T | undefined {
	let best: T | undefined
	let d = Infinity
	for (const e of els) {
		const n = distance2(e.getBoundingClientRect(), x, y)
		if (n < d) [best, d] = [e, n]
	}
	return best
}

/**
 * Where an item dropped at the point would go among items: before the
 * first item below it, or, in a list that wraps (a row of cards), before
 * the first one below it or to its right. items.length: at the end.
 */
export function insertionIndex(items: readonly Element[], x: number, y: number, wraps: boolean): number {
	const i = items.findIndex((c) => {
		const r = c.getBoundingClientRect()
		return wraps ? y < r.top || (y < r.bottom && x < r.left + r.width / 2) : y < r.top + r.height / 2
	})
	return i < 0 ? items.length : i
}

export interface PointerDragOptions {
	/** The items that can be dragged, inside the element. */
	items: string
	/** A touch starts a drag only on an item's handle; the rest of it scrolls the page. */
	handle?: string
	/** The element that follows the pointer while it drags, holding a copy of the item. */
	ghost: HTMLElement
	/** What the ghost shows of an item; the item itself by default. */
	copy?: (item: HTMLElement) => Node
	/** How far a mouse or pen moves before a press becomes a drag (px); a finger's is at once. */
	threshold?: number
	/** False keeps presses from starting a drag. */
	enabled?: () => boolean
	/** Called on a press that will drag, before anything else. */
	pressed?: () => void
	lift(item: HTMLElement): void
	aim(x: number, y: number): void
	drop(): void
	cancel(): void
}

export interface PointerDrag {
	/** Whether a pointer holds an item. */
	readonly dragging: boolean
	/** Ends the press, putting a lifted item back (Escape). */
	abort(): void
}

/**
 * A drag with the pointer inside el: the main button of a mouse or pen
 * anywhere on an item, a finger on its handle. el captures the pointer, so
 * the drag goes on outside it. Presses are heard on the window: while a
 * view transition runs, Chromium gives a press to what lies under the
 * transition's scope, or to the root element, so the item under the
 * pointer is found by its box when the target is not one.
 */
export function pointerDrag(el: HTMLElement, scope: Scope, o: PointerDragOptions): PointerDrag {
	type Press = { item: HTMLElement; pointer: number; x: number; y: number; dx: number; dy: number; lifted: boolean }
	let press: Press | null = null
	const threshold = o.threshold ?? 4

	const follow = (p: Press, x: number, y: number): void => {
		const box = el.getBoundingClientRect()
		o.ghost.style.translate = `${x - box.left - p.dx}px ${y - box.top - p.dy}px`
	}
	const lift = (p: Press): void => {
		p.lifted = true
		o.lift(p.item)
	}
	const ours = (evt: PointerEvent): Press | null => (press && evt.pointerId === press.pointer ? press : null)

	scope.listen(window, 'pointerdown', (evt) => {
		if (evt.button !== 0 || (o.enabled && !o.enabled())) return
		const target = evt.target
		if (!(target instanceof Element) || (target !== document.documentElement && !el.contains(target))) return
		const x = evt.clientX
		const y = evt.clientY
		const under = (e: Element | null): boolean => !!e && within(e.getBoundingClientRect(), x, y)
		const hit = target.closest<HTMLElement>(o.items)
		const item = hit && el.contains(hit) ? hit : [...el.querySelectorAll<HTMLElement>(o.items)].find(under)
		if (!item) return
		const touch = evt.pointerType === 'touch'
		if (touch && o.handle && !(hit ? target.closest(o.handle) : under(item.querySelector(o.handle)))) return
		o.pressed?.()
		const r = item.getBoundingClientRect()
		const p: Press = { item, pointer: evt.pointerId, x, y, dx: x - r.left, dy: y - r.top, lifted: false }
		press = p
		el.setPointerCapture(evt.pointerId)
		o.ghost.replaceChildren(o.copy ? o.copy(item) : item.cloneNode(true))
		o.ghost.style.inlineSize = r.width + 'px'
		if (touch) {
			lift(p)
			follow(p, x, y)
		}
	})
	scope.listen(el, 'pointermove', (evt) => {
		const p = ours(evt)
		if (!p) return
		if (!p.lifted) {
			if (Math.hypot(evt.clientX - p.x, evt.clientY - p.y) < threshold) return
			lift(p)
		}
		follow(p, evt.clientX, evt.clientY)
		o.aim(evt.clientX, evt.clientY)
	})
	scope.listen(el, 'pointerup', (evt) => {
		const p = ours(evt)
		if (!p) return
		press = null
		if (p.lifted) o.drop()
	})
	// The browser took the drag over: a scroll, a second finger.
	const lost = (evt: PointerEvent): void => {
		const p = ours(evt)
		if (!p) return
		press = null
		if (p.lifted) o.cancel()
	}
	scope.listen(el, 'pointercancel', lost)
	scope.listen(el, 'lostpointercapture', lost)

	return {
		get dragging() {
			return !!press?.lifted
		},
		abort() {
			const p = press
			press = null
			if (!p) return
			if (el.hasPointerCapture(p.pointer)) el.releasePointerCapture(p.pointer)
			if (p.lifted) o.cancel()
		},
	}
}
