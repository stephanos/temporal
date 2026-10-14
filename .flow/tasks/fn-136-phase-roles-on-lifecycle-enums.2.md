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
The activity Product, System and record phase enums now declare roles. Each `states` predicate whose body was a single `in(...)` over exactly one role's cases is now a `p.in[R]` role test; its name, signature and callers are unchanged. A framework closedness check, `Refinement.unclosed` in `model/umpire/Refine.scala`, runs by name on `ActivitySystem.refinement`, `ActivityRecord.refinement` and `HeldDispatch.refinement`, and reports no violations.

stage: impl-review - skipped(config: REVIEW_MODE=none - DSL batch: reviews run once at the batch's end)
stage: plan-sync - skipped(config: planSync.enabled=false)
Tier: lane C, IMPLEMENTER claude-opus-5-5 at high (actual_model: claude-opus-5-5)

Integrated on `umpire` as `a7b4557fe9`, immediately after fn-135.4's integrated commit.
After integration, the combined Models suite passed, including all three Activity closedness tests
and the existing Nexus role/refinement tests; the two Model lint targets and `git diff --check`
also passed.

Baseline: the tree at fn-135.4's commit 41a917b50 in lane-C-p3. The Models compile, and their munit tests are green (43 passed). The lifter suite has only the 3 Model-derived stale fixtures that fn-135.4 declared.

### What changed
- **Product** (`product/Product.scala`, on top of fn-135.4's status declarations):
  - Roles: `scheduled` Waiting, `started` and `cancelRequested` Held, `paused` Suspended. The closures are Succeeded, Failed, Canceled, Terminated and TimedOut, each written `case started extends Phase(Fact.statusStarted), Held`.
  - `terminal` is now `p.in[Closed]`, and `held` is now `is(phase.in[Held])`, a role test inside an `is` block. It lifts like one in a def.
  - Unchanged: `over`, `paused` (OP_EQ), `running` (`started` alone), `pausable`, `status`, `end`, and the rules' inline `in(...)` headings.
- **System** (`system/System.scala`):
  - Roles: `unstarted` none, `scheduled` Waiting, `backingOff` Retrying, `started`, `pauseRequested` and `cancelRequested` Held, `paused` Suspended, and the five closures. The enum is now one case per line.
  - `terminal` is now `p.in[Closed]`, `live` is `p.in[Live]`, `held` is `p.in[Held]` and `waiting` is `p.in[Waiting]`. `end` is unchanged.
- **Record** (`system/Record.scala`, `AdmissionPhase`):
  - Roles: `scheduled` Waiting, `paused` Suspended, `started` Held, `completed` Succeeded and `timedOut` TimedOut. `pausedWhileHeld` has no role and gets a one-line comment saying why.
  - `terminal` is now `p.in[Closed]`. `paused`, `running`, `stopped` and `ActivityRecord.end` (`stopped`) are unchanged.
- **Closedness check** (`model/umpire/Refine.scala`):
  - Signature: `Refinement.unclosed[S, P](refinement)(phase: S => Any, productPhase: P => Any)(using Finite[S]): IndexedSeq[(S, P)]`.
  - It returns every state whose `Closed` role differs from its image's, each paired with that image. `Closed` is tested by a type pattern, because the framework may not use the sugar `in[R]` or `isInstanceOf`.
- **Test** (`standalone/RoleRefinements.test.scala`, `package umpire` as in `StandaloneActivityPins.test.scala`): three tests, one per refinement named, each requiring an empty result.
  - Negative control: with the System's `toProduct` temporarily mapping `timedOut` to the product's `started`, the ActivitySystem test failed and listed each `timedOut` System state with its image. Restored afterwards; the commit does not contain it.

### Identity (R7)
- Script: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/dsl-batch/fn-136.2/ir_positions.py`. It drops `position` keys only, with no other normalization.
- Command: `python3 .flow/tmp/dsl-batch/fn-136.2/ir_positions.py <before>/model/ir <after>/model/ir`. I compared scratch lifts of the Models at fn-135.4's commit (`/tmp/laneC3-ir-1`) and at this commit (`/tmp/laneC3-ir-2`) without regenerating model/ir.
  - All five activity files are "equal but for positions".
  - In the raw diff, every changed line is a `"line"` value or the `:<line>` suffix of a `"position"` string.
  - Every non-activity file is byte-identical.
- Negative control: swapping two members of an `OP_CONTAINS` case list reports 2 differences.
- At the batch regeneration, against the batch baseline, apply fn-135.4's `ir_equiv.py`, which allows positions plus the `states.phase`->`states.status` rename. It covers this task's delta too, because this task adds positions only.

### Declared IR delta (for the batch regeneration)
- Source positions only, in `activity-standalone.json`, `activity-standalone-record.json`, `activity-standalone-race.json`, `activity-standalone.laws.json` and `activity-standalone-record.laws.json`. The shifts come from the enums growing to one case per line: System.scala lines after line 22 (+9 at the end of the file), Record.scala lines after line 33 (+6), and Product.scala's predicate lines.
- No case list, function, name, Definition, Query or Case change. `model/cases` should be byte-identical.
- Lifter fixtures derived from the Models (`expected/hints.json`, `rejections.json`, `hintsRefused.json`): positions only, on top of fn-135.4's delta. Each was checked as "equal but for positions and the renamed status projection" against the checked-in expected file, so the batch regeneration rewrites them.

### Tests run
- `make model/build/model-scala.jar` rc=0
- `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal` rc=0, 46 passed (43 before, plus the 3 RoleRefinements tests)
- the same with `--test-only umpire.RoleRefinements` under the negative control: rc=1, as expected (restored)
- `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen` rc=1: 102 passed. The only failures are the 3 Model-derived stale fixtures, which are the expected batch delta. The role conflict check (R4) ran on all three enums during lifting and passed.
- Scratch lift plus `ir_positions.py`: equal but for positions
- `scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check` (all model trees) rc=0; `make lint-model-models` rc=0; `make lint-model-syntax` rc=0
- GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model, the Behavior-pin regeneration and the Cases byte check run at the batch's single regeneration

### Decisions (owner unavailable; my calls)
- The closedness check takes the refinement object and both phase projections as functions. A refinement does not know its own system machine's projection, and a machine's `Phased` projection is optional (fn-137), so passing `_.phase` explicitly keeps it usable on every refinement.
- The test file sits in `standalone/`, beside `StandaloneActivityPins.test.scala`, as the task suggested. It is not in a level folder.

### For the integrator / later tasks
- Lane B's `NexusRoleRefinements.test.scala` (fn-136.3) has its own `unclosed` helper because this check did not exist yet. Once both lanes are integrated, replace the helper with `Refinement.unclosed(NexusSystem.refinement)(_.phase, _.phase)`. That is a one-line change per test.
- fn-136.5 (retire predicates): the activity predicates that are now role tests are product `terminal` and `held`, System `terminal`, `live`, `held` and `waiting`, and record `terminal`. Product and record `terminal` feed Closable's `terminal` field until fn-137.
- A role-carrying case's type is `Phase & Role`. No activity inference site needed an ascription, and `phase = started` in effect blocks still derives the status, because the lifter finds the enum constructor call among the case's parents.

### Touches
All edits are inside the declared Touches. No MILESTONES.md edit is needed.
## Evidence
- Commits: a7b4557fe92d99c5f1411d53c53ca7ba9aa0684e
- Tests: lane verification: Model tests rc=0 (46 passed, including 3 RoleRefinements); negative control failed as expected and was restored; lifter tests 102 passed with only 3 declared stale generated fixtures; IR identity harness equal except positions; negative control detected, integration: make model/build/model-scala.jar rc=0, integration: mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal rc=0 (combined suite green, including Activity and Nexus role/refinement tests), integration: make lint-model-syntax lint-model-models rc=0; git diff --check rc=0, GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: model gate and Cases byte check run at the batch regeneration
- PRs: