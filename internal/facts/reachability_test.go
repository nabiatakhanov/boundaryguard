package facts

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func TestFactValidation(t *testing.T) {
	valid := Build("m/domain", "policy", []Import{{Path: "m/storage", Layer: "storage"}})
	data, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip Reachability
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatal(err)
	}
	if err := roundtrip.Validate("policy", "m/domain"); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		edit func(*Reachability)
	}{
		{"schema", func(f *Reachability) { f.Version++ }},
		{"fingerprint", func(f *Reachability) { f.Fingerprint = "other" }},
		{"boundary", func(f *Reachability) { f.InModule = false }},
		{"origin", func(f *Reachability) { f.Paths[0].Path[0] = "m/other" }},
		{"cyclic", func(f *Reachability) { f.Paths[0].Path[1] = "m/domain" }},
		{"duplicate layer", func(f *Reachability) { f.Paths = append(f.Paths, f.Paths[0]) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			var f Reachability
			if err := json.Unmarshal(data, &f); err != nil {
				t.Fatal(err)
			}
			test.edit(&f)
			if err := f.Validate("policy", "m/domain"); err == nil {
				t.Fatal("accepted incompatible fact")
			}
		})
	}
}

func TestBuildDoesNotAliasOrHideSameLayer(t *testing.T) {
	child := Build("m/shared", "p", []Import{{Path: "m/storage", Layer: "storage"}})
	parent := Build("m/domain", "p", []Import{{Path: "m/shared", Fact: child}})
	parent.Paths[0].Path[1] = "changed"
	if child.Paths[0].Path[0] != "m/shared" {
		t.Fatal("parent aliases imported fact")
	}
	self := Build("m/domain", "p", []Import{{Path: "m/domain/child", Layer: "domain"}})
	if got := self.Paths[0].Path; !reflect.DeepEqual(got, []string{"m/domain", "m/domain/child"}) {
		t.Fatalf("same-layer import hidden by self path: %v", got)
	}
}

func TestRandomDAGAgainstBFS(t *testing.T) {
	const seed int64 = 20261029
	rng := rand.New(rand.NewSource(seed))
	for graph := 0; graph < 150; graph++ {
		n := 2 + rng.Intn(25)
		names := make([]string, n)
		layers := make([]string, n)
		edges := make([][]int, n)
		for i := range names {
			names[i] = fmt.Sprintf("m/p%02d", i)
			if rng.Intn(3) != 0 {
				layers[i] = fmt.Sprintf("layer%d", rng.Intn(4))
			}
			for j := i + 1; j < n; j++ {
				if rng.Intn(5) == 0 {
					edges[i] = append(edges[i], j)
				}
			}
		}
		computed := make([]*Reachability, n)
		for i := n - 1; i >= 0; i-- {
			var imports []Import
			for _, j := range edges[i] {
				imports = append(imports, Import{Path: names[j], Layer: layers[j], Fact: computed[j]})
			}
			// Scramble input order to catch accidental map/import-order dependence.
			rng.Shuffle(len(imports), func(a, b int) { imports[a], imports[b] = imports[b], imports[a] })
			computed[i] = Build(names[i], "p", imports)
			want := bfs(i, edges, names, layers)
			if !reflect.DeepEqual(computed[i].Paths, want) {
				t.Fatalf("seed=%d graph=%d node=%d edges=%v\ngot=%v\nwant=%v", seed, graph, i, edges, computed[i].Paths, want)
			}
		}
	}
}

// bfs independently visits vertices by shortest, lexicographic path. It keeps
// only the first path to each vertex, then the first endpoint for each layer.
func bfs(start int, edges [][]int, names, layers []string) []Witness {
	type item struct {
		node int
		path []string
	}
	queue := []item{{start, []string{names[start]}}}
	visited := make(map[int]bool)
	best := make(map[string][]string)
	for len(queue) != 0 {
		x := queue[0]
		queue = queue[1:]
		if visited[x.node] {
			continue
		}
		visited[x.node] = true
		if x.node != start && layers[x.node] != "" && best[layers[x.node]] == nil {
			best[layers[x.node]] = x.path
		}
		for _, next := range edges[x.node] {
			path := append(append([]string(nil), x.path...), names[next])
			queue = append(queue, item{next, path})
		}
	}
	var keys []string
	for k := range best {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var result []Witness
	for _, k := range keys {
		result = append(result, Witness{Layer: k, Path: best[k]})
	}
	return result
}

func BenchmarkBuild(b *testing.B) {
	imports := make([]Import, 16)
	for i := range imports {
		imports[i] = Import{Path: fmt.Sprintf("m/p%d", i), Layer: fmt.Sprintf("layer%d", i%4)}
	}
	b.ReportAllocs()
	for b.Loop() {
		Build("m/root", "policy", imports)
	}
}
