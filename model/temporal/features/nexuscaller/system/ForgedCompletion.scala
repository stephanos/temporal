/* The Nexus caller's control: a caller design that predicts success for a failed completion, which
 * the forged-completion Query must refuse. A subject of the System level, beside
 * system/System.scala (fn-126 decisions 16 and 22). It is the protocol machine without its
 * refinement, its completion forged and the caller's inspection added, declared rather than derived
 * from NexusProtocol: a derivation lifts its source machines, and nexus-control.json holds this
 * machine alone.
 */
package temporal
package features.nexuscaller
package system

import umpire.*
import umpire.realize.{Alternative, Cleanup, Conformance, Disposition, Exploration, Reason}
import umpire.realize.{PropertyOutcome, RunExpectation, Variation}
import shared.worker.worker

// ### Signature

// A failed callback completes the forged control's operation two ways: as the success the control
// forges, and as the failure the runtime still sends.
val forged = choice
val sent = choice

// ### The control

object ForgedCompletion extends Machine[ProtocolState, Outcome, ProtocolFact], NegativeControl:
  val init = NexusProtocol.init
  def end(s: State) = NexusProtocol.states.terminalPhase(s.phase)
  val evidence: PartialFunction[ProtocolFact, String] = {
    case ProtocolFact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
    case ProtocolFact.pendingAttempts           => pendingAttempts.name
  }
  val unobservable = List(timers.backoff)

  object effects:
    def inspect(s: State) = stay(s)

    /**
     * The protocol's completion of an operation once scheduled: not found once it is over, and
     * resolved while it runs. One effect rather than two rules, because the forged completion names
     * both alternatives of a failed callback in every such phase, the not-found ones included.
     */
    def settle(s: State, resolution: Resolution) =
      if NexusProtocol.states.terminalPhase(s.phase) then NexusProtocol.effects.notFound(s)
      else NexusProtocol.effects.complete(s, resolution)

    // The control deliberately predicts success for a failed callback. The runtime still sends
    // failure.
    def forgedComplete(s: State, resolution: Resolution) =
      if resolution == Resolution.failed then
        choose(forged -> settle(s, Resolution.succeeded), sent -> settle(s, resolution))
      else settle(s, resolution)

  // The protocol machine's rules, its completion forged and the inspection added.
  object rules extends Rules(_.phase):
    on(caller.schedule)(in(Phase.unscheduled) ~> NexusProtocol.effects.schedule)
    on(handler.handlerReply)(in(Phase.scheduled) ~> NexusProtocol.effects.handlerReply)
    on(handler.complete)(in(NexusProtocol.states.created) ~> effects.forgedComplete)
    on(caller.inspect)(always ~> effects.inspect)
    on(network.transportFault)(in(Phase.scheduled) ~> NexusProtocol.effects.backOff)
    on(worker.workerStop)(always ~> NexusProtocol.effects.keep)
    on(timers.backoff)(in(Phase.backingOff) ~> NexusProtocol.effects.retry)
    on(deadline.scheduleToClose) {
      in(NexusProtocol.states.running).where(
        _.scheduleToClose == Timeout.expires
      ) ~> (NexusProtocol.effects.timeOut(_, TimeoutType.scheduleToClose))
    }
    on(deadline.scheduleToStart) {
      in(NexusProtocol.states.waiting).where(
        _.scheduleToStart == Timeout.expires
      ) ~> (NexusProtocol.effects.timeOut(_, TimeoutType.scheduleToStart))
    }
    on(deadline.startToClose) {
      where(s =>
        s.phase == Phase.started && s.startToClose == Timeout.expires
      ) ~> (NexusProtocol.effects.timeOut(_, TimeoutType.startToClose))
    }

  object properties:
    /**
     * A failed completion is recorded as completed: what the control predicts and no runtime sends.
     */
    val forgedSuccess = property when handler.complete(Resolution.failed) holds
      (_.records(ProtocolFact.nexusOperationCompleted))

  object queries:
    /**
     * The forged control, which every modeled execution that explains the evidence refutes, on a
     * path that inspects the operation around a failed completion.
     */
    val forgedCompletion =
      (query find properties.forgedSuccess in scenario("inspectedFailure").actions(
        caller.schedule(),
        caller.inspect,
        handler.handlerReply(Reply.async),
        caller.inspect,
        handler.complete(Resolution.failed)
      ) limits control)
        .expect(
          RunExpectation(
            Conformance.inconclusive,
            PropertyOutcome.violated,
            contract = PropertyOutcome.violated,
            disposition = Disposition.stoppedByMonitor,
            cleanup = Cleanup.succeeded,
            reason = Some(Reason.everyExplanationViolates),
            conformanceReason = Some(Reason.incomplete)
          )
        )
        .explore(
          Exploration(
            "nexusControl",
            Vector(
              Variation(
                1,
                Vector(
                  Alternative("twice", 20, Vector(caller.inspect, caller.inspect)),
                  Alternative("once", 10, Vector(caller.inspect)),
                  Alternative("none", 0, Vector.empty)
                )
              )
            ),
            runs = 1,
            edits = 8,
            dropPrefix = true
          )
        )
