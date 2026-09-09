import Temporal.Feature.Nexus3.Nexus
import Temporal.Testpilot.CaseSupport
import Umpire.Case.Compiler

/-!
# Nexus3 Testpilot producer

This module carries the checked completion Query and its selected witness into generated values for
Umpire Case assembly. The Nexus-specific Program is fixed realization; the history-correlation
monitor is derived from the Facts the witness records, through the evidence projections this module
declares. Checked values are never compared against an expected model.
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

private def textOutcome : InstructionOutcomeDefinition :=
  Program.outcome #[
    Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_STATUS statusType,
    Program.outcomeField .INSTRUCTION_OUTCOME_FIELD_VALUE textType]

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

/-- One declared Nexus history projection that witnesses a modeled Fact, keyed by the Fact's
modeled name. Only the Facts named here are lowerable, so renaming a Fact constructor rejects by
name until this declaration set names the new spelling. -/
private structure FactEvidence where
  factKey : String
  transitionId : String
  correlatedStateId : String
  attributeGroup : String
  /-- Correlation keys read as `(captured scheduled-event attribute, observed attribute)`. A `none`
  captured attribute reads the captured event's own identifier. -/
  correlations : List (Option String × String)

private def historyObservation := "history-event"
private def scheduledCapture := "scheduled-event"
private def eventIdField := "event_id"
private def pendingStateId := "pending"
private def scheduledStateId := "scheduled-correlated"
private def scheduledAttributes := "nexus_operation_scheduled_event_attributes"

/-- `correlatedStateId` names the monitor state each stage enters; the last stage of a derived
chain is the satisfied one, so the shipped success chain enters `satisfied` and the shipped Case
bytes are unchanged by this derivation. -/
private def factEvidence : List FactEvidence := [
  { factKey := "started"
    transitionId := "match-started-reference"
    correlatedStateId := "started-correlated"
    attributeGroup := "nexus_operation_started_event_attributes"
    correlations := [(none, "scheduled_event_id")] },
  { factKey := "succeeded"
    transitionId := "match-completed-event"
    correlatedStateId := "satisfied"
    attributeGroup := "nexus_operation_completed_event_attributes"
    correlations := [(some "request_id", "request_id"), (none, "scheduled_event_id")] }
]

private def capturedSide (capturedKey : Option String) : ContractExpression :=
  match capturedKey with
  | none => projected (captured scheduledCapture) (field eventIdField)
  | some name => projected (captured scheduledCapture) (historyAttribute scheduledAttributes name)

private def observedSide (group name : String) : ContractExpression :=
  projected (observed historyObservation) (historyAttribute group name)

/-- Anchor the correlation on the scheduled event the Nexus operation recorded for this Run. -/
private def anchorTransition : ContractTransitionDefinition :=
  Monitor.transition "capture-scheduled-event" pendingStateId scheduledStateId
    #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
    (ContractExpr.all #[
      ContractExpr.present (observed historyObservation),
      ContractExpr.present (projected (observed historyObservation) (field eventIdField)),
      ContractExpr.present (observedSide scheduledAttributes "request_id")])
    .CONTRACT_SUPPORT_KIND_MATCHING_EVENT
    #[Monitor.captureAssignment scheduledCapture historyObservation]

private def correlationTransition (source : String) (evidence : FactEvidence) :
    ContractTransitionDefinition :=
  let capturedPresence := evidence.correlations.filterMap fun correlation =>
    correlation.1.map fun name => ContractExpr.present (capturedSide (some name))
  let observedPresence := evidence.correlations.map fun correlation =>
    ContractExpr.present (observedSide evidence.attributeGroup correlation.2)
  let equalities := evidence.correlations.map fun correlation =>
    ContractExpr.equals (capturedSide correlation.1)
      (observedSide evidence.attributeGroup correlation.2)
  Monitor.transition evidence.transitionId source evidence.correlatedStateId
    #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
    (ContractExpr.all ([
        ContractExpr.present (captured scheduledCapture),
        ContractExpr.present (capturedSide none)] ++
      capturedPresence ++ observedPresence ++ equalities).toArray)
    .CONTRACT_SUPPORT_KIND_MATCHING_EVENT

private def correlationTransitions (source : String) :
    List FactEvidence → List ContractTransitionDefinition
  | [] => []
  | evidence :: rest =>
      correlationTransition source evidence ::
        correlationTransitions evidence.correlatedStateId rest

/-- The final correlated stage is the satisfied state; every earlier stage is still pending. -/
private def correlatedStates : List FactEvidence → List ContractStateDefinition
  | [] => []
  | [evidence] => [Monitor.state evidence.correlatedStateId .CONTRACT_STATE_STATUS_SATISFIED]
  | evidence :: rest =>
      Monitor.state evidence.correlatedStateId .CONTRACT_STATE_STATUS_NONTERMINAL ::
        correlatedStates rest

/-- Build the correlated-history rule from the Facts the selected witness records, in trace
order. -/
private def correlatedRule (ruleId : String) (chain : List FactEvidence) :
    ContractRuleDefinition :=
  Monitor.rule ruleId .CONTRACT_RULE_KIND_SAFETY pendingStateId
    (Monitor.state pendingStateId .CONTRACT_STATE_STATUS_NONTERMINAL ::
      Monitor.state scheduledStateId .CONTRACT_STATE_STATUS_NONTERMINAL ::
      correlatedStates chain).toArray
    (anchorTransition :: correlationTransitions scheduledStateId chain).toArray
    (captures := #[Monitor.capture scheduledCapture
      (Monitor.messageCapture "temporal.api.history.v1.HistoryEvent")])

private def matchesStep
    (pattern : PropertyPattern)
    (step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue) : Bool :=
  let matchesValue (value : ModelValue) : Bool :=
    pattern.reference == value.definitionId &&
      match pattern.constraint with
      | .present => true
      | .equals expected => expected == value.value
      | _ => false
  match pattern.field with
  | .selectedAction => matchesValue step.selectedAction
  | .modelOutcome => matchesValue step.modelOutcome
  | .resultingState => matchesValue step.resultingState
  | .observation => step.observations.any matchesValue
  | _ => false

/-- A clause is lowerable when the selected witness records a step that carries it, so the
correlated rule built from that witness is evidence for the clause. -/
private def clauseIsWitnessed
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
    (clause : ResolvedPropertyClause) : Bool :=
  match clause with
  | .transitionContract _ trigger response
  | .inputOutput _ trigger response =>
      steps.any fun step => matchesStep trigger step && matchesStep response step
  | .stateInvariant _ invariant => steps.any (matchesStep invariant)
  | _ => false

private def loweringError (definitionId construct : String) : LoweringError := {
  sourceDefinitionId := definitionId
  source := Authoring.source
  construct
}

private def checkedCompletion : Except LoweringError (Authoring.CheckedModel lifecycle) :=
  completion.mapError fun _ =>
    loweringError "temporal.nexus3.query.completion" "checked-completion"

/-- Lower a checked Nexus3 completion Query and its selected witness into a Case. The checked
values are carried, never compared against an expected model: a different Target, Behavior, Query,
or witness produces different Case bytes. Only a clause this Producer cannot witness rejects. -/
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
  let selectedWitness ← match witness? with
    | some selectedWitness => pure selectedWitness
    | none => throw (loweringError checkedQuery.id.value "witness.absent")
  let chain ← (selectedWitness.trace.steps.flatMap (·.observations)).mapM fun factValue =>
    match factEvidence.find? (·.factKey == factValue.value) with
    | some evidence => pure evidence
    | none => throw (loweringError factValue.definitionId.value "witness.fact-evidence")
  if chain.isEmpty then
    throw (loweringError checkedQuery.id.value "witness.fact-evidence.absent")
  for clause in checkedProperty.clauses do
    unless clauseIsWitnessed selectedWitness.trace.steps clause do
      throw (loweringError clause.id.value "property.clause-evidence")
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
    properties := [.monitor propertyBinding
      (correlatedRule (checkedProperty.id.value ++ ".correlated-history") chain)]
    contractLimits
  }

/-- The checked Nexus3 completion declaration lowered to the closed Case format. -/
def completionCase : Except LoweringError temporal.server.api.testpilot.v1.Case := do
  let checked ← checkedCompletion
  produceCompletionCase checked.target checked.property checked.behavior checked.query
    (some checked.witness)

end Temporal.Feature.Nexus3.Testpilot
