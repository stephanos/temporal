/* The standalone activity's Product: what DescribeActivityExecution reports, the level a caller
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
import ActivityFamily.given

// ### The product machine: what DescribeActivityExecution shows, with no account of how. A retry
// reads as scheduled again, a pause of a running attempt as started until the worker yields.

/** Every status the product machine records is confirmed by the status observation of its name. */
object ActivityProduct extends Machine[ProductState, Outcome, ProductFact]:
  import ProductPhase.*

  val entity = activity
  val init = ProductState(scheduled)
  def end(s: State) = states.over(s)

  /** The product's status sets and the constant its capabilities cite. */
  object states extends Section:
    def phase(s: State) = s.phase

    def terminal(p: ProductPhase) = p.in(completed, failed, canceled, terminated, timedOut)

    /** A path ends where the activity is over: `end` reads this named predicate. */
    def over(s: State) = terminal(s.phase)

    /** A paused activity, which no worker is given. */
    def paused(s: State) = s.phase == ProductPhase.paused

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

  object effects extends Section:
    import ProductFact.*

    def startAttempt(s: State) = enter(s.copy(phase = started), statusStarted)

    def complete(s: State) = enter(s.copy(phase = completed), statusCompleted)

    def fail(s: State) = enter(s.copy(phase = failed), statusFailed)

    /**
     * Unlike the Nexus caller, a retryable failure reads SCHEDULED again with a higher attempt count
     * (TransitionRescheduled); the protocol adds the backoff.
     */
    def retry(s: State) = enter(s.copy(phase = scheduled), statusScheduled)

    def cancel(s: State) = enter(s.copy(phase = canceled), statusCanceled)

    /** A control on an activity that is over is not found. */
    def notFound(s: State): List[ProductStep] = List(Step(Outcome.notFound, s))

    def pause(s: State) = enter(s.copy(phase = ProductPhase.paused), statusPaused)

    def resume(s: State) = enter(s.copy(phase = scheduled), statusScheduled)

    def requestCancel(s: State) =
      enter(s.copy(phase = cancelRequested), statusCancelRequested)

    def terminate(s: State) = enter(s.copy(phase = terminated), statusTerminated)

    /** One of the activity's deadlines firing. Which deadline is the protocol's account of how. */
    def timeOut(s: State) = enter(s.copy(phase = timedOut), statusTimedOut)

  object rules extends Rules(_.phase):
    in(scheduled)(worker.attemptStart ~> effects.startAttempt)

    // A worker's answer settles an attempt it holds. A retryable failure is retried, or canceled
    // under a cancel request; a canceled answer settles only an activity whose cancellation was
    // requested.
    when(s => states.held(s)) {
      worker.attemptResult(AttemptResult.completed) ~> effects.complete
      worker.attemptResult(AttemptResult.failed(false)) ~> effects.fail
    }
    in(started)(worker.attemptResult(AttemptResult.failed(true)) ~> effects.retry)
    in(cancelRequested) {
      worker.attemptResult(AttemptResult.failed(true)) ~> effects.cancel
      worker.attemptResult(AttemptResult.canceled) ~> effects.cancel
    }

    // A control on an activity that is over is not found. A pause of a paused or cancel-requested
    // activity, or an unpause of one not paused, is FailedPrecondition; the protocol lists them.
    when(s => states.terminal(s.phase))(caller.control ~> effects.notFound)
    when(s => states.pausable(s))(caller.control(Control.pause) ~> effects.pause)
    when(s => states.paused(s))(caller.control(Control.unpause) ~> effects.resume)
    in(scheduled, started, paused, cancelRequested) {
      caller.control(Control.requestCancel) ~> effects.requestCancel
      caller.control(Control.terminate) ~> effects.terminate
    }

    // The worker stopping is a fault the Run records and the activity does not feel.
    disabled(process.workerStop)

    in(scheduled, started, paused, cancelRequested) {
      timers.timeout ~> effects.timeOut
    }

  /**
   * What the product machine is, as the laws of model/temporal/capabilities read it, and so the laws
   * it receives without listing them: it closes, and a control of an activity that is over is not
   * found, by the code `states.notFoundCode` cites; it pauses; and a worker's poll hands out its work. It
   * receives terminalStatesAreFinal, closedIsRejectedUniformly and pausedIsNotDispatched (pause with
   * poll), each `activityProduct.<law>`, read on the protocol through the map under the bound held
   * there.
   */
  object implements extends Section:
    val all = capabilities(limits = three)(
      Closable(
        status = states.phase,
        terminal = states.terminal,
        rejected = cited(Outcome.notFound, states.notFoundCode)
      ),
      Pausable(
        pause = caller.control(Control.pause),
        unpause = caller.control(Control.unpause),
        paused = states.paused
      ),
      Pollable(dispatch = worker.attemptStart, running = states.running)
    )
