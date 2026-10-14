# Shop demonstration

Run `make demo` from the repository root, or `go run ./scripts/demo` when GNU Make is not installed. The portable runner builds the CLI, compiles both nested Go modules and checks the expected exit codes and diagnostics. An expected violation is a successful demonstration; an unexpected exit or missing diagnostic fails the runner.

GNU Make is an optional command wrapper. In the verification environment it is installed at `C:\msys64\usr\bin\make.exe` and is not on `PATH`; from PowerShell invoke `& 'C:\msys64\usr\bin\make.exe' demo`. All recipes use Go commands and have equivalent direct commands in the README.

`examples/shop/clean` and `examples/shop/violating` are separate modules with the same example import path, `example.com/shop`. This is a local demonstration path, not a published repository. Each has its own checked `boundaryguard.json`. Nested modules keep deliberate violations out of the product's `go test ./...` and self-analysis.

The violating module demonstrates:

- `domain/direct → storage`: BG001 at the import, rule `domain-no-storage`.
- `domain/transitive → shared → storage`: BG003 at the import of shared. Shared is intentionally unclassified.
- `transport` captures `http.Get`: BG002, rule `transport-no-global-get`, even without calling it.

The clean module keeps storage access in application/shared. Its domain contains data and transport accepts an explicit HTTP client. Fixing a dependency can mean moving orchestration into application or providing a domain-owned interface; replacing an HTTP global can mean passing a configured client. These design choices require application context, so BoundaryGuard offers no automatic source edits.

For a manual run, build with `make build`, change to either example directory, then run `../../../bin/boundaryguard -config boundaryguard.json ./...` (append `.exe` on Windows). Clean analysis exits 0; diagnostics exit 3 in both text and JSON mode. The CLI preserves the upstream driver's JSON data format and checks its result to provide a failing CI exit status.

The GitHub Actions workflow uses Ubuntu 24.04 and Go 1.26.2, runs the build, tests, formatting/vet/self-analysis, demonstration and race detector. Actions are fixed to the release commits of [checkout v4.2.2](https://github.com/actions/checkout/releases/tag/v4.2.2) and [setup-go v5.5.0](https://github.com/actions/setup-go/releases/tag/v5.5.0). The [first hosted run](https://github.com/nabiatakhanov/boundaryguard/actions/runs/40959561087) passed; local and hosted evidence is recorded in [acceptance.md](acceptance.md). Expected demo diagnostics may produce GitHub annotations while the overall job succeeds.

An adjacent durableq checkout was available for a separate read-only analysis. Its exact revision, configuration, commands and observed result are recorded in [the real-project run](real-project.md). The shop demonstration remains self-contained.
