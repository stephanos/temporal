// The Nexus caller's control: a caller design that predicts success for a failed completion, which
// the forged-completion Query must refuse. A subject of the System level, beside
// system/System.scala (fn-126 decisions 16 and 22). It is the System machine without its
// refinement, its completion forged and the caller's inspection added, declared rather than derived
// from NexusSystem: a derivation lifts its source machines, and nexus-workflow-control.json holds this
// machine alone.
package temporal
package features.nexus
package workflow
package system

import umpire.*
import umpire.realize.{Alternative, Cleanup, Conformance, Disposition, Exploration, Reason}
import umpire.realize.{PropertyOutcome, RunExpectation, Variation}
import temporal.shared.worker.worker

// ### Signature

// A failed callback completes the forged control's operation two ways: as the success the control
// forges, and as the failure the runtime still sends.
val forged = choice
val sent = choice

// ### The control

object TrustingCaller extends Machine[system.State, Outcome, system.Fact], NegativeControl:
  val init = NexusSystem.init
  def end(s: State) = NexusSystem.states.terminalPhase(s.phase)
  val evidence: PartialFunction[Fact, String] = {
    case Fact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
    case Fact.pendingAttempts           => pendingAttempts.name
  }
  val unobservable = List(timers.backoff)

  object effects:
    def inspect(s: State) = stay(s)

    // The System's completion of an operation once scheduled: not found once it is over, and
    // resolved while it runs. One effect rather than two rules, because the forged completion names
    // both alternatives of a failed callback in every such phase, the not-found ones included.
    def settle(s: State, resolution: Resolution) =
      if NexusSystem.states.terminalPhase(s.phase) then NexusSystem.effects.notFound(s)
      else NexusSystem.effects.complete(s, resolution)

    // The control deliberately predicts success for a failed callback. The runtime still sends
    // failure.
    def forgedComplete(s: State, resolution: Resolution) =
      if resolution == Resolution.failed then
        choose(forged -> settle(s, Resolution.succeeded), sent -> settle(s, resolution))
      else settle(s, resolution)

  // The System machine's rules, its completion forged and the inspection added.
  object rules extends Rules(_.phase):
    on(caller.schedule)(in(Phase.unscheduled) ~> NexusSystem.effects.schedule)
    on(handler.reply)(in(Phase.scheduled) ~> NexusSystem.effects.reply)
    on(handler.complete)(in(NexusSystem.states.created) ~> effects.forgedComplete)
    on(caller.inspect)(always ~> effects.inspect)
    on(network.fault)(in(Phase.scheduled) ~> NexusSystem.effects.backOff)
    on(worker.stop)(always ~> NexusSystem.effects.keep)
    on(timers.backoff)(in(Phase.backingOff) ~> NexusSystem.effects.retry)
    on(deadline.scheduleToClose) {
      in(NexusSystem.states.running).where(
        _.scheduleToClose == Timeout.expires
      ) ~> (NexusSystem.effects.timeOut(_, TimeoutType.scheduleToClose))
    }
    on(deadline.scheduleToStart) {
      in(NexusSystem.states.waiting).where(
        _.scheduleToStart == Timeout.expires
      ) ~> (NexusSystem.effects.timeOut(_, TimeoutType.scheduleToStart))
    }
    on(deadline.startToClose) {
      where(s =>
        s.phase == Phase.started && s.startToClose == Timeout.expires
      ) ~> (NexusSystem.effects.timeOut(_, TimeoutType.startToClose))
    }

  object properties:
    // A failed completion is recorded as completed: what the control predicts and no runtime sends.
    val forgedSuccess = property when handler.complete(Resolution.failed) holds
      (_.records(Fact.nexusOperationCompleted))

  object queries:
    val inspectedFailure = scenario.actions(
      caller.schedule(),
      caller.inspect,
      handler.reply(Reply.async),
      caller.inspect,
      handler.complete(Resolution.failed)
    )

    // The forged control, which every modeled execution that explains the evidence refutes, on a
    // path that inspects the operation around a failed completion.
    val forgedCompletion =
      (query find properties.forgedSuccess in inspectedFailure limits control)
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
