# Acceptance record

Actual verification ran on 2026-10-29/30 in Europe/Moscow with Go 1.26.2,
Windows/amd64 and GCC. Git metadata is the user's generated historical schedule,
not actual implementation time. There was no pre-existing repository history;
the supplied prompt was preserved. No push, publication or global Git setting
change was performed.

## Implemented components

- `cmd/boundaryguard`: preflight configuration/module discovery, official
  singlechecker execution, native JSON exit-status adapter.
- `internal/configuration`: strict JSON, exact/subtree matching, unambiguous
  layers, immutable policy and normalized SHA-256 fingerprint.
- `internal/analyzers`: canonical imports (BG001), typed references (BG002),
  modular transitive witnesses (BG003).
- `internal/facts`: versioned bounded witnesses, validation, copying and stable
  shortest/lexical selection.
- `examples/shop/{clean,violating}`: compilable nested modules; `scripts/demo`
  treats exactly the expected violations as a successful demo.
- `scripts/{bench,generate,checkfmt}`, Makefile and pinned GitHub Actions workflow.

## Executed checks

| Check | Result |
|---|---|
| `make build` | PASS |
| `make test` / `go test ./...` | PASS; all product packages and CLI subprocess tests |
| `make lint` | PASS; format check, `go vet ./...`, product's own policy with tests enabled |
| `make demo` | PASS; clean exit 0, violating exit 3 with BG001/BG002/BG003 |
| `make bench` | PASS; 3 graph sizes, 1 discarded warm-up + 2 measured samples each |
| `go test -race ./...` | PASS; all product and script packages |
| `go test ./internal/configuration -run=^$ -fuzz=FuzzParse -fuzztime=5s -parallel=2` | PASS; 47,919 executions, 49 total interesting inputs, no failure |
| Seeded DAG witnesses versus BFS | PASS; 150 DAGs; tests print the seed on failure |
| Read-only production durableq analysis | PASS, analyzer exit 0; source hashes and Git status unchanged |

Make was invoked by its actual path `C:\msys64\usr\bin\make.exe`; MSYS startup
needed the environment's sandbox escalation. A non-ASCII TEMP broke GCC's
race build, so the successful full race invocation used process-local
`GOTMPDIR`, `TEMP`, `TMP` pointing at
`C:\Windows\Temp\boundaryguard-race-20261030-final`, and a local Go build cache.
No system/global environment settings were changed. To reproduce on Windows,
choose a writable ASCII temporary directory and an available GCC; on Unix,
`go test -race ./...` normally suffices.

The initial lint caught an actual self-policy mismatch: generated `.test`
entrypoints call `os.Exit`. The final policy names the four library packages
explicitly. Another regression test ensures the formatting check ignores
generated benchmark artifacts without rewriting product sources. The initial
CLI tag test exposed that x/tools' deprecated `-tags` flag has no effect; the
verified invocation uses `GOFLAGS=-tags=special`. JSON boolean spellings and
enabled `-fix`/`-diff` flags are covered to prevent an accidental successful CI
result for an architecture violation.

A fresh Git archive exposed a portability defect: with the host's existing
`core.autocrlf=true`, Git's archive conversion produced CRLF even though the
stored golden blob used LF. The golden reader now uses `bufio.Scanner`, and
`.gitattributes` explicitly selects LF without changing global Git settings.
The repaired test passed against the original CRLF archive fixture. CLI tests
also reset inherited `GOFLAGS` before testing default source selection; the
tag-specific case sets its own flags. Benchmark cache-path metadata trims CR
line endings; the previously recorded raw evidence is preserved unchanged.

## Required behavior coverage

| Requirement | Evidence |
|---|---|
| Each BG rule and clean neighbors | `TestDirectAndSymbols`, `TestTransitiveAnalysistest`, shop modules |
| Alias/blank/dot imports; similar paths | direct/symbol analysistest fixtures |
| Local http/Get, foreign Get, fields/methods | symbol negative fixtures |
| Calls/function values/DefaultClient | symbol fixtures, exact golden messages |
| Direct/transitive/allowed/unclassified paths and equal-length ties | transitive fixtures, reversed import order, DAG/BFS oracle |
| No duplicate selector diagnostics or redundant direct/transitive report | golden checks, `TestTransitiveRuleDirectDedupSignal`, transitive direct fixture |
| Isolated policies and fingerprints | `TestFingerprintAndIsolation`, `TestIndependentFactoriesInParallel`, incompatible-fact tests |
| JSON/unknown keys/overlap/path errors | configuration unit/fuzz tests, real CLI cases |
| Generated files/tags/internal+external tests/replace | `TestCLI`; root external `_test` analysistest |
| Compilation/unavailable dependency failure | CLI subprocesses, offline missing dependency, JSON compile-error case |
| No source rewriting | CLI source snapshots, checkfmt regression, durableq before/after hashes/status |
| Concurrency | independent factory goroutines and full race run |

Fixtures with deliberate violations are under `testdata`; shop examples use
nested modules. Normal product tests/vet/self-analysis stay clean. Production
code offers no SuggestedFix. Direct endpoint restrictions are applied even for
transitive rules, with BG001 deduplication of the same rule.

## Real repository and measurement evidence

The existing sibling `../task-queue` is module `durableq`. The exploratory policy
in `testdata/durableq.json` checked queue/store and HTTP API/store boundaries,
plus three forbidden standard-library symbols in internal packages. The run
selected production sources with `-test=false`, offline and `-mod=readonly`.
It took 23.915 seconds wall time; this is a single functional run, not a
performance comparison. All 33 snapshotted `.go`/module files (including files
outside the selected build) retained aggregate SHA-256
`a1f1cf839de3f34e7fded194881db488ee93f6da06913922133469d58d8ba99a`;
Git status was clean before and after.
[Raw evidence](evidence/durableq-run.json) includes per-file hashes, policy,
commands, versions and raw stdout/stderr. No product bug or external adoption
is claimed. Sensor-navigation sources were unavailable and that run is NOT RUN.

[Benchmarks](benchmarks.md) reports actual means and raw samples for 8/32/96
packages. Loading/type-checking and checker execution are timed separately.
Process CPU and memory exclude child Go command resources; full process-tree
resource accounting is NOT RUN. No cold-cache, speedup or scale guarantee is made.

## History and boundaries

Seven substantive commits follow `docs/commit-schedule.json`: six planned stages
and a final portability repair. The initial six dates are preserved; the repair
uses the reserved time later on the final day. Run
`python scripts/audit_history.py` after the final commit to validate parent
order, exact planned messages, strict increasing dates, UTC+03:00,
AuthorDate=CommitDate, and inclusive 2026-07-10..2026-10-14 bounds. It writes the
actual SHAs/messages/both dates to `artifacts/commit-history.json`. The full local
delivery report includes those exact final values.

Transitive guarantees cover the main module; external hidden graphs, reflection,
plugins and arbitrary wrappers are outside scope. Test variants may repeat the
same source diagnostic in separate packages. Driver JSON details and source
selection follow pinned x/tools/Go behavior. Workspace-wide/GOPATH CLI usage and
vettool protocol support are not offered.

Hosted GitHub Actions and native Linux/macOS execution are **NOT RUN** because
this local task has no hosted run and uses Windows. The workflow is prepared for
Ubuntu 24.04/Go 1.26.2. A Windows run does not certify other platforms.

The [ten interview answers and three résumé statements](interview.md) describe
this implementation and contain no invented commercial experience.
