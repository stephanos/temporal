/* The Nexus caller's control: a caller design that predicts success for a failed completion, which
 * the forged-completion Query must refuse. A subject of the System level, beside
 * system/System.scala (fn-126 decisions 16 and 22). It keeps its own family, and its one Definition
 * ID, its inspection's, hangs off the control's former owner, which this file pins. It is the protocol
 * machine without its refinement, its completion forged and an inspection added, declared rather
 * than derived from NexusProtocol: a derivation lifts its source machines, and nexus-control.json
 * holds this machine alone.
 */
package temporal
package features.nexuscaller
package system

import umpire.*
import umpire.realize.{Alternative, Cleanup, Conformance, Disposition, Exploration, Reason}
import umpire.realize.{PropertyOutcome, RunExpectation, Variation}
import shared.worker.worker

// Moved from temporal.nexuscaller with the control; the pin keeps the inspection's Definition ID.
given forgedCompletionScope: DefinitionScope = DefinitionScope("temporal.nexuscaller.Control$")

// ### Signature

/**
 * The caller's inspection of its workflow, which only this control takes. The section keeps the
 * Definition ID this file's pin gives it, the one it had in the control's object.
 */
object inspection extends Section:
  val inspect = action(caller).on(operation)

// A failed callback completes the forged control's operation two ways: as the success the control
// forges, and as the failure the runtime still sends.
val forged = choice
val sent = choice

// ### The control

object ForgedCompletion
    extends Machine[ProtocolState, Outcome, ProtocolFact](using
      ControlFamily.family,
      summon,
      summon,
      summon
    ),
      NegativeControl:
  val entity = operation
  val init = NexusProtocol.init
  def end(s: State) = NexusProtocol.states.terminalPhase(s.phase)
  val evidence: PartialFunction[ProtocolFact, String] = {
    case ProtocolFact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
    case ProtocolFact.pendingAttempts           => pendingAttempts.name
  }
  val unobservable = List(timers.backoff)

  object effects extends Section:
    def inspect(s: State): List[ProtocolStep] = stay(s)

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
    in(Phase.unscheduled)(caller.schedule ~> NexusProtocol.effects.schedule)
    in(Phase.scheduled)(handler.handlerReply ~> NexusProtocol.effects.handlerReply)
    when(s => NexusProtocol.states.created(s.phase))(handler.complete ~> effects.forgedComplete)
    when(_ => true)(inspection.inspect ~> effects.inspect)
    in(Phase.scheduled)(network.transportFault ~> NexusProtocol.effects.backOff)
    when(_ => true)(worker.workerStop ~> NexusProtocol.effects.keep)
    in(Phase.backingOff)(timers.backoff ~> NexusProtocol.effects.retry)
    when(s => NexusProtocol.states.running(s.phase) && s.scheduleToClose == Timeout.expires) {
      deadline.scheduleToClose ~> (s =>
        NexusProtocol.effects.timeOut(s, TimeoutType.scheduleToClose)
      )
    }
    when(s => NexusProtocol.states.waiting(s.phase) && s.scheduleToStart == Timeout.expires) {
      deadline.scheduleToStart ~> (s =>
        NexusProtocol.effects.timeOut(s, TimeoutType.scheduleToStart)
      )
    }
    when(s => s.phase == Phase.started && s.startToClose == Timeout.expires) {
      deadline.startToClose ~> (s => NexusProtocol.effects.timeOut(s, TimeoutType.startToClose))
    }

  object properties extends Section:
    /**
     * A failed completion is recorded as completed: what the control predicts and no runtime sends.
     */
    val forgedSuccess = property when handler.complete(Resolution.failed) holds
      (_.records(ProtocolFact.nexusOperationCompleted))

  object queries extends Section:
    /**
     * The forged control, which every modeled execution that explains the evidence refutes, on a
     * path that inspects the operation around a failed completion.
     */
    val forgedCompletion =
      (query find properties.forgedSuccess in scenario("inspectedFailure").actions(
        caller.schedule(),
        inspection.inspect,
        handler.handlerReply(Reply.async),
        inspection.inspect,
        handler.complete(Resolution.failed)
      ) limits control total 960)
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
                  Alternative("twice", 20, Vector(inspection.inspect, inspection.inspect)),
                  Alternative("once", 10, Vector(inspection.inspect)),
                  Alternative("none", 0, Vector.empty)
                )
              )
            ),
            runs = 1,
            edits = 8,
            dropPrefix = true
          )
        )
