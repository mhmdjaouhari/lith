package town

import (
	"net/http"
	"time"

	"github.com/zweiundeins/lith/writer"

	"github.com/zweiundeins/lith/site/internal/demo"
)

func mount(k *demo.Kit) {
	// The starting town, laid out once, when the writer starts.
	if err := k.W.Send(Seed(time.Now())); err != nil {
		k.Log.Warn("town: seed", "err", err)
	}
	PlaceAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req PlaceReq) {
		if !demo.ValidID(req.Op) {
			w.WriteHeader(http.StatusBadRequest) // no page of ours sends it
			return
		}
		cmd := Place{Viewer: v, Who: whoOf(v.Session), Op: req.Op, Tile: req.Tile, Kind: req.Kind, Seen: req.Seen, At: time.Now()}
		// A retried request (a flaky network) changes the tile once.
		k.Act(w, r, v, k.AckOf(S.Ack.Set(req.Op)), writer.Once(Slug+":"+v.Session+":"+req.Op, cmd))
	})
	// The slider: a position each tick while it is dragged (SeekAction's
	// rate). A notice ends the seek as the tab's view would, by its
	// number.
	SeekAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, req SeekReq) {
		k.Act(w, r, v, k.AckOf(S.Seq.Set(req.Seq)), Seek{Viewer: v, At: req.At, Seq: req.Seq, Now: time.Now(), Hub: k.Hub})
	})
	FollowAction.Mount(k, func(w http.ResponseWriter, r *http.Request, v demo.Viewer, _ FollowReq) {
		k.Act(w, r, v, nil, Follow{Viewer: v, Hub: k.Hub})
	})
}
