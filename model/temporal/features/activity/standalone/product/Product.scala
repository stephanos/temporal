// The standalone activity's Product: what DescribeActivityExecution reports, the level a client
// reads (fn-126 decision 16). The level's own file holds the product machine, ActivityProduct, which
// refines nothing; system/System.scala refines it. The two package clauses read the feature's
// package as well as this one, so its types and signature are in scope.
package temporal
package features.activity
package standalone
package product

import umpire.*
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

// The accessors the blocks of ActivityProduct read and assign a state's phase by: the setter hands
// the draft the phase it assigns, whose status the step records.
def phase(using v: View[State]): Phase = v.get(_.phase)
def phase_=(p: Phase)(using d: Draft[State, ?, Fact]): Unit = d.set(p)(_.copy(phase = p))

// ### The product machine: what DescribeActivityExecution shows, with no account of how. A retry
// reads as scheduled again, a pause of a running attempt as started until the worker yields.

// Every status the product machine records is confirmed by the status observation of its name.
object ActivityProduct extends Machine[State, Outcome, Fact]:
  import Phase.*

  val init = product.State(scheduled)
  def end(s: State) = states.over(s)

  object states:
    def status(s: State) = s.phase
    def terminal(p: Phase) = p.in[Closed]
    val over = is(terminal(phase))
    val paused = is(phase == Phase.paused)
    val running = is(phase == started)
    val held = is(phase.in[Held])
    val pausable = is(phase.in(scheduled, started))

  object effects:
    val startAttempt = effect { phase = started }
    val complete = effect { phase = completed }
    val fail = effect { phase = failed }
    val retry = effect { phase = scheduled }
    val cancel = effect { phase = canceled }
    val pause = effect { phase = Phase.paused }
    val resume = effect { phase = scheduled }
    val requestCancel = effect { phase = cancelRequested }
    val terminate = effect { phase = terminated }
    val timeOut = effect { phase = timedOut }
    val notFound = effect(reject(Outcome.notFound))

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

  object capabilities extends Capabilities:
    val closable: Capability = Closable(
      status = states.status,
      terminal = states.terminal,
      rejected = Outcome.notFound
    )
    val pausable: Capability = Pausable(
      pause = client.control(Control.pause),
      unpause = client.control(Control.unpause),
      paused = states.paused
    )
    val pollable: Capability = Pollable(dispatch = worker.poll, running = states.running)

  object queries:
    capabilities.bound(three)
