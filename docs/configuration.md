# Configuration

Pass `-config path/to/boundaryguard.json` explicitly. There is no implicit file
discovery or empty-policy fallback. The CLI resolves this path against its launch
directory once, loads and validates it before package analysis, and shares the
immutable result with the analyzer. A missing file, malformed JSON or invalid
policy fails the invocation.

The checked-in [`testdata/config/valid.json`](../testdata/config/valid.json) is a
validated example of the complete version 1 format:

```json
{
  "version": 1,
  "layers": [
    {"name": "domain", "packages": ["example.com/shop/internal/domain/..."]},
    {"name": "storage", "packages": ["example.com/shop/internal/storage/..."]}
  ],
  "dependency_rules": [
    {"id": "domain-storage", "from_layer": "domain", "to_layer": "storage", "kind": "transitive"}
  ],
  "forbidden_symbols": [
    {"id": "no-http-get", "packages": ["example.com/shop/internal/transport/..."], "symbol": "net/http.Get"}
  ]
}
```

`version` must be `1`. The three collection fields may be omitted or empty when
unused. Every layer and symbol rule needs at least one package pattern. Layer
names and rule IDs use nonempty ASCII letters, digits, `_`, `-`, or `.`. Rule IDs
are unique across dependency and symbol rules; layer names are unique separately.

`packages` always describes canonical Go import paths. An exact path matches
only that package. A path ending in `/...` matches its root and descendants:
`example.com/shop/domain/...` includes `example.com/shop/domain` and
`example.com/shop/domain/model`, but excludes `example.com/shop/domainish`.
There are no arbitrary globs or regular expressions. Relative paths, empty path
segments, dot segments, whitespace, backslashes, wildcards, and duplicate patterns
are rejected. Patterns in different layers must never overlap, even if an overlap
does not occur in the selected source tree. Overlap inside one layer is harmless.

Dependency rules forbid the specified direction only. `kind: "direct"` examines
immediate imports. `kind: "transitive"` forbids both immediate and reachable
dependencies; an immediate violation produces BG001 and an indirect violation
BG003. Allowed directions are implicit: an edge without a matching prohibition
is allowed. Unclassified packages are allowed and still carry in-module
transitive facts, so they cannot hide a forbidden dependency.

Symbol rules apply only when the importing package matches a rule's `packages`.
`symbol` uses `package-path.Identifier` with the final dot separating the package
path from the Go identifier. Version 1 checks package-level functions and variables,
including saved function values and variable reads; it does not match methods,
fields, types, constants or local declarations. Parsing checks the spelling;
type information determines the actual object during analysis. An unused or
nonexistent configured symbol is allowed, because projects need not import every
package named by policy. `net/http.Get`, `net/http.DefaultClient`, and `log.Fatal`
are valid examples.

Unknown fields (including incorrect capitalization), repeated JSON object keys,
trailing JSON values, duplicate rule IDs, missing layer references, unsupported
rule kinds and ambiguous layers are configuration errors. Collections are sorted
after validation; every getter returns a copy. SHA-256 of the normalized complete
policy is the fact fingerprint. Reordering layers, rules or patterns does not
change it; changing a rule does. This is an identity check for analysis facts,
not a security signature.
