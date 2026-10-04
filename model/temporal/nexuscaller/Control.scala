package temporal.nexuscaller

import umpire.*
import umpire.realize.{Alternative, Conformance, Exploration, RunExpectation, Variation}
import umpire.realize.Outcome as ExpectedOutcome

object Control:
  val inspect = action("inspect", caller).on(operation)

  def inspectStep(s: ProtocolState): List[ProtocolStep] =
    List(Step(Outcome.accepted, s, Nil))

  // The control deliberately predicts success for a failed callback. The runtime still sends failure.
  def forgedComplete(s: ProtocolState, resolution: Resolution): List[ProtocolStep] =
    if resolution == Resolution.failed then
      Protocol.completeStep(s, Resolution.succeeded) ++ Protocol.completeStep(s, resolution)
    else Protocol.completeStep(s, resolution)

  val forged: Machine[ProtocolState, Outcome, ProtocolFact] =
    machine[ProtocolState, Outcome, ProtocolFact](
      umpire.Family("temporal.nexus.control"),
      "forgedCompletion"
    ) {
      forEntity(operation)
      starts(unscheduled)
      ends(s => Protocol.terminalPhase(s.phase))
      unobservable(backoff)
      evidence {
        case ProtocolFact.nexusOperationScheduled   => "nexusOperationScheduled"
        case ProtocolFact.nexusOperationStarted     => "nexusOperationStarted"
        case ProtocolFact.nexusOperationCompleted   => "nexusOperationCompleted"
        case ProtocolFact.nexusOperationFailed      => "nexusOperationFailed"
        case ProtocolFact.nexusOperationCanceled    => "nexusOperationCanceled"
        case ProtocolFact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
        case ProtocolFact.pendingAttempts           => pendingAttempts.name
      }
      steps(
        schedule ~> Protocol.scheduleStep,
        handlerReply ~> Protocol.handlerReplyStep,
        complete ~> forgedComplete,
        inspect ~> inspectStep,
        transportFault ~> Protocol.transportFaultStep,
        workerStop ~> Protocol.workerStopStep,
        backoff ~> Protocol.backoffStep,
        scheduleToClose ~> Protocol.scheduleToCloseStep,
        scheduleToStart ~> Protocol.scheduleToStartStep,
        startToClose ~> Protocol.startToCloseStep
      )
    }

  val forgedSuccess: Property[ProtocolState] = forged
    .property("forgedSuccess")
    .when(complete(Resolution.failed))
    .holds(_.facts.contains(ProtocolFact.nexusOperationCompleted))
  val inspected: Scenario[ProtocolState] = forged
    .scenario("inspectedFailure")
    .starts(unscheduled)
    .actions(
      schedule(Timeout.unset, Timeout.unset, Timeout.unset),
      inspect,
      handlerReply(Reply.async),
      inspect,
      complete(Resolution.failed)
    )
  val forgedCompletion: Query =
    (query("forgedCompletion") find forgedSuccess in inspected limits Limits(
      "control",
      8,
      8,
      262144
    ) total 960)
      .expect(
        RunExpectation(
          Conformance.inconclusive,
          ExpectedOutcome.violated,
          "every modeled execution that explains the evidence violates it",
          contract = ExpectedOutcome.violated
        )
      )
      .explore(
        Exploration(
          "nexusControl",
          Vector(
            Variation(
              1,
              Vector(
                Alternative("twice", 20, Vector(inspect, inspect)),
                Alternative("once", 10, Vector(inspect)),
                Alternative("none", 0, Vector.empty)
              )
            )
          ),
          runs = 1,
          edits = 8,
          dropPrefix = true
        )
      )
