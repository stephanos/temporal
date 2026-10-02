# Specimen: Nexus caller close and reset

This specimen asks whether authored promises and assumptions expose a reviewed design flaw before
the feature ships. The flaw involves three things:

- irreversible handler effects;
- retained knowledge of the outcome;
- ownership transfer to a reset successor.

The flaw comes from the design review the spec links. It is design evidence, not a claim about the
current server. The author has to declare outcome preservation and the handler's reporting
assumptions for a checker to find it. This specimen does neither of the following:

- resume the deferred Nexus cancellation implementation;
- implement a CallerClosePolicy.

See [README.md](README.md) for what "supported" and "unsupported" mean here and for the scratch
check behind the results quoted below.

## Boundaries

**Product.** The product is an authored design promise over one logical operation, not an existing
product machine. It makes two promises:

1. The operation's single outcome reaches whichever run owns the operation.
2. An acknowledgment to the handler implies the outcome is kept: in the owner's history, or retained
   at the operation with an outstanding delivery obligation to the current or future owner.
   Reset must preserve that obligation until the successor commits the outcome.

`nexusProduct` and `nexusProtocol` (`../../scala/temporal/nexuscaller/Model.scala:97`, `:129`)
model neither close nor reset. They stay unchanged, and the designs below do not refine them. Task 5
compares only behavior both represent: an open caller accepting an asynchronous completion, which
is the existing `complete` rows.

**System.** The system has these participants:

- the caller's runs, the original and one reset successor (`Caller`);
- small workflow close and reset contracts (`callerClose`, `reset`);
- one handler whose outcome is an irreversible effect (`Handler.done`);
- the completion channel from handler to caller, with capacity 1 (`Completion`);
- durable retention at the operation (`Retained`).

The design keeps apart six things the spec asks to separate:

| Concept | Sketch field or action |
| --- | --- |
| Cancel intent | `cancel` |
| Handler effect | `handlerFinish` |
| Reported outcome | `Completion.inFlight` |
| Delivery answer | `Answer` |
| Durable retention | `Retained` |
| Caller knowledge | `Knowledge` |

**Identities.** The logical operation and its request identity must stay stable across distinct
run identities. The existing `operation` entity is keyed by `scheduledEvent` (`Model.scala:36`),
which belongs to one run's history. That key cannot follow the operation into a reset successor
(finding F9). The supported sketch has one operation, so it needs no key. The proposed layer
declares a stable key and a run role (E8).

`Retained.pending(r)` represents a durable outcome together with its outstanding delivery
obligation. Bare stored outcome bytes are insufficient. `resetStep` discharges this encoded
obligation only in the same atomic step that records `Knowledge.successor(r)`. An acknowledgment
that stores an outcome but drops its transferable obligation violates the provider contract.

**Close and reset.**

- Closing a workflow freezes its history: `Knowledge.original` never changes once the caller is
  `closed`. Detached handler work continues after close.
- Reset builds the successor from a point after the start and before the close or cancellation. It
  transfers ownership and reconstructs retained evidence, and it keeps the cancel intent.
- Reset cannot undo a handler effect.

## Domain manifest

| Dimension | Bound in the supported sketch | Notes |
| --- | --- | --- |
| Logical operations | 1 | |
| Caller runs | the original, and at most one reset successor | `Caller.open`, `closed`, `resetOpen` |
| Handlers | 1, finishing at most once | |
| Cancellations | at most 1, only while the handler runs and the caller is not closed | The intent survives reset |
| Outcomes | succeeded, failed, canceled | Kernel `Resolution`. Canceled needs a cancel intent |
| Completion channel | capacity 1, the message addressed to the operation | |
| Delivery answers | accepted, retained, rejectedTransient, rejectedPermanent | |
| Faults | transient rejection; lost acknowledgment (the report arrives again) | Nondeterministic alternatives |
| Fault budget in the supported trace scope | at most 4 or 6 fault choices, from the corresponding step ceiling | No independent fault counter; E2's proposed bounded-retry channel is a tighter future scope |
| Timeouts | none | The proposed layer adds a schedule-to-close deadline (N8) |
| Ends (`settled`) | the handler is running; or the owner knows the outcome; or a closed run's operation retained it | A reachable non-end state with no step is a lost outcome |
| Catalog / reachable | 2,688 states; 61, 76 and 71 reachable in the three designs | |
| Limits | `four` pinned; `six` free | |

## Sketch: supported today

The three designs differ only in the `Policy` literal their delivery step passes. Changing the
policy is a one-line edit to feature Scala. This block compiled, answered and lifted in the scratch
check.

```scala
package specimens.nexus

import temporal.nexuscaller.{Resolution, caller, complete, handler, operation, workflow}
import temporal.nexuscaller.given
import umpire.*

val Family: umpire.Family = umpire.Family("temporal.nexus.caller.closereset")

// ### Design vocabulary: one logical operation, the original run and one reset successor

/** The caller's runs. `closed` is the original run closed with its history frozen; `resetOpen` is the
  * successor, which owns the operation from the reset on. */
enum Caller derives Finite:
  case open, closed, resetOpen

/** What the handler has done. `done` is an irreversible effect: no reset undoes it. */
enum Handler derives Finite:
  case running
  case done(result: Resolution)

/** The one completion report in the bounded channel from handler to caller. While it is in flight
  * the handler still owes the report. */
enum Completion derives Finite:
  case none
  case inFlight(result: Resolution)

/** An outcome retained durably at the operation, keyed by the stable operation identity rather than a run. */
enum Retained derives Finite:
  case none
  case pending(result: Resolution)

/** Which run's history records the outcome. */
enum Knowledge derives Finite:
  case none
  case original(result: Resolution)
  case successor(result: Resolution)

final case class CloseResetState(
    caller: Caller,
    cancel: Boolean,
    handler: Handler,
    channel: Completion,
    retained: Retained,
    known: Knowledge,
) derives Finite

/** What a delivery attempt reports back to the handler. */
enum Answer derives Finite:
  case accepted, retained, rejectedTransient, rejectedPermanent

/** A close/reset design records nothing a Run reads yet: its evidence is task 5's to declare. */
type CloseResetStep = Step[CloseResetState, Answer, Nothing]

/** Where a completion goes once the original run can no longer take it. */
enum Policy derives Finite:
  /** Faulty: a closed run rejects the completion permanently, and the handler stops reporting. */
  case rejectAfterClose
  /** Faulty: after a reset the original run acknowledges the completion; the successor never learns it. */
  case ackByOriginal
  /** Corrected: retain at the operation and route to the current owner. */
  case retainAndRoute

val callerClose = action("callerClose", caller) on workflow
val reset = action("reset", caller) on workflow
val requestCancel = action("requestCancel", caller) on operation
val handlerFinish = action("handlerFinish", handler).on(operation).input[Resolution]("result")

val opened: CloseResetState = CloseResetState(Caller.open, false, Handler.running, Completion.none, Retained.none, Knowledge.none)

// ### Step functions

def closeStep(s: CloseResetState): List[CloseResetStep] =
  if s.caller != Caller.open then Nil else List(Step(Answer.accepted, s.copy(caller = Caller.closed)))

def cancelStep(s: CloseResetState): List[CloseResetStep] =
  if s.cancel || s.caller == Caller.closed || s.handler != Handler.running then Nil
  else List(Step(Answer.accepted, s.copy(cancel = true)))

/** The handler's irreversible effect, and its first report. A canceled result needs a cancel intent. */
def finishStep(s: CloseResetState, r: Resolution): List[CloseResetStep] =
  if s.handler != Handler.running then Nil
  else if r == Resolution.canceled && !s.cancel then Nil
  else List(Step(Answer.accepted, s.copy(handler = Handler.done(r), channel = Completion.inFlight(r))))

/** The owner commits the outcome. Transient rejection keeps the report in flight; a lost
  * acknowledgment commits and keeps it in flight too, so the report arrives again. */
def committed(s: CloseResetState, k: Knowledge): List[CloseResetStep] = List(
  Step(Answer.accepted, s.copy(known = k, channel = Completion.none)),
  Step(Answer.rejectedTransient, s),
  Step(Answer.accepted, s.copy(known = k), Nil, "the acknowledgment is lost"),
)

def keptAtOperation(s: CloseResetState, r: Resolution): List[CloseResetStep] = List(
  Step(Answer.retained, s.copy(retained = Retained.pending(r), channel = Completion.none)),
  Step(Answer.rejectedTransient, s),
  Step(Answer.retained, s.copy(retained = Retained.pending(r)), Nil, "the acknowledgment is lost"),
)

def deliverStep(p: Policy, s: CloseResetState, r: Resolution): List[CloseResetStep] =
  if s.channel != Completion.inFlight(r) then Nil
  else s.caller match
    case Caller.open => committed(s, Knowledge.original(r))
    case Caller.closed =>
      if p == Policy.rejectAfterClose then
        List(Step(Answer.rejectedPermanent, s.copy(channel = Completion.none)), Step(Answer.rejectedTransient, s))
      else keptAtOperation(s, r)
    case Caller.resetOpen =>
      if p == Policy.ackByOriginal then
        List(Step(Answer.accepted, s.copy(channel = Completion.none), Nil, "the original run acknowledges it"))
      else committed(s, Knowledge.successor(r))

/** Reset builds the successor from a point after the start and before the close. It transfers
  * ownership, reapplies the outcome the original run recorded or the operation retained, and keeps
  * the cancel intent. It cannot undo the handler's effect. */
def resetStep(s: CloseResetState): List[CloseResetStep] =
  if s.caller == Caller.resetOpen then Nil
  else List(Step(Answer.accepted, s.copy(caller = Caller.resetOpen, known = reapplied(s), retained = Retained.none)))

def reapplied(s: CloseResetState): Knowledge = s.known match
  case Knowledge.original(r) => Knowledge.successor(r)
  case _ =>
    s.retained match
      case Retained.pending(r) => Knowledge.successor(r)
      case Retained.none    => Knowledge.none

/** Where a path may end: the handler is still working, or its outcome reached the owner, or the
  * operation retained it for a successor a closed run may never get. Any other state with no step is
  * a lost outcome. */
def settled(s: CloseResetState): Boolean = s.handler match
  case Handler.running => true
  case Handler.done(r) => ownerKnows(s, r) || (s.caller == Caller.closed && s.retained == Retained.pending(r))

// ### The three designs differ only in the policy their delivery step passes

val rejectAfterCloseDesign: Machine[CloseResetState, Answer, Nothing] =
  machine[CloseResetState, Answer, Nothing](Family, "rejectAfterClose") {
    forEntity(operation)
    starts(opened)
    ends(settled)
    steps(callerClose ~> closeStep, reset ~> resetStep, requestCancel ~> cancelStep, handlerFinish ~> finishStep,
      complete ~> ((s, r) => deliverStep(Policy.rejectAfterClose, s, r)))
  }

val ackByOriginalDesign: Machine[CloseResetState, Answer, Nothing] =
  machine[CloseResetState, Answer, Nothing](Family, "ackByOriginal") {
    forEntity(operation)
    starts(opened)
    ends(settled)
    steps(callerClose ~> closeStep, reset ~> resetStep, requestCancel ~> cancelStep, handlerFinish ~> finishStep,
      complete ~> ((s, r) => deliverStep(Policy.ackByOriginal, s, r)))
  }

val retainAndRouteDesign: Machine[CloseResetState, Answer, Nothing] =
  machine[CloseResetState, Answer, Nothing](Family, "retainAndRoute") {
    forEntity(operation)
    starts(opened)
    ends(settled)
    steps(callerClose ~> closeStep, reset ~> resetStep, requestCancel ~> cancelStep, handlerFinish ~> finishStep,
      complete ~> ((s, r) => deliverStep(Policy.retainAndRoute, s, r)))
  }

// ### Promises

/** The run that owns the operation records this outcome. */
def ownerKnows(s: CloseResetState, r: Resolution): Boolean =
  if s.caller == Caller.resetOpen then s.known == Knowledge.successor(r) else s.known == Knowledge.original(r)

/** A decided outcome is always somewhere a current or future owner can learn it: its history, the
  * operation's retention, or a report still in flight. */
def outcomePreserved(after: CloseResetStep): Boolean = after.state.handler match
  case Handler.running => true
  case Handler.done(r) =>
    ownerKnows(after.state, r) || after.state.retained == Retained.pending(r) || after.state.channel == Completion.inFlight(r)

/** An acknowledgment ends the handler's report only once the owner committed or the operation retained the outcome. */
def ackOnlyWhenKept(before: CloseResetState, after: CloseResetStep): Boolean = before.channel match
  case Completion.inFlight(r) =>
    (after.outcome != Answer.accepted && after.outcome != Answer.retained) || after.state.channel != Completion.none ||
      ownerKnows(after.state, r) || after.state.retained == Retained.pending(r)
  case Completion.none => true

val four: Limits = Limits("four", steps = 4, actions = 4, search = 1 << 20)
val six: Limits = Limits("six", steps = 6, actions = 6, search = 1 << 22)

def closeResetQueries(m: Machine[CloseResetState, Answer, Nothing]): Vector[Query] =
  val preserved = m.property("outcomePreserved") holds outcomePreserved
  val acked = m.property("ackOnlyWhenKept") holdsAcross ackOnlyWhenKept
  val closedThenFinished = m.scenario("closedThenFinished").starts(opened)
    .actions(callerClose, handlerFinish(Resolution.succeeded), complete(Resolution.succeeded), reset)
  val resetThenDelivered = m.scenario("resetThenDelivered").starts(opened)
    .actions(handlerFinish(Resolution.failed), reset, complete(Resolution.failed))
  val canceledAcrossReset = m.scenario("canceledAcrossReset").starts(opened)
    .actions(requestCancel, handlerFinish(Resolution.canceled), reset, complete(Resolution.canceled))
  val ackedThenReset = m.scenario("ackedThenReset").starts(opened)
    .actions(handlerFinish(Resolution.succeeded), complete(Resolution.succeeded), reset)
  val any = m.scenario("any").starts(opened).free
  Vector(
    query(s"${m.name}.closedThenFinished") verify preserved in closedThenFinished limits four,
    query(s"${m.name}.resetThenDelivered.ackOnlyWhenKept") verify acked in resetThenDelivered limits four,
    query(s"${m.name}.resetThenDelivered.outcomePreserved") verify preserved in resetThenDelivered limits four,
    query(s"${m.name}.canceledAcrossReset") verify preserved in canceledAcrossReset limits four,
    query(s"${m.name}.ackedThenReset") verify preserved in ackedThenReset limits four,
    query(s"${m.name}.any.outcomePreserved") verify preserved in any limits six,
    query(s"${m.name}.any.ackOnlyWhenKept") verify acked in any limits six,
  )

val rejectAfterCloseQueries: Vector[Query] = closeResetQueries(rejectAfterCloseDesign)
val ackByOriginalQueries: Vector[Query] = closeResetQueries(ackByOriginalDesign)
val retainAndRouteQueries: Vector[Query] = closeResetQueries(retainAndRouteDesign)
```

## Sketch: proposed extensions

```scala
// UNSUPPORTED: nothing below exists in model/scala/umpire, and this block does not compile. Each
// declaration is an extension from README.md#proposed-extensions.

// E8: identity roles. The logical operation and its request identity span runs; the run is a role.
val closeResetIdentity = identity(operation, stable = "requestId", runs = Run.roles("original", "successor"))

// E2: the completion channel with its declared delivery semantics. Go derives transient rejection,
// permanent rejection and lost acknowledgment from it, instead of the hand-written alternatives in
// `committed`, `keptAtOperation` and `deliverStep`.
val completions = channel[Completion]("completion", capacity = 1, order = Order.fifo,
  loss = Loss.none, duplicates = Duplicates.untilAck(max = 2),
  rejections = Rejections(transient = Retry.bounded(1), permanent = StopsReporting))

// E3: operation-level retention as a persistence provider with a durable guarantee.
val retention = provider("operationRetention") {
  val keep = event[Resolution]("keep", results = List("committed"))
  val transfer = event[Run]("transfer")
  allows(noLossOfCommitted)
}

// E1: the two promises as passive monitors, and the single-outcome monitor.
val preservedOutcome = monitor[Retained]("outcomePreserved", initial = Retained.none) {
  on(handlerFinish)((_, r) => Retained.pending(r))
  violated(s => !outcomePreserved(s))
  evaluate(atEveryState = true)
}
val singleOutcome = monitor[Retained]("singleOutcome", initial = Retained.none) {
  onKnowledge((m, r) => if m == Retained.none then Retained.pending(r) else m)
  violated((m, r) => m != Retained.none && m != Retained.pending(r))
}

// E5: conditional progress. A finite prefix that ends open is not a counterexample, and a finite
// timeout turns an indefinite wait into a lost-outcome or unnecessary-wait result.
val reporting = assume("handlerReportsUntilAckOrPermanent")
val retryFair = assume("transientRejectionEventuallyAccepted")
val retentionDurable = assume("retentionSurvivesCrash")
val recovery = assume("currentOwnerEventuallyRecoversAndReappliesRetainedOutcome")
val deliveryFair = assume("enabledDeliveryAndRecoveryActionsEventuallyRun")
val deadline = assume("scheduleToCloseExpires")
val delivered = leadsTo("outcomeReachesOwner", from = isDone, to = settled, within = 6,
  under = List(reporting, retryFair, recovery, deliveryFair, deadline, retentionDurable))

// Cancellation principal and lifetime across reset, which the supported sketch reduces to one Boolean.
val principal = survivesReset(cancelIntent, keys = List("principal", "requestId"))
```

## Trace oracles

Search oracles apply to the named design and limits. "Scratch" means the existing Scala search
produced the witness, with the given explored product-state count. Task 5 must reproduce each
oracle through the generic Go semantics.

State keys spell six fields in order: caller, cancel, handler, channel, retained, known. For
example, `closed-false-done-succeeded-none-none-none` is a closed original run, no cancel intent, a
handler that finished succeeded, nothing in flight, nothing retained, and no run that knows the
outcome.

### N1: permanent rejection after close, then reset (pinned control 1)

Design: `rejectAfterClose`. Scenario: `closedThenFinished`.

| # | Row → answer state | Public observation | Design-only |
| --- | --- | --- | --- |
| 1 | `…-callerClose` → accepted `closed-false-running-none-none-none` | The workflow closes; its history is frozen | |
| 2 | `…-handlerFinish-succeeded` → accepted `closed-false-done-succeeded-inFlight-succeeded-none-none` | none | The handler's effect is irreversible |
| 3 | `…-complete-succeeded` → rejectedPermanent `closed-false-done-succeeded-none-none-none` | The completion request fails permanently | The handler stops reporting |
| 4 | `…-reset` → accepted `resetOpen-false-done-succeeded-none-none-none` | The successor's history shows the operation still open | Nothing to reapply |

The evaluation point is after step 3:

- `outcomePreserved` is violated. Scratch: `counterexample-found`, 7 states. The search stops
  there, so step 4 is read from the table.
- After step 4 the state is stuck: it is not `settled`, and no step leaves it. Scala `check` and
  goir both report `resetOpen-false-done-succeeded-none-none-none`. That is the deadlock witness for
  the lost outcome. With a modeled deadline (N8) it becomes a timeout with a lost-outcome
  assessment instead of a hang.
- The free search (`six`) returns the same shape, with failed as the outcome (61 states).
- `ackOnlyWhenKept` holds in this design (61 states), because a permanent rejection is not an
  acknowledgment. The two controls are caught by different promises. That is the Property alone:
  in the lifted fixture the design also names the `retainedOutcome` monitor, which watches the same
  search and is violated where the outcome is lost, so the Query `rejectAfterClose.any.ackOnlyWhenKept`
  is a counterexample that names that monitor, while the Property fails on no step.

**N1′, the corrected design.** Design `retainAndRoute`, same schedule:

- Step 3 is answered `retained`, reaching `closed-false-done-succeeded-none-pending-succeeded-none`.
- Step 4 reapplies the outcome, reaching `resetOpen-false-done-succeeded-none-none-successor-succeeded`.
- `verified-within-limits`, 9 states.

**N1 with a transient rejection.** Take the transient alternative at step 3 instead. The report
stays in flight and `outcomePreserved` holds. Transient and permanent rejection are distinct
assessments.

### N2: acknowledgment by the original run after reset (pinned control 2)

Design: `ackByOriginal`. Scenario: `resetThenDelivered`.

| # | Row → answer state | Public observation | Design-only |
| --- | --- | --- | --- |
| 1 | `…-handlerFinish-failed` → accepted `open-false-done-failed-inFlight-failed-none-none` | none | The report is in flight |
| 2 | `…-reset` → accepted `resetOpen-false-done-failed-inFlight-failed-none-none` | The successor exists and owns the operation | |
| 3 | `…-complete-failed` → accepted `resetOpen-false-done-failed-none-none-none` | The completion request succeeds | The original run acknowledged it; the successor stays uninformed |

The evaluation point is after step 3:

- `ackOnlyWhenKept` is violated, and so is `outcomePreserved`. Scratch: 4 states each.
- The state after step 3 has the lost-outcome shape of N1, with `failed` in its key rather than
  `succeeded`.
- The free search returns these three steps for both promises (76 states).

**N2′, the corrected design.** Step 3 routes the completion to the successor:
`resetOpen-false-done-failed-none-none-successor-failed`. `verified-within-limits`, 6 states.

### N3: canceled outcome across reset

Scenario: `canceledAcrossReset`, which is requestCancel, `handlerFinish-canceled`, reset, then
`complete-canceled`.

- The cancel intent (`true`) survives the reset in every design.
- `ackByOriginal` is violated at step 4 (scratch: 5 states).
- `rejectAfterClose` and `retainAndRoute` are verified (7 states).

Together, N1 to N3 cover all three outcomes: succeeded, failed and canceled.

### N4: reset after acknowledgment

Scenario: `ackedThenReset`, which is `handlerFinish-succeeded`, then `complete-succeeded` (accepted;
the original run knows the outcome), then reset. The reset reapplies the outcome to the successor.
All three designs are verified (8 states).

A reset that truncates to a point before the completion and does not reapply the outcome would
violate `outcomePreserved` at the reset step. That mutation is hand-reviewed only; the sketch does
not include it.

### N5: lost acknowledgment and duplicate completion

The sequence is:

1. `handlerFinish`;
2. `complete`, lost-acknowledgment alternative: the owner knows the outcome and the report stays in
   flight;
3. `complete` again, accepted.

The second delivery changes no knowledge. No run ever knows two outcomes, because the channel
carries the handler's one result. Both promises are covered by the free search, which
`retainAndRoute` verifies within six steps (71 states).

### N6: reset before and after retention

These are two different paths, and receipts must keep them apart:

| Order | Path | Oracle |
| --- | --- | --- |
| Reset before retention | The report is in flight at the reset and is routed to the successor | N2′ |
| Reset after retention | Retained at close, then reapplied by the reset | N1′ |

### Hand-reviewed oracles that need the proposed layer

**N7: principal loss.** A mutation of reset drops the cancel intent or its principal. The successor
then owns a canceled outcome it never requested. That must be a distinct assessment:
"cancellation principal lost across reset", not an outcome-preservation failure.

**N8: timeout resolution.** With `scheduleToCloseExpires` assumed, N1's and N2's stuck state gains a
timeout step. The result is a lost-outcome or unnecessary-wait assessment. It is not a liveness
counterexample unless a separate progress violation is established.

**N9: fair non-progress.** This hand-reviewed progress control selects a retry-until-ack
channel variant, whose persistent rejection fault is counted once. Transient rejection forever is
a self-loop of `complete` with `rejectedTransient`. The bounded-retry channel proposed above
instead stops at its retry ceiling and reports retry exhaustion or deadlock; a receipt must name
which variant it checks.

- Under delivery-action fairness but without `transientRejectionEventuallyAccepted`, it is a
  fair non-progress cycle for
  `outcomeReachesOwner`.
- With the assumption, it is excluded.
- Today's search treats the loop as a revisited state and reports nothing. Progress checking belongs
  to task 3.

**N10: deadlock versus open prefix.** A bounded prefix that ends with the handler still running is
not a counterexample. Only the stuck states of N1 and N2, or N9's cycle, are progress witnesses.

**N11: unavailable recovery.** Retain an outcome durably, crash the current owner, and keep
recovery unavailable. Retention safety still holds; owner knowledge remains unresolved. An
operational recovery failure is reported separately, and progress is conditional on
`currentOwnerEventuallyRecoversAndReappliesRetainedOutcome`. It becomes `inconclusive` when that
assumption lacks support. Under supported recovery and delivery fairness, recovery must reapply
the retained outcome within the declared modeled deadline; missing that deadline is a progress
violation. This differs from exhausted handler retries, loss of retained storage, and a scheduler
that never runs enabled recovery. None may be renamed as another assumption.

## Evidence and capability matrix

Caller close and reset stay design checks unless the implementation exposes the needed behavior and
commitments (spec, Edge Cases). The runtime seam this prototype demonstrates is the current
synchronous and asynchronous completion.

| Fact or observation | Kind | Source today | Testpilot today | Task |
| --- | --- | --- | --- | --- |
| Handler reply and asynchronous completion | public | Handler reply; completion through a handle slot | `NexusHandlerReply`, `NexusOperationCompletion` (`../../scala/temporal/nexuscaller/Realization.scala:307-327`) | 9 (existing) |
| Caller knowledge: NexusOperationCompleted, Failed or Canceled | public | Caller history (`Realization.scala:68-132`) | `GetWorkflowExecutionHistory` evidence | existing |
| Caller close | public | The workflow finishes | `Finish` instruction | existing |
| Reset | public | ResetWorkflowExecution | Admissible through `InvokeRPC`; no Case uses it | 6 |
| Successor knowledge | public | The successor run's history | History evidence; needs run correlation (E8) | 6, 7 |
| Permanent versus transient rejection | public to the handler | The completion's delivery status | The outcome of `NexusOperationCompletion`; not checked whether its typed result separates the two | 6, 11 |
| Operation-level retention and the delivery obligation | internal | None: the server has no such feature, and it is design only | none | design check only |
| Handler effect (irreversible) | internal to the handler | The handler script | Handler entrypoint | existing |
