---
satisfies: [R12]
---
# fn-160-one-ir-file-per-feature-collected-from.6 Rename the System level to Protocol

## Description
Rename the middle level from System to Protocol everywhere in the Scala model, so the three levels read Product / Protocol / Realization and sort alphabetically in their conceptual order (owner, 2026-10-10).

**Size:** M
**Files:** every `system/` level folder (`features/activity/standalone/system`, `features/nexus/{standalone,workflow}/system`, `foundations/taskqueue/system`), the level machines and their files (`ActivitySystem` → `ActivityProtocol`, `NexusSystem` → `NexusProtocol`, task-queue System → Protocol, `System.scala` → `Protocol.scala`), references across model/temporal, the structure lint's level vocabulary in model/irgen and model/check, fixtures, `model/README.md`, `model/framework/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md` and docs that name the level.
**Touches:** [model/temporal/**, model/irgen/**, model/check/**, model/README.md, model/framework/SEMANTICS.md, .plans/UMPIRE_MODULES.md]

### Approach
- Rename folders, packages, files and level machines in one pass with the fn-160.3/.4 moves, so files move once.
- Keep the `system` actor/timer names that are not the level (e.g. `system.backoff` steps) unless fn-123's history/matching actor split renames them; only the level is renamed here.
- Update the structure lint's level names (product/system → product/protocol) and its fixtures.
- Record every changed machine-scoped id (machines, targets, states, actions, facts, outcomes, Case names) in the Batch 1d rename map.

### Batching
- Batch 1d (Scala DSL, immediate; see MILESTONES.md): land the Scala changes now; IR and Cases stay stale until Batch 5b. Verify with the Scala build, framework/model tests, lifter fixtures and `make umpire-check-equivalence ... STEPS_ONLY=1` under the rename map.

## Acceptance
- [ ] No level folder, package, machine or file is named System/system for the level; the levels are Product / Protocol / Realization in code, lint vocabulary and docs.
- [ ] Step tables are equal to the baseline under the batch rename map (STEPS_ONLY=1); Scala build, framework/model tests and lifter fixtures pass.

## Done summary
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
