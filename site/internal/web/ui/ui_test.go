package ui

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestTypeCheck runs tsc over the site's TypeScript, lith-ui and the
// demos' controllers, against the interfaces dsgen writes from the Go
// structs. tsc is TypeScript 7's native compiler, a Go tool pinned in
// go.mod, so the check needs no Node and always runs.
func TestTypeCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("tsc: the first go tool tsc builds the compiler")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "tool", "tsc", "-p", root)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("tsc: %v\n%s", err, out)
	}
}

// TestTranspile checks that esbuild strips the types and keeps the
// imports for the page's import map.
func TestTranspile(t *testing.T) {
	js, err := Transpile("lith-ui.ts", Source, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(js), `from"datastar"`) || strings.Contains(string(js), "interface ") {
		t.Errorf("lith-ui.js: %.300s", js)
	}
	if _, err := Transpile("bad.ts", "const x: = 1", true); err == nil || !strings.Contains(err.Error(), "bad.ts") {
		t.Errorf("a syntax error: %v", err)
	}
}
