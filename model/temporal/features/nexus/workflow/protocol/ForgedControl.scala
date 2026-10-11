// The ForgedControl facet of TrustingCaller (fn-161.4): its actions' rules, the effects they use and its
// capabilities.
package temporal
package features.nexus
package workflow
package protocol
import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import temporal.actors.worker.worker

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait ForgedControl extends Facet[protocol.State, Outcome, protocol.Fact], Phased[protocol.State, Phase]:
  object forgedControl:
    object effects:
      def inspect(s: State) = stay(s)
      // The Protocol's completion of an operation once scheduled: not found once it is over, and
      // resolved while it runs. One effect rather than two rules, because the forged completion names
      // both alternatives of a failed callback in every such phase, the not-found ones included.
      def settle(s: State, resolution: Resolution) =
        if s.phase.in[Closed] then reject(Outcome.rejected(Rejection.notFound), s)
        else NexusProtocol.operation.effects.complete(s, resolution)
      // The control deliberately predicts success for a failed callback. The runtime still sends
      // failure.
      def forgedComplete(s: State, resolution: Resolution) = // format: off
        if resolution == Resolution.failed then // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
          choose(forged -> settle(s, Resolution.succeeded), sent -> settle(s, resolution)) // format: on
        else settle(s, resolution)

    // The Protocol machine's rules, its completion forged and the inspection added.
    object rules extends Rules:
      on(caller.schedule) {
        when(Phase.unscheduled) ~> NexusProtocol.operation.effects.schedule
      }
      on(handler.reply) {
        when(Phase.scheduled) ~> NexusProtocol.operation.effects.reply
      }
      on(handler.complete) {
        when(NexusProtocol.states.created) ~> effects.forgedComplete
      }
      on(caller.inspect) {
        always ~> effects.inspect
      }
      on(network.fault) {
        when(Phase.scheduled) ~> NexusProtocol.operation.effects.backOff
      }
      on(worker.stop) {
        always ~> NexusProtocol.operation.effects.keep
      }
      on(timers.backoff) {
        when(Phase.backingOff) ~> enters(Phase.scheduled)
      }
      on(deadline.scheduleToClose) {
        when[Live].where(
          _.scheduleToClose == Timeout.expires // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        ) ~> (NexusProtocol.operation.effects.timeOut(_, TimeoutType.scheduleToClose))
      }
      on(deadline.scheduleToStart) {
        when[Waiting].where(
          _.scheduleToStart == Timeout.expires // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        ) ~> (NexusProtocol.operation.effects.timeOut(_, TimeoutType.scheduleToStart))
      }
      on(deadline.startToClose) {
        where(s =>
          s.phase == Phase.started && s.startToClose == Timeout.expires // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
        ) ~> (NexusProtocol.operation.effects.timeOut(_, TimeoutType.startToClose))
      }
