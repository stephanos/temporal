import Temporal.Testpilot
import Temporal.Feature.Nexus3.Testpilot
import Umpire.Space.Tests.Lowering

namespace Temporal.TestpilotTests

open temporal.server.api.testpilot.v1

private def activationKind : EntrypointDefinition → Option Nat
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
                request.response_projections.map (·.kind) == #[.PROJECTION_KIND_ONE] &&
                rule.rule_id == "server-version-present"
            | _ => false
          | _ => false
        | _, _ => false
      | _, _ => false
  | .error _ => false

#guard match Temporal.Feature.Nexus3.Testpilot.completionCase with
  | .ok output =>
      output.program.map (fun program => program.entrypoints.map activationKind) ==
        some #[some 0, some 1, some 2] &&
      match output.contract.map (·.rules.toList) with
      | some [rule] =>
          rule.kind == .CONTRACT_RULE_KIND_SAFETY && rule.horizon.isNone &&
          rule.captures.map (·.capture_id) == #["scheduled-event"] &&
          rule.transitions.map (·.transition_id) == #[
            "capture-scheduled-event",
            "match-started-reference",
            "match-completed-event"
          ] && rule.transitions.map (·.support_kind) == #[
            .CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
            .CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
            .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]
      | _ => false
  | .error _ => false

-- The worker-outage Case's stop instruction is the lowered fault intent itself, not a hand-written
-- copy of one: the Program the fixture renders carries exactly what `lower` produced.
#guard match Temporal.Testpilot.stopIntent.lower Temporal.Testpilot.stopRealization,
    Temporal.Testpilot.workerOutageCase with
  | .ok stop, .ok artifact =>
      (artifact.program.bind fun program =>
        (program.entrypoints.find? fun entrypoint => entrypoint.entrypoint_id == "controller").bind
          fun controller => controller.instructions[0]?).any
        (Umpire.SpaceLoweringTests.sameFaultNode · stop)
  | _, _ => false

-- The outage rule is the checked-in `rule_events` horizon: no elapsed-time bound participates.
#guard match Temporal.Testpilot.workerOutageCase with
  | .ok artifact =>
      (artifact.contract.bind fun contract =>
        (contract.rules.find? fun rule => rule.rule_id == "worker-outage-order").bind
          (·.horizon)).any fun horizon =>
            horizon.rule_events == Temporal.Testpilot.workerOutageHorizon &&
              horizon.elapsed_milliseconds == 0 && horizon.violation_state_id == "expired"
  | .error _ => false

end Temporal.TestpilotTests
