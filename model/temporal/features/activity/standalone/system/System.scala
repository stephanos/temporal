// The standalone activity's System: how the server gets there (fn-126 decision 16). The level's own
// file holds the System machine, ActivitySystem, whose refinement says what the Product reads of
// it; ActivityWorker, the worker of its task queue; and StandaloneActivity, the System with that
// worker. Beside it, one file per subject: Record.scala, history's record of the activity, and
// WithTaskQueue.scala, that record composed with the shared task queue.
package temporal
package features.activity
package standalone
package system

import scala.annotation.unused
import umpire.*
import umpire.outcomes.{Outcome, Rejection}
import umpire.realize.Reason
import temporal.capabilities.*
import temporal.realize.{inconclusive, satisfied}
import shared.Bounds.{four, three}
import shared.worker.{worker as process, Phase as WorkerPhase, State as WorkerState}
import product.ActivityProduct
import Timeout.expires

// It begins before the activity exists, so unstarted is one phase.
enum Phase derives Finite:
  case unstarted
  case scheduled extends Phase, Waiting
  case backingOff extends Phase, Retrying
  case started extends Phase, Held
  case paused extends Phase, Suspended
  case pauseRequested extends Phase, Held
  case cancelRequested extends Phase, Held
  case completed extends Phase, Succeeded
  case failed extends Phase, Failed
  case canceled extends Phase, Canceled
  case terminated extends Phase, Terminated
  case timedOut extends Phase, TimedOut

// 12 phases, 3 attempt counts and 3 deadline flags: 288 states.
final case class State(
    phase: Phase,
    attempts: UpTo[2],
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout
) derives Finite

// What the System machine records; `attemptCount` is named after its observation.
enum Fact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested
  case statusCompleted, statusFailed, statusCanceled, statusTerminated
  case statusTimedOut(timeoutType: TimeoutType)
  case attemptCount

// The System machine and its worker, as the standalone activity composition holds them.
final case class StandaloneActivityState(activity: State, worker: WorkerState)

// ### The System machine adds the retry, the pause request, the timers and the attempt count. It
// begins before the activity exists, so unstarted is a phase and the start sets the deadlines.

object ActivitySystem extends Machine[State, Outcome, Fact], Phased[State, Phase](_.phase):
  import Phase.*

  // Where every path begins: before the activity exists, with every deadline at its first value.
  val init = system.State(
    phase = Phase.unstarted,
    attempts = UpTo(0),
    scheduleToClose = Timeout.unset,
    scheduleToStart = Timeout.unset,
    startToClose = Timeout.unset
  )

  // A timeout is confirmed by the one status observation, whichever deadline fired.
  val evidence: PartialFunction[Fact, String] = {
    case Fact.statusTimedOut(_) => "statusTimedOut"
    case Fact.attemptCount      => attemptCount.name
  }

  // The System's status sets and its attempt count's bound.
  object states:
    // Bounds the attempt count, as the type of `State.attempts` does.
    val attemptBound = 2

    def saturatingSucc(a: UpTo[2]): UpTo[2] = UpTo((a + 1).min(attemptBound))

  // The System refines the product: what each of its states reads as there.
  object refinement extends Refinement(ActivityProduct):
    // Unstarted and backing off read as scheduled. A pause request reads as started: the worker
    // still holds the attempt, its every answer is a product row from started, and the request
    // stutters.
    def toProduct(s: State): product.State = s.phase match
      case Phase.unstarted | Phase.scheduled | Phase.backingOff =>
        product.State(product.Phase.scheduled)
      case Phase.started | Phase.pauseRequested => product.State(product.Phase.started)
      case Phase.paused                         => product.State(product.Phase.paused)
      case Phase.cancelRequested                => product.State(product.Phase.cancelRequested)
      case Phase.completed                      => product.State(product.Phase.completed)
      case Phase.failed                         => product.State(product.Phase.failed)
      case Phase.canceled                       => product.State(product.Phase.canceled)
      case Phase.terminated                     => product.State(product.Phase.terminated)
      case Phase.timedOut                       => product.State(product.Phase.timedOut)

    // The backoff timer records nothing a Run can read.
    val unobservable = List(timers.backoff)

  object effects:
    import Fact.*

    // The start creates the activity, with the deadlines its inputs set.
    def schedule(
        @unused s: State,
        scheduleToClose: Timeout,
        scheduleToStart: Timeout,
        startToClose: Timeout
    ) =
      enter(
        system.State(
          phase = scheduled,
          attempts = UpTo(0),
          scheduleToClose = scheduleToClose,
          scheduleToStart = scheduleToStart,
          startToClose = startToClose
        ),
        statusScheduled
      )

    // The worker's poll takes the attempt and raises the count the client reads back.
    def startAttempt(s: State) =
      enter(
        s.copy(phase = started, attempts = states.saturatingSucc(s.attempts)),
        statusStarted,
        Fact.attemptCount
      )

    def complete(s: State) = enter(s.copy(phase = completed), statusCompleted)

    def fail(s: State) = enter(s.copy(phase = failed), statusFailed)

    // A retryable failure backs a started attempt off: read as scheduled again, one attempt higher.
    def backOff(s: State) =
      enter(s.copy(phase = backingOff), statusScheduled, Fact.attemptCount)
        .because("a retryable failure backs off; the client reads scheduled again")

    def cancel(s: State) = enter(s.copy(phase = canceled), statusCanceled)

    def pause(s: State) = enter(s.copy(phase = paused), statusPaused)

    // A pause of a held attempt is a request the worker learns of on its next heartbeat, so it is its
    // own phase the client reads as paused.
    def requestPause(s: State) =
      enter(s.copy(phase = pauseRequested), statusPaused)
        .because("the worker learns of the pause on its next heartbeat")

    def resume(s: State) = enter(s.copy(phase = scheduled), statusScheduled)

    // An unpause before the worker yields: it keeps the attempt it holds.
    def withdrawPause(s: State) = enter(s.copy(phase = started), statusStarted)

    def requestCancel(s: State) =
      enter(s.copy(phase = cancelRequested), statusCancelRequested)

    def terminate(s: State) = enter(s.copy(phase = terminated), statusTerminated)

    // Keeps the state and records nothing; the next step's evidence confirms it, a Known Gap.
    def keep(s: State) = stay(s)

    // The backoff timer. A retry writes nothing the client can read.
    def retry(s: State) = enter(s.copy(phase = scheduled))

    def timeOut(s: State, t: TimeoutType) =
      enter(s.copy(phase = timedOut), statusTimedOut(t))

  object rules extends Rules:
    from(client) {
      import client.*

      on(start) {
        when(Phase.unstarted) ~> effects.schedule
      }

      // A control on an activity that is over is not found; an unstarted one has no control. A pause
      // is disabled in paused and pauseRequested (already paused, or asked to be) and cancelRequested
      // (a cancel request is not pausable), an unpause in scheduled, backingOff, started and
      // cancelRequested (not paused): the server answers FailedPrecondition ("activity is in
      // non-pausable state", "... non-unpausable state", chasm/lib/activity/operator_commands.go), and
      // a rejecting row would add rows to the table, so they stay disabled until the behavior freeze
      // lifts.
      on(pause, unpause, requestCancel, terminate) {
        when[Closed] ~> rejects(Rejection.notFound)
      }
      on(pause) {
        when(scheduled, backingOff) ~> effects.pause
        when(started) ~> effects.requestPause
      }
      on(unpause) {
        when(paused) ~> effects.resume
        when(pauseRequested) ~> effects.withdrawPause
      }
      on(requestCancel) {
        when[Live] ~> effects.requestCancel
      }
      on(terminate) {
        when[Live] ~> effects.terminate
      }
    }

    from(temporal.features.activity.standalone.worker) {
      import temporal.features.activity.standalone.worker.*

      on(poll) {
        when(scheduled) ~> effects.startAttempt
      }

      // A worker's answer settles the attempt it holds. A retryable failure backs a started attempt
      // off, settles a cancel-requested one as canceled and lands a pause-requested one in paused
      // (TransitionAttemptFailedWhilePauseRequested). A canceled answer needs a cancel request.
      // Where a worker holds the attempt: what start-to-close covers and a worker's answer settles.
      on(respondCompleted) {
        when[Held] ~> effects.complete
      }
      on(respondFailed(Failure.fatal)) {
        when[Held] ~> effects.fail
      }

      on(respondFailed(Failure.retryable)) {
        when(started) ~> effects.backOff
        when(cancelRequested) ~> effects.cancel
        when(pauseRequested) ~> effects.pause
      }
      on(respondCanceled) {
        when(cancelRequested) ~> effects.cancel
      }
    }

    from(process) {
      import process.*

      on(stop) {
        always ~> effects.keep
      }
    }

    from(timers) {
      import timers.*

      on(backoff) {
        when(backingOff) ~> effects.retry
      }
    }

    from(deadline) {
      import deadline.*

      // Started and not over: the phases a deadline can fire in.
      on(scheduleToClose) {
        when[Live].where(_.scheduleToClose == Timeout.expires) ~> (effects.timeOut(
          _,
          TimeoutType.scheduleToClose
        ))
      }
      // Waiting for a worker: the phases before an attempt is held, which schedule-to-start covers.
      on(scheduleToStart) {
        when[Waiting].where(_.scheduleToStart == Timeout.expires) ~> (effects.timeOut(
          _,
          TimeoutType.scheduleToStart
        ))
      }
      on(startToClose) {
        when[Held]
          .where(_.startToClose == Timeout.expires) ~> (effects.timeOut(
          _,
          TimeoutType.startToClose
        ))
      }
    }

  // What the System promises of its own: the settlement claims. The cross-entity claim of the
  // activity and its worker is the composition's, and the history record's are Record.scala's.
  object properties:
    val completes =
      property when worker.respondCompleted holds { s =>
        s.state.phase == Phase.completed && s.records(Fact.statusCompleted)
      }

    val nonRetryableFails =
      property when worker.respondFailed(Failure.fatal) holds { s =>
        s.state.phase == Phase.failed && s.records(Fact.statusFailed)
      }

    // Completed on the second attempt of an activity with no deadline set.
    val completedOnRetry =
      system.State(
        phase = Phase.completed,
        attempts = UpTo(states.attemptBound),
        scheduleToClose = Timeout.unset,
        scheduleToStart = Timeout.unset,
        startToClose = Timeout.unset
      )

    // The attempt count saturates at `states.attemptBound`, so the claim is bounded by it: a completion on
    // any later attempt than the second reads as this one.
    val retryCompletes =
      property when worker.respondCompleted holds { s =>
        s.state == completedOnRetry && s.records(Fact.statusCompleted)
      }

    val cancelRequestedWhileStarted =
      property when client.requestCancel holds { s =>
        s.state.phase == Phase.cancelRequested && s.records(Fact.statusCancelRequested)
      }

    val canceledByWorker =
      property when worker.respondCanceled holds { s =>
        s.state.phase == Phase.canceled && s.records(Fact.statusCanceled)
      }

    val terminated = property when client.terminate holds { s =>
      s.state.phase == Phase.terminated && s.records(Fact.statusTerminated)
    }

    // Each deadline times the activity out and the status records which it was. With both
    // schedule-to-start and schedule-to-close set and no attempt started, either may fire first.
    val scheduleToStartFires = property when deadline.scheduleToStart holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(Fact.statusTimedOut(TimeoutType.scheduleToStart))
    }

    val scheduleToCloseFires = property when deadline.scheduleToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(Fact.statusTimedOut(TimeoutType.scheduleToClose))
    }

    val startToCloseFires = property when deadline.startToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(Fact.statusTimedOut(TimeoutType.startToClose))
    }

  // What the System machine is, as its capability Properties read it, which a find through its
  // realization asks: a terminate settles it, a cancel request is recorded, and
  // DescribeActivityExecution reports its status by `activityStatus`. Each Property's find starts the
  // activity and stops the worker before the control, so no attempt is in flight when it lands, as
  // `terminatedWhileScheduled` does. A Run explains an unobserved control of an activity that is over
  // too, which answers notFound and records nothing, so the claim's explanations disagree. It reads
  // the realization, which reads this machine, so it waits in a section, which initializes on its
  // first use.
  object capabilities extends Capabilities:
    val terminable: Capability = Terminable(
      terminate = client.terminate,
      settled = Fact.statusTerminated,
      reach = Seq(client.start(), process.stop),
      expect = inconclusive(Reason.explanationsDisagree)
    )
    val cancelable: Capability = Cancelable(
      requestCancel = client.requestCancel,
      requested = Fact.statusCancelRequested,
      reach = Seq(client.start(), process.stop),
      expect = inconclusive(Reason.explanationsDisagree)
    )
    val describable: Capability = Describable(statusTable = activityStatus)

  // The paths, then one functional Query per side effect that settles the activity, and the Queries
  // that carry the other promises into the lifted Model. Each path starts before the activity exists;
  // a start that sets no deadline is `start()`, each input at `unset`. A path one Query takes is
  // written in it.
  object queries:
    capabilities.bound(three)

    val cancelRequestedThenCanceled = scenario.actions(
      client.start(),
      worker.poll,
      client.requestCancel,
      worker.respondCanceled
    )
    val completed = scenario.actions(
      client.start(),
      worker.poll,
      worker.respondCompleted
    )
    val nonRetryable = scenario.actions(
      client.start(),
      worker.poll,
      worker.respondFailed(Failure.fatal)
    )
    val retriedThenCompleted = scenario.actions(
      client.start(),
      worker.poll,
      worker.respondFailed(Failure.retryable),
      timers.backoff,
      worker.poll,
      worker.respondCompleted
    )
    val terminatedWhileScheduled = scenario.actions(
      client.start(),
      process.stop,
      client.terminate
    )
    val pausedThenCompleted = scenario.actions(
      client.start(),
      client.pause,
      client.unpause,
      worker.poll,
      worker.respondCompleted
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

    // Asks the one Property no functional Query asks, over the path that takes a cancel request.
    val cancelRequest =
      query find properties.cancelRequestedWhileStarted in cancelRequestedThenCanceled limits
        four

// ### The worker of the activity's task queue, as the activity sees it: its stop and its serving.

object ActivityWorker extends Derived(shared.worker.Polling.restrict(process.stop, process.serve))

// ### With the worker of its task queue, the stop is the worker's own phase change and every
// attempt start is the worker serving, so an attempt has a row only while the worker polls.

object StandaloneActivity
    extends Composition[StandaloneActivityState](
      _.activity -> ActivitySystem,
      _.worker -> ActivityWorker
    ),
      Phased[StandaloneActivityState, Phase](_.activity.phase):
  object syncs extends Syncs:
    sync(_.activity -> process.stop, _.worker -> process.stop)
    sync(_.activity -> worker.poll, _.worker -> process.serve)

  object properties:
    // The cross-entity claim: no stopped worker starts an attempt.
    val startedByPollingWorker = property
      .whenAction(synced(_.activity -> worker.poll))
      .holds(_.state.worker.phase == WorkerPhase.polling)

  object queries:
    // The cross-entity claim, over the path on which a stopped worker never takes the retry: the
    // first attempt fails retryably and backs off, then the worker stops, so the retry is never
    // dispatched; the attempt start makes the verification exercise the claim. The start is stated:
    // a default would take the worker's from shared/worker/Worker.scala; fn-115's golden compares
    // positions.
    val stoppedBeforeRetry = scenario
      .starts(StandaloneActivityState(ActivitySystem.init, WorkerState(WorkerPhase.polling)))
      .actions(
        own(_.activity, client.start(scheduleToStart := expires)),
        synced(_.activity -> worker.poll),
        own(_.activity, worker.respondFailed(Failure.retryable)),
        own(_.activity, timers.backoff),
        synced(_.activity -> process.stop),
        own(_.activity, deadline.scheduleToStart)
      )
    val stoppedWorkerStartsNothing =
      query verify properties.startedByPollingWorker in stoppedBeforeRetry limits six
