import Temporal.Testpilot
import Temporal.Feature.Nexus3.Testpilot

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

end Temporal.TestpilotTests
