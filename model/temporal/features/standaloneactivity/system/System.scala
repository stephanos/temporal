/* The standalone activity's System: how the server gets there (fn-126 decision 16). The level's own
 * file holds the protocol machine, ActivityProtocol, whose refinement says what the Product reads of
 * it; ActivityWorker, the worker of its task queue; and StandaloneActivity, the protocol with that
 * worker. Beside it, one file per subject: Record.scala, history's record of the activity, and
 * WithTaskQueue.scala, that record composed with the shared task queue.
 */
package temporal
package features.standaloneactivity
package system

import scala.annotation.unused
import umpire.*
import umpire.realize.Reason
import temporal.capabilities.{given, *}
import temporal.realize.{inconclusive, satisfied}
import shared.Bounds.{four, three}
import shared.worker.{worker as process, Phase as WorkerPhase, State as WorkerState}
import product.ActivityProduct
import ActivityFamily.given
import Timeout.expires

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
