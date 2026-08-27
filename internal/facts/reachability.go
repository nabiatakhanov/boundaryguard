// Package facts stores bounded witnesses for modular dependency analysis.
package facts

import (
	"fmt"
	"sort"
	"strings"
)

// Schema is incremented whenever the serialized fact contract changes.
const Schema = 1

// Witness is one shortest, lexicographically stable dependency path to Layer.
// Paths include the exporting package and contain at least one import edge.
type Witness struct {
	Layer string
	Path  []string
}

// Reachability contains at most one witness per configured layer. A boundary
// fact identifies a package in another module that shares the main path prefix.
// Exporters must not mutate any part of a fact after ExportPackageFact.
type Reachability struct {
	Version     int
	Fingerprint string
	InModule    bool
	Paths       []Witness
}

func (*Reachability) AFact() {}

func (f *Reachability) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "reachability(v%d,module=%t", f.Version, f.InModule)
	for _, w := range f.Paths {
		fmt.Fprintf(&b, ";%s=%s", w.Layer, strings.Join(w.Path, " -> "))
	}
	b.WriteByte(')')
	return b.String()
}

// Validate rejects incompatible or malformed facts instead of treating them as
// empty reachability. Policy membership of endpoint layers is checked by callers.
func (f *Reachability) Validate(policy, origin string) error {
	if f.Version != Schema {
		return fmt.Errorf("fact for %s has schema %d, want %d", origin, f.Version, Schema)
	}
	if f.Fingerprint != policy {
		return fmt.Errorf("fact for %s has a different policy fingerprint", origin)
	}
	if !f.InModule && len(f.Paths) != 0 {
		return fmt.Errorf("boundary fact for %s contains dependency paths", origin)
	}
	previous := ""
	for _, w := range f.Paths {
		if w.Layer == "" || w.Layer <= previous || len(w.Path) < 2 || w.Path[0] != origin {
			return fmt.Errorf("fact for %s has malformed or unsorted witnesses", origin)
		}
		previous = w.Layer
		seen := make(map[string]bool, len(w.Path))
		for _, p := range w.Path {
			if p == "" || seen[p] {
				return fmt.Errorf("fact for %s has an empty or cyclic path", origin)
			}
			seen[p] = true
		}
	}
	return nil
}

// Import describes a direct edge, with an optional validated internal fact.
type Import struct {
	Path  string
	Layer string
	Fact  *Reachability
}

// Build computes witnesses from direct edges and immutable dependency facts.
// Every candidate is copied, so its result never aliases an imported fact.
func Build(origin, policy string, imports []Import) *Reachability {
	best := make(map[string][]string)
	consider := func(layer string, path []string) {
		if layer == "" {
			return
		}
		if old, ok := best[layer]; !ok || Better(path, old) {
			best[layer] = append([]string(nil), path...)
		}
	}
	for _, edge := range imports {
		consider(edge.Layer, []string{origin, edge.Path})
		if edge.Fact == nil || !edge.Fact.InModule {
			continue
		}
		for _, w := range edge.Fact.Paths {
			path := make([]string, 1, len(w.Path)+1)
			path[0] = origin
			path = append(path, w.Path...)
			consider(w.Layer, path)
		}
	}
	layers := make([]string, 0, len(best))
	for layer := range best {
		layers = append(layers, layer)
	}
	sort.Strings(layers)
	f := &Reachability{Version: Schema, Fingerprint: policy, InModule: true}
	for _, layer := range layers {
		f.Paths = append(f.Paths, Witness{Layer: layer, Path: best[layer]})
	}
	return f
}

// Better compares first by edge count and then lexicographically by full path.
func Better(candidate, current []string) bool {
	if len(candidate) != len(current) {
		return len(candidate) < len(current)
	}
	for i := range candidate {
		if candidate[i] != current[i] {
			return candidate[i] < current[i]
		}
	}
	return false
}
