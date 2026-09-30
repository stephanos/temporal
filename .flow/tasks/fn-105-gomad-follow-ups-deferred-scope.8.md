---
satisfies: [R8]
---
# fn-105-gomad-follow-ups-deferred-scope.8 D8: closure-mode support for downstream targets

## Description
Origin: fn-104 C3/R2. Deferred 2026-09-29 because F9 qualifies in linked mode. Revived 2026-09-30 by fn-107 R7/R9: the downstream workflow manifest requires closure-mode preparation. Consume the final fn-107.4 target after its native workflow smoke passes; do not duplicate this implementation in fn-107. Add exact signal-handling metrics adapters only for libraries still reachable in the final closure, with per-file digests, inventories, per-platform pins and upstream-edit/version negative tests. Elimination is acceptable with fresh closure evidence. Files/Touches: tools/gomad3 adapter registry, exact adapter overlays and focused tests. Quick: focused adapter/prepare tests, generator checks and final-target closure analysis.
## Acceptance
Revival is recorded. Closure-mode analysis of the final downstream Walker-backed workflow target reports supported on both qualified platforms. Every reachable metrics adapter is exact, excludes signal registration, has identity-drift negative evidence and passes its native focused tests; otherwise prove the library was eliminated. fn-107.5 consumes the reports.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
