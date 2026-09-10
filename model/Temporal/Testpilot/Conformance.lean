import Temporal.Testpilot.GetSystemInfo

namespace Temporal.Testpilot

open CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

private def conformanceProperty (caseId : String) :=
  binding (caseId ++ ".property") (caseId ++ "/property/v1") .property

private def conformanceRule
    (terminal : ContractStateStatus)
    (matchesEvent : Bool) : ContractRuleDefinition :=
  Contract.rule "result" .CONTRACT_RULE_KIND_SAFETY "pending"
    #[Contract.state "pending" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Contract.state "terminal" terminal]
    #[Contract.transition "complete" "pending" "terminal"
      #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
      (ContractExpr.literal (Value.boolean matchesEvent))
      .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]

private def conformanceNode (instructionId : String) : InstructionDefinition :=
  Program.node instructionId (Program.invokeRPC workflowServiceRole getSystemInfoMethod)
    bounds (outcome := some statusOutcome)

private def conformanceProgram (caseId : String) (cleanupFailure : Bool) : Program :=
  Program.make (caseId ++ ".program")
    #[Program.role workflowServiceRole .ROLE_KIND_ENDPOINT]
    #[] #[]
    #[Program.controller "controller" #[conformanceNode "execute"]]
    (Program.cleanup "cleanup"
      (if cleanupFailure then #[conformanceNode "fail-cleanup"] else #[]))
    programLimits

private def conformanceCase
    (caseId : String)
    (terminal : ContractStateStatus)
    (matchesEvent : Bool)
    (cleanupFailure := false) : Except Umpire.Case.Compiler.Error Case :=
  let property := conformanceProperty caseId
  let definitions := [
    binding "temporal.workflow-service" "temporal-workflow-service/v1" .target,
    property
  ]
  Umpire.Case.Compiler.compile {
    version := { major := 1 }
    caseId
    producerId := "temporal.case.compiler"
    producerVersion := "1"
    definitions
    sources := [source]
    knownGaps := []
    program := conformanceProgram caseId cleanupFailure
    contractId := caseId ++ ".contract"
    properties := [.monitor property (conformanceRule terminal matchesEvent)]
    contractLimits
  }

/-- Deterministic public-facade fixtures kept small enough for exact cross-language comparison. -/
def conformanceSatisfiedCase : Except Umpire.Case.Compiler.Error Case :=
  conformanceCase "temporal.case.conformance.satisfied" .CONTRACT_STATE_STATUS_SATISFIED true

def conformanceViolatedCase : Except Umpire.Case.Compiler.Error Case :=
  conformanceCase "temporal.case.conformance.violated" .CONTRACT_STATE_STATUS_VIOLATED true

def conformanceInconclusiveCase : Except Umpire.Case.Compiler.Error Case :=
  conformanceCase "temporal.case.conformance.inconclusive" .CONTRACT_STATE_STATUS_SATISFIED false

def conformanceCleanupFailureCase : Except Umpire.Case.Compiler.Error Case :=
  conformanceCase "temporal.case.conformance.cleanup-failure" .CONTRACT_STATE_STATUS_VIOLATED true true

def conformanceCrossRunIsolationCase : Except Umpire.Case.Compiler.Error Case :=
  conformanceCase "temporal.case.conformance.cross-run-isolation" .CONTRACT_STATE_STATUS_SATISFIED true

def conformanceStaticRejectionCase : Except Umpire.Case.Compiler.Error Case :=
  (conformanceCase "temporal.case.conformance.static-rejection"
    .CONTRACT_STATE_STATUS_SATISFIED true).map fun output =>
    let invalidContract := output.contract.map fun contract =>
      { contract with rules := contract.rules.map fun rule =>
          { rule with initial_state_id := "missing" } }
    { output with contract := invalidContract }

end Temporal.Testpilot
