module

public import Testpilot.Authoring
public import Testpilot.ProtoJSON

public section

/-! A producer-neutral, bounded Testpilot Case used for cross-language admission tests. -/

namespace Testpilot.Examples.Synthetic

open temporal.server.api.testpilot.v1
open Testpilot.Authoring

private def instructionLimits := Program.instructionLimits 1000 1 1 1024

private def formatVersion : google.protobuf.Any := {
  type_url := "type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion"
  value := ByteArray.mk #[8, 1, 16, 2]
}

private def program : temporal.server.api.testpilot.v1.Program := Program.make
  "testpilot.synthetic.program"
  #[Program.role "worker" .ROLE_KIND_WORKER (namespaceBindingId := "namespace"),
    Program.role "task.queue" .ROLE_KIND_TASK_QUEUE
      (namespaceBindingId := "namespace") (resourceBindingId := "task.queue")]
  #[]
  #[]
  #[Program.workflow "workflow" "SyntheticWorkflow" "worker" "task.queue"
    #[Program.node "finish"
      (Program.finish (ProgramExpr.literal (Value.messageValue formatVersion))) instructionLimits
      (outcome := some (Program.outcome #[Program.outcomeField
        .INSTRUCTION_OUTCOME_FIELD_VALUE (Types.singular (Types.messageType
          "temporal.server.api.testpilot.v1.FormatVersion"))]))]]
  (Program.cleanup "cleanup" #[])
  (Program.limits 1 1 1 1 1 8 8 4 1024 1024 1000 1000)
  (environment := #[Program.environment "namespace", Program.environment "task.queue"])

private def contract : Contract := Monitor.contract "testpilot.synthetic.contract" #[
  Monitor.rule "completion" .CONTRACT_RULE_KIND_SAFETY "open"
    #[Monitor.state "open" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Monitor.state "done" .CONTRACT_STATE_STATUS_SATISFIED]
    #[Monitor.transition "complete" "open" "done" #[.RUN_EVENT_KIND_RUN_CLOSED]
      (ContractExpr.literal (Value.boolean true))]
] (Monitor.limits 1 2 1 8 32 64 1 1024)

/-- A deterministic Case authored without an Umpire or Temporal dependency. -/
def case : Case := Testpilot.Authoring.case 1 "testpilot.synthetic.case" program contract
  (provenance "standalone.lean.testpilot" "1" (ByteArray.mk #[0, 255, 128]))

/-- Render the synthetic Case through Testpilot's canonical ProtoJSON policy. -/
def canonical : IO (Except Testpilot.ProtoJSON.Error String) :=
  Testpilot.ProtoJSON.canonical case

end Testpilot.Examples.Synthetic
