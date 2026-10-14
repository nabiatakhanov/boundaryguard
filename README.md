# BoundaryGuard

BoundaryGuard checks explicit architecture rules in Go modules:

- **BG001**: forbidden direct imports, including renamed, blank and dot imports.
- **BG002**: references to forbidden package-level functions and variables,
  identified through `go/types`, including function values and dot imports.
- **BG003**: forbidden transitive dependencies, with one shortest, lexically
  stable witness through main-module packages, including unclassified packages.

The CLI uses the official `singlechecker` driver. JSON is the driver's nested
package/analyzer format; a small parent process preserves its bytes and makes
diagnostics fail CI. The tool reads selected Go sources and does not rewrite them.

## Build and use

Go **1.26.2** is the minimum module version and the tested toolchain;
`golang.org/x/tools v0.52.0` is pinned in `go.mod`/`go.sum`. These are actual
implementation/test versions, independent of the requested historical Git dates.
See [official Go release policy](https://go.dev/doc/devel/release) and
[the pinned analysis driver](https://pkg.go.dev/golang.org/x/tools@v0.52.0/go/analysis/singlechecker).
Validation was performed on Windows/amd64; the Ubuntu CI workflow is prepared,
but a hosted run has not been executed.

From this checkout:

```sh
make build
make test
make lint
make demo
make bench
```

GNU Make is optional: equivalent commands are `go build -o bin/ ./cmd/boundaryguard`,
`go test ./...`, `go run ./scripts/checkfmt`, `go vet ./...`,
`bin/boundaryguard -config boundaryguard.json ./...`, `go run ./scripts/demo`,
and `go run ./scripts/bench`. On Windows append `.exe` to executable names;
this host's GNU Make is `C:\msys64\usr\bin\make.exe`. Quote a comma-containing
flag in PowerShell, for example `'-sizes=8,32,96'`.

Install the actual local module's command from this checkout:

```sh
go install boundaryguard.local/boundaryguard/cmd/boundaryguard
```

`go install ./cmd/boundaryguard` is equivalent. The binary goes into `GOBIN` or
`GOPATH/bin`. `boundaryguard.local/boundaryguard` is the real module path in this
repository, not a published service; installation with a remote `@version` is
not offered. No repository has been published or pushed.

Run the binary **inside the target Go module**, supplying its explicit policy:

```sh
/absolute/path/boundaryguard -config boundaryguard.json ./...
/absolute/path/boundaryguard -config boundaryguard.json -json ./...
GOFLAGS=-tags=special /absolute/path/boundaryguard -config boundaryguard.json ./...
```

In PowerShell use `$env:GOFLAGS = '-tags=special'` before the command and remove
it afterwards with `Remove-Item Env:GOFLAGS`. The driver's deprecated `-tags`
flag has no effect. All driver flags precede package patterns. Configuration
paths resolve relative to the invocation directory once. A current single Go
module is required; workspace-wide/GOPATH CLI runs are unsupported.

| Result | Exit |
|---|---:|
| Clean text or JSON analysis; help | 0 |
| Loading, compilation or analyzer failure; missing package patterns | 1 |
| Invalid configuration or flags | 2 |
| Rule diagnostics, text or JSON | 3 |

Treat any nonzero code as CI failure. Combined loading/diagnostic failures remain
nonzero. `-fix` and `-diff` are rejected: architecture changes need a design decision.
`go vet -vettool` compatibility is not claimed; invoke the executable directly.

## Policy and scope

Copy a checked example from [shop/clean](examples/shop/clean/boundaryguard.json)
and follow [configuration.md](docs/configuration.md). Patterns are exact package
paths or `/...` subtrees including their root; segment boundaries matter. All
unlisted dependency directions are allowed. Unclassified packages do not match
layer rules, but carry facts through the main module.

Selected generated files are included. Tests are included by default; use
`-test=false` to omit variants. Repeated source diagnostics across distinct test
package variants are possible. Root external tests have the canonical path
`module_test` and need an explicit policy pattern if they should match a rule.
Go build constraints, platform and `replace` control the loaded source set.

External packages can be classified direct endpoints. Their hidden dependency
graphs are outside the transitive guarantee. References through reflection,
plugins and arbitrary wrappers are outside BG002's guarantee. This tool checks
an explicit policy, not all architecture or security properties.

The repository's [own policy](boundaryguard.json) keeps policy/facts independent
of analyzers and forbids process exit in the four library packages. That rule
uses exact library paths: Go's generated `.test` main packages must call
`os.Exit`. CLI/script entry points own process exit. Deliberately violating
fixtures live under `testdata`, and shop examples are separate nested modules.

## Evidence and implementation

- [Rules and exact messages](docs/rules.md), [architecture](docs/architecture.md),
  [ADR 001](docs/adr/001-policy-and-typed-symbols.md), [ADR 002](docs/adr/002-modular-facts.md).
- [Demo commands and expected results](docs/demo.md).
- [Measured performance and raw samples](docs/benchmarks.md).
- [Read-only durableq experiment](docs/real-project.md).
- [Acceptance report](docs/acceptance.md), [interview answers and résumé facts](docs/interview.md).

Core code is in `internal/configuration`, `internal/analyzers` and `internal/facts`;
the CLI is in `cmd/boundaryguard`. Tests use `analysistest`, golden messages,
subprocesses, seeded DAG/BFS comparisons, fuzzing and the race detector.
