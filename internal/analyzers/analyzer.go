// Package analyzers implements BoundaryGuard's architecture rules.
package analyzers

import (
	"fmt"
	"strings"

	"boundaryguard.local/boundaryguard/internal/configuration"
	"boundaryguard.local/boundaryguard/internal/facts"
	"golang.org/x/tools/go/analysis"
)

// New creates an independent analyzer for an immutable policy and main module.
// modulePath is resolved by the CLI once, before the analysis driver is started.
func New(config *configuration.Config, modulePath string) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name:      "boundaryguard",
		Doc:       "checks configured Go package boundaries and forbidden symbols",
		FactTypes: []analysis.Fact{new(facts.Reachability)},
	}
	a.Run = func(pass *analysis.Pass) (any, error) {
		if config == nil || modulePath == "" {
			return nil, fmt.Errorf("boundaryguard requires a validated policy and main module path")
		}
		direct := checkDirect(pass, config)
		checkSymbols(pass, config)
		if !withinModule(pass.Pkg.Path(), modulePath) {
			return nil, nil
		}
		if pass.Module != nil && pass.Module.Path != "" && pass.Module.Path != modulePath {
			pass.ExportPackageFact(&facts.Reachability{Version: facts.Schema, Fingerprint: config.Fingerprint()})
			return nil, nil
		}
		return nil, checkTransitive(pass, config, modulePath, direct)
	}
	return a
}

func withinModule(path, module string) bool {
	// The external test package of a module's root has canonical path
	// module_test. Subpackage test variants already match the segment prefix.
	return path == module || path == module+"_test" || strings.HasPrefix(path, module+"/")
}
