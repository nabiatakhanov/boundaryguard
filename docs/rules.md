# Rule semantics and diagnostics

BoundaryGuard applies explicit prohibitions. A direction or symbol without an
applicable rule is allowed. Layer matching uses canonical import paths and segment
boundaries; import aliases never affect package classification. Rules operate on
the files selected by the Go loader, including generated files by default.

## BG001: forbidden direct import

Every import spec from `from_layer` to `to_layer` produces one diagnostic per
matching dependency rule, for both `direct` and `transitive` kinds. Regular,
renamed, blank and dot imports have identical dependency semantics. Each import
occurrence is a source location; importing the same forbidden package in two files
can produce two diagnostics.

```text
BG001 [domain-storage] example.com/shop/internal/domain imports forbidden example.com/shop/internal/storage
```

The position is the import spec, including its alias when present. Unclassified
source or target packages do not produce BG001. They can still participate in the
BG003 graph. No inverse rule is inferred: forbidding domain → storage does not
forbid storage → domain.

## BG002: forbidden typed symbol reference

For each identifier in `TypesInfo.Uses`, BoundaryGuard checks its `types.Object`.
Only package-scope `*types.Func` values without a receiver and package-scope
`*types.Var` values are eligible. The defining package path and object name must
equal the configured symbol. A rule applies only to source packages within its
configured `packages` patterns.

```text
BG002 [transport-get] example.com/shop/internal/transport references forbidden symbol net/http.Get
```

The position is the referenced identifier (`Get` or `DefaultClient`). Calls,
function values such as `get := http.Get`, and variable references are all checked.
An alias (`web.Get`) or dot import (`Get`) resolves to the same object. Walking
identifiers rather than both identifiers and selectors avoids double reporting
one reference. Multiple explicitly configured rules for the same symbol can each
report their own rule ID.

Local variables named `http`, local functions named `Get`, struct fields,
methods, constants, types, and the `Get` function of another package do not match
`net/http.Get`. A reference to `http.DefaultClient` is checked directly; a different
value of type `http.Client` does not match it. A package's own references to its
configured package-level symbols also match when that package is in scope.

## BG003: forbidden reachable dependency

`transitive` rules reject any reachable target in `to_layer`. In-module package
facts carry one witness per target layer, including through unclassified packages
and permitted intermediate layers. A classified external package can terminate a
witness at its direct import edge. BoundaryGuard does not inspect hidden edges
beyond that external endpoint.

```text
BG003 [domain-storage] forbidden dependency path: example.com/shop/internal/domain -> example.com/shop/internal/shared -> example.com/shop/internal/storage
```

The witness has the fewest import edges; equal-length witnesses are compared
lexicographically by the entire sequence of package paths. The position is the
first import in the source package along that witness. If that package imports
the first step in several files, the earliest source position is used. There is
one BG003 per matching rule, never an enumeration of all paths. If any direct
import already triggers BG001 for that same rule ID, BG003 is suppressed for it.

Malformed or incompatible expected facts cause analysis errors. Missing facts
for expected in-module dependencies do not mean those packages are clean.
External nested modules that share the main module's path prefix are boundaries
when module metadata identifies them separately.

## Output and remediation

Each message starts with its stable BG code and contains the configuration rule
ID. The official driver adds filename, line and column in text output. JSON uses
the driver's package/analyzer structure, diagnostic position, message and category;
see [architecture](architecture.md) for CLI behavior. Exact source positions and
the number of package variants depend on the Go loader. Tests fix the rule
messages and witness selection; driver-wide output ordering and JSON schema are
not a separate BoundaryGuard compatibility promise.

The analyzer produces no `SuggestedFix`. Resolve BG001/BG003 by moving the
dependency behind a suitable application boundary, introducing a domain-owned
interface, or correcting an inaccurate architectural policy. Resolve BG002 by
using the project's injected client, logger or other approved abstraction. The
right change depends on program behavior; automatically deleting an import or
replacing a call would not establish a correct fix.

This tool checks known typed source references and the described modular import
graph. Reflection, plugins, calls through arbitrary external wrappers, runtime
configuration, complete program behavior and general security guarantees are
outside version 1.
