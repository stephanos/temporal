package temporal.feature.activity.standalone

// authoring: header

/* The standalone activity Model
 *
 * A Temporal activity started directly through `StartActivityExecution`, with no workflow. Grounded
 * in `chasm/lib/activity/statemachine.go` and `proto/v1/activity_state.proto`. Reset is deferred
 * (like cancellation in the Nexus Model) and not modeled. Heartbeat timeout is not modeled.
 * Standalone activities write no history events, so every fact is a status read through
 * `DescribeActivityExecution` or a result read through `PollActivityExecution`.
 *
 * Read from top to bottom: vocabulary -> the two machines -> what they promise -> what the set asks.
 */

import umpire.*
import temporal.feature.worker as Worker
import Worker.{workerStop, serve, WorkerState}
// `Timeout` is the Nexus Model's domain, reused as-is: an activity's deadline is set or it expires.
import temporal.feature.nexus.caller.{Timeout, unset, expires}

// authoring: entities

/* Entities
 *
 * An activity is named by the id the caller chose; there is no workflow to refer to. */

val activity: Entity = entity("activity") key "activityId"

// authoring: domains

/* The input domains
 *
 * `failed(retryable: Boolean)` is one constructor and two classes, like the Nexus handler error.
 * `Control` is the caller's four post-start requests, one action with four classes rather than
 * four actions, because they share one result domain and one row shape. */

enum AttemptResult derives Finite:
  case completed
  case failed(retryable: Boolean)
  case canceled

enum Delivery derives Finite:
  case accepted, notFound

enum Control derives Finite:
  case pause, unpause, requestCancel, terminate

// authoring: actions

/* Actions
 *
 * The caller starts and controls; the worker attempts. The worker's stop is the shared fault of
 * the worker module, so the composition can `sync` it. */

val start: Action[(Timeout, Timeout, Timeout)] =
  action("start")
    .party(caller)
    .creates(activity)
    .schema("temporal.api.workflowservice.v1.StartActivityExecutionRequest")
    .input[Timeout]("scheduleToClose")
    .input[Timeout]("scheduleToStart")
    .input[Timeout]("startToClose")

/** The worker's poll receives the task (`PollActivityTaskQueue`). */
val attemptStart: Action[EmptyTuple] =
  action("attemptStart")
    .party(worker)
    .on(activity)
    .schema("temporal.api.workflowservice.v1.PollActivityTaskQueueResponse")

val attemptResult: Action[AttemptResult *: EmptyTuple] =
  action("attemptResult")
    .party(worker)
    .on(activity)
    .schema("RespondActivityTaskCompletedRequest | RespondActivityTaskFailedRequest | RespondActivityTaskCanceledRequest")
    .input[AttemptResult]("result")
    .examples(
      Tuple1(AttemptResult.failed(retryable = false)) -> "ApplicationFailure nonRetryable",
      Tuple1(AttemptResult.failed(retryable = true)) -> "ApplicationFailure retryable",
    )

val control: Action[Control *: EmptyTuple] =
  action("control")
    .party(caller)
    .on(activity)
    .schema("PauseActivityExecutionRequest | UnpauseActivityExecutionRequest | RequestCancelActivityExecutionRequest | TerminateActivityExecutionRequest")
    .input[Control]("control")
    .results[Delivery]

// (`workerStop` is imported from `temporal.feature.worker`.)

// authoring: observation

/* The derived observation
 *
 * No history event records an attempt, so the count is read back through
 * `DescribeActivityExecution`. */

val attemptCount: Observation = observation("attemptCount") on activity read "attempt"

// authoring: product

/* The product machine
 *
 * What the caller sees through Describe, with no account of how the server gets there. Every
 * Property written against it is carried to the protocol machine by the refinement declared
 * there. The refinement is by mapped states: a protocol row is accounted for when its two states
 * map to the same product state, or to two the product has any row between. */

enum ProductPhase derives Finite:
  case scheduled, started, paused, cancelRequested, completed, failed, canceled, terminated, timedOut

final case class ProductState(phase: ProductPhase) derives Finite, CanEqual

enum ProductOutcome derives Finite:
  case accepted, notFound

enum ProductFact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested, statusCompleted,
    statusFailed, statusCanceled, statusTerminated, statusTimedOut

type ProductStep = Step[ProductState, ProductOutcome, ProductFact]

private def productStep(phase: ProductPhase, recorded: ProductFact): List[ProductStep] =
  List(Step(ProductOutcome.accepted, ProductState(phase), List(recorded)))

/** The five phases the product machine ends on. */
def productTerminal(state: ProductState): Boolean = state.phase match
  case ProductPhase.completed | ProductPhase.failed | ProductPhase.canceled | ProductPhase.terminated |
      ProductPhase.timedOut => true
  case ProductPhase.scheduled | ProductPhase.started | ProductPhase.paused | ProductPhase.cancelRequested => false

/** A worker picks the task up. Only a scheduled activity has a task to pick up. */
def attemptStartStep(state: ProductState): List[ProductStep] = state.phase match
  case ProductPhase.scheduled => productStep(ProductPhase.started, ProductFact.statusStarted)
  case _ => Nil

/** The worker's result for the attempt. Unlike the Nexus operation, a retry is visible here: a
  * retryable failure puts the activity back to scheduled and Describe reads SCHEDULED again
  * (`TransitionRescheduled`), or settles it as canceled when a cancel was requested. What stays
  * invisible is the backoff between the two, which is the protocol machine's account. A cancel
  * result counts only when a cancel was requested. */
def attemptResultStep(state: ProductState, result: AttemptResult): List[ProductStep] =
  state.phase match
    case ProductPhase.started => result match
      case AttemptResult.completed => productStep(ProductPhase.completed, ProductFact.statusCompleted)
      case AttemptResult.failed(false) => productStep(ProductPhase.failed, ProductFact.statusFailed)
      case AttemptResult.failed(true) => productStep(ProductPhase.scheduled, ProductFact.statusScheduled)
      case AttemptResult.canceled => Nil
    case ProductPhase.cancelRequested => result match
      case AttemptResult.completed => productStep(ProductPhase.completed, ProductFact.statusCompleted)
      case AttemptResult.failed(false) => productStep(ProductPhase.failed, ProductFact.statusFailed)
      case AttemptResult.failed(true) => productStep(ProductPhase.canceled, ProductFact.statusCanceled)
      case AttemptResult.canceled => productStep(ProductPhase.canceled, ProductFact.statusCanceled)
    case _ => Nil

/** The caller's four requests. On a finished activity every one is not found and changes nothing;
  * a repeated cancel request is idempotent and reads the same status back. */
def controlStep(state: ProductState, control: Control): List[ProductStep] =
  if productTerminal(state) then List(Step(ProductOutcome.notFound, state, Nil))
  else control match
    case Control.pause => state.phase match
      case ProductPhase.scheduled | ProductPhase.started => productStep(ProductPhase.paused, ProductFact.statusPaused)
      case _ => Nil
    case Control.unpause => state.phase match
      case ProductPhase.paused => productStep(ProductPhase.scheduled, ProductFact.statusScheduled)
      case _ => Nil
    case Control.requestCancel => state.phase match
      case ProductPhase.scheduled | ProductPhase.started | ProductPhase.paused | ProductPhase.cancelRequested =>
        productStep(ProductPhase.cancelRequested, ProductFact.statusCancelRequested)
      case _ => Nil
    case Control.terminate => productStep(ProductPhase.terminated, ProductFact.statusTerminated)

/** The worker stopping is a fault the Run records and the activity does not feel. The product
  * machine cannot see it: a step that kept the state and recorded nothing would be
  * indistinguishable from a stutter. */
def workerStopStep(state: ProductState): List[ProductStep] = Nil

/** One of the activity's deadlines firing. Which deadline is the protocol's account of how, so the
  * product machine has one timer, and it fires while the activity is not over. */
val timeout: Action[EmptyTuple] = timer("timeout")

def timeoutStep(state: ProductState): List[ProductStep] = state.phase match
  case ProductPhase.scheduled | ProductPhase.started | ProductPhase.cancelRequested | ProductPhase.paused =>
    productStep(ProductPhase.timedOut, ProductFact.statusTimedOut)
  case _ => Nil

val activityProduct: Machine[ProductState, ProductOutcome, ProductFact] =
  machine[ProductState, ProductOutcome, ProductFact]("activityProduct"):
    forEntity(activity)
    starts(ProductPhase.scheduled)
    ends(ProductPhase.completed, ProductPhase.failed, ProductPhase.canceled, ProductPhase.terminated,
      ProductPhase.timedOut)
    timers(timeout)
    evidence:
      case ProductFact.statusScheduled => "statusScheduled"
      case ProductFact.statusStarted => "statusStarted"
      case ProductFact.statusPaused => "statusPaused"
      case ProductFact.statusCancelRequested => "statusCancelRequested"
      case ProductFact.statusCompleted => "statusCompleted"
      case ProductFact.statusFailed => "statusFailed"
      case ProductFact.statusCanceled => "statusCanceled"
      case ProductFact.statusTerminated => "statusTerminated"
      case ProductFact.statusTimedOut => "statusTimedOut"
    steps(
      attemptStart ~> attemptStartStep,
      attemptResult ~> attemptResultStep,
      control ~> controlStep,
      workerStop ~> workerStopStep,
      timeout ~> timeoutStep,
    )

// authoring: protocol

/* The protocol machine
 *
 * How the server gets there: the retry the product cannot see, the pause request a running
 * attempt has to finish before it takes effect, the three timers the start request sets, and the
 * attempt count. Written against the same actions, so a Property proved on the product machine is
 * carried here by the refinement.
 *
 * The machine begins before the activity exists: `unstarted` is the "no instance yet" member, and
 * it is what makes the three deadline fields reachable at anything but their first value. */

enum Phase derives Finite:
  case unstarted, scheduled, backingOff, started, paused, pauseRequested, cancelRequested,
    completed, failed, canceled, terminated, timedOut

/** Which timer fired. The Describe status does not say, but the failure it reads back does, so a
  * Contract that did not check it would pass a run that timed out on the wrong deadline. */
enum TimeoutType derives Finite:
  case scheduleToClose, scheduleToStart, startToClose

/** The attempt count is bounded by the Limits in the design; the bound is written here and the
  * saturating successor keeps a retry inside it. */
inline val attemptBound = 2
type Attempts = Bounded[attemptBound.type]
object Attempts:
  inline def apply(inline n: Int): Attempts = Bounded[attemptBound.type](n)

final case class ProtocolState(
    phase: Phase,
    attempts: Attempts,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout,
) derives Finite, CanEqual

enum ProtocolOutcome derives Finite:
  case accepted, notFound

enum ProtocolFact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested, statusCompleted,
    statusFailed, statusCanceled, statusTerminated
  case statusTimedOut(timeoutType: TimeoutType)
  case attemptCount

type ProtocolStep = Step[ProtocolState, ProtocolOutcome, ProtocolFact]

/** The five phases the design ends on. A control that arrives after one of them is not found. */
def terminalPhase(phase: Phase): Boolean = phase match
  case Phase.completed | Phase.failed | Phase.canceled | Phase.terminated | Phase.timedOut => true
  case Phase.unstarted | Phase.scheduled | Phase.backingOff | Phase.started | Phase.paused |
      Phase.pauseRequested | Phase.cancelRequested => false

/** Started and not yet over: the phases the schedule-to-close deadline covers. */
def running(phase: Phase): Boolean = phase match
  case Phase.scheduled | Phase.backingOff | Phase.started | Phase.paused | Phase.pauseRequested |
      Phase.cancelRequested => true
  case _ => false

private def moves(state: ProtocolState, phase: Phase, recorded: List[ProtocolFact]): List[ProtocolStep] =
  List(Step(ProtocolOutcome.accepted, state.copy(phase = phase), recorded))

/** The caller's start request. It names the activity's three deadlines, and every one of them is
  * a state field because whether a timer fires is a question about the activity and not about the
  * request that started it. */
def startStep(
    state: ProtocolState,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout,
): List[ProtocolStep] =
  if state.phase != Phase.unstarted then Nil
  else
    List(Step(
      ProtocolOutcome.accepted,
      ProtocolState(Phase.scheduled, Attempts(0), scheduleToClose, scheduleToStart, startToClose),
      List(ProtocolFact.statusScheduled),
    ))

/** A worker picks the task up: the attempt count rises, and Describe reads both the status and
  * the count. */
def protocolAttemptStartStep(state: ProtocolState): List[ProtocolStep] =
  if state.phase != Phase.scheduled then Nil
  else
    List(Step(
      ProtocolOutcome.accepted,
      state.copy(phase = Phase.started, attempts = state.attempts.saturatingSucc),
      List(ProtocolFact.statusStarted, ProtocolFact.attemptCount),
    ))

/** The worker's result. What the product cannot see is the `backingOff` phase a retryable failure
  * passes through from a plain start; it is read as a cancel when a cancel was requested
  * (`statemachine.go` lists CANCEL_REQUESTED as a source of Canceled), and lands in `paused` when
  * a pause was requested (`TransitionAttemptFailedWhilePauseRequested`). A cancel result with no
  * cancel requested is not a row. */
def protocolAttemptResultStep(state: ProtocolState, result: AttemptResult): List[ProtocolStep] =
  state.phase match
    case Phase.started => result match
      case AttemptResult.completed => moves(state, Phase.completed, List(ProtocolFact.statusCompleted))
      case AttemptResult.failed(false) => moves(state, Phase.failed, List(ProtocolFact.statusFailed))
      case AttemptResult.failed(true) => moves(state, Phase.backingOff, List(ProtocolFact.attemptCount))
      case AttemptResult.canceled => Nil
    case Phase.cancelRequested => result match
      case AttemptResult.completed => moves(state, Phase.completed, List(ProtocolFact.statusCompleted))
      case AttemptResult.failed(false) => moves(state, Phase.failed, List(ProtocolFact.statusFailed))
      case AttemptResult.failed(true) => moves(state, Phase.canceled, List(ProtocolFact.statusCanceled))
      case AttemptResult.canceled => moves(state, Phase.canceled, List(ProtocolFact.statusCanceled))
    case Phase.pauseRequested => result match
      case AttemptResult.completed => moves(state, Phase.completed, List(ProtocolFact.statusCompleted))
      case AttemptResult.failed(false) => moves(state, Phase.failed, List(ProtocolFact.statusFailed))
      case AttemptResult.failed(true) => moves(state, Phase.paused, List(ProtocolFact.statusPaused))
      case AttemptResult.canceled => Nil
    case _ => Nil

/** The caller's four requests. A pause of a running attempt is only requested (`TransitionPaused`
  * is for the phases with no attempt in flight); Describe reads PAUSE_REQUESTED, but the product
  * cannot tell the two apart, so both record `statusPaused`. A control on a finished activity is
  * not found, and none is a row before the start. */
def protocolControlStep(state: ProtocolState, control: Control): List[ProtocolStep] =
  if terminalPhase(state.phase) then List(Step(ProtocolOutcome.notFound, state, Nil))
  else if state.phase == Phase.unstarted then Nil
  else control match
    case Control.pause => state.phase match
      case Phase.scheduled | Phase.backingOff => moves(state, Phase.paused, List(ProtocolFact.statusPaused))
      case Phase.started => moves(state, Phase.pauseRequested, List(ProtocolFact.statusPaused))
      case _ => Nil
    case Control.unpause => state.phase match
      case Phase.paused => moves(state, Phase.scheduled, List(ProtocolFact.statusScheduled))
      case Phase.pauseRequested => moves(state, Phase.started, List(ProtocolFact.statusStarted))
      case _ => Nil
    case Control.requestCancel =>
      moves(state, Phase.cancelRequested, List(ProtocolFact.statusCancelRequested))
    case Control.terminate =>
      moves(state, Phase.terminated, List(ProtocolFact.statusTerminated))

/** The worker stopping is a fault the Run records and the activity does not feel, so the step
  * keeps the state and records nothing. On a path it is confirmed by the evidence of the step
  * after it. */
def protocolWorkerStopStep(state: ProtocolState): List[ProtocolStep] =
  List(Step(ProtocolOutcome.accepted, state, Nil))

/** The backoff timer. It is what makes `backingOff` a phase the activity leaves rather than a
  * state it is stuck in, and it records nothing. */
val backoff: Action[EmptyTuple] = timer("backoff")
val scheduleToClose: Action[EmptyTuple] = timer("scheduleToClose")
val scheduleToStart: Action[EmptyTuple] = timer("scheduleToStart")
val startToClose: Action[EmptyTuple] = timer("startToClose")

def backoffStep(state: ProtocolState): List[ProtocolStep] =
  if state.phase != Phase.backingOff then Nil else moves(state, Phase.scheduled, Nil)

/** The schedule-to-close deadline covers the whole activity, paused or not, so it fires in every
  * running phase -- and only when the start request set it. */
def scheduleToCloseStep(state: ProtocolState): List[ProtocolStep] =
  if running(state.phase) && state.scheduleToClose == expires then
    moves(state, Phase.timedOut, List(ProtocolFact.statusTimedOut(TimeoutType.scheduleToClose)))
  else Nil

/** The schedule-to-start deadline covers the wait for a worker, so it stops at the start. */
def scheduleToStartStep(state: ProtocolState): List[ProtocolStep] =
  if (state.phase == Phase.scheduled || state.phase == Phase.backingOff) && state.scheduleToStart == expires then
    moves(state, Phase.timedOut, List(ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart)))
  else Nil

/** The start-to-close deadline covers one attempt, so it runs while an attempt is in flight,
  * whatever the caller has asked of it meanwhile. */
def startToCloseStep(state: ProtocolState): List[ProtocolStep] =
  state.phase match
    case Phase.started | Phase.pauseRequested | Phase.cancelRequested if state.startToClose == expires =>
      moves(state, Phase.timedOut, List(ProtocolFact.statusTimedOut(TimeoutType.startToClose)))
    case _ => Nil

/** How a protocol state reads as a product state. Backing off is still scheduled, because the
  * product cannot see the wait; not yet started reads as scheduled, because the product begins
  * there; a requested pause is still started, because the attempt is still running and only its
  * end honors the request. Every other field is hidden. */
def productOf(state: ProtocolState): ProductState = ProductState(state.phase match
  case Phase.unstarted | Phase.scheduled | Phase.backingOff => ProductPhase.scheduled
  case Phase.started | Phase.pauseRequested => ProductPhase.started
  case Phase.paused => ProductPhase.paused
  case Phase.cancelRequested => ProductPhase.cancelRequested
  case Phase.completed => ProductPhase.completed
  case Phase.failed => ProductPhase.failed
  case Phase.canceled => ProductPhase.canceled
  case Phase.terminated => ProductPhase.terminated
  case Phase.timedOut => ProductPhase.timedOut)

/** The map as a type-level fact, so the machine's `refines` line and a `verify` Query that reads a
  * product Property on a protocol Scenario both find it. */
given productView: Refines[ProtocolState, ProductState] = Refines(productOf)

val activityProtocol: Machine[ProtocolState, ProtocolOutcome, ProtocolFact] =
  machine[ProtocolState, ProtocolOutcome, ProtocolFact]("activityProtocol"):
    forEntity(activity)
    refines(activityProduct)
    starts(Phase.unstarted)
    ends(Phase.completed, Phase.failed, Phase.canceled, Phase.terminated, Phase.timedOut)
    timers(backoff, scheduleToClose, scheduleToStart, startToClose)
    unobservable(backoff)
    evidence:
      case ProtocolFact.statusScheduled => "statusScheduled"
      case ProtocolFact.statusStarted => "statusStarted"
      case ProtocolFact.statusPaused => "statusPaused"
      case ProtocolFact.statusCancelRequested => "statusCancelRequested"
      case ProtocolFact.statusCompleted => "statusCompleted"
      case ProtocolFact.statusFailed => "statusFailed"
      case ProtocolFact.statusCanceled => "statusCanceled"
      case ProtocolFact.statusTerminated => "statusTerminated"
      case ProtocolFact.statusTimedOut(_) => "statusTimedOut"
      case ProtocolFact.attemptCount => attemptCount
    steps(
      start ~> startStep,
      attemptStart ~> protocolAttemptStartStep,
      attemptResult ~> protocolAttemptResultStep,
      control ~> protocolControlStep,
      workerStop ~> protocolWorkerStopStep,
      backoff ~> backoffStep,
      scheduleToClose ~> scheduleToCloseStep,
      scheduleToStart ~> scheduleToStartStep,
      startToClose ~> startToCloseStep,
    )

// authoring: properties

/* What the machines promise
 *
 * Same-step claims name their action under `when`; the two transition claims are declared on the
 * product machine and read on the protocol machine through the map. */

/* Once an activity is over, no step changes its phase. */
val terminalIsFinal: Property[ProductState] =
  property("terminalIsFinal")(activityProduct) holds: (before, after) =>
    !productTerminal(before.state) || after.state.phase == before.state.phase

/* A completed result settles the activity as completed, and Describe reads it. */
val completes: Property[ProtocolState] =
  property("completes")(activityProtocol) when attemptResult(AttemptResult.completed) holds: step =>
    step.state.phase == Phase.completed && step.facts.contains(ProtocolFact.statusCompleted)

/* A non-retryable failure settles the activity as failed. */
val nonRetryableFails: Property[ProtocolState] =
  property("nonRetryableFails")(activityProtocol) when attemptResult(AttemptResult.failed(false)) holds: step =>
    step.state.phase == Phase.failed && step.facts.contains(ProtocolFact.statusFailed)

/** Completed on the second attempt of an activity with no deadline set. A claim fixes one state,
  * so every field is named. */
val completedOnRetry: ProtocolState =
  ProtocolState(Phase.completed, Attempts(2), unset, unset, unset)

/* The retried attempt completes the activity on its second attempt. */
val retryCompletes: Property[ProtocolState] =
  property("retryCompletes")(activityProtocol) when attemptResult(AttemptResult.completed) holds: step =>
    step.state == completedOnRetry && step.facts.contains(ProtocolFact.statusCompleted)

/* A cancel request on a running attempt is recorded as requested; the worker settles it. */
val cancelRequestedWhileStarted: Property[ProtocolState] =
  property("cancelRequestedWhileStarted")(activityProtocol) when control(Control.requestCancel) holds: step =>
    step.state.phase == Phase.cancelRequested && step.facts.contains(ProtocolFact.statusCancelRequested)

/* The worker's cancel result settles a cancel-requested activity as canceled. */
val canceledByWorker: Property[ProtocolState] =
  property("canceledByWorker")(activityProtocol) when attemptResult(AttemptResult.canceled) holds: step =>
    step.state.phase == Phase.canceled && step.facts.contains(ProtocolFact.statusCanceled)

/* A terminate settles the activity as terminated. */
val terminated: Property[ProtocolState] =
  property("terminated")(activityProtocol) when control(Control.terminate) holds: step =>
    step.state.phase == Phase.terminated && step.facts.contains(ProtocolFact.statusTerminated)

/* A paused activity is never dispatched: no step takes it straight to started. */
val pausedIsNotDispatched: Property[ProductState] =
  property("pausedIsNotDispatched")(activityProduct) holds: (before, after) =>
    before.state.phase != ProductPhase.paused || after.state.phase != ProductPhase.started

/* The schedule-to-start deadline settles an activity no worker picked up as timed out, and the
 * failure read back says which deadline it was. */
val scheduleToStartFires: Property[ProtocolState] =
  property("scheduleToStartFires")(activityProtocol) when scheduleToStart holds: step =>
    step.state.phase == Phase.timedOut &&
      step.facts.contains(ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart))

/* The start-to-close deadline settles an attempt no worker finished as timed out. */
val startToCloseFires: Property[ProtocolState] =
  property("startToCloseFires")(activityProtocol) when startToClose holds: step =>
    step.state.phase == Phase.timedOut &&
      step.facts.contains(ProtocolFact.statusTimedOut(TimeoutType.startToClose))

// authoring: scenarios

/* The paths the Queries run
 *
 * Each path is one functional test's shape: the start request with no deadline set unless the
 * path is about one, then the side effects that settle the activity. */

val completed: Scenario[ProtocolState] =
  scenario("completed")(activityProtocol) starts Phase.unstarted actions (
    start(unset, unset, unset), attemptStart, attemptResult(AttemptResult.completed))

val nonRetryable: Scenario[ProtocolState] =
  scenario("nonRetryable")(activityProtocol) starts Phase.unstarted actions (
    start(unset, unset, unset), attemptStart, attemptResult(AttemptResult.failed(false)))

/* The retryable failure backs the activity off; the backoff timer fires and records nothing; a
 * worker picks the retry up and completes it. */
val retriedThenCompleted: Scenario[ProtocolState] =
  scenario("retriedThenCompleted")(activityProtocol) starts Phase.unstarted actions (
    start(unset, unset, unset), attemptStart, attemptResult(AttemptResult.failed(true)), backoff,
    attemptStart, attemptResult(AttemptResult.completed))

val cancelRequestedThenCanceled: Scenario[ProtocolState] =
  scenario("cancelRequestedThenCanceled")(activityProtocol) starts Phase.unstarted actions (
    start(unset, unset, unset), attemptStart, control(Control.requestCancel),
    attemptResult(AttemptResult.canceled))

/* The worker stops before anything is dispatched, so the caller's terminate is what settles it. */
val terminatedWhileScheduled: Scenario[ProtocolState] =
  scenario("terminatedWhileScheduled")(activityProtocol) starts Phase.unstarted actions (
    start(unset, unset, unset), workerStop, control(Control.terminate))

val pausedThenCompleted: Scenario[ProtocolState] =
  scenario("pausedThenCompleted")(activityProtocol) starts Phase.unstarted actions (
    start(unset, unset, unset), control(Control.pause), control(Control.unpause), attemptStart,
    attemptResult(AttemptResult.completed))

/* The start request sets the schedule-to-start deadline; the worker stops, so nothing picks the
 * task up; the deadline fires. */
val scheduleToStartExpires: Scenario[ProtocolState] =
  scenario("scheduleToStartExpires")(activityProtocol) starts Phase.unstarted actions (
    start(unset, expires, unset), workerStop, scheduleToStart)

/* The start request sets the start-to-close deadline; a worker picks the task up and never
 * finishes; the deadline fires. */
val startToCloseExpires: Scenario[ProtocolState] =
  scenario("startToCloseExpires")(activityProtocol) starts Phase.unstarted actions (
    start(unset, unset, expires), attemptStart, startToClose)

val three: Limits = limits("three")(steps = 3, actions = 3, search = 4096)
val four: Limits = limits("four")(steps = 4, actions = 4, search = 32768)
val six: Limits = limits("six")(steps = 6, actions = 6, search = 262144)

// authoring: queries

/* The Queries
 *
 * Eight find their same-step claim on their path and are realized by the functional set; the two
 * product claims are verified over every trace of one path each, outside the set. */

val completion = query("completion") find completes in completed limits three
val nonRetryableFailure = query("nonRetryableFailure") find nonRetryableFails in nonRetryable limits three
val retry = query("retry") find retryCompletes in retriedThenCompleted limits six
val cancel = query("cancel") find canceledByWorker in cancelRequestedThenCanceled limits four
val terminate = query("terminate") find terminated in terminatedWhileScheduled limits three
val pauseResume = query("pauseResume") find completes in pausedThenCompleted limits six
val scheduleToStartTimeout =
  query("scheduleToStartTimeout") find scheduleToStartFires in scheduleToStartExpires limits three
val startToCloseTimeout =
  query("startToCloseTimeout") find startToCloseFires in startToCloseExpires limits three
val terminalHolds = query("terminalHolds") verify terminalIsFinal in completed limits three
val pauseHolds = query("pauseHolds") verify pausedIsNotDispatched in pausedThenCompleted limits six

// authoring: set

/* The functional set
 *
 * The Case drives the caller and the worker. No `repeat`: standalone activities are CHASM only,
 * so there is no implementation switch to run under. */

val standaloneActivityTests: Set = set("standaloneActivityTests"):
  purpose(functional)
  bind(caller -> driven, worker -> driven)
  queries(completion, nonRetryableFailure, retry, cancel, terminate, pauseResume,
    scheduleToStartTimeout, startToCloseTimeout)

/* The canary set
 *
 * The worker is `observed`: a deployment runs its own worker, and the verifier reads which result
 * occurred and checks the machine allows it. Both paths record evidence at every step. */

val standaloneActivityCanary: Set = set("standaloneActivityCanary"):
  purpose(canary)
  bind(caller -> driven, worker -> observed)
  queries(completion, cancel)

/* The exploratory set
 *
 * An exploration covers the protocol machine rather than listing Queries: the rows within the
 * budget's steps of a start, the results they reach and the members of the classes they claim. */

val standaloneActivityExploration: Set = set("standaloneActivityExploration"):
  purpose(exploratory)
  bind(caller -> driven, worker -> driven)
  machine(activityProtocol)
  cover(rows | results | classMembers)
  budget(four)

// authoring: composition

/* The activity and its worker
 *
 * The protocol machine's worker stop is a stutter row: the activity cannot see the worker of its
 * task queue. Composed with that worker, the stop is the worker's own phase change and every
 * attempt start is the worker serving, so a start has a row only while the worker polls. */

/** The caller's view of the worker: it stops and it serves, and never resumes (see the Nexus
  * Model for why a resume would admit a stop, a resume and then a start). */
val activityWorker: Machine[WorkerState, ?, ?] = Worker.polling restrict (workerStop, serve)

final case class StandaloneActivityState(activity: ProtocolState, worker: WorkerState) derives Finite, CanEqual

object standaloneActivity extends Compose[StandaloneActivityState]("standaloneActivity"):
  val activity = member("activity", activityProtocol)(_.activity)
  val worker = member("worker", activityWorker)(_.worker)
  sync(workerStop, activity(workerStop) || worker(workerStop))
  sync(attemptStart, activity(attemptStart) || worker(serve))
  starts(activity at Phase.unstarted, worker at Worker.Phase.polling)
  ends(activity at Phase.completed, activity at Phase.failed, activity at Phase.canceled,
    activity at Phase.terminated, activity at Phase.timedOut)

/* Every attempt start leaves the worker polling: no worker picks a task up while stopped. */
val startedByPollingWorker: Property[StandaloneActivityState] =
  property("startedByPollingWorker")(standaloneActivity) when attemptStart holds: step =>
    step.state.worker.phase == Worker.Phase.polling

/* A polling worker picks the first attempt up and fails it retryably; the backoff fires; the
 * worker then stops, so the retry is never picked up and the schedule-to-start deadline settles
 * the activity. The path performs `attemptStart` once while polling, so the claim is exercised,
 * not vacuous. */
val stoppedBeforeRetry: Scenario[StandaloneActivityState] =
  scenario("stoppedBeforeRetry")(standaloneActivity) starts (standaloneActivity.activity at Phase.unstarted) actions (
    standaloneActivity.activity(start)(unset, expires, unset), attemptStart,
    standaloneActivity.activity(attemptResult)(AttemptResult.failed(true)),
    standaloneActivity.activity(backoff), workerStop, standaloneActivity.activity(scheduleToStart))

val stoppedWorkerStartsNothing =
  query("stoppedWorkerStartsNothing") verify startedByPollingWorker in stoppedBeforeRetry limits six

// authoring: end
