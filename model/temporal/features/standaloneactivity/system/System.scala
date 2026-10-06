/* The standalone activity's System: how the server gets there (fn-126 decision 16). The level's own
 * file holds the System machine, ActivitySystem, whose refinement says what the Product reads of
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
import Timeout.expires

// ### The System machine adds the retry, the pause request, the timers and the attempt count. It
// begins before the activity exists, so unstarted is a phase and the start sets the deadlines.

object ActivitySystem extends Machine[SystemState, Outcome, SystemFact]:
  import Phase.*

  /** Where every path begins: before the activity exists, with every deadline at its first value. */
  val init = SystemState(
    phase = Phase.unstarted,
    attempts = UpTo(0),
    scheduleToClose = Timeout.unset,
    scheduleToStart = Timeout.unset,
    startToClose = Timeout.unset
  )
  def end(s: State) = states.terminal(s.phase)

  /** A timeout is confirmed by the one status observation, whichever deadline fired. */
  val evidence: PartialFunction[SystemFact, String] = {
    case SystemFact.statusTimedOut(_) => "statusTimedOut"
    case SystemFact.attemptCount      => attemptCount.name
  }

  /** The protocol's status sets and its attempt count's bound. */
  object states:
    /** Bounds the attempt count, as the type of `SystemState.attempts` does. */
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

  object effects:
    import SystemFact.*

    /** The start creates the activity, with the deadlines its inputs set. */
    def schedule(
        @unused s: State,
        scheduleToClose: Timeout,
        scheduleToStart: Timeout,
        startToClose: Timeout
    ) =
      enter(
        SystemState(
          phase = scheduled,
          attempts = UpTo(0),
          scheduleToClose = scheduleToClose,
          scheduleToStart = scheduleToStart,
          startToClose = startToClose
        ),
        statusScheduled
      )

    /** The worker's poll takes the attempt and raises the count the client reads back. */
    def startAttempt(s: State) =
      enter(
        s.copy(phase = started, attempts = states.saturatingSucc(s.attempts)),
        statusStarted,
        SystemFact.attemptCount
      )

    def complete(s: State) = enter(s.copy(phase = completed), statusCompleted)

    def fail(s: State) = enter(s.copy(phase = failed), statusFailed)

    /** A retryable failure backs a started attempt off: read as scheduled again, one attempt higher. */
    def backOff(s: State) =
      enter(s.copy(phase = backingOff), statusScheduled, SystemFact.attemptCount)
        .because("a retryable failure backs off; the client reads scheduled again")

    def cancel(s: State) = enter(s.copy(phase = canceled), statusCanceled)

    /** A control on an activity that is over is not found. */
    def notFound(s: State) = reject(Outcome.notFound, s)

    def pause(s: State) = enter(s.copy(phase = paused), statusPaused)

    /**
     * A pause of a held attempt is a request the worker learns of on its next heartbeat, so it is its
     * own phase the client reads as paused.
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
    def keep(s: State) = stay(s)

    /** The backoff timer. A retry writes nothing the client can read. */
    def retry(s: State) = enter(s.copy(phase = scheduled))

    def timeOut(s: State, t: TimeoutType) =
      enter(s.copy(phase = timedOut), statusTimedOut(t))

  object rules extends Rules(_.phase):
    on(client.start)(in(Phase.unstarted) ~> effects.schedule)
    on(worker.poll)(in(scheduled) ~> effects.startAttempt)

    // A worker's answer settles the attempt it holds. A retryable failure backs a started attempt
    // off, settles a cancel-requested one as canceled and lands a pause-requested one in paused
    // (TransitionAttemptFailedWhilePauseRequested). A canceled answer needs a cancel request.
    on(worker.respond(AttemptResult.completed))(in(states.held) ~> effects.complete)
    on(worker.respond(AttemptResult.failed(false)))(in(states.held) ~> effects.fail)
    on(worker.respond(AttemptResult.failed(true))) {
      in(started) ~> effects.backOff
      in(cancelRequested) ~> effects.cancel
      in(pauseRequested) ~> effects.pause
    }
    on(worker.respond(AttemptResult.canceled))(in(cancelRequested) ~> effects.cancel)

    // A control on an activity that is over is not found; an unstarted one has no control. A pause
    // is disabled in paused and pauseRequested (already paused, or asked to be) and cancelRequested
    // (a cancel request is not pausable), an unpause in scheduled, backingOff, started and
    // cancelRequested (not paused): the server answers FailedPrecondition ("activity is in
    // non-pausable state", "... non-unpausable state", chasm/lib/activity/operator_commands.go), and
    // a rejecting row would add rows to the table, so they stay disabled until the behavior freeze
    // lifts.
    on(client.control)(in(states.terminal) ~> effects.notFound)
    on(client.control(Control.pause)) {
      in(scheduled, backingOff) ~> effects.pause
      in(started) ~> effects.requestPause
    }
    on(client.control(Control.unpause)) {
      in(paused) ~> effects.resume
      in(pauseRequested) ~> effects.withdrawPause
    }
    on(client.control(Control.requestCancel))(in(states.live) ~> effects.requestCancel)
    on(client.control(Control.terminate))(in(states.live) ~> effects.terminate)
    on(process.stop)(always ~> effects.keep)
    on(timers.backoff)(in(backingOff) ~> effects.retry)
    on(deadline.scheduleToClose) {
      in(states.live).where(_.scheduleToClose == Timeout.expires) ~> (effects.timeOut(
        _,
        TimeoutType.scheduleToClose
      ))
    }
    on(deadline.scheduleToStart) {
      in(states.waiting).where(_.scheduleToStart == Timeout.expires) ~> (effects.timeOut(
        _,
        TimeoutType.scheduleToStart
      ))
    }
    on(deadline.startToClose) {
      in(states.held)
        .where(_.startToClose == Timeout.expires) ~> (effects.timeOut(_, TimeoutType.startToClose))
    }

  /**
   * What the protocol promises of its own: the settlement claims. The cross-entity claim of the
   * activity and its worker is the composition's, and the history record's are Record.scala's.
   */
  object properties:
    val completes =
      property when worker.respond(AttemptResult.completed) holds { s =>
        s.state.phase == Phase.completed && s.records(SystemFact.statusCompleted)
      }

    val nonRetryableFails =
      property when worker.respond(AttemptResult.failed(false)) holds { s =>
        s.state.phase == Phase.failed && s.records(SystemFact.statusFailed)
      }

    /** Completed on the second attempt of an activity with no deadline set. */
    val completedOnRetry =
      SystemState(
        phase = Phase.completed,
        attempts = UpTo(states.attemptBound),
        scheduleToClose = Timeout.unset,
        scheduleToStart = Timeout.unset,
        startToClose = Timeout.unset
      )

    /**
     * The attempt count saturates at `states.attemptBound`, so the claim is bounded by it: a completion on
     * any later attempt than the second reads as this one.
     */
    val retryCompletes =
      property when worker.respond(AttemptResult.completed) holds { s =>
        s.state == completedOnRetry && s.records(SystemFact.statusCompleted)
      }

    val cancelRequestedWhileStarted =
      property when client.control(Control.requestCancel) holds { s =>
        s.state.phase == Phase.cancelRequested && s.records(SystemFact.statusCancelRequested)
      }

    val canceledByWorker =
      property when worker.respond(AttemptResult.canceled) holds { s =>
        s.state.phase == Phase.canceled && s.records(SystemFact.statusCanceled)
      }

    val terminated = property when client.control(Control.terminate) holds { s =>
      s.state.phase == Phase.terminated && s.records(SystemFact.statusTerminated)
    }

    // Each deadline times the activity out and the status records which it was. With both
    // schedule-to-start and schedule-to-close set and no attempt started, either may fire first.
    val scheduleToStartFires = property when deadline.scheduleToStart holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(SystemFact.statusTimedOut(TimeoutType.scheduleToStart))
    }

    val scheduleToCloseFires = property when deadline.scheduleToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(SystemFact.statusTimedOut(TimeoutType.scheduleToClose))
    }

    val startToCloseFires = property when deadline.startToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(SystemFact.statusTimedOut(TimeoutType.startToClose))
    }

  /**
   * What the System machine is, as the functional laws read it, which a find through its
   * realization asks: a terminate settles it, a cancel request is recorded, and
   * DescribeActivityExecution reports its status by `activityStatus`. Each law's find starts the
   * activity and stops the worker before the control, so no attempt is in flight when it lands, as
   * `terminatedWhileScheduled` does. A Run explains an unobserved control of an activity that is over
   * too, which answers notFound and records nothing, so the claim's explanations disagree. It reads
   * the realization, which reads this machine, so it waits in a section, which initializes on its
   * first use.
   */
  object implements
      extends Implements(limits = three)(
        Terminable(
          terminate = client.control(Control.terminate),
          settled = SystemFact.statusTerminated,
          reach = Seq(client.start(), process.stop),
          expect = inconclusive(Reason.explanationsDisagree)
        ),
        Cancelable(
          requestCancel = client.control(Control.requestCancel),
          requested = SystemFact.statusCancelRequested,
          reach = Seq(client.start(), process.stop),
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
  object queries:
    val cancelRequestedThenCanceled = scenario.actions(
      client.start(),
      worker.poll,
      client.control(Control.requestCancel),
      worker.respond(AttemptResult.canceled)
    )
    val completed = scenario.actions(
      client.start(),
      worker.poll,
      worker.respond(AttemptResult.completed)
    )
    val nonRetryable = scenario.actions(
      client.start(),
      worker.poll,
      worker.respond(AttemptResult.failed(false))
    )
    val retriedThenCompleted = scenario.actions(
      client.start(),
      worker.poll,
      worker.respond(AttemptResult.failed(true)),
      timers.backoff,
      worker.poll,
      worker.respond(AttemptResult.completed)
    )
    val terminatedWhileScheduled = scenario.actions(
      client.start(),
      process.stop,
      client.control(Control.terminate)
    )
    val pausedThenCompleted = scenario.actions(
      client.start(),
      client.control(Control.pause),
      client.control(Control.unpause),
      worker.poll,
      worker.respond(AttemptResult.completed)
    )
    val scheduleToStartExpires = scenario.actions(
      client.start(scheduleToStart := expires),
      process.stop,
      deadline.scheduleToStart
    )
    val startToCloseExpires = scenario.actions(
      client.start(startToClose := expires),
      worker.poll,
      deadline.startToClose
    )

    // One Query per side effect that settles the activity, apart from the Properties they find.
    val completion =
      (query find properties.completes in completed limits three).expect(satisfied)
    val nonRetryableFailure =
      (query find properties.nonRetryableFails in nonRetryable limits three).expect(satisfied)
    val retry =
      (query find properties.retryCompletes in retriedThenCompleted limits six)
        .expect(inconclusive(Reason.explanationsDisagree))
    val cancel =
      query find properties.canceledByWorker in cancelRequestedThenCanceled limits four
    // The worker stops before the start, so no attempt is in flight when the client terminates.
    val terminate =
      (query find properties.terminated in terminatedWhileScheduled limits three)
        .expect(inconclusive(Reason.explanationsDisagree))
    val pauseResume =
      (query find properties.completes in pausedThenCompleted limits six)
        .expect(satisfied)
    val scheduleToStartTimeout =
      (query find properties.scheduleToStartFires in scheduleToStartExpires limits three)
        .expect(inconclusive(Reason.neverEvaluated))
    val startToCloseTimeout =
      query find properties.startToCloseFires in startToCloseExpires limits three

    // The Query that carries the other promise into the lifted Model, where Go's are compared.

    /** Asks the one Property no functional Query asks, over the path that takes a cancel request. */
    val cancelRequest =
      query find properties.cancelRequestedWhileStarted in cancelRequestedThenCanceled limits
        four

// ### The worker of the activity's task queue, as the activity sees it: its stop and its serving.

object ActivityWorker
    extends Derived(shared.worker.Polling.restrict(process.stop, process.serve))

// ### With the worker of its task queue, the stop is the worker's own phase change and every
// attempt start is the worker serving, so an attempt has a row only while the worker polls.

object StandaloneActivity
    extends Composition[StandaloneActivityState](
      _.activity -> ActivitySystem,
      _.worker -> ActivityWorker
    ):
  def end(s: State) = ActivitySystem.states.terminal(s.activity.phase)

  object syncs extends Syncs:
    sync(_.activity -> process.stop, _.worker -> process.stop)
    sync(_.activity -> worker.poll, _.worker -> process.serve)

  object properties:
    /** The cross-entity claim: no stopped worker starts an attempt. */
    val startedByPollingWorker = property
      .whenAction(synced(_.activity -> worker.poll))
      .holds(_.state.worker.phase == WorkerPhase.polling)

  object queries:
    /**
     * The cross-entity claim, over the path on which a stopped worker never takes the retry: the
     * first attempt fails retryably and backs off, then the worker stops, so the retry is never
     * dispatched; the attempt start makes the verification exercise the claim. The start is stated:
     * a default would take the worker's from shared/worker/Worker.scala; fn-115's golden compares
     * positions.
     */
    val stoppedBeforeRetry = scenario
        .starts(StandaloneActivityState(ActivitySystem.init, WorkerState(WorkerPhase.polling)))
        .actions(
          own(_.activity, client.start(scheduleToStart := expires)),
          synced(_.activity -> worker.poll),
          own(_.activity, worker.respond(AttemptResult.failed(true))),
          own(_.activity, timers.backoff),
          synced(_.activity -> process.stop),
          own(_.activity, deadline.scheduleToStart)
        )
    val stoppedWorkerStartsNothing =
      query verify properties.startedByPollingWorker in stoppedBeforeRetry limits six
