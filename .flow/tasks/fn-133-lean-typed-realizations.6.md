---
satisfies: [R9, R15, R16]
---
# fn-133-lean-typed-realizations.6 Typed realization objects, derived defaults, derived realizations

## Description
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
Part C, R9, R15, R16.

The activity's typed realization objects remain in `model/temporal/features/activity/standalone/system/Realization.scala`, preserving fn-126.11's placement through fn-132. All three existing activity System realizations stay in that file; replacing wrapper objects does not move them to the root or introduce a Product realization file.

- **Typed objects.** `object Standalone extends Realization(ActivityProtocol)`, typed by the machine's state, outcome and fact types. A header holds what is not derived, and named sections in a fixed order: controller, worker scripts, evidence, server steps, controls. Evidence facts, status-table keys and `perform`/`onPath` classes are checked against the machine, with one negative compile fixture each. The wrapper objects (`ActivityRealization`, `NexusRealization`, `OperationRealization`) go.
- **Lint.** The structure lint (fn-126 R20) checks realization sections and their order, and one realization per object.
- **Derived defaults.** The operation is the machine's entity. Roles are those named by the scripts' calls and activations. Deadline and backoff timers get the kit's default server steps. An explicit value equal to the derived one is a lint finding.
- **Derived realizations.** A realization declared from another with steps added or replaced; `forgedCompletion` derives from `asyncNexus`, and the local `realization(machine, steps*)` factory and its concatenated controller go.

Realization and script IDs follow the new objects' fully qualified names (fn-126 decision 23): prove the projection with an ID map.

## Acceptance
- [ ] Every realization is its own typed object; the negative compile fixtures fail to compile; the lint refuses an out-of-order section and an object holding two realizations.
- [ ] No realization states its operation, roles or default server steps.
- [ ] `forgedCompletion` is derived, and replacing a step its base lacks is refused at its line.
- [ ] A before/after projection with the ID map applied is identical.
- [ ] The spec's Verification gates pass.

## Done summary
Every realization is now its own object, typed by its machine, with named sections in a fixed order. Each object's operation, roles and default server steps are derived. `ForgedControl` is derived from `AsyncNexus`. A scratch lift shows each realization equal to the baseline's apart from its ID (and one name), once positions and the fn-133.4/.5 deltas are set aside.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: IMPLEMENTER claude-opus-5-5 at high

### What changed
- **`model/temporal/realize/Objects.scala` (kit):**
  - `abstract class Realizes[S, O, F <: AnyRef](machine, learned, observations, requiredSettings)`.
  - Section bases `Controller`, `Workers`, `Evidences`, `ServerSteps` and `Controls`.
  - `DerivesFrom(base, machine)` with `Changes(inserting(after = step)(items*), replacing(step)(items*))`.
  - Typed members `answered`, `answeredAs` and `delivered` take `fact: F | EveryValue`.
  - `everyValue(Fact.case)` names a fact case with fields by its companion. A machine's fact type holds the case's values, not the companion, so typed declarations take a companion only through this.
- **`DescribedStatus` (Modules.scala)** is now `final class DescribedStatus[F, Req, Rsp, Info, V](machine: Machine[?, ?, F], calls, …)(entries: (F | EveryValue, V)*)`, and `apply`/`await` take `F | EveryValue`. It stopped being a case class because the order lint reads a case class with a Machine field as a Model bundle.
- **Lifter** (`Realizations.scala` §"Realization objects"; `Lifting.liftRoot` dispatches realization objects):
  - It reads the header and sections, then synthesizes the `temporalRealization(…)` call. It emits that call with the usual machinery, so every declaration lifts exactly as before.
  - The realization is named after its object with the first letter lowered; its ID is the object's qualified name.
  - Derived values:
    - The operation is the machine's entity: the `Entity` val of that name closest to the object's package.
    - The roles are the WorkflowService, the Case's task queue, and each kit role whose id or resource binding the scripts and controls name, in kit order.
    - The server steps are deliveries for each activity script's `starts`; `timers.backoff` at `firstRetryBackoffMs` where an activity script runs; and the deadline timer of each Timeout input a bound class expires, at `deadlineMs`.
  - A stated `serverSteps` entry overrides the derived step of its class. Stating one equal to the derived step is refused.
  - Refused at their lines:
    - a section that is not one of the five, or out of order;
    - anything else in the object;
    - a second realization in an object;
    - a binding of an action the machine does not bind;
    - a stated derived server step;
    - a derived change at a step its base lacks.
- **Realization files:**
  - activity system/Realization.scala: `Standalone`, `HeldDelivery` and `LostAdmissionResponse`. Shared declarations are package-level private vals. HeldDelivery has its own `pauseDescribed` DescribedStatus over `HeldDispatch`, so its typed fact stays the machine's; the await name `await-paused` and the evidence id are unchanged.
  - nexus/workflow/Realization.scala: `AsyncNexus`, and `ForgedControl` deriving from it. Shared declarations live in `private object CallerDeclarations`, because package level collides with Workflow.scala's `operation` and `pendingAttempts`. `realization(machine, steps*)` and the concatenated controller are gone.
  - nexus/standalone/Realization.scala: `Standalone`.
- **Outside the realization files (needed exceptions):**
  - Standalone.scala and Workflow.scala of each feature, for the `irFile` roots.
  - Both features' system/System.scala, where `Describable(status = activityStatus / operationStatus)` now reads the package-level `described.table` vals.
  - README describes realization objects.

### Decisions (owner unavailable)
- **Typed classes are checked by the lifter, not the compiler.** A class does not carry its machine in its type. The class check (`Foreign`) is therefore a lifter refusal. Facts and DescribedStatus keys are compile errors, covered by testdata/typedRealizations.
- **The base class is named `Realizes`,** because `umpire.realize.Realization` is the core case class.
- **The Nexus control realization's object is `ForgedControl`,** so its name becomes `forgedControl`. `ForgedCompletion` is retired vocabulary (tools/umpire/ir/layout_test.go `retiredModelVocabulary`).
- **The task-queue role is always bound.** The standalone Nexus realization listed `temporal.task-queue` with no script naming it, so the derivation keeps it alongside the WorkflowService (`boundRoles`).
- **The backoff step applies where an activity script runs.** The Nexus machine has `timers.backoff` too, but its realization never had that step.

### ID map (declared IR delta for the batch regeneration)
| file | old realization id | new realization id | name |
|---|---|---|---|
| activity-standalone.json | temporal.features.activity.standalone.system.ActivityRealization.standalone | temporal.features.activity.standalone.system.Standalone | standalone |
| activity-standalone-race.json | …system.ActivityRealization.heldDelivery | …system.HeldDelivery | heldDelivery |
| activity-standalone-race.json | …system.ActivityRealization.lostAdmissionResponse | …system.LostAdmissionResponse | lostAdmissionResponse |
| nexus-workflow.json | temporal.features.nexus.workflow.NexusRealization.asyncNexus | temporal.features.nexus.workflow.AsyncNexus | asyncNexus |
| nexus-workflow-control.json | temporal.features.nexus.workflow.NexusRealization.forgedCompletion | temporal.features.nexus.workflow.ForgedControl | forgedCompletion → **forgedControl** |
| nexus-standalone.json | temporal.features.nexus.standalone.OperationRealization.standalone | temporal.features.nexus.standalone.Standalone | standalone |

- Each IR file's header `source` lists these objects as roots.
- Script, command and evidence IDs are unchanged by this task.
- With the ID map applied, every realization is equal with positions stripped, apart from the fn-133.4/.5 deltas. Checked per field with /tmp/laneE-tools/rcmp.py on a scratch lift; the ForgedControl controller is item-equal, with inspect inserted after await-scheduled.

### For the batch regeneration
- Go tests that pin the old IDs or names against model/ir need the ID map after regeneration:
  - tools/umpire/check/realizer_test.go:21 (`…NexusRealization.asyncNexus`);
  - tools/umpire/ir/realization_test.go:733 (`heldDelivery`; the name is unchanged, OK);
  - any test naming realization `forgedCompletion` (waits_test and generated_test name the *Query* `forgedCompletion`, which is unchanged).
- Case and manifest files that carry realization IDs change by the ID map.
- `go test ./tools/umpire/ir` fails before the regeneration, on stale model/ir positions; that is expected in-batch. Its other failures (Effects.test.scala and the waivers file) belong to other tasks.
- The model/ir/*.lint.json acceptances added in fn-133.4 keep their owners (realization names): `standalone`, `asyncNexus`, and `forgedCompletion`. The last must become **`forgedControl`** in model/ir/nexus-workflow-control.lint.json after regeneration. It is not renamed here because the lint reads the stale IR until then.

### Tests
- `mise exec -- scala-cli test model/irgen`: 96 passed. Overlap first failed on a fixture that copied model lines; the fixture was rewritten and Overlap re-run with `--test-only`.
- New:
  - the compile refusals `Typed.scala:16:46` (evidence of another machine's fact) and `Typed.scala:25:3` (a DescribedStatus keyed by one);
  - the lifter refusals Unordered (ScriptRejects.scala:191), Doubled (:196), Foreign (:201), Restated (:215) and Replaced (:222).
  - The rejects.txt ScriptRejects block was regenerated from the lifter for those roots; earlier lines moved by one where the `machine =` argument was added.
- The model unit tests pass.
- `--check-syntax` and `--check-comments`: clean. scalafmt: clean.

### Line counts
After .6: activity 303, nexus workflow 305, nexus standalone 83. Baseline was 333 / 514 / 101.

### For later tasks
- Realization objects are roots by their object (`irFile(...)(system.Standalone)`).
- Per-class bindings live in the emitted IR's Performances, which fn-133.8 derives carriers from.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 66439b692d
- Tests: mise exec -- scala-cli test model/irgen, mise exec -- scala-cli test model/irgen --test-only umpire.irgen.Overlap, mise exec -- scala-cli test model/project.scala model/umpire model/temporal, scala-cli run model/check -- --check-syntax, scala-cli run model/check -- --check-comments, scratch lift --ir; realizations equal under the ID map apart from fn-133.4/.5 deltas
- PRs: