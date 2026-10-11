// The Nexus Product: what an operation does in either form (fn-126 decision 16). This level's
// own file holds NexusProduct, which refines nothing; the workflow and standalone Protocol
// machines refine it. The package clauses read the kind's package as well as this one,
// so its shared types and signature are in scope.
package temporal
package features.nexus
package shared
package product

import framework.*
import framework.Finite.given
import framework.outcomes.Outcome

// What an operation does. Each case declares the fact a step that enters it records, which
// `enters(Phase.x)` records.
enum Phase(val status: Fact) extends Recorded[Fact]:
  case scheduled extends Phase(Fact.nexusOperationScheduled), Waiting
  case started extends Phase(Fact.nexusOperationStarted), Held
  case succeeded extends Phase(Fact.nexusOperationCompleted), Succeeded
  case failed extends Phase(Fact.nexusOperationFailed), Failed
  case canceled extends Phase(Fact.nexusOperationCanceled), Canceled
  case timedOut extends Phase(Fact.nexusOperationTimedOut), TimedOut
  case terminated extends Phase(Fact.nexusOperationTerminated), Terminated

final case class State(phase: Phase)

enum Fact:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed,
    nexusOperationCanceled, nexusOperationTimedOut, nexusOperationTerminated

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried to
// the Protocol machine by the refinement declared there.

object NexusProduct
    extends Machine[State, Outcome, Fact],
      Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p)),
      Operation:
  val init = product.State(Phase.scheduled)
  override def end(s: State) = s.phase.in[Closed]

  // The product's phase sets.
  object states:
    // The five phases the product machine ends on.
    def productTerminal(s: State) = s.phase.in[Closed]

  object rules extends Facets(operation.rules)

  // A same-step claim names the action it is about under `when` and holds of the step that action
  // produces; a transition claim holds of the state before and the step after. A functional Query
  // realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
  // action the Case performs; a transition claim is searched and verified, never realized.

  object properties:
    // Once an operation is over, no step changes its phase. Declared on the product machine and read
    // on the Protocol machine through the map.
    val terminalIsFinal = property.once(states.productTerminal).keeps(_.phase)
