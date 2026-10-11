// The standalone activity's lifecycle System and its Product refinement.
// Derived subject models and dispatch compositions live beside it in subject files.
package temporal
package features.activity
package standalone
package system

import scala.annotation.unused
import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import framework.realize.Reason
import temporal.capabilities.*
import temporal.realize.inconclusive
import Bounds.{eight, three}
import actors.worker.worker as process
import product.ActivityProduct

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

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

    def exhausted(s: State): Boolean = !retriesRemaining(s)
    def endsTerminally(s: State): Boolean = pendingCancel(s) || exhausted(s)

    def attemptCount(s: State): Int = s.attempts

    def maximumAttempts(s: State): Option[UpTo[MaxAttempts.Bound]] = s.maxAttempts match
      case MaxAttempts.unlimited => None
      case MaxAttempts.one       => Some(UpTo(1))
      case MaxAttempts.two       => Some(UpTo(MaxAttempts.bound))
    // format: off
    def pendingPause(s: State): Boolean = s.phase == Phase.pauseRequested // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    def pendingCancel(s: State): Boolean = s.phase == Phase.cancelRequested // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    def pendingReset(s: State): Boolean = s.phase.in(Phase.resetRequested, Phase.resetKeepingPause)
    // format: on
    def landingPhase(s: State): Phase =
      if s.phase.in(Phase.pauseRequested, Phase.resetKeepingPause) then Phase.paused
      else Phase.scheduled
    // format: off
    def landingStatus(s: State): Fact =
      if landingPhase(s) == Phase.paused then Fact.statusPaused else Fact.statusScheduled // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons

    // A reset discards a retry backoff but keeps a start delay (operator_commands.go:410-416).
    def resetDispatch(d: Dispatch): Dispatch = if d == Dispatch.backoff then Dispatch.now else d // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons

    // A pending reset applied: a first attempt, dispatched at once, paused only where it was kept.
    def resetApplied(before: State, after: State): Boolean =
      after.attempts == 0 && after.dispatch == Dispatch.now && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        after.phase == (if before.phase == Phase.resetKeepingPause then Phase.paused else Phase.scheduled) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons

    def scheduleToCloseArmed(s: State): Boolean =
      s.scheduleToClose == Timeout.expires && s.dispatch != Dispatch.startDelay // scalafix:ok DisableSyntax.==, DisableSyntax.!=; strictEquality checks native comparisons
    def scheduleToStartArmed(s: State): Boolean =
      s.scheduleToStart == Timeout.expires && s.dispatch == Dispatch.now // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    def startToCloseArmed(s: State): Boolean = s.startToClose == Timeout.expires // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    def heartbeatArmed(s: State): Boolean = s.heartbeat == Timeout.expires // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    // format: on
    val timeoutFacts = Seq(
      Fact.statusTimedOut(TimeoutType.scheduleToClose),
      Fact.statusTimedOut(TimeoutType.scheduleToStart),
      Fact.statusTimedOut(TimeoutType.startToClose),
      Fact.statusTimedOut(TimeoutType.heartbeat)
    )

    def afterRetry(s: State, phase: Phase) = s.copy(phase = phase, dispatch = Dispatch.backoff)

    val nominalTimeWindow =
      "The configured time window is nominal, not proof that a real timer fired; the rule's guards select its applicability. Scheduling deadlines start after start delay; schedule-to-start also waits through retry backoff (chasm/lib/activity/model/model.go:300-310,312-335,355-376)."

    val pendingResetReason =
      "a pending reset applies first (chasm/lib/activity/attempt.go:201-216)"

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
    // format: off
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
          phase = Phase.scheduled,
          dispatch = if startDelay == Timeout.expires then Dispatch.startDelay else Dispatch.now, // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
          attempts = UpTo(0),
          scheduleToClose = scheduleToClose,
          scheduleToStart = scheduleToStart,
          startToClose = startToClose,
          heartbeat = heartbeat,
          maxAttempts = maxAttempts
        ),
        statusScheduled
      )
    // format: on
    // This independent counter advances on delivery; Describe counts an attempt from scheduling.
    def startAttempt(s: State) =
      enter(
        s.copy(phase = Phase.started, attempts = states.saturatingSucc(s.attempts)),
        statusStarted,
        Fact.attemptCount
      )

    def complete(s: State) = enter(s.copy(phase = Phase.completed), statusCompleted)
    def heartbeat(s: State) = enter(s, heartbeatReceived)
    def heartbeatBackOff(s: State) =
      enter(
        states.afterRetry(s, states.landingPhase(s)),
        states.landingStatus(s),
        Fact.attemptCount,
        heartbeatTimedOut
      )
        .because(states.nominalTimeWindow)
    def heartbeatTimeOut(s: State) =
      enter(
        s.copy(phase = Phase.timedOut),
        statusTimedOut(TimeoutType.heartbeat),
        heartbeatTimedOut
      )
        .because(states.nominalTimeWindow)

    def fail(s: State) = enter(s.copy(phase = Phase.failed), statusFailed)

    // A retryable failure backs a started attempt off: read as scheduled again, one attempt higher.
    def backOff(s: State) =
      enter(
        states.afterRetry(s, states.landingPhase(s)),
        states.landingStatus(s),
        Fact.attemptCount
      )
        .because("a retryable attempt backs off; the client reads scheduled again")

    // A pause requested during the attempt takes effect on its delayed retry (model.go:130-137).
    def backOffPaused(s: State) =
      enter(
        states.afterRetry(s, states.landingPhase(s)),
        states.landingStatus(s),
        Fact.attemptCount
      )

    def backOffAfterDeadline(s: State) =
      enter(
        states.afterRetry(s, states.landingPhase(s)),
        states.landingStatus(s),
        Fact.attemptCount
      )
        .because(states.nominalTimeWindow)

    def cancel(s: State) = enter(s.copy(phase = Phase.canceled), statusCanceled)

    def pause(s: State) = enter(s.copy(phase = Phase.paused), statusPaused)

    // A pause of a held attempt is a request the worker learns of on its next heartbeat, so it is its
    // own phase the client reads as paused.
    def requestPause(s: State) =
      enter(s.copy(phase = Phase.pauseRequested), statusPaused)
        .because("the worker learns of the pause on its next heartbeat")

    def resume(s: State) = enter(s.copy(phase = Phase.scheduled), statusScheduled)

    // An unpause before the worker yields: it keeps the attempt it holds.
    def withdrawPause(s: State) = enter(s.copy(phase = Phase.started), statusStarted)

    def requestCancel(s: State) =
      enter(s.copy(phase = Phase.cancelRequested), statusCancelRequested)

    def terminate(s: State) = enter(s.copy(phase = Phase.terminated), statusTerminated)

    // A reset of a waiting or paused activity applies at once: its independent count restarts at
    // zero and a retry backoff is discarded (operator_commands.go:521-563).
    def resetWaiting(s: State) =
      enter(
        s.copy(
          phase = Phase.scheduled,
          attempts = UpTo(0),
          dispatch = states.resetDispatch(s.dispatch)
        ),
        statusScheduled
      )
    def resetPaused(s: State) =
      enter(
        s.copy(
          phase = Phase.paused,
          attempts = UpTo(0),
          dispatch = states.resetDispatch(s.dispatch)
        ),
        statusPaused
      )

    // A reset of a held attempt is recorded until the attempt ends; Describe still reads the
    // attempt as started, or pause-requested when the pause is kept (responses.go:55-62).
    def requestReset(s: State) =
      enter(s.copy(phase = Phase.resetRequested), statusStarted)
        .because("the worker learns of the reset on its next heartbeat")
    def requestResetKeepingPause(s: State) =
      enter(s.copy(phase = Phase.resetKeepingPause), statusPaused)
        .because("the worker learns of the reset on its next heartbeat")

    // A pending reset applies when the held attempt fails or one of its own deadlines ends it,
    // whatever the failure's retryability or the policy left: the next attempt is a first one,
    // dispatched at once (attempt.go:201-216, statemachine.go:317-345).
    def applyReset(s: State) =
      enter(
        s.copy(phase = states.landingPhase(s), attempts = UpTo(0), dispatch = Dispatch.now),
        states.landingStatus(s),
        Fact.attemptCount
      )
        .because("the deferred reset applies as the attempt ends")
    def applyResetOnHeartbeat(s: State) =
      enter(
        s.copy(phase = states.landingPhase(s), attempts = UpTo(0), dispatch = Dispatch.now),
        states.landingStatus(s),
        Fact.attemptCount,
        heartbeatTimedOut
      ).because(states.nominalTimeWindow)

    // Keeps the state and records nothing; the next step's evidence confirms it, a Known Gap.
    def keep(s: State) = stay(s)

    // A delay expires without changing status, including while paused (model.go:355-376).
    def dispatchNow(s: State) =
      enter(s.copy(dispatch = Dispatch.now)).because(states.nominalTimeWindow)

    def timeOut(s: State, t: TimeoutType) =
      enter(s.copy(phase = Phase.timedOut), statusTimedOut(t))
        .because(states.nominalTimeWindow)

  object rules extends Rules:
    from(client) {
      on(client.start) {
        when(Phase.unstarted) ~> effects.schedule
      }

      // Closed controls answer NotFound before the live-state checks (operator_commands.go:249-254).
      on(client.pause, client.unpause, client.requestCancel, client.terminate) {
        when[Closed] ~> rejects(Rejection.notFound)
      }
      on(client.pause) {
        when(Phase.scheduled) ~> effects.pause
        when(Phase.started) ~> effects.requestPause
        when(
          Phase.paused,
          Phase.pauseRequested,
          Phase.cancelRequested,
          Phase.resetRequested,
          Phase.resetKeepingPause
        ) ~>
          rejects(Rejection.failedPrecondition)
            .because(pausedOrControlPending)
      }
      on(client.unpause) {
        when(Phase.paused) ~> effects.resume
        when(Phase.pauseRequested) ~> effects.withdrawPause
        when(
          Phase.scheduled,
          Phase.started,
          Phase.cancelRequested,
          Phase.resetRequested,
          Phase.resetKeepingPause
        ) ~>
          rejects(Rejection.failedPrecondition)
            .because(activityNotPaused)
      }
      // Cancel > Reset > Pause: a cancellation replaces a pending reset, a reset a requested
      // pause, and neither a pause nor a reset undoes a cancellation (model.go:156-158).
      on(client.reset) {
        when(Phase.unstarted) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled) ~> effects.resetWaiting
        when(Phase.started) ~> effects.requestReset
        when(Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(resetWithPendingCancellation)
        when(Phase.resetRequested, Phase.resetKeepingPause) ~> rejects(Rejection.failedPrecondition)
          .because(resetWithPendingReset)
      }
      on(client.reset(ResetPause.resume)) {
        when(Phase.paused) ~> effects.resetWaiting
        when(Phase.pauseRequested) ~> effects.requestReset
      }
      on(client.reset(ResetPause.keepPaused)) {
        when(Phase.paused) ~> effects.resetPaused
        when(Phase.pauseRequested) ~> effects.requestResetKeepingPause
      }
      on(client.requestCancel) {
        when(Phase.scheduled, Phase.paused) ~> effects.cancel
        when(
          Phase.started,
          Phase.pauseRequested,
          Phase.resetRequested,
          Phase.resetKeepingPause
        ) ~> effects.requestCancel
        when(Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(cancellationAlreadyRequested)
      }
      on(client.terminate) {
        when[Live] ~> effects.terminate
      }
    }

    // Held completion, pending-reset failure, fatal failure, retry/exhaustion and cancellation
    // settlement must match the corresponding By-ID rows.
    from(activityWorker) {
      // format: off
      on(activityWorker.poll) {
        when(Phase.scheduled).where(_.dispatch == Dispatch.now) ~> effects.startAttempt // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      }
      on(activityWorker.heartbeat) {
        when[Held] ~> effects.heartbeat
        when(Phase.unstarted, Phase.scheduled, Phase.paused) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
      }
      // format: on
      // A worker's answer settles the attempt it holds. A retryable failure backs a started attempt
      // off, settles a cancel-requested one as canceled and lands a pause-requested one in paused
      // (TransitionAttemptFailedWhilePauseRequested). A canceled answer needs a cancel request.
      // Where a worker holds the attempt: what start-to-close covers and a worker's answer settles.
      on(activityWorker.respondCompleted) {
        when[Held] ~> effects.complete
      }
      // A pending reset applies on either failure, before its retryability is read.
      on(activityWorker.respondFailed) {
        when(Phase.resetRequested) ~> effects.applyReset
        when(Phase.resetKeepingPause) ~> effects.applyReset
      }
      on(activityWorker.respondFailed(Failure.fatal)) {
        when(Phase.started, Phase.pauseRequested, Phase.cancelRequested) ~> effects.fail
      }

      on(activityWorker.respondFailed(Failure.retryable)) {
        when(Phase.started).where(states.retriesRemaining) ~> effects.backOff
        when(Phase.cancelRequested) ~> effects.cancel
        when(Phase.pauseRequested).where(states.retriesRemaining) ~> effects.backOffPaused
        when(Phase.started, Phase.pauseRequested).where(states.exhausted) ~> effects.fail
      }
      on(activityWorker.respondCanceled) {
        when(Phase.cancelRequested) ~> effects.cancel
        when(Phase.started, Phase.pauseRequested, Phase.resetRequested, Phase.resetKeepingPause) ~>
          rejects(Rejection.invalidArgument)
            .because(cancellationNotRequested)
      }
    }

    // Held completion, pending-reset failure, fatal failure, retry/exhaustion and cancellation
    // settlement must match worker. By-ID keeps its notFound checks, broader Live completion,
    // cancellation reason and row order.
    from(completionClient) {
      on(
        completionClient.respondCompletedByID,
        completionClient.respondFailedByID,
        completionClient.respondCanceledByID
      ) {
        when(Phase.unstarted) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
      }
      on(completionClient.respondCompletedByID) {
        when[Live] ~> effects.complete
      }
      on(completionClient.respondFailedByID) {
        when(Phase.resetRequested) ~> effects.applyReset
        when(Phase.resetKeepingPause) ~> effects.applyReset
      }
      on(completionClient.respondFailedByID(Failure.fatal)) {
        when(Phase.started, Phase.pauseRequested, Phase.cancelRequested) ~> effects.fail
      }
      on(completionClient.respondFailedByID(Failure.retryable)) {
        when(Phase.started).where(states.retriesRemaining) ~> effects.backOff
        when(Phase.pauseRequested).where(states.retriesRemaining) ~> effects.backOffPaused
        when(Phase.started, Phase.pauseRequested).where(states.exhausted) ~> effects.fail
        when(Phase.cancelRequested) ~> effects.cancel
      }
      on(completionClient.respondFailedByID, completionClient.respondCanceledByID) {
        when(Phase.scheduled, Phase.paused) ~> rejects(Rejection.notFound)
      }
      on(completionClient.respondCanceledByID) {
        when(Phase.cancelRequested) ~> effects.cancel
        when(Phase.started, Phase.pauseRequested, Phase.resetRequested, Phase.resetKeepingPause) ~>
          rejects(Rejection.invalidArgument)
            .because(cancellationNotRequestedByID)
      }
    }

    from(process) {
      on(process.stop) {
        always ~> effects.keep
      }
    }

    from(timers) {
      // format: off
      on(timers.startDelay) {
        when[Live].where(_.dispatch == Dispatch.startDelay) ~> effects.dispatchNow // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      }
      on(timers.backoff) {
        when[Live].where(_.dispatch == Dispatch.backoff) ~> effects.dispatchNow // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      }
    }
      // format: on
    from(deadline) {
      // Started and not over: the phases a deadline can fire in.
      on(deadline.scheduleToClose) {
        when[Live].where(states.scheduleToCloseArmed) ~> (effects.timeOut(
          _,
          TimeoutType.scheduleToClose
        ))
      }
      // Waiting for a worker: the phases before an attempt is held, which schedule-to-start covers.
      on(deadline.scheduleToStart) {
        when[Waiting].where(states.scheduleToStartArmed) ~> (effects.timeOut(
          _,
          TimeoutType.scheduleToStart
        ))
      }
      on(deadline.startToClose) {
        when(Phase.started).where(s =>
          states.startToCloseArmed(s) && states.retriesRemaining(s)
        ) ~> effects.backOffAfterDeadline
        when(Phase.pauseRequested).where(s =>
          states.startToCloseArmed(s) && states.retriesRemaining(s)
        ) ~> effects.backOffAfterDeadline
        when(Phase.started, Phase.pauseRequested, Phase.cancelRequested)
          .where(s => states.startToCloseArmed(s) && states.endsTerminally(s)) ~> (effects.timeOut(
          _,
          TimeoutType.startToClose
        ))
        when(Phase.resetRequested).where(states.startToCloseArmed) ~> effects.applyReset
        when(Phase.resetKeepingPause).where(states.startToCloseArmed) ~> effects.applyReset
      }
      on(deadline.heartbeat) {
        when(Phase.started).where(s => states.heartbeatArmed(s) && states.retriesRemaining(s)) ~>
          effects.heartbeatBackOff
        when(Phase.pauseRequested).where(s =>
          states.heartbeatArmed(s) && states.retriesRemaining(s)
        ) ~> effects.heartbeatBackOff
        when(Phase.started, Phase.pauseRequested, Phase.cancelRequested).where(s =>
          states.heartbeatArmed(s) && states.endsTerminally(s)
        ) ~> effects.heartbeatTimeOut
        when(Phase.resetRequested).where(states.heartbeatArmed) ~> effects.applyResetOnHeartbeat
        when(Phase.resetKeepingPause).where(states.heartbeatArmed) ~> effects.applyResetOnHeartbeat
      }
    }
  // format: off
  // The lifecycle-wide control and reset laws; focused settlement claims live in subject files.
  // The activity and worker's cross-entity claim is the composition's, and the dispatch record's
  // are Dispatch.scala's.
  object properties:
    val cancelIsNotUndone = property.holdsAcross { (before, after) =>
      (before.phase == Phase.cancelRequested) implies // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        (after.state.phase == Phase.cancelRequested || after.state.phase.in[Closed]) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    }

    val terminated = property when client.terminate holds { s =>
      s.state.phase == Phase.terminated && s.records(Fact.statusTerminated) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    }

    // A pending reset ends only as its held attempt does: a completion wins, a cancellation or a
    // terminate replaces it, schedule-to-close stays terminal, and a failure or one of the
    // attempt's own deadlines applies it, whatever the policy left. The next attempt is a first
    // one, dispatched at once, and paused only where the pause was kept; no failure or timeout is
    // recorded for it. Any other step leaves it as it was. With no reset pending, an existing
    // attempt's count never goes back but where a waiting or paused activity restarts, with no
    // backoff left.
    val resetSettles = property.holdsAcross { (before, after) =>
      val restarted = after.state.attempts == 0 && after.state.dispatch == Dispatch.now && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        after.state.scheduleToClose == before.scheduleToClose && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        after.state.scheduleToStart == before.scheduleToStart && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        after.state.startToClose == before.startToClose && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        after.state.heartbeat == before.heartbeat && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        after.state.maxAttempts == before.maxAttempts && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        !after.records(Fact.statusFailed) &&
        !after.records(Fact.statusTimedOut(TimeoutType.scheduleToClose)) &&
        !after.records(Fact.statusTimedOut(TimeoutType.scheduleToStart)) &&
        !after.records(Fact.statusTimedOut(TimeoutType.startToClose)) &&
        !after.records(Fact.statusTimedOut(TimeoutType.heartbeat))
      if states.pendingReset(before) then
        after.state == before || // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        (after.state.phase == Phase.completed && after.records(Fact.statusCompleted)) || // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        after.state.phase == Phase.cancelRequested || // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        after.state.phase == Phase.terminated || // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        (after.state.phase == Phase.timedOut && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
          after.records(Fact.statusTimedOut(TimeoutType.scheduleToClose))) ||
        (restarted && after.state.phase == states.landingPhase(before) && // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
          after.records(states.landingStatus(before)))
      else
        before.phase == Phase.unstarted || after.state.attempts >= before.attempts || // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        (before.phase.in(Phase.scheduled, Phase.paused) &&
          after.state.phase.in(Phase.scheduled, Phase.paused) &&
          after.state.attempts == 0 && after.state.dispatch != Dispatch.backoff) // scalafix:ok DisableSyntax.==, DisableSyntax.!=; strictEquality checks native comparisons
    }

    // A reset of a waiting or paused activity restarts it at once: its count is zero and a retry
    // backoff is discarded, a start delay kept. Without keep_paused it is scheduled again.
    val resetResumes = property.when(client.reset(ResetPause.resume)) holdsAcross {
      (before, after) =>
        !before.phase.in(Phase.scheduled, Phase.paused) ||
        (after.state == before.copy( // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
          phase = Phase.scheduled,
          attempts = UpTo(0),
          dispatch = states.resetDispatch(before.dispatch)
        ) && after.records(Fact.statusScheduled))
    }

    // With keep_paused a paused activity stays paused, and a scheduled one is scheduled again.
    val resetKeepsPaused =
      property.when(client.reset(ResetPause.keepPaused)) holdsAcross { (before, after) =>
        !before.phase.in(Phase.scheduled, Phase.paused) ||
        (after.state == before.copy( // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
          attempts = UpTo(0),
          dispatch = states.resetDispatch(before.dispatch)
        ) && after.records(
          if before.phase == Phase.paused then Fact.statusPaused else Fact.statusScheduled // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        ))
      }
  // format: on
    // Retryable's and Expirable's laws as the activity keeps them: a pending reset applies first, as
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
    // format: off
    // Cancel > Reset > Pause: no step makes a reset pending over a pending cancellation, and no
    // step turns a pending reset back into a pause request or a plain attempt (model.go:156-158).
    val controlPrecedence = property.holdsAcross { (before, after) =>
      (before.phase != Phase.cancelRequested || !states.pendingReset(after.state)) && // scalafix:ok DisableSyntax.!=; strictEquality checks native comparisons
      (!states.pendingReset(before) || !after.state.phase.in(Phase.started, Phase.pauseRequested))
    }
    // format: on
  // A terminate settles it, and DescribeActivityExecution reports its status by `activityStatus`.
  // The terminate find starts the activity and stops the worker before the control. The held
  // cancellation-request claim is authored on ByIDCancellation's complete completion client path below.
  object capabilities extends Capabilities:
    val terminable: Capability = Terminable(
      terminate = client.terminate,
      settled = Fact.statusTerminated,
      reach = Seq(client.start(), process.stop),
      expect = inconclusive(Reason.explanationsDisagree)
    )
    val describable: Capability = Describable(statusTable = activityStatus)
    val retryableFailure: Capability = Retryable[State, Phase, MaxAttempts.Bound](
      failure = activityWorker.respondFailed(Failure.retryable),
      retryable = true,
      attemptCount = states.attemptCount,
      maximumAttempts = states.maximumAttempts,
      retriesRemaining = states.retriesRemaining,
      pendingPause = Some(states.pendingPause),
      pendingCancel = Some(states.pendingCancel)
    )
    val fatalFailure: Capability = Retryable[State, Phase, MaxAttempts.Bound](
      failure = activityWorker.respondFailed(Failure.fatal),
      retryable = false,
      attemptCount = states.attemptCount,
      maximumAttempts = states.maximumAttempts,
      retriesRemaining = states.retriesRemaining,
      pendingPause = Some(states.pendingPause),
      pendingCancel = Some(states.pendingCancel)
    )
    val scheduleToCloseDeadline: Capability = Expirable[State, Phase, Live, Fact](
      timer = deadline.scheduleToClose,
      armed = states.scheduleToCloseArmed,
      timeout = Fact.statusTimedOut(TimeoutType.scheduleToClose),
      timeoutFacts = states.timeoutFacts
    )
    val scheduleToStartDeadline: Capability = Expirable[State, Phase, Waiting, Fact](
      timer = deadline.scheduleToStart,
      armed = states.scheduleToStartArmed,
      timeout = Fact.statusTimedOut(TimeoutType.scheduleToStart),
      timeoutFacts = states.timeoutFacts
    )
    val startToCloseDeadline: Capability = Expirable[State, Phase, Held, Fact](
      timer = deadline.startToClose,
      armed = states.startToCloseArmed,
      timeout = Fact.statusTimedOut(TimeoutType.startToClose),
      timeoutFacts = states.timeoutFacts,
      retryable = true,
      retriesRemaining = Some(states.retriesRemaining),
      pendingPause = Some(states.pendingPause),
      pendingCancel = Some(states.pendingCancel)
    )
    val heartbeatDeadline: Capability = Expirable[State, Phase, Held, Fact](
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
      Retryable.failureReturnsToWaiting[State, Phase] -> properties.resetFailureReturnsToWaiting,
      because = states.pendingResetReason,
      of = Seq(retryableFailure, fatalFailure)
    )
    overriding(
      Retryable.failureEndsFailed[State, Phase] -> properties.resetFailureEndsFailed,
      because = states.pendingResetReason,
      of = Seq(retryableFailure, fatalFailure)
    )
    overriding(
      Retryable.failurePauses[State, Phase] -> properties.resetFailurePauses,
      because = states.pendingResetReason,
      of = Seq(retryableFailure, fatalFailure)
    )
    overriding(
      (Expirable.deadlineTimesOut[State, Phase]: AnyRef) ->
        (properties.resetDeadlineTimesOut: AnyRef),
      because = states.pendingResetReason,
      of = Seq(startToCloseDeadline, heartbeatDeadline)
    )
    overriding(
      (Expirable.deadlineReturnsToWaiting[State, Phase]: AnyRef) ->
        (properties.resetDeadlineReturnsToWaiting: AnyRef),
      because = states.pendingResetReason,
      of = Seq(startToCloseDeadline, heartbeatDeadline)
    )
    overriding(
      (Expirable.deadlinePauses[State, Phase]: AnyRef) ->
        (properties.resetDeadlinePauses: AnyRef),
      because = states.pendingResetReason,
      of = Seq(startToCloseDeadline, heartbeatDeadline)
    )

  // Capability laws and the lifecycle-wide control and reset Queries.
  object queries:
    capabilities.bound(
      three,
      Retryable.failureReturnsToWaiting[State, Phase] -> eight,
      Retryable.failureEndsFailed[State, Phase] -> eight,
      Retryable.failurePauses[State, Phase] -> eight,
      Retryable.failureCancels[State, Phase] -> eight,
      Retryable.attemptCountIsWithinPolicy[State, MaxAttempts.Bound] -> eight,
      Expirable.firesInWindow[State, Phase, Live] -> eight,
      (Expirable.deadlineTimesOut[State, Phase]: AnyRef) -> eight,
      (Expirable.deadlineReturnsToWaiting[State, Phase]: AnyRef) -> eight,
      (Expirable.deadlinePauses[State, Phase]: AnyRef) -> eight
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
