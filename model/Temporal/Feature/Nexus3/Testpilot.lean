import Temporal.Feature.Nexus3.Nexus
import Temporal.Testpilot.CaseSupport

/-!
# Nexus3 Testpilot producer

This module validates the checked completion Query and witness before lowering the Nexus-specific
Program, history correlation, and success monitor through the generic Case compiler.
-/

namespace Temporal.Feature.Nexus3.Testpilot

open Umpire
open Umpire.Case
open Umpire.Case.Compiler
open Temporal.Testpilot.CaseSupport

private def workflowServiceRole := "temporal.workflow-service"
private def workerRole := "temporal.worker"
private def taskQueueRole := "temporal.task-queue"
private def nexusEndpointRole := "temporal.nexus-endpoint"
private def startWorkflowMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
private def getHistoryMethod :=
  "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"

private def historyEventType : ValueType :=
  .singular (.message "temporal.api.history.v1.HistoryEvent")

private def textOutcome : InstructionOutcomeSchema :=
  { fields := [{ field := .status, type := statusType }, { field := .value, type := textType }] }

private def nested (names : List String) : FieldPath :=
  { segments := names.map fun name => { field := name } }

private def historyEvents : FieldPath := {
  segments := [{ field := "history" }, { field := "events", selector := some .repeated }]
}

private def historyAttribute (selected name : String) : FieldPath := {
  segments := [{ field := "attributes", selector := some (.oneof selected) }, { field := name }]
}

private def text (value : String) : ValueExpression := .literal (.text value)
private def signedInteger (value : Int) : ValueExpression := .literal (.signedInteger value)
private def observed (id : String) : ValueExpression := .observation { observationId := id }
private def captured (id : String) : ValueExpression := .capture { captureId := id }
private def runId : ValueExpression := .runEvent .runId
private def projected (value : ValueExpression) (path : FieldPath) : ValueExpression := .path value path

private def succeeded (entrypoint instruction : String) : ValueExpression :=
  let status := ValueExpression.outcome {
      instruction := { entrypointId := entrypoint, instructionId := instruction }
      field := .status
    }
  .all [.present status, .equals status (.literal (.enumValue 1))]

private def assign (target : FieldPath) (value : ValueExpression) : RequestAssignment :=
  { target, value }

private def rpc
    (id method : String)
    (dependencies : List InstructionReference)
    (assignments : List RequestAssignment)
    (projections : List ResponseProjection)
    (guard : Option ValueExpression := none)
    (reservations : List ActivationReservation := []) : InstructionNode := {
  instructionId := id
  dependencies
  guard
  instruction := .invokeRPC {
    endpointRoleId := workflowServiceRole, method,
    requestAssignments := assignments, responseProjections := projections
  }
  «outcome» := statusOutcome
  bounds := bounds 10000 128
  activationReservations := reservations
}

private def historyAssignments : List RequestAssignment := [
  assign (field "namespace") (text "default"),
  assign (nested ["execution", "workflow_id"]) runId,
  assign (field "maximum_page_size") (signedInteger 64),
  assign (field "wait_new_event") (boolean true)
]

private def historyNode : InstructionNode :=
  rpc "history" getHistoryMethod [{
    entrypointId := "controller", instructionId := "complete-nexus-operation"
  }] historyAssignments [
    project historyEvents "history-event" .emitEach
  ] (some (succeeded "controller" "complete-nexus-operation"))

private def program : Program := {
  programId := "temporal.case.async-nexus.program"
  roles := [
    { roleId := workflowServiceRole, kind := .endpoint },
    { roleId := workerRole, kind := .worker },
    { roleId := taskQueueRole, kind := .taskQueue },
    { roleId := nexusEndpointRole, kind := .endpoint }
  ]
  slots := [{
    slotId := "completion-authority", type := .singular .opaqueCapability,
    kind := .opaqueCapability
  }]
  observations := [{ observationId := "history-event", type := historyEventType }]
  entrypoints := [
    {
      entrypointId := "controller"
      context := .controller
      activation := .controller
      nodes := [
        rpc "start-workflow" startWorkflowMethod [] [
          assign (field "namespace") (text "default"),
          assign (field "workflow_id") runId,
          assign (nested ["workflow_type", "name"]) (text "umpire-async-nexus-workflow"),
          assign (nested ["task_queue", "name"]) (text "umpire-async-nexus-workflow-queue"),
          assign (field "request_id") runId
        ] [] none [
          { entrypointId := "workflow", count := 1 },
          { entrypointId := "handler", count := 1 }
        ],
        {
          instructionId := "await-completion-authority"
          dependencies := [{ entrypointId := "controller", instructionId := "start-workflow" }]
          guard := some (succeeded "controller" "start-workflow")
          instruction := .awaitSlot { slotId := "completion-authority" }
          «outcome» := statusOutcome
          bounds := bounds 10000
        },
        {
          instructionId := "complete-nexus-operation"
          dependencies := [{
            entrypointId := "controller", instructionId := "await-completion-authority"
          }]
          guard := some (succeeded "controller" "await-completion-authority")
          instruction := .completeNexusOperation {
            capabilitySlotId := "completion-authority", result := text "completed"
          }
          «outcome» := statusOutcome
          bounds := bounds 10000
        },
        historyNode
      ]
    },
    {
      entrypointId := "workflow"
      context := .workflow
      activation := .workflow {
        workflowType := "umpire-async-nexus-workflow"
        workerRoleId := workerRole
        taskQueueRoleId := taskQueueRole
      }
      nodes := [
        {
          instructionId := "start-nexus-operation"
          dependencies := []
          instruction := .startNexusOperation {
            endpointRoleId := nexusEndpointRole
            service := "umpire.case.service"
            operation := "complete"
            input := text "request"
          }
          «outcome» := statusOutcome
          bounds := bounds 10000
        },
        {
          instructionId := "await-nexus-operation"
          dependencies := [{
            entrypointId := "workflow", instructionId := "start-nexus-operation"
          }]
          instruction := .awaitOutcome {
            instruction := {
              entrypointId := "workflow", instructionId := "start-nexus-operation"
            }
          }
          «outcome» := textOutcome
          bounds := bounds 10000
        },
        {
          instructionId := "finish-workflow"
          dependencies := [{
            entrypointId := "workflow", instructionId := "await-nexus-operation"
          }]
          guard := some (succeeded "workflow" "await-nexus-operation")
          instruction := .finish {
            result := .outcome {
              instruction := {
                entrypointId := "workflow", instructionId := "await-nexus-operation"
              }
              field := .value
            }
          }
          «outcome» := statusOutcome
          bounds := bounds
        }
      ]
    },
    {
      entrypointId := "handler"
      context := .nexusHandler
      activation := .nexusHandler {
        service := "umpire.case.service", operation := "complete",
        workerRoleId := workerRole, taskQueueRoleId := taskQueueRole
      }
      nodes := [{
        instructionId := "respond-async"
        dependencies := []
        instruction := .respondNexus {
          kind := .asynchronous, result := text "accepted",
          capabilitySlotId := "completion-authority"
        }
        «outcome» := statusOutcome
        bounds := bounds
      }]
    }
  ]
  cleanup := { entrypointId := "cleanup", context := .controller, nodes := [] }
  «limits» := programLimits
}

private def successRule (checkedProperty : CheckedProperty) : ContractRule := {
  ruleId := checkedProperty.id.value ++ ".correlated-history"
  kind := .safety
  initialState := "pending"
  «states» := [
    { stateId := "pending" }, { stateId := "scheduled-correlated" },
    { stateId := "started-correlated" },
    { stateId := "satisfied", «terminal» := .satisfied }
  ]
  «transitions» := [
    {
      transitionId := "capture-scheduled-event"
      sourceState := "pending"
      targetState := "scheduled-correlated"
      eventKinds := [.instructionCompleted]
      predicate := .all [
        .present (observed "history-event"),
        .present (projected (observed "history-event") (field "event_id")),
        .present (projected (observed "history-event")
          (historyAttribute "nexus_operation_scheduled_event_attributes" "request_id"))
      ]
      support := .matchingEvent
      captureAssignments := [{
        captureId := "scheduled-event", observation := { observationId := "history-event" }
      }]
    },
    {
      transitionId := "match-started-reference"
      sourceState := "scheduled-correlated"
      targetState := "started-correlated"
      eventKinds := [.instructionCompleted]
      predicate := .all [
        .present (captured "scheduled-event"),
        .present (projected (captured "scheduled-event") (field "event_id")),
        .present (projected (observed "history-event")
          (historyAttribute "nexus_operation_started_event_attributes" "scheduled_event_id")),
        .equals
          (projected (captured "scheduled-event") (field "event_id"))
          (projected (observed "history-event")
            (historyAttribute "nexus_operation_started_event_attributes" "scheduled_event_id"))
      ]
      support := .matchingEvent
    },
    {
      transitionId := "match-completed-event"
      sourceState := "started-correlated"
      targetState := "satisfied"
      eventKinds := [.instructionCompleted]
      predicate := .all [
        .present (captured "scheduled-event"),
        .present (projected (captured "scheduled-event") (field "event_id")),
        .present (projected (captured "scheduled-event")
          (historyAttribute "nexus_operation_scheduled_event_attributes" "request_id")),
        .present (projected (observed "history-event")
          (historyAttribute "nexus_operation_completed_event_attributes" "request_id")),
        .present (projected (observed "history-event")
          (historyAttribute "nexus_operation_completed_event_attributes" "scheduled_event_id")),
        .equals
          (projected (captured "scheduled-event")
            (historyAttribute "nexus_operation_scheduled_event_attributes" "request_id"))
          (projected (observed "history-event")
            (historyAttribute "nexus_operation_completed_event_attributes" "request_id")),
        .equals
          (projected (captured "scheduled-event") (field "event_id"))
          (projected (observed "history-event")
            (historyAttribute "nexus_operation_completed_event_attributes" "scheduled_event_id"))
      ]
      support := .matchingEvent
    }
  ]
  captures := [{
    captureId := "scheduled-event", type := .message "temporal.api.history.v1.HistoryEvent"
  }]
}

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
    (witness? : Option BehaviorTrace) : Except LoweringError Case := do
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
  compile {
    version := { major := 1 }
    caseId := "temporal.case.async-nexus-success"
    producerId := "temporal.nexus3.testpilot"
    producerVersion := "1"
    definitions := [
      binding target.id.value target.behaviorFingerprint.render .target,
      binding checkedBehavior.id.value checkedBehavior.behaviorFingerprint.render .«behavior»,
      binding checkedQuery.id.value checkedQuery.behaviorFingerprint.render .«query»,
      propertyBinding
    ]
    sources := [target.source, checkedBehavior.source, checkedQuery.source, checkedProperty.source]
    knownGaps := checkedQuery.authoredKnownGaps.toCaseKnownGaps
    program
    contractId := "temporal.case.async-nexus-success.contract"
    properties := [.monitor propertyBinding (successRule checkedProperty)]
    contractLimits
  }

/-- The checked Nexus3 completion declaration lowered to the closed Case format. -/
def completionCase : Except LoweringError Case := do
  let checked ← checkedCompletion
  produceCompletionCase checked.target checked.property checked.behavior checked.query
    (some checked.witness)

end Temporal.Feature.Nexus3.Testpilot
