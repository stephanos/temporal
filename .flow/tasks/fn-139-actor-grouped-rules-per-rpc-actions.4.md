---
satisfies: [R5, R7]
---
# fn-139-actor-grouped-rules-per-rpc-actions.4 Standalone activity: one action per RPC with a Failure enum; every consumer on the new actions

## Description
Splits the standalone activity's signature into one action per RPC and moves every consumer onto the new actions, keeping today's case forms and outcome type. The `from` grouping of the rules is task .5, and the `in` to `when` rename is task .7. This task changes names and their declared structural consequences (spec R7), nothing else.

**Size:** M
**Files:** model/temporal/features/activity/standalone/Standalone.scala, .../product/Product.scala, .../system/System.scala, .../system/Record.scala, .../system/WithTaskQueue.scala, .../system/Realization.scala, any Go test that hard-codes the old activity action or class names (confirm by grep)
**Touches:** [model/temporal/features/activity/**, tools/umpire/lower/**, tools/umpire/conformance/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- These files will already carry fn-135.4, fn-136.2/.5, fn-134.3 and fn-137.3-.6 edits (MILESTONES DSL batch, step 3). Write against the tree as it stands.
- Signature (Standalone.scala:42-115): `client.pause`, `unpause`, `requestCancel`, `terminate`, each `.on(activity)` with its own `.schema[…Request]`. `worker.respondCompleted`, `respondFailed` and `respondCanceled`, each with its own schema. `respondFailed` takes one input `failure = input[Failure]`, with `enum Failure derives Finite: case fatal, retryable`. Move the two `.example(...)` lines to `respondFailed` as `Failure.fatal` → `"ApplicationFailureNonRetryable"` and `Failure.retryable` → `"ApplicationFailureRetryable"`. Give each control the `.results("Delivery")` the old `control` carried. Remove `Control`, `AttemptResult`, `result`, and `Inputs` with its `control` input.
- Rules: each `on(client.control(Control.x))` becomes `on(client.x)`, and `on(worker.respond(AttemptResult.y))` becomes the matching `respond*` action or class (`respondFailed(Failure.retryable)`). The four-control not-found rule `on(client.control)(in(states.terminal) ~> effects.notFound)` becomes one `on(client.pause, client.unpause, client.requestCancel, client.terminate)` block (task .2's multi-action `on`).
- Consumers: properties (`property when worker.respond(...)`), capability arguments (`Pausable(pause = …)` and the others), scenarios, `own(_.activity, …)` in System.scala and WithTaskQueue.scala, Record.scala's rules and derivations, and Realization.scala's `onPath`/`perform` table, where `worker.respond` classes map to `completeAttempt`, `failAttempt`, `failActivity` and `cancelAttempt` (:117-136, :170-173, :209-216, :296-297). `respondFailed`'s two classes keep the failAttempt / failActivity split. Update `restrict(...)`/`rebind(...)` lists that name the old actions.
- Equivalence pin (R7): every rule, property, scenario and realization entry maps one-to-one from an old class to a new action or class. Record the old → new class map in the commit message, with the structural deltas the split causes: the `control` and `respond` step functions' input `Match` disappears (Declarations.scala:670-678, `byInputs`), each new action gets its own step function with the not-found arm repeated in each control, and the schema, `results` and example metadata move to the new actions. The batch regeneration's diff is classified against that list.
- Go edits stay in the files that pin activity action or class names (expected under tools/umpire/lower and tools/umpire/conformance). If a needed edit falls outside Touches, widen it in the commit message, since task .6 edits tools/umpire/check.
- Grep Go and Scala for `client.control`, `worker.respond`, `AttemptResult`, `Control.` and the class strings (`control-pause`, `respond-failed-…`) once you are done. Fix hand-written references, and leave generated IR and Cases to the batch regeneration (memory: glossary renames).

### Investigation targets
**Required:**
- model/temporal/features/activity/standalone/Standalone.scala:42-135
- model/temporal/features/activity/standalone/system/System.scala:179-440
- model/temporal/features/activity/standalone/system/Realization.scala:110-300
- model/temporal/features/activity/standalone/system/WithTaskQueue.scala:60-220
- model/temporal/features/activity/standalone/system/Record.scala:205-430

**Optional:**
- tools/umpire/lower/grouping_spike_test.go:143: fixture action names; check whether they read Models or fixtures
## Acceptance
- [ ] The signature declares `client.start`, `pause`, `unpause`, `requestCancel`, `terminate` and `worker.poll`, `respondCompleted`, `respondFailed` (input `Failure` with `fatal`, `retryable`), `respondCanceled`. `Control`, `AttemptResult`, `result` and `Inputs` are gone.
- [ ] No hand-written Scala or Go outside generated `model/ir` and `model/cases` names an old action, enum or class (grep evidence in the commit message).
- [ ] The commit message lists the old → new class map that the batch diff is checked against (R7).
- [ ] The Models compile (`scala-cli compile` of the model tree), and `StandaloneActivityPins.test.scala` passes.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
