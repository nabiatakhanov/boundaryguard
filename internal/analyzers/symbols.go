package analyzers

import (
	"go/ast"
	"go/types"
	"sort"

	"boundaryguard.local/boundaryguard/internal/configuration"
	"golang.org/x/tools/go/analysis"
)

func checkSymbols(pass *analysis.Pass, cfg *configuration.Config) {
	source := pass.Pkg.Path()
	rules := cfg.ForbiddenSymbols()
	// Uses contains the selected identifier, not its enclosing SelectorExpr.
	// Traversing it once also handles dot imports and function values naturally.
	idents := make([]*ast.Ident, 0, len(pass.TypesInfo.Uses))
	for ident := range pass.TypesInfo.Uses {
		idents = append(idents, ident)
	}
	sort.Slice(idents, func(i, j int) bool { return idents[i].Pos() < idents[j].Pos() })
	for _, ident := range idents {
		obj := pass.TypesInfo.Uses[ident]
		if obj.Pkg() == nil || obj.Parent() != obj.Pkg().Scope() {
			continue
		}
		switch typed := obj.(type) {
		case *types.Func:
			if signature, ok := typed.Type().(*types.Signature); !ok || signature.Recv() != nil {
				continue
			}
		case *types.Var:
			if typed.IsField() {
				continue
			}
		default:
			continue
		}
		fullName := obj.Pkg().Path() + "." + obj.Name()
		for _, rule := range rules {
			if rule.Symbol != fullName || !symbolScopeMatches(rule.Packages, source) {
				continue
			}
			pass.Report(analysis.Diagnostic{
				Pos: ident.Pos(), End: ident.End(), Category: "BG002",
				Message: "BG002 [" + rule.ID + "] " + source + " references forbidden symbol " + fullName,
			})
		}
	}
}

func symbolScopeMatches(patterns []string, source string) bool {
	for _, pattern := range patterns {
		if configuration.Match(pattern, source) {
			return true
		}
	}
	return false
}
