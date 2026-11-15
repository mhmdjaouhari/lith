package board

import (
	"net/http"
	"time"

	"github.com/zweiundeins/lith/writer"

	"github.com/zweiundeins/lith/site/internal/demo"
)

func mount(k *demo.Kit) {
	MoveAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req MoveReq) {
		if !demo.ValidID(req.Op) {
			w.WriteHeader(http.StatusBadRequest) // no page of ours sends it
			return
		}
		cmd := Move{Viewer: v, Who: WhoOf(v.Session), Op: req.Op, Card: int64(req.Card), Lane: int(req.Lane), Index: req.Index, Seen: req.Seen, At: time.Now()}
		// A retried request (a flaky network) moves the card once.
		k.Act(w, r, v, k.AckOf(S.Ack.Set(req.Op), S.AckV.Set(0), S.OK.Set(false)), writer.Once(Slug+":"+v.Session+":"+req.Op, cmd))
	})
	HelloAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, _ HelloReq) {
		k.Act(w, r, v, nil, Hello{Viewer: v, Who: WhoOf(v.Session), At: time.Now()})
	})
}
