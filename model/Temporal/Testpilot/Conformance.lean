import Temporal.Testpilot.GetSystemInfo

namespace Temporal.Testpilot

open CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

private def conformanceProperty (caseId : String) :=
  binding (caseId ++ ".property") (caseId ++ "/property/v1") .property

private def conformanceRule
    (terminal : ContractStateStatus)
    (matchesEvent : Bool) : ContractRule :=
  Contract.rule "result" .CONTRACT_RULE_KIND_SAFETY "pending"
    #[Contract.state "pending" .CONTRACT_STATE_STATUS_PENDING,
      Contract.state "terminal" terminal]
    #[Contract.transition "complete" "pending" "terminal"
      #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
      (Expr.literal (Value.boolean matchesEvent))
      .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]

private def conformanceNode (instructionId : String) (guard : Option Expression := none) :
    InstructionNode :=
  Program.node instructionId (Program.invokeRpc workflowServiceRole getSystemInfoMethod)
    (Program.instructionLimits (timeoutMilliseconds := some 5000)) (guard := guard)

private def conformanceProgram (caseId : String) (cleanupFailure : Bool)
    (guard : Option Expression) : Program :=
  Program.make (caseId ++ ".program")
    #[Program.role workflowServiceRole .ROLE_KIND_ENDPOINT]
    #[] #[]
    #[Program.controller "controller" #[conformanceNode "execute" guard]]
    (Program.cleanup "cleanup"
      (if cleanupFailure then #[conformanceNode "fail-cleanup"] else #[]))

private def conformanceCase
    (caseId : String)
    (terminal : ContractStateStatus)
    (matchesEvent : Bool)
    (cleanupFailure := false)
    (guard : Option Expression := none) : Except Umpire.Case.Compiler.Error Case :=
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
    program := conformanceProgram caseId cleanupFailure guard
    contractId := caseId ++ ".contract"
    properties := [.monitor property (conformanceRule terminal matchesEvent)]
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

/-- A second static-preparation rejection: the instruction guard reads an Observation, which only a
Contract predicate may read. Program and Contract share one expression type, so the Case is
authored and rendered; Go preparation rejects the reference outside its context at its path. -/
def conformanceExpressionContextRejectionCase : Except Umpire.Case.Compiler.Error Case :=
  conformanceCase "temporal.case.conformance.static-rejection.expression-context"
    .CONTRACT_STATE_STATUS_SATISFIED true (guard := some (Expr.observation "server-version"))

/-! ### Typed worker instruction rejections

Each of the four rejections R10 names is a Case the conformance Profile prepares against: a
workflow entrypoint carrying one workflow command, and a handler entrypoint carrying one reply. The
Profile admits the schedule command type alone, so a command of another type rejects as one the
Profile does not admit; the other three carry a message preparation rejects on its own. -/

private def workerRole := "temporal.worker"
private def queueRole := "temporal.task-queue"
private def nexusEndpointRole := "temporal.nexus-endpoint"

private def typedProgram (caseId : String) (command : Instruction) (reply : Instruction)
    (handleSlot : Bool := false) : Program :=
  Program.make (caseId ++ ".program")
    #[Program.role workflowServiceRole .ROLE_KIND_ENDPOINT,
      Program.role workerRole .ROLE_KIND_WORKER (namespaceBindingId := "temporal.worker.namespace"),
      Program.role queueRole .ROLE_KIND_TASK_QUEUE
        (namespaceBindingId := "temporal.worker.namespace")
        (resourceBindingId := "temporal.task-queue.resource"),
      Program.role nexusEndpointRole .ROLE_KIND_ENDPOINT
        (resourceBindingId := "temporal.nexus-endpoint.resource")]
    (if handleSlot then #[Program.handleSlot "completion-authority"] else #[]) #[]
    #[Program.controller "controller" #[conformanceNode "execute"],
      Program.workflow "workflow" "umpire-conformance-workflow" workerRole queueRole
        #[Program.node "schedule" command (Program.instructionLimits (timeoutMilliseconds := some 5000)),
          Program.node "finish" (Program.finish (text "done"))
            (Program.instructionLimits (timeoutMilliseconds := some 5000))],
      Program.nexusHandler "handler" "umpire.conformance.service" "operation" workerRole queueRole
        #[Program.node "reply" reply (Program.instructionLimits (timeoutMilliseconds := some 5000))]]
    (Program.cleanup "cleanup" #[])

private def typedRejectionCase (variant : String) (command : Instruction) (reply : Instruction)
    (handleSlot : Bool := false) : Except Umpire.Case.Compiler.Error Case :=
  let caseId := "temporal.case.conformance.static-rejection." ++ variant
  let property := conformanceProperty caseId
  Umpire.Case.Compiler.compile {
    version := { major := 1 }
    caseId
    producerId := "temporal.case.compiler"
    producerVersion := "1"
    definitions := [binding "temporal.workflow-service" "temporal-workflow-service/v1" .target, property]
    sources := [source]
    knownGaps := []
    program := typedProgram caseId command reply handleSlot
    contractId := caseId ++ ".contract"
    properties := [.monitor property (conformanceRule .CONTRACT_STATE_STATUS_SATISFIED true)]
  }

private def scheduleCommand
    (scheduleToClose : Option google.protobuf.Duration := none) : Instruction :=
  Program.scheduleNexusOperation nexusEndpointRole "umpire.conformance.service" "operation"
    (Payload.text "request") (scheduleToClose := scheduleToClose)

private def syncReply : Instruction := Program.nexusSyncReply (Payload.text "done")

/-- A command type the Profile does not admit: a timer, where the Profile admits Nexus schedules. -/
def conformanceCommandTypeRejectionCase : Except Umpire.Case.Compiler.Error Case :=
  typedRejectionCase "command-type"
    (Program.workflowCommand {
      command_type := .COMMAND_TYPE_START_TIMER
      attributes := some (.start_timer_command_attributes {
        timer_id := "timer", start_to_fire_timeout := some (Duration.seconds 1) }) })
    syncReply

/-- An invalid duration: a negative schedule-to-close timeout. -/
def conformanceInvalidDurationRejectionCase : Except Umpire.Case.Compiler.Error Case :=
  typedRejectionCase "invalid-duration" (scheduleCommand (some (Duration.seconds (-1)))) syncReply

/-- A field the Driver cannot set: the command's user metadata. -/
def conformanceUnsettableFieldRejectionCase : Except Umpire.Case.Compiler.Error Case :=
  typedRejectionCase "unsettable-field"
    (Program.workflowCommand {
      command_type := .COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION
      attributes := some (.schedule_nexus_operation_command_attributes {
        endpoint := nexusEndpointRole, service := "umpire.conformance.service"
        operation := "operation" })
      user_metadata := some {} })
    syncReply

/-- A reply the activation does not admit: a synchronous reply that publishes a handle. -/
def conformanceReplyRejectionCase : Except Umpire.Case.Compiler.Error Case :=
  typedRejectionCase "reply-not-admitted" (scheduleCommand)
    (Program.nexusHandlerReply { variant := some (.sync_success { payload := some (Payload.text "done") }) }
      "completion-authority")
    (handleSlot := true)

end Temporal.Testpilot
