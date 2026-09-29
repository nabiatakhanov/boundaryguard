// Package synthetic creates reproducible acyclic Go package graphs for experiments.
package synthetic

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Generate requires a new directory: an existing tree is never overwritten.
func Generate(dir string, size, fanout int) error {
	if size < 2 || fanout < 1 {
		return fmt.Errorf("size >= 2 and fanout >= 1 required")
	}
	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}
	write := func(name, data string) error { return os.WriteFile(filepath.Join(dir, name), []byte(data), 0644) }
	if err := write("go.mod", "module bench.test/graph\n\ngo 1.26.0\n"); err != nil {
		return err
	}
	for i := 0; i < size; i++ {
		name := fmt.Sprintf("p%04d", i)
		if err := os.Mkdir(filepath.Join(dir, name), 0755); err != nil {
			return err
		}
		var src strings.Builder
		fmt.Fprintf(&src, "package %s\n", name)
		for j := i + 1; j < size && j <= i+fanout; j++ {
			fmt.Fprintf(&src, "import _ %q\n", fmt.Sprintf("bench.test/graph/p%04d", j))
		}
		if err := write(filepath.Join(name, "graph.go"), src.String()); err != nil {
			return err
		}
	}
	policy := map[string]any{"version": 1, "layers": []any{map[string]any{"name": "entry", "packages": []string{"bench.test/graph/p0000"}}, map[string]any{"name": "target", "packages": []string{fmt.Sprintf("bench.test/graph/p%04d", size-1)}}}, "dependency_rules": []any{map[string]any{"id": "synthetic", "from_layer": "entry", "to_layer": "target", "kind": "transitive"}}, "forbidden_symbols": []any{}}
	data, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return err
	}
	return write("boundaryguard.json", string(data)+"\n")
}
