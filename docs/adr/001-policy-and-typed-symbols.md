# ADR 001: strict immutable policies and typed symbol references

Status: accepted.

## Context

An architecture checker is useful only when policy errors fail visibly and symbol
matching does not confuse names that happen to look alike. Go allows renamed and
dot imports, local shadowing, package-level function values and similarly named
struct fields. Matching selector text or only `CallExpr` would miss valid cases
and produce false positives.

## Decision

Use JSON version 1 with exact import paths or recursive `/...` suffixes, explicit
dependency directions and scoped full symbol names. Reject unknown and duplicate
JSON keys, ambiguous layer patterns and missing rule references before loading
the target graph. Normalize the policy, compute its complete SHA-256 fingerprint
and expose only copied collections. Construct a separate analyzer closure for
each immutable policy.

For BG002, visit `TypesInfo.Uses` once, ordered by source position. Match the
defining package path and name of package-scope functions without receivers and
package-scope variables. Do not treat methods, local names, fields, constants or
types as those symbols. Report each matching identifier, whether it is called,
assigned or read.

## Consequences

Alias and dot imports work without special name-resolution code. Local `http`,
local `Get`, another package's `Get` and fields do not accidentally match
`net/http.Get`. Saving a function to a variable is still a visible reference.
Immutability makes parallel analyzer instances independent and allows race tests
without synchronization around policy state.

Configuration can mention a symbol absent from the target source tree: syntax is
validated before analysis, while actual object identity is checked only when a
reference exists. The checker deliberately does not infer calls through runtime
reflection or arbitrary wrappers. There is no custom symbol resolver or
whole-program SSA analysis. Strict spelling can reject a mistyped policy early;
extending the schema requires an explicit versioned design.
