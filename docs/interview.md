# Implementation interview notes

1. **Why compare imports through `types.PkgName`?** `TypesInfo.PkgNameOf(spec)`
   yields the imported package independently of its normal, renamed, blank or dot
   spelling. BG001 reports at that import spec and compares its canonical path.
2. **Why does `get := http.Get` trigger BG002?** The check iterates
   `TypesInfo.Uses`, not calls. The `Get` identifier resolves to the package-scope
   `*types.Func`; assigning the function still references the forbidden object.
3. **How are shadowing and methods excluded?** The object must be a function or
   variable whose parent is its package scope. Methods have receivers; fields,
   locals and package names do not qualify. The owning package path and object
   name form the compared symbol identity.
4. **How does an unclassified shared package carry violations?** Every main-module
   package exports reachability facts, irrespective of whether it belongs to a
   layer. Classification controls rules and endpoints, not graph traversal.
5. **Why keep one path per layer?** Prepending the same importer preserves shortest
   length and lexical ordering. The best witness from each dependency is enough
   to select the best witness in the importer; storing all paths would grow
   exponentially in some DAGs. The bound is on alternatives, not total path bytes.
6. **How is determinism tested?** Imported packages and target layers are sorted,
   complete path sequences break equal-length ties, golden messages are compared,
   import order is reversed in a fixture, and seeded DAGs are checked against BFS.
7. **What prevents facts from a different policy being accepted?** The validated,
   normalized complete configuration has a SHA-256 fingerprint. Each fact carries
   it and a schema version; mismatches and missing expected facts fail analysis.
8. **Where does modular analysis stop?** Main-module packages propagate facts;
   nested independently versioned modules export boundary markers. External
   packages can be classified direct endpoints, but hidden external edges are
   outside the guarantee. No reflection, plugin or arbitrary wrapper proof is made.
9. **Why wrap JSON execution?** The pinned official driver returns zero when
   reporting JSON diagnostics. A parent preserves the exact native JSON bytes,
   then maps diagnostics to exit 3 and error objects to exit 1. Boolean flag
   spellings such as `-json=1` are regression-tested. There is buffering and a
   process startup cost.
10. **What would need care before adding more symbol kinds or caching?** Methods
    need an explicitly defined receiver identity and interface semantics; aliases
    alone cannot replace object identity. Persistent fact caching must preserve
    policy/schema keys, source/build selection and module boundaries. Version 1
    deliberately has neither an expanded symbol model nor a custom cache.

# Factual résumé statements

- Implemented a Go CLI on `go/analysis` for direct/transitive architecture
  boundaries and forbidden package-level symbol references using `go/types`.
- Added deterministic bounded package facts with schema/policy validation and
  tests for false positives, shadowing, import aliases and shortest witnesses.
- Prepared compilable clean/violating modules, subprocess CI acceptance tests,
  race/fuzz checks and a reproducible benchmark harness separating package
  loading from analysis scheduling/fact propagation.
