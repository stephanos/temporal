// The Nexus Product: what an operation does in either form (fn-126 decision 16). This level's
// own file holds NexusProduct, which refines nothing; the workflow and standalone System
// machines refine it. The package clauses read the kind's package as well as this one,
// so its shared types and signature are in scope.
package temporal
package features.nexus
package product

import umpire.*
import temporal.shared.worker.worker

// What an operation does.
enum Phase derives Finite:
  case scheduled extends Phase, Waiting
  case started extends Phase, Held
  case succeeded extends Phase, Succeeded
  case failed extends Phase, Failed
  case canceled extends Phase, Canceled
  case timedOut extends Phase, TimedOut
  case terminated extends Phase, Terminated

final case class State(phase: Phase) derives Finite

enum Fact derives Finite:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed,
    nexusOperationCanceled, nexusOperationTimedOut, nexusOperationTerminated

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried to
// the System machine by the refinement declared there.

object NexusProduct extends Machine[State, Outcome, Fact], Phased[State, Phase](_.phase):
  import Phase.*

  val init = product.State(scheduled)
  def end(s: State) = s.phase.in[Closed]

  // The product's phase sets.
  object states:
    // The five phases the product machine ends on.
    def productTerminal(s: State) = s.phase.in[Closed]

  // Both forms record these facts through their history or Describe evidence.
  object effects:
    import Fact.*

    def succeed(s: State) = enter(s.copy(phase = succeeded), nexusOperationCompleted)

    def complete(s: State, resolution: Resolution) =
      val startedFirst = if s.phase != started then List(nexusOperationStarted) else Nil
      resolution match
        case Resolution.succeeded =>
          enter(s.copy(phase = succeeded), (startedFirst ++ List(nexusOperationCompleted))*)
        case Resolution.failed =>
          enter(s.copy(phase = failed), (startedFirst ++ List(nexusOperationFailed))*)
        case Resolution.canceled =>
          enter(s.copy(phase = canceled), (startedFirst ++ List(nexusOperationCanceled))*)

    def terminate(s: State) = enter(s.copy(phase = terminated), nexusOperationTerminated)

    def start(s: State) = enter(s.copy(phase = started), nexusOperationStarted)

    def fail(s: State) = enter(s.copy(phase = failed), nexusOperationFailed)

    def cancel(s: State) = enter(s.copy(phase = canceled), nexusOperationCanceled)

    // A completion that arrives after the operation is over is not found, and changes nothing.
    def notFound(s: State) = reject(Outcome.notFound, s)

    // One of the operation's deadlines firing. Which deadline is the System's account of how, so
    // the product machine has one timer.
    def timeOut(s: State) = enter(s.copy(phase = timedOut), nexusOperationTimedOut)

  object rules extends Rules:
    // The handler's reply to the server's start request moves only an operation that has not
    // started yet. A retryable handler error leaves the operation where it is, so no rule fires it:
    // the product machine does not know about backing off, which is the whole of what the System
    // machine adds.
    on(handler.reply(Reply.syncSuccess))(in(scheduled) ~> effects.succeed)
    on(handler.reply(Reply.async))(in(scheduled) ~> effects.start)
    on(handler.reply(Reply.operationFailed))(in(scheduled) ~> effects.fail)
    on(handler.reply(Reply.operationCanceled))(in(scheduled) ~> effects.cancel)
    on(handler.reply(Reply.handlerError(false)))(in(scheduled) ~> effects.fail)

    // An asynchronous completion settles a running operation, and is not found once it is over.
    on(handler.complete)(when[Closed] ~> effects.notFound)
    on(handler.complete(Resolution.succeeded))(
      in(scheduled, started) ~> ((s: State) => effects.complete(s, Resolution.succeeded))
    )
    on(handler.complete(Resolution.failed))(
      in(scheduled, started) ~> ((s: State) => effects.complete(s, Resolution.failed))
    )
    on(handler.complete(Resolution.canceled))(
      in(scheduled, started) ~> ((s: State) => effects.complete(s, Resolution.canceled))
    )

    // A transport fault is an ordinary action of the network, and the handler's worker stopping is a
    // fault the Run records. The product machine sees neither: whether a delivery was retried is the
    // System's account of how, not what, and a step that kept the state and recorded nothing would
    // be indistinguishable from a stutter, which the refinement would read as this step.
    disabled(network.fault, worker.stop)
    on(client.terminate)(in(scheduled, started) ~> effects.terminate)

    // The deadline fires while the operation runs.
    on(timers.timeout)(in(scheduled, started) ~> effects.timeOut)

  // A same-step claim names the action it is about under `when` and holds of the step that action
  // produces; a transition claim holds of the state before and the step after. A functional Query
  // realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
  // action the Case performs; a transition claim is searched and verified, never realized.

  object properties:
    // Once an operation is over, no step changes its phase. Declared on the product machine and read
    // on the System machine through the map.
    val terminalIsFinal = property.once(states.productTerminal).keeps(_.phase)
