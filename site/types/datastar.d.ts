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
