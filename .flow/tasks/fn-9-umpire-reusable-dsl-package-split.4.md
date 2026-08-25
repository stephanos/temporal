---
satisfies: [R1, R4, R5]
---
# fn-9-umpire-reusable-dsl-package-split.4 Move Artifact and seal the Planning engine

## Description
Move portable artifacts and the pure planner downstream of the checked Query interface (R1, R4, R5). Preserve planner termination authority and artifact identity/provenance behavior.

**Size:** M
**Files:** `model/Umpire.lean`, `model/UmpireTests.lean`, `model/Umpire/Artifact.lean`, `model/Umpire/Planning.lean`, `model/Umpire/PlanningTests.lean`, `model/Umpire/PlanningVisibilityTests.lean`
**Touches:** [model/Umpire.lean, model/UmpireTests.lean, model/Umpire/Artifact.lean, model/Umpire/Planning.lean, model/Umpire/PlanningTests.lean, model/Umpire/PlanningVisibilityTests.lean]

### Approach
- Move `DrivePlan`, `ExperimentSpec`, portable property/provenance data, canonical artifact serialization, and selection compilation to Artifact.
- Move planner pull/backend protocols, incremental kernel/state, enumeration, private termination/finalization, outcomes, results, and `plan` to Planning.
- Preserve private result construction and add a negative compile module that imports `Umpire.Planning` itself and proves both `Umpire.finalizePlanning` and `Umpire.PlanningResult.mk` are inaccessible.
- Port planner tests without collapsing unsatisfiable, budget-exhausted, found, or verified states.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Experiment/Artifact.lean:1-241` — portable artifacts and canonical serialization
- `model/Temporal/Experiment/Artifact.lean:299-308` — provenance source aggregation
- `model/Temporal/Experiment/Planner.lean:16-115` — kernel, state, private result construction, and outcomes
- `model/Temporal/Experiment/Planner.lean:408-457` — private finalization and public planning entry point
- `model/Temporal/Experiment/PlannerTests.lean:240-360` — outcome and deterministic planning coverage

**Optional** (reference as needed):
- `model/Temporal/Experiment/Query.lean:478-485` — planner pull/backend protocol to relocate

### Key context
Semantic identity excludes provenance while canonical artifact JSON records it. Preserve that distinction; do not introduce an artifact migration mechanism.
## Acceptance
- [ ] Artifact and Planning depend only on approved upstream Umpire modules.
- [ ] DrivePlan/ExperimentSpec identities, fields, omissions, ordering, and serialization remain stable.
- [ ] A negative compile test importing `Umpire.Planning` confirms both the finalizer and `PlanningResult.mk` are inaccessible to external callers.
- [ ] Unsatisfiable, exhausted, found, verified, and invalid paths remain distinct and covered.
- [ ] No runtime execution or observation/evidence type enters either module.
- [ ] `make umpire-check-regression` remains green.
## Done summary
Moved canonical portable artifacts and the deterministic planning engine downstream of Umpire.Query without semantic redesign, preserving artifact identity/provenance behavior and every existing comment. Kept planning outcomes distinct, sealed finalization/result construction, and added narrow-import negative compile coverage for both private authorities.

baseline: green via receipt
GATE_SKIPPED:smoke:green-receipt 3dd8d585 - baseline reused from prior post-gate pass
stage: impl-review - ran [2026-08-25T19:43:42Z..2026-08-25T19:46:57Z] (SHIP)
## Evidence
- Commits: 5f57472352fb21311c6eb9824df94d9bf8515bc8
- Tests: GATE_SKIPPED:smoke:green-receipt 3dd8d585 - baseline reused from prior post-gate pass, mise exec -- lake build UmpireTests, make umpire-check-regression
- PRs: