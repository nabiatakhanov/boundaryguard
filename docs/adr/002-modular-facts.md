# ADR 002: bounded modular witnesses with analysis package facts

Status: accepted.

## Context

A direct import checker cannot see domain → shared → storage. Searching every
possible path would duplicate work and produce exponentially many alternatives.
The official analysis driver already orders package analysis by imports and
transports package facts. External packages and the standard library do not
provide BoundaryGuard facts under every compatible driver.

## Decision

Each in-module package exports one shortest, lexicographically stable import
path per reachable target layer. Importers combine direct endpoints with those
facts and prepend themselves to candidate paths. Unclassified packages propagate
facts too. External packages can be classified endpoints, but their hidden
dependencies are outside the traversal guarantee. Driver module metadata marks
independent nested modules as boundaries when they share a path prefix.

The serialized contract includes schema version 1, a fingerprint of the complete
normalized policy, an in-module marker and sorted witness slices. Copy slices
when composing candidates; never mutate after export. Missing expected internal
facts or incompatible/malformed facts fail analysis. A policy change cannot
silently turn stale data into a clean result.

Report one BG003 witness per rule at the first import step. Prefer the shortest
path, then lexicographic package sequence. If BG001 already explains that rule
through a direct import, do not also report BG003.

## Consequences

The approach fits `go/analysis` and avoids a second module loader or global graph.
It handles permitted first edges and unclassified intermediate packages. A
simple BFS oracle can independently test path selection on small random DAGs.

Facts retain a bounded number of alternatives per package, although full path
length and aggregate serialization still grow with module depth and layer
count. This is not a constant-memory algorithm. It cannot establish transitive
closure inside arbitrary external modules, reflection, runtime plugins or
source files excluded by build constraints. Those limits are explicit rather
than hidden behind missing facts. Full-program analysis and exhaustive path
enumeration are unnecessary for version 1's contract.
