---
satisfies: [R3]
---
# fn-166-mini-activity-a-small-complete-model-to.2 Mini activity: feature file and product level by subtraction from standalone

## Description
The mini activity's feature file and product level, by subtraction from `features/activity_standalone/` ((R3)).

**Size:** M
**Files:** `model/temporal/features/activity_mini/MiniActivity.scala` (signature: `client.start` with `scheduleToStart`, `startToClose`, `maxAttempts` bounded at two; `client.cancel`; the worker's `poll`, `respondCompleted`, `respondFailed`; the two timers; entity, outcomes, `Bounds` entries), `mini/product/Product.scala` (phases with statuses and status facts, `init`, `end`, facets `Dispatching` and `Completion`, capabilities `Closable` and `Pollable`, one `except` waiver with a reason, Properties, one `verify`, one `find`)
**Touches:** [model/temporal/features/activity_mini/**, model/temporal/Bounds.scala, model/framework/IrFile.scala, model/framework/IrFile.test.scala, model/irgen/Structure.scala, model/irgen/Lifting.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/layout/mini/**]

### Approach
- Add `mini` to the existing feature-form discovery and layout support, with focused regression coverage, so both compiled-root discovery and lifting name the form `activity-mini`. This enables the new example without changing the DSL's behavior or editing real features.
- Copy the standalone feature's declarations and delete: pause, unpause, reset, heartbeat, describe, the by-ID client, their phases, facts, effects, rules, Properties and Queries. Keep the names of what stays so the real feature's vocabulary is the mini feature's.
- Two facets at the product level, with one rule in `Completion` whose guard reads a `Dispatching` phase, so the cross-facet case exists from the start.
- The feature file's header comment lists the hard instances the mini feature holds and where each is, and is updated by tasks 3 and 4.
- Lift with `make model-mini`; read `activity_mini/generated/ir.json` end to end once and record its size on the task.

## Acceptance
- [ ] The feature file and product level compile, lift to `features/activity_mini/generated/ir.json` and pass the Go checks and the lint under `FEATURES=activity_mini`.
- [ ] The product level holds two facets with a cross-facet guard, `Closable` and `Pollable`, one waiver with a reason, one `verify` and one `find`, with bounds in `Bounds`.
- [ ] The header comment lists the hard instances present so far.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
