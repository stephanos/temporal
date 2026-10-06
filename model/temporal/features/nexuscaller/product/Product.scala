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

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried to
// the System machine by the refinement declared there.

object NexusProduct extends Machine[ProductState, Outcome, ProductFact]:
  import ProductPhase.*

  val init = ProductState(scheduled)
  def end(s: State) = states.productTerminal(s)

  /** The product's phase sets. */
  object states:
    /** The four phases the product machine ends on. */
    def productTerminal(s: State) = s.phase.in(succeeded, failed, canceled, timedOut)

  /** Every fact the product machine records is confirmed by the history event of its name. */
  object effects:
    import ProductFact.*

    def succeed(s: State) = enter(s.copy(phase = succeeded), nexusOperationCompleted)

    def start(s: State) = enter(s.copy(phase = started), nexusOperationStarted)

    def fail(s: State) = enter(s.copy(phase = failed), nexusOperationFailed)

    def cancel(s: State) = enter(s.copy(phase = canceled), nexusOperationCanceled)

    /** A completion that arrives after the operation is over is not found, and changes nothing. */
    def notFound(s: State) = reject(Outcome.notFound, s)

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
    on(handler.reply(Reply.syncSuccess))(in(scheduled) ~> effects.succeed)
    on(handler.reply(Reply.async))(in(scheduled) ~> effects.start)
    on(handler.reply(Reply.operationFailed))(in(scheduled) ~> effects.fail)
    on(handler.reply(Reply.operationCanceled))(in(scheduled) ~> effects.cancel)
    on(handler.reply(Reply.handlerError(false)))(in(scheduled) ~> effects.fail)

    // An asynchronous completion settles a running operation, and is not found once it is over.
    on(handler.complete)(where(states.productTerminal) ~> effects.notFound)
    on(handler.complete(Resolution.succeeded))(in(scheduled, started) ~> effects.succeed)
    on(handler.complete(Resolution.failed))(in(scheduled, started) ~> effects.fail)
    on(handler.complete(Resolution.canceled))(in(scheduled, started) ~> effects.cancel)

    // A transport fault is an ordinary action of the network, and the handler's worker stopping is a
    // fault the Run records. The product machine sees neither: whether a delivery was retried is the
    // protocol's account of how, not what, and a step that kept the state and recorded nothing would
    // be indistinguishable from a stutter, which the refinement would read as this step.
    disabled(network.fault, worker.stop)

    // The deadline fires while the operation runs.
    on(timers.timeout)(in(scheduled, started) ~> effects.timeOut)

  // A same-step claim names the action it is about under `when` and holds of the step that action
  // produces; a transition claim holds of the state before and the step after. A functional Query
  // realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
  // action the Case performs; a transition claim is searched and verified, never realized.

  object properties:
    /**
     * Once an operation is over, no step changes its phase. Declared on the product machine and read
     * on the System machine through the map.
     */
    val terminalIsFinal = property.once(states.productTerminal).keeps(_.phase)
