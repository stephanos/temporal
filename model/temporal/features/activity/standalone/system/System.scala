// The standalone activity's lifecycle System and its Product refinement.
// Derived subject models and dispatch compositions live beside it in subject files.
package temporal
package features.activity
package standalone
package system

import scala.annotation.unused
import framework.*
import framework.outcomes.{Outcome, Rejection}
import framework.realize.Reason
import temporal.capabilities.*
import temporal.realize.inconclusive
import Bounds.three
import actors.worker.worker as process
import product.ActivityProduct

// It begins before the activity exists, so unstarted is one phase.
enum Phase derives Finite:
  case unstarted
  case scheduled extends Phase, Waiting
  case started extends Phase, Held
  case paused extends Phase, Suspended
  case pauseRequested extends Phase, Held
  case cancelRequested extends Phase, Held
  // A reset of a held attempt waits for the attempt to end; the second keeps a requested pause
  // (operator_commands.go:485-519).
  case resetRequested extends Phase, Held
  case resetKeepingPause extends Phase, Held
  case completed extends Phase, Succeeded
  case failed extends Phase, Failed
  case canceled extends Phase, Canceled
  case terminated extends Phase, Terminated
  case timedOut extends Phase, TimedOut

// Whether a waiting attempt can be dispatched, independent of its status (model.go:239-244).
enum Dispatch derives Finite:
  case now, startDelay, backoff

// 13 phases, 3 dispatch values, 3 attempt counts, 4 deadline flags and 3 retry policies: 5616 states.
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
    def pendingReset(s: State): Boolean = s.phase.in(resetRequested, resetKeepingPause)

    // A reset discards a retry backoff but keeps a start delay (operator_commands.go:410-416).
    def resetDispatch(d: Dispatch): Dispatch = if d == Dispatch.backoff then Dispatch.now else d

    // A pending reset applied: a first attempt, dispatched at once, paused only where it was kept.
    def resetApplied(before: State, after: State): Boolean =
      after.attempts == 0 && after.dispatch == Dispatch.now &&
        after.phase == (if before.phase == resetKeepingPause then paused else scheduled)

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
      case Phase.started | Phase.pauseRequested | Phase.resetRequested | Phase.resetKeepingPause =>
        product.State(product.Phase.started)
      case Phase.paused          => product.State(product.Phase.paused)
      case Phase.cancelRequested => product.State(product.Phase.cancelRequested)
      case Phase.completed       => product.State(product.Phase.completed)
      case Phase.failed          => product.State(product.Phase.failed)
      case Phase.canceled        => product.State(product.Phase.canceled)
      case Phase.terminated      => product.State(product.Phase.terminated)
      case Phase.timedOut        => product.State(product.Phase.timedOut)

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

    // A reset of a waiting or paused activity applies at once: its independent count restarts at
    // zero and a retry backoff is discarded (operator_commands.go:521-563).
    def resetWaiting(s: State) =
      enter(
        s.copy(phase = scheduled, attempts = UpTo(0), dispatch = states.resetDispatch(s.dispatch)),
        statusScheduled
      )
    def resetPaused(s: State) =
      enter(
        s.copy(phase = paused, attempts = UpTo(0), dispatch = states.resetDispatch(s.dispatch)),
        statusPaused
      )

    // A reset of a held attempt is recorded until the attempt ends; Describe still reads the
    // attempt as started, or pause-requested when the pause is kept (responses.go:55-62).
    def requestReset(s: State) =
      enter(s.copy(phase = resetRequested), statusStarted)
        .because("the worker learns of the reset on its next heartbeat")
    def requestResetKeepingPause(s: State) =
      enter(s.copy(phase = resetKeepingPause), statusPaused)
        .because("the worker learns of the reset on its next heartbeat")

    // A pending reset applies when the held attempt fails or one of its own deadlines ends it,
    // whatever the failure's retryability or the policy left: the next attempt is a first one,
    // dispatched at once (attempt.go:201-216, statemachine.go:317-345).
    def applyReset(s: State) =
      enter(
        s.copy(phase = scheduled, attempts = UpTo(0), dispatch = Dispatch.now),
        statusScheduled,
        Fact.attemptCount
      )
        .because("the deferred reset applies as the attempt ends")
    def applyResetPaused(s: State) =
      enter(
        s.copy(phase = paused, attempts = UpTo(0), dispatch = Dispatch.now),
        statusPaused,
        Fact.attemptCount
      )
        .because("the deferred reset applies as the attempt ends")
    def applyResetOnHeartbeat(s: State) =
      enter(
        s.copy(phase = scheduled, attempts = UpTo(0), dispatch = Dispatch.now),
        statusScheduled,
        Fact.attemptCount,
        heartbeatTimedOut
      ).because(states.nominalTimeWindow)
    def applyResetPausedOnHeartbeat(s: State) =
      enter(
        s.copy(phase = paused, attempts = UpTo(0), dispatch = Dispatch.now),
        statusPaused,
        Fact.attemptCount,
        heartbeatTimedOut
      ).because(states.nominalTimeWindow)

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
        when(paused, pauseRequested, cancelRequested, resetRequested, resetKeepingPause) ~>
          rejects(Rejection.failedPrecondition)
            .because(
              "already paused, or a cancellation or reset pending (chasm/lib/activity/model/model.go:232)"
            )
      }
      on(unpause) {
        when(paused) ~> effects.resume
        when(pauseRequested) ~> effects.withdrawPause
        when(scheduled, started, cancelRequested, resetRequested, resetKeepingPause) ~>
          rejects(Rejection.failedPrecondition)
            .because("activity is not paused (chasm/lib/activity/model/model.go:251)")
      }
      // Cancel > Reset > Pause: a cancellation replaces a pending reset, a reset a requested
      // pause, and neither a pause nor a reset undoes a cancellation (model.go:156-158).
      on(reset) {
        when(unstarted) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
        when(scheduled) ~> effects.resetWaiting
        when(started) ~> effects.requestReset
        when(cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(
            "cannot reset an activity with a pending cancellation (operator_commands.go:458)"
          )
        when(resetRequested, resetKeepingPause) ~> rejects(Rejection.failedPrecondition)
          .because("cannot reset an activity with a pending reset (operator_commands.go:460-464)")
      }
      on(reset(ResetPause.resume)) {
        when(paused) ~> effects.resetWaiting
        when(pauseRequested) ~> effects.requestReset
      }
      on(reset(ResetPause.keepPaused)) {
        when(paused) ~> effects.resetPaused
        when(pauseRequested) ~> effects.requestResetKeepingPause
      }
      on(requestCancel) {
        when(scheduled, paused) ~> effects.cancel
        when(started, pauseRequested, resetRequested, resetKeepingPause) ~> effects.requestCancel
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
      // A pending reset applies on either failure, before its retryability is read.
      on(respondFailed) {
        when(resetRequested) ~> effects.applyReset
        when(resetKeepingPause) ~> effects.applyResetPaused
      }
      on(respondFailed(Failure.fatal)) {
        when(started, pauseRequested, cancelRequested) ~> effects.fail
      }

      on(respondFailed(Failure.retryable)) {
        when(started).where(states.retriesRemaining) ~> effects.backOff
        when(cancelRequested) ~> effects.cancel
        when(pauseRequested).where(states.retriesRemaining) ~> effects.backOffPaused
        when(started, pauseRequested).where(s => !states.retriesRemaining(s)) ~> effects.fail
      }
      on(respondCanceled) {
        when(cancelRequested) ~> effects.cancel
        when(started, pauseRequested, resetRequested, resetKeepingPause) ~>
          rejects(Rejection.invalidArgument)
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
      on(respondFailedByID) {
        when(resetRequested) ~> effects.applyReset
        when(resetKeepingPause) ~> effects.applyResetPaused
      }
      on(respondFailedByID(Failure.fatal)) {
        when(started, pauseRequested, cancelRequested) ~> effects.fail
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
        when(started, pauseRequested, resetRequested, resetKeepingPause) ~>
          rejects(Rejection.invalidArgument)
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
        when(started, pauseRequested, cancelRequested)
          .where(s =>
            s.startToClose == Timeout.expires &&
              (s.phase == cancelRequested || !states.retriesRemaining(s))
          ) ~> (effects.timeOut(
          _,
          TimeoutType.startToClose
        ))
        when(resetRequested).where(_.startToClose == Timeout.expires) ~> effects.applyReset
        when(resetKeepingPause).where(_.startToClose == Timeout.expires) ~> effects.applyResetPaused
      }
      on(heartbeat) {
        when(started).where(s => s.heartbeat == Timeout.expires && states.retriesRemaining(s)) ~>
          effects.heartbeatBackOff
        when(pauseRequested).where(s =>
          s.heartbeat == Timeout.expires && states.retriesRemaining(s)
        ) ~> effects.heartbeatBackOffPaused
        when(started, pauseRequested, cancelRequested).where(s =>
          s.heartbeat == Timeout.expires &&
            (s.phase == cancelRequested || !states.retriesRemaining(s))
        ) ~> effects.heartbeatTimeOut
        when(resetRequested).where(_.heartbeat == Timeout.expires) ~> effects.applyResetOnHeartbeat
        when(resetKeepingPause).where(_.heartbeat == Timeout.expires) ~>
          effects.applyResetPausedOnHeartbeat
      }
    }

  // The lifecycle-wide control and reset laws; focused settlement claims live in subject files.
  // The activity and worker's cross-entity claim is the composition's, and the dispatch record's
  // are Dispatch.scala's.
  object properties:
    val cancelIsNotUndone = property.holdsAcross { (before, after) =>
      (before.phase == Phase.cancelRequested) implies
        (after.state.phase == Phase.cancelRequested || after.state.phase.in[Closed])
    }

    val terminated = property when client.terminate holds { s =>
      s.state.phase == Phase.terminated && s.records(Fact.statusTerminated)
    }

    // A pending reset ends only as its held attempt does: a completion wins, a cancellation or a
    // terminate replaces it, schedule-to-close stays terminal, and a failure or one of the
    // attempt's own deadlines applies it, whatever the policy left. The next attempt is a first
    // one, dispatched at once, and paused only where the pause was kept; no failure or timeout is
    // recorded for it. Any other step leaves it as it was. With no reset pending, an existing
    // attempt's count never goes back but where a waiting or paused activity restarts, with no
    // backoff left.
    val resetSettles = property.holdsAcross { (before, after) =>
      val restarted = after.state.attempts == 0 && after.state.dispatch == Dispatch.now &&
        after.state.scheduleToClose == before.scheduleToClose &&
        after.state.scheduleToStart == before.scheduleToStart &&
        after.state.startToClose == before.startToClose &&
        after.state.heartbeat == before.heartbeat &&
        after.state.maxAttempts == before.maxAttempts &&
        !after.records(Fact.statusFailed) &&
        !after.records(Fact.statusTimedOut(TimeoutType.scheduleToClose)) &&
        !after.records(Fact.statusTimedOut(TimeoutType.scheduleToStart)) &&
        !after.records(Fact.statusTimedOut(TimeoutType.startToClose)) &&
        !after.records(Fact.statusTimedOut(TimeoutType.heartbeat))
      if before.phase == Phase.resetRequested then
        after.state == before ||
        (after.state.phase == Phase.completed && after.records(Fact.statusCompleted)) ||
        after.state.phase == Phase.cancelRequested ||
        after.state.phase == Phase.terminated ||
        (after.state.phase == Phase.timedOut &&
          after.records(Fact.statusTimedOut(TimeoutType.scheduleToClose))) ||
        (restarted && after.state.phase == Phase.scheduled && after.records(Fact.statusScheduled))
      else if before.phase == Phase.resetKeepingPause then
        after.state == before ||
        (after.state.phase == Phase.completed && after.records(Fact.statusCompleted)) ||
        after.state.phase == Phase.cancelRequested ||
        after.state.phase == Phase.terminated ||
        (after.state.phase == Phase.timedOut &&
          after.records(Fact.statusTimedOut(TimeoutType.scheduleToClose))) ||
        (restarted && after.state.phase == Phase.paused && after.records(Fact.statusPaused))
      else
        before.phase == Phase.unstarted || after.state.attempts >= before.attempts ||
        (before.phase.in(Phase.scheduled, Phase.paused) &&
          after.state.phase.in(Phase.scheduled, Phase.paused) &&
          after.state.attempts == 0 && after.state.dispatch != Dispatch.backoff)
    }

    // A reset of a waiting or paused activity restarts it at once: its count is zero and a retry
    // backoff is discarded, a start delay kept. Without keep_paused it is scheduled again.
    val resetResumes = property.when(client.reset(ResetPause.resume)) holdsAcross {
      (before, after) =>
        !before.phase.in(Phase.scheduled, Phase.paused) ||
        (after.state == before.copy(
          phase = Phase.scheduled,
          attempts = UpTo(0),
          dispatch = if before.dispatch == Dispatch.backoff then Dispatch.now else before.dispatch
        ) && after.records(Fact.statusScheduled))
    }

    // With keep_paused a paused activity stays paused, and a scheduled one is scheduled again.
    val resetKeepsPaused =
      property.when(client.reset(ResetPause.keepPaused)) holdsAcross { (before, after) =>
        !before.phase.in(Phase.scheduled, Phase.paused) ||
        (after.state == before.copy(
          attempts = UpTo(0),
          dispatch = if before.dispatch == Dispatch.backoff then Dispatch.now else before.dispatch
        ) && after.records(
          if before.phase == Phase.paused then Fact.statusPaused else Fact.statusScheduled
        ))
      }

    // Retries' and Deadline's laws as the activity keeps them: a pending reset applies first, as
    // a failure or one of the attempt's own deadlines ends the held attempt, whatever the failure's
    // retryability or the policy left (attempt.go:201-216, statemachine.go:317-345). With no reset
    // pending each states its companion's settlement exactly.
    def resetFailureReturnsToWaiting(m: Declares[State])(
        failure: ClassRef,
        retryable: Boolean,
        retriesRemaining: State => Boolean,
        pendingPause: State => Boolean,
        pendingCancel: State => Boolean
    ): Property[State] =
      m.property.when(failure) holdsAcross ((before, after) =>
        if states.pendingReset(before) then states.resetApplied(before, after.state)
        else
          !(retryable && !pendingCancel(before) && retriesRemaining(before) &&
            !pendingPause(before)) || after.state.phase.in[Waiting]
      )

    def resetFailureEndsFailed(m: Declares[State])(
        failure: ClassRef,
        retryable: Boolean,
        retriesRemaining: State => Boolean,
        pendingCancel: State => Boolean
    ): Property[State] =
      m.property.when(failure) holdsAcross ((before, after) =>
        if states.pendingReset(before) then states.resetApplied(before, after.state)
        else
          !(!retryable || (!pendingCancel(before) && !retriesRemaining(before))) ||
          after.state.phase.in[Failed]
      )

    def resetFailurePauses(m: Declares[State])(
        failure: ClassRef,
        retryable: Boolean,
        retriesRemaining: State => Boolean,
        pendingPause: State => Boolean,
        pendingCancel: State => Boolean
    ): Property[State] =
      m.property.when(failure) holdsAcross ((before, after) =>
        if states.pendingReset(before) then states.resetApplied(before, after.state)
        else
          !(retryable && !pendingCancel(before) && retriesRemaining(before) &&
            pendingPause(before)) || after.state.phase.in[Suspended]
      )

    def resetDeadlineTimesOut(m: Declares[State])(
        timer: ClassRef,
        retryable: Boolean,
        timeout: m.Fact,
        retriesRemaining: State => Boolean,
        pendingCancel: State => Boolean
    ): Property[State] =
      m.property.when(timer) holdsAcross ((before, after) =>
        if states.pendingReset(before) then
          states.resetApplied(before, after.state) && !after.records(timeout)
        else
          !(pendingCancel(before) || !retryable || !retriesRemaining(before)) ||
          (after.state.phase.in[TimedOut] && after.records(timeout))
      )

    def resetDeadlineReturnsToWaiting(m: Declares[State])(
        timer: ClassRef,
        retryable: Boolean,
        retriesRemaining: State => Boolean,
        timeoutFacts: Seq[m.Fact],
        pendingPause: State => Boolean,
        pendingCancel: State => Boolean
    ): Property[State] =
      m.property.when(timer) holdsAcross ((before, after) =>
        if states.pendingReset(before) then
          states.resetApplied(before, after.state) && timeoutFacts.forall(f => !after.records(f))
        else
          !(retryable && !pendingCancel(before) && retriesRemaining(before) &&
            !pendingPause(before)) ||
          (after.state.phase.in[Waiting] && timeoutFacts.forall(f => !after.records(f)))
      )

    def resetDeadlinePauses(m: Declares[State])(
        timer: ClassRef,
        retryable: Boolean,
        retriesRemaining: State => Boolean,
        pendingPause: State => Boolean,
        timeoutFacts: Seq[m.Fact],
        pendingCancel: State => Boolean
    ): Property[State] =
      m.property.when(timer) holdsAcross ((before, after) =>
        if states.pendingReset(before) then
          states.resetApplied(before, after.state) && timeoutFacts.forall(f => !after.records(f))
        else
          !(retryable && !pendingCancel(before) && retriesRemaining(before) &&
            pendingPause(before)) ||
          (after.state.phase.in[Suspended] && timeoutFacts.forall(f => !after.records(f)))
      )

    // Cancel > Reset > Pause: no step makes a reset pending over a pending cancellation, and no
    // step turns a pending reset back into a pause request or a plain attempt (model.go:156-158).
    val controlPrecedence = property.holdsAcross { (before, after) =>
      (before.phase != Phase.cancelRequested || !states.pendingReset(after.state)) &&
      (!states.pendingReset(before) || !after.state.phase.in(Phase.started, Phase.pauseRequested))
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
    // A pending reset applies as a failure or the attempt's own deadline ends it, before its
    // retryability or the policy is read (attempt.go:201-216, statemachine.go:317-345); the
    // schedule-to-close deadline keeps its terminal law.
    overriding(
      Retries.failureReturnsToWaiting[State, Phase] -> properties.resetFailureReturnsToWaiting,
      because = "a pending reset applies first (chasm/lib/activity/attempt.go:201-216)",
      of = Seq(retryableFailure, fatalFailure)
    )
    overriding(
      Retries.failureEndsFailed[State, Phase] -> properties.resetFailureEndsFailed,
      because = "a pending reset applies first (chasm/lib/activity/attempt.go:201-216)",
      of = Seq(retryableFailure, fatalFailure)
    )
    overriding(
      Retries.failurePauses[State, Phase] -> properties.resetFailurePauses,
      because = "a pending reset applies first (chasm/lib/activity/attempt.go:201-216)",
      of = Seq(retryableFailure, fatalFailure)
    )
    overriding(
      (Deadline.deadlineTimesOut[State, Phase]: AnyRef) ->
        (properties.resetDeadlineTimesOut: AnyRef),
      because = "a pending reset applies first (chasm/lib/activity/attempt.go:201-216)",
      of = Seq(startToCloseDeadline, heartbeatDeadline)
    )
    overriding(
      (Deadline.deadlineReturnsToWaiting[State, Phase]: AnyRef) ->
        (properties.resetDeadlineReturnsToWaiting: AnyRef),
      because = "a pending reset applies first (chasm/lib/activity/attempt.go:201-216)",
      of = Seq(startToCloseDeadline, heartbeatDeadline)
    )
    overriding(
      (Deadline.deadlinePauses[State, Phase]: AnyRef) ->
        (properties.resetDeadlinePauses: AnyRef),
      because = "a pending reset applies first (chasm/lib/activity/attempt.go:201-216)",
      of = Seq(startToCloseDeadline, heartbeatDeadline)
    )

  // Capability laws and the lifecycle-wide control and reset Queries.
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

    val terminatedWhileScheduled = scenario.actions(
      client.start(),
      process.stop,
      client.terminate
    )
    val cancellationKeepsPrecedence =
      query verify properties.cancelIsNotUndone in any limits eight
    val resetSettlement =
      query verify properties.resetSettles in any limits eight
    val directResetResumes =
      query verify properties.resetResumes in any limits eight
    val directResetKeepsPaused =
      query verify properties.resetKeepsPaused in any limits eight
    val controlsKeepPrecedence =
      query verify properties.controlPrecedence in any limits eight

    // The worker stops before the start, so no attempt is in flight when the client terminates.
    val terminate =
      (query find properties.terminated in terminatedWhileScheduled limits three)
        .expect(inconclusive(Reason.explanationsDisagree))
