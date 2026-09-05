import Umpire.Case.Compiler

/-!
Temporal Case producers lower checked scenario choices into the generic Case compiler input. Runtime
coordinates, clients, credentials, and callback authority remain Host-owned.
-/

namespace Temporal.CaseRuntime

open Umpire
open Umpire.Case
open Umpire.Case.Compiler

def workflowServiceRole := "temporal.workflow-service"
def workerRole := "temporal.worker"
def taskQueueRole := "temporal.task-queue"
def nexusEndpointRole := "temporal.nexus-endpoint"
def getSystemInfoMethod := "/temporal.api.workflowservice.v1.WorkflowService/GetSystemInfo"
def startWorkflowMethod := "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
def getHistoryMethod := "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"

private def source : SourceLocation := {
  path := "Temporal/CaseRuntime.lean", line := 1, column := 1, provenance := "checked-model"
}

private def binding (id fingerprint : String) (kind : CaseDefinitionKind) :
    CaseDefinitionBinding :=
  { definitionId := id, behaviorFingerprint := fingerprint, kind }

private def textType : ValueType := .singular (.scalar .text)
private def historyEventType : ValueType :=
  .singular (.message "temporal.api.history.v1.HistoryEvent")
private def statusType : ValueType :=
  .singular (.enumeration "temporal.server.api.umpire.v1.InstructionOutcomeStatus")
private def statusOutcome : InstructionOutcomeSchema :=
  { fields := [{ field := .status, type := statusType }] }
private def textOutcome : InstructionOutcomeSchema :=
  { fields := [{ field := .status, type := statusType }, { field := .value, type := textType }] }
private def bounds (timeout := 5000) (emitted := 8) : InstructionBounds := {
  timeoutMilliseconds := timeout, maxAttempts := 1, maxEmittedEvents := emitted,
  maxResponseBytes := 4096
}
private def field (name : String) : FieldPath := { segments := [{ field := name }] }
private def nested (names : List String) : FieldPath :=
  { segments := names.map fun name => { field := name } }
private def historyEvents : FieldPath := {
  segments := [{ field := "history" }, { field := "events", selector := some .repeated }]
}
private def historyAttribute (selected name : String) : FieldPath := {
  segments := [{ field := "attributes", selector := some (.oneof selected) }, { field := name }]
}
private def text (value : String) : ValueExpression := .literal (.text value)
private def boolean (value : Bool) : ValueExpression := .literal (.boolean value)
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
private def project
    (source : FieldPath)
    (observationId : String)
    (cardinality := ProjectionCardinality.one) : ResponseProjection :=
  { source, cardinality, sinks := [.observation observationId] }
private def programLimits : ProgramLimits := {
  maxEntrypoints := 4, maxNodes := 16, maxEdges := 24, maxActivations := 8,
  maxAttempts := 16, maxRunEvents := 256, maxExpressionDepth := 12, maxPathFanout := 32,
  maxRequestBytes := 32768, maxResponseBytes := 4096,
  maxTotalDurationMilliseconds := 30000, maxCleanupDurationMilliseconds := 5000
}
private def contractLimits : ContractLimits := {
  maxRules := 4, maxStates := 16, maxTransitions := 16, maxExpressionDepth := 12,
  maxWorkPerEvent := 100000, maxTotalWork := 1000000000,
  maxCaptures := 4, maxCaptureBytes := 8192
}

private def getSystemInfoProperty :=
  binding "temporal.case.get-system-info.property.server-version"
    "temporal-case-get-system-info-property/v1" .property
private def getSystemInfoRule : ContractRule := {
  ruleId := "server-version-present"
  kind := .safety
  initialState := "pending"
  states := [{ stateId := "pending" }, { stateId := "satisfied", terminal := .satisfied }]
  transitions := [{
    transitionId := "observe-server-version"
    sourceState := "pending"
    targetState := "satisfied"
    eventKinds := [.instructionCompleted]
    predicate := .present (observed "server-version")
    support := .matchingEvent
  }]
}

/-- An orthogonal unary Case with an empty request and typed response projection. -/
def getSystemInfoCase : Except LoweringError Case := compile {
  version := { major := 1 }
  caseId := "temporal.case.get-system-info"
  producerId := "temporal.case.compiler"
  producerVersion := "1"
  definitions := [
    binding "temporal.workflow-service" "temporal-workflow-service/v1" .target,
    getSystemInfoProperty
  ]
  sources := [source]
  knownGaps := []
  program := {
    programId := "temporal.case.get-system-info.program"
    roles := [{ roleId := workflowServiceRole, kind := .endpoint }]
    slots := []
    observations := [{ observationId := "server-version", type := textType }]
    entrypoints := [{
      entrypointId := "controller"
      context := .controller
      activation := .controller
      nodes := [{
        instructionId := "get-system-info"
        dependencies := []
        instruction := .invokeRPC {
          endpointRoleId := workflowServiceRole
          method := getSystemInfoMethod
          requestAssignments := []
          responseProjections := [project (field "server_version") "server-version"]
        }
        outcome := statusOutcome
        bounds := bounds
      }]
    }]
    cleanup := { entrypointId := "cleanup", context := .controller, nodes := [] }
    limits := programLimits
  }
  contractId := "temporal.case.get-system-info.contract"
  properties := [.monitor getSystemInfoProperty getSystemInfoRule]
  contractLimits
}

private def asyncProperty :=
  binding "temporal.case.async-nexus.property.correlated-completion"
    "temporal-case-async-nexus-property/v1" .property
private def asyncRule : ContractRule := {
  ruleId := "correlated-nexus-completion"
  kind := .boundedLiveness
  initialState := "pending"
  states := [
    { stateId := "pending" }, { stateId := "scheduled-correlated" },
    { stateId := "started-correlated" },
    { stateId := "satisfied", terminal := .satisfied },
    { stateId := "violated", terminal := .violated }
  ]
  transitions := [
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
  horizon := some { elapsedMilliseconds := 30000, violationStateId := "violated" }
  captures := [
    { captureId := "scheduled-event", type := .message "temporal.api.history.v1.HistoryEvent" }
  ]
}

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
  outcome := statusOutcome
  bounds := bounds 10000 128
  activationReservations := reservations
}

private def historyAssignments : List RequestAssignment :=
  [
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

private def asyncProgram : Program := {
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
  observations := [
    { observationId := "history-event", type := historyEventType }
  ]
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
          outcome := statusOutcome
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
          outcome := statusOutcome
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
          outcome := statusOutcome
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
          outcome := textOutcome
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
          outcome := statusOutcome
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
        outcome := statusOutcome
        bounds := bounds
      }]
    }
  ]
  cleanup := { entrypointId := "cleanup", context := .controller, nodes := [] }
  limits := programLimits
}

/-- The checked async Nexus success slice lowered with no runtime-specific instruction. -/
def asyncNexusCase : Except LoweringError Case := compile {
  version := { major := 1 }
  caseId := "temporal.case.async-nexus-success"
  producerId := "temporal.case.compiler"
  producerVersion := "1"
  definitions := [
    binding "temporal.workflow-service" "temporal-workflow-service/v1" .target,
    binding "temporal.nexus.sdk" "temporal-nexus-sdk/v1" .provider,
    asyncProperty
  ]
  sources := [source]
  knownGaps := []
  program := asyncProgram
  contractId := "temporal.case.async-nexus-success.contract"
  properties := [.monitor asyncProperty asyncRule]
  contractLimits
}

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
    (cleanupFailure := false) : Except LoweringError Case :=
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
def conformanceSatisfiedCase : Except LoweringError Case :=
  conformanceCase "temporal.case.conformance.satisfied" .satisfied true

def conformanceViolatedCase : Except LoweringError Case :=
  conformanceCase "temporal.case.conformance.violated" .violated true

def conformanceInconclusiveCase : Except LoweringError Case :=
  conformanceCase "temporal.case.conformance.inconclusive" .satisfied false

def conformanceCleanupFailureCase : Except LoweringError Case :=
  conformanceCase "temporal.case.conformance.cleanup-failure" .violated true true

def conformanceCrossRunIsolationCase : Except LoweringError Case :=
  conformanceCase "temporal.case.conformance.cross-run-isolation" .satisfied true

def conformanceStaticRejectionCase : Except LoweringError Case :=
  (conformanceCase "temporal.case.conformance.static-rejection" .satisfied true).map fun output =>
    let invalidRules := output.contract.rules.map fun rule => { rule with initialState := "missing" }
    { output with contract := { output.contract with rules := invalidRules } }

end Temporal.CaseRuntime
