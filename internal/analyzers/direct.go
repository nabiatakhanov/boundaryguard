package analyzers

import (
	"go/ast"
	"strconv"

	"boundaryguard.local/boundaryguard/internal/configuration"
	"golang.org/x/tools/go/analysis"
)

// checkDirect returns the IDs for which a direct import already explains the
// violation. The transitive check uses this to avoid a redundant BG003.
func checkDirect(pass *analysis.Pass, cfg *configuration.Config) map[string]bool {
	violated := make(map[string]bool)
	source := pass.Pkg.Path()
	fromLayer := cfg.LayerFor(source)
	if fromLayer == "" {
		return violated
	}
	rules := cfg.DependencyRules()
	for _, file := range pass.Files {
		for _, spec := range file.Imports {
			target := directImportPath(pass, spec)
			toLayer := cfg.LayerFor(target)
			if toLayer == "" {
				continue
			}
			for _, rule := range rules {
				if rule.FromLayer == fromLayer && rule.ToLayer == toLayer {
					pass.Report(analysis.Diagnostic{
						Pos: spec.Pos(), End: spec.End(), Category: "BG001",
						Message: "BG001 [" + rule.ID + "] " + source + " imports forbidden " + target,
					})
					violated[rule.ID] = true
				}
			}
		}
	}
	return violated
}

func directImportPath(pass *analysis.Pass, spec *ast.ImportSpec) string {
	if name := pass.TypesInfo.PkgNameOf(spec); name != nil {
		return name.Imported().Path()
	}
	// Well-typed files have a PkgName even for blank and dot imports. Keep this
	// fallback useful for focused tests and drivers with partial TypesInfo.
	path, _ := strconv.Unquote(spec.Path.Value)
	return path
}
