package synthetic

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGenerate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "module")
	if err := Generate(dir, 8, 2); err != nil {
		t.Fatal(err)
	}
	c := exec.Command("go", "test", "./...")
	c.Dir = dir
	c.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off")
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if err := Generate(dir, 8, 2); err == nil {
		t.Fatal("overwrote existing directory")
	}
	if err := Generate(filepath.Join(t.TempDir(), "invalid"), 1, 0); err == nil {
		t.Fatal("accepted invalid size")
	}
}
