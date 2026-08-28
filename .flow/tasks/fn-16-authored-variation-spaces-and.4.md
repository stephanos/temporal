---
satisfies: [R1, R2, R3, R4, R5, R6, R8]
---
# fn-16-authored-variation-spaces-and.4 Implement point lowering and atomic batch compilation

## Description
Implement the reusable exact-assignment lowering seam and complete Cartesian batch compiler for R1-R6/R8. Every point must pass existing semantic checkers and the existing proof-carrying planner.

**Size:** M
**Files:** `model/Umpire/Space/Compiler.lean`, `model/Umpire/Space/Tests/Compilation.lean`, `model/Umpire/Space/Tests/Determinism.lean`
**Touches:** [model/Umpire/Space/Compiler.lean, model/Umpire/Space/Tests/Compilation.lean, model/Umpire/Space/Tests/Determinism.lean]

### Approach
- Enumerate complete assignments lexicographically after checked multiplication and derive stable point/Behavior/Query IDs from the canonical assignment digest.
- Apply role restrictions through existing Behavior checking, recheck the derived Query against the same target, and return a dependent `LoweredSpacePoint` containing the checked Query, checked artifact intent, and proof that its target equals the base Query target.
- Make `compileBatch` accept `IncrementalPlannerKernel space.baseQuery.target`; transport that exact kernel across each point's target-equality proof before ordinary planning, and apply intent only to a selected artifact. Never rebuild or duplicate a kernel.
- Reject the whole batch on the first canonical invalid/unsatisfiable/duplicate/budget-exhausted/verified-without-artifact point; never expose accumulated partial specs.
- Detect duplicate derived point or final ExperimentSpec identities.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Behavior/Language.lean:816-900` — Behavior rechecking
- `model/Umpire/Query/Language.lean:362-406` — Query rechecking and semantic identity
- `model/Umpire/Planning/Engine.lean:432-470` — target-owned planning facade
- `model/Umpire/Artifact.lean:311-382` — canonical artifact construction
- `model/Umpire/Planning/Tests/Fixtures.lean:25-233` — proof-carrying synthetic kernel pattern
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean:445-449` — existing target-equality kernel transport helper

### Acceptance
- [ ] Exact assignment errors identify missing/extra/unknown choices and the canonical point.
- [ ] Every derived Behavior/Query is newly checked and identity-distinct, and carries a target-equality proof usable to transport the single caller-supplied base kernel.
- [ ] A valid bounded batch returns every point in canonical order; any failed/non-artifact point returns no list.
- [ ] Planner target actions, outcomes, states, observations, and selection reasons are never caller-authored or patched.
- [ ] A focused dependent-type test compiles a derived point by transporting the base kernel; no alternate kernel constructor exists in Space.
- [ ] Source/axis/choice ordering permutations yield identical point and artifact bytes.

## Acceptance
- [ ] Point lowering uses existing semantic checkers, returns target equality, and does not plan.
- [ ] Batch compilation is bounded, deterministic, all-or-nothing, and target-owned.
- [ ] Every error carries the first canonical point identity with no partial output.

## Done summary
Implemented exact canonical Space point lowering and atomic complete-batch compilation through the existing Behavior, Query, Artifact-intent, and proof-carrying planning seams. The compiler transports one caller-owned kernel, preserves target-owned artifact fields, emits deterministic canonical errors/bytes, rejects every non-artifact point without partial output, and prevents derived identity collisions.

Baseline was green for existing Validation, Metadata, Switch, aggregate, and regression targets; Compilation was the expected pre-feature target owned by this task and now passes. Final verification passes Validation, Compilation, Metadata, Switch, the combined focused suites (including Intent and Determinism), the 118-job aggregate, and the 152-job regression gate. `Temporal.Feature.Nexus.Examples.VariationSpaceTests` remains the explicitly expected task .5 absence; no scope-violating stub was added. Gate receipts were non-blockingly unavailable because the preserved unrelated `.plans/UMPIRE4_ORDER.md` diff keeps the worktree dirty. Review fixed universal derived-identity freshness and returned SHIP; optional memory capture was non-blockingly unavailable because memory is not initialized.

stage: impl-review - ran [2026-08-28T01:56:09Z..2026-08-28T02:07:04Z]
## Evidence
- Commits: 5f1f4730ebe793975db6c925112da5f2261f0c10, 7a13ebd9a3359d782b93d1889fed4b46afb21e48
- Tests: cd model && mise exec -- lake build Umpire.Space.Tests.Validation, cd model && mise exec -- lake build Umpire.Space.Tests.Compilation, cd model && mise exec -- lake build Umpire.Space.Tests.Metadata, cd model && mise exec -- lake build Umpire.Examples.SwitchTests, cd model && mise exec -- lake build Umpire.Space.Tests.Validation Umpire.Space.Tests.Intent Umpire.Space.Tests.Metadata Umpire.Space.Tests.Compilation Umpire.Space.Tests.Determinism Umpire.Examples.SwitchTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, EXPECTED_FUTURE_TARGET: cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.VariationSpaceTests - absent for task .5
- PRs: