package demo

import (
	"io/fs"

	"github.com/a-h/templ"
)

// Attrs joins sets of attributes into one, a later key over an earlier
// one: the posts of a stage's actions (Command.On), spread on it at once.
func Attrs(sets ...templ.Attributes) templ.Attributes {
	out := templ.Attributes{}
	for _, set := range sets {
		for k, v := range set {
			out[k] = v
		}
	}
	return out
}

// MustSub is the subtree dir of fsys, a demo's embedded migrations
// folder: a dir the embed lacks is a bug of the build, so it panics.
func MustSub(fsys fs.FS, dir string) fs.FS {
	f, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return f
}
