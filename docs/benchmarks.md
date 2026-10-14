# Reproducible measurement

Run `make bench` or `go run ./scripts/bench '-sizes=8,32,96' -fanout=2 -repeats=2`.
Each invocation creates a new timestamped directory below `artifacts/bench`;
existing results are preserved. It saves generated modules/configurations,
each raw sample including discarded warm-ups, and `results.json` with toolchain,
build/dependency information and cache paths. For an independent graph use
`go run ./scripts/generate -out=/new/path -packages=96 -fanout=2` (parent exists).
The generator refuses to overwrite an existing graph.

The graph is a reproducible DAG: package i imports the next `fanout` packages,
within bounds. Entry p0000 is forbidden from reaching the final target package.
There is one expected diagnostic. No cyclic Go packages are generated.

## Recorded run

Actual execution: **2026-10-30, Europe/Moscow** (raw timestamps use UTC),
Windows/amd64, Go 1.26.2, x/tools v0.52.0, 16 logical CPUs. The code was executed
in the working tree before the final documentation commit; raw build information
records compiler/dependency versions and build settings. Historical Git dates are unrelated.
The run used an explicit local `artifacts/go-cache`, warm module/build caches,
and discarded one warm-up per graph. Other verification commands had finished.
No cache eviction or cold-cache claim is made. An earlier concurrent exploratory
run is excluded from this table and the checked-in evidence.

Values below are means of **two** measured subprocesses per graph; memory is
the maximum across those two. Such a small experiment does not establish
statistical significance, speedup or production capacity.

| Packages | Edges | Load/type-check ms | Analyze ms | Process wall ms | Process CPU ms | Peak working set MiB |
|---:|---:|---:|---:|---:|---:|---:|
| 8 | 13 | 357.02 | 0.54 | 384.16 | 70.31 | 13.09 |
| 32 | 61 | 757.03 | 1.09 | 788.12 | 85.94 | 14.38 |
| 96 | 189 | 1549.77 | 5.62 | 1583.59 | 132.81 | 17.86 |

Raw metadata and all samples are in [evidence/benchmark](evidence/benchmark/results.json);
`policy-8.json`, `policy-32.json` and `policy-96.json` preserve the exact policies.
The generator plus package count/fanout reproduces source contents.

## Measurement boundaries

Each worker uses the same official loading profile as the pinned fact-aware
driver: `packages.LoadAllSyntax | packages.NeedModule`, tests enabled. It times
that standard library call separately from public `checker.Analyze` with the
production analyzer. Analyze time includes scheduling, imported fact transport,
validation and the rules; it is not a count of only AST instructions. Diagnostic
printing and JSON CLI parent buffering are outside that stage. Process wall time
also includes worker startup and initialization. This harness is an experiment,
not a replacement production loader/type checker.

CPU is the worker process's user+system time, from `ProcessState`; **Go loader
subprocess CPU is excluded**. Windows memory is the worker's peak working-set
counter sampled every 5 ms with a final read; it excludes Go child processes.
On Unix the harness uses `wait4` maxrss with platform unit normalization, but
that path has not been executed here. Whole-process-tree CPU/memory measurement
is **NOT RUN**; no aggregate loading resource claim is made. Wall loading time
does include waiting for the Go commands. CPU resolution and timer noise matter
for these small samples.

Within this experiment loading/type checking dominates the measured analyze
stage. It compares graph sizes under one harness and policy, not this tool
against another implementation. Imported standard-library ASTs can dominate real
modules because fact-aware `singlechecker` loads all syntax; these synthetic
packages import no standard library.

The [durableq run](real-project.md) is separate functional evidence, not a
comparable performance sample. No real external-user benchmarks are claimed.
