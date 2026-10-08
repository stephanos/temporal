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
  case started extends Phase, Held
  case paused extends Phase, Suspended
  case pauseRequested extends Phase, Held
  case cancelRequested extends Phase, Held
  case completed extends Phase, Succeeded
  case failed extends Phase, Failed
  case canceled extends Phase, Canceled
  case terminated extends Phase, Terminated
  case timedOut extends Phase, TimedOut

// Whether a waiting attempt can be dispatched, independent of its status (model.go:239-244).
enum Dispatch derives Finite:
  case now, startDelay, backoff

// 11 phases, 3 dispatch values, 3 attempt counts, 4 deadline flags and 3 retry policies: 4752 states.
final case class State(
    phase: Phase,
    dispatch: Dispatch,
    attempts: UpTo[MaxAttempts.Bound],
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout,
    heartbeat: Timeout,
    maxAttempts: MaxAttempts
) derives Finite

// What the System machine records; `attemptCount` is named after its observation.
enum Fact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested
  case statusCompleted, statusFailed, statusCanceled, statusTerminated
  case statusTimedOut(timeoutType: TimeoutType)
  case attemptCount
  case heartbeatReceived, heartbeatTimedOut

// The System machine and its worker, as the standalone activity composition holds them.
final case class StandaloneActivityState(activity: State, worker: WorkerState)

// ### The System machine adds the retry, the pause request, the timers and the attempt count. It
// begins before the activity exists, so unstarted is a phase and the start sets the deadlines.

object ActivitySystem extends Machine[State, Outcome, Fact], Phased[State, Phase](_.phase):
  import Phase.*

  // Where every path begins: before the activity exists, with every deadline at its first value.
  val init = system.State(
    phase = Phase.unstarted,
    dispatch = Dispatch.now,
    attempts = UpTo(0),
    scheduleToClose = Timeout.unset,
    scheduleToStart = Timeout.unset,
    startToClose = Timeout.unset,
    heartbeat = Timeout.unset,
    maxAttempts = MaxAttempts.unlimited
  )

  // A timeout is confirmed by the one status observation, whichever deadline fired.
  val evidence: PartialFunction[Fact, String] = {
    case Fact.statusTimedOut(_) => "statusTimedOut"
    case Fact.attemptCount      => attemptCount.name
  }

  // The System's status sets and its attempt count's bound.
  object states:
    // Bounds the attempt count, as the type of `State.attempts` does.
    val attemptBound = MaxAttempts.bound

    def saturatingSucc(a: UpTo[MaxAttempts.Bound]): UpTo[MaxAttempts.Bound] =
      UpTo((a + 1).min(attemptBound))

    def retriesRemaining(s: State): Boolean = s.maxAttempts match
      case MaxAttempts.unlimited => true
      case MaxAttempts.one       => s.attempts < 1
      case MaxAttempts.two       => s.attempts < attemptBound

    def attemptCount(s: State): Int = s.attempts

    def maximumAttempts(s: State): Option[UpTo[MaxAttempts.Bound]] = s.maxAttempts match
      case MaxAttempts.unlimited => None
      case MaxAttempts.one       => Some(UpTo(1))
      case MaxAttempts.two       => Some(UpTo(MaxAttempts.bound))

    def pendingPause(s: State): Boolean = s.phase == pauseRequested
    def pendingCancel(s: State): Boolean = s.phase == cancelRequested

    def scheduleToCloseArmed(s: State): Boolean =
      s.scheduleToClose == Timeout.expires && s.dispatch != Dispatch.startDelay
    def scheduleToStartArmed(s: State): Boolean =
      s.scheduleToStart == Timeout.expires && s.dispatch == Dispatch.now
    def startToCloseArmed(s: State): Boolean = s.startToClose == Timeout.expires
    def heartbeatArmed(s: State): Boolean = s.heartbeat == Timeout.expires

    val timeoutFacts = Seq(
      Fact.statusTimedOut(TimeoutType.scheduleToClose),
      Fact.statusTimedOut(TimeoutType.scheduleToStart),
      Fact.statusTimedOut(TimeoutType.startToClose),
      Fact.statusTimedOut(TimeoutType.heartbeat)
    )

    def afterRetry(s: State, phase: Phase) = s.copy(phase = phase, dispatch = Dispatch.backoff)

    val nominalTimeWindow =
      "The configured time window is nominal, not proof that a real timer fired; the rule's guards select its applicability. Scheduling deadlines start after start delay; schedule-to-start also waits through retry backoff (chasm/lib/activity/model/model.go:300-310,312-335,355-376)."

  // The System refines the product: what each of its states reads as there.
  object refinement extends Refinement(ActivityProduct):
    // Every waiting state reads as scheduled, whatever its dispatch. A pause request reads as
    // started: the worker still holds the attempt, its every answer is a product row from started,
    // and the Product carries the pause observation without changing that state.
    def toProduct(s: State): product.State = s.phase match
      case Phase.unstarted | Phase.scheduled =>
        product.State(product.Phase.scheduled)
      case Phase.started | Phase.pauseRequested => product.State(product.Phase.started)
      case Phase.paused                         => product.State(product.Phase.paused)
      case Phase.cancelRequested                => product.State(product.Phase.cancelRequested)
      case Phase.completed                      => product.State(product.Phase.completed)
      case Phase.failed                         => product.State(product.Phase.failed)
      case Phase.canceled                       => product.State(product.Phase.canceled)
      case Phase.terminated                     => product.State(product.Phase.terminated)
      case Phase.timedOut                       => product.State(product.Phase.timedOut)

    def visible(f: Fact) = f match
      case Fact.statusScheduled | Fact.statusStarted | Fact.statusPaused |
          Fact.statusCancelRequested | Fact.statusCompleted | Fact.statusFailed |
          Fact.statusCanceled | Fact.statusTerminated | Fact.statusTimedOut(_) |
          Fact.heartbeatReceived | Fact.heartbeatTimedOut =>
        true
      case Fact.attemptCount => false

    // Dispatch-delay timers record nothing a Run can read.
    val unobservable = List(timers.startDelay, timers.backoff)

  object effects:
    import Fact.*

    // The start creates the activity, with the deadlines its inputs set.
    def schedule(
        @unused s: State,
        scheduleToClose: Timeout,
        scheduleToStart: Timeout,
        startToClose: Timeout,
        heartbeat: Timeout,
        startDelay: Timeout,
        maxAttempts: MaxAttempts
    ) =
      enter(
        system.State(
          phase = scheduled,
          dispatch = if startDelay == Timeout.expires then Dispatch.startDelay else Dispatch.now,
          attempts = UpTo(0),
          scheduleToClose = scheduleToClose,
          scheduleToStart = scheduleToStart,
          startToClose = startToClose,
          heartbeat = heartbeat,
          maxAttempts = maxAttempts
        ),
        statusScheduled
      )

    // This independent counter advances on delivery; Describe counts an attempt from scheduling.
    def startAttempt(s: State) =
      enter(
        s.copy(phase = started, attempts = states.saturatingSucc(s.attempts)),
        statusStarted,
        Fact.attemptCount
      )

    def complete(s: State) = enter(s.copy(phase = completed), statusCompleted)
    def heartbeat(s: State) = enter(s, heartbeatReceived)
    def heartbeatBackOff(s: State) =
      enter(states.afterRetry(s, scheduled), statusScheduled, Fact.attemptCount, heartbeatTimedOut)
        .because(states.nominalTimeWindow)
    def heartbeatBackOffPaused(s: State) =
      enter(states.afterRetry(s, paused), statusPaused, Fact.attemptCount, heartbeatTimedOut)
        .because(states.nominalTimeWindow)
    def heartbeatTimeOut(s: State) =
      enter(s.copy(phase = timedOut), statusTimedOut(TimeoutType.heartbeat), heartbeatTimedOut)
        .because(states.nominalTimeWindow)

    def fail(s: State) = enter(s.copy(phase = failed), statusFailed)

    // A retryable failure backs a started attempt off: read as scheduled again, one attempt higher.
    def backOff(s: State) =
      enter(
        states.afterRetry(s, scheduled),
        statusScheduled,
        Fact.attemptCount
      )
        .because("a retryable attempt backs off; the client reads scheduled again")

    // A pause requested during the attempt takes effect on its delayed retry (model.go:130-137).
    def backOffPaused(s: State) =
      enter(states.afterRetry(s, paused), statusPaused, Fact.attemptCount)

    def backOffAfterDeadline(s: State) =
      enter(states.afterRetry(s, scheduled), statusScheduled, Fact.attemptCount)
        .because(states.nominalTimeWindow)

    def backOffPausedAfterDeadline(s: State) =
      enter(states.afterRetry(s, paused), statusPaused, Fact.attemptCount)
        .because(states.nominalTimeWindow)

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

    // A delay expires without changing status, including while paused (model.go:355-376).
    def dispatchNow(s: State) =
      enter(s.copy(dispatch = Dispatch.now)).because(states.nominalTimeWindow)

    def timeOut(s: State, t: TimeoutType) =
      enter(s.copy(phase = timedOut), statusTimedOut(t))
        .because(states.nominalTimeWindow)

  object rules extends Rules:
    from(client) {
      import client.*

      on(start) {
        when(Phase.unstarted) ~> effects.schedule
      }

      // Closed controls answer NotFound before the live-state checks (operator_commands.go:249-254).
      on(pause, unpause, requestCancel, terminate) {
        when[Closed] ~> rejects(Rejection.notFound)
      }
      on(pause) {
        when(scheduled) ~> effects.pause
        when(started) ~> effects.requestPause
        when(paused, pauseRequested, cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because("already paused or cancellation pending (chasm/lib/activity/model/model.go:232)")
      }
      on(unpause) {
        when(paused) ~> effects.resume
        when(pauseRequested) ~> effects.withdrawPause
        when(scheduled, started, cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because("activity is not paused (chasm/lib/activity/model/model.go:251)")
      }
      on(requestCancel) {
        when(scheduled, paused) ~> effects.cancel
        when(started, pauseRequested) ~> effects.requestCancel
        when(cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because("cancellation already requested (chasm/lib/activity/model/model.go:201-202)")
      }
      on(terminate) {
        when[Live] ~> effects.terminate
      }
    }

    from(temporal.features.activity.standalone.worker) {
      import temporal.features.activity.standalone.worker.*

      on(poll) {
        when(scheduled).where(_.dispatch == Dispatch.now) ~> effects.startAttempt
      }
      on(heartbeat) {
        when[Held] ~> effects.heartbeat
        when(unstarted, scheduled, paused) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
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
        when(started).where(states.retriesRemaining) ~> effects.backOff
        when(cancelRequested) ~> effects.cancel
        when(pauseRequested).where(states.retriesRemaining) ~> effects.backOffPaused
        when(started, pauseRequested).where(s => !states.retriesRemaining(s)) ~> effects.fail
      }
      on(respondCanceled) {
        when(cancelRequested) ~> effects.cancel
        when(started, pauseRequested) ~> rejects(Rejection.invalidArgument)
          .because("cancellation was not requested (chasm/lib/activity/model/model.go:171)")
      }
    }

    from(service) {
      import service.*

      on(respondCompletedByID, respondFailedByID, respondCanceledByID) {
        when(unstarted) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
      }
      on(respondCompletedByID) {
        when[Live] ~> effects.complete
      }
      on(respondFailedByID(Failure.fatal)) {
        when[Held] ~> effects.fail
      }
      on(respondFailedByID(Failure.retryable)) {
        when(started).where(states.retriesRemaining) ~> effects.backOff
        when(pauseRequested).where(states.retriesRemaining) ~> effects.backOffPaused
        when(started, pauseRequested).where(s => !states.retriesRemaining(s)) ~> effects.fail
        when(cancelRequested) ~> effects.cancel
      }
      on(respondFailedByID, respondCanceledByID) {
        when(scheduled, paused) ~> rejects(Rejection.notFound)
      }
      on(respondCanceledByID) {
        when(cancelRequested) ~> effects.cancel
        when(started, pauseRequested) ~> rejects(Rejection.invalidArgument)
          .because("cancellation was not requested")
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

      on(startDelay) {
        when[Live].where(_.dispatch == Dispatch.startDelay) ~> effects.dispatchNow
      }
      on(backoff) {
        when[Live].where(_.dispatch == Dispatch.backoff) ~> effects.dispatchNow
      }
    }

    from(deadline) {
      import deadline.*

      // Started and not over: the phases a deadline can fire in.
      on(scheduleToClose) {
        when[Live].where(s =>
          s.scheduleToClose == Timeout.expires && s.dispatch != Dispatch.startDelay
        ) ~> (effects.timeOut(
          _,
          TimeoutType.scheduleToClose
        ))
      }
      // Waiting for a worker: the phases before an attempt is held, which schedule-to-start covers.
      on(scheduleToStart) {
        when[Waiting].where(s =>
          s.scheduleToStart == Timeout.expires && s.dispatch == Dispatch.now
        ) ~> (effects.timeOut(
          _,
          TimeoutType.scheduleToStart
        ))
      }
      on(startToClose) {
        when(started).where(s =>
          s.startToClose == Timeout.expires && states.retriesRemaining(s)
        ) ~> effects.backOffAfterDeadline
        when(pauseRequested).where(s =>
          s.startToClose == Timeout.expires && states.retriesRemaining(s)
        ) ~> effects.backOffPausedAfterDeadline
        when[Held]
          .where(s =>
            s.startToClose == Timeout.expires &&
              (s.phase == cancelRequested || !states.retriesRemaining(s))
          ) ~> (effects.timeOut(
          _,
          TimeoutType.startToClose
        ))
      }
      on(heartbeat) {
        when(started).where(s => s.heartbeat == Timeout.expires && states.retriesRemaining(s)) ~>
          effects.heartbeatBackOff
        when(pauseRequested).where(s =>
          s.heartbeat == Timeout.expires && states.retriesRemaining(s)
        ) ~> effects.heartbeatBackOffPaused
        when[Held].where(s =>
          s.heartbeat == Timeout.expires &&
            (s.phase == cancelRequested || !states.retriesRemaining(s))
        ) ~> effects.heartbeatTimeOut
      }
    }

  // What the System promises of its own: the settlement claims. The cross-entity claim of the
  // activity and its worker is the composition's, and the history record's are Record.scala's.
  object properties:
    val cancelIsNotUndone = property.holdsAcross { (before, after) =>
      (before.phase == Phase.cancelRequested) implies
        (after.state.phase == Phase.cancelRequested || after.state.phase.in[Closed])
    }

    // A dispatch delay remains pending across pause/unpause (model.go:239-244,355-376).
    val dispatchRequiresReady = property.holdsAcross { (before, after) =>
      (before.phase == Phase.scheduled && after.records(Fact.statusStarted)) implies
        (before.dispatch == Dispatch.now)
    }

    // Schedule-to-start counts from dispatch, after either delay (model.go:312-322).
    val scheduleToStartRequiresDispatch = property.holdsAcross { (before, after) =>
      after.records(Fact.statusTimedOut(TimeoutType.scheduleToStart)) implies
        (before.phase == Phase.scheduled && before.dispatch == Dispatch.now)
    }

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
        dispatch = Dispatch.now,
        attempts = UpTo(states.attemptBound),
        scheduleToClose = Timeout.unset,
        scheduleToStart = Timeout.unset,
        startToClose = Timeout.unset,
        heartbeat = Timeout.unset,
        maxAttempts = MaxAttempts.unlimited
      )

    // The attempt count saturates at `states.attemptBound`, so the claim is bounded by it: a completion on
    // any later attempt than the second reads as this one.
    val retryCompletes =
      property when worker.respondCompleted holds { s =>
        s.state == completedOnRetry && s.records(Fact.statusCompleted)
      }

    // The pinned exhaustion path takes this action twice, so both failures must satisfy the
    // Property: the exact first retry, then the exact exhausted settlement.
    val retryExhausts = property when worker.respondFailed(Failure.retryable) holds { s =>
      (s.state == completedOnRetry.copy(
        phase = Phase.scheduled,
        dispatch = Dispatch.backoff,
        attempts = UpTo(1),
        maxAttempts = MaxAttempts.two
      ) && s.records(Fact.statusScheduled) && s.records(Fact.attemptCount)) ||
      (s.state == completedOnRetry.copy(phase = Phase.failed, maxAttempts = MaxAttempts.two) &&
        s.records(Fact.statusFailed))
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

    // Terminal deadline witnesses record which deadline settled the activity. Start-to-close
    // uses a one-attempt start; with both scheduling deadlines set, either may fire first.
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

  // A terminate settles it, and DescribeActivityExecution reports its status by `activityStatus`.
  // The terminate find starts the activity and stops the worker before the control. The held
  // cancellation-request claim is authored on ByIDCancellation's complete service path below.
  object capabilities extends Capabilities:
    val terminable: Capability = Terminable(
      terminate = client.terminate,
      settled = Fact.statusTerminated,
      reach = Seq(client.start(), process.stop),
      expect = inconclusive(Reason.explanationsDisagree)
    )
    val describable: Capability = Describable(statusTable = activityStatus)
    val retryableFailure: Capability = Retries[State, Phase, MaxAttempts.Bound](
      failure = worker.respondFailed(Failure.retryable),
      retryable = true,
      attemptCount = states.attemptCount,
      maximumAttempts = states.maximumAttempts,
      retriesRemaining = states.retriesRemaining,
      pendingPause = Some(states.pendingPause),
      pendingCancel = Some(states.pendingCancel)
    )
    val fatalFailure: Capability = Retries[State, Phase, MaxAttempts.Bound](
      failure = worker.respondFailed(Failure.fatal),
      retryable = false,
      attemptCount = states.attemptCount,
      maximumAttempts = states.maximumAttempts,
      retriesRemaining = states.retriesRemaining,
      pendingPause = Some(states.pendingPause),
      pendingCancel = Some(states.pendingCancel)
    )
    val scheduleToCloseDeadline: Capability = Deadline[State, Phase, Live, Fact](
      timer = deadline.scheduleToClose,
      armed = states.scheduleToCloseArmed,
      timeout = Fact.statusTimedOut(TimeoutType.scheduleToClose),
      timeoutFacts = states.timeoutFacts
    )
    val scheduleToStartDeadline: Capability = Deadline[State, Phase, Waiting, Fact](
      timer = deadline.scheduleToStart,
      armed = states.scheduleToStartArmed,
      timeout = Fact.statusTimedOut(TimeoutType.scheduleToStart),
      timeoutFacts = states.timeoutFacts
    )
    val startToCloseDeadline: Capability = Deadline[State, Phase, Held, Fact](
      timer = deadline.startToClose,
      armed = states.startToCloseArmed,
      timeout = Fact.statusTimedOut(TimeoutType.startToClose),
      timeoutFacts = states.timeoutFacts,
      retryable = true,
      retriesRemaining = Some(states.retriesRemaining),
      pendingPause = Some(states.pendingPause),
      pendingCancel = Some(states.pendingCancel)
    )
    val heartbeatDeadline: Capability = Deadline[State, Phase, Held, Fact](
      timer = deadline.heartbeat,
      armed = states.heartbeatArmed,
      timeout = Fact.statusTimedOut(TimeoutType.heartbeat),
      timeoutFacts = states.timeoutFacts,
      retryable = true,
      retriesRemaining = Some(states.retriesRemaining),
      pendingPause = Some(states.pendingPause),
      pendingCancel = Some(states.pendingCancel)
    )

  // The paths, then one functional Query per side effect that settles the activity, and the Queries
  // that carry the other promises into the lifted Model. Each path starts before the activity exists;
  // a start that sets no deadline is `start()`, each input at `unset`. A path one Query takes is
  // written in it.
  object queries:
    capabilities.bound(
      three,
      Retries.failureReturnsToWaiting[State, Phase] -> eight,
      Retries.failureEndsFailed[State, Phase] -> eight,
      Retries.failurePauses[State, Phase] -> eight,
      Retries.failureCancels[State, Phase] -> eight,
      Retries.attemptCountIsWithinPolicy[State, MaxAttempts.Bound] -> eight,
      Deadline.firesInWindow[State, Phase, Live] -> eight,
      (Deadline.deadlineTimesOut[State, Phase]: AnyRef) -> eight,
      (Deadline.deadlineReturnsToWaiting[State, Phase]: AnyRef) -> eight,
      (Deadline.deadlinePauses[State, Phase]: AnyRef) -> eight
    )

    val any = scenario.free

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
    val delayedThenCompleted = scenario.actions(
      client.start(startDelay := expires),
      timers.startDelay,
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
    val exhausted = scenario.actions(
      client.start(maxAttempts := MaxAttempts.two),
      worker.poll,
      worker.respondFailed(Failure.retryable),
      timers.backoff,
      worker.poll,
      worker.respondFailed(Failure.retryable)
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
      client.start(startToClose := expires, maxAttempts := MaxAttempts.one),
      worker.poll,
      deadline.startToClose
    )

    val delayedAttemptsAreNotDispatched =
      query verify properties.dispatchRequiresReady in any limits eight
    val scheduleToStartWaitsForDispatch =
      query verify properties.scheduleToStartRequiresDispatch in any limits eight
    val cancellationKeepsPrecedence =
      query verify properties.cancelIsNotUndone in any limits eight

    // One Query per side effect that settles the activity, apart from the Properties they find.
    val completion =
      (query find properties.completes in completed limits three).expect(satisfied)
    val startDelayedCompletion =
      (query find properties.completes in delayedThenCompleted limits four).expect(satisfied)
    val nonRetryableFailure =
      (query find properties.nonRetryableFails in nonRetryable limits three).expect(satisfied)
    val retry =
      (query find properties.retryCompletes in retriedThenCompleted limits six)
        .expect(inconclusive(Reason.explanationsDisagree))
    val retryExhaustionByFailures =
      query verify properties.retryExhausts in exhausted limits six
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

// The same System state and rules, with a realization whose second delivery confirms a timeout,
// not a failed answer. One kind confirms all the occurrences it names, so the failure-retry kind
// cannot evidence a path on which no failed answer occurs.
object TimeoutRetry extends Derived(ActivitySystem.unmonitored):
  object properties:
    val completesAfterTimeout = property when worker.respondCompleted holds { s =>
      s.state == ActivitySystem.properties.completedOnRetry.copy(
        startToClose = Timeout.expires,
        maxAttempts = MaxAttempts.two
      ) && s.records(Fact.statusCompleted)
    }
    val failsAfterTimeout = property when worker.respondFailed(Failure.retryable) holds { s =>
      s.state == ActivitySystem.properties.completedOnRetry.copy(
        phase = Phase.failed,
        startToClose = Timeout.expires,
        maxAttempts = MaxAttempts.two
      ) && s.records(Fact.statusFailed)
    }
  object queries:
    val timedOutThenCompleted = scenario.actions(
      client.start(startToClose := expires, maxAttempts := MaxAttempts.two),
      worker.poll,
      deadline.startToClose,
      timers.backoff,
      worker.poll,
      worker.respondCompleted
    )
    val timedOutThenFailed = scenario.actions(
      client.start(startToClose := expires, maxAttempts := MaxAttempts.two),
      worker.poll,
      deadline.startToClose,
      timers.backoff,
      worker.poll,
      worker.respondFailed(Failure.retryable)
    )
    // Both full-State finds predict explanationsDisagree from status/attempt-only evidence.
    // The shared live gate must check actual Property assessments and reasons.
    val retryAfterTimeout =
      (query find properties.completesAfterTimeout in timedOutThenCompleted limits six)
        .expect(inconclusive(Reason.explanationsDisagree))
    val retryExhaustion =
      (query find properties.failsAfterTimeout in timedOutThenFailed limits six)
        .expect(inconclusive(Reason.explanationsDisagree))

object HeartbeatCompletion extends Derived(ActivitySystem.unmonitored):
  object properties:
    val heartbeatCompletes = property when worker.respondCompleted holds { s =>
      s.state.phase == Phase.completed && s.records(Fact.statusCompleted)
    }
  object queries:
    val heartbeatCompleted = scenario.actions(
      client.start(heartbeat := expires),
      worker.poll,
      worker.heartbeat,
      worker.respondCompleted
    )
    val heartbeatThenCompletes =
      (query find properties.heartbeatCompletes in heartbeatCompleted limits four)
        .total(19008)
        .expect(satisfied)

object HeartbeatRetry extends Derived(ActivitySystem.unmonitored):
  object properties:
    val heartbeatRetryCompletes = property when worker.respondCompleted holds { s =>
      s.state.phase == Phase.completed && s.records(Fact.statusCompleted)
    }
  object queries:
    val heartbeatRetried = scenario.actions(
      client.start(heartbeat := expires, maxAttempts := MaxAttempts.two),
      worker.poll,
      worker.heartbeat,
      deadline.heartbeat,
      timers.backoff,
      worker.poll,
      worker.respondCompleted
    )
    val heartbeatTimeoutRetriesThenCompletes =
      (query find properties.heartbeatRetryCompletes in heartbeatRetried limits eight)
        .total(33264)
        .expect(satisfied)

object HeartbeatExhaustion extends Derived(ActivitySystem.unmonitored):
  object properties:
    val heartbeatExhausts = property.when(deadline.heartbeat) holds (after =>
      after.records(Fact.heartbeatTimedOut) &&
        (ActivitySystem.states.retriesRemaining(after.state) ||
          (after.state.phase == Phase.timedOut && after.records(
            Fact.statusTimedOut(TimeoutType.heartbeat)
          )))
    )
  object queries:
    val heartbeatExhausted = scenario.actions(
      client.start(heartbeat := expires, maxAttempts := MaxAttempts.one),
      worker.poll,
      worker.heartbeat,
      deadline.heartbeat
    )
    val heartbeatTimeoutExhausts =
      (query find properties.heartbeatExhausts in heartbeatExhausted limits four)
        .total(19008)
        .expect(satisfied)

// A closed activity answers a repeated By-ID call NotFound and records nothing, so a Run cannot rule
// out a silent rejected repeat: the explanations of each By-ID witness disagree, as terminate's do.
object ByIDCompletion extends Derived(ActivitySystem.unmonitored):
  object properties:
    val completedByID = property when service.respondCompletedByID holds { after =>
      after.state.phase == Phase.completed && after.records(Fact.statusCompleted)
    }
  object queries:
    val scheduledCompletion = scenario.actions(client.start(), service.respondCompletedByID)
    val scheduledCompletedByID =
      (query find properties.completedByID in scheduledCompletion limits three)
        .total(9504)
        .expect(inconclusive(Reason.explanationsDisagree))

object ByIDFailure extends Derived(ActivitySystem.unmonitored):
  object properties:
    val fatalFailureByID = property when service.respondFailedByID(Failure.fatal) holds { after =>
      after.state.phase == Phase.failed && after.records(Fact.statusFailed)
    }
  object queries:
    val heldFatalFailure = scenario.actions(
      client.start(),
      worker.poll,
      service.respondFailedByID(Failure.fatal)
    )
    val heldFailedByID =
      (query find properties.fatalFailureByID in heldFatalFailure limits three)
        .total(14256)
        .expect(inconclusive(Reason.explanationsDisagree))

object ByIDCancellation extends Derived(ActivitySystem.unmonitored):
  object properties:
    val cancelIsRequested =
      property("activitySystem.cancelIsRequested") when client.requestCancel holds (
        _.records(Fact.statusCancelRequested)
      )
    val canceledByID = property when service.respondCanceledByID holds { after =>
      after.state.phase == Phase.canceled && after.records(Fact.statusCanceled)
    }
  object queries:
    val heldCancellation = scenario.actions(
      client.start(),
      worker.poll,
      client.requestCancel,
      service.respondCanceledByID
    )
    val heldCanceledByID =
      (query find properties.canceledByID in heldCancellation limits four)
        .total(19008)
        .expect(inconclusive(Reason.explanationsDisagree))
    val cancelIsRequested =
      (query(
        "activitySystem.cancelIsRequested"
      ) find properties.cancelIsRequested in heldCancellation limits four)
        .total(19008)
        .expect(inconclusive(Reason.explanationsDisagree))

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
