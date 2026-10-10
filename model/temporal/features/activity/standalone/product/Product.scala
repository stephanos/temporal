// The standalone activity's Product: what DescribeActivityExecution reports, the level a client
// reads (fn-126 decision 16). The level's own file holds the product machine, ActivityProduct, which
// refines nothing; system/System.scala refines it. The two package clauses read the feature's
// package as well as this one, so its types and signature are in scope.
package temporal
package features.activity
package standalone
package product

import scala.annotation.unused
import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import temporal.capabilities.*
import Bounds.three

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

// What DescribeActivityExecution shows. Each case declares the status fact a step that enters it
// records.
enum Phase(val status: Fact) extends Recorded[Fact] derives Finite:
  case scheduled extends Phase(Fact.statusScheduled), Waiting
  case started extends Phase(Fact.statusStarted), Held
  case paused extends Phase(Fact.statusPaused), Suspended
  case cancelRequested extends Phase(Fact.statusCancelRequested), Held
  case completed extends Phase(Fact.statusCompleted), Succeeded
  case failed extends Phase(Fact.statusFailed), Failed
  case canceled extends Phase(Fact.statusCanceled), Canceled
  case terminated extends Phase(Fact.statusTerminated), Terminated
  case timedOut extends Phase(Fact.statusTimedOut), TimedOut

final case class State(phase: Phase) derives Finite

enum Fact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested
  case statusCompleted, statusFailed, statusCanceled, statusTerminated, statusTimedOut
  case heartbeatReceived, heartbeatTimedOut

// The accessors the blocks of ActivityProduct read and assign a state's phase by: the setter hands
// the draft the phase it assigns, whose status the step records.
def phase(using v: View[State]): Phase = v.get(_.phase)
def phase_=(p: Phase)(using d: Draft[State, ?, Fact]): Unit = d.set(p)(_.copy(phase = p))

// ### The product machine: what DescribeActivityExecution shows, with no account of how. A retry
// reads as scheduled again, a pause of a running attempt as started until the worker yields.

// Every status the product machine records is confirmed by the status observation of its name.
object ActivityProduct extends Machine[State, Outcome, Fact], Phased[State, Phase](_.phase):
  val init = product.State(Phase.scheduled)
  override def end(s: State) = s.phase.in[Closed]

  object states:
    def status(s: State) = s.phase // format: off
    val paused = is(phase == Phase.paused) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
    // format: on
  object effects:
    val pauseApplied = choice
    val pausePending = choice
    val pauseAlreadyRequested = choice
    def schedule(
        s: State,
        @unused scheduleToClose: Timeout,
        @unused scheduleToStart: Timeout,
        @unused startToClose: Timeout,
        @unused heartbeat: Timeout,
        @unused startDelay: Timeout,
        @unused maxAttempts: MaxAttempts
    ) = enter(s, Fact.statusScheduled)
    val startAttempt = effect { phase = Phase.started }
    val complete = effect { phase = Phase.completed }
    val fail = effect { phase = Phase.failed }
    val retry = effect { phase = Phase.scheduled }
    val retryScheduled = choice
    val retryExhausted = choice
    val retryPaused = choice
    def serviceRetry(s: State) = choose(
      retryScheduled -> retry(s),
      retryPaused -> pause(s),
      retryExhausted -> fail(s)
    )
    val cancel = effect { phase = Phase.canceled }
    val pause = effect { phase = Phase.paused }
    // started also reads a held attempt whose pause is pending (System.refinement.toProduct).
    def pauseHeld(s: State) = choose(
      pauseApplied -> pause(s),
      pausePending -> enter(s, Fact.statusPaused),
      pauseAlreadyRequested -> reject(Outcome.rejected(Rejection.failedPrecondition), s)
        .because(pauseAlreadyRequestedReason)
    )
    val resume = effect { phase = Phase.scheduled }
    val pauseWithdrawn = choice
    val unpauseNotRequested = choice
    def unpauseHeld(s: State) = choose(
      pauseWithdrawn -> startAttempt(s),
      unpauseNotRequested -> reject(Outcome.rejected(Rejection.failedPrecondition), s)
        .because(activityNotPaused)
    )
    val requestCancel = effect { phase = Phase.cancelRequested }
    // A reset of a waiting activity applies at once; of a held attempt it waits for the attempt to
    // end, which Describe still reads as started, or as pause-requested where a pause is kept.
    def resetWaiting(s: State) = retry(s)
    def resetKeepsPaused(s: State) = pause(s)
    val resetPending = choice
    val resetPausePending = choice
    val resetAlreadyRequested = choice
    def resetHeld(s: State) = choose(
      resetPending -> startAttempt(s),
      resetAlreadyRequested -> reject(Outcome.rejected(Rejection.failedPrecondition), s)
        .because(resetAlreadyPending)
    )
    def resetHeldKeepingPause(s: State) = choose(
      resetPending -> startAttempt(s),
      resetPausePending -> enter(s, Fact.statusPaused),
      resetAlreadyRequested -> reject(Outcome.rejected(Rejection.failedPrecondition), s)
        .because(resetAlreadyPending)
    )
    // A held attempt's failure ends the activity, or applies a pending reset whatever the
    // failure's retryability: the activity is scheduled again, or paused where a pause was kept.
    val failureFatal = choice
    val resetApplied = choice
    val resetAppliedPaused = choice
    def failOrReset(s: State) = choose(
      failureFatal -> fail(s),
      resetApplied -> retry(s),
      resetAppliedPaused -> pause(s)
    )
    val terminate = effect { phase = Phase.terminated }
    val timeOut = effect { phase = Phase.timedOut }
    def heartbeat(s: State) = enter(s, Fact.heartbeatReceived)
    val heartbeatRetry = choice
    val heartbeatPaused = choice
    val heartbeatExhausted = choice
    def heartbeatExpires(s: State) = choose(
      heartbeatRetry -> enter(
        s.copy(phase = Phase.scheduled),
        Fact.statusScheduled,
        Fact.heartbeatTimedOut
      ),
      heartbeatPaused -> enter(
        s.copy(phase = Phase.paused),
        Fact.statusPaused,
        Fact.heartbeatTimedOut
      ),
      heartbeatExhausted -> heartbeatTimeOut(s)
    )
    def heartbeatTimeOut(s: State) =
      enter(s.copy(phase = Phase.timedOut), Fact.statusTimedOut, Fact.heartbeatTimedOut)

  object rules extends Rules:
    from(client) {
      on(client.start) {
        when(Phase.scheduled) ~> effects.schedule
      }

      // Closed controls answer NotFound before the live-state checks (operator_commands.go:249-254).
      on(client.pause, client.unpause, client.requestCancel, client.terminate) {
        when[Closed] ~> rejects(Rejection.notFound)
      }
      on(client.pause) {
        when(Phase.scheduled) ~> effects.pause
        when(Phase.started) ~> effects.pauseHeld
        when(Phase.paused, Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(pausedOrCancelPending)
      }
      on(client.unpause) {
        where(states.paused) ~> effects.resume
        when(Phase.started) ~> effects.unpauseHeld
        when(Phase.scheduled, Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(activityNotPaused)
      }
      on(client.requestCancel) {
        when(Phase.scheduled, Phase.paused) ~> effects.cancel
        when(Phase.started) ~> effects.requestCancel
        when(Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(cancellationAlreadyRequested)
      }
      on(client.terminate) {
        when(
          Phase.scheduled,
          Phase.started,
          Phase.paused,
          Phase.cancelRequested
        ) ~> effects.terminate
      }
      on(client.reset) {
        when[Closed] ~> rejects(Rejection.notFound)
        when(Phase.scheduled) ~> effects.resetWaiting
        when(Phase.cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because(resetWithPendingCancellation)
      }
      on(client.reset(ResetPause.resume)) {
        where(states.paused) ~> effects.resume
        when(Phase.started) ~> effects.resetHeld
      }
      on(client.reset(ResetPause.keepPaused)) {
        where(states.paused) ~> effects.resetKeepsPaused
        when(Phase.started) ~> effects.resetHeldKeepingPause
      }
    }

    // Held completion, failure/reset/retry and cancellation settlement must match the By-ID rows.
    from(activityWorker) {
      on(activityWorker.poll) {
        when(Phase.scheduled) ~> effects.startAttempt
      }
      on(activityWorker.heartbeat) {
        when[Held] ~> effects.heartbeat
        when(Phase.scheduled, Phase.paused) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
      }

      // A worker's answer settles an attempt it holds. A retryable failure is retried, or canceled
      // under a cancel request; a canceled answer settles only an activity whose cancellation was
      // requested.
      on(activityWorker.respondCompleted) {
        when[Held] ~> effects.complete
      }
      on(activityWorker.respondFailed(Failure.fatal)) {
        when(Phase.started) ~> effects.failOrReset
        when(Phase.cancelRequested) ~> effects.fail
      }

      on(activityWorker.respondFailed(Failure.retryable)) {
        when(Phase.started) ~> effects.serviceRetry
        when(Phase.cancelRequested) ~> effects.cancel
      }
      on(activityWorker.respondCanceled) {
        when(Phase.cancelRequested) ~> effects.cancel
        when(Phase.started) ~> rejects(Rejection.invalidArgument)
          .because(cancellationNotRequested)
      }
    }

    // Held completion, failure/reset/retry and cancellation settlement must match the worker rows.
    // By-ID keeps its notFound checks, broader Live completion and cancellation reason.
    from(completionClient) {
      on(
        completionClient.respondCompletedByID,
        completionClient.respondFailedByID,
        completionClient.respondCanceledByID
      ) {
        when[Closed] ~> rejects(Rejection.notFound)
      }
      on(completionClient.respondCompletedByID) {
        when[Live] ~> effects.complete
      }
      on(completionClient.respondFailedByID(Failure.fatal)) {
        when(Phase.started) ~> effects.failOrReset
        when(Phase.cancelRequested) ~> effects.fail
      }
      on(completionClient.respondFailedByID(Failure.retryable)) {
        when(Phase.started) ~> effects.serviceRetry
        when(Phase.cancelRequested) ~> effects.cancel
      }
      on(completionClient.respondFailedByID, completionClient.respondCanceledByID) {
        when(Phase.scheduled, Phase.paused) ~> rejects(Rejection.notFound)
      }
      on(completionClient.respondCanceledByID) {
        when(Phase.cancelRequested) ~> effects.cancel
        when(Phase.started) ~> rejects(Rejection.invalidArgument)
          .because(cancellationNotRequestedByID)
      }
    }

    // The worker process stopping is not the product's: a client reads nothing of it, so the
    // product binds no stop, and the System keeps the stop as a step that changes nothing.
    from(timers) {
      on(timers.timeout) {
        when(Phase.scheduled, Phase.started, Phase.paused, Phase.cancelRequested) ~> effects.timeOut
      }
    }
    on(deadline.heartbeat) {
      when(Phase.started) ~> effects.heartbeatExpires
      when(Phase.cancelRequested) ~> effects.heartbeatTimeOut
    }

  object properties:
    // The public cancelRequested phase can mean either that a worker still holds an attempt or that
    // a control followed a paused activity. The product state cannot tell those histories apart, so
    // its pause property keeps the former, narrower meaning of "running": started. The protocol
    // machine has enough state to use Pausable's role-derived Property directly.
    def pausedDoesNotStartAttempt(m: Declares[State]): Property[State] =
      m.property // format: off
        .never(s => s.state.phase == Phase.started) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        .from(_.phase == Phase.paused) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        // format: on
  object capabilities extends Capabilities:
    val closable: Capability = Closable(
      rejected = Outcome.rejected(Rejection.notFound)
    )
    val pausable: Capability = Pausable(
      pause = client.pause,
      unpause = client.unpause
    )
    val pollable: Capability = Pollable(dispatch = activityWorker.poll)
    overriding(
      Pausable.pausedIsNotDispatched[State, Phase] -> properties.pausedDoesNotStartAttempt,
      because = "cancelRequested does not reveal whether a worker still holds an attempt"
    )

  object queries:
    capabilities.bound(three)
