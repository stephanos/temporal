import Temporal.Testpilot.CaseSupport
import Umpire.Case.Compiler

namespace Temporal.Testpilot

open CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

def workflowServiceRole := "temporal.workflow-service"
def getSystemInfoMethod := "/temporal.api.workflowservice.v1.WorkflowService/GetSystemInfo"

private def getSystemInfoProperty :=
  binding "temporal.case.get-system-info.property.server-version"
    "temporal-case-get-system-info-property/v1" .property

private def getSystemInfoRule : ContractRuleDefinition :=
  Contract.rule "server-version-present" .CONTRACT_RULE_KIND_SAFETY "pending"
    #[Contract.state "pending" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Contract.state "satisfied" .CONTRACT_STATE_STATUS_SATISFIED]
    #[Contract.transition "observe-server-version" "pending" "satisfied"
      #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
      (ContractExpr.present (ContractExpr.observation "server-version"))
      .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]

/-- An orthogonal unary Case with an empty request and typed response projection. -/
def getSystemInfoCase : Except Umpire.Case.Compiler.Error Case :=
  let definitions := [
    binding "temporal.workflow-service" "temporal-workflow-service/v1" .target,
    getSystemInfoProperty
  ]
  let program := Program.make "temporal.case.get-system-info.program"
    #[Program.role workflowServiceRole .ROLE_KIND_ENDPOINT]
    #[]
    #[Program.observation "server-version" textType]
    #[Program.controller "controller" #[Program.node "get-system-info"
      (Program.invokeRPC workflowServiceRole getSystemInfoMethod #[]
        #[project (field "server_version") "server-version"])
      bounds (outcome := some statusOutcome)]]
    (Program.cleanup "cleanup" #[])
    programLimits
  Umpire.Case.Compiler.compile {
    version := { major := 1 }
    caseId := "temporal.case.get-system-info"
    producerId := "temporal.case.compiler"
    producerVersion := "1"
    definitions
    sources := [source]
    knownGaps := []
    program
    contractId := "temporal.case.get-system-info.contract"
    properties := [.monitor getSystemInfoProperty getSystemInfoRule]
    contractLimits
  }

end Temporal.Testpilot
