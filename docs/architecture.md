# Architecture

BoundaryGuard is one `go/analysis` analyzer with three diagnostic categories.
`cmd/boundaryguard` handles policy preflight and delegates flag parsing, package
loading, type checking, analysis scheduling, fact transport and native diagnostic
formatting to the official `singlechecker` driver. It has no custom Go loader,
type checker, SSA graph, server or database.

## Components and data flow

1. The CLI resolves and validates the explicit configuration before starting the
   analysis driver. `go list -m -json` identifies the module of the invocation
   directory. CLI analysis requires one current Go module; a workspace-wide or
   GOPATH invocation is unsupported.
2. `internal/configuration` produces an immutable policy with a normalized SHA-256
   fingerprint. It knows policy syntax and package patterns, not ASTs or drivers.
3. `internal/analyzers.New(config, modulePath)` creates a fresh analyzer closure.
   No mutable policy maps or run results are shared globally. `go/analysis` gives
   each package a pass with syntax, type information and dependency facts.
4. The analyzer checks direct imports and typed symbol uses, then assembles
   bounded reachability witnesses for in-module packages. It reports diagnostics
   and exports a completed package fact.
5. `internal/facts` compares and combines paths without AST or driver dependency.
   The analyzer adapter imports, validates and exports those facts.

Dependency facts run before importers as scheduled by the driver. They include
unclassified in-module packages, so putting a storage import in `shared` does not
hide it from domain's transitive rule. Intermediate allowed dependencies do not
stop propagation.

## Fact contract

`Reachability` has schema `Version: 1`, the complete policy `Fingerprint`, an
`InModule` marker, and sorted `Paths`. Each `Witness` contains a target `Layer` and
the full import `Path`, starting with the exporting package. Every path has at
least one edge. For each configured target layer, only the shortest path survives;
equal lengths use lexicographic comparison of the package path sequence.

The number of witnesses per package is bounded by the number of layers. Path
lengths can still grow with graph depth, so this bounds path alternatives, not
total bytes independently of module size. The dynamic program considers direct
endpoints and prepends the current package to imported witnesses. In a valid Go
import DAG, this yields the same shortest witness as a BFS with the same tie
ordering; randomized DAG tests compare it to that oracle and print their seed on
failure.

Facts use exported serializable fields and sorted slices instead of maps. The
builder copies path slices. Exported facts are never mutated. Validation rejects
unknown schemas, different policy fingerprints, unsorted or repeated target
layers, missing roots, empty/cyclic paths and endpoints inconsistent with the
configured layer. A missing expected internal fact fails analysis explicitly.

The CLI's module path and the driver's `Pass.Module` metadata establish the
boundary. Packages in an independently versioned nested module with the same path
prefix export an empty boundary marker, rather than their hidden edges. Packages
outside the main module path are direct endpoints only, and do not require facts.
This includes the standard library. Focused `analysistest` runs without module
metadata use exact module-path-or-slash-prefix membership, also recognizing the
external root test package `modulePath_test` as belonging to the module. A policy
pattern `modulePath/...` still does not match that test package's different
canonical path; include `modulePath_test` explicitly when it needs a scope.
Guaranteeing traversal
of every external module is deliberately outside this contract.

## Source selection

The official Go loader selects package files for the active GOOS/GOARCH, module
requirements, local `replace` directives and build tags. Use `./...` for the
current module's packages, set `GOFLAGS=-tags=tag1,tag2` for additional tags and
use `-test=false` to omit test variants. The pinned driver's legacy `-tags` flag
has no effect; use the Go environment setting instead. The driver includes tests
by default, including internal
and external test packages and generated test harnesses. Consequently the same
source violation may appear in more than one package variant; JSON package keys
distinguish them. BoundaryGuard does not merge different analysis passes.

Generated Go source is included by default and its imports remain graph edges.
Files excluded by the Go build constraints are not analyzed; malformed or
unavailable selected dependencies cause a failed invocation. Source files are
read, never rewritten. There is no suppression syntax, generated-file exemption
or architecture autofix in version 1.

## CLI and CI behavior

Native text comes from `singlechecker`. With `-json`, stdout is the unmodified
driver JSON object keyed by package and analyzer. It is not a flat diagnostics
array. Filename spelling, package test-variant keys, ordering between packages
and future driver JSON details are driver concerns. BoundaryGuard keeps its own
message wording, category codes and witness choices deterministic.

The pinned driver's native JSON diagnostics can exit successfully. For CI, a
small parent invocation captures that JSON, writes it unchanged and maps a
nonempty diagnostic array to exit 3 and an analyzer error object to exit 1. The
child process uses the same binary and official loading/analysis path. This adds
process startup and JSON buffering to JSON-mode cost. Text mode uses the driver
directly.

The supported CLI contract is exit 0 for a clean run, 3 for rule diagnostics,
1 for loading/compilation or analyzer failures, and 2 for invalid configuration
or parameters. Help exits successfully. Subprocess tests verify actual behavior,
including native JSON, compilation errors and unavailable dependencies. If a
load error and diagnostics coexist, the invocation remains unsuccessful; callers
should treat every nonzero result as CI failure instead of depending on an
exhaustive combined-failure priority order.

`go vet -vettool` is not a supported integration: this CLI requires launch-time
configuration and module preflight, and no vet protocol compatibility is claimed.
Use the direct executable in CI. See the [ADR on modular
facts](adr/002-modular-facts.md) for the reasoning behind this design.
