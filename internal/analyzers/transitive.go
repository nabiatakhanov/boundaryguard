package analyzers

import (
	"fmt"
	"go/token"
	"slices"
	"sort"
	"strings"

	"boundaryguard.local/boundaryguard/internal/configuration"
	"boundaryguard.local/boundaryguard/internal/facts"
	"golang.org/x/tools/go/analysis"
)

func checkTransitive(pass *analysis.Pass, config *configuration.Config, modulePath string, direct map[string]bool) error {
	origin := pass.Pkg.Path()
	policy := config.Fingerprint()
	imports := slices.Clone(pass.Pkg.Imports())
	sort.Slice(imports, func(i, j int) bool { return imports[i].Path() < imports[j].Path() })
	edges := make([]facts.Import, 0, len(imports))
	for _, imported := range imports {
		edge := facts.Import{Path: imported.Path(), Layer: config.LayerFor(imported.Path())}
		if withinModule(imported.Path(), modulePath) {
			var f facts.Reachability
			if !pass.ImportPackageFact(imported, &f) {
				return fmt.Errorf("missing BoundaryGuard fact for internal package %s", imported.Path())
			}
			if err := f.Validate(policy, imported.Path()); err != nil {
				return err
			}
			for _, witness := range f.Paths {
				if config.LayerFor(witness.Path[len(witness.Path)-1]) != witness.Layer {
					return fmt.Errorf("fact for %s has an endpoint inconsistent with layer %s", imported.Path(), witness.Layer)
				}
				for _, intermediate := range witness.Path[:len(witness.Path)-1] {
					if !withinModule(intermediate, modulePath) {
						return fmt.Errorf("fact for %s crosses the module boundary at %s", imported.Path(), intermediate)
					}
				}
			}
			edge.Fact = &f
		}
		edges = append(edges, edge)
	}
	f := facts.Build(origin, policy, edges)
	positions := importPositions(pass)
	from := config.LayerFor(origin)
	for _, rule := range config.DependencyRules() {
		if rule.Kind != "transitive" || rule.FromLayer != from || direct[rule.ID] {
			continue
		}
		for _, witness := range f.Paths {
			if witness.Layer != rule.ToLayer || len(witness.Path) < 3 {
				continue
			}
			pos, ok := positions[witness.Path[1]]
			if !ok {
				return fmt.Errorf("cannot locate first import in witness for rule %s", rule.ID)
			}
			pass.Report(analysis.Diagnostic{Pos: pos, Category: "BG003", Message: fmt.Sprintf("BG003 [%s] forbidden dependency path: %s", rule.ID, strings.Join(witness.Path, " -> "))})
		}
	}
	pass.ExportPackageFact(f)
	return nil
}

// importPositions deterministically chooses the first source import for a path.
func importPositions(pass *analysis.Pass) map[string]token.Pos {
	positions := make(map[string]token.Pos)
	for _, file := range pass.Files {
		for _, spec := range file.Imports {
			path := directImportPath(pass, spec)
			if old, ok := positions[path]; !ok || spec.Pos() < old {
				positions[path] = spec.Pos()
			}
		}
	}
	return positions
}
