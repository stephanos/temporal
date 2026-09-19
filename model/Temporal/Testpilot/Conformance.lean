import Temporal.Case.ReadKind
import Temporal.Case.Support
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

/-! ### One accepted Case per evidence source, and the two declaration rejections

Each Case declares one evidence kind on its Program and lifts it once: a history read names the
declaration by its rule, a Run Event is lifted as the runtime records it, and a read declaration is
polled by a `ReadEvidence` instruction. The Contract is satisfied by the event that carries the
lifted evidence, so the Verdict pins the lift and not only the instruction. -/

private def evidenceObservation := "evidence"

/-- The monitor rule that reads the lifted evidence off the event kind that carries it. -/
private def evidenceRule (carrier : RunEventKind) : ContractRule :=
  Contract.rule "result" .CONTRACT_RULE_KIND_SAFETY "pending"
    #[Contract.state "pending" .CONTRACT_STATE_STATUS_PENDING,
      Contract.state "terminal" .CONTRACT_STATE_STATUS_SATISFIED]
    #[Contract.transition "complete" "pending" "terminal" #[carrier]
      (Expr.present (Expr.observation evidenceObservation))
      .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]

private def evidenceProgram (caseId : String) (roles : Array Role)
    (nodes : Array InstructionNode) (evidence : Array EvidenceDeclaration) : Program :=
  Program.make (caseId ++ ".program")
    (#[Program.role workflowServiceRole .ROLE_KIND_ENDPOINT] ++ roles)
    #[] #[Program.observation evidenceObservation Temporal.Case.Support.correlatedEvidenceType]
    #[Program.controller "controller" nodes]
    (Program.cleanup "cleanup" #[]) evidence

private def evidenceCase (variant : String) (carrier : RunEventKind) (roles : Array Role)
    (nodes : Array InstructionNode) (evidence : Array EvidenceDeclaration) :
    Except Umpire.Case.Compiler.Error Case :=
  let caseId := "temporal.case.conformance.satisfied." ++ variant
  let property := conformanceProperty caseId
  Umpire.Case.Compiler.compile {
    version := { major := 1 }
    caseId
    producerId := "temporal.case.compiler"
    producerVersion := "1"
    definitions := [binding "temporal.workflow-service" "temporal-workflow-service/v1" .target, property]
    sources := [source]
    knownGaps := []
    program := evidenceProgram caseId roles nodes evidence
    contractId := caseId ++ ".contract"
    properties := [.monitor property (evidenceRule carrier)]
  }

private def startedArm := "nexus_operation_started_event_attributes"
private def historySourceId := "history"

private def historyDeclaration (evidenceId : String) : EvidenceDeclaration :=
  Program.historyEvidenceDeclaration evidenceId historySourceId startedArm
    (historyAttribute startedArm "scheduled_event_id")
    #[Program.evidenceScope "run" "conformance"]

private def historyNode (ruleName : String) : InstructionNode :=
  Program.node "history"
    (Program.invokeRpc workflowServiceRole Temporal.Case.Support.getHistoryMethod #[]
      #[Program.responseRead historyEvents .READ_CARDINALITY_EMIT_EACH
        #[Program.correlatedEvidenceTarget evidenceObservation
          #[Program.declaredEvidenceRule ruleName]]])
    (Program.instructionLimits (timeoutMilliseconds := some 5000))

/-- A history event lifted by the rule that names its declaration. -/
def conformanceHistoryEvidenceCase : Except Umpire.Case.Compiler.Error Case :=
  evidenceCase "history-evidence" .RUN_EVENT_KIND_INSTRUCTION_COMPLETED #[]
    #[historyNode "started"] #[historyDeclaration "started"]

/-- A Run Event lifted as the runtime records it: the fault the controller injects. -/
def conformanceRunEventEvidenceCase : Except Umpire.Case.Compiler.Error Case :=
  evidenceCase "run-event-evidence" .RUN_EVENT_KIND_FAULT_INJECTED
    #[Program.role workerRole .ROLE_KIND_WORKER (namespaceBindingId := "temporal.worker.namespace"),
      Program.role queueRole .ROLE_KIND_TASK_QUEUE
        (namespaceBindingId := "temporal.worker.namespace")
        (resourceBindingId := "temporal.task-queue.resource")]
    #[Program.node "fault" (Program.injectFault queueRole .FAULT_KIND_WORKER_STOP)
      (Program.instructionLimits (timeoutMilliseconds := some 5000))]
    #[Program.runEventEvidenceDeclaration "faultInjected" "run-events"
      .RUN_EVENT_KIND_FAULT_INJECTED (field "role_id")
      #[Program.evidenceScope "run" "conformance"]]

/-- The pending operation's attempt count read back through `DescribeWorkflowExecution`, polled
until an element's attempt is above one. -/
def conformanceReadEvidenceCase : Except Umpire.Case.Compiler.Error Case :=
  let read := Temporal.Case.ReadKind.pendingAttempts
  evidenceCase "read-evidence" .RUN_EVENT_KIND_INSTRUCTION_COMPLETED #[]
    #[Program.node "pending-attempts"
      (Program.readEvidence read.name workflowServiceRole #[]
        (Expr.compare .COMPARISON_OPERATOR_GREATER_THAN
          (Expr.path Expr.projectedValue (field "attempt")) (signedInteger 1))
        100)
      (Program.instructionLimits (timeoutMilliseconds := some 5000))]
    #[Program.readEvidenceDeclaration read.name "describe" read.method read.path
      read.operationKey #[Program.evidenceScope "run" "conformance"]
      (read.fields.toArray.map fun (fieldId, path) => Program.evidenceField fieldId path)]

private def evidenceRejectionCase (variant : String) (nodes : Array InstructionNode)
    (evidence : Array EvidenceDeclaration) : Except Umpire.Case.Compiler.Error Case :=
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
    program := evidenceProgram caseId #[] nodes evidence
    contractId := caseId ++ ".contract"
    properties := [.monitor property (conformanceRule .CONTRACT_STATE_STATUS_SATISFIED true)]
  }

/-- A lift rule naming a kind no declaration carries. -/
def conformanceUndeclaredEvidenceRejectionCase : Except Umpire.Case.Compiler.Error Case :=
  evidenceRejectionCase "undeclared-evidence" #[historyNode "completed"]
    #[historyDeclaration "started"]

/-- The same source and operation key path declared twice, under two identities. -/
def conformanceDuplicateEvidenceRejectionCase : Except Umpire.Case.Compiler.Error Case :=
  evidenceRejectionCase "duplicate-evidence" #[historyNode "started"]
    #[historyDeclaration "started", historyDeclaration "started-again"]

end Temporal.Testpilot
