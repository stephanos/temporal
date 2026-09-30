# Specimen: standalone activity admission and its dispatch queue

This specimen asks whether an author can state an admission promise, have the checker find a race
that breaks it, and then replace an opaque durable queue with a detailed one without changing the
promise.

The race control is a deliberately faulty design, not a claim about a known server defect. It goes
like this:

1. The activity starts scheduled, with no attempt admitted.
2. A dispatch is enqueued while the activity is eligible.
3. The activity is paused.
4. The old message is delivered, and admission accepts a new attempt without checking current
   eligibility. No unpause happens between these steps.

The corrected design checks eligibility at authoritative admission. Work admitted before the pause
is a separate case, and it is legal.

See [README.md](README.md) for what "supported" and "unsupported" mean here and for the scratch
check behind the results quoted below.

## Boundaries

**Product.** `activityProduct` (`../../scala/temporal/standaloneactivity/Model.scala:182`), as it
is: what `DescribeActivityExecution` and `PollActivityExecution` show a caller. Its Property
`pausedIsNotDispatched` (`Claims.scala:21-24`) is the product promise the race breaks.

**System.** The admission designs below. They cover:

- history's authoritative activity record;
- the dispatch channel from history to matching;
- admission at `RecordActivityTaskStarted`.

The corrected design follows the server's route:

1. The dispatch task's `Validate` checks `TransitionStarted.Possible` and the attempt stamp
   (`chasm/lib/activity/tasks.go:46-54`).
2. `Execute` sends `AddActivityTask` to matching (`tasks.go:56-83`), through the optional
   `DispatchTaskHook`, which wraps the send after validation (`tasks.go:16-26`, `:69-77`).
3. Admission, `Activity.HandleStarted`, checks the stamp and the transition again and answers a
   stale task with `ObsoleteMatchingTask` (`chasm/lib/activity/activity.go:225-239`).

The faulty design drops the check in step 3.

**Identities.** There are three:

- the logical activity, named by `activityId` (`Model.scala:33`);
- the attempt, named by its number and stamp;
- the delivery, which is one dispatch message, possibly delivered more than once.

The supported sketch folds the stamp into `Message.queued` and counts admitted attempts in
`Active`. The proposed layer (E8) separates the three.

**Go comparison.** `model/go/standaloneactivity` mirrors `activityProduct` and `activityProtocol`.
Task 4's parity covers those two machines over their shared domain, disabled actions included. The
admission designs are new behavior with no Go counterpart, so they are checked, not compared.

## Domain manifest

| Dimension | Bound in the supported sketch | Notes |
| --- | --- | --- |
| Logical activities | 1 | Task 4 may raise it to 2 |
| Dispatch messages in flight | at most 1 | `Message.empty`, `Message.queued` or `Message.redelivery` |
| Admitted, unclosed attempts | 0, 1, 2 | `Active`. 2 exists only as a violation witness |
| Controls | pause | Unpause, request-cancel and terminate are disabled (empty step lists). That is out of scope, not a hole |
| Attempt results | completed | Failed and canceled are disabled |
| Faults | redelivery before acknowledgment, at most once per logical message | A nondeterministic alternative of admission |
| Faults, proposed layer | enqueue commit failure; ordinary crash (ephemeral loss); acknowledgment loss; committed-storage loss, disabled unless its assumption is selected | See [Queue and persistence refinement](#queue-and-persistence-refinement) |
| Timers | none | The proposed layer adds competing schedule-to-start and schedule-to-close (oracle A6) |
| Ends | paused, paused while held, completed | With no unpause, a pause is where a path may end |
| Limits | `three` (3 steps) pinned; `five` (5 steps) free | Reachable states: 10 of 45 in the corrected design; goir builds no table for the stale design, because its refinement fails |

## Sketch: supported today

This block uses only the vocabulary listed in [README.md](README.md#the-vocabulary-the-sketches-use).
It compiled, answered and lifted in the scratch check. The Queries are Scala today; lifting them is
E12.

```scala
package specimens.activity

import temporal.standaloneactivity.{AttemptResult, Control, Outcome, ProductPhase, ProductState, activity,
  activityProduct, attemptResult, attemptStart, control, pausedIsNotDispatched}
import umpire.*

val Family: umpire.Family = umpire.Family("temporal.activity.standalone.admission")

// ### System vocabulary: one logical activity, one dispatch message, the attempts admission committed

/** The activity as history's authoritative record has it. `pausedWhileHeld` is a pause that
  * arrived after admission: the attempt it holds is work admitted before the pause. */
enum AdmissionPhase derives Finite:
  case scheduled, paused, pausedWhileHeld, started, completed

/** The dispatch channel holds at most one message. It was validated when it was enqueued, and
  * nothing re-reads that eligibility while it is in flight. */
enum Message derives Finite:
  case empty, queued, redelivery

/** Attempts admission committed and no result closed. An enum rather than an Int, because the
  * lifter gives every Int field of one record the same range. */
enum Active derives Finite:
  case none, one, two

final case class AdmissionState(phase: AdmissionPhase, message: Message, active: Active) derives Finite

/** The statuses the product reads, by their product names, and the internal facts no product fact
  * is named after, which the refinement therefore drops. */
enum AdmissionFact derives Finite:
  case statusStarted, statusPaused, statusCompleted
  case dispatchEnqueued, attemptAdmitted, admissionRejected

type AdmissionStep = Step[AdmissionState, Outcome, AdmissionFact]

/** History's dispatch task sends the message. System-internal, so spelled as a timer today. */
val dispatch = timer("dispatch")

val scheduledEmpty: AdmissionState = AdmissionState(AdmissionPhase.scheduled, Message.empty, Active.none)

// ### Step functions shared by both designs

def dispatchStep(s: AdmissionState): List[AdmissionStep] =
  if s.phase != AdmissionPhase.scheduled || s.message != Message.empty then Nil
  else List(Step(Outcome.accepted, s.copy(message = Message.queued), List(AdmissionFact.dispatchEnqueued)))

/** Only pause is in scope; no unpause intervenes. A pause keeps the message in flight. */
def pauseStep(s: AdmissionState, c: Control): List[AdmissionStep] = c match
  case Control.pause =>
    s.phase match
      case AdmissionPhase.scheduled =>
        List(Step(Outcome.accepted, s.copy(phase = AdmissionPhase.paused), List(AdmissionFact.statusPaused)))
      case AdmissionPhase.started =>
        List(Step(Outcome.accepted, s.copy(phase = AdmissionPhase.pausedWhileHeld), List(AdmissionFact.statusPaused)))
      case _ => Nil
  case Control.unpause | Control.requestCancel | Control.terminate => Nil

def oneMore(a: Active): Active = a match
  case Active.none => Active.one
  case _           => Active.two

def oneLess(a: Active): Active = a match
  case Active.two => Active.one
  case _          => Active.none

/** A committed admission. The channel is at-least-once: the message may be consumed, or retained
  * and delivered again before matching acknowledges it. */
def admitted(s: AdmissionState): List[AdmissionStep] =
  val next = s.copy(phase = AdmissionPhase.started, active = oneMore(s.active))
  if s.message == Message.redelivery then
    List(Step(Outcome.accepted, next.copy(message = Message.empty), List(AdmissionFact.statusStarted, AdmissionFact.attemptAdmitted)))
  else List(
    Step(Outcome.accepted, next.copy(message = Message.empty), List(AdmissionFact.statusStarted, AdmissionFact.attemptAdmitted)),
    Step(Outcome.accepted, next.copy(message = Message.redelivery), List(AdmissionFact.statusStarted, AdmissionFact.attemptAdmitted),
      "the channel may deliver the message again"),
  )

/** The corrected design: admission re-reads current eligibility and drops a stale message. */
def admitCurrent(s: AdmissionState): List[AdmissionStep] = s.message match
  case Message.empty => Nil
  case Message.queued | Message.redelivery =>
    if s.phase == AdmissionPhase.scheduled then admitted(s)
    else List(Step(Outcome.accepted, s.copy(message = Message.empty), List(AdmissionFact.admissionRejected)))

/** The deliberately faulty design: admission trusts the eligibility the message was enqueued with. */
def admitStale(s: AdmissionState): List[AdmissionStep] = s.message match
  case Message.empty  => Nil
  case Message.queued | Message.redelivery => admitted(s)

def resultStep(s: AdmissionState, r: AttemptResult): List[AdmissionStep] = r match
  case AttemptResult.completed =>
    if s.phase != AdmissionPhase.started then Nil
    else List(Step(Outcome.accepted, s.copy(phase = AdmissionPhase.completed, active = oneLess(s.active)),
      List(AdmissionFact.statusCompleted)))
  case AttemptResult.failed(_) | AttemptResult.canceled => Nil

def productOfAdmission(s: AdmissionState): ProductState = s.phase match
  case AdmissionPhase.scheduled                              => ProductState(ProductPhase.scheduled)
  case AdmissionPhase.paused | AdmissionPhase.pausedWhileHeld => ProductState(ProductPhase.paused)
  case AdmissionPhase.started                                => ProductState(ProductPhase.started)
  case AdmissionPhase.completed                              => ProductState(ProductPhase.completed)

def admissionEvidence(f: AdmissionFact): String = f match
  case AdmissionFact.statusStarted     => "statusStarted"
  case AdmissionFact.statusPaused      => "statusPaused"
  case AdmissionFact.statusCompleted   => "statusCompleted"
  case AdmissionFact.dispatchEnqueued  => "dispatchEnqueued"
  case AdmissionFact.attemptAdmitted   => "attemptAdmitted"
  case AdmissionFact.admissionRejected => "admissionRejected"

/** No unpause is in scope, so a pause is where a path may end, as a completion is. */
def admissionEnds(s: AdmissionState): Boolean = s.phase != AdmissionPhase.scheduled && s.phase != AdmissionPhase.started

// ### The two designs

val currentAdmission: Machine[AdmissionState, Outcome, AdmissionFact] =
  machine[AdmissionState, Outcome, AdmissionFact](Family, "currentAdmission") {
    forEntity(activity)
    refines(activityProduct)(productOfAdmission)
    starts(scheduledEmpty)
    ends(admissionEnds)
    evidence(admissionEvidence)
    steps(dispatch ~> dispatchStep, control ~> pauseStep, attemptStart ~> admitCurrent, attemptResult ~> resultStep)
  }

val staleAdmission: Machine[AdmissionState, Outcome, AdmissionFact] =
  machine[AdmissionState, Outcome, AdmissionFact](Family, "staleAdmission") {
    forEntity(activity)
    refines(activityProduct)(productOfAdmission)
    starts(scheduledEmpty)
    ends(admissionEnds)
    evidence(admissionEvidence)
    steps(dispatch ~> dispatchStep, control ~> pauseStep, attemptStart ~> admitStale, attemptResult ~> resultStep)
  }

// ### Promises, written once and declared on each design

def notAdmittedWhilePaused(before: AdmissionState, after: AdmissionStep): Boolean =
  before.phase != AdmissionPhase.paused || after.state.phase != AdmissionPhase.started

def atMostOneActive(after: AdmissionStep): Boolean = after.state.active != Active.two

def terminalStays(before: AdmissionState, after: AdmissionStep): Boolean =
  before.phase != AdmissionPhase.completed || after.state.phase == AdmissionPhase.completed

val three: Limits = Limits("three", steps = 3, actions = 3, search = 4096)
val five: Limits = Limits("five", steps = 5, actions = 5, search = 65536)

/** Every claim and path, declared on one design: a Property or Scenario belongs to one machine. */
def admissionQueries(m: Machine[AdmissionState, Outcome, AdmissionFact]): Vector[Query] =
  val notPaused = m.property("notAdmittedWhilePaused") holdsAcross notAdmittedWhilePaused
  val oneActive = m.property("atMostOneActive") holds atMostOneActive
  val terminal = m.property("terminalStays") holdsAcross terminalStays
  val stale = m.scenario("staleDeliveryAfterPause").starts(scheduledEmpty)
    .actions(dispatch, control(Control.pause), attemptStart)
  val prePause = m.scenario("admittedBeforePause").starts(scheduledEmpty)
    .actions(dispatch, attemptStart, control(Control.pause))
  val duplicate = m.scenario("duplicateDelivery").starts(scheduledEmpty).actions(dispatch, attemptStart, attemptStart)
  val any = m.scenario("any").starts(scheduledEmpty).free
  Vector(
    query(s"${m.name}.staleDelivery") verify notPaused in stale limits three,
    query(s"${m.name}.admittedBeforePause") verify notPaused in prePause limits three,
    query(s"${m.name}.duplicateDelivery") verify oneActive in duplicate limits three,
    query(s"${m.name}.any.notAdmittedWhilePaused") verify notPaused in any limits five,
    query(s"${m.name}.any.atMostOneActive") verify oneActive in any limits five,
    query(s"${m.name}.any.terminalStays") verify terminal in any limits five,
    // The product's own Property, read through the design's declared refinement.
    query(s"${m.name}.product.pausedIsNotDispatched").verify(pausedIsNotDispatched)
      .in(stale)(using Reads.through(m, activityProduct)) limits three,
  )

val currentQueries: Vector[Query] = admissionQueries(currentAdmission)
val staleQueries: Vector[Query] = admissionQueries(staleAdmission)
```

## Sketch: proposed extensions

```scala
// UNSUPPORTED: nothing below exists in model/scala/umpire, and this block does not compile. Each
// declaration is an extension from README.md#proposed-extensions.

// E2: the dispatch channel as a declaration. Go derives, at each interruption point, the redelivery
// alternative that `admitted` writes by hand today, and every other fault the declaration permits.
val dispatchChannel = channel[Message]("dispatch", capacity = 1,
  order = Order.unordered, loss = Loss.none, duplicates = Duplicates.beforeAck(max = 1))

// E7: system steps that are not timers.
val dispatch = internal("dispatch")
val commitAdmission = internal("commitAdmission")
val ackDelivery = internal("ackDelivery")

// E1: at most one admitted active attempt, as a passive monitor. Its count leaves the machine state,
// so `productOfAdmission` no longer has to ignore it.
val oneActiveAttempt = monitor[Active]("atMostOneActiveAttempt", initial = Active.none) {
  on(AdmissionFact.attemptAdmitted)(oneMore)
  on(AdmissionFact.statusCompleted)(oneLess)
  violated(_ == Active.two)
  evaluate(after = AdmissionFact.attemptAdmitted)
}

// E4: the product sees statuses only. A stutter that records one of them is a refinement error.
refines(activityProduct)(productOfAdmission).visible(AdmissionFact.statusStarted, AdmissionFact.statusPaused,
  AdmissionFact.statusCompleted)

// E3: the queue the dispatch rides, as an opaque provider, then replaced in scope by the detailed one.
val dispatchQueue = provider("dispatchQueue") {
  val enqueue = event[Message]("enqueue", results = List("committed", "failed"))
  val deliver = event[Message]("deliver")
  val ack = event[Message]("ack")
  allows(redelivery(before = ack, max = 1), unordered, noLossOfCommitted(unless = assumed("storageLoss")))
}
val scopedQueue = replaces(dispatchQueue, detailedQueue, project = queueEvents)

// E8: evidence with its commitment and correlation. Invocation does not imply receiver effect.
val admissionCommitted = observe("attemptAdmissionCommitted", AdmissionFact.attemptAdmitted,
  commitment = Commitment.durable,
  keys = Correlation(operation = "activityId", attempt = "attempt", delivery = "stamp", causal = "pause"))

// E9: the manifest above, as data a receipt repeats.
val admissionScope = scope(entities = Map(activity -> 1), messages = 1, attempts = 2,
  faults = FaultBudget(1), excluded = List(control(Control.unpause), control(Control.requestCancel),
    control(Control.terminate)))

// E10: the local race scenario. It needs a hold-delivery actuator, which the canary profile lacks,
// so preparation rejects the canary Case before I/O.
val pauseRace = scenarioDag("pauseRace") {
  val id = bind(start(Timeout.unset, Timeout.unset, Timeout.unset), learns = "activityId")
  requires(holdDelivery(dispatchChannel))
  after(id)(control(Control.pause))
  release(dispatchChannel)
  observe(admissionCommitted, statusRead(id))
}
```

## Trace oracles

Search oracles apply to the named design and limits. "Scratch" means the existing Scala search
produced the witness, with the given explored product-state count. Task 4 must reproduce each
oracle through the generic Go semantics.

### A1: stale delivery after pause (the negative control)

Design: `staleAdmission`. Scenario: `staleDeliveryAfterPause`.

| # | Row → outcome state [facts] | Public observation | Internal observation |
| --- | --- | --- | --- |
| 1 | `scheduled-empty-none-dispatch` → accepted `scheduled-queued-none` [dispatchEnqueued] | none | Dispatch validated and sent (`DispatchTaskHook`) |
| 2 | `scheduled-queued-none-control-pause` → accepted `paused-queued-none` [statusPaused] | PauseActivityExecution accepted; Describe reads PAUSED | Pause committed |
| 3 | `paused-queued-none-attemptStart` → accepted `started-empty-one` [statusStarted, attemptAdmitted] | The worker's poll returns attempt 1; Describe reads STARTED | Admission committed for attempt 1, causally after the pause |

The evaluation point is after step 3:

- `notAdmittedWhilePaused` is violated. Scratch: `counterexample-found`, 5 states.
- The free search (`five`) returns the same three steps as the shortest witness (22 states).
- The declared refinement fails at row `paused-queued-none-attemptStart`, because
  `activityProduct` has no step from paused to started. The Scala and goir rules agree.

**A1′, the corrected design.** Design `currentAdmission`, same schedule. Step 3 becomes
`paused-queued-none-attemptStart` → accepted `paused-empty-none` [admissionRejected].

- That step is a stutter that records no product fact.
- Publicly, Describe keeps reading PAUSED and no attempt starts. The rejection shows only
  internally, as `ObsoleteMatchingTask`.
- `verified-within-limits`, 4 states. The free search also verifies (10 states).

### A2: admitted before the pause (legal, in both designs)

Scenario: `admittedBeforePause`, which is `dispatch`, then `attemptStart` (to `started-*-one`), then
`control-pause` (to `pausedWhileHeld-*-one` [statusPaused]).

- `notAdmittedWhilePaused` holds in both designs. Scratch: `verified-within-limits`, 6 states each.
- A2 differs from A1 only in the causal order of the admission commit and the pause commit, which
  is why E8 carries a causal key.

### A3: duplicate delivery

Scenario: `duplicateDelivery`, which is `dispatch`, then `attemptStart` taking the retained
alternative (to `started-redelivery-one`), then `attemptStart` again.

- **Stale design:** the second admission lands in `started-empty-two` [statusStarted,
  attemptAdmitted]. `atMostOneActive` is violated (scratch: 5 states; free search: 13 states).
- **Corrected design:** the second delivery is rejected as a stutter. `verified-within-limits`,
  5 states.
- The stale design's duplicate admission goes from started to started. Today's refinement rule
  accepts that as a stutter although it records statusStarted (finding F3). So the monitor, not the
  refinement, is what catches it until E4 exists.

### A4: completed activity started again

This comes from the free search of the stale design:

1. `dispatch`;
2. `attemptStart`, retained;
3. `attemptResult-completed` → `completed-redelivery-none` [statusCompleted];
4. `attemptStart` → `started-empty-one`.

`terminalStays` is violated (scratch: 16 states). It is verified in the corrected design.

### A5: the product Property through the refinement

`pausedIsNotDispatched` is read through `Reads.through(design, activityProduct)` on the A1 schedule.

- **Corrected design:** `verified-within-limits`, 4 states.
- **Stale design:** a model error. The refinement check fails before any search.

A receipt must keep "the design does not refine the product" apart from "the product Property has a
counterexample" (finding F8).

### Hand-reviewed oracles that need the proposed layer

**A6: competing timers.** Both schedule-to-start and schedule-to-close are set and expire before an
attempt starts.

- Two traces end in distinct facts: statusTimedOut(scheduleToStart) and
  statusTimedOut(scheduleToClose). The existing `activityProtocol` has both timers
  (`Model.scala:324-337`).
- Neither order may be pruned until a causal or declared scheduling constraint orders them.

**A7: failed commit at admission.** Admission is invoked and its durable update fails.

- No attemptAdmitted fact is recorded, and the message stays deliverable.
- Publicly, the poll gets no task and Describe reads SCHEDULED.
- A7 differs from A8 by the absence of the commit observation.

**A8: lost acknowledgment.** Admission commits, and the response to the poller is lost.

- attemptAdmitted is present internally, and Describe reads STARTED.
- The worker never runs the attempt. Only start-to-close, when set, settles it.
- A later redelivery meets the committed attempt and is answered idempotently (`activity.go:230`,
  matching request id). It is not a second admission.

**A9: ordinary crash versus storage loss.**

- An ordinary crash loses ephemeral state only: an in-memory sync match or an un-persisted add.
  Committed queue state survives and is redelivered, possibly twice.
- Committed-storage loss is a separate fault with its own ID. It is enabled only when its assumption
  is selected, and every receipt that uses it names the assumption.
- A provider that loses committed state on an ordinary crash fails scoped refinement (V2 below).

**A10: candidate evidence ambiguity (R8).** This is an uncontrolled concurrent pause/poll
scenario, separate from the held `pauseRace` DAG. Its admitted evidence contains no hold/release
control establishing admission order. Remove the admission-commit observation. The Run then records
only this:

- PauseActivityExecution accepted;
- the worker's poll returned attempt 1;
- no status read ordered after both.

Two executions are still compatible:

| Execution | Design | `notAdmittedWhilePaused` |
| --- | --- | --- |
| A1 | stale | violated |
| A2 | either | satisfied |

The verdict at the evaluation point is therefore `inconclusive`. Public responses on different
connections carry no causal order, and host clocks cannot supply one (EVD-07). Either of these
resolves it:

- a Describe read causally after both responses: PAUSED means A2, STARTED means A1;
- the commit observation with its causal pause key.

**A10 held-control counterpart.** In the proposed `pauseRace`, the declared control holds the
first dispatch before matching receives it, waits for pause, then releases it. If the control is
realized and recorded with that causal order, a returned first-attempt poll excludes A2 and
establishes admission after pause. The faulty design can therefore be `violated` from sufficient
public plus control evidence even without the internal commit observation. If the control
observation or its causal relation is missing, or public evidence cannot establish an admission,
the assessment remains `inconclusive` whenever compatible executions disagree. Removing a field
never removes causal knowledge already established by other admitted evidence.

## Queue and persistence refinement

The activity's durable dispatch queue is first an opaque provider, `dispatchQueue` (E3). Every
check that uses it names the assumption `dispatchQueue.opaque`.

**Interface.** The provider has three events:

| Event | Meaning |
| --- | --- |
| `enqueue(m)` | Answers `committed` or `failed` |
| `deliver(m)` | Hands the message to a consumer |
| `ack(m)` | Removes the message |

Interface state is the set of outstanding messages, at most one, each committed-not-delivered or
delivered-not-acked. The interface allows exactly this, and nothing else:

- enqueue may fail, leaving no message;
- a committed message may be delivered up to twice before its acknowledgment;
- delivery order is unspecified;
- no committed message is lost, unless `storageLoss` is assumed;
- a crash changes nothing at the interface except that a delivered-not-acked message may be
  delivered again.

**Detailed provider.** It follows the selected route:

1. The CHASM dispatch task is durable in history and at-least-once.
2. `Execute` calls `AddActivityTask`.
3. Matching either sync-matches the task, which is ephemeral, or persists it in the task queue,
   which is durable.
4. A poll delivers it.
5. `RecordActivityTaskStarted` reaches admission (`HandleStarted`).
6. Matching completes the task, which is the acknowledgment.

| Detailed transition | Interface event |
| --- | --- |
| Dispatch task durably scheduled in history | `enqueue → committed`; history retains the delivery obligation |
| `AddActivityTask` invoked | none; invocation implies no receiver effect |
| Task persisted in matching | none (stutter); matching acquires durable custody |
| Sync match reserved for a waiting poller | none (stutter); the reservation is ephemeral |
| History retries the dispatch after an error | none until `deliver`; it preserves the same logical message obligation |
| Poll returns the task | `deliver` |
| Admission durably committed, then task completed in matching | `ack`; the outstanding delivery obligation is discharged |
| Crash: in-memory match lost; task not persisted | none; the durable history obligation remains and drives redelivery |
| `storageLoss` (separate fault, assumption-gated) | drops a committed message |

The provider includes durable history dispatch and matching custody together. A sync match alone
never establishes `committed`; successful admission transfers the obligation into the durable
activity record. A concrete refinement in task 4 must check that custody is never absent while a
committed message remains outstanding.

The interruption points are after invocation, after persistence, after delivery, after the
admission commit, and after the acknowledgment. At each point Go derives the permitted faults from
E2's declaration.

**Violating providers.** Each must fail scoped refinement at the named step:

| Provider | Fault | Expected failure |
| --- | --- | --- |
| V1 | Discards the durable history delivery obligation on an `AddActivityTask` invocation, before matching persistence or durable activity admission takes custody | An ordinary crash loses the remaining ephemeral message. No durable custodian remains, yet the interface still calls the message committed; refinement fails at the crash step without `storageLoss` |
| V2 | An ordinary crash wipes persisted tasks | Fails at the crash step under a fault ID distinct from `storageLoss`. Its receipt must not credit the storage-loss assumption |

**Expected results.**

- The detailed provider passes scoped refinement.
- With it in place, the admission promises keep A1′'s verdicts.
- With the stale design, A1 is still found.
- Receipts list the provider, the assumptions and the bounds.

## Evidence and capability matrix

| Fact or observation | Kind | Source today | Testpilot today | Functional profile | Canary profile | Task |
| --- | --- | --- | --- | --- | --- | --- |
| statusScheduled, statusPaused, statusStarted, statusCompleted | public | DescribeActivityExecution | Admissible through `InvokeRPC` and `ReadEvidence`; not used by any Case yet | yes | yes | 6, 9 |
| Pause accepted | public | PauseActivityExecution response | `InvokeRPC` | yes | yes | 6, 9 |
| The worker receives attempt n | public to the worker | PollActivityTaskQueue response | Missing: no activity activation | after task 13 | observed only | 13 |
| dispatchEnqueued | internal | `DispatchTaskHook` (`tasks.go:16-26`) | Missing | after task 10 | unavailable | 10 |
| attemptAdmitted (durable commit) | internal | `HandleStarted` commit (`activity.go:225-239`), emitted only after the durable update commits | Missing: no durable-commit observation | after task 10 | unavailable; the property may be inconclusive (A10) | 10, 6 |
| admissionRejected | internal | `ObsoleteMatchingTask` (`activity.go:234`, `:238`) | Missing | after task 10 | unavailable | 10 |
| Hold-delivery control | actuator | `DispatchTaskHook` can hold after `Validate` | Missing: `FaultKind` is worker lifecycle only | after task 10 | rejected before I/O | 10 |
| The one realized fault | Run Event | `FAULT_INJECTED` (`proto/internal/temporal/server/api/testpilot/v1/run.proto:79-83`) | Worker stop and resume exist; server faults are missing | yes (worker) | per profile | 10 |
| Correlation: activity, attempt, delivery, causal pause | keys | Correlated evidence scope and operation key | Operation keys and parents exist; attempt, delivery and causal keys are missing | after tasks 6 and 10 | public keys only | 6, 7, 10 |
