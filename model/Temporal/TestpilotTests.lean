import Temporal.Testpilot
import Umpire.Variations.Tests.Lowering

namespace Temporal.TestpilotTests

open temporal.server.api.testpilot.v1

private def activationKind : Entrypoint → Option Nat
  | { activation := some (.controller _), .. } => some 0
  | { activation := some (.workflow _), .. } => some 1
  | { activation := some (.nexus_handler _), .. } => some 2
  | _ => none

#guard match Temporal.Testpilot.getSystemInfoCase with
  | .ok output =>
      match output.program, output.contract with
      | some program, some contract =>
        match program.entrypoints.toList, contract.rules.toList with
        | [entrypoint], [rule] =>
          match entrypoint.instructions.toList with
          | [node] =>
            match node.instruction.bind (·.instruction) with
            | some (.invoke_rpc request) =>
                request.endpoint_role_id == Temporal.Testpilot.workflowServiceRole &&
                request.method == Temporal.Testpilot.getSystemInfoMethod &&
                request.request_assignments.isEmpty &&
                request.response_reads.map (·.cardinality) == #[.READ_CARDINALITY_ONE] &&
                rule.rule_id == "server-version-present"
            | _ => false
          | _ => false
        | _, _ => false
      | _, _ => false
  | .error _ => false

-- The worker-outage Case's stop instruction is the lowered fault intent itself, not a hand-written
-- copy of one: the Program the fixture renders carries exactly what `lower` produced.
#guard match Temporal.Testpilot.stopIntent.lower Temporal.Testpilot.stopRealization,
    Temporal.Testpilot.workerOutageCase with
  | .ok stop, .ok artifact =>
      (artifact.program.bind fun program =>
        (program.entrypoints.find? fun entrypoint => entrypoint.entrypoint_id == "controller").bind
          fun controller => controller.instructions[0]?).any
        (Umpire.VariationsLoweringTests.sameFaultNode · stop)
  | _, _ => false

-- The outage rule is the checked-in `rule_events` deadline: no elapsed-time bound participates.
#guard match Temporal.Testpilot.workerOutageCase with
  | .ok artifact =>
      (artifact.contract.bind fun contract =>
        (contract.rules.find? fun rule => rule.rule_id == "worker-outage-order").bind
          (·.deadline)).any fun deadline =>
            (match deadline.bound with
              | some (.rule_events events) => events == Temporal.Testpilot.workerOutageDeadline
              | _ => false) && deadline.violation_state_id == "expired"
  | .error _ => false

end Temporal.TestpilotTests
