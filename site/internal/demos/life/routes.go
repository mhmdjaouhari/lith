package life

import (
	"net/http"
	"time"

	"github.com/zweiundeins/lith/writer"

	"github.com/zweiundeins/lith/site/internal/demo"
)

func mount(k *demo.Kit) {
	if k.Hub != nil {
		c := &clock{w: k.W, hub: k.Hub, log: k.Log}
		go c.run()
	}
	// A stroke sent twice draws the same cells: no Once needed.
	DrawAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req DrawReq) {
		if !demo.ValidID(req.Op) {
			w.WriteHeader(http.StatusBadRequest) // no page of ours sends it
			return
		}
		k.Act(w, r, v, k.AckOf(S.Ack.Set(req.Op)), Draw{Viewer: v, Op: req.Op, Cells: req.Cells, Alive: req.Alive, At: time.Now()})
	})
	DropAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req DropReq) {
		if !demo.ValidID(req.Op) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		cmd := Drop{Viewer: v, Op: req.Op, Pattern: req.Pattern, Cell: req.Cell, Turn: req.Turn, At: time.Now()}
		k.Act(w, r, v, k.AckOf(S.Ack.Set(req.Op)), writer.Once(Slug+":"+v.Session+":"+req.Op, cmd))
	})
	ResetAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req ResetReq) {
		if !demo.ValidID(req.Op) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		k.Act(w, r, v, k.AckOf(S.Ack.Set(req.Op)), writer.Once(Slug+":"+v.Session+":"+req.Op, Reset{Viewer: v, Op: req.Op, At: time.Now()}))
	})
	ColourAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req ColourReq) {
		// A notice sets the swatch back to the colour the session has.
		was, err := colourOf(r.Context(), k.DB.R, v.Session)
		if err != nil {
			k.Fail(w, r, err)
			return
		}
		k.Act(w, r, v, k.AckOf(S.Pick.Set(was)), Pick{Viewer: v, Colour: req.Colour, At: time.Now()})
	})
}
