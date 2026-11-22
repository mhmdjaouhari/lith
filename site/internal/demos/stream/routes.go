package stream

import (
	"net/http"
	"time"

	"github.com/zweiundeins/lith/writer"

	"github.com/zweiundeins/lith/site/internal/demo"
)

func mount(k *demo.Kit) {
	PayloadAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req PayloadReq) {
		if !demo.ValidID(req.Op) {
			w.WriteHeader(http.StatusBadRequest) // no page of ours sends it
			return
		}
		cmd := SetPayload{Viewer: v, Op: req.Op, Size: req.Size, Churn: req.Churn, At: time.Now()}
		k.Act(w, r, v, k.AckOf(S.Ack.Set(req.Op)), writer.Once(Slug+":"+v.Session+":"+req.Op, cmd))
	})
}
