// The standalone activity's Product: what DescribeActivityExecution reports, the level a client
// reads (fn-126 decision 16). The level's own file holds the product machine, ActivityProduct, which
// refines nothing; system/System.scala refines it. The two package clauses read the feature's
// package as well as this one, so its types and signature are in scope.
package temporal
package features.activity
package standalone
package product

import scala.annotation.unused
import umpire.*
import umpire.outcomes.{Outcome, Rejection}
import temporal.capabilities.*
import shared.Bounds.three
import shared.worker.worker as process

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
  import Phase.*

  val init = product.State(scheduled)
  override def end(s: State) = s.phase.in[Closed]

  object states:
    def status(s: State) = s.phase
    val paused = is(phase == Phase.paused)

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
    val startAttempt = effect { phase = started }
    val complete = effect { phase = completed }
    val fail = effect { phase = failed }
    val retry = effect { phase = scheduled }
    val retryScheduled = choice
    val retryExhausted = choice
    def retryOrFail(s: State) = choose(
      retryScheduled -> retry(s),
      retryExhausted -> fail(s)
    )
    val retryPaused = choice
    def serviceRetry(s: State) = choose(
      retryScheduled -> retry(s),
      retryPaused -> enter(s.copy(phase = Phase.paused), Fact.statusPaused),
      retryExhausted -> fail(s)
    )
    val cancel = effect { phase = canceled }
    val pause = effect { phase = Phase.paused }
    // started also reads a held attempt whose pause is pending (System.refinement.toProduct).
    def pauseHeld(s: State) = choose(
      pauseApplied -> pause(s),
      pausePending -> enter(s, Fact.statusPaused),
      pauseAlreadyRequested -> reject(Outcome.rejected(Rejection.failedPrecondition), s)
        .because("pause already requested (chasm/lib/activity/model/model.go:232)")
    )
    val resume = effect { phase = scheduled }
    val pauseWithdrawn = choice
    val unpauseNotRequested = choice
    def unpauseHeld(s: State) = choose(
      pauseWithdrawn -> enter(s, Fact.statusStarted),
      unpauseNotRequested -> reject(Outcome.rejected(Rejection.failedPrecondition), s)
        .because("activity is not paused (chasm/lib/activity/model/model.go:251)")
    )
    val requestCancel = effect { phase = cancelRequested }
    val terminate = effect { phase = terminated }
    val timeOut = effect { phase = timedOut }
    def heartbeat(s: State) = enter(s, Fact.heartbeatReceived)
    val heartbeatRetry = choice
    val heartbeatPaused = choice
    val heartbeatExhausted = choice
    def heartbeatExpires(s: State) = choose(
      heartbeatRetry -> enter(
        s.copy(phase = scheduled),
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
      enter(s.copy(phase = timedOut), Fact.statusTimedOut, Fact.heartbeatTimedOut)

  object rules extends Rules:
    from(client) {
      import client.*

      on(start) {
        when(scheduled) ~> effects.schedule
      }

      // Closed controls answer NotFound before the live-state checks (operator_commands.go:249-254).
      on(pause, unpause, requestCancel, terminate) {
        when[Closed] ~> rejects(Rejection.notFound)
      }
      on(pause) {
        when(scheduled) ~> effects.pause
        when(started) ~> effects.pauseHeld
        when(paused, cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because("already paused or cancellation pending (chasm/lib/activity/model/model.go:232)")
      }
      on(unpause) {
        where(states.paused) ~> effects.resume
        when(started) ~> effects.unpauseHeld
        when(scheduled, cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because("activity is not paused (chasm/lib/activity/model/model.go:251)")
      }
      on(requestCancel) {
        when(scheduled, paused) ~> effects.cancel
        when(started) ~> effects.requestCancel
        when(cancelRequested) ~> rejects(Rejection.failedPrecondition)
          .because("cancellation already requested (chasm/lib/activity/model/model.go:201-202)")
      }
      on(terminate) {
        when(scheduled, started, paused, cancelRequested) ~> effects.terminate
      }
    }

    from(temporal.features.activity.standalone.worker) {
      import temporal.features.activity.standalone.worker.*

      on(poll) {
        when(scheduled) ~> effects.startAttempt
      }
      on(heartbeat) {
        when[Held] ~> effects.heartbeat
        when(scheduled, paused) ~> rejects(Rejection.notFound)
        when[Closed] ~> rejects(Rejection.notFound)
      }

      // A worker's answer settles an attempt it holds. A retryable failure is retried, or canceled
      // under a cancel request; a canceled answer settles only an activity whose cancellation was
      // requested.
      on(respondCompleted) {
        when[Held] ~> effects.complete
      }
      on(respondFailed(Failure.fatal)) {
        when[Held] ~> effects.fail
      }

      on(respondFailed(Failure.retryable)) {
        when(started) ~> effects.retryOrFail
        when(cancelRequested) ~> effects.cancel
      }
      on(respondCanceled) {
        when(cancelRequested) ~> effects.cancel
        when(started) ~> rejects(Rejection.invalidArgument)
          .because("cancellation was not requested (chasm/lib/activity/model/model.go:171)")
      }
    }

    from(service) {
      import service.*

      on(respondCompletedByID, respondFailedByID, respondCanceledByID) {
        when[Closed] ~> rejects(Rejection.notFound)
      }
      on(respondCompletedByID) {
        when[Live] ~> effects.complete
      }
      on(respondFailedByID(Failure.fatal)) {
        when[Held] ~> effects.fail
      }
      on(respondFailedByID(Failure.retryable)) {
        when(started) ~> effects.serviceRetry
        when(cancelRequested) ~> effects.cancel
      }
      on(respondFailedByID, respondCanceledByID) {
        when(scheduled, paused) ~> rejects(Rejection.notFound)
      }
      on(respondCanceledByID) {
        when(cancelRequested) ~> effects.cancel
        when(started) ~> rejects(Rejection.invalidArgument)
          .because("cancellation was not requested")
      }
    }

    // The worker stopping is a fault the Run records and the activity does not feel.
    disabled(process.stop)

    from(timers) {
      import timers.*

      on(timeout) {
        when(scheduled, started, paused, cancelRequested) ~> effects.timeOut
      }
    }
    on(deadline.heartbeat) {
      when(started) ~> effects.heartbeatExpires
      when(cancelRequested) ~> effects.heartbeatTimeOut
    }

  object properties:
    // The public cancelRequested phase can mean either that a worker still holds an attempt or that
    // a control followed a paused activity. The product state cannot tell those histories apart, so
    // its pause property keeps the former, narrower meaning of "running": started. The protocol
    // machine has enough state to use Pausable's role-derived Property directly.
    def pausedDoesNotStartAttempt(m: Declares[State]): Property[State] =
      m.property
        .never(s => s.state.phase == started)
        .from(_.phase == paused)

  object capabilities extends Capabilities:
    val closable: Capability = Closable(
      rejected = Outcome.rejected(Rejection.notFound)
    )
    val pausable: Capability = Pausable(
      pause = client.pause,
      unpause = client.unpause
    )
    val pollable: Capability = Pollable(dispatch = worker.poll)
    overriding(
      Pausable.pausedIsNotDispatched[State, Phase] -> properties.pausedDoesNotStartAttempt,
      because = "cancelRequested does not reveal whether a worker still holds an attempt"
    )

  object queries:
    capabilities.bound(three)
