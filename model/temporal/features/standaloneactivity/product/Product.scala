/* The standalone activity's Product: what DescribeActivityExecution reports, the level a client
 * reads (fn-126 decision 16). The level's own file holds the product machine, ActivityProduct, which
 * refines nothing; system/System.scala refines it. The two package clauses read the feature's
 * package as well as this one, so its types and signature are in scope.
 */
package temporal
package features.standaloneactivity
package product

import umpire.*
import temporal.capabilities.{given, *}
import shared.Bounds.three
import shared.worker.worker as process

/** What DescribeActivityExecution shows. */
enum Phase derives Finite:
  case scheduled, started, paused, cancelRequested
  case completed, failed, canceled, terminated, timedOut

final case class State(phase: Phase) derives Finite

enum Fact derives Finite:
  case statusScheduled, statusStarted, statusPaused, statusCancelRequested
  case statusCompleted, statusFailed, statusCanceled, statusTerminated, statusTimedOut

// ### The product machine: what DescribeActivityExecution shows, with no account of how. A retry
// reads as scheduled again, a pause of a running attempt as started until the worker yields.

/** Every status the product machine records is confirmed by the status observation of its name. */
object ActivityProduct extends Machine[State, Outcome, Fact]:
  import Phase.*

  val init = product.State(scheduled)
  def end(s: State) = states.over(s)

  /** The product's status sets and the constant its capabilities cite. */
  object states:
    def phase(s: State) = s.phase

    def terminal(p: Phase) = p.in(completed, failed, canceled, terminated, timedOut)

    /** A path ends where the activity is over: `end` reads this named predicate. */
    def over(s: State) = terminal(s.phase)

    /** A paused activity, which no worker is given. */
    def paused(s: State) = s.phase == Phase.paused

    def running(s: State) = s.phase == started

    /** Where a worker holds the attempt, so its answer settles the activity. */
    def held(s: State) = s.phase.in(started, cancelRequested)

    /** Where a pause takes effect: before an attempt starts or while one runs. */
    def pausable(s: State) = s.phase.in(scheduled, started)

    /**
     * The server code that answers a control of an activity that is over NotFound
     * (activity.go:106). Only the lifter reads a citation.
     */
    val notFoundCode = "chasm/lib/activity/activity.go"

  object effects:
    import Fact.*

    def startAttempt(s: State) = enter(s.copy(phase = started), statusStarted)

    def complete(s: State) = enter(s.copy(phase = completed), statusCompleted)

    def fail(s: State) = enter(s.copy(phase = failed), statusFailed)

    /**
     * Unlike the Nexus client, a retryable failure reads SCHEDULED again with a higher attempt count
     * (TransitionRescheduled); the System adds the backoff.
     */
    def retry(s: State) = enter(s.copy(phase = scheduled), statusScheduled)

    def cancel(s: State) = enter(s.copy(phase = canceled), statusCanceled)

    /** A control on an activity that is over is not found. */
    def notFound(s: State) = reject(Outcome.notFound, s)

    def pause(s: State) = enter(s.copy(phase = Phase.paused), statusPaused)

    def resume(s: State) = enter(s.copy(phase = scheduled), statusScheduled)

    def requestCancel(s: State) =
      enter(s.copy(phase = cancelRequested), statusCancelRequested)

    def terminate(s: State) = enter(s.copy(phase = terminated), statusTerminated)

    /** One of the activity's deadlines firing. Which deadline is the System's account of how. */
    def timeOut(s: State) = enter(s.copy(phase = timedOut), statusTimedOut)

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

  /**
   * What the product machine is, as the laws of model/temporal/capabilities read it, and so the laws
   * it receives without listing them: it closes, and a control of an activity that is over is not
   * found, by the code `states.notFoundCode` cites; it pauses; and a worker's poll hands out its work. It
   * receives terminalStatesAreFinal, closedIsRejectedUniformly and pausedIsNotDispatched (pause with
   * poll), each `activityProduct.<law>`, read on the System through the map under the bound held
   * there.
   */
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
