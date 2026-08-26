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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
