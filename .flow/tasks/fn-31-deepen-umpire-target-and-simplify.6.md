---
satisfies: [R5, R6]
---
# fn-31-deepen-umpire-target-and-simplify.6 Lock Target compatibility with migration fixtures

## Description

Close the migration boundary with an executable compatibility matrix after the public Target, Query, and Planning adapters land.

**Size:** M
**Files:** `model/Umpire/Tests/MigrationCompatibility.lean`, `model/Umpire/Examples/SwitchTests.lean`, `model/Temporal/Feature/Nexus/LifecycleTests.lean`, `model/Temporal/Feature/Nexus/OperationsTests.lean`, `model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean`, `model/UmpireTests.lean`, `model/TemporalModelTests.lean`, `model/TemporalExperimentalTests.lean`
**Touches:** [model/Umpire/Tests/MigrationCompatibility.lean, model/Umpire/Examples/SwitchTests.lean, model/Temporal/Feature/Nexus/LifecycleTests.lean, model/Temporal/Feature/Nexus/OperationsTests.lean, model/Temporal/Feature/Nexus/Experimental/CallerClosureTests.lean, model/UmpireTests.lean, model/TemporalModelTests.lean, model/TemporalExperimentalTests.lean]

### Approach

- Record before/after checked target identity, stable-`SemanticSource` canonical target/query/artifact bytes, exact role/action-domain token strings, Query checking, and planner outcomes for Switch in the downstream `Umpire.Tests.MigrationCompatibility`/`Umpire.Examples.SwitchTests` boundary, never under `Umpire.Target.*`.
- Cover the `Lifecycle` and `Experimental.CallerClosure` target authors plus the
  `Operations.AsyncStart`, `Operations.Cancellation`, and `Operations.SuccessfulCompletion`
  consumers of `Lifecycle.target`; include every additional live ordinary Nexus target consumer
  present when this task starts, including Observation coverage if fn-4 has landed.
- Pin representative valid/invalid diagnostics at exact authoring file/line/column while separately proving that reordered or relocated nonsemantic occurrence spans cannot affect stable `SemanticSource`, semantic digests, canonical metadata, or artifact bytes.
- Run the matrix through the ordinary Umpire and Temporal aggregates before downstream authoring-language work.

### Investigation targets

**Required** (read before coding):
- `model/Umpire/Examples/Switch.lean` and `SwitchTests.lean`
- `model/Umpire/Tests/MigrationCompatibility.lean` — downstream cross-layer matrix that may import Query, Planning, and Artifact
- `model/Temporal/Feature/Nexus/Lifecycle.lean`
- `model/Temporal/Feature/Nexus/Operations.lean`
- `model/Temporal/Feature/Nexus/Experimental/CallerClosure.lean`
- `model/Umpire/Query/Tests/**` and `model/Umpire/Planning/Tests/**`

## Acceptance

- [ ] Switch plus the two named Temporal target authors and three named Operations query consumers preserve checked meaning, target/query semantic identities, exact role/action-domain token strings, canonical Query/artifact bytes, and planner results; every additional live ordinary Nexus consumer is covered without creating a hard dependency on unfinished work.
- [ ] Invalid identity, provider, connector, law, kernel-availability, Query-bound, finite-completeness, and target-kernel mismatch cases remain at their assigned typed boundary.
- [ ] Exact diagnostic file/line/column changes with the authored occurrence while the separate stable `SemanticSource`, semantic digests, canonical metadata, and artifact bytes remain unchanged.
- [ ] The executable inventory fails if any named family is omitted and runs through `UmpireTests`, `TemporalModelTests`, and `TemporalExperimentalTests` without downstream Space, Exploration, runtime, Verify, Veil, or Umpire3 imports.
- [ ] Cross-layer Query/Planning/artifact fixtures live outside `Umpire.Target.*`; Target tests remain import-pure and require no lint exemption.
- [ ] Existing comments are preserved.

## Done summary
Implemented the downstream Umpire Target compatibility matrix for Switch, both live Nexus target authors, and all three live ordinary Nexus target consumers. The matrix pins stable semantic identity, exact tokens, canonical Query/artifact bytes, planner results, typed error ownership, relocation invariants, and aggregate inventory/import purity; no ordinary Nexus Observation target consumer exists in the current tree.

Committed operation artifact goldens and an independent relocated Planning/Artifact proof address the implementation review findings.

baseline: green (the first targeted Lake build hit an inherited missing-output transient; its exact retry and every other Quick command passed)
stage: impl-review - ran [2026-08-27T05:23:39Z..2026-08-27T05:36:31Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1053d7cd47b75194815c77f34a46c420d18cdeec, bf4ffaa6593c726ca313eec02be4476bb992f2b0, e2911be441f42915ef0c1db56f06c9b713f4dff0, 65648879bc813ca1e69f16cbc34402e5900690b0
- Tests: cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests, cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.Feature.Nexus.OperationsTests Temporal.Feature.Nexus.Experimental.CallerClosureTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, make lint-model, cd model && mise exec -- lake build TemporalExperimentalTests
- PRs:
