// The standalone activity's Product: what DescribeActivityExecution reports, the level a client
// reads (fn-126 decision 16). The level's own file holds the product machine, ActivityProduct, which
// refines nothing; system/System.scala refines it. The two package clauses read the feature's
// package as well as this one, so its types and signature are in scope.
package temporal
package features.activity
package standalone
package product

import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
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

// The accessor the blocks of ActivityProduct read a state's phase by. Its rules enter a phase by
// `enters(Phase.x)`, which writes it through the machine's `Phased` writer and records the status
// the phase declares.
def phase(using v: View[State]): Phase = v.get(_.phase)

// ### The product machine: what DescribeActivityExecution shows, with no account of how. A retry
// reads as scheduled again, a pause of a running attempt as started until the worker yields.

// Every status the product machine records is confirmed by the status observation of its name.
object ActivityProduct
    extends Machine[State, Outcome, Fact],
      Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p)),
      Dispatching,
      Pausing,
      Cancellation,
      Lifecycle,
      Reset,
      Heartbeat,
      Completion,
      Retry,
      Timeouts:
  val init = product.State(Phase.scheduled)
  override def end(s: State) = s.phase.in[Closed]

  object states:
    def status(s: State) = s.phase
    val paused = is(phase == Phase.paused)

  // Each action's rules sit in the facet that owns it (fn-161), in this order. The worker process
  // stopping is not the product's: a client reads nothing of it, so the product binds no stop, and
  // the System keeps the stop as a step that changes nothing.
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

  object properties:
    // The public cancelRequested phase can mean either that a worker still holds an attempt or that
    // a control followed a paused activity. The product state cannot tell those histories apart, so
    // its pause property keeps the former, narrower meaning of "running": started. The protocol
    // machine has enough state to use Pausable's role-derived Property directly.
    def pausedDoesNotStartAttempt(m: Declares[State]): Property[State] =
      m.property
        .never(s => s.state.phase == Phase.started)
        .from(_.phase == Phase.paused)

  // The capabilities each facet declares for the actions it owns.
  object capabilities
      extends FacetCapabilities(
        dispatching.capabilities,
        pausing.capabilities,
        lifecycle.capabilities
      )

  object queries:
    capabilities.bound(three)
