package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestTypeCheck runs tsc over the site's TypeScript, lith-ui and the
// demos' controllers, against the interfaces dsgen writes from the Go
// structs. It needs the pinned TypeScript: npm ci in site/. CI runs it.
func TestTypeCheck(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	tsc := filepath.Join(root, "node_modules", ".bin", "tsc")
	if _, err := os.Stat(tsc); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("no tsc: run npm ci in site/")
		}
		t.Skip("no tsc: run npm ci in site/ to type-check the TypeScript")
	}
	cmd := exec.Command(tsc, "-p", root)
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
