// The Operation facet of NexusProduct (fn-161.4): its actions' rules, the effects they use and its
// capabilities.
package temporal
package features.nexus
package shared
package product
import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import temporal.actors.worker.worker

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Operation extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object operation:
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
