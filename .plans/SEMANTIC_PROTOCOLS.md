# Semantic protocols: capabilities and the laws they bring

Design exploration, 2026-10-03. Question: instead of every Model restating what Describe, Terminate,
Pause, Cancel, timeouts and retries mean, declare each as a reusable behavioral contract that a feature
instantiates. Grounded in `model/temporal/*`, `model/SEMANTICS.md`, fn-112/114/118/119/120, and the
server's handlers (`chasm/lib/{activity,nexusoperation,scheduler}`, `service/history/api/*`).
Nothing here is a spec yet; the last section is the proposed spec text.

## 1. Operations that recur, and their families of promises

Legend: **U** universal (holds for every entity that has the operation), **P** parameter (true of every
entity, but the value differs), **X** entity-specific (not a promise of the protocol; stays in the feature).

### Close (every entity with a terminal status set)

The base every other control rests on. Entities: workflow, standalone activity, Nexus operation
(caller-side and CHASM), schedule, callback, admission record.

- U `terminalStatesAreFinal`: no row leaves the terminal set (today `terminalIsFinal` ×2, `terminalStays` ×3).
- U `closedIsRejectedUniformly`: every step from a terminal state is a stutter with a rejecting outcome
  (the Model's `Outcome.notFound` on every control); phrased over all steps because a transition
  Property with `when` is unsupported (SEMANTICS, Claims).
- U `describeReflectsClose`: the status read maps the terminal phase (Describable below).
- P `rejection`: what a mutation of a closed entity answers. Activity `NotFound` (`errClosed`), workflow
  `NotFound` (`ErrWorkflowCompleted`), Nexus op `ErrOperationAlreadyCompleted`, schedule
  `FailedPrecondition` (`ErrClosed`); workflow *cancel* on a closed workflow answers success (noop).
- X retention/deletion after close.

### Terminate (workflow, standalone activity, Nexus operation, schedule)

- U from a live phase, terminate settles the entity as `terminated` in one step and records it once
  (activity `statusTerminated`; workflow `WorkflowExecutionTerminated`; Nexus `EventTerminated`).
- U in-flight work is dropped: a started workflow task is failed `FORCE_CLOSE`; a later worker answer
  for an activity attempt is `NotFound` ("activity task not found"); the outcome is overwritten.
- U a reason and an identity are recorded (activity `TerminatedFailureInfo{identity}` + message;
  workflow event; Nexus op terminate state). Schedule records neither: `Closed = true` only. So the
  promise is U for executions, **absent** for the schedule; the law must be declared opt-out there.
- P second terminate: activity same request id → OK, else `rejection`; Nexus op same id → OK, other id →
  `FailedPrecondition("already terminated with request ID …")`; workflow no request id, `rejection`.
- X terminate of a child only (`childWorkflowOnly`), `FirstExecutionRunId` across runs, `deleteAfterTerminate`.

### Pause / Unpause (workflow, standalone activity, schedule)

- U paused is live, not terminal; `unpause` returns to the pre-pause progress (activity `paused → scheduled`,
  `pauseRequested → started`); pause/unpause record identity and reason (workflow events; activity
  `LastPauseState`; schedule `Notes` + event log).
- U×Dispatchable `pausedIsNotDispatched`: while paused, no work is handed to a worker (activity:
  `notAdmittedWhilePaused`; schedule: `useScheduledAction` returns false; workflow: no workflow task is
  created on pause). This is the interaction law the owner named.
- P pause of held work: activity → `pauseRequested`, reported `RUNNING` until the worker yields
  (grounded: fn-118 inventory); workflow/schedule pause at once.
- P already paused: workflow `FailedPrecondition` unless same request id; activity `FailedPrecondition`
  ("non-pausable state") unless same request id; schedule idempotent. Unpause when not paused: workflow
  and activity `FailedPrecondition`; schedule idempotent.
- X reset keeps paused (activity `resetKeepPaused`); pause-on-failure of a schedule.

### Cancel (request) (workflow, activity, Nexus operation)

- U cancel is a *request*: it records intent (identity, reason, request id) and leaves the entity live
  while work is in flight; a canceled answer settles only a cancel-requested entity (`canceledByWorker`;
  Nexus schedules the cancellation only once started).
- P with no work in flight: activity settles `canceled` at once; workflow needs a workflow task.
- P already requested: workflow noop success; activity same id OK, other id `FailedPrecondition`;
  Nexus `ErrCancellationAlreadyRequested`.
- P on closed: see Close `rejection` (workflow: success).

### Describe (every entity)

- U the read is a function `σ: Phase → Status` of the current phase, keyed by the entity's key, and
  stays readable after close; U the fact a step records is the status it lands in (the activity Model's
  nine `status*` facts are this law applied nine times).
- P `σ` and the status enum (`pauseRequested ↦ RUNNING`); P visibility at once or eventually per
  write→read pair (fn-118 owns this; it is a hint, not a law).

### Deadlines (workflow, activity, Nexus operation, admission record)

- U a timer armed for phase set `P` fires only there, settles `timedOut` and records which deadline;
  U competing deadlines: the first to fire wins, the other never fires afterwards (Close law).
  Today 7 Properties (`scheduleToStartFires` ×2, `startToCloseFires` ×2, `scheduleToCloseFires`,
  `scheduleToStartTimesOut`, `scheduleToCloseTimesOut`).
- P which timers exist and their phase sets.

### Retry (activity attempts, Nexus invocation attempts, callbacks)

- U a retryable failure returns to the scheduled-like phase with attempt+1 and no visible event besides
  the count; non-retryable settles `failed`; the backoff is unobservable. Today `retryCompletes`,
  `retrySucceeds`, `nonRetryableFails`, `handlerErrorFails`.
- P attempt bound, which answers are retryable, whether the count is readable.

### Not recurring enough to be a protocol now

Reset (workflow and activity differ in what is kept; the activity Model defers it), Poll/await-result
(fn-118 blocking-read candidate; no Model reads results yet), Update, Signal, Query.

## 2. What a protocol is, in Umpire terms

Three things, none of which needs an IR schema field.

**A capability** is a typed declaration on a machine that binds the protocol's parameters to the
machine's own vocabulary: status predicates (`terminal`, `paused`, `running`), actions or classes
(`terminate = control(Control.terminate)`, `dispatch = attemptStart`), the rejecting outcome, a status
map for Describable, and how to reach a live state (`reach = Seq(start())`) for the Scenarios laws need.

```scala
object Product:
  def terminal(s: State): Boolean = s.phase.in(completed, failed, canceled, terminated, timedOut)
  def paused(s: State): Boolean   = s.phase == paused
  def running(s: State): Boolean  = s.phase == started

val activityProduct = machine[Product] { … }          // unchanged

val activityCapabilities = capabilities(activityProduct)(
  Closable(terminal = Product.terminal, rejected = Outcome.notFound),
  Terminable(terminate = control(Control.terminate), settled = ProductFact.statusTerminated,
             reach = Seq(start())),
  Pausable(pause = control(Control.pause), unpause = control(Control.unpause), paused = Product.paused,
           reach = Seq(start())),
  Dispatchable(dispatch = attemptStart, running = Product.running),
  Describable(status = Field[ActivityExecutionInfo, ActivityExecutionStatus](_.status),
              σ = Product.statusOf)
)
```

**A law** is an ordinary top-level Scala `def` in `temporal/laws` (Temporal kit) or `umpire/laws`
(entity-neutral), whose body the lifter lifts like any step function. Its parameters are the
capability's fields. Three forms, each lowering to IR that exists:

| Law form | Lowers to | Example |
| --- | --- | --- |
| single capability, one machine | transition or same-step Property on that machine + a `verify` over the machine's free Scenario from its start | `terminalStatesAreFinal(terminal)(before, after) = terminal(before) implies after.state == before.state` |
| interaction of two capabilities of one machine | the same, over both capabilities' parameters | `pausedIsNotDispatched(paused, running)(before, after) = paused(before) implies !running(after.state)` |
| interaction across two entities | a Property of a composition, `after.records(_.member, fact)` (fn-112 R3) | `servedOnlyByPollingWorker(dispatch)`: today `startedByPollingWorker`, `repliedByPollingWorker` |
| functional (a `find` with a Scenario) | Scenario from `reach ++ Seq(action)`, a `when action holds` Property, `.expect` | `terminateSettles`: `reach, terminate` ⇒ `terminal(after.state) && after.records(settled)` |

A law states its scope in its doc comment and in a `promises`/`doesNotPromise` pair that lint prints:
`terminalStatesAreFinal` promises no row leaves the terminal set; it does not promise the status is
reported, that a second terminate is rejected, or that history records the close once. Each of those is
its own law so that an entity can hold one and waive another.

**The catalog** says which capability, and which pair, brings which laws. Pairs are found by the
lifter from the capability set, never listed by the author: a machine declaring `Pausable` and
`Dispatchable` gets `pausedIsNotDispatched`; one declaring `Closable` and `Terminable` gets
`terminateSettles` and `secondTerminateIsRejected`. Laws through a refinement: a capability declared on
the product is read on the protocol through the refinement the protocol declares (fn-112 R8), and a
law that needs protocol-only vocabulary (`pauseRequested`) is declared on the protocol's own capability.

**Opt-out and override, with a recorded reason:**

```scala
capabilities(schedule)(Terminable(…)).except(recordsReasonAndIdentity, because = "a schedule's close records no reason: chasm/lib/scheduler/scheduler.go Terminate")
capabilities(workflow)(Cancelable(…)).overriding(closedIsRejectedUniformly -> Workflow.cancelOfClosedSucceeds, because = "requestcancelworkflow/api.go answers success on a completed workflow")
```

`except` lifts no Property and no Query for that law; `overriding` lifts the entity's own def under the
law's name. Both are refused without a reason. The reasons are written by the lifter into the
accepted-findings file fn-120 R7 defines, so lint has one source of intended gaps and fails when a
reason names a law that is no longer in the catalog.

**Where this sits against the task-queue contract (fn-112 R20).** The queue is a *reusable entity*: an
opaque contract machine that providers refine and compositions replace. A capability is not a machine;
the entity's own machine stays the only machine, and the laws are read on it. The two mechanisms meet
in cross-entity interaction laws, which are composition Properties the queue entity and the feature
share. Trying to make every capability an opaque machine fails for a reason the DSL already has: a
machine declares at most one `refines`, and the protocol already spends it on the product.

**What is missing.** (a) Lifter: binding function-valued arguments at lift time. `fold(…, env)` binds a
def's machine argument today (`admissionQueries(m)`), but `resolve` refuses `Flags.Param` and `callee`
resolves only top-level `DefDef`s, so `terminal(before)` inside a law body is refused. The fix is in the
fold: an argument that is a reference to a lifted top-level def is bound in the env and `callee` reads
it there. (b) Lifter: the `capabilities(m)(…)` declaration and its catalog expansion into Properties
and Queries named `<machine>.<law>`, plus `except`/`overriding`. (c) Reader: nothing. A transition
Property with `when` stays unsupported and every law above avoids it. (d) IR: nothing; `Query.total`
(fn-112 R19) is computed by the lifter for generated Queries from the same formula.

## 3. Value and risks

**Claims that become law instances** (Property declarations today, across all Models):

| Law | Capability | Instances today |
| --- | --- | --- |
| `terminalStatesAreFinal` | Closable | `terminalIsFinal` (activity, nexus), `terminalStays` (admission + 2 composition families) = 5 |
| `pausedIsNotDispatched` | Pausable × Dispatchable | `pausedIsNotDispatched`, `notAdmittedWhilePaused` ×3 = 4 |
| `atMostOneActive` | Dispatchable (admission) | 3 |
| `servedOnlyByPollingWorker` | Dispatchable × Worker | `startedByPollingWorker`, `repliedByPollingWorker` = 2 |
| `deadlineFires(timer, phases, fact)` | Deadlined | 7 |
| `terminateSettles`, `cancelIsRequested`, `canceledAnswerSettlesOnlyRequested` | Terminable, Cancelable, Cancelable × Attempted | 3 |
| `retryReschedules`, `nonRetryableSettles` | Retryable | 4 |
| `recordedStays(projection)` | Closable on a field | `knowledgeIsFinal`, `closedHistoryIsFrozen`, `handlerEffectIsIrreversible` = 3 (borderline: finality of a projection, not of the phase) |

About 31 of roughly 60 Property declarations, and the `*.any.*`, `terminalHolds` ×2, `pauseHolds`,
`cancelRequest` and `stoppedWorker*` ×2 Queries (about 15 verify Queries) are generated. Genuinely
feature-specific and staying: the close-policy design claims (`outcomePreserved`, `ackOnlyWhenKept`,
`noUnnecessaryWait`, `routedToSuccessor`, `lostAfterReset`, `reappliesRetained`, the three
intent/receipt/knowledge gaps), `failedCommitKeepsTheMessage`, `staleDeliveryRejected`,
`committedDespiteLostResponse`, `asyncStarts`, the competing-timer Scenarios, the forged-success
control, and the queue's `delivers`/`committedStays`/`storageLossDrops`, which become the queue entity's
own laws under R20. Settlement tables (`completes`, `syncSucceeds`, `completionSucceeds`…) could be one
table-driven law but read better as the feature's own content; not proposed.

### Inventory: every claim of `model/temporal`, classified (fn-122 task 1)

The table above is the design-time estimate. This is the classification fn-122 counts by. It covers
every Property, monitor and progress claim under `model/temporal/**` at fn-122 task 1: 60
Properties, 6 monitors and 10 progress claims, 76 in all. Each falls in one of three classes. A
single-capability law or an interaction law names its law. A feature-specific claim names why it
stays authored. No claim is left unclassified. `Pollable` is the spec's name for this document's
`Dispatchable`.

**The catalog.** The defs live in `model/temporal/capabilities/{Close,Terminate,Cancel,Pause}.scala`
(fn-122 task 8 moved the two formerly entity-neutral Close laws there from `model/umpire/laws`). The
one `given Catalog` is in `model/temporal/capabilities/Catalog.scala`, beside the six capability
kinds (`Capabilities.scala`); the framework keeps only `CapabilityOf`, `CapabilityKind`, `Law` and
`Catalog`.

| Law | Brought by | Instantiating machines (planned) | Today's instances |
| --- | --- | --- | --- |
| `terminalStatesAreFinal` | Closable | `activityProduct`, the admission record (`AdmissionState`), `nexusOperation` | `terminalIsFinal` (activity product), `terminalStays` (admission bundle; 7 machines, 1 entity) |
| `closedIsRejectedUniformly` | Closable | same | none: a new claim (the activity product's `control` arm on a closed phase states it inside the step function) |
| `pausedIsNotDispatched` | Pausable × Pollable | `activityProduct`, the admission record | `pausedIsNotDispatched` (activity product), `notAdmittedWhilePaused` (admission bundle; 7 machines, 1 entity) |
| `terminateSettles` | Terminable | `activityProtocol`, `nexusOperation` | `terminated` (activity protocol) |
| `cancelIsRequested` | Cancelable | `activityProtocol`, `nexusOperation` | `cancelRequestedWhileStarted` (activity protocol) |

**Planned capability declarations** (the set the catalog test counts until task 3 declares them).
An instantiating entity is a machine with its own state type that declares the capability.
`staleAdmission` is derived from `currentAdmission`. The compositions read the record through their
`activity` member. `heldAdmission` shares `AdmissionState`. None of these counts again.

| Machine | State type | Declares | Task |
| --- | --- | --- | --- |
| `activityProduct` | `ProductState` | Closable, Pausable, Pollable | 3 |
| `currentAdmission`, `staleAdmission` and the composition families | `AdmissionState` | Closable, Pausable, Pollable | 3 |
| `activityProtocol` | `ProtocolState` | Terminable, Cancelable, Describable | 3 |
| `nexusOperation` | its own | Closable, Terminable, Cancelable, Describable | 4 |

**Classification.** Machines are listed by the declarations that carry each claim. "The 7" are
`currentAdmission`, `staleAdmission`, `currentOverQueue`, `staleOverQueue`, `currentOverMatching`,
`staleOverMatching` and `currentOverLossyMatching`. They instantiate the admission bundles
(`AdmissionClaims`, `OverQueueClaims`, `OverMatchingClaims`), all over the one record entity.

| Claim | Kind | Machine(s) | File | Class | Law or reason |
| --- | --- | --- | --- | --- | --- |
| `terminalIsFinal` | Property | activityProduct | standaloneactivity/Properties.scala | single-capability law | `terminalStatesAreFinal` |
| `pausedIsNotDispatched` | Property | activityProduct | standaloneactivity/Properties.scala | interaction law | `pausedIsNotDispatched` |
| `terminalStays` | Property | the 7 | admission/, compositions/Properties.scala | single-capability law | `terminalStatesAreFinal` |
| `notAdmittedWhilePaused` | Property | the 7 | admission/, compositions/Properties.scala | interaction law | `pausedIsNotDispatched` |
| `terminated` | Property | activityProtocol | standaloneactivity/Properties.scala | single-capability law | `terminateSettles`; it also fixes the phase, so retiring it waits for the Case-equality test of R5 |
| `cancelRequestedWhileStarted` | Property | activityProtocol | standaloneactivity/Properties.scala | single-capability law | `cancelIsRequested`; also fixes the phase, as above |
| `terminalFinality` | monitor | currentAdmission, staleAdmission, heldAdmission | admission/Model.scala | single-capability law | `terminalStatesAreFinal` in monitor form; weaker than `terminalStays` (it does not keep the phase), so the generated Property subsumes it. It stays as the Run monitor the realization reads |
| `atMostOneActive` | Property | the 7 | standaloneactivity/Properties.scala | feature-specific | one instantiating entity: only the record counts active attempts |
| `atMostOneActiveAttempt` | monitor | currentAdmission, staleAdmission, heldAdmission | admission/Model.scala | feature-specific | the record's active count, as above |
| `startedByPollingWorker` | Property | standaloneActivity | standaloneactivity/Properties.scala | feature-specific | cross-entity worker claim; its twin `repliedByPollingWorker` is in the Nexus caller, which declares no capability here |
| `completes`, `canceledByWorker` | Property ×2 | activityProtocol | standaloneactivity/Properties.scala | feature-specific | settlement table (Boundaries); the canceled-answer law would need an Attempted capability |
| `nonRetryableFails`, `retryCompletes` | Property ×2 | activityProtocol | standaloneactivity/Properties.scala | feature-specific | retry: no Retryable capability in the pilot |
| `scheduleToStartFires`, `scheduleToCloseFires`, `startToCloseFires` | Property ×3 | activityProtocol | standaloneactivity/Properties.scala | feature-specific | deadline: no Deadlined capability in the pilot |
| `scheduleToStartTimesOut`, `scheduleToCloseTimesOut` | Property ×2 | currentAdmission, staleAdmission | admission/Properties.scala | feature-specific | deadline, as above |
| `staleDeliveryRejected` | Property | heldAdmission | admission/Properties.scala | feature-specific | the held-delivery race's Run claim |
| `committedDespiteLostResponse` | Property | admissionResponseLoss | admission/Properties.scala | feature-specific | response-loss fault |
| `failedCommitKeepsTheMessage` | Property | currentOverQueue, staleOverQueue | compositions/Properties.scala | feature-specific | durable-commit fault over the queue |
| `delivers`, `committedStays` | Property ×2 | matchingQueue and its three derived providers | taskqueue/Properties.scala | feature-specific | the queue entity's own laws (`queueLaws`, fn-112 R20) |
| `storageLossDrops` | Property | lossyMatchingQueue | taskqueue/Properties.scala | feature-specific | storage-loss fault of one provider |
| `terminalIsFinal` | Property | nexusProduct | nexuscaller/Claims.scala | single-capability law (not declared) | `terminalStatesAreFinal`; fn-114 owns the Nexus caller, which declares no capability in this spec |
| `syncSucceeds`, `completionSucceeds`, `completionFails` | Property ×3 | nexusProtocol | nexuscaller/Claims.scala | feature-specific | settlement table |
| `asyncStarts` | Property | nexusProtocol | nexuscaller/Claims.scala | feature-specific | async start, a Nexus-only phase |
| `handlerErrorFails`, `retrySucceeds` | Property ×2 | nexusProtocol | nexuscaller/Claims.scala | feature-specific | retry |
| `scheduleToStartFires`, `startToCloseFires` | Property ×2 | nexusProtocol | nexuscaller/Claims.scala | feature-specific | deadline |
| `repliedByPollingWorker` | Property | nexusCaller | nexuscaller/Claims.scala | feature-specific | cross-entity worker claim (fn-114) |
| `forgedSuccess` | Property | Control.forged | nexuscaller/Control.scala | feature-specific | the deliberately faulty negative control |
| 16 design Properties (`outcomePreserved` … `routedToSuccessor`) and `completionSucceeds`, `completionFails` | Property ×18 | the five `designQueries` designs | closepolicy/Claims.scala | feature-specific | close-policy design claims; the two completions are the baseline's settlement table |
| `outcomePreserved`, `ackOnlyWhenKept` | Property ×2 | retainAndRouteBoundedRetry | closepolicy/Claims.scala | feature-specific | close-policy design claims |
| `outcomePreserved`, `ackOnlyWhenKept`, `noUnnecessaryWait`, `expiresWithNothingOwed`, `expiresWhileOwed`, `lateCompletionIsDropped` | Property ×6 | the three WithDeadline designs | closepolicy/Claims.scala | feature-specific | close-policy design claims (deadline variant) |
| `retainedOutcome`, `ownerAcknowledgment`, `singleOutcome`, `cancelPrincipal` | monitor ×4 | all nine designs | closepolicy/Model.scala | feature-specific | close-policy design claims |
| seven `outcomeReachesOwner` claims and `retainedReachesOwner`, `retainedWaitsWithoutRecovery`, `retainedReachesOwnerBoundedRetry` | progress ×10 | one or two designs each | closepolicy/Claims.scala | feature-specific | close-policy design claims |

Totals: 8 claims are law instances. They are `terminalIsFinal` ×2, `terminalStays`,
`terminalFinality`, `pausedIsNotDispatched`, `notAdmittedWhilePaused`, `terminated` and
`cancelRequestedWhileStarted`. The other 68 are feature-specific: 38 close-policy, 7 settlement, 7
deadline, 4 retry, 5 run, fault, control or Nexus-only (`staleDeliveryRejected`,
`committedDespiteLostResponse`, `failedCommitKeepsTheMessage`, `forgedSuccess`, `asyncStarts`), 3
queue, 2 active count and 2 cross-entity worker.

**Rolled out (fn-122 task 3).** The standalone activity declares its capabilities
(`productCapabilities`, `protocolCapabilities`, `admissionCapabilities`, `overQueueCapabilities`,
`overMatchingCapabilities`). `terminalIsFinal`, `pausedIsNotDispatched`, `terminalStays` and
`notAdmittedWhilePaused` are no longer written in its files. Each is the generated
`<machine>.<law>` Property, and its free `verify` Queries (`terminalHolds`, `pauseHolds`, `*.any.*`)
are retired for the generated twins. The pinned paths and `*.product.pausedIsNotDispatched` read
the generated Property through `claim`. The designs over a queue waive `closedIsRejectedUniformly`
with a reason: their queue member keeps stepping after the record closes.
`model/ir/activity-system.laws.json` lists `activityProduct` and `currentAdmission` as the two
instantiating machines of each law the record brings. `terminated` and its find stay authored:
`activityProtocol.terminateSettles` lowers to a Case of its own, not to theirs byte for byte.

**Second entity (fn-122 task 4).** `model/temporal/features/nexusoperation` declares Closable, Terminable,
Cancelable and Describable on `nexusOperation`. Its rejection is a parameter (`alreadyCompleted`,
FailedPrecondition, where the activity answers NotFound), and it overrides
`closedIsRejectedUniformly` with `closedRejectsOrRepeats`: a repeated request id is answered OK after
close. Every law of the catalog now has two checked-in instantiating machines:
`terminalStatesAreFinal` and `closedIsRejectedUniformly` (activityProduct, currentAdmission,
nexusOperation), `pausedIsNotDispatched` (activityProduct, currentAdmission), `terminateSettles` and
`cancelIsRequested` (activityProtocol, nexusOperation). Its generated finds run live. Its
realization requires `nexusoperation.enableStandalone = true` (`requiredSettings`), which its Cases
carry; preparation refuses an environment that leaves it off, naming the flag.

Open finding of task 4, for the owner:
- The Driver reserves a Nexus handler only through a workflow's or an activity's start, so the
  operation's handler paths (`handlerReply`, `complete`) are modeled and verified but no Case through
  them lowers.

**Where it stands (fn-122 closed, task 6).** The vocabulary and the laws are in
`model/temporal/capabilities` (task 8), the framework keeps only the mechanism, and each adopting
Model folder declares its capabilities in its own `Capabilities.scala`; `model/README.md`
("Capabilities and their laws") documents the surface with the activity's two declarations as its
worked example. The table view (`umpire-lint --tables`) prints each machine's laws after its
per-operation table: what each promises and does not, read from the sidecar, and the modality it
pins on the cells of its capabilities' actions (the sidecar records each action field's class,
`Pollable.dispatch: attemptStart`): `pausedIsNotDispatched` is MUST NOT on the `paused` cells of
`attemptStart`, `control-pause` and `control-unpause`, and Closable's laws, whose capability names no
action, MUST NOT on the terminal cells of every class. It lists the cells of those actions no law
pins, and marks the product's laws on the protocol `inherited, unchecked`, since no Query over the
protocol's Scenarios asks them. Claims across `model/ir`, authored/generated, from fn-112's close
to fn-122's: Properties 172/0 to 155/26, Queries 264/0 to 248/26 (activity 11/0 to 9/5 and 12/0
to 10/5, activity-system 39/0 to 24/17 and 84/0 to 70/17, the new nexus-operation 0/4 and 0/4, the
Nexus caller, close and control files unchanged). Pausable on the fn-119 workflow (task 7, R10) is
deferred until fn-119's example exists; until then the admission record is the second machine of
`pausedIsNotDispatched`.

**For the owner.** No claim is unclassified. These are the borderline calls, each decided above:

- `terminated` and `cancelRequestedWhileStarted` fix the phase as well as the recorded fact, so they
  are stronger than `terminateSettles` and `cancelIsRequested`. R5 keeps them until a generated
  Case equals theirs byte for byte.
- `terminalFinality` is the Close law in monitor form. It is weaker than `terminalStays` on the
  same machines. As a Run monitor it is not a Property the catalog can generate.
- `requestedButUnreceived` (closepolicy) looks like `cancelIsRequested`. It tracks intent in the
  state rather than as a fact, and its point is the receipt gap.
- `lateCompletionIsDropped` resembles `closedIsRejectedUniformly`. It covers one action and records a
  fact, so it is not a rejecting stutter.

**Findings of task 1** (the early proof point):

- `terminalStatesAreFinal` and `pausedIsNotDispatched` lift through the existing fold. Their two
  instances each, product and record, carry the frozen names, IDs and answers.
- The bodies of `closedIsRejectedUniformly`, `terminateSettles` and `cancelIsRequested` lift only
  once the fold binds value arguments, not just function-valued and integer ones: an outcome, an
  action class and a fact. A throwaway probe lifted each one on the activity. The first was refused
  at the outcome `Outcome.notFound`, the other two at the class `control(Control.terminate)`. That
  binding belongs with task 2's capability fields (R2, R3). The approach stands: these laws are
  ordinary defs, and the refusal is the missing binding, not the law's shape.
- Close, Poll and Describe bring no Temporal law of their own:
  - Close's laws are entity-neutral (`umpire/laws`).
  - Pollable alone has no law with two instantiating machines: `atMostOneActive` and
    `startedByPollingWorker` have one each.
  - Describable's field is the realization's status table, which no Property of a machine reads.
    Its law is the await that generated Cases derive from it (task 3). A Describe law stated over
    a describe action ("a describe changes nothing") would need a transition Property with `when`.
    That is a finding for a reader spec, not written here.
- The law's `terminal: P => Boolean` takes the status, so `Product.terminal` and
  `Admission.terminal` now take the phase, as `Protocol.terminal` did. `Product.phase` and
  `Product.ends` are new, and the `OverQueue`/`OverMatching.terminal` forwarders are gone. The IR's
  function bodies change shape (calls of `phase` where field reads were) and mean the same: tables,
  Definition IDs, Property rows, answers and Case bytes are the baseline's. The only golden-config
  entries are two `source_path_merges` for the law files' positions. No function name changed, since
  a claim's function is named `<machine>.property.<name>`.
- The framework's TASTy was not lifted at all. The lifter now reads `umpire/laws` beside the
  Models, so an entity-neutral law folds like a Model's own def. A def's body takes the name of
  the val that declares its call, so a law's instance is named by its val
  (`val terminalStays = terminalStatesAreFinal(m)(…)`).

**Realization.** The activity realization's four `controlBinding` performs and five `awaitStatus`
items (about 85 of 860 lines) become one `Describable` table plus the kit's `perform(control(c) ->
rpc(…))` lines fn-112.9 already plans; the protocol spec adds the derived await, not the performs.

**Free for a new entity** declaring Closable + Terminable + Pausable + Dispatchable + Deadlined +
Describable, with a realization that performs its controls: the verify Queries above, and the find
Queries `terminateSettles`, `pauseThenUnpause`, one `deadlineFires` per timer, each a Case whose await
is derived from `Describable` (and from fn-118's visibility hint once it lands).

**Risks.** Over-abstraction: a capability with one instance is a feature claim with a longer name;
the catalog admits a law only when two entities instantiate it (the rule fn-118 uses for hints).
Wrong universals: `recordsReasonAndIdentity` is already false for the schedule; the opt-out with a
reason is how a wrong universal surfaces instead of hiding, and each law cites the server code it rests
on. Semantic drift between entities: parameters (`rejection`, pause-of-held-work) are where the
server differs on purpose, and a lint kind "law instantiated with a parameter no server citation
backs" keeps them honest. Definition IDs: a Property's ID is family + name, so two machines of one
family must not both instantiate one law under one name; generated names are `<machine>.<law>`,
which also keeps fn-112 R1's frozen IDs untouched (generated claims are additions, the existing ones
are retired by name in a later task). Reader cost: free-Scenario verifies at depth ≥ 5 are the
expensive Queries today; a law's default limit is the machine's declared `twelve`-style bound and
`Query.total` is checked as for any Query.

## 4. Sequencing

**Must wait.** Capabilities use `in`, `implies`, `records` (fn-112 R5), name capture and `given Family`
(R7), the refinement default (R8), typed compositions and `after.records(_.member, fact)` (R3), the
script helpers (R13) and the queue entity (R20); the Nexus Models must be on those forms (fn-114 R6)
before a law can be read on both entities. Generated-claim lint and the accepted-findings file are
fn-120 R5/R7. Derived awaits in generated Cases wait for fn-118's Describe visibility hint.

**What fn-112 should do now** (small, and none changes behavior):

1. R4: write `terminalStays`, `notAdmittedWhilePaused`, `atMostOneActive` as top-level defs whose
   state-dependent parts are *parameters* (`terminal: S => Boolean`, `paused`, `running`), not
   closures over `AdmissionState`; put them beside, not inside, `admission/`. These are the first law
   bodies.
2. R10: expose the status sets as named defs on each vocabulary object (`Product.terminal`,
   `Product.paused`, `Product.running`, `Protocol.held`) rather than inlining `in(…)` at use sites.
3. Lifter (task .3/.4, while the fold is open): bind a function-typed argument that references a
   lifted top-level def into the fold env, and resolve `callee` through the env; one lifting fixture,
   one refusal (a lambda literal argument). Without this a law cannot take `terminal`.
4. R7: do not make "a Property is found by its `val`" a rule of the lifter; a generated claim has no
   `val`. The R1 harness must treat an *added* Property or Query as an allowed delta category once
   this spec starts, never as a waived fingerprint.
5. R13/fn-118 helper seam: the `awaitStatus(fact, enumValue)` pairs become one declared table
   (`Describable.σ`) consulted by the helper, not a pair per call site.
6. R20: declare the queue's own shared properties through a `def queueLaws(m)` taking the contract
   machine, the shape the catalog later generalizes.

---

## Proposed spec: Capabilities and their laws (pilot: Terminate and Pause)

**Goal.** A Temporal entity declares the capabilities it has, binds each to its own vocabulary, and
receives the laws of each capability and of each pair of capabilities as lifted Properties and
Queries, without listing them. The pilot instantiates Close, Terminate, Pause, Dispatch and Describe
on the standalone activity (product, protocol, admission designs, both composition families) and
Close, Terminate and Cancel on a new minimal Nexus operation Model grounded in
`chasm/lib/nexusoperation`; Pause is instantiated on the fn-119 workflow when it lands.

**Requirements.**

- R1 `temporal/laws` holds one top-level def per law with its server citation, its `promises` and
  `doesNotPromise` text, and the parameters it takes; `umpire/laws` holds the entity-neutral ones
  (`terminalStatesAreFinal`, `closedIsRejectedUniformly`, `recordedStays`). A law's body uses only
  constructs SEMANTICS.md defines and no transition Property with `when`.
- R2 `umpire` provides `capabilities(m)(…)`, the capability types `Closable`, `Terminable`,
  `Pausable`, `Cancelable`, `Dispatchable`, `Deadlined`, `Describable` with typed fields, and `except`
  and `overriding`, each requiring a reason. The lifter lifts the declaration into Properties and
  Queries named `<machine>.<law>` with an authored or computed `total`, refuses a capability whose
  action the machine does not bind, a predicate of another state type, an `except` of a law the
  catalog does not bring, and an `overriding` whose def has another signature, each at its line.
- R3 The lifter binds function-valued arguments that reference lifted top-level defs; fixtures for
  the lift and the refusal.
- R4 The catalog, in the lifter, maps each capability and each unordered pair to its laws; a law
  enters the catalog only with two instantiating entities. Pairs are found from the declared set.
- R5 The activity Models declare their capabilities; `terminalIsFinal`, `pausedIsNotDispatched`,
  `terminalStays`, `notAdmittedWhilePaused`, `atMostOneActive`, `terminated`, `startedByPollingWorker`
  and their Queries are retired by name and replaced by generated ones with the same answers;
  Case bytes of existing lowered Queries are unchanged, and generated find Queries lower to Cases
  whose awaits come from `Describable`.
- R6 A Nexus operation Model (`temporal/nexusoperation`: start, describe, request-cancel, terminate,
  handler completion; no retry, no deadline) declares Closable, Terminable, Cancelable and Describable,
  passes the gate, and its generated `terminateSettles` and `cancelIsRequested` lower to Cases that
  run live. `rejection` differs from the activity's and is a parameter, not an override.
- R7 Opt-outs and overrides are written into the accepted-findings file with their reasons; lint
  reports a law waived with no reason, a reason naming no law, a capability parameter with no
  citation, and a law with one instance.
- R8 `model/README.md` explains capabilities, and each law's `promises`/`doesNotPromise` appears in the
  generated table view beside its Properties.
- R9 Gate, `make lint-model`, Go tests and `make lint-code-fast` pass; the done summary counts
  Properties and Queries authored vs generated before and after, per Model.

**Tasks.** (1) Catalog and law bodies as plain defs, with citations, and the fn-112 R4 defs moved
under them; (2) lifter: function-argument binding + fixtures; (3) `capabilities` declaration, lifting,
`except`/`overriding`, fixtures; (4) activity rollout and retirement by name, R5 harness; (5) Nexus
operation Model, realization, live Cases; (6) lint kinds and accepted-findings integration; (7) docs,
counts, closing gates; (8, gated on fn-119) Pausable on the workflow example.

**Boundaries.** No IR schema change. No capability as a machine; no second refinement per machine.
No law on Reset, Poll-result, Update or Signal. No change to the Go reader's Property semantics; a law
that needs a transition Property with `when` is a finding for a reader spec, not a workaround here.
No change to existing Case bytes. No law with a single instantiating entity.

**Decision context.** Laws are defs, not expression templates in the lifter, so a law is reviewable as
Scala and lifts through the same fold as every step function; the lifter owns only the catalog table.
Pairs are found, not listed, because the owner's point is that `Pausable × Pollable` must apply
without an author remembering it. Opt-out needs a reason because a wrong universal is the main risk
and the schedule already shows one.
