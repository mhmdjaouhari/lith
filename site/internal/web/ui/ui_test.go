package ui

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
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

// TestBundle checks that a demo's module and what it imports beside it
// become one module, the import map's modules left as imports.
func TestBundle(t *testing.T) {
	fsys := fstest.MapFS{
		"demo.ts":   {Data: []byte("import { effect } from 'datastar'\nimport { must } from 'lith-ui'\nimport { ClassX, type Roots } from './ds.gen'\nconst r: Roots | null = null\neffect(() => must(document, '.' + ClassX))\nexport { r }\n")},
		"ds.gen.ts": {Data: []byte("export interface Roots { _x: number }\nexport const ClassX = \"x-thing\"\n")},
		"bad.ts":    {Data: []byte("import { y } from 'elsewhere'\nconsole.log(y)\n")},
	}
	js, err := Bundle(fsys, "demo.ts", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`from "datastar"`, `from "lith-ui"`, `"x-thing"`} {
		if !strings.Contains(string(js), want) {
			t.Errorf("no %s in:\n%s", want, js)
		}
	}
	if strings.Contains(string(js), `from "./ds.gen"`) || strings.Contains(string(js), "interface") {
		t.Errorf("the generated module is not inlined:\n%s", js)
	}
	if _, err := Bundle(fsys, "bad.ts", true); err == nil || !strings.Contains(err.Error(), "elsewhere") {
		t.Errorf("a bare import outside the import map: %v", err)
	}
}
