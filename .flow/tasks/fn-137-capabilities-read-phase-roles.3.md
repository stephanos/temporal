---
satisfies: [R4, R5]
---
# fn-137-capabilities-read-phase-roles.3 Migrate every Model to Phased and argument-less Rules

## Description
A mechanical migration of the Temporal Models: each object that names phases mixes in `Phased` and drops its `Rules` argument. Compositions also mix in `Phased` now so later tasks can read them. Behaviour pin: `make umpire-gen-model` must leave model/ir and model/cases byte-identical.

**Size:** M
**Files:** model/temporal/features/activity/standalone/product/Product.scala, model/temporal/features/activity/standalone/system/{System,Record,WithTaskQueue}.scala, model/temporal/features/nexus/workflow/system/{System,TrustingCaller,ClosePolicy}.scala, model/temporal/features/nexus/product/Product.scala, model/temporal/features/nexus/standalone/system/System.scala, model/temporal/shared/worker/Worker.scala, model/temporal/shared/taskqueue/{product,system}/*.scala
**Touches:** [model/temporal/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- `Rules(...)` sites: activity product Product.scala:87; activity system System.scala:179; Record.scala:209 (`ActivityRecord`) and :395 (`HeldDispatch`, a plain `Machine` declared at :369); nexus workflow System.scala:213, TrustingCaller.scala:53, ClosePolicy.scala:559 (`_.caller`, object `RejectAfterClose` at :266); nexus product Product.scala:70; nexus standalone System.scala:93; Worker.scala:68; task queue product Product.scala:75 (`_.outstanding`); task queue system System.scala:143 (`_.custody`). Record.scala:480 is already argument-less and needs no projection.
- Compositions: the activity composition (activity System.scala, around :412, `_.activity.phase`), the Nexus composition (nexus System.scala, around :431, `_.operation.phase`) and the WithTaskQueue compositions.
- Write `Phased[<State>, <PhaseType>](<projection>)` after `Machine[...]` in the parent list. Keep the existing comments.
- If fn-135 has landed, ActivityProduct's projection is named `status`. Mirror whatever the code says at that point.

### Investigation targets
**Required:**
- each site above, read in place
- model/temporal/IrFiles.test.scala: constructs every IR root

### Acceptance
- [ ] No `extends Rules(` remains under model/temporal (grep).
- [ ] Every object listed in spec R4 mixes in `Phased`.
- [ ] `make umpire-gen-model` leaves `git diff -- model/ir model/cases` empty.
- [ ] `make umpire-check-model` and `make lint-model` pass.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
