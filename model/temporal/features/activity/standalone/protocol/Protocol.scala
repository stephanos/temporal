// The standalone activity's lifecycle System and its Product refinement.
// Derived subject models and dispatch compositions live beside it in subject files.
package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
import framework.realize.Reason
import temporal.capabilities.system.*
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

object ActivityProtocol
    extends Machine[State, Outcome, Fact],
      Phased[State, Phase](_.phase),
      Dispatching,
      Pausing,
      Cancellation,
      Lifecycle,
      Reset,
      Heartbeat,
      Completion,
      Retry,
      Timeouts:
  // Where every path begins: before the activity exists, with every deadline at its first value.
  val init = protocol.State(
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
    def pendingPause(s: State): Boolean = s.phase == Phase.pauseRequested
    def pendingCancel(s: State): Boolean = s.phase == Phase.cancelRequested
    def pendingReset(s: State): Boolean = s.phase.in(Phase.resetRequested, Phase.resetKeepingPause)
    def landingPhase(s: State): Phase =
      if s.phase.in(Phase.pauseRequested, Phase.resetKeepingPause) then Phase.paused
      else Phase.scheduled
    def landingStatus(s: State): Fact =
      if landingPhase(s) == Phase.paused then Fact.statusPaused else Fact.statusScheduled

    // A reset discards a retry backoff but keeps a start delay (operator_commands.go:410-416).
    def resetDispatch(d: Dispatch): Dispatch = if d == Dispatch.backoff then Dispatch.now else d

    // A pending reset applied: a first attempt, dispatched at once, paused only where it was kept.
    def resetApplied(before: State, after: State): Boolean =
      after.attempts == 0 && after.dispatch == Dispatch.now &&
        after.phase == (if before.phase == Phase.resetKeepingPause then Phase.paused
                        else Phase.scheduled)

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

    val pendingResetReason =
      "a pending reset applies first (chasm/lib/activity/attempt.go:201-216)"

    // Completed on the second attempt of an activity with no deadline set.
    val completedOnRetry =
      ActivityProtocol.init.copy(phase = Phase.completed, attempts = UpTo(attemptBound))

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

  // Each action's rules sit in the facet that owns it (fn-161), in this order.
  object rules
      extends Facets(
        dispatching.rules,
        pausing.rules,
        cancellation.rules,
        lifecycle.rules,
        reset.rules,
        heartbeat.rules,
        completion.rules,
        retry.rules,
        timeouts.rules
      )

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
      if states.pendingReset(before) then
        after.state == before ||
        (after.state.phase == Phase.completed && after.records(Fact.statusCompleted)) ||
        after.state.phase == Phase.cancelRequested ||
        after.state.phase == Phase.terminated ||
        (after.state.phase == Phase.timedOut &&
          after.records(Fact.statusTimedOut(TimeoutType.scheduleToClose))) ||
        (restarted && after.state.phase == states.landingPhase(before) &&
          after.records(states.landingStatus(before)))
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
          dispatch = states.resetDispatch(before.dispatch)
        ) && after.records(Fact.statusScheduled))
    }

    // With keep_paused a paused activity stays paused, and a scheduled one is scheduled again.
    val resetKeepsPaused =
      property.when(client.reset(ResetPause.keepPaused)) holdsAcross { (before, after) =>
        !before.phase.in(Phase.scheduled, Phase.paused) ||
        (after.state == before.copy(
          attempts = UpTo(0),
          dispatch = states.resetDispatch(before.dispatch)
        ) && after.records(
          if before.phase == Phase.paused then Fact.statusPaused else Fact.statusScheduled
        ))
      }
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
    // Cancel > Reset > Pause: no step makes a reset pending over a pending cancellation, and no
    // step turns a pending reset back into a pause request or a plain attempt (model.go:156-158).
    val controlPrecedence = property.holdsAcross { (before, after) =>
      (before.phase != Phase.cancelRequested || !states.pendingReset(after.state)) &&
      (!states.pendingReset(before) || !after.state.phase.in(Phase.started, Phase.pauseRequested))
    }

  // The capabilities each facet declares for the actions it owns.
  object capabilities
      extends FacetCapabilities(
        lifecycle.capabilities,
        retry.capabilities,
        timeouts.capabilities,
        heartbeat.capabilities
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
        .live(
          inconclusive(
            Reason.explanationsDisagree,
            because =
              "a control of an activity that is over is not found and records nothing, so a second terminate is a step no evidence reports, on which the claim fails"
          )
        )
