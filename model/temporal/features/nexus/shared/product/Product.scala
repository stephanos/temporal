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
import framework.outcomes.{Outcome, Rejection}
import temporal.actors.worker.worker

// What an operation does. Each case declares the fact a step that enters it records, which
// `enters(Phase.x)` records.
enum Phase(val status: Fact) extends Recorded[Fact] derives Finite:
  case scheduled extends Phase(Fact.nexusOperationScheduled), Waiting
  case started extends Phase(Fact.nexusOperationStarted), Held
  case succeeded extends Phase(Fact.nexusOperationCompleted), Succeeded
  case failed extends Phase(Fact.nexusOperationFailed), Failed
  case canceled extends Phase(Fact.nexusOperationCanceled), Canceled
  case timedOut extends Phase(Fact.nexusOperationTimedOut), TimedOut
  case terminated extends Phase(Fact.nexusOperationTerminated), Terminated

final case class State(phase: Phase) derives Finite

enum Fact derives Finite:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed,
    nexusOperationCanceled, nexusOperationTimedOut, nexusOperationTerminated

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried to
// the Protocol machine by the refinement declared there.

object NexusProduct
    extends Machine[State, Outcome, Fact],
      Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p)):
  val init = product.State(Phase.scheduled)
  override def end(s: State) = s.phase.in[Closed]

  // The product's phase sets.
  object states:
    // The five phases the product machine ends on.
    def productTerminal(s: State) = s.phase.in[Closed]

  // Both forms record these facts through their history or Describe evidence. A step whose only
  // work is entering a phase is `enters(Phase.x)` at its rule, which records the phase's fact.
  object effects:
    import Fact.*

    def complete(s: State, resolution: Resolution) = // format: off
      val startedFirst = if s.phase != Phase.started then List(nexusOperationStarted) else Nil // scalafix:ok DisableSyntax.!=; strictEquality checks native comparisons
      resolution match // format: on
        case Resolution.succeeded =>
          enter(s.copy(phase = Phase.succeeded), (startedFirst ++ List(nexusOperationCompleted))*)
        case Resolution.failed =>
          enter(s.copy(phase = Phase.failed), (startedFirst ++ List(nexusOperationFailed))*)
        case Resolution.canceled =>
          enter(s.copy(phase = Phase.canceled), (startedFirst ++ List(nexusOperationCanceled))*)

  object rules extends Rules:
    // The handler's reply to the server's start request moves only an operation that has not
    // started yet. A retryable handler error leaves the operation where it is, so no rule fires it:
    // the product machine does not know about backing off, which is the whole of what the Protocol
    // machine adds.
    on(handler.reply(Reply.syncSuccess)) {
      when(Phase.scheduled) ~> enters(Phase.succeeded)
    }
    on(handler.reply(Reply.async)) {
      when(Phase.scheduled) ~> enters(Phase.started)
    }
    on(handler.reply(Reply.operationFailed)) {
      when(Phase.scheduled) ~> enters(Phase.failed)
    }
    on(handler.reply(Reply.operationCanceled)) {
      when(Phase.scheduled) ~> enters(Phase.canceled)
    }
    on(handler.reply(Reply.handlerError(false))) {
      when(Phase.scheduled) ~> enters(Phase.failed)
    }

    // An asynchronous completion settles a running operation, and is not found once it is over.
    on(handler.complete) {
      when[Closed] ~> rejects(Rejection.notFound)
    }
    on(handler.complete(Resolution.succeeded)) {
      when(Phase.scheduled, Phase.started) ~> ((s: State) =>
        effects.complete(s, Resolution.succeeded)
      )
    }
    on(handler.complete(Resolution.failed)) {
      when(Phase.scheduled, Phase.started) ~> ((s: State) => effects.complete(s, Resolution.failed))
    }
    on(handler.complete(Resolution.canceled)) {
      when(Phase.scheduled, Phase.started) ~> ((s: State) =>
        effects.complete(s, Resolution.canceled)
      )
    }

    // A transport fault is an ordinary action of the network, and the handler's worker stopping is a
    // fault the Run records. The product machine sees neither: whether a delivery was retried is the
    // Protocol's account of how, not what, and a step that kept the state and recorded nothing would
    // be indistinguishable from a stutter, which the refinement would read as this step.
    disabled(network.fault, worker.stop)
    on(client.terminate) {
      when(Phase.scheduled, Phase.started) ~> enters(Phase.terminated)
    }

    // One of the operation's deadlines fires while it runs. Which deadline is the Protocol's account
    // of how, so the product machine has one timer.
    on(timers.timeout) {
      when(Phase.scheduled, Phase.started) ~> enters(Phase.timedOut)
    }

  // A same-step claim names the action it is about under `when` and holds of the step that action
  // produces; a transition claim holds of the state before and the step after. A functional Query
  // realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
  // action the Case performs; a transition claim is searched and verified, never realized.

  object properties:
    // Once an operation is over, no step changes its phase. Declared on the product machine and read
    // on the Protocol machine through the map.
    val terminalIsFinal = property.once(states.productTerminal).keeps(_.phase)
