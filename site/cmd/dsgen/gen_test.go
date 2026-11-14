package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGenerated fails when a package's generated seam is not what its
// structs say: run go generate in it.
func TestGenerated(t *testing.T) {
	dirs := map[string]bool{}
	err := filepath.WalkDir("../../internal", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if filepath.Base(path) == goFile {
			dirs[filepath.Dir(path)] = true
			return nil
		}
		src, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(src), "\n//ds:") {
			dirs[filepath.Dir(path)] = true
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) == 0 {
		t.Fatal("no package uses dsgen")
	}
	for dir := range dirs {
		if err := Run(dir, true); err != nil {
			t.Errorf("%s: %v", dir, err)
		}
	}
}
