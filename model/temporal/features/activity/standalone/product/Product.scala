// The standalone activity's Product: what DescribeActivityExecution reports, the level a client
// reads (fn-126 decision 16). The level's own file holds the product machine, ActivityProduct, which
// refines nothing; system/System.scala refines it. The two package clauses read the feature's
// package as well as this one, so its types and signature are in scope.
package temporal
package features.activity.standalone
package product

import umpire.*
import temporal.capabilities.{given, *}
import shared.Bounds.three
import shared.worker.worker as process

// What DescribeActivityExecution shows.
enum Phase derives Finite:
  case scheduled, started, paused, cancelRequested
  case completed, failed, canceled, terminated, timedOut

final case class State(phase: Phase) derives Finite

enum Fact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested
  case statusCompleted, statusFailed, statusCanceled, statusTerminated, statusTimedOut

// ### The product machine: what DescribeActivityExecution shows, with no account of how. A retry
// reads as scheduled again, a pause of a running attempt as started until the worker yields.

// Every status the product machine records is confirmed by the status observation of its name.
object ActivityProduct extends Machine[State, Outcome, Fact]:
  import Phase.*

  val init = product.State(scheduled)
  def end(s: State) = states.over(s)

  object states:
    def phase(s: State) = s.phase
    def terminal(p: Phase) = p.in(completed, failed, canceled, terminated, timedOut)
    def over(s: State) = terminal(s.phase)
    def paused(s: State) = s.phase == Phase.paused
    def running(s: State) = s.phase == started
    def held(s: State) = s.phase.in(started, cancelRequested)
    def pausable(s: State) = s.phase.in(scheduled, started)
    val notFoundCode = "chasm/lib/activity/activity.go"

  object effects:
    import Fact.*
    def startAttempt(s: State) = enter(s.copy(phase = started), statusStarted)
    def complete(s: State) = enter(s.copy(phase = completed), statusCompleted)
    def fail(s: State) = enter(s.copy(phase = failed), statusFailed)
    def retry(s: State) = enter(s.copy(phase = scheduled), statusScheduled)
    def cancel(s: State) = enter(s.copy(phase = canceled), statusCanceled)
    def pause(s: State) = enter(s.copy(phase = Phase.paused), statusPaused)
    def resume(s: State) = enter(s.copy(phase = scheduled), statusScheduled)
    def requestCancel(s: State) = enter(s.copy(phase = cancelRequested), statusCancelRequested)
    def terminate(s: State) = enter(s.copy(phase = terminated), statusTerminated)
    def timeOut(s: State) = enter(s.copy(phase = timedOut), statusTimedOut)
    def notFound(s: State) = reject(Outcome.notFound, s)

  object rules extends Rules(_.phase):
    on(worker.poll)(in(scheduled) ~> effects.startAttempt)

    // A worker's answer settles an attempt it holds. A retryable failure is retried, or canceled
    // under a cancel request; a canceled answer settles only an activity whose cancellation was
    // requested.
    on(worker.respond(AttemptResult.completed))(where(states.held) ~> effects.complete)
    on(worker.respond(AttemptResult.failed(false)))(where(states.held) ~> effects.fail)
    on(worker.respond(AttemptResult.failed(true))) {
      in(started) ~> effects.retry
      in(cancelRequested) ~> effects.cancel
    }
    on(worker.respond(AttemptResult.canceled))(in(cancelRequested) ~> effects.cancel)

    // A control on an activity that is over is not found. A pause of a paused or cancel-requested
    // activity, or an unpause of one not paused, is FailedPrecondition; the System lists them.
    on(client.control)(in(states.terminal) ~> effects.notFound)
    on(client.control(Control.pause))(where(states.pausable) ~> effects.pause)
    on(client.control(Control.unpause))(where(states.paused) ~> effects.resume)
    on(client.control(Control.requestCancel)) {
      in(scheduled, started, paused, cancelRequested) ~> effects.requestCancel
    }
    on(client.control(Control.terminate)) {
      in(scheduled, started, paused, cancelRequested) ~> effects.terminate
    }

    // The worker stopping is a fault the Run records and the activity does not feel.
    disabled(process.stop)
    on(timers.timeout)(in(scheduled, started, paused, cancelRequested) ~> effects.timeOut)

  object implements
      extends Implements(limits = three)(
        Closable(
          status = states.phase,
          terminal = states.terminal,
          rejected = cited(Outcome.notFound, states.notFoundCode)
        ),
        Pausable(
          pause = client.control(Control.pause),
          unpause = client.control(Control.unpause),
          paused = states.paused
        ),
        Pollable(dispatch = worker.poll, running = states.running)
      )
