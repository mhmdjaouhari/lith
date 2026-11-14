// Command dsgen writes the typed side of a package's Datastar seam from its
// Go structs, so that the page, the server and the controller share one
// definition. Run it in the package's directory (go:generate):
//
//	//go:generate go run github.com/zweiundeins/lith/site/cmd/dsgen
//
// It reads two directives from type declarations' comments:
//
//	//ds:signals [root var]
//	type Signals struct{ ... }
//
// writes SignalsRefs, a struct with a ds.Ref per field (nested structs
// get their own refs, embedded ones are promoted), and NewSignalsRefs,
// which builds them under a path. With root and var it also writes
// `var <var> = NewSignalsRefs("<root>")`.
//
//	//ds:command <slug> <name>
//	type MoveReq struct{ Tab string `json:"tab"`; ... }
//
// marks the payload of the action POST /cmd/<slug>/<name>: it writes the
// method demo.Command reads the route from.
//
// Into ds_gen.go goes the Go, into ds.gen.d.ts the same structs as
// TypeScript interfaces, and the commands' payloads as the map of the
// DOM events a controller dispatches to send them (<slug>-<name>), for
// tsc to check the controller against. -check reports whether both files
// are current instead of writing them (TestGenerated runs it).
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	check := flag.Bool("check", false, "fail when the generated files are not current, instead of writing them")
	flag.Parse()
	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}
	if err := Run(dir, *check); err != nil {
		fmt.Fprintln(os.Stderr, "dsgen:", err)
		os.Exit(1)
	}
}
