/* The standalone activity Model: one activity started directly through StartActivityExecution, with
 * no workflow around it, grounded in chasm/lib/activity/statemachine.go. The product machine says
 * what DescribeActivityExecution reports, the protocol how the server gets there. No history
 * event is written, so every evidence line names an observation: a status read through
 * DescribeActivityExecution or a result read through PollActivityExecution. Reset is deferred, like
 * cancellation in the Nexus caller Model, and the heartbeat timeout is not modeled.
 *
 * Read top to bottom: the types; the signature (the activity and its inputs; the caller and its
 * actions, the worker's actions on the activity, its timers and deadlines; and the bounds); then
 * one object per machine, each before the machines that use it -- ActivityProduct, the product
 * machine; ActivityProtocol, the protocol machine that refines it; ActivityWorker, the worker of its
 * task queue; StandaloneActivity, the protocol with that worker -- and last exports, its IR files.
 * A machine object reads its header (entity, init, end, evidence), then its sections in order:
 * states, refinement, effects, monitors, rules, properties, implements and queries. A composition
 * reads end, then states, syncs, properties, implements and queries. Realization.scala realizes it;
 * record/ holds the system contract and withTaskQueue/ the contract composed with the shared task
 * queue.
 */
package temporal
package features.standaloneactivity

import scala.annotation.unused
import umpire.*
import umpire.realize.Reason
import temporal.capabilities.{given, *}
import temporal.realize.{inconclusive, satisfied}
import shared.Bounds.{four, three}
import shared.worker.{worker as process, Phase as WorkerPhase, State as WorkerState}
import io.temporal.api.workflowservice.v1.*
import ActivityFamily.given
import Timeout.expires

// Moved from temporal.standaloneactivity; the pin keeps its Definition IDs and type names.
given DefinitionScope = DefinitionScope("temporal.standaloneactivity.Model$package$")

/** The family of the activity's machines; the system contract takes `SystemFamily`. */
object ActivityFamily:
  given family: Family = Family("temporal.activity.standalone")

/** The system contract's family; the shared task queue, first written there, keeps it too. */
object SystemFamily:
  given family: Family = Family("temporal.activity.standalone.system")

// ### Types

/** Whether the start request sets a deadline. */
enum Timeout derives Finite:
  case unset, expires

/** The worker's answer; failed(retryable) is two classes, like ApplicationFailure's flag. */
enum AttemptResult derives Finite:
  case completed
  case failed(retryable: Boolean)
  case canceled

enum Control derives Finite:
  case pause, unpause, requestCancel, terminate

/** A step's outcome, shared by both machines by name. */
enum Outcome derives Finite:
  case accepted, notFound

/** The product machine's phases: what DescribeActivityExecution shows. */
enum ProductPhase derives Finite:
  case scheduled, started, paused, cancelRequested
  case completed, failed, canceled, terminated, timedOut

final case class ProductState(phase: ProductPhase) derives Finite

enum ProductFact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested
  case statusCompleted, statusFailed, statusCanceled, statusTerminated, statusTimedOut

type ProductStep = Step[ProductState, Outcome, ProductFact]

/** The protocol machine's phases. It begins before the activity exists, so unstarted is one. */
enum Phase derives Finite:
  case unstarted, scheduled, backingOff, started, paused, pauseRequested, cancelRequested
  case completed, failed, canceled, terminated, timedOut

/** Which deadline fired. */
enum TimeoutType derives Finite:
  case scheduleToClose, scheduleToStart, startToClose

/**
 * 12 phases, 3 attempt counts (`0..ActivityProtocol.attemptBound`) and 3 deadline flags: 288
 * states.
 */
final case class ProtocolState(
    phase: Phase,
    attempts: UpTo[2],
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout
) derives Finite

/** What the protocol machine records; `attemptCount` is named after its observation. */
enum ProtocolFact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested
  case statusCompleted, statusFailed, statusCanceled, statusTerminated
  case statusTimedOut(timeoutType: TimeoutType)
  case attemptCount

type ProtocolStep = Step[ProtocolState, Outcome, ProtocolFact]

final case class StandaloneActivityState(activity: ProtocolState, worker: WorkerState)

// ### Signature

/** Named by the id the caller chose: every read carries it, so no run id or event id is needed. */
val activity = Entity(key = "activityId")

// The start's inputs, which the deadline timers no longer collide with, and the worker's answer.
val scheduleToClose = input[Timeout]
val scheduleToStart = input[Timeout]
val startToClose = input[Timeout]
val result = input[AttemptResult]

/** The control's input, apart because inside `caller` its name is the control action. */
object Inputs:
  val control = input[Control]

// Who acts, and on what. Actor and section objects are transparent to Definition IDs, so every
// action keeps the ID the file's pin gives it.

/** The caller starts and controls the activity. */
object caller extends Actor:
  val start = action(this)
    .input(scheduleToClose)
    .input(scheduleToStart)
    .input(startToClose)
    .creates(activity)
    .schema[StartActivityExecutionRequest]

  // The four controls are one action because they share a result: on an activity that is over, a
  // control is not found. The result text is metadata of the action, not a domain a state holds.
  val control = action(this)
    .on(activity)
    .input(Inputs.control)
    .schema[PauseActivityExecutionRequest]
    .schema[UnpauseActivityExecutionRequest]
    .schema[RequestCancelActivityExecutionRequest]
    .schema[TerminateActivityExecutionRequest]
    .results("Delivery")

/**
 * The shared worker party's actions on this activity: its poll receives the task for the current
 * attempt, and its answer settles it. The worker's stop is the party's own action,
 * `process.workerStop`: nothing it records names the activity, so the activity's machines keep
 * their state.
 */
object worker extends Section:
  val attemptStart = action(process).on(activity).schema[PollActivityTaskQueueResponse]

  val attemptResult = action(process)
    .on(activity)
    .input(result)
    .schema[RespondActivityTaskCompletedRequest]
    .schema[RespondActivityTaskFailedRequest]
    .schema[RespondActivityTaskCanceledRequest]
    .example(AttemptResult.failed(false), "ApplicationFailureNonRetryable")
    .example(AttemptResult.failed(true), "ApplicationFailureRetryable")

/** One of the activity's deadlines firing, as the product machine sees it, and the backoff. */
object timers extends Section:
  val timeout = timer
  val backoff = timer

/** The protocol's three deadlines, each armed by the start's input of its name. */
object deadline extends Section:
  val scheduleToClose = timer
  val scheduleToStart = timer
  val startToClose = timer

// A retry shows the caller only the attempt count DescribeActivityExecution reports. The statuses
// observe one status field; whether a catalog tells them apart is left to the realization.
val attemptCount = Observation(on = activity, read = "attempt")

given Ok[Outcome] = Ok(Outcome.accepted)

// The bounds of these Queries and the system contract's, beside three and four (shared.Bounds).
val five = Limits(steps = 5, actions = 5, search = 65536)
val six = Limits(steps = 6, actions = 6, search = 262144)
val eight = Limits(steps = 8, actions = 8, search = 262144)

// ### The product machine: what DescribeActivityExecution shows, with no account of how. A retry
// reads as scheduled again, a pause of a running attempt as started until the worker yields.

/** Every status the product machine records is confirmed by the status observation of its name. */
object ActivityProduct extends Machine[ProductState, Outcome, ProductFact]:
  import ProductPhase.*

  val entity = activity
  val init = ProductState(scheduled)
  def end(s: State) = states.over(s)

  /** The product's status sets and the constant its capabilities cite. */
  object states extends Section:
    def phase(s: State) = s.phase

    def terminal(p: ProductPhase) = p.in(completed, failed, canceled, terminated, timedOut)

    /** A path ends where the activity is over: `end` reads this named predicate. */
    def over(s: State) = terminal(s.phase)

    /** A paused activity, which no worker is given. */
    def paused(s: State) = s.phase == ProductPhase.paused

    def running(s: State) = s.phase == started

    /** Where a worker holds the attempt, so its answer settles the activity. */
    def held(s: State) = s.phase.in(started, cancelRequested)

    /** Where a pause takes effect: before an attempt starts or while one runs. */
    def pausable(s: State) = s.phase.in(scheduled, started)

    /**
     * The server code that answers a control of an activity that is over NotFound
     * (activity.go:106). Only the lifter reads a citation.
     */
    val notFoundCode = "chasm/lib/activity/activity.go"

  object effects extends Section:
    import ProductFact.*

    def startAttempt(s: State) = enter(s.copy(phase = started), statusStarted)

    def complete(s: State) = enter(s.copy(phase = completed), statusCompleted)

    def fail(s: State) = enter(s.copy(phase = failed), statusFailed)

    /**
     * Unlike the Nexus caller, a retryable failure reads SCHEDULED again with a higher attempt count
     * (TransitionRescheduled); the protocol adds the backoff.
     */
    def retry(s: State) = enter(s.copy(phase = scheduled), statusScheduled)

    def cancel(s: State) = enter(s.copy(phase = canceled), statusCanceled)

    /** A control on an activity that is over is not found. */
    def notFound(s: State): List[ProductStep] = List(Step(Outcome.notFound, s))

    def pause(s: State) = enter(s.copy(phase = ProductPhase.paused), statusPaused)

    def resume(s: State) = enter(s.copy(phase = scheduled), statusScheduled)

    def requestCancel(s: State) =
      enter(s.copy(phase = cancelRequested), statusCancelRequested)

    def terminate(s: State) = enter(s.copy(phase = terminated), statusTerminated)

    /** One of the activity's deadlines firing. Which deadline is the protocol's account of how. */
    def timeOut(s: State) = enter(s.copy(phase = timedOut), statusTimedOut)

  object rules extends Rules(_.phase):
    in(scheduled)(worker.attemptStart ~> effects.startAttempt)

    // A worker's answer settles an attempt it holds. A retryable failure is retried, or canceled
    // under a cancel request; a canceled answer settles only an activity whose cancellation was
    // requested.
    when(s => states.held(s)) {
      worker.attemptResult(AttemptResult.completed) ~> effects.complete
      worker.attemptResult(AttemptResult.failed(false)) ~> effects.fail
    }
    in(started)(worker.attemptResult(AttemptResult.failed(true)) ~> effects.retry)
    in(cancelRequested) {
      worker.attemptResult(AttemptResult.failed(true)) ~> effects.cancel
      worker.attemptResult(AttemptResult.canceled) ~> effects.cancel
    }

    // A control on an activity that is over is not found. A pause of a paused or cancel-requested
    // activity, or an unpause of one not paused, is FailedPrecondition; the protocol lists them.
    when(s => states.terminal(s.phase))(caller.control ~> effects.notFound)
    when(s => states.pausable(s))(caller.control(Control.pause) ~> effects.pause)
    when(s => states.paused(s))(caller.control(Control.unpause) ~> effects.resume)
    in(scheduled, started, paused, cancelRequested) {
      caller.control(Control.requestCancel) ~> effects.requestCancel
      caller.control(Control.terminate) ~> effects.terminate
    }

    // The worker stopping is a fault the Run records and the activity does not feel.
    disabled(process.workerStop)

    in(scheduled, started, paused, cancelRequested) {
      timers.timeout ~> effects.timeOut
    }

  /**
   * What the product machine is, as the laws of model/temporal/capabilities read it, and so the laws
   * it receives without listing them: it closes, and a control of an activity that is over is not
   * found, by the code `states.notFoundCode` cites; it pauses; and a worker's poll hands out its work. It
   * receives terminalStatesAreFinal, closedIsRejectedUniformly and pausedIsNotDispatched (pause with
   * poll), each `activityProduct.<law>`, read on the protocol through the map under the bound held
   * there.
   */
  object implements extends Section:
    val all = capabilities(limits = three)(
      Closable(
        status = states.phase,
        terminal = states.terminal,
        rejected = cited(Outcome.notFound, states.notFoundCode)
      ),
      Pausable(
        pause = caller.control(Control.pause),
        unpause = caller.control(Control.unpause),
        paused = states.paused
      ),
      Pollable(dispatch = worker.attemptStart, running = states.running)
    )

// ### The protocol machine adds the retry, the pause request, the timers and the attempt count. It
// begins before the activity exists, so unstarted is a phase and the start sets the deadlines.

object ActivityProtocol extends Machine[ProtocolState, Outcome, ProtocolFact]:
  import Phase.*

  val entity = activity

  /** Where every path begins: before the activity exists, with every deadline at its first value. */
  val init = ProtocolState(Phase.unstarted, UpTo(0), Timeout.unset, Timeout.unset, Timeout.unset)
  def end(s: State) = states.terminal(s.phase)

  /** A timeout is confirmed by the one status observation, whichever deadline fired. */
  val evidence: PartialFunction[ProtocolFact, String] = {
    case ProtocolFact.statusTimedOut(_) => "statusTimedOut"
    case ProtocolFact.attemptCount      => attemptCount.name
  }

  /** The protocol's status sets and its attempt count's bound. */
  object states extends Section:
    /** Bounds the attempt count, as the type of `ProtocolState.attempts` does. */
    val attemptBound = 2

    def terminal(p: Phase) = p.in(completed, failed, canceled, terminated, timedOut)

    /** Started and not over: the phases a deadline can fire in. */
    def live(p: Phase) =
      p.in(scheduled, backingOff, started, paused, pauseRequested, cancelRequested)

    /** Where a worker holds the attempt: what start-to-close covers and a worker's answer settles. */
    def held(p: Phase) = p.in(started, pauseRequested, cancelRequested)

    /** Waiting for a worker: the phases before an attempt is held, which schedule-to-start covers. */
    def waiting(p: Phase) = p.in(scheduled, backingOff)

    def saturatingSucc(a: UpTo[2]): UpTo[2] = UpTo((a + 1).min(attemptBound))

  /** The protocol refines the product: what each of its states reads as there. */
  object refinement extends Refinement(ActivityProduct):
    /**
     * Unstarted and backing off read as scheduled. A pause request reads as started: the worker
     * still holds the attempt, its every answer is a product row from started, and the request
     * stutters.
     */
    def toProduct(s: State): ProductState = s.phase match
      case Phase.unstarted | Phase.scheduled | Phase.backingOff =>
        ProductState(ProductPhase.scheduled)
      case Phase.started | Phase.pauseRequested => ProductState(ProductPhase.started)
      case Phase.paused                         => ProductState(ProductPhase.paused)
      case Phase.cancelRequested                => ProductState(ProductPhase.cancelRequested)
      case Phase.completed                      => ProductState(ProductPhase.completed)
      case Phase.failed                         => ProductState(ProductPhase.failed)
      case Phase.canceled                       => ProductState(ProductPhase.canceled)
      case Phase.terminated                     => ProductState(ProductPhase.terminated)
      case Phase.timedOut                       => ProductState(ProductPhase.timedOut)

    /** The backoff timer records nothing a Run can read. */
    val unobservable = List(timers.backoff)

  object effects extends Section:
    import ProtocolFact.*

    /** The start creates the activity, with the deadlines its inputs set. */
    def schedule(
        @unused s: State,
        scheduleToClose: Timeout,
        scheduleToStart: Timeout,
        startToClose: Timeout
    ) =
      enter(
        ProtocolState(scheduled, UpTo(0), scheduleToClose, scheduleToStart, startToClose),
        statusScheduled
      )

    /** The worker's poll takes the attempt and raises the count the caller reads back. */
    def startAttempt(s: State) =
      enter(
        s.copy(phase = started, attempts = states.saturatingSucc(s.attempts)),
        statusStarted,
        ProtocolFact.attemptCount
      )

    def complete(s: State) = enter(s.copy(phase = completed), statusCompleted)

    def fail(s: State) = enter(s.copy(phase = failed), statusFailed)

    /** A retryable failure backs a started attempt off: read as scheduled again, one attempt higher. */
    def backOff(s: State) =
      enter(s.copy(phase = backingOff), statusScheduled, ProtocolFact.attemptCount)
        .because("a retryable failure backs off; the caller reads scheduled again")

    def cancel(s: State) = enter(s.copy(phase = canceled), statusCanceled)

    /** A control on an activity that is over is not found. */
    def notFound(s: State): List[ProtocolStep] = List(Step(Outcome.notFound, s))

    def pause(s: State) = enter(s.copy(phase = paused), statusPaused)

    /**
     * A pause of a held attempt is a request the worker learns of on its next heartbeat, so it is its
     * own phase the caller reads as paused.
     */
    def requestPause(s: State) =
      enter(s.copy(phase = pauseRequested), statusPaused)
        .because("the worker learns of the pause on its next heartbeat")

    def resume(s: State) = enter(s.copy(phase = scheduled), statusScheduled)

    /** An unpause before the worker yields: it keeps the attempt it holds. */
    def withdrawPause(s: State) = enter(s.copy(phase = started), statusStarted)

    def requestCancel(s: State) =
      enter(s.copy(phase = cancelRequested), statusCancelRequested)

    def terminate(s: State) = enter(s.copy(phase = terminated), statusTerminated)

    /** Keeps the state and records nothing; the next step's evidence confirms it, a Known Gap. */
    def keep(s: State): List[ProtocolStep] = stay(s)

    /** The backoff timer. A retry writes nothing the caller can read. */
    def retry(s: State): List[ProtocolStep] = enter(s.copy(phase = scheduled))

    def timeOut(s: State, t: TimeoutType) =
      enter(s.copy(phase = timedOut), statusTimedOut(t))

  object rules extends Rules(_.phase):
    in(Phase.unstarted)(caller.start ~> effects.schedule)
    in(scheduled)(worker.attemptStart ~> effects.startAttempt)

    // A worker's answer settles the attempt it holds. A retryable failure backs a started attempt
    // off, settles a cancel-requested one as canceled and lands a pause-requested one in paused
    // (TransitionAttemptFailedWhilePauseRequested). A canceled answer needs a cancel request.
    when(s => states.held(s.phase)) {
      worker.attemptResult(AttemptResult.completed) ~> effects.complete
      worker.attemptResult(AttemptResult.failed(false)) ~> effects.fail
    }
    in(started)(worker.attemptResult(AttemptResult.failed(true)) ~> effects.backOff)
    in(cancelRequested) {
      worker.attemptResult(AttemptResult.failed(true)) ~> effects.cancel
      worker.attemptResult(AttemptResult.canceled) ~> effects.cancel
    }
    in(pauseRequested)(worker.attemptResult(AttemptResult.failed(true)) ~> effects.pause)

    // A control on an activity that is over is not found; an unstarted one has no control. A pause
    // is disabled in paused and pauseRequested (already paused, or asked to be) and cancelRequested
    // (a cancel request is not pausable), an unpause in scheduled, backingOff, started and
    // cancelRequested (not paused): the server answers FailedPrecondition ("activity is in
    // non-pausable state", "... non-unpausable state", chasm/lib/activity/operator_commands.go), and
    // a rejecting row would add rows to the table, so they stay disabled until the behavior freeze
    // lifts.
    when(s => states.terminal(s.phase))(caller.control ~> effects.notFound)
    in(scheduled, backingOff)(caller.control(Control.pause) ~> effects.pause)
    in(started)(caller.control(Control.pause) ~> effects.requestPause)
    in(paused)(caller.control(Control.unpause) ~> effects.resume)
    in(pauseRequested)(caller.control(Control.unpause) ~> effects.withdrawPause)
    when(s => states.live(s.phase)) {
      caller.control(Control.requestCancel) ~> effects.requestCancel
      caller.control(Control.terminate) ~> effects.terminate
    }

    when(_ => true)(process.workerStop ~> effects.keep)
    in(backingOff)(timers.backoff ~> effects.retry)

    when(s => states.live(s.phase) && s.scheduleToClose == Timeout.expires) {
      deadline.scheduleToClose ~> (s => effects.timeOut(s, TimeoutType.scheduleToClose))
    }
    when(s => states.waiting(s.phase) && s.scheduleToStart == Timeout.expires) {
      deadline.scheduleToStart ~> (s => effects.timeOut(s, TimeoutType.scheduleToStart))
    }
    when(s => states.held(s.phase) && s.startToClose == Timeout.expires) {
      deadline.startToClose ~> (s => effects.timeOut(s, TimeoutType.startToClose))
    }

  /**
   * What the protocol promises of its own: the settlement claims. The cross-entity claim of the
   * activity and its worker is the composition's, and the system contract's are record/'s.
   */
  object properties extends Section:
    val completes =
      property when worker.attemptResult(AttemptResult.completed) holds { s =>
        s.state.phase == Phase.completed && s.records(ProtocolFact.statusCompleted)
      }

    val nonRetryableFails =
      property when worker.attemptResult(AttemptResult.failed(false)) holds { s =>
        s.state.phase == Phase.failed && s.records(ProtocolFact.statusFailed)
      }

    /** Completed on the second attempt of an activity with no deadline set. */
    val completedOnRetry =
      ProtocolState(
        Phase.completed,
        UpTo(states.attemptBound),
        Timeout.unset,
        Timeout.unset,
        Timeout.unset
      )

    /**
     * The attempt count saturates at `states.attemptBound`, so the claim is bounded by it: a completion on
     * any later attempt than the second reads as this one.
     */
    val retryCompletes =
      property when worker.attemptResult(AttemptResult.completed) holds { s =>
        s.state == completedOnRetry && s.records(ProtocolFact.statusCompleted)
      }

    val cancelRequestedWhileStarted =
      property when caller.control(Control.requestCancel) holds { s =>
        s.state.phase == Phase.cancelRequested && s.records(ProtocolFact.statusCancelRequested)
      }

    val canceledByWorker =
      property when worker.attemptResult(AttemptResult.canceled) holds { s =>
        s.state.phase == Phase.canceled && s.records(ProtocolFact.statusCanceled)
      }

    val terminated = property when caller.control(Control.terminate) holds { s =>
      s.state.phase == Phase.terminated && s.records(ProtocolFact.statusTerminated)
    }

    // Each deadline times the activity out and the status records which it was. With both
    // schedule-to-start and schedule-to-close set and no attempt started, either may fire first.
    val scheduleToStartFires = property when deadline.scheduleToStart holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart))
    }

    val scheduleToCloseFires = property when deadline.scheduleToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(ProtocolFact.statusTimedOut(TimeoutType.scheduleToClose))
    }

    val startToCloseFires = property when deadline.startToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(ProtocolFact.statusTimedOut(TimeoutType.startToClose))
    }

  /**
   * What the protocol machine is, as the functional laws read it, which a find through its
   * realization asks: a terminate settles it, a cancel request is recorded, and
   * DescribeActivityExecution reports its status by `activityStatus`. Each law's find starts the
   * activity and stops the worker before the control, so no attempt is in flight when it lands, as
   * `terminatedWhileScheduled` does. A Run explains an unobserved control of an activity that is over
   * too, which answers notFound and records nothing, so the claim's explanations disagree. It reads
   * the realization, which reads this machine, so it waits in a section, which initializes on its
   * first use.
   */
  object implements extends Section:
    val all = capabilities(limits = three)(
      Terminable(
        terminate = caller.control(Control.terminate),
        settled = ProtocolFact.statusTerminated,
        reach = Seq(caller.start(), process.workerStop),
        expect = inconclusive(Reason.explanationsDisagree)
      ),
      Cancelable(
        requestCancel = caller.control(Control.requestCancel),
        requested = ProtocolFact.statusCancelRequested,
        reach = Seq(caller.start(), process.workerStop),
        expect = inconclusive(Reason.explanationsDisagree)
      ),
      Describable(status = ActivityRealization.activityStatus)
    )

  /**
   * The paths, then one functional Query per side effect that settles the activity, and the Queries
   * that carry the other promises into the lifted Model. Each path starts before the activity exists;
   * a start that sets no deadline is `start()`, each input at `unset`. A path one Query takes is
   * written in it.
   */
  object queries extends Section:
    val cancelRequestedThenCanceled = scenario.actions(
      caller.start(),
      worker.attemptStart,
      caller.control(Control.requestCancel),
      worker.attemptResult(AttemptResult.canceled)
    )

    // One Query per side effect that settles the activity, apart from the Properties they find.
    val completion =
      (query find properties.completes in scenario("completed").actions(
        caller.start(),
        worker.attemptStart,
        worker.attemptResult(AttemptResult.completed)
      ) limits three total 864).expect(satisfied)
    val nonRetryableFailure =
      (query find properties.nonRetryableFails in scenario("nonRetryable").actions(
        caller.start(),
        worker.attemptStart,
        worker.attemptResult(AttemptResult.failed(false))
      ) limits three total 864).expect(satisfied)
    val retry =
      (query find properties.retryCompletes in scenario("retriedThenCompleted").actions(
        caller.start(),
        worker.attemptStart,
        worker.attemptResult(AttemptResult.failed(true)),
        timers.backoff,
        worker.attemptStart,
        worker.attemptResult(AttemptResult.completed)
      ) limits six total 1728)
        .expect(inconclusive(Reason.explanationsDisagree))
    val cancel =
      query find properties.canceledByWorker in cancelRequestedThenCanceled limits four total 1152
    // The worker stops before the start, so no attempt is in flight when the caller terminates.
    val terminate =
      (query find properties.terminated in scenario("terminatedWhileScheduled").actions(
        caller.start(),
        process.workerStop,
        caller.control(Control.terminate)
      ) limits three total 864).expect(inconclusive(Reason.explanationsDisagree))
    val pauseResume =
      (query find properties.completes in scenario("pausedThenCompleted").actions(
        caller.start(),
        caller.control(Control.pause),
        caller.control(Control.unpause),
        worker.attemptStart,
        worker.attemptResult(AttemptResult.completed)
      ) limits six total 1440)
        .expect(satisfied)
    val scheduleToStartTimeout =
      (query find properties.scheduleToStartFires in scenario("scheduleToStartExpires").actions(
        caller.start(scheduleToStart := expires),
        process.workerStop,
        deadline.scheduleToStart
      ) limits three total 864).expect(inconclusive(Reason.neverEvaluated))
    val startToCloseTimeout =
      query find properties.startToCloseFires in scenario("startToCloseExpires").actions(
        caller.start(startToClose := expires),
        worker.attemptStart,
        deadline.startToClose
      ) limits three total 864

    val all = Vector(
      completion,
      nonRetryableFailure,
      retry,
      cancel,
      terminate,
      pauseResume,
      scheduleToStartTimeout,
      startToCloseTimeout
    )

    // The Queries that carry the other promises into the lifted Model, where Go's are compared.

    /** Asks the one Property no functional Query asks, over the path that takes a cancel request. */
    val cancelRequest =
      query find properties.cancelRequestedWhileStarted in cancelRequestedThenCanceled limits
        four total 1152

    /**
     * Neither deadline is ordered before the other: each firing is a trace of its own. Both
     * deadlines are set and no attempt started, so either may fire first.
     */
    val competingTimers = Vector(
      query("competingTimers.scheduleToStartFirst") find properties.scheduleToStartFires in
        scenario("bothDeadlinesStartFirst").actions(
          caller.start(scheduleToClose := expires, scheduleToStart := expires),
          deadline.scheduleToStart
        ) limits three total 576,
      query("competingTimers.scheduleToCloseFirst") find properties.scheduleToCloseFires in
        scenario("bothDeadlinesCloseFirst").actions(
          caller.start(scheduleToClose := expires, scheduleToStart := expires),
          deadline.scheduleToClose
        ) limits three total 576
    )

// ### The worker of the activity's task queue, as the activity sees it: its stop and its serving.

object ActivityWorker
    extends Derived(shared.worker.Polling.restrict(process.workerStop, process.serve))

// ### With the worker of its task queue, the stop is the worker's own phase change and every
// attempt start is the worker serving, so an attempt has a row only while the worker polls.

object StandaloneActivity
    extends Composition[StandaloneActivityState](
      _.activity -> ActivityProtocol,
      _.worker -> ActivityWorker
    ):
  def end(s: State) = ActivityProtocol.states.terminal(s.activity.phase)

  object syncs extends Syncs:
    sync(_.activity -> process.workerStop, _.worker -> process.workerStop)
    sync(_.activity -> worker.attemptStart, _.worker -> process.serve)

  object properties extends Section:
    /** The cross-entity claim: no stopped worker starts an attempt. */
    val startedByPollingWorker = property
      .whenAction(synced(_.activity -> worker.attemptStart))
      .holds(_.state.worker.phase == WorkerPhase.polling)

  object queries extends Section:
    /**
     * The cross-entity claim, over the path on which a stopped worker never takes the retry: the
     * first attempt fails retryably and backs off, then the worker stops, so the retry is never
     * dispatched; the attempt start makes the verification exercise the claim. The start is stated:
     * a default would take the worker's from shared/worker/Worker.scala; fn-115's golden compares
     * positions.
     */
    val stoppedWorkerStartsNothing =
      query verify properties.startedByPollingWorker in scenario("stoppedBeforeRetry")
        .starts(StandaloneActivityState(ActivityProtocol.init, WorkerState(WorkerPhase.polling)))
        .actions(
          own(_.activity, caller.start(scheduleToStart := expires)),
          synced(_.activity -> worker.attemptStart),
          own(_.activity, worker.attemptResult(AttemptResult.failed(true))),
          own(_.activity, timers.backoff),
          synced(_.activity -> process.workerStop),
          own(_.activity, deadline.scheduleToStart)
        ) limits six total 3456

// ### The checked-in IR files of the standalone activity Models (umpire.irFile).

object exports:
  // The activity Model. Its cross-entity Query, stoppedWorkerStartsNothing, carries the composition
  // and its claim.
  val activity = irFile("activity")(
    StandaloneActivity,
    ActivityProduct,
    ActivityProtocol.queries.all,
    ActivityProduct.implements.all,
    ActivityProtocol.implements.all,
    ActivityProtocol.queries.cancelRequest,
    StandaloneActivity.queries.stoppedWorkerStartsNothing,
    ActivityRealization.standalone
  )

  // Its system contract, the admission designs, and the shared task queue's providers it composes.
  // A composition no Query runs over is a root of its own.
  val activitySystem = irFile("activity-system")(
    record.CurrentAdmission.queries.currentQueries,
    record.StaleAdmission.queries.staleQueries,
    ActivityProtocol.queries.competingTimers,
    shared.taskqueue.MatchingQueue.queries.matchingQueueQueries,
    shared.taskqueue.ForgetfulQueue.queries.forgetfulQueueQueries,
    shared.taskqueue.VolatileQueue.queries.volatileQueueQueries,
    shared.taskqueue.LossyMatchingQueue.queries.lossyMatchingQueueQueries,
    shared.taskqueue.LossyMatchingQueue.queries.storageLossQuery,
    withTaskQueue.CurrentOverQueue.queries.currentOverQueueQueries,
    withTaskQueue.StaleOverQueue.queries.staleOverQueueQueries,
    withTaskQueue.CurrentOverMatching.queries.currentOverMatchingQueries,
    withTaskQueue.StaleOverMatching.queries.staleOverMatchingQueries,
    withTaskQueue.CurrentOverLossyMatching.queries.currentOverLossyMatchingQueries,
    withTaskQueue.CurrentOverForgetful,
    withTaskQueue.CurrentOverVolatile
  )

  // The held race a server is run through, and the realization that runs it. It is a Model of its
  // own, so the system contract's Queries are the ones its checkers were given.
  val activityRace = irFile("activity-race")(
    record.HeldAdmission.queries.heldStaleDelivery,
    ActivityRealization.heldDelivery,
    record.AdmissionResponseLoss.queries.lostAdmissionResponseQuery,
    ActivityRealization.lostAdmissionResponse
  )
