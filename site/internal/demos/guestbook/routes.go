package guestbook

import (
	"net/http"
	"time"

	"github.com/zweiundeins/lith/writer"

	"github.com/zweiundeins/lith/site/internal/demo"
)

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
