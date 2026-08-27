---
satisfies: [R5, R6]
---
# fn-31-deepen-umpire-target-and-simplify.6 Lock Target compatibility with migration fixtures

## Description

Close the migration boundary with an executable compatibility matrix after the public Target, Query, and Planning adapters land.

**Size:** M
**Files:** `model/Umpire/Tests/MigrationCompatibility.lean`, `model/Umpire/Examples/SwitchTests.lean`, `model/Temporal/Feature/Nexus/Examples/BasicLifecycleTests.lean`, `model/Temporal/Feature/Nexus/Examples/BasicOperationsTests.lean`, `model/Temporal/Feature/Nexus/CallerClosureTests.lean`, `model/UmpireTests.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Umpire/Tests/MigrationCompatibility.lean, model/Umpire/Examples/SwitchTests.lean, model/Temporal/Feature/Nexus/Examples/BasicLifecycleTests.lean, model/Temporal/Feature/Nexus/Examples/BasicOperationsTests.lean, model/Temporal/Feature/Nexus/CallerClosureTests.lean, model/UmpireTests.lean, model/TemporalModelTests.lean]

### Approach

- Record before/after checked target identity, stable-`SemanticSource` canonical target/query/artifact bytes, exact role/action-domain token strings, Query checking, and planner outcomes for Switch in the downstream `Umpire.Tests.MigrationCompatibility`/`Umpire.Examples.SwitchTests` boundary, never under `Umpire.Target.*`.
- Cover the `BasicLifecycle` and `CallerClosure` target authors plus the `BasicOperations.AsyncStart` and `BasicOperations.SuccessfulCompletion` consumers of `BasicLifecycle.target`; include every additional live ordinary Nexus target consumer present when this task starts, including Observation coverage if fn-4 has landed.
- Pin representative valid/invalid diagnostics at exact authoring file/line/column while separately proving that reordered or relocated nonsemantic occurrence spans cannot affect stable `SemanticSource`, semantic digests, canonical metadata, or artifact bytes.
- Run the matrix through the ordinary Umpire and Temporal aggregates before downstream authoring-language work.

### Investigation targets

**Required** (read before coding):
- `model/Umpire/Examples/Switch.lean` and `SwitchTests.lean`
- `model/Umpire/Tests/MigrationCompatibility.lean` — downstream cross-layer matrix that may import Query, Planning, and Artifact
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean`
- `model/Temporal/Feature/Nexus/Examples/BasicOperations.lean`
- `model/Temporal/Feature/Nexus/CallerClosure.lean`
- `model/Umpire/Query/Tests/**` and `model/Umpire/Planning/Tests/**`

## Acceptance

- [ ] Switch plus the two named Temporal target authors and two named BasicOperations query consumers preserve checked meaning, target/query semantic identities, exact role/action-domain token strings, canonical Query/artifact bytes, and planner results; every additional live ordinary Nexus consumer is covered without creating a hard dependency on unfinished work.
- [ ] Invalid identity, provider, connector, law, kernel-availability, Query-bound, finite-completeness, and target-kernel mismatch cases remain at their assigned typed boundary.
- [ ] Exact diagnostic file/line/column changes with the authored occurrence while the separate stable `SemanticSource`, semantic digests, canonical metadata, and artifact bytes remain unchanged.
- [ ] The executable inventory fails if any named family is omitted and runs through `UmpireTests` and `TemporalModelTests` without downstream Space, Exploration, runtime, Verify, Veil, or Umpire3 imports.
- [ ] Cross-layer Query/Planning/artifact fixtures live outside `Umpire.Target.*`; Target tests remain import-pure and require no lint exemption.
- [ ] Existing comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

