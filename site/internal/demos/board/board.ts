// The board's controller, on the stage (data-board): lith-ui's sortable
// does the drag and the keyboard over the lanes and cards; the board adds
// what it says, the move it sends, the pending move and the looks. It
// reads the page by the names that dom.go shares with the templates: the
// lanes carry their index and name, the cards their id, version and
// title. On a narrow board a lane is a row of cards that wraps, and
// board.css says so with PropFlow.
//
// Nothing here moves a card. It moves the page's signals ($_board, typed
// by Signals from signals.go) and its own, and dispatches the move, typed
// by MoveReq, which the stage posts; the board's next frame moves the
// card.

import { announcer, attr, commands, controller, local, must, num, opId, pending, type Place, type Rule, Sheet, signals, sortable } from 'lith-ui'
import {
	AnimBusy,
	AnimNudge,
	AttrCard,
	AttrHello,
	AttrLane,
	AttrLaneName,
	AttrTitle,
	AttrVersion,
	ClassCard,
	ClassGhost,
	ClassGrip,
	ClassLaneCards,
	ClassLanes,
	ClassMarker,
	ClassSlot,
	type Commands,
	isLaneID,
	PropBusy,
	PropEdge,
	PropFlow,
	PropLift,
	PropNudge,
	PropOn,
	PropPick,
	PropSlot,
	PropTarget,
	PropWait,
	type Roots,
} from './ds.gen'

/** The properties the controller's sheet sets, which board.css reads. */
type Look = typeof PropLift | typeof PropPick | typeof PropWait | typeof PropBusy | typeof PropNudge | typeof PropTarget | 'anchor-name'

controller('board', (el, scope) => {
	// The page's signals: what the page's attributes and the server read.
	const s = signals<Roots>()._board
	// The controller's own, reactive for its effects but no part of the
	// page's signals, which the server could patch: the version of the card
	// in hand when it was picked up, the card of the pending move, and a
	// refused one.
	const h = local({ seen: 0, pcard: 0, nudge: 0 })
	const say = announcer(() => s.say, (t) => (s.say = t))
	const send = commands<Commands>(el)
	const marker = must(el, `.${ClassMarker}`)

	const name = (n: number): string => el.querySelector(attr(AttrLane, n))?.getAttribute(AttrLaneName) ?? ''
	const title = (c: HTMLElement): string => c.getAttribute(AttrTitle) ?? ''
	const laneOf = (c: HTMLElement): number => num(c.closest(`[${AttrLane}]`) ?? el, AttrLane)
	const where = (p: Place): string => `${name(p.group)}, ${p.index + 1} of ${p.count + 1}`

	const sort = sortable(el, scope, {
		groups: AttrLane,
		items: AttrCard,
		bounds: must(el, `.${ClassLanes}`),
		handle: `.${ClassGrip}`,
		ghost: must(el, `.${ClassGhost}`),
		copy: (c) => must(c, `.${ClassCard}`).cloneNode(true),
		own: [marker],
		wraps: (l) => getComputedStyle(l).getPropertyValue(PropFlow).trim() === '1',
		// One move at a time: none while one is pending.
		enabled: () => !s.op,
		absent: () => h.pcard,
		lift(c, hold, from) {
			h.seen = num(c, AttrVersion)
			if (hold === 'keys') say(`Picked up “${title(c)}”: ${where(from)}. Arrows choose a place, Space drops it, Escape puts it back.`)
		},
		aimed: (to, home) => say(where(to) + (home ? ', where it is' : '') + '.'),
		cancel(c, tell) {
			if (tell && c) say(`“${title(c)}” stays in ${name(laneOf(c))}.`)
		},
		// drop sends the move, with the card's version from when it was
		// picked up; the place stays marked while the move is pending.
		drop(c, to) {
			if (!isLaneID(to.group)) throw new Error(`board: no lane ${to.group}`)
			const move = { op: opId(), card: num(c, AttrCard), lane: to.group, index: to.index, seen: h.seen }
			h.pcard = move.card
			s.problem = ''
			s.op = move.op
			say(`Moving “${title(c)}” to ${name(to.group)}.`)
			send('board-move', move)
		},
		busy: () => say('One move at a time: the last one is on its way.'),
	})
	const st = sort.state

	// The page shows how a card is held: the stage's class, the ghost.
	scope.effect(() => {
		s.mode = st.hold
	})

	// The tab says hello, for the presence row, each time its stream brings
	// the tab's row after (re)connecting: the shell marks the board out of
	// sync when the stream opens, retries or the tab hides (demo.Resync),
	// and the tab's view sets synced with its first frame after. That is the
	// page's load, a deploy, a network retry and a hidden tab shown again,
	// whose hello the sweep may have forgotten. A board already in sync as
	// the controller mounts says hello at once.
	const hello = commands<Commands>(must(el, `[${AttrHello}]`))
	let synced = false
	scope.effect(() => {
		if (s.synced && !synced) hello('board-hello', {})
		synced = s.synced
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
			if (!st.item) sort.aim(-1, 0)
		},
		down: 'That move did not reach the server. Try again in a moment.',
		failed: 'That did not work. Try again.',
		lost: 'That move did not go through. Try again.',
		retrying: 'The server does not answer. Trying again.',
	})

	// settle ends the pending move once its outcome has come and the board
	// shows the version that has it. A refused card shakes once (AnimNudge),
	// and the shake ends with its animation: board.css owns how long it
	// takes, and reduced motion, which shortens it, ends it at once.
	scope.effect(() => {
		const op = s.op
		if (!op || s.ack !== op || s.v < s.ackv) return
		if (!s.ok) h.nudge = h.pcard
		s.op = ''
		h.pcard = 0
		s.problem = ''
		if (!st.item) sort.aim(-1, 0)
	})
	const shaken = (evt: AnimationEvent): void => {
		if (evt.animationName === AnimNudge) h.nudge = 0
	}
	scope.listen(el, 'animationend', shaken)
	// A frame that replaces the card mid-shake ends it too.
	scope.listen(el, 'animationcancel', shaken)

	// looks is the style of the card in hand (--bd-lift under the pointer,
	// which leaves a hole behind its ghost; --bd-pick from the keyboard,
	// which rises), of a card pending or refused, and of the place the card
	// would go; board.css draws them. The cards and lanes are the stream's,
	// which morphs them with every move, so their looks are rules of a
	// sheet the controller owns, which match whatever elements a frame
	// brings. The marker is the stage's own: its properties are set on it.
	// It sits above the card that would follow the moved one, below the
	// lane's last card, or at the top of an empty lane (the anchor), and
	// stays hidden while the place is the card's own.
	const sheet = new Sheet<Look>(scope)
	scope.effect(() => {
		const rules: Rule<Look>[] = []
		const card = (id: number): string => attr(AttrCard, id)
		if (st.item) rules.push([card(st.item), st.hold === 'keys' ? { [PropPick]: 1 } : { [PropLift]: 1 }])
		if (h.pcard) rules.push([card(h.pcard), { [PropWait]: 1, [PropBusy]: AnimBusy }])
		if (h.nudge) rules.push([card(h.nudge), { [PropNudge]: AnimNudge }])
		let on = false
		let edge = false
		if (st.group >= 0) {
			const lane = attr(AttrLane, st.group)
			rules.push([lane, { [PropTarget]: 1 }])
			on = !(st.item && st.group === st.from && st.index === st.at)
			if (on && st.count === 0) rules.push([`${lane} .${ClassLaneCards}`, { 'anchor-name': PropSlot }])
			else if (on) {
				edge = st.index >= st.count
				const of = `of :not(${card(st.item || h.pcard)})`
				rules.push([`${lane} .${ClassSlot}:${edge ? `nth-last-child(1 ${of})` : `nth-child(${st.index + 1} ${of})`}`, { 'anchor-name': PropSlot }])
			}
		}
		sheet.set(rules)
		marker.style.setProperty(PropOn, on ? '1' : '0')
		marker.style.setProperty(PropEdge, edge ? '1' : '0')
	})
})
