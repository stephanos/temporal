import Temporal.Testpilot.CaseSupport

namespace Temporal.Testpilot

open Umpire
open Umpire.Case
open Umpire.Case.Compiler
open CaseSupport

def workflowServiceRole := "temporal.workflow-service"
def getSystemInfoMethod := "/temporal.api.workflowservice.v1.WorkflowService/GetSystemInfo"

private def getSystemInfoProperty :=
  binding "temporal.case.get-system-info.property.server-version"
    "temporal-case-get-system-info-property/v1" .property

private def getSystemInfoRule : ContractRule := {
  ruleId := "server-version-present"
  kind := .safety
  initialState := "pending"
  states := [{ stateId := "pending" }, { stateId := "satisfied", terminal := .satisfied }]
  transitions := [{
    transitionId := "observe-server-version"
    sourceState := "pending"
    targetState := "satisfied"
    eventKinds := [.instructionCompleted]
    predicate := .present (.observation { observationId := "server-version" })
    support := .matchingEvent
  }]
}

/-- An orthogonal unary Case with an empty request and typed response projection. -/
def getSystemInfoCase : Except LoweringError temporal.server.api.testpilot.v1.Case := compile {
  version := { major := 1 }
  caseId := "temporal.case.get-system-info"
  producerId := "temporal.case.compiler"
  producerVersion := "1"
  definitions := [
    binding "temporal.workflow-service" "temporal-workflow-service/v1" .target,
    getSystemInfoProperty
  ]
  sources := [source]
  knownGaps := []
  program := {
    programId := "temporal.case.get-system-info.program"
    roles := [{ roleId := workflowServiceRole, kind := .endpoint }]
    slots := []
    observations := [{ observationId := "server-version", type := textType }]
    entrypoints := [{
      entrypointId := "controller"
      context := .controller
      activation := .controller
      nodes := [{
        instructionId := "get-system-info"
        dependencies := []
        instruction := .invokeRPC {
          endpointRoleId := workflowServiceRole
          method := getSystemInfoMethod
          requestAssignments := []
          responseProjections := [project (field "server_version") "server-version"]
        }
        outcome := statusOutcome
        bounds := bounds
      }]
    }]
    cleanup := { entrypointId := "cleanup", context := .controller, nodes := [] }
    limits := programLimits
  }
  contractId := "temporal.case.get-system-info.contract"
  properties := [.monitor getSystemInfoProperty getSystemInfoRule]
  contractLimits
}

end Temporal.Testpilot
