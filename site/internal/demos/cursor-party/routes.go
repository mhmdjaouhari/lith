package cursorparty

import (
	"errors"
	"net/http"
	"time"

	"github.com/zweiundeins/lith/edge"
	"github.com/zweiundeins/lith/writer"

	"github.com/zweiundeins/lith/site/internal/demo"
)

func mount(k *demo.Kit) {
	// A move tells the person nothing: their own pointer is on their page,
	// and the next move replaces this one. What fails it answers a bare
	// status, and no notice.
	MoveAction.MountOpen(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req MoveReq) {
		edge.Quiet(r) // 15 a second from every pointer: a log line each at Debug only
		if demo.Limited(r) {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		cmd := Move{Viewer: v, A: req.A, X: req.X, Y: req.Y, Seq: req.Seq, At: time.Now()}
		if err := cmd.Validate(); err != nil {
			w.WriteHeader(http.StatusBadRequest) // the page sends only moves that pass
			return
		}
		// Nobody waits for a move: the page has its own pointer, and the
		// next move replaces this one anyway.
		if err := k.W.Send(cmd); err != nil {
			if !errors.Is(err, writer.ErrQueueFull) && !errors.Is(err, writer.ErrClosed) {
				k.Fail(w, r, err)
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable) // the next move replaces this one
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
