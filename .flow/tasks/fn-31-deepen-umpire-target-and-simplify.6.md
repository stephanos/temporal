---
satisfies: [R5, R6]
---
# fn-31-deepen-umpire-target-and-simplify.6 Lock Target compatibility with migration fixtures

## Description

Close the migration boundary with an executable compatibility matrix after the public Target, Query, and Planning adapters land.

**Size:** M
**Files:** `model/Umpire/Target/Tests/MigrationCompatibility.lean`, `model/Umpire/Examples/SwitchTests.lean`, `model/Temporal/Feature/Nexus/Examples/BasicLifecycleTests.lean`, `model/Temporal/Feature/Nexus/Examples/BasicOperationsTests.lean`, `model/Temporal/Feature/Nexus/CallerClosureTests.lean`, `model/UmpireTests.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Umpire/Target/Tests/MigrationCompatibility.lean, model/Umpire/Examples/SwitchTests.lean, model/Temporal/Feature/Nexus/Examples/BasicLifecycleTests.lean, model/Temporal/Feature/Nexus/Examples/BasicOperationsTests.lean, model/Temporal/Feature/Nexus/CallerClosureTests.lean, model/UmpireTests.lean, model/TemporalModelTests.lean]

### Approach

- Record before/after checked target identity, canonical target/query/artifact bytes, Query checking, and planner outcomes for Switch.
- Cover exactly `BasicLifecycle`, `BasicOperations.AsyncStart`, `BasicOperations.SuccessfulCompletion`, and `CallerClosure`; distinguish their target authors from downstream query consumers.
- Pin representative valid/invalid diagnostics at exact authoring file/line/column while separately proving stable `SemanticSource` provenance and canonical bytes do not change.
- Run the matrix through the ordinary Umpire and Temporal aggregates before downstream authoring-language work.

### Investigation targets

**Required** (read before coding):
- `model/Umpire/Examples/Switch.lean` and `SwitchTests.lean`
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean`
- `model/Temporal/Feature/Nexus/Examples/BasicOperations.lean`
- `model/Temporal/Feature/Nexus/CallerClosure.lean`
- `model/Umpire/Query/Tests/**` and `model/Umpire/Planning/Tests/**`

## Acceptance

- [ ] Switch plus the four named Temporal target/query families preserve checked meaning, target/query semantic identities, canonical artifact bytes, and planner results.
- [ ] Invalid identity, provider, connector, law, kernel-availability, Query-bound, finite-completeness, and target-kernel mismatch cases remain at their assigned typed boundary.
- [ ] Exact diagnostic file/line/column changes with the authoring site while stable provenance, semantic digests, and artifact bytes remain unchanged.
- [ ] The executable inventory fails if any named family is omitted and runs through `UmpireTests` and `TemporalModelTests` without downstream Space, Exploration, runtime, Verify, Veil, or Umpire3 imports.
- [ ] Existing comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

