import Temporal.CaseRuntime

namespace Temporal.CaseRuntimeTests

open Umpire.Case

#guard match Temporal.CaseRuntime.getSystemInfoCase with
  | .ok output =>
      match output.program.entrypoints, output.contract.rules with
      | [entrypoint], [rule] =>
          match entrypoint.nodes with
          | [node] =>
              node.instruction == Instruction.invokeRPC {
                endpointRoleId := Temporal.CaseRuntime.workflowServiceRole
                method := Temporal.CaseRuntime.getSystemInfoMethod
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

#guard match Temporal.CaseRuntime.asyncNexusCase with
  | .ok output =>
      output.program.entrypoints.map (·.context) == [.controller, .workflow, .nexusHandler] &&
      match output.contract.rules with
      | [rule] =>
          rule.captures.map (·.captureId) == ["scheduled-event"] &&
          rule.transitions.map (·.support) == [
            ContractSupport.matchingEvent,
            ContractSupport.matchingEvent,
            ContractSupport.matchingEvent
          ]
      | _ => false
  | .error _ => false

end Temporal.CaseRuntimeTests
