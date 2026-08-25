---
satisfies: [R1, R4, R5]
---
# fn-9-umpire-reusable-dsl-package-split.3 Move the Query DSL onto Umpire modules

## Description
Move Query as the first combining layer over checked Property, Behavior, and Search products (R1, R4, R5). Keep planner mechanics out of the Query interface.

**Size:** M
**Files:** `model/Umpire.lean`, `model/UmpireTests.lean`, `model/Umpire/Query.lean`, `model/Umpire/QueryTests.lean`
**Touches:** [model/Umpire.lean, model/UmpireTests.lean, model/Umpire/Query.lean, model/Umpire/QueryTests.lean]

### Approach
- Move query forms, declarations, checked targets, completeness evidence, validation, canonicalization, and planning input contracts to `Umpire.Query`.
- Consume Search types rather than retaining duplicate definitions.
- Leave planner pull/backend mechanics for the Planning task.
- Port Query tests, including exact behavior/query canonical contracts and the compile guard that Query alone exposes no public planning finalizer.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Experiment/Query.lean:1-218` — query authoring and checking contracts
- `model/Temporal/Experiment/Query.lean:397-485` — canonical output, validation, metadata, and planner boundary
- `model/Temporal/Experiment/QueryTests.lean:7-12` — external finalizer absence guard
- `model/Temporal/Experiment/QueryTests.lean:256-347` — deterministic query validation/canonical tests

**Optional** (reference as needed):
- `model/Temporal/Experiment/Behavior.lean:535-624` — behavior validation/admission seam consumed by Query

### Key context
Query must combine only checked siblings; it must not regain planner result construction or runtime/evidence concerns.

## Acceptance
- [ ] `Umpire.Query` imports Property, Behavior, and Search and is the first module combining them.
- [ ] Query has no duplicate Search definitions and no planner result/finalization authority.
- [ ] Completeness, invalid-bound, unsupported-form, canonical-ordering, and exact-behavior tests retain their outcomes.
- [ ] The external guard confirms no public `Umpire.finalizePlanning` is exposed by Query.
- [ ] `make umpire-check-regression` remains green.

## Done summary
Moved Query authoring, completeness, validation, and canonicalization onto `Umpire.Query` as the first layer combining checked Property, Behavior, and Search products, without copying Search definitions or exposing planner mechanics/finalization. Ported the Query suite to `Umpire.QueryTests`, retaining completeness and exact-trace outcomes and adding focused invalid-bound, unsupported-strategy, canonical-ordering, and `Umpire.finalizePlanning` absence guards.

baseline: green via receipt
GATE_SKIPPED:smoke:green-receipt 774f1c3d - baseline reused from prior post-gate pass
stage: impl-review - ran (SHIP; completed 2026-08-25T19:35:49.168715Z)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3dd8d585b0e4c8d3b20d40b654a6ac80c20399f8
- Tests: GATE_SKIPPED:smoke:green-receipt 774f1c3d - baseline reused from prior post-gate pass, mise exec -- lake build UmpireTests, make umpire-check-regression
- PRs:
