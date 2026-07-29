package analyzers

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"boundaryguard.local/boundaryguard/internal/configuration"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

func directSymbolConfig(t *testing.T) *configuration.Config {
	t.Helper()
	cfg, err := configuration.Parse([]byte(`{
		"version":1,
		"layers":[{"name":"domain","packages":["bgdirect/domain/..."]},{"name":"storage","packages":["bgstorage/..."]}],
		"dependency_rules":[{"id":"no-storage","from_layer":"domain","to_layer":"storage","kind":"direct"}],
		"forbidden_symbols":[
			{"id":"no-get","packages":["bgsymbols/..."],"symbol":"net/http.Get"},
			{"id":"no-default","packages":["bgsymbols/..."],"symbol":"net/http.DefaultClient"},
			{"id":"no-fatal","packages":["bgsymbols/..."],"symbol":"log.Fatal"}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestDirectAndSymbols(t *testing.T) {
	t.Parallel()
	cfg := directSymbolConfig(t)
	analyzer := &analysis.Analyzer{
		Name: "boundaryguarddirectsymbols", Doc: "test direct and symbol checks",
		Run: func(pass *analysis.Pass) (any, error) {
			checkDirect(pass, cfg)
			checkSymbols(pass, cfg)
			return nil, nil
		},
	}
	results := analysistest.Run(t, analysistest.TestData(), analyzer, "bgdirect/...", "bgsymbols/...", "bgsymbolsish")
	var messages []string
	for _, result := range results {
		seen := make(map[string]bool)
		for _, diagnostic := range result.Diagnostics {
			key := result.Pass.Fset.Position(diagnostic.Pos).String() + diagnostic.Message
			if seen[key] {
				t.Errorf("duplicate diagnostic: %s", key)
			}
			seen[key] = true
			if diagnostic.Category == "" || len(diagnostic.SuggestedFixes) != 0 {
				t.Errorf("unexpected diagnostic metadata: %+v", diagnostic)
			}
			messages = append(messages, diagnostic.Message)
		}
	}
	sort.Strings(messages)
	golden, err := os.ReadFile(filepath.Join("testdata", "direct-symbols.golden"))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Split(strings.TrimSpace(string(golden)), "\n")
	if !reflect.DeepEqual(messages, want) {
		t.Errorf("messages differ\ngot: %q\nwant: %q", messages, want)
	}
}

func TestTransitiveRuleDirectDedupSignal(t *testing.T) {
	cfg, err := configuration.Parse([]byte(`{"version":1,"layers":[{"name":"domain","packages":["bgdirect/domain/..."]},{"name":"storage","packages":["bgstorage/..."]}],"dependency_rules":[{"id":"no-storage","from_layer":"domain","to_layer":"storage","kind":"transitive"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	analyzer := &analysis.Analyzer{Name: "boundaryguarddirectdedup", Doc: "test direct dedup signal", Run: func(pass *analysis.Pass) (any, error) {
		violations := checkDirect(pass, cfg)
		if pass.Pkg.Path() == "bgdirect/domain" && !violations["no-storage"] {
			t.Error("direct violation missing dedup signal")
		}
		return nil, nil
	}}
	analysistest.Run(t, analysistest.TestData(), analyzer, "bgdirect/...")
}
