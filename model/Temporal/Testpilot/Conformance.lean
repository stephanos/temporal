import Temporal.Testpilot.GetSystemInfo

namespace Temporal.Testpilot

open Umpire
open Umpire.Case
open Umpire.Case.Compiler
open CaseSupport

private def conformanceProperty (caseId : String) :=
  binding (caseId ++ ".property") (caseId ++ "/property/v1") .property

private def conformanceRule
    (terminal : ContractTerminalState)
    (matchesEvent : Bool) : ContractRule := {
  ruleId := "result"
  kind := .safety
  initialState := "pending"
  states := [
    { stateId := "pending" },
    { stateId := "terminal", terminal }
  ]
  transitions := [{
    transitionId := "complete"
    sourceState := "pending"
    targetState := "terminal"
    eventKinds := [.instructionCompleted]
    predicate := boolean matchesEvent
    support := .matchingEvent
  }]
}

private def conformanceNode (instructionId : String) : InstructionNode := {
  instructionId
  dependencies := []
  instruction := .invokeRPC {
    endpointRoleId := workflowServiceRole
    method := getSystemInfoMethod
    requestAssignments := []
    responseProjections := []
  }
  outcome := statusOutcome
  bounds := bounds
}

private def conformanceProgram (caseId : String) (cleanupFailure : Bool) : Program := {
  programId := caseId ++ ".program"
  roles := [{ roleId := workflowServiceRole, kind := .endpoint }]
  slots := []
  observations := []
  entrypoints := [{
    entrypointId := "controller"
    context := .controller
    activation := .controller
    nodes := [conformanceNode "execute"]
  }]
  cleanup := {
    entrypointId := "cleanup"
    context := .controller
    nodes := if cleanupFailure then [conformanceNode "fail-cleanup"] else []
  }
  limits := programLimits
}

private def conformanceCase
    (caseId : String)
    (terminal : ContractTerminalState)
    (matchesEvent : Bool)
    (cleanupFailure := false) : Except LoweringError temporal.server.api.testpilot.v1.Case :=
  let property := conformanceProperty caseId
  compile {
    version := { major := 1 }
    caseId
    producerId := "temporal.case.compiler"
    producerVersion := "1"
    definitions := [
      binding "temporal.workflow-service" "temporal-workflow-service/v1" .target,
      property
    ]
    sources := [source]
    knownGaps := []
    program := conformanceProgram caseId cleanupFailure
    contractId := caseId ++ ".contract"
    properties := [.monitor property (conformanceRule terminal matchesEvent)]
    contractLimits
  }

/-- Deterministic public-facade fixtures kept small enough for exact cross-language comparison. -/
def conformanceSatisfiedCase : Except LoweringError temporal.server.api.testpilot.v1.Case :=
  conformanceCase "temporal.case.conformance.satisfied" .satisfied true

def conformanceViolatedCase : Except LoweringError temporal.server.api.testpilot.v1.Case :=
  conformanceCase "temporal.case.conformance.violated" .violated true

def conformanceInconclusiveCase : Except LoweringError temporal.server.api.testpilot.v1.Case :=
  conformanceCase "temporal.case.conformance.inconclusive" .satisfied false

def conformanceCleanupFailureCase : Except LoweringError temporal.server.api.testpilot.v1.Case :=
  conformanceCase "temporal.case.conformance.cleanup-failure" .violated true true

def conformanceCrossRunIsolationCase : Except LoweringError temporal.server.api.testpilot.v1.Case :=
  conformanceCase "temporal.case.conformance.cross-run-isolation" .satisfied true

def conformanceStaticRejectionCase : Except LoweringError temporal.server.api.testpilot.v1.Case :=
  (conformanceCase "temporal.case.conformance.static-rejection" .satisfied true).map fun output =>
    let invalidContract := output.contract.map fun contract =>
      { contract with rules := contract.rules.map fun rule => { rule with initial_state_id := "missing" } }
    { output with contract := invalidContract }

end Temporal.Testpilot
