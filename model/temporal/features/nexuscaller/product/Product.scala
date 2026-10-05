/* The Nexus caller's Product: what an operation does, the level a caller reads (fn-126 decision
 * 16). The level's own file holds the product machine, NexusProduct, which refines nothing;
 * system/System.scala refines it. The two package clauses read the feature's package as well as
 * this one, so its types and signature are in scope.
 */
package temporal
package features.nexuscaller
package product

import umpire.*
import shared.worker.worker
import CallerFamily.given

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried to
// the protocol machine by the refinement declared there.

object NexusProduct extends Machine[ProductState, Outcome, ProductFact]:
  import ProductPhase.*

  val entity = operation
  val init = ProductState(scheduled)
  def end(s: State) = states.productTerminal(s)

  /** The product's phase sets. */
  object states extends Section:
    /** The four phases the product machine ends on. */
    def productTerminal(s: State) = s.phase.in(succeeded, failed, canceled, timedOut)

  /** Every fact the product machine records is confirmed by the history event of its name. */
  object effects extends Section:
    import ProductFact.*

    def succeed(s: State) = enter(s.copy(phase = succeeded), nexusOperationCompleted)

    def start(s: State) = enter(s.copy(phase = started), nexusOperationStarted)

    def fail(s: State) = enter(s.copy(phase = failed), nexusOperationFailed)

    def cancel(s: State) = enter(s.copy(phase = canceled), nexusOperationCanceled)

    /** A completion that arrives after the operation is over is not found, and changes nothing. */
    def notFound(s: State): List[ProductStep] = List(Step(Outcome.notFound, s))

    /**
     * One of the operation's deadlines firing. Which deadline is the protocol's account of how, so
     * the product machine has one timer.
     */
    def timeOut(s: State) = enter(s.copy(phase = timedOut), nexusOperationTimedOut)

  object rules extends Rules(_.phase):
    // The handler's reply to the server's start request moves only an operation that has not
    // started yet. A retryable handler error leaves the operation where it is, so no rule fires it:
    // the product machine does not know about backing off, which is the whole of what the protocol
    // machine adds.
    in(scheduled) {
      handler.handlerReply(Reply.syncSuccess) ~> effects.succeed
      handler.handlerReply(Reply.async) ~> effects.start
      handler.handlerReply(Reply.operationFailed) ~> effects.fail
      handler.handlerReply(Reply.operationCanceled) ~> effects.cancel
      handler.handlerReply(Reply.handlerError(false)) ~> effects.fail
    }

    // An asynchronous completion settles a running operation, and is not found once it is over.
    when(s => states.productTerminal(s))(handler.complete ~> effects.notFound)
    in(scheduled, started) {
      handler.complete(Resolution.succeeded) ~> effects.succeed
      handler.complete(Resolution.failed) ~> effects.fail
      handler.complete(Resolution.canceled) ~> effects.cancel
    }

    // A transport fault is an ordinary action of the network, and the handler's worker stopping is a
    // fault the Run records. The product machine sees neither: whether a delivery was retried is the
    // protocol's account of how, not what, and a step that kept the state and recorded nothing would
    // be indistinguishable from a stutter, which the refinement would read as this step.
    disabled(network.transportFault, worker.workerStop)

    // The deadline fires while the operation runs.
    in(scheduled, started)(timers.timeout ~> effects.timeOut)

  // A same-step claim names the action it is about under `when` and holds of the step that action
  // produces; a transition claim holds of the state before and the step after. A functional Query
  // realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
  // action the Case performs; a transition claim is searched and verified, never realized.

  object properties extends Section:
    /**
     * Once an operation is over, no step changes its phase. Declared on the product machine and read
     * on the protocol machine through the map.
     */
    val terminalIsFinal = property.once(states.productTerminal).keeps(_.phase)
