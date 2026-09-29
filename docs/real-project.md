# Read-only check of durableq

The local sibling repository `../task-queue` contains the real Go module
`durableq`. Its source inventory includes `internal/queue`, `internal/store`,
`internal/httpapi`, `internal/worker`, `internal/handlers`, `internal/observe`,
`cmd/api` and `cmd/worker`. No copy of those sources is embedded in BoundaryGuard.
The repository and its dependency cache must already be present to repeat this
check; it is not a publicly claimed deployment or an external user of this tool.

From the BoundaryGuard checkout, the portable evidence runner performs the
before/after snapshots and saves its command and raw output automatically:

```text
go build -o bin/boundaryguard.exe ./cmd/boundaryguard
python scripts/realcheck.py --repo ../task-queue --binary bin/boundaryguard.exe --config testdata/durableq.json --out artifacts/durableq.json
```

Use `bin/boundaryguard` on Unix. The runner writes only its `--out` file outside
the target repository and refuses an output path inside that repository. It
records source hashes, Git status, executable/config hashes, policy, timestamps,
tool versions and the analyzer's actual exit code. Its own exit 0 means the
experiment completed with unchanged sources/status and analyzer exit 0 or 3;
consult `analysis.exit_code` to distinguish a clean policy result from expected
diagnostics. Loading errors, unavailable dependencies, failed snapshots/status or
changed target files fail the runner. Use the analyzer directly for a CI gate.

[`testdata/durableq.json`](../testdata/durableq.json) supplies a small exploratory
policy based on that inventory. Queue types must not depend on the storage layer,
even through another in-module package. The HTTP API must not directly import
the concrete store. Internal packages must not reference `net/http.Get`,
`net/http.DefaultClient` or `log.Fatal`. Command entry points are outside the
symbol scopes. These are candidate architecture rules for this experiment,
not a claim that durableq's author has adopted this policy. Packages such as
`internal/handlers` remain unclassified and still propagate in-module facts.

Build the executable from BoundaryGuard, then run it in the durableq directory
with an absolute policy path. The following PowerShell commands do not install
dependencies or modify the target's module files:

```powershell
# From the BoundaryGuard checkout:
go build -o bin/boundaryguard.exe ./cmd/boundaryguard
$boundaryguardRoot = (Get-Location).Path
$targetRepository = (Resolve-Path -LiteralPath '../task-queue').Path
$binary = Join-Path $boundaryguardRoot 'bin/boundaryguard.exe'
$policy = Join-Path $boundaryguardRoot 'testdata/durableq.json'
$env:GOWORK = 'off'
$env:GOPROXY = 'off'
$env:GOFLAGS = '-mod=readonly'

# Capture status and hashes before and after; use a separate terminal or restore
# these process environment variables afterwards if needed for other work.
git --no-optional-locks -C $targetRepository status --porcelain=v1
Push-Location -LiteralPath $targetRepository
try {
    & $binary -config $policy -test=false ./...
    $analysisExitCode = $LASTEXITCODE
} finally {
    Pop-Location
}
git --no-optional-locks -C $targetRepository status --porcelain=v1
"BoundaryGuard exit code: $analysisExitCode"
```

On Unix, build `bin/boundaryguard` and use
`GOWORK=off GOPROXY=off GOFLAGS=-mod=readonly /absolute/path/bin/boundaryguard -config /absolute/path/testdata/durableq.json -test=false ./...`
from the durableq module directory.

`-test=false` selects production package files; this experiment does not compile
or execute durableq's database/process integration tests. Build constraints and
the active platform still control source selection. Go package loading/type
checking needs cached dependencies; an offline missing-dependency failure is
reported as a failed run, not a clean architecture result. The Go build cache may
be populated outside the target repository.

For preservation evidence, hash every `.go` file plus `go.mod` and `go.sum` in
sorted relative-path order before and after, and compare both the aggregate hash
and `git --no-optional-locks status --porcelain=v1`. Record the command, Go
version, selected policy, exit code and raw stdout/stderr in BoundaryGuard's
`artifacts/` directory. An equal before/after tree hash and Git status establish
that this run left the selected target sources and module metadata unchanged.
Git status alone does not verify untracked source contents.

A clean result supports only this explicit policy and selected source set. No
claim of discovered product bugs, exhaustive architecture correctness, security
proof, commercial adoption or runtime verification follows from it.
