## The command

Every signature is a `Sign` command, and the writer is the only code that writes. `Apply` reads the newest entry and decides in the same transaction, so two people who send the same words at once get one entry, and the second hears why.

```go title=commands.go
func (c Sign) Apply(tx *writer.Tx) error {
	name, message := clean(c.Name), clean(c.Message)
	state := TabState{Op: c.Op}
	var lastName, lastMessage string
	err := tx.QueryRow(`SELECT name, message FROM guestbook_entries ORDER BY id DESC LIMIT 1`).Scan(&lastName, &lastMessage)
	switch {
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return err
	case !fits(name, MaxName):
		state.Refusal = fmt.Sprintf("A name has 1 to %d characters.", MaxName)
	case !fits(message, MaxMessage):
		state.Refusal = fmt.Sprintf("A message has 1 to %d characters.", MaxMessage)
	case links.MatchString(name + " " + message):
		state.Refusal = "Links stay out of this guestbook."
	case err == nil && lastName == name && lastMessage == message:
		state.Refusal = "That is the newest entry already."
	default:
		if _, err := tx.Exec(`INSERT INTO guestbook_entries (name, message, at) VALUES (?, ?, ?)`,
			name, message, c.At.Unix()); err != nil {
			return err
		}
		tx.Touch(Topic)
	}
	return demo.SaveTab(tx, Slug, c.Viewer, state, c.At)
}
```

A refusal is not an error: the command still commits, with the reason in the tab's row and no entry. A name or a message of the wrong length is a refusal too, since the person typed it. `demo.SaveTab` writes that row of `site_tabs` and touches the tab's topic, so only this tab's view renders the refusal.

## The action

The action's payload is a Go struct, `SignReq`, which names its route with `//ds:command guestbook sign`. `SignAction` mounts it at `/cmd/guestbook/sign` behind the site's rate limits and decodes it strictly: a key the struct lacks, one it has that the request lacks, or a null is a bare 400, never an empty name. The handler waits for the commit and answers 204; `writer.Once` makes a request that the network sends twice sign once. Its status carries nothing a person reads: over the limits, or when the write fails, `Act` answers with the status alone and sends the tab a notice with the op id, which ends the pending state.

```go title=routes.go
func mount(k *demo.Kit) {
	SignAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req SignReq) {
		if !demo.ValidID(req.Op) {
			w.WriteHeader(http.StatusBadRequest) // no page of ours sends it
			return
		}
		cmd := Sign{Viewer: v, Op: req.Op, Name: req.Name, Message: req.Message, At: time.Now()}
		// A retried request (a flaky network) signs once.
		k.Act(w, r, v, k.AckOf(S.Ack.Set(req.Op), S.Accepted.Set(false)), writer.Once(Slug+":"+v.Session+":"+req.Op, cmd))
	})
}
```

## The views

The list is one view for everyone: its key has nothing of the viewer in it. It renders once per tick in which the `guestbook` topic changed, and every tab gets the same frame.

```go title=live.go
func entriesView() live.View {
	return live.View{
		Key:    Slug + ":entries",
		Topics: []string{Topic},
		Render: func(ctx context.Context, tx *sql.Tx) (any, error) {
			list, total, err := latest(ctx, demo.Querier(ctx, tx), Shown)
			if err != nil {
				return nil, err
			}
			return demo.Animated(ctx, Entries(list, total, time.Now()), sse.Patch{ViewTransitionSelector: "#guestbook-entries"})
		},
	}
}
```

`demo.Animated` renders the frame twice over: the first frame a tab gets morphs at once, and every later one inside a view transition (Datastar's `useViewTransition`). The tab's own view, keyed by its tab id, renders the line under the button and patches the signal `$_guestbook.ack` with the op id of the outcome, a typed assignment: `ds.Tree(S.Ack.Set(state.Op), S.Accepted.Set(state.Accepted()), S.Synced.Set(true))`.

## The page

The page keeps its state in signals under `$_guestbook`, which stay in the browser. They are a Go struct too: `go generate` (dsgen) writes `S`, a typed reference to each, which the templates and the expressions use, so no expression names a signal or a route by hand, and `dscheck` fails the tests on a `$signal` the struct does not declare.

```go title=signals.go
type Signals struct {
	// op is the signature on its way, ack the one the tab's outcome
	// answers; inflight and recheck what demo.Pending keeps.
	demo.PendingSignals

	Name     string `json:"name"`     // the form's fields, bound to its inputs
	Message  string `json:"message"`  //
	Accepted bool   `json:"accepted"` // the tab's last signature added an entry: the form empties its message
}
```

Pressing **Sign** makes an op id and posts the form; the inputs turn read-only and the button busy while `$_guestbook.op` is set. `PostSignals` builds the payload from the signals and checks it against `SignReq` as the package loads: a key it lacks or misspells, or a signal of the wrong type, stops the site before any page sends it.

```go title=expr.go
var sign = S.Op.Eq("") + " && " + trimmed(S.Name.String()) + " && " + trimmed(S.Message.String()) + ` && (
	` + S.Problem.Assign("") + `,
	` + S.Op.String() + ` = Date.now().toString(36) + Math.random().toString(36).slice(2, 7),
	` + SignAction.PostSignals(demo.Post{RequestCancellation: "disabled"},
	demo.From("op", S.Op),
	demo.From("name", S.Name),
	demo.From("message", S.Message),
) + `
)`
```

When the tab's view brings back the same op id, the pending state ends, and an accepted message is cleared:

```go title=expr.go
var settle = pending + " && " + S.Ack.String() + " === " + S.Op.String() + " && (" + S.Accepted.String() + " && (" + S.Message.Assign("") + "), " + S.Op.Assign("") + ")"
```

The form needs no more than these lines, so it has no controller. A demo whose page logic grows past a line moves it into a TypeScript controller that tsc checks against the same structs, as the board, the checkboxes and the Game of Life do.

## The arrival

Each entry has a view-transition name of its own, so when the list's frame arrives, the entries already there glide down to their new places. A new entry starts from its `@starting-style`: blurred, a little higher, its edge glowing, and settles in.

```css title=guestbook.css
.gb__entry[data-fresh] {
	border-inline-start-color: var(--lith-accent);

	@starting-style {
		opacity: 0;
		translate: 0 -0.75rem;
		filter: blur(6px);
		border-inline-start-color: var(--lith-ember-1);
		box-shadow: 0 0 28px var(--lith-glow-color), inset 0 0 0 1px var(--lith-accent);
	}
}
```

An entry counts as fresh for eight seconds after it was signed. The list is the same for every tab, so a tab that opens in those seconds sees the entry arrive too.
