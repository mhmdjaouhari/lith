// The part of Datastar's module API (v1.0.5, lith's datastar package) that
// the site's TypeScript uses. The page's import map points 'datastar' at
// the build the page loaded, so a module that imports it shares the
// page's signals and plugins.

/**
 * The page's signals: a reactive proxy. Reading a signal inside an
 * effect subscribes the effect to it; writing one notifies its readers.
 * Reading a signal that does not exist creates it as "", so read only
 * what the page declared: lith-ui's signals() types the object by the
 * interfaces dsgen writes from the Go structs.
 */
export const root: Record<string, unknown>

/** Runs fn now and again whenever a signal it read changes; returns a function that stops it. */
export function effect(fn: () => void): () => void

/** Holds the effects of the writes between them until endBatch. */
export function beginBatch(): void
export function endBatch(): void

/** A signal of its own, outside the page's: s() reads it (in an effect, subscribes), s(v) sets it. */
export function signal<T>(initial: T): { (): T; (value: T): boolean }

/** Reads the signal at a dotted path. */
export function getPath(path: string): unknown

export type Requirement = 'allowed' | 'must' | 'denied' | 'exclusive'

export interface AttributeContext {
	el: HTMLElement
	key: string | undefined
	rawKey: string
	value: string
	mods: Map<string, Set<string>>
	error(name: string): Error
}

export interface AttributePlugin {
	/** The attribute's name after data-, such as 'board' for data-board. */
	name: string
	requirement?: Requirement | { key?: Requirement; value?: Requirement }
	/** Runs for each element with the attribute; what it returns runs when the element or the attribute goes. */
	apply(ctx: AttributeContext): void | (() => void)
}

/** Registers an attribute plugin. One registered after the page started applies to the elements already there. */
export function attribute(plugin: AttributePlugin): void

export interface ActionContext {
	/** The element whose expression called the action. */
	el: HTMLElement
	/** The event of a data-on expression; undefined elsewhere. */
	evt: Event | undefined
	error(name: string): Error
}

export interface ActionPlugin {
	/** The name an expression calls it by: lifeDrop for @lifeDrop(). Letters, digits, _ and $. */
	name: string
	/** Runs for each call, with the call's arguments; what it returns is the call's value. */
	apply(ctx: ActionContext, ...args: never[]): unknown
}

/** Registers an action, which expressions call as @name(...). A second one of the same name replaces the first. */
export function action(plugin: ActionPlugin): void

/** What a request of an action (@get, @post...) went through, in the order it may: started, retrying, error, retries-failed, finished. */
export type FetchType = 'started' | 'finished' | 'error' | 'retrying' | 'retries-failed'

/** The detail of datastar-fetch, which Datastar dispatches on the document for each step of a request. */
export interface FetchDetail {
	type: FetchType
	/** The element whose expression sent the request. */
	el: HTMLElement
	/** The step's details; an error's status is a string, such as "429". */
	argsRaw: Record<string, string | undefined>
}

/** The signals a patch changed, nested by path, with null for a removed one: the detail of datastar-signal-patch. */
export type SignalPatch = { [key: string]: SignalPatch | string | number | boolean | null }

declare global {
	interface DocumentEventMap {
		'datastar-fetch': CustomEvent<FetchDetail>
		'datastar-signal-patch': CustomEvent<SignalPatch>
	}
	interface HTMLElementEventMap {
		/** A morph set a property of the element (an input's checked) to what the server sent; it bubbles. */
		'datastar-prop-change': Event
	}
}
