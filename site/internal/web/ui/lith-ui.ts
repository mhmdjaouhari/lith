// lith-ui: what the site's controllers share. A controller is the code of
// a demo's stage that a Datastar expression would hold badly: a drag, the
// keyboard, an effect that draws from many signals. It lives in a module
// that tsc checks, typed by the interfaces dsgen writes from the demo's Go
// structs, and reaches the page through Datastar's own API: its signals,
// its effects, and an attribute plugin that mounts it on its element.
// The server stays the only one that changes state: a controller only
// moves local signals and dispatches commands, which the page posts.

import { action, attribute, beginBatch, effect, endBatch, root, signal } from 'datastar'

/** The page's signals, typed by the Roots that dsgen writes: signals<Roots>()._board. */
export function signals<R extends object>(): R {
	return root as unknown as R
}

/**
 * A controller's own state: an object whose fields are Datastar signals
 * outside the page's, so an effect that reads them runs again when they
 * change, while the page's attributes, its data-signals and the server
 * never see them. Sealed: a misspelt field is an error at runtime too.
 */
export function local<T extends Record<string, unknown>>(init: T): T {
	const out = {} as T
	for (const key of Object.keys(init) as (keyof T & string)[]) {
		const s = signal(init[key])
		Object.defineProperty(out, key, {
			enumerable: true,
			get: () => s(),
			set: (v: T[typeof key]) => {
				s(v)
			},
		})
	}
	return Object.seal(out)
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
	listen<K extends keyof DocumentEventMap>(target: Document, type: K, fn: (evt: DocumentEventMap[K]) => void, opts?: AddEventListenerOptions): void
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
 *
 * Why an attribute plugin and not a light-DOM Rocket component: Rocket
 * applies Datastar again to its host's whole subtree after every morph
 * inside it, which reruns the subtree's data-init (the board's hello went
 * out with every frame) and, measured on the board (TestBoardStageCost,
 * 40 moves), cost a watching tab about 3 times the script time, 9 ms a
 * move against 3.3, with 5 times the re-scans. The plugin leaves the
 * server's children to the morph alone.
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

/** A call of a controller's action: the event of the data-on that called it, the element it is on, and the call's arguments. */
export interface ActionCall {
	evt: Event | undefined
	el: HTMLElement
	args: readonly unknown[]
}

/** The controllers' elements that implement each action, by its name. */
const bound = new Map<string, Set<{ el: HTMLElement; fn: (call: ActionCall) => unknown }>>()

/**
 * Implements the actions the markup inside el calls: data-on:click=
 * "@lifeDrop()" on an element in el runs impl.lifeDrop with the click.
 * The markup says which element answers which event; the controller,
 * how. A is dsgen's Actions, the Act… names of //ds:dom, so tsc fails on
 * an action the markup calls that impl lacks, or one it does not call:
 * actions<Actions>(el, scope, {...}). Datastar's actions are global: each
 * name is registered once, and a call runs the implementation of the
 * controller whose element holds the caller.
 */
export function actions<A extends string>(el: HTMLElement, scope: Scope, impl: Record<A, (call: ActionCall) => unknown>): void {
	for (const name of Object.keys(impl) as A[]) {
		let set = bound.get(name)
		if (!set) {
			const mine = new Set<{ el: HTMLElement; fn: (call: ActionCall) => unknown }>()
			set = mine
			bound.set(name, mine)
			action({
				name,
				apply({ el: from, evt }, ...args: unknown[]) {
					for (const b of mine) if (b.el.contains(from)) return b.fn({ evt, el: from, args })
					throw new Error(`lith-ui: @${name}() outside its controller's element`)
				},
			})
		}
		const b = { el, fn: impl[name] }
		set.add(b)
		scope.defer(() => set.delete(b))
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

/** How an item is held: not at all, by a pointer, or from the keyboard. */
export type Hold = '' | 'pointer' | 'keys'

/** A place in a group: its index there, among count other items. */
export interface Place {
	group: number
	index: number
	count: number
}

export interface SortableOptions {
	/** The attribute that holds each group's index, on the group. */
	groups: `data-${string}`
	/** The attribute that holds each item's id (a number above 0), on the item, a child of its group's list. */
	items: `data-${string}`
	/** The groups' box: a pointer well outside it aims nowhere. */
	bounds: HTMLElement
	/** A touch picks an item up by this part of it only; the rest scrolls the page. */
	handle?: string
	/** The element that follows the pointer, and what of the item it shows. */
	ghost: HTMLElement
	copy?: (item: HTMLElement) => Node
	/** Elements besides the ghost that the controller changes while it drags (a marker): their changes move nothing. */
	own?: readonly Element[]
	/** Whether a group is a row that wraps, rather than a column. */
	wraps?(group: HTMLElement): boolean
	/** False keeps both the pointer and the keyboard from picking an item up. */
	enabled?(): boolean
	/** An item that counts as gone from its group, such as one whose move is pending; 0: none. */
	absent?(): number
	/** Called when an item is picked up, with where it is. */
	lift?(item: HTMLElement, hold: Hold, from: Place): void
	/** Called when the keyboard aims somewhere, home true where the item already is. */
	aimed?(to: Place, home: boolean): void
	/** Called when the item goes back where it was; tell: the keyboard put it back. */
	cancel?(item: HTMLElement | null, tell: boolean): void
	/** Called when the item is dropped somewhere else. The aim stays until aim(-1, 0). */
	drop(item: HTMLElement, to: Place): void
	/** Called when the keyboard tries to pick an item up while enabled is false. */
	busy?(): void
}

export interface Sortable {
	/**
	 * The item held (0: none), how, the group and index it came from, and
	 * where it would go: group -1 is nowhere, count is how many other
	 * items that group has. Signals: an effect that reads them runs again.
	 */
	readonly state: { item: number; hold: Hold; from: number; at: number; group: number; index: number; count: number }
	/** Aims at index in group, clamped to it; group -1 aims nowhere. */
	aim(group: number, index: number): void
}

/**
 * Lists of items in groups that a person sorts with a pointer or the
 * keyboard: the board's cards in lanes. On an item, Space or Enter picks
 * it up and the arrows move the focus from item to item; picked up, the
 * arrows choose a place, Space or Enter drops it, Escape or Tab puts it
 * back, as does the focus leaving el. A pointer drags it (pointerDrag).
 * Nothing moves an item: sortable keeps where it would go, and the
 * caller sends the move (drop) and draws the state (an effect).
 */
export function sortable(el: HTMLElement, scope: Scope, o: SortableOptions): Sortable {
	const st = local({ item: 0, hold: '' as Hold, from: -1, at: 0, group: -1, index: 0, count: 0 })
	const gsel = `[${o.groups}]`
	const isel = `[${o.items}]`
	const groups = (): HTMLElement[] => [...el.querySelectorAll<HTMLElement>(gsel)]
	const group = (n: number): HTMLElement | null => el.querySelector<HTMLElement>(attr(o.groups, n))
	const item = (id: number): HTMLElement | null => (id ? el.querySelector<HTMLElement>(attr(o.items, id)) : null)
	const groupOf = (it: HTMLElement): number => {
		const g = it.closest<HTMLElement>(gsel)
		if (!g) throw new Error('lith-ui: an item outside a group')
		return num(g, o.groups)
	}
	const indexOf = (it: HTMLElement): number => (it.parentElement ? [...it.parentElement.children].indexOf(it) : 0)
	const gone = (): number => st.item || (o.absent?.() ?? 0)
	const others = (n: number): HTMLElement[] => [...(group(n)?.querySelectorAll<HTMLElement>(isel) ?? [])].filter((it) => num(it, o.items) !== gone())
	const place = (): Place => ({ group: st.group, index: st.index, count: st.count })
	const enabled = (): boolean => o.enabled?.() ?? true

	const aim = (n: number, index: number): void => {
		if (n < 0 || !group(n)) {
			st.group = -1
			return
		}
		const count = others(n).length
		st.index = Math.max(0, Math.min(index, count))
		st.count = count
		st.group = n
	}

	// The groups' and items' boxes, measured once and kept until a frame, a
	// scroll or a resize may have moved them.
	const geometry = measured(el, scope, [o.ghost, ...(o.own ?? [])], () => ({
		bounds: o.bounds.getBoundingClientRect(),
		groups: groups().map((g) => ({
			n: num(g, o.groups),
			box: g.getBoundingClientRect(),
			wraps: o.wraps?.(g) ?? false,
			items: [...g.querySelectorAll(isel)].map((it) => ({ id: num(it, o.items), box: it.getBoundingClientRect() })),
		})),
	}))
	const aimAt = (x: number, y: number): void => {
		const g = geometry()
		const b = g.bounds
		if (x < b.left - 48 || x > b.right + 48 || y < b.top - 64 || y > b.bottom + 64) {
			aim(-1, 0)
			return
		}
		const near = nearest(g.groups, x, y)
		if (!near) return
		const boxes = near.items.filter((it) => it.id !== gone()).map((it) => it.box)
		aim(near.n, insertionIndex(boxes, x, y, near.wraps))
	}

	const lift = (it: HTMLElement, hold: Hold): void => {
		const from = groupOf(it)
		st.item = num(it, o.items)
		st.hold = hold
		st.from = from
		st.at = indexOf(it)
		aim(from, st.at)
		o.lift?.(it, hold, place())
	}
	const cancel = (tell: boolean): void => {
		const it = item(st.item)
		st.item = 0
		st.hold = ''
		aim(-1, 0)
		o.cancel?.(it, tell)
	}
	const drop = (): void => {
		const it = item(st.item)
		if (!it || st.group < 0 || (st.group === st.from && st.index === st.at)) {
			cancel(true)
			return
		}
		const to = place()
		st.item = 0
		st.hold = ''
		o.drop(it, to)
	}
	const walk = (it: HTMLElement, key: string): void => {
		let n = groupOf(it)
		const i = indexOf(it)
		let next: Element | null = null
		if (key === 'ArrowUp') next = it.previousElementSibling
		if (key === 'ArrowDown') next = it.nextElementSibling
		if (key === 'ArrowLeft' || key === 'ArrowRight') {
			const step = key === 'ArrowLeft' ? -1 : 1
			for (n += step; !next && n >= 0 && group(n); n += step) {
				const items = group(n)?.querySelectorAll(isel) ?? []
				next = items[Math.min(i, items.length - 1)] ?? null
			}
		}
		const target = next?.querySelector<HTMLElement>('button, [tabindex]') ?? (next instanceof HTMLElement ? next : null)
		target?.focus()
	}

	const drag = pointerDrag(el, scope, {
		items: isel,
		...(o.handle ? { handle: o.handle } : {}),
		ghost: o.ghost,
		...(o.copy ? { copy: o.copy } : {}),
		enabled,
		pressed: () => {
			if (st.hold === 'keys') cancel(false)
		},
		lift: (it) => lift(it, 'pointer'),
		aim: aimAt,
		drop,
		cancel: () => cancel(false),
	})

	scope.listen(el, 'keydown', (evt) => {
		const k = evt.key
		const arrow = k.startsWith('Arrow')
		if (st.hold === 'pointer') {
			if (k === 'Escape') {
				evt.preventDefault()
				drag.abort()
			}
			return
		}
		if (st.hold === 'keys') {
			if (arrow) {
				evt.preventDefault()
				const step = k === 'ArrowLeft' ? -1 : k === 'ArrowRight' ? 1 : 0
				const n = Math.max(0, Math.min(groups().length - 1, st.group + step))
				aim(n, step ? st.index : st.index + (k === 'ArrowUp' ? -1 : 1))
				o.aimed?.(place(), n === st.from && st.index === st.at)
			} else if (k === ' ' || k === 'Enter') {
				evt.preventDefault()
				drop()
			} else if (k === 'Escape' || k === 'Tab') {
				if (k === 'Escape') evt.preventDefault()
				cancel(true)
			}
			return
		}
		const it = evt.target instanceof Element ? evt.target.closest<HTMLElement>(isel) : null
		if (!it || evt.altKey || evt.ctrlKey || evt.metaKey) return
		if (k === ' ' || k === 'Enter') {
			evt.preventDefault()
			if (enabled()) lift(it, 'keys')
			else o.busy?.()
		} else if (arrow) {
			evt.preventDefault()
			walk(it, k)
		}
	})
	scope.listen(el, 'focusout', (evt) => {
		const to = evt.relatedTarget
		if (st.hold === 'keys' && !(to instanceof Node && el.contains(to))) cancel(true)
	})

	return { state: st, aim }
}

/** A rule of a Sheet: a selector and the properties it sets, P's only. */
export type Rule<P extends string> = readonly [selector: string, props: Partial<Record<P, string | number>>]

/**
 * A stylesheet a controller owns, adopted by the document while its scope
 * lasts, for looks the page's state gives elements that the stream morphs:
 * a frame resets what a script sets on such an element (its style
 * attribute, its classes), or replaces the element, while a rule matches
 * whatever element the frame brings. P names the properties it may set,
 * so the CSS that reads them and the controller agree on a list.
 */
export class Sheet<P extends `--${string}` | 'anchor-name'> {
	#sheet = new CSSStyleSheet()
	#text = ''

	constructor(scope: Scope) {
		document.adoptedStyleSheets = [...document.adoptedStyleSheets, this.#sheet]
		scope.defer(() => {
			document.adoptedStyleSheets = document.adoptedStyleSheets.filter((s) => s !== this.#sheet)
		})
	}

	/** Replaces the sheet's rules. */
	set(rules: readonly Rule<P>[]): void {
		const css = rules.map(([sel, props]) => `${sel}{${Object.entries(props).map(([k, v]) => `${k}:${String(v)}`).join(';')}}`)
		const text = css.join('\n')
		// A new sheet makes the browser match every element again: only
		// when the rules changed.
		if (text === this.#text) return
		this.#text = text
		this.#sheet.replaceSync(text)
	}
}

/** The selector of the elements whose attribute name is value, escaped. */
export function attr(name: `data-${string}`, value: string | number): string {
	return `[${name}="${CSS.escape(String(value))}"]`
}

/** The signals of a pending op (Go: demo.OpSignals), which a demo's signals embed. */
export interface OpSignals {
	/** The op waiting for its ack, "" for none. */
	op: string
	/** The op the tab's last outcome answers. */
	ack: string
	/** The stream brought the tab's row since it (re)connected: the shell sets it false, the tab's view true. */
	synced: boolean
	/** The page's line for what went wrong. */
	problem: string
}

export interface PendingOptions {
	/** The demo's signals, and the name they are declared under ('_board'). */
	signals: OpSignals
	root: string
	/** Ends the pending state: clears op, and whatever else the demo keeps of it. */
	end(): void
	/** The page's words when a request never got through, when a status came without a notice, when the op was lost to a reconnect. */
	down: string
	failed: string
	lost: string
	/** The words while a request is retried; none by default. */
	retrying?: string
	/**
	 * Whether the op still waits for its ack; by default while ack is not
	 * op. A demo whose ack answers every older op too (the checkboxes:
	 * op ids from orderedOpId) waits while ack < op.
	 */
	waiting?: () => boolean
	/**
	 * Counts the requests of the elements inside el too, not only el's: a
	 * swatch's data-on:click that posts by itself. None of them may be one
	 * a frame removes, or its count leaks.
	 */
	within?: boolean
}

/** Opens the page's stream again, as sse.Reconnect has the page do. */
export function reopenStream(id: string = streamID): void {
	const s = document.getElementById(id)
	s?.replaceWith(s.cloneNode(true))
}

/**
 * Ends the op the element's requests carry on a signal, never on a clock
 * (the Go side: demo.Pending, for demos without a controller): its ack,
 * which the demo's own effect settles; Datastar's retries-failed; an
 * answer of 400 or another 4xx but 409 and 429, which comes without a
 * notice; or the stream connecting again with a tab row that does not
 * ack it, once no request of the op is in flight. Act sends a notice with
 * every 429 and 5xx but a closed writer's 503, whose stream is ending:
 * those leave the op to the stream, and so does a 409, a tab without a
 * stream, whose reconnect decides. When the last request ends after a
 * reconnect that found no ack, it opens the stream once more, so that a
 * fresh frame of the tab's row decides. The requests must go out from el,
 * which no frame removes, so the count cannot leak.
 */
export function pending(el: HTMLElement, scope: Scope, o: PendingOptions): void {
	const s = o.signals
	let inflight = 0
	let recheck = false
	const waiting = o.waiting ?? ((): boolean => s.op !== '' && s.ack !== s.op)
	const fail = (words: string): void => {
		o.end()
		s.problem = words
	}
	scope.listen(document, 'datastar-fetch', (evt) => {
		const { type, el: from, argsRaw } = evt.detail
		if (o.within ? !el.contains(from) : from !== el) return
		switch (type) {
			case 'started':
				inflight++
				break
			case 'finished':
				inflight = Math.max(0, inflight - 1)
				if (inflight === 0 && recheck) {
					recheck = false
					if (waiting()) reopenStream()
				}
				break
			case 'retrying':
				if (o.retrying && s.op) s.problem = o.retrying
				break
			case 'error': {
				const status = Number(argsRaw['status'])
				if (waiting() && status !== 409 && status !== 429 && status < 500) fail(o.failed)
				break
			}
			case 'retries-failed':
				if (waiting()) fail(o.down)
				break
		}
	})
	scope.listen(document, 'datastar-signal-patch', (evt) => {
		const mine = evt.detail[o.root]
		if (typeof mine !== 'object' || mine === null || mine['synced'] !== true || !waiting()) return
		if (inflight > 0) recheck = true
		else fail(o.lost)
	})
}

/** The id of the element that opens the page's stream (Go: views.StreamID; TestStreamID ties them). */
export const streamID = 'stream'

/**
 * A new op id: 96 random bits in base64url, 16 characters that
 * demo.ValidID accepts. crypto.getRandomValues, unlike randomUUID, works
 * outside a secure context too (a page over plain HTTP on a LAN address).
 */
export function opId(): string {
	const b = crypto.getRandomValues(new Uint8Array(12))
	return btoa(String.fromCharCode(...b)).replaceAll('+', '-').replaceAll('/', '_')
}

/**
 * A new op id that sorts after every older one, as strings compare in Go
 * and in JS: the time in milliseconds, base 36 in 9 digits (enough until
 * the year 5188), and 8 random characters of opId for ops of the same
 * millisecond. An ack of the newest such id answers the older ones too.
 */
export function orderedOpId(): string {
	return Date.now().toString(36).padStart(9, '0') + opId().slice(0, 8)
}

// It is an action too, so that an expression makes its op id as a
// controller does: $_x.op = @orderedOpId() (ds.OrderedOpID). dscheck
// knows it (Actions).
action({ name: 'orderedOpId', apply: () => orderedOpId() })

/** The value of a data- attribute as a number; a missing one is an error in the page. */
export function num(el: Element, name: `data-${string}`): number {
	const v = el.getAttribute(name)
	if (v === null || v === '' || Number.isNaN(+v)) throw new Error(`lith-ui: ${name} is no number`)
	return +v
}

/** The element of a selector inside root; a missing one is an error in the page. */
export function must<T extends Element = HTMLElement>(root: ParentNode, selector: string): T {
	const e = root.querySelector<T>(selector)
	if (!e) throw new Error(`lith-ui: no ${selector}`)
	return e
}

/** v held between lo and hi. */
export function clamp(v: number, lo: number, hi: number): number {
	return Math.min(hi, Math.max(lo, v))
}

/** n with a comma between thousands, as Go's format.Grouped writes it: 12,345. */
export function grouped(n: number): string {
	return n.toLocaleString('en-US', { maximumFractionDigits: 0 })
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

/** The one of items whose box is nearest the point. */
export function nearest<T extends { box: DOMRect }>(items: readonly T[], x: number, y: number): T | undefined {
	let best: T | undefined
	let d = Infinity
	for (const it of items) {
		const n = distance2(it.box, x, y)
		if (n < d) [best, d] = [it, n]
	}
	return best
}

/**
 * Where an item dropped at the point would go among the boxes of a list:
 * before the first one below it, or, in a list that wraps (a row of
 * cards), before the first one below it or to its right. boxes.length:
 * at the end.
 */
export function insertionIndex(boxes: readonly DOMRect[], x: number, y: number, wraps: boolean): number {
	const i = boxes.findIndex((r) => (wraps ? y < r.top || (y < r.bottom && x < r.left + r.width / 2) : y < r.top + r.height / 2))
	return i < 0 ? boxes.length : i
}

/**
 * Returns get, which measures the page with measure once and returns the
 * same measurements until they may have changed: a scroll, a resize, or a
 * change of the DOM under root but in the elements own (what the
 * controller itself moves, such as a drag's ghost, whose changes never
 * move the rest). A drag reads boxes at every pointer move; reading them
 * from the page each time forces a layout each time.
 */
export function measured<T>(root: Element, scope: Scope, own: readonly Element[], measure: () => T): () => T {
	let cache: { value: T } | null = null
	const reset = (): void => {
		cache = null
	}
	scope.listen(window, 'scroll', reset, { capture: true, passive: true })
	scope.listen(window, 'resize', reset, { passive: true })
	const mo = new MutationObserver((records) => {
		if (records.some((r) => !own.some((o) => o.contains(r.target)))) reset()
	})
	mo.observe(root, { subtree: true, childList: true, attributes: true, characterData: true })
	scope.defer(() => mo.disconnect())
	return () => (cache ??= { value: measure() }).value
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
	// A pointer moves more often than the page paints: the ghost and the aim
	// follow its latest place once a frame. A drop or a cancel first runs
	// the frame's move, so the drop goes where the pointer let go.
	let at: { p: Press; x: number; y: number } | null = null
	let frame = 0
	const step = (): void => {
		frame = 0
		const m = at
		at = null
		if (!m || m.p !== press) return
		batch(() => {
			follow(m.p, m.x, m.y)
			o.aim(m.x, m.y)
		})
	}
	const flush = (): void => {
		if (frame) cancelAnimationFrame(frame)
		step()
	}
	scope.defer(() => cancelAnimationFrame(frame))
	scope.listen(el, 'pointermove', (evt) => {
		const p = ours(evt)
		if (!p) return
		if (!p.lifted) {
			if (Math.hypot(evt.clientX - p.x, evt.clientY - p.y) < threshold) return
			lift(p)
		}
		at = { p, x: evt.clientX, y: evt.clientY }
		frame ||= requestAnimationFrame(step)
	})
	scope.listen(el, 'pointerup', (evt) => {
		const p = ours(evt)
		if (!p) return
		flush()
		press = null
		if (p.lifted) o.drop()
	})
	// The browser took the drag over: a scroll, a second finger.
	const lost = (evt: PointerEvent): void => {
		const p = ours(evt)
		if (!p) return
		at = null
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
			at = null
			if (!p) return
			if (el.hasPointerCapture(p.pointer)) el.releasePointerCapture(p.pointer)
			if (p.lifted) o.cancel()
		},
	}
}
