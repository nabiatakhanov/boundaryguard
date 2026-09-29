package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIgnoresGeneratedArtifacts(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, dir := range []string{"artifacts/bench/graph/p0", "bin", ".git", "vendor/example"} {
		path := filepath.Join(root, filepath.FromSlash(dir))
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "intentionally-invalid.go"), []byte("not Go source"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := check(root); err != nil {
		t.Fatalf("generated outputs must not break repository lint: %v", err)
	}
}

func TestReportsWithoutChangingSource(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := filepath.Join(root, "example.go")
	source := "package example\nfunc Value()int{return 1}\n"
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	if err := check(root); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("want filename in formatting error, got %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != source {
		t.Fatal("format checking changed the source")
	}
}
