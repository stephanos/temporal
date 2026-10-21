# Workflow Update: a real feature at the mini feature's size, over a workflow task foundation

## Execution status (2026-10-21 backlog audit)

Deferred by the owner. The Rust decision has been made and the active workspace is `model/`; the former Scala tree is frozen under `model/archived/` and is not expected to build (AGENTS.md and MILESTONES.md). The earlier “until the Rust verdict” note is historical, not an automatic release of this spec. Existing task states and old SHIP reviews describe historical checkpoints, not current implementation readiness.

Resume only after an explicit owner decision and a fresh scope/dependency review against the Rust architecture. Scala paths and commands below describe the archived layout; do not run them as current gates or mechanically port the plan. Preserve unfinished requirements and completion evidence; successor Rust work does not discharge this spec without an explicit requirement mapping. Taskless proposals remain deferred captures, not executable plans.


## Conversation Evidence

> user (2026-10-11): "instead of temporal/model/temporal/features/activity_mini - is there a feature in Temporal that is small enough to be completely defined by a model the size of activity_mini? while being a showcase of the capabiltities of the Umpire DSL and a basis for its rapid development trhough iterations?"
> user (2026-10-11): "let's write a spec to model workflow update as a feature."
> agent assessment (accepted by the user's next turn): Workflow Update is the one real feature whose natural semantics cover fn-166's hard-instance list almost one to one; its complete definition needs the workflow task factored as a foundation, the way the task queue is; the timer, signal and query are smaller but show a third of the list or less.

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 40% [user], 30% [paraphrase], 30% [inferred] -->

The mini activity (fn-166) is the DSL's fast loop, but it is a subtraction from the standalone activity: a restriction to bounded inputs with heartbeat, pause and reset left out. The owner asked for a real Temporal feature that is small enough to be **completely** defined at the mini feature's size, is a showcase of the DSL's capabilities, and is a basis for the DSL's rapid development through iterations [user]. Workflow Update is that feature [paraphrase]. Its semantics, documented in `docs/architecture/workflow-update.md`, hold one natural instance of nearly every hard construct fn-166 lists: a refinement from the caller's stages to the registry's states, a status fact that one path records and another never does, a failure model with a negative control (a cleared registry loses admitted updates; a stale workflow task completion must not accept one), bounded in-flight updates, fair progress on the worker, a stage that never rewinds, and a stable public API whose evidence is in history. It also exercises constructs the mini activity does not: the `Idempotent` capability through update ID deduplication, an outcome the caller sees but history does not (rejection), and a composition of the feature with a foundation that itself has attempts and a deadline [inferred].

What makes Update small is the same thing that makes the mini activity small: the dispatch mechanism is a foundation, not part of the feature. The activity owns its dispatch and composes with the task queue; the update is delivered on a workflow task and must compose with one. So this spec adds **the workflow task as a foundation**, `foundations/workflowtask`, beside `foundations/taskqueue`, and **the update feature** over it. The foundation is reusable: query, signal and the Nexus caller all deliver through the workflow task [inferred].

The feature is built on fn-166's ground: a feature folder, its own `generated/` tree, `make model FEATURES=update`, the hard instances named in its header, and migration by scalafix rule like every feature. It is a second proving ground for the DSL batch, not a replacement of the mini activity; whether it becomes the loop's default feature and the guide's running example is the owner's call once it exists [inferred].

## Architecture & Data Models
<!-- scope: technical -->

**The workflow task foundation.** [inferred] `model/temporal/foundations/workflowtask/`, grounded in `docs/architecture/speculative-workflow-task.md` and `service/history/workflow`. One task at a time, as the task queue holds one message at a time.

- The signature: the entity `workflowTask`, keyed by its scheduled event; the actions `system.schedule(kind)` with `kind` in `normal | speculative`, `worker.start` (the poll that delivers it), `worker.complete`, `worker.fail`; the timer `timers.startToClose` (the workflow task timeout); the inputs the feature shares with it.
- Product level, the opaque contract a feature may rely on: phases `none`, `scheduled`, `started`, `completed`, `failed`, `timedOut`; attempts `UpTo[2]`; a completed or failed or timed-out task returns to `none` for the next one. Facts: `taskScheduled`, `taskStarted`, `taskCompleted`, `taskFailed`, `taskTimedOut`.
- Protocol level: what the Product does not show. A *speculative* task is never written: its scheduled and started events are recorded only if its completion persists them (an update is accepted, or the workflow produced commands), and discarded otherwise. A *transient* task (attempt two) records its scheduled and started events only on completion. A stale completion, one whose task the server no longer holds, is rejected as not found (`Fenced`). The Protocol refines the Product: a discarded speculative task maps to `none` with no fact, which is the shared-status-fact and unobservable-outcome instance of this spec.
- Capabilities: `Retryable` (attempts bounded at two, with the failed-task backoff timer), `Expirable` (the start-to-close deadline), `Fenced` (stale completion rejected), `Pollable` (start is the worker's poll).
- Like the task queue, the foundation declares no Queries or realization of its own: the features that compose it export it.

**The update feature.** [inferred] `model/temporal/features/update/`, three levels, with a header naming each hard instance it holds.

- The signature: the entity `update`, keyed by `updateId` and referring to its `workflow`; the inputs `waitStage` (`accepted | completed`) and `updateId` (`fresh | repeated`); the client's actions `client.update(waitStage, updateId)` and `client.poll` (`PollWorkflowExecutionUpdate`); the worker's answers on the delivering task, `worker.accept`, `worker.reject`, `worker.complete(result)` with `result` in `success | failure`; the workflow's own `workflow.close` (completion, termination or timeout, which abort in-flight updates alike at the Product level); the fault actor's `fault.clearRegistry` and `fault.failTask`; the timer `timers.longPoll` (the server's long-poll expiration, which returns the reached stage instead of an error).
- Product level, what a caller relies on: phases `admitted`, `accepted`, `completed`, `rejected`, `aborted`. Facts `updateAccepted` and `updateCompleted` (the two history events); `rejected` records nothing and ends the entity, since a rejected update leaves no trace and cannot be deduplicated. Rules: an update to a closed workflow is rejected with the server's error; a repeated `updateId` of an in-flight or completed update is answered as the first was (`Idempotent`); `client.poll` answers the reached stage and the outcome once completed; `workflow.close` aborts an admitted update with an error and fails an accepted one with the server's "workflow completed" failure (the abort matrix, rows Admitted and Accepted). Properties: `stageNeverRewinds`, `rejectedLeavesNoHistory` (an `.excludes` witness), `acceptedSettles` (accepted leads to completed or aborted).
- Protocol level, how the server gets there: phases `created`, `admitted`, `sent`, `accepted`, `completed`, `rejected`, `aborted`; `inFlight: UpTo[2]` for `maxInFlightUpdates`; the registry's durability as a field, `durable` false until accepted. The refinement maps `created | admitted | sent` to the Product's `admitted`; the provisional states of `state.go` are the Protocol's unobservable steps. The failure model: `fault.clearRegistry` loses an update in `created | admitted | sent` (the registry is in memory; the caller is told to retry), and `fault.failTask` aborts a `sent` update with the matrix's non-retryable error while an `admitted` one is retried. The negative control, `TrustingRegistry`, is the derivation that accepts a worker's acceptance for an update the registry no longer holds; the real Protocol answers it not found, and the `verify` of the control's `find` is the stale-answer pair of this feature. The hole: resurrection (an acceptance that carries the original request recreates the update) is declared as the path this feature leaves open. The monitor: `stageNeverRewinds` over the Protocol's phases, with the hole as its one named exception. Progress: `admitted` leads to `sent` under `assume.fair(worker.start)`, within the dispatch bound.
- Composition: `UpdateOverTask`, the Protocol with the workflow task Product, two syncs: the update's `send` with the task's `start`, and each worker answer with the task's `complete`. The worker's answers travel in one completion, so the channel is the task's completion, and a lost completion is the task's timeout, not a separate lossy channel. `cross-facet guard`: the Completion facet's abort rule reads the Admission facet's `durable` field.
- Facets: `Admission` (update, poll, dedup, limits), `Delivery` (send, the task syncs, fair progress), `Completion` (accept, reject, complete, abort, close). One `except` waiver: `Closable.closedIsRejectedUniformly` is waived because a repeated `updateId` of a completed update is answered with its outcome after close (the `Idempotent` reading, as the standalone Nexus operation's controls already do). One `overriding` with a reason: the Protocol's abort of an accepted update overrides `Retryable`'s "attempt again" because an accepted update is failed, not retried, when the workflow closes.
- Realization level: a controller starts a workflow with the Case's own worker, calls `UpdateWorkflowExecution` through `InvokeRpc` on the workflow service, and the worker answers the delivering workflow task with the update's acceptance, rejection or completion message. Evidence: `WorkflowExecutionUpdateAccepted` and `WorkflowExecutionUpdateCompleted` through `HistoryEvidence`, and the RPC's response (reached stage, outcome, rejection failure) through `ResponseRead`, which is how a rejection is observed although history records nothing. `.live` witnesses: `syncCompleted` (accepted and completed on one task) expects `satisfied`; `rejected` expects `satisfied` through the response alone; `abortedByClose` expects `satisfied` with the failure; the registry-clearing fault's witness expects `inconclusive` with its reason until the fault is injectable.

**The hard instances, one of each, beside the mini activity's.** [inferred]

| Concept | Instance in Update |
| --- | --- |
| Refinement | Protocol `created | admitted | sent` refine the Product's `admitted`, one `query verify` of `stageNeverRewinds` through it |
| Shared input | `waitStage` on `client.update` and on the long-poll timer's rule |
| Shared status fact | a discarded speculative task and no task both show the Product's `none`; a rejected update and no update both record nothing |
| Facets | `Admission`, `Delivery`, `Completion`; `Completion`'s abort reads `Admission`'s `durable` |
| Capabilities | `Closable`, `Idempotent`, `Pollable`, `Retryable`, `Expirable`, `Fenced`; one `except` waiver, one `overriding` with a reason |
| Composition and channel | the Protocol over the workflow task Product, two syncs; the task's completion is the channel |
| Derivations | `TrustingRegistry`, the `NegativeControl` that accepts a stale acceptance |
| Progress | `admitted` leads to `sent` under `assume.fair(worker.start)` |
| Monitor, hole, assumption | `stageNeverRewinds`; resurrection as the hole; `registryVolatile` as the assumption |
| Realization | every fact's evidence, `.live` witnesses with expectations and reasons, one `deadlines` use (the workflow task timeout) |
| Queries | one `find`, one `verify`, witnesses, bounds in `Bounds` |

**Bounds.** [inferred] `inFlight` and the task's `attempts` are `UpTo[2]`; the Queries name `three` to `six` from `Bounds`; the composition's free verify names one bound past the composed table's depth, added to `Bounds` with its reason, as the Nexus close policy's were.

**Size.** [inferred] The foundation about 400 lines across its two levels; the feature about 800 across three, the mini activity's budget. The hard-instance table is the completeness check: a construct the DSL gains later is added to the feature as an instance, not to the mini activity alone.

## API Contracts
<!-- scope: technical -->

- `make model FEATURES=update` compiles the feature, `foundations/workflowtask`, `foundations/taskqueue` (if the workflow task Protocol composes it) and the shared code; its outputs are `features/update/generated/`. [inferred]
- `foundations/workflowtask` exports its Product and Protocol machines through the features that compose it, declares no Query, Case or realization, and declares its bounds in `Bounds`. [inferred]
- The update feature's IR declares: the Product and Protocol machines, the composition, the derivation, every Property, Query and witness, the realization with its evidence declarations and `.live` expectations. [inferred]
- The realization's worker answer is one new worker instruction, `UpdateReply` (acceptance, rejection with a failure, or completion with a payload or failure), beside `NexusHandlerReply`; its Go half (the instruction, the Driver's handling, and the SDK worker's message in `RespondWorkflowTaskCompleted`) is deferred to the sweep with every other Go change. Until then the realization lowers to Cases whose worker step the lowering reports as unrealizable, by name. [inferred]
- Every rejection code and failure the Product names is read from the server, not guessed: `ErrWorkflowCompleted` for an update to a closed workflow, `NotFound` for a repeated rejected update, `acceptedUpdateCompletedWorkflowFailure` for an accepted update whose workflow closes. [paraphrase]

## Edge Cases & Constraints
<!-- scope: technical -->

- A rejected update cannot be deduplicated: a repeated `updateId` after rejection is a new update, which the Product's `Idempotent` reading must state as the exception, and `rejectedLeavesNoHistory` must hold on every path. [paraphrase]
- A speculative task carrying a rejected update is discarded with its events; one carrying an accepted update is persisted with them. The foundation's Protocol must show both, and the refinement to `none` must not lose the accepted case. [paraphrase]
- An old SDK that ignores the update message is rejected on its behalf by the server; the feature does not model old SDKs and says so in its header. [paraphrase]
- `maxInFlightUpdates` is a bound, not a rejection the feature designs around: the third in-flight update is rejected with resource-exhausted, one row. [paraphrase]
- Continue-as-new routes admitted updates to the new run and fails accepted ones; the feature closes the workflow without a new run and leaves continue-as-new out (Boundaries). [paraphrase]
- The long-poll expiration returns the reached stage, not an error; the Product's `client.update` therefore has an outcome per stage and the realization's `ResponseRead` reads the stage. [paraphrase]
- Only `model/` is edited (owner, 2026-10-11): the feature, the foundation, `Bounds`, the realization's Scala declarations. No Go, no regeneration of other features, no live run until the sweep. [user, MILESTONES current mode]
- The feature is written against the DSL as it stands when its task starts; every later DSL spec in the batch migrates it by rule like the real features (fn-166 R6). [inferred]

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `model/temporal/foundations/workflowtask/` holds a Product and a Protocol level, the Protocol refining the Product with speculative, transient and normal tasks, bounded attempts, the start-to-close timer and a stale completion rejected; within about 400 lines; compiled in every feature set that composes it. [inferred] Errors: a stale completion that is not rejected fails the `Fenced` law; a speculative task whose discard records a fact fails the refinement.
- **R2:** `model/temporal/features/update/` holds Product, Protocol and realization levels within about 800 lines, lifts to `features/update/generated/ir.json`, passes the Go checks, and every row of the hard-instance table is present and named in the feature file's header comment. [user, inferred] Errors: a missing instance is a review finding on the task; a lift failure names the construct.
- **R3:** The Product's claims hold on every path within their bounds: `stageNeverRewinds`, `rejectedLeavesNoHistory`, `acceptedSettles`, the `Closable` and `Idempotent` laws with the one waiver. [inferred] Errors: a violated claim is reported with its path by the checker; a waiver without a reason fails the lint.
- **R4:** The Protocol's failure model finds the stale acceptance in `TrustingRegistry` (`find`) and verifies its rejection in the real Protocol (`verify`) at the same bound and total; the registry-clearing fault loses an admitted update and never an accepted one. [inferred] Errors: a `find` that returns no path or a `verify` that fails names the Scenario.
- **R5:** The composition's syncs and fair-progress claim check within their bounds, with the composed free verify's bound declared in `Bounds` with its reason. [inferred] Errors: a bound declared in the feature file fails the lint.
- **R6:** The realization declares evidence for every fact, a `.live` expectation with a reason for every witness it runs, one `deadlines` use, and the `UpdateReply` worker instruction as a Scala declaration whose Go half is recorded as deferred; lowering reports the unrealizable step by name rather than failing. [inferred] Errors: a fact without evidence fails the lift; a witness without an expectation fails the lint.
- **R7:** `make model FEATURES=update` runs compile, lift, Go checks and the IR diff under fn-166's loop budget, and the full set (`make model`) is not made red by this spec beyond features already behind. [inferred] Errors: a run over budget records its stage times on the task.
- **R8:** The hard-instance table in this spec and the feature's header agree, and `MILESTONES.md` names the feature as the second proving ground. [inferred] Errors: none beyond review.

**Error cases (negative-cases discipline):** each criterion states its error cases inline.

## Boundaries
<!-- scope: business -->

- Not modeled: update-with-start, reapply after reset, continue-as-new routing, the payload-size and total-updates limits with their continue-as-new suggestion, durable admitted (future), sticky task queues, replication, old SDKs that ignore messages. Each is one line in the feature's header. [inferred]
- The foundation models one workflow task at a time and no history buffer: events between scheduled and started are out of scope. [inferred]
- No Go change: `UpdateReply`, the Driver, the lowering's new step, Case generation for the feature, pinning and the live run are the sweep's. [user, current mode]
- The mini activity is not replaced, moved or edited; making the update the loop's default feature or the guide's example is a separate owner decision. [inferred]
- Query and signal over the same foundation are follow-on features, not this spec. [inferred]

## Decision Context
<!-- scope: both -->

- Candidates weighed (2026-10-11): the user timer (complete in about 400 lines, no worker, shows a third of the list), signal (durable at request time, no failure model worth a control), query (no attempts, evidence outside history), and update. Update was chosen because its documented semantics map onto fn-166's hard-instance list almost one to one, add `Idempotent` and an unrecorded outcome, and its API is stable with evidence in history. [paraphrase]
- The workflow task is a foundation, not part of the feature, for the reason the task queue is: the feature stays at the mini size, and query, signal and the Nexus caller reuse it. Modeling the task inline was rejected as doubling the feature. [inferred]
- The feature composes with the foundation's Product, as the mini activity composes with `TaskQueueProduct`; the foundation's own Protocol is checked in the foundation, so a feature's check stays bounded. [inferred]
- The worker's answers are not a separate lossy channel because they travel in one task completion; the task's timeout and the stale completion are the loss and the staleness. The mini activity keeps the explicit channel instance. [inferred]
- Placement: after fn-166 and before fn-159, so the batch iterates on two proving grounds and the update is migrated by rule with the real features; written against the final DSL instead would avoid migration churn but lose it as a proving ground. The owner may reorder. [inferred]
- One folder, `features/update/`, since the kind has one form; a `<kind>_<form>` split is added when a second form (update-with-start) arrives. [inferred]
- Deferred by the owner on 2026-10-11, the same day it was captured, listed under "Deferred for now" in `MILESTONES.md`; the placement decision above is taken when it is picked up. [user, "mark as deferred for now"]

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.

## Early proof point

Tasks 1 and 2 together: if the foundation's Product and the update's Product do not compose within `six` with the inventory tests passing, or the update's Product cannot state `rejectedLeavesNoHistory` without a construct the DSL lacks, the construct is a DSL spec of its own before the Protocol level is written.

## Execution

Order: the workflow task foundation (1), the update's Product (2), the update's Protocol with its failure model, composition and derivations (3), the realization's Scala declarations and `.live` witnesses (4). Each task ends with the mini feature's loop for the update (`make model FEATURES=update`), the lifter fixture suite, and the Scala suites once for the full set.

## Quick commands

```sh
make model FEATURES=update              # the loop for this feature
make model FEATURES=activity_mini,update  # both proving grounds
make model                              # full set, at task end
mise exec -- scala-cli test model/irgen   # at task end
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Workflow task foundation, two levels | 1 | |
| R2 | Update feature with the hard instances | 2, 3, 4 | |
| R3 | Product claims and capability laws | 2 | |
| R4 | Failure model, negative control, fault | 3 | |
| R5 | Composition, progress, bounds | 3 | |
| R6 | Realization declarations, `UpdateReply` deferred | 4 | |
| R7 | Loop budget, full set not reddened | 1, 2, 3, 4 | |
| R8 | Spec, header and MILESTONES agree | 4 | |

## Redesign impact (fn-173, 2026-10-13)

Owner-approved redesign `fn-173` (reachable exploration; `model/design/reachable-exploration.md`) drops full catalog tables, Scala step tables and equivalence sealing, and Query totals; the Behavior Fingerprint becomes an IR structural hash. For this spec:

- **R4:** drop authored totals.
