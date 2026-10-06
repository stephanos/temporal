---
satisfies: [R5, R6, R7]
---
# fn-136-phase-roles-on-lifecycle-enums.2 Activity models adopt roles; refinement closedness check

## Description
The activity product, activity system and activity record declare roles and their `states` predicate bodies become role tests, with identical IR. Adds the framework closedness check and runs it on the activity's refinements.

**Size:** M
**Files:** `model/temporal/features/activity/standalone/product/Product.scala`, `model/temporal/features/activity/standalone/system/System.scala`, `model/temporal/features/activity/standalone/system/Record.scala`, `model/umpire/Refine.scala` (closedness check), a new munit test beside the activity models (e.g. `model/temporal/features/activity/standalone/RoleRefinements.test.scala`), `model/ir/**` (positions only)
**Touches:** [model/temporal/features/activity/**, model/umpire/Refine.scala, model/ir/**, model/cases/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Assign roles per spec (pending requests are `Held`; `unstarted` has none). Product: `scheduled` Waiting, `started`/`cancelRequested` Held, `paused` Suspended, closures. System: also `backingOff` Retrying, `pauseRequested` Held. Record `AdmissionPhase`: `scheduled` Waiting, `paused` Suspended, `started` Held, `completed` Succeeded, `timedOut` TimedOut, `pausedWhileHeld` no role.
- Keep every `states` predicate's name, signature and callers. Replace a body only when it is a single `in(...)` listing exactly a role's cases in declaration order (e.g. `terminal`, `live`, `held`, `waiting`); only that lifts to identical IR. Keep: `end`/`over` and every other body that calls named predicates, product `pausable`, product `paused` (`s.phase == paused` lifts as OP_EQ even though its set equals `Suspended`), `running` = `started` alone, record `paused`/`running`/`stopped`.
- `end` bodies do not change: they already call the named predicate (e.g. `ActivityProduct.states.over`), which now reads the role. `ActivityRecord.end` stays `stopped`.
- Closedness check: a framework function that, given a refinement, the system's `Finite` states, both phase projections and `toProduct`, returns the states whose `Closed` role differs from their image's. A munit test calls it for `ActivitySystem.refinement`, `ActivityRecord.refinement` and `HeldDispatch.refinement`, by name, and requires an empty result.
- Behavior pin (refactor): snapshot `model/ir` and `model/cases` under `.flow/tmp/fn-136.2/` before editing, regenerate with `make MODEL_GATE_ARGS=--skip-go-checks umpire-gen-model`, and compare IR field-for-field with a strict script that has a positive and a negative control (follow the fn-132.1 identity proof in `.flow/tasks/fn-132-group-the-nexus-and-activity-models-by.1.md`); Cases byte-for-byte.

### Investigation targets
**Required** (read before coding):
- `model/temporal/features/activity/standalone/product/Product.scala:15-58,93-130` - phases, states, rules, capability fields
- `model/temporal/features/activity/standalone/system/System.scala:21-23,62,71-108,215-232` - phases, end, states, refinement, deadline rules
- `model/temporal/features/activity/standalone/system/Record.scala:32-34,98-146,377` - admission phases, end, states, refinements
- `model/umpire/Refine.scala` - Refinement and toProduct
- `.flow/tasks/fn-132-group-the-nexus-and-activity-models-by.1.md` - identity proof procedure

**Optional** (reference as needed):
- `model/temporal/features/activity/standalone/system/WithTaskQueue.scala:59,94-110,162,180-196` - compositions reading record predicates through `through(...)`

### Key context
- fn-135 (a gate of this spec) renames the product's `states.phase` projection to `states.status` (the `phase` field is unchanged) and turns predicates into `is { }` blocks; work against the tree as it is after fn-135 closes. A role test inside an `is { }` block must lift like one in a plain def; if it does not, fix the lifter here.
- If fn-128.1 has landed first, `backingOff` is gone from the system; assign roles to whatever phases exist.

## Acceptance
- [ ] Product, system and record phase enums carry roles as specified; `pausedWhileHeld` carries none.
- [ ] Every replaced `states` predicate keeps its name and callers and its body is a role test; only single-`in(...)` bodies equal to a role's cases were replaced; `end`, product `paused` and other kept bodies are unchanged.
- [ ] The closedness test names `ActivitySystem.refinement`, `ActivityRecord.refinement` and `HeldDispatch.refinement`.
- [ ] The closedness check exists in the framework and runs with no violations.
- [ ] Identity proof: IR identical field-for-field except source positions (strict script with positive and negative controls), `model/cases` byte-identical, every Query answer and Definition ID unchanged.
- [ ] `make MODEL_GATE_ARGS=--skip-go-checks umpire-check-model` and `make lint-model` pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
