package checkboxes

import (
	"net/http"
	"time"

	"github.com/zweiundeins/lith/site/internal/demo"
)

func mount(k *demo.Kit) {
	// A click: the box and the state it should take, with an op id that
	// the tab's clicks view acks (Clicks). Setting a box twice changes it
	// once, so a retried request needs no Once.
	SetAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req SetReq) {
		if !demo.ValidID(req.Op) {
			w.WriteHeader(http.StatusBadRequest) // no page of ours sends it
			return
		}
		k.Act(w, r, v, k.AckOf(S.Ack.Set(req.Op)), SetBox{Viewer: v, Op: req.Op, Box: req.Box, Checked: req.Checked, At: time.Now()})
	})

	// A window: what the list asks for as it scrolls (sb-window), and its
	// tiles per row.
	WindowAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req WindowReq) {
		if req.Offset < 0 || req.Offset >= Chunks {
			w.WriteHeader(http.StatusBadRequest) // no page of ours sends it
			return
		}
		win := windowFor(req.Offset, min(max(req.Count, 1), Chunks), req.Cols)
		k.Act(w, r, v, nil, SetWindow{Viewer: v, Window: win, At: time.Now(), Hub: k.Hub})
	})
}
