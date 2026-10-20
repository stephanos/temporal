---
satisfies: [R1, R2, R3, R4, R7, R8]
---
# fn-231-gomad-site-aware-minimizer-ordering-and.1 Scoped proposal ordering with checkpoint key and parent-scope default

Touches: tools/gomad/internal/runner/internal/minimizer/{minimizer.go,choice.go,workspace.go}, runner/minimize_operation.go, cli/minimize.go

## Description
Accept `--site-scope` on minimize with the explore validator; default to the parent artifact's recorded scope when absent. Build the fingerprint→symbol map from the parent's retained module once and resolve the scope. In `proposals`, apply a stable sort key: all-out-of-scope first, mixed, then in-scope only, preserving existing order within a class; the choice-prefix minimizer is unchanged. Add the ordering key (patterns, resolved digest) to persisted state and the implementation digest so old state and changed keys are refused as today. Add a test with a fixed reduction set showing identical final candidate under scoped and unscoped ordering and different attempt counts.

## Acceptance
Invalid scope exits 2 before any attempt; parent scope is used by default; the ordering test passes; resume with a changed key is refused with status 3; without any scope the attempt sequence is byte-identical to before.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
