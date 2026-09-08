import Temporal.Feature.Nexus3.Nexus
import Temporal.Testpilot.CaseSupport
import Umpire.Case.Compiler

/-!
# Nexus3 Testpilot producer

This module validates the checked completion Query and witness before lowering the Nexus-specific
Program, history correlation, and success monitor into generated values for Umpire Case assembly.
-/

namespace Temporal.Feature.Nexus3.Testpilot

open Umpire
open Umpire.Case.Compiler
open Temporal.Testpilot.CaseSupport
open Testpilot.Authoring
open temporal.server.api.testpilot.v1

private def workflowServiceRole := "temporal.workflow-service"
private def workerRole := "temporal.worker"
private def taskQueueRole := "temporal.task-queue"
private def nexusEndpointRole := "temporal.nexus-endpoint"
private def workerNamespaceBinding := "temporal.worker.namespace"
private def taskQueueBinding := "temporal.task-queue.resource"
private def nexusEndpointBinding := "temporal.nexus-endpoint.resource"
private def startWorkflowMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
private def getHistoryMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"

private def historyEventType : ValueType :=
  Types.singular (Types.messageType "temporal.api.history.v1.HistoryEvent")

private def textOutcome : InstructionOutcomeDefinition :=
  Program.outcome #[
    Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_STATUS statusType,
    Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_VALUE textType]

private def nested (names : List String) : FieldPath :=
  Path.make (names.map Path.field).toArray

private def historyEvents : FieldPath :=
  Path.make #[Path.field "history", Path.repeated "events"]

private def historyAttribute (selected name : String) : FieldPath :=
  Path.make #[Path.oneofSelector "attributes" selected, Path.field name]

private def text (value : String) : ProgramExpression := ProgramExpr.literal (Value.text value)
private def signedInteger (value : Int) : ProgramExpression :=
  ProgramExpr.literal (Value.signedInteger value)
private def observed (id : String) : ContractExpression := ContractExpr.observation id
private def captured (id : String) : ContractExpression := ContractExpr.capture id
private def runId : ProgramExpression := ProgramExpr.run
private def projected (value : ContractExpression) (path : FieldPath) : ContractExpression :=
  ContractExpr.path value path

private def succeeded (entrypoint instruction : String) : ProgramExpression :=
  let status := ProgramExpr.outcome (Ref.instruction entrypoint instruction)
    .INSTRUCTION_OUTCOME_FIELD_STATUS
  ProgramExpr.all #[ProgramExpr.present status,
    ProgramExpr.equals status (ProgramExpr.literal (Value.enumeration 1))]

private def assign (target : FieldPath) (value : ProgramExpression) : RequestAssignment :=
  Program.requestAssignment target value

private def rpc
    (id method : String)
    (dependencies : Array InstructionRef)
    (assignments : Array RequestAssignment)
    (projections : Array ResponseProjection)
    (guard : Option ProgramExpression := none)
    (reservations : Array ActivationReservationDefinition := #[]) : InstructionDefinition :=
  Program.node id (Program.invokeRPC workflowServiceRole method assignments projections)
    (bounds 10000 128) dependencies guard (some statusOutcome) reservations

private def historyAssignments : Array RequestAssignment := #[
  Program.environmentAssignment (field "namespace") workerNamespaceBinding,
  assign (nested ["execution", "workflow_id"]) runId,
  assign (field "maximum_page_size") (signedInteger 64),
  assign (field "wait_new_event") (boolean true)
]

private def historyNode : InstructionDefinition :=
  rpc "history" getHistoryMethod #[Ref.instruction "controller" "complete-nexus-operation"]
    historyAssignments #[
    project historyEvents "history-event" .PROJECTION_KIND_EMIT_EACH
  ] (some (succeeded "controller" "complete-nexus-operation"))

private def program : Program :=
  Program.make "temporal.case.async-nexus.program"
    #[
      Program.role workflowServiceRole .ROLE_KIND_ENDPOINT,
      Program.role workerRole .ROLE_KIND_WORKER
        (namespaceBindingId := workerNamespaceBinding),
      Program.role taskQueueRole .ROLE_KIND_TASK_QUEUE
        (namespaceBindingId := workerNamespaceBinding) (resourceBindingId := taskQueueBinding),
      Program.role nexusEndpointRole .ROLE_KIND_ENDPOINT
        (resourceBindingId := nexusEndpointBinding)]
    #[Program.capabilitySlot "completion-authority"]
    #[Program.observation "history-event" historyEventType]
    #[
      Program.controller "controller" #[
        rpc "start-workflow" startWorkflowMethod #[] #[
          Program.environmentAssignment (field "namespace") workerNamespaceBinding,
          assign (field "workflow_id") runId,
          assign (nested ["workflow_type", "name"]) (text "umpire-async-nexus-workflow"),
          Program.environmentAssignment (nested ["task_queue", "name"]) taskQueueBinding,
          assign (field "request_id") runId
        ] #[] none #[
          Program.reservation "workflow" 1,
          Program.reservation "handler" 1
        ],
        Program.node "await-completion-authority" (Program.awaitSlot "completion-authority")
          (bounds 10000) #[Ref.instruction "controller" "start-workflow"]
          (some (succeeded "controller" "start-workflow")) (some statusOutcome),
        Program.node "complete-nexus-operation"
          (Program.completeNexusOperation "completion-authority" (text "completed"))
          (bounds 10000) #[Ref.instruction "controller" "await-completion-authority"]
          (some (succeeded "controller" "await-completion-authority")) (some statusOutcome),
        historyNode],
      Program.workflow "workflow" "umpire-async-nexus-workflow" workerRole taskQueueRole #[
        Program.node "start-nexus-operation"
          (Program.startNexusOperation nexusEndpointRole "umpire.case.service" "complete"
            (text "request"))
          (bounds 10000) #[] none (some statusOutcome),
        Program.node "await-nexus-operation"
          (Program.awaitOutcome (Ref.instruction "workflow" "start-nexus-operation"))
          (bounds 10000) #[Ref.instruction "workflow" "start-nexus-operation"]
          none (some textOutcome),
        Program.node "finish-workflow"
          (Program.finish (ProgramExpr.outcome
            (Ref.instruction "workflow" "await-nexus-operation")
            .INSTRUCTION_OUTCOME_FIELD_VALUE))
          bounds #[Ref.instruction "workflow" "await-nexus-operation"]
          (some (succeeded "workflow" "await-nexus-operation")) (some statusOutcome)],
      Program.nexusHandler "handler" "umpire.case.service" "complete" workerRole taskQueueRole #[
        Program.node "respond-async"
          (Program.respondNexus .NEXUS_RESPONSE_KIND_ASYNCHRONOUS
            (text "accepted") "completion-authority")
          bounds #[] none (some statusOutcome)]]
    (Program.cleanup "cleanup" #[])
    programLimits
    (environment := #[
      Program.environment workerNamespaceBinding,
      Program.environment taskQueueBinding,
      Program.environment nexusEndpointBinding])

private def successRule (checkedProperty : CheckedProperty) : ContractRuleDefinition :=
  Monitor.rule (checkedProperty.id.value ++ ".correlated-history")
    .CONTRACT_RULE_KIND_SAFETY "pending"
    #[
      Monitor.state "pending" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Monitor.state "scheduled-correlated" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Monitor.state "started-correlated" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Monitor.state "satisfied" .CONTRACT_STATE_STATUS_SATISFIED]
    #[
      Monitor.transition "capture-scheduled-event" "pending" "scheduled-correlated"
        #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
        (ContractExpr.all #[
          ContractExpr.present (observed "history-event"),
          ContractExpr.present (projected (observed "history-event") (field "event_id")),
          ContractExpr.present (projected (observed "history-event")
            (historyAttribute "nexus_operation_scheduled_event_attributes" "request_id"))])
        .CONTRACT_SUPPORT_KIND_MATCHING_EVENT
        #[Monitor.captureAssignment "scheduled-event" "history-event"],
      Monitor.transition "match-started-reference" "scheduled-correlated" "started-correlated"
        #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
        (ContractExpr.all #[
          ContractExpr.present (captured "scheduled-event"),
          ContractExpr.present (projected (captured "scheduled-event") (field "event_id")),
          ContractExpr.present (projected (observed "history-event")
            (historyAttribute "nexus_operation_started_event_attributes" "scheduled_event_id")),
          ContractExpr.equals
            (projected (captured "scheduled-event") (field "event_id"))
            (projected (observed "history-event")
              (historyAttribute "nexus_operation_started_event_attributes" "scheduled_event_id"))])
        .CONTRACT_SUPPORT_KIND_MATCHING_EVENT,
      Monitor.transition "match-completed-event" "started-correlated" "satisfied"
        #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
        (ContractExpr.all #[
          ContractExpr.present (captured "scheduled-event"),
          ContractExpr.present (projected (captured "scheduled-event") (field "event_id")),
          ContractExpr.present (projected (captured "scheduled-event")
            (historyAttribute "nexus_operation_scheduled_event_attributes" "request_id")),
          ContractExpr.present (projected (observed "history-event")
            (historyAttribute "nexus_operation_completed_event_attributes" "request_id")),
          ContractExpr.present (projected (observed "history-event")
            (historyAttribute "nexus_operation_completed_event_attributes" "scheduled_event_id")),
          ContractExpr.equals
            (projected (captured "scheduled-event")
              (historyAttribute "nexus_operation_scheduled_event_attributes" "request_id"))
            (projected (observed "history-event")
              (historyAttribute "nexus_operation_completed_event_attributes" "request_id")),
          ContractExpr.equals
            (projected (captured "scheduled-event") (field "event_id"))
            (projected (observed "history-event")
              (historyAttribute "nexus_operation_completed_event_attributes"
                "scheduled_event_id"))])
        .CONTRACT_SUPPORT_KIND_MATCHING_EVENT]
    (captures := #[Monitor.capture "scheduled-event"
      (Monitor.messageCapture "temporal.api.history.v1.HistoryEvent")])

private def supportsSuccessProperty
    (checkedProperty : CheckedProperty)
    (values : Authoring.ModelVocabulary) : Bool :=
  match checkedProperty.clauses with
  | [.inputOutput _ selectedAction successFactPattern,
      .transitionContract _ selectedAction' successOutcomePattern,
      .transitionContract _ selectedAction'' successStatePattern] =>
      selectedAction == PropertyPattern.selectedAction values.awaitSuccessAction &&
      selectedAction' == PropertyPattern.selectedAction values.awaitSuccessAction &&
      selectedAction'' == PropertyPattern.selectedAction values.awaitSuccessAction &&
      successStatePattern == PropertyPattern.resultingState values.succeededState &&
      successOutcomePattern == PropertyPattern.modelOutcome values.completedOutcome &&
      successFactPattern == PropertyPattern.fact values.succeededFact
  | _ => false

private def loweringError (definitionId construct : String) : LoweringError := {
  sourceDefinitionId := definitionId
  source := Authoring.source
  construct
}

private def sameTarget
    (left : QueryTarget LawStatement)
    (right : QueryTarget lifecycle.lawStatement) : Bool :=
  left.id == right.id && left.source == right.source && left.definitions == right.definitions &&
    left.requiredCapabilities == right.requiredCapabilities &&
    left.behaviorDescription == right.behaviorDescription &&
    left.canonicalMetadata == right.canonicalMetadata &&
    left.behaviorFingerprint == right.behaviorFingerprint

private def sameQuery
    (left : CheckedQuery LawStatement)
    (right : CheckedQuery lifecycle.lawStatement) : Bool :=
  left.id == right.id && left.source == right.source && left.version == right.version &&
    left.form == right.form && left.quantifier == right.quantifier && left.claim == right.claim &&
    left.behavior == right.behavior && sameTarget left.target right.target &&
    left.limits == right.limits && left.policy == right.policy &&
    left.authoredKnownGaps == right.authoredKnownGaps &&
    left.targetComposition == right.targetComposition && left.documentation == right.documentation &&
    left.canonicalMetadata == right.canonicalMetadata &&
    left.behaviorFingerprint == right.behaviorFingerprint

private def checkedCompletion : Except LoweringError (Authoring.CheckedModel lifecycle) :=
  completion.mapError fun _ =>
    loweringError "temporal.nexus3.query.completion" "checked-completion"

/-- Lower only the exact checked Nexus3 completion Query and its selected witness. -/
def produceCompletionCase
    (target : QueryTarget LawStatement)
    (checkedProperty : CheckedProperty)
    (checkedBehavior : CheckedBehavior)
    (checkedQuery : CheckedQuery LawStatement)
    (witness? : Option BehaviorTrace) :
    Except LoweringError temporal.server.api.testpilot.v1.Case := do
  if let some clause := checkedProperty.scopedClauses.head? then
    throw {
      sourceDefinitionId := clause.declaration.id.value
      source := clause.declaration.source
      construct := "property.scoped-eventually-within/v1" }
  let expected ← checkedCompletion
  unless sameTarget target expected.target do
    throw (loweringError target.id.value "target")
  unless checkedProperty == expected.property do
    throw (loweringError checkedProperty.id.value "property")
  unless supportsSuccessProperty checkedProperty expected.vocabulary do
    throw (loweringError checkedProperty.id.value "property.success-evidence-mapping")
  unless checkedBehavior == expected.behavior do
    throw (loweringError checkedBehavior.id.value "behavior")
  unless sameQuery checkedQuery expected.query do
    throw (loweringError checkedQuery.id.value "query")
  let selectedWitness ← match witness? with
    | some selectedWitness => pure selectedWitness
    | none => throw (loweringError checkedQuery.id.value "witness.absent")
  unless selectedWitness == expected.witness do
    throw (loweringError checkedQuery.id.value "witness.mismatch")
  let propertyBinding := binding checkedProperty.id.value
    checkedProperty.behaviorFingerprint.render .«property»
  let definitions := [
      binding target.id.value target.behaviorFingerprint.render .target,
      binding checkedBehavior.id.value checkedBehavior.behaviorFingerprint.render .«behavior»,
      binding checkedQuery.id.value checkedQuery.behaviorFingerprint.render .«query»,
      propertyBinding
    ]
  compile {
    version := { major := 1 }
    caseId := "temporal.case.async-nexus-success"
    producerId := "temporal.nexus3.testpilot"
    producerVersion := "1"
    definitions
    sources := [target.source, checkedBehavior.source, checkedQuery.source, checkedProperty.source]
    knownGaps := checkedQuery.authoredKnownGaps.toCaseKnownGaps
    program
    contractId := "temporal.case.async-nexus-success.contract"
    properties := [.monitor propertyBinding (successRule checkedProperty)]
    contractLimits
  }

/-- The checked Nexus3 completion declaration lowered to the closed Case format. -/
def completionCase : Except LoweringError temporal.server.api.testpilot.v1.Case := do
  let checked ← checkedCompletion
  produceCompletionCase checked.target checked.property checked.behavior checked.query
    (some checked.witness)

end Temporal.Feature.Nexus3.Testpilot
