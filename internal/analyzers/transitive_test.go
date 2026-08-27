package analyzers

import (
	"go/types"
	"strings"
	"sync"
	"testing"

	"boundaryguard.local/boundaryguard/internal/configuration"
	"boundaryguard.local/boundaryguard/internal/facts"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

func transitiveConfig(t *testing.T) *configuration.Config {
	t.Helper()
	c, err := configuration.Parse([]byte(`{
		"version":1,
		"layers":[
			{"name":"domain","packages":["bg/domain","bg/direct","bg/domainunknown","bg/clean"]},
			{"name":"storage","packages":["bg/storage/...","bg/allowed"]},
			{"name":"application","packages":["bg/shared/a"]}
		],
		"dependency_rules":[{"id":"no-storage","from_layer":"domain","to_layer":"storage","kind":"transitive"}],
		"forbidden_symbols":[]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestTransitiveAnalysistest(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(transitiveConfig(t), "bg"), "bg/domain", "bg/direct", "bg/domainunknown", "bg/clean", "bg/allowed")
}

func TestRootExternalTestPackage(t *testing.T) {
	config, err := configuration.Parse([]byte(`{
		"version":1,
		"layers":[
			{"name":"domain","packages":["bgroot_test"]},
			{"name":"storage","packages":["bgroot/storage"]}
		],
		"dependency_rules":[{"id":"root-test-storage","from_layer":"domain","to_layer":"storage","kind":"transitive"}],
		"forbidden_symbols":[]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	analysistest.Run(t, analysistest.TestData(), New(config, "bgroot"), "bgroot")
}

func TestMissingAndIncompatibleFactsFail(t *testing.T) {
	config := transitiveConfig(t)
	for _, test := range []struct {
		name string
		fact *facts.Reachability
		want string
	}{
		{"missing", nil, "missing BoundaryGuard fact"},
		{"schema", &facts.Reachability{Version: facts.Schema + 1, Fingerprint: config.Fingerprint(), InModule: true}, "schema"},
		{"policy", &facts.Reachability{Version: facts.Schema, Fingerprint: "different", InModule: true}, "fingerprint"},
		{"endpoint", &facts.Reachability{Version: facts.Schema, Fingerprint: config.Fingerprint(), InModule: true, Paths: []facts.Witness{{Layer: "storage", Path: []string{"bg/unknown", "bg/harmless"}}}}, "endpoint"},
	} {
		t.Run(test.name, func(t *testing.T) {
			pkg := types.NewPackage("bg/domain", "domain")
			pkg.SetImports([]*types.Package{types.NewPackage("bg/unknown", "unknown")})
			pass := &analysis.Pass{Pkg: pkg, ImportPackageFact: func(_ *types.Package, f analysis.Fact) bool {
				if test.fact == nil {
					return false
				}
				*f.(*facts.Reachability) = *test.fact
				return true
			}}
			err := checkTransitive(pass, config, "bg", nil)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want error containing %s", err, test.want)
			}
		})
	}
}

func TestBoundaryModuleDoesNotTraverse(t *testing.T) {
	config := transitiveConfig(t)
	pkg := types.NewPackage("bg/nested", "nested")
	pkg.SetImports([]*types.Package{types.NewPackage("bg/nested/hidden", "hidden")})
	var exported *facts.Reachability
	pass := &analysis.Pass{
		Pkg:               pkg,
		Module:            &analysis.Module{Path: "bg/nested"},
		TypesInfo:         &types.Info{},
		Report:            func(analysis.Diagnostic) {},
		ExportPackageFact: func(f analysis.Fact) { exported = f.(*facts.Reachability) },
		ImportPackageFact: func(*types.Package, analysis.Fact) bool { t.Fatal("crossed module boundary"); return false },
	}
	if _, err := New(config, "bg").Run(pass); err != nil {
		t.Fatal(err)
	}
	if exported == nil || exported.InModule || len(exported.Paths) != 0 {
		t.Fatalf("invalid boundary fact: %+v", exported)
	}
}

func TestIndependentFactoriesInParallel(t *testing.T) {
	first := transitiveConfig(t)
	second, err := configuration.Parse([]byte(`{"version":1,"layers":[],"dependency_rules":[],"forbidden_symbols":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if first.Fingerprint() == second.Fingerprint() {
		t.Fatal("different policies have identical fingerprints")
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		for _, config := range []*configuration.Config{first, second} {
			wg.Add(1)
			go func(config *configuration.Config) {
				defer wg.Done()
				var exported *facts.Reachability
				pass := &analysis.Pass{
					Pkg:               types.NewPackage("bg/empty", "empty"),
					TypesInfo:         &types.Info{},
					Report:            func(analysis.Diagnostic) { t.Error("unexpected diagnostic") },
					ExportPackageFact: func(f analysis.Fact) { exported = f.(*facts.Reachability) },
				}
				if _, err := New(config, "bg").Run(pass); err != nil {
					t.Error(err)
				}
				if exported == nil || exported.Fingerprint != config.Fingerprint() {
					t.Errorf("cross-policy contamination: %+v", exported)
				}
			}(config)
		}
	}
	wg.Wait()
}
