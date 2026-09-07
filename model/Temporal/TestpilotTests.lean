import Temporal.Testpilot
import Temporal.Feature.Nexus3.Testpilot

namespace Temporal.TestpilotTests

open Umpire.Case

#guard match Temporal.Testpilot.getSystemInfoCase with
  | .ok output =>
      match output.program.entrypoints, output.contract.rules with
      | [entrypoint], [rule] =>
          match entrypoint.nodes with
          | [node] =>
              node.instruction == Instruction.invokeRPC {
                endpointRoleId := Temporal.Testpilot.workflowServiceRole
                method := Temporal.Testpilot.getSystemInfoMethod
                requestAssignments := []
                responseProjections := [{
                  source := { segments := [{ field := "server_version" }] }
                  cardinality := .one
                  sinks := [.observation "server-version"]
                }]
              } && rule.ruleId == "server-version-present"
          | _ => false
      | _, _ => false
  | .error _ => false

#guard match Temporal.Feature.Nexus3.Testpilot.completionCase with
  | .ok output =>
      output.program.entrypoints.map (·.context) == [.controller, .workflow, .nexusHandler] &&
      match output.contract.rules with
      | [rule] =>
          rule.kind == .safety && rule.horizon.isNone &&
          rule.captures.map (·.captureId) == ["scheduled-event"] &&
          rule.transitions.map (·.transitionId) == [
            "capture-scheduled-event",
            "match-started-reference",
            "match-completed-event"
          ] &&
          rule.transitions.map (·.support) == [
            ContractSupport.matchingEvent,
            ContractSupport.matchingEvent,
            ContractSupport.matchingEvent
          ]
      | _ => false
  | .error _ => false

end Temporal.TestpilotTests
