module

public import Testpilot.Authoring
public import Testpilot.ProtoJSON

public section

/-! A producer-neutral, bounded Testpilot Case used for cross-language admission tests. -/

namespace Testpilot.Examples.Synthetic

open temporal.server.api.testpilot.v1
open Testpilot.Authoring

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
      (Program.finish (Expr.literal (Value.messageValue formatVersion)))]]
  (Program.cleanup "cleanup" #[])

private def contract : Contract := Contract.contract "testpilot.synthetic.contract" #[
  Contract.rule "completion" .CONTRACT_RULE_KIND_SAFETY "open"
    #[Contract.state "open" .CONTRACT_STATE_STATUS_PENDING,
      Contract.state "done" .CONTRACT_STATE_STATUS_SATISFIED]
    #[Contract.transition "complete" "open" "done" #[.RUN_EVENT_KIND_RUN_CLOSED]
      (Expr.literal (Value.boolean true))]
]

/-- A deterministic Case authored without an Umpire or Temporal dependency. -/
def case : Case := Testpilot.Authoring.case 1 "testpilot.synthetic.case" program contract
  (provenance "standalone.lean.testpilot" "1")

/-- Render the synthetic Case through Testpilot's canonical ProtoJSON policy. -/
def canonical : IO (Except Testpilot.ProtoJSON.Error String) :=
  Testpilot.ProtoJSON.canonical case

end Testpilot.Examples.Synthetic
