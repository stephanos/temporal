import Lean.Data.Json

namespace Umpire3.Temporal.Parity

structure Evidence where
  proof : String
  executable : String
  monitor : String
  negativeControl : String
  deriving DecidableEq, Repr

structure Entry where
  category : String
  legacyName : String
  semanticIdentifier : String
  disposition : String
  fidelity : String
  evidenceLevel : String
  evidenceStatus : String
  owner : String
  evidence : Evidence
  deriving DecidableEq, Repr

structure Ledger where
  entries : List Entry
  deriving DecidableEq, Repr

private def entryMetadataValid (entry : Entry) : Bool :=
  entry.category != "" && entry.legacyName != "" && entry.semanticIdentifier != "" &&
    entry.owner != "" &&
    (entry.evidenceStatus = "metadata-present" || entry.evidenceStatus = "metadata-missing") &&
    (entry.disposition = "equivalent" || entry.disposition = "replaced" ||
      entry.disposition = "intentionally-unsupported" || entry.disposition = "not-yet-implemented") &&
    (entry.fidelity = "exact" || entry.fidelity = "semantic-equivalent" ||
      entry.fidelity = "partial" || entry.fidelity = "inventory-only") &&
    (entry.evidenceLevel = "inventory" || entry.evidenceLevel = "model-proof" ||
      entry.evidenceLevel = "local-integration" || entry.evidenceLevel = "profile-qualified") &&
    if entry.disposition = "equivalent" || entry.disposition = "replaced" then
      (entry.fidelity = "exact" || entry.fidelity = "semantic-equivalent") &&
      entry.evidenceLevel != "inventory" && entry.evidenceStatus = "metadata-present" &&
      entry.evidence.proof != "" && entry.evidence.executable != "" &&
        entry.evidence.monitor != "" && entry.evidence.negativeControl != ""
    else
      (entry.fidelity = "partial" || entry.fidelity = "inventory-only") &&
        entry.evidenceStatus = "metadata-missing"

def Ledger.metadataValid (ledger : Ledger) : Bool :=
  ledger.entries.length = 20 && ledger.entries.all entryMetadataValid &&
    (ledger.entries.map (fun entry => entry.category ++ ":" ++ entry.legacyName)).eraseDups.length =
      ledger.entries.length

def Ledger.MetadataValid (ledger : Ledger) : Prop := ledger.metadataValid = true

private def target (legacyName semanticIdentifier owner proof executable monitor
    negativeControl : String) : Entry := {
  category := "target"
  legacyName
  semanticIdentifier
  disposition := "not-yet-implemented"
  fidelity := "partial"
  evidenceLevel := "model-proof"
  evidenceStatus := "metadata-missing"
  owner
  evidence := {
    proof
    executable
    monitor
    negativeControl
  }
}

private def taskAckTarget : Entry := {
  category := "target"
  legacyName := "foundation-backlog-ack"
  semanticIdentifier := "foundation-backlog-ack"
  disposition := "equivalent"
  fidelity := "exact"
  evidenceLevel := "local-integration"
  evidenceStatus := "metadata-present"
  owner := "Temporal.Product.TaskAck"
  evidence := {
    proof := "Umpire3.Temporal.Product.TaskAck.acknowledged_removes_backlog"
    executable := "Umpire3.Temporal.Product.TaskAck.bounded"
    monitor := "monitor.task-delivery.acknowledged-removes-backlog"
    negativeControl := "Umpire3.Temporal.Product.TaskAck.acknowledgementMutationNegativeControl"
  }
}

private def nexusClosureProperty : Entry := {
  category := "property"
  legacyName := "NexusOperationClosure"
  semanticIdentifier := "nexus-operation.closure"
  disposition := "equivalent"
  fidelity := "exact"
  evidenceLevel := "local-integration"
  evidenceStatus := "metadata-present"
  owner := "Temporal.Product.NexusClosure"
  evidence := {
    proof := "Umpire3.Temporal.Product.NexusClosure.closureSafe"
    executable := "Umpire3.Temporal.Product.NexusClosure.bounded"
    monitor := "Umpire3.Temporal.Monitors.nexusOperationClosure_monitor_equivalent"
    negativeControl := "Umpire3.Temporal.Product.NexusClosure.unsafeClosureMutation"
  }
}

private def exactProperty (legacyName semanticIdentifier owner proof executable monitor
    negativeControl : String) : Entry := {
  category := "property"
  legacyName
  semanticIdentifier
  disposition := "equivalent"
  fidelity := "exact"
  evidenceLevel := "local-integration"
  evidenceStatus := "metadata-present"
  owner
  evidence := { proof, executable, monitor, negativeControl }
}

private def exactTarget (legacyName semanticIdentifier owner proof executable monitor
    negativeControl : String) : Entry := {
  category := "target"
  legacyName
  semanticIdentifier
  disposition := "equivalent"
  fidelity := "exact"
  evidenceLevel := "local-integration"
  evidenceStatus := "metadata-present"
  owner
  evidence := { proof, executable, monitor, negativeControl }
}

def ledger : Ledger where
  entries := [
    exactProperty "SpeculativeTaskCreation" "workflow-task.speculative-creation"
      "Temporal.Product.SpeculativeTask"
      "Umpire3.Temporal.Product.SpeculativeTask.speculativeCreationSafe"
      "Umpire3.Temporal.Product.SpeculativeTask.bounded"
      "Umpire3.Temporal.Monitors.speculativeTask_monitor_equivalent"
      "Umpire3.Temporal.Product.SpeculativeTask.orphanedTaskMutationNegativeControl",
    nexusClosureProperty,
    exactProperty "NexusActivityLinkConsistency" "nexus-activity.link-consistency"
      "Temporal.Product.NexusActivityLink"
      "Umpire3.Temporal.Product.NexusActivityLink.linkConsistencySafe"
      "Umpire3.Temporal.Product.NexusActivityLink.bounded"
      "Umpire3.Temporal.Monitors.nexusActivityLink_monitor_equivalent"
      "Umpire3.Temporal.Product.NexusActivityLink.missingReverseMutationNegativeControl",
    exactProperty "NexusOperationTimeoutSemantics" "nexus-operation.timeout-semantics"
      "Temporal.Product.NexusTimeout"
      "Umpire3.Temporal.Product.NexusTimeout.timeoutSemanticsSafe"
      "Umpire3.Temporal.Product.NexusTimeout.bounded"
      "Umpire3.Temporal.Monitors.nexusTimeout_monitor_equivalent"
      "Umpire3.Temporal.Product.NexusTimeout.timeoutMetadataMutationNegativeControl",
    exactProperty "CallbackReferenceConsistency" "callback.reference-consistency"
      "Temporal.Product.CallbackReference"
      "Umpire3.Temporal.Product.CallbackReference.referenceConsistencySafe"
      "Umpire3.Temporal.Product.CallbackReference.bounded"
      "Umpire3.Temporal.Monitors.callbackReference_monitor_equivalent"
      "Umpire3.Temporal.Product.CallbackReference.wrongReferenceMutationNegativeControl",
    exactProperty "CallbackResponseConsistency" "callback.response-consistency"
      "Temporal.Product.CallbackResponse"
      "Umpire3.Temporal.Product.CallbackResponse.responseConsistencySafe"
      "Umpire3.Temporal.Product.CallbackResponse.bounded"
      "Umpire3.Temporal.Monitors.callbackResponse_monitor_equivalent"
      "Umpire3.Temporal.Product.CallbackResponse.conflictingResponseMutationNegativeControl",
    exactProperty "WorkflowTaskStarvation" "workflow-task.starvation"
      "Temporal.Product.WorkflowProgress"
      "Umpire3.Temporal.Product.WorkflowProgress.workflowTaskStarvationSafe"
      "Umpire3.Temporal.Product.WorkflowProgress.bounded"
      "Umpire3.Temporal.Monitors.workflowTaskStarvation_monitor_equivalent"
      "Umpire3.Temporal.Product.WorkflowProgress.starvationMutationNegativeControl",
    exactProperty "EntityProgress" "entity.progress"
      "Temporal.Product.WorkflowProgress"
      "Umpire3.Temporal.Product.WorkflowProgress.entityProgressSafe"
      "Umpire3.Temporal.Product.WorkflowProgress.bounded"
      "Umpire3.Temporal.Monitors.entityProgress_monitor_equivalent"
      "Umpire3.Temporal.Product.WorkflowProgress.progressMutationNegativeControl",
    target "feature-nexus" "feature-nexus" "Temporal.Product.NexusClosure"
      "Umpire3.Temporal.Product.Nexus.cancellation_won_excludes_success"
      "Umpire3.Temporal.Product.NexusClosure.bounded"
      "monitor.nexus-operation.closure"
      "Umpire3.Temporal.Product.NexusClosure.unsafeClosureMutation",
    exactTarget "feature-workflow-speculative-delivery" "feature-workflow-speculative-delivery"
      "Temporal.Refinement.SpeculativeTask"
      "Umpire3.Temporal.System.SpeculativeTask.speculativeTaskRefinesProduct"
      "Umpire3.Temporal.System.SpeculativeTask.bounded"
      "Umpire3.Temporal.Monitors.speculativeTask_monitor_equivalent"
      "Umpire3.Temporal.Product.SpeculativeTask.orphanedTaskMutationNegativeControl",
    taskAckTarget,
    exactTarget "foundation-delivery-safety" "foundation-delivery-safety"
      "Temporal.Refinement.WorkflowProgress"
      "Umpire3.Temporal.System.WorkflowProgress.workflowProgressRefinesProduct"
      "Umpire3.Temporal.System.WorkflowProgress.bounded"
      "Umpire3.Temporal.Monitors.entityProgress_monitor_equivalent"
      "Umpire3.Temporal.Product.WorkflowProgress.progressMutationNegativeControl",
    exactTarget "foundation-ownership-fencing" "foundation-ownership-fencing"
      "Temporal.Refinement.WorkflowOwnership"
      "Umpire3.Temporal.System.WorkflowOwnership.workflowOwnershipRefinesProduct"
      "Umpire3.Temporal.System.WorkflowOwnership.bounded"
      "Umpire3.Temporal.Monitors.workflowOwnership_monitor_equivalent"
      "Umpire3.Temporal.Product.WorkflowOwnership.staleCompletionMutationNegativeControl",
    target "foundation-routing-isolation" "foundation-routing-isolation"
      "Temporal.Product.WorkflowRouting"
      "Umpire3.Temporal.Product.WorkflowRouting.routingIsolationSafe"
      "Umpire3.Temporal.Product.WorkflowRouting.bounded"
      "Umpire3.Temporal.Monitors.workflowRouting_monitor_equivalent"
      "Umpire3.Temporal.Product.WorkflowRouting.crossingRouteMutationNegativeControl",
    target "integration-activity-delivery" "integration-activity-delivery"
      "Temporal.Product.WorkflowProgress"
      "Umpire3.Temporal.Product.WorkflowProgress.entityProgressSafe"
      "Umpire3.Temporal.Product.WorkflowProgress.bounded"
      "Umpire3.Temporal.Monitors.entityProgress_monitor_equivalent"
      "Umpire3.Temporal.Product.WorkflowProgress.progressMutationNegativeControl",
    exactTarget "integration-callback-nexus" "integration-callback-nexus"
      "Temporal.Refinement.CallbackReference"
      "Umpire3.Temporal.System.CallbackReference.callbackReferenceRefinesProduct"
      "Umpire3.Temporal.System.CallbackReference.bounded"
      "Umpire3.Temporal.Monitors.callbackReference_monitor_equivalent"
      "Umpire3.Temporal.Product.CallbackReference.wrongReferenceMutationNegativeControl",
    exactTarget "integration-callback-workflow" "integration-callback-workflow"
      "Temporal.Refinement.CallbackResponse"
      "Umpire3.Temporal.System.CallbackResponse.callbackResponseRefinesProduct"
      "Umpire3.Temporal.System.CallbackResponse.bounded"
      "Umpire3.Temporal.Monitors.callbackResponse_monitor_equivalent"
      "Umpire3.Temporal.Product.CallbackResponse.conflictingResponseMutationNegativeControl",
    exactTarget "integration-nexus-activity" "integration-nexus-activity"
      "Temporal.Refinement.NexusActivityLink"
      "Umpire3.Temporal.System.NexusActivityLink.nexusActivityLinkRefinesProduct"
      "Umpire3.Temporal.System.NexusActivityLink.bounded"
      "Umpire3.Temporal.Monitors.nexusActivityLink_monitor_equivalent"
      "Umpire3.Temporal.Product.NexusActivityLink.missingReverseMutationNegativeControl",
    exactTarget "integration-workflow-delivery" "integration-workflow-delivery"
      "Temporal.Refinement.WorkflowProgress"
      "Umpire3.Temporal.System.WorkflowProgress.workflowProgressRefinesProduct"
      "Umpire3.Temporal.System.WorkflowProgress.bounded"
      "Umpire3.Temporal.Monitors.workflowTaskStarvation_monitor_equivalent"
      "Umpire3.Temporal.Product.WorkflowProgress.starvationMutationNegativeControl",
    target "protocol-atomic" "protocol-atomic" "Temporal.Product.CallbackResponse"
      "Umpire3.Temporal.Product.CallbackResponse.responseConsistencySafe"
      "Umpire3.Temporal.Product.CallbackResponse.bounded"
      "Umpire3.Temporal.Monitors.callbackResponse_monitor_equivalent"
      "Umpire3.Temporal.Product.CallbackResponse.conflictingResponseMutationNegativeControl",
  ]

theorem ledgerMetadataValid : ledger.MetadataValid := by
  change ledger.metadataValid = true
  decide

private def Evidence.toJson (evidence : Evidence) : Lean.Json := Lean.Json.mkObj [
  ("proof", evidence.proof),
  ("executable", evidence.executable),
  ("monitor", evidence.monitor),
  ("negativeControl", evidence.negativeControl),
]

private def Entry.toJson (entry : Entry) : Lean.Json := Lean.Json.mkObj [
  ("category", entry.category),
  ("legacyName", entry.legacyName),
  ("semanticIdentifier", entry.semanticIdentifier),
  ("disposition", entry.disposition),
  ("fidelity", entry.fidelity),
  ("evidenceLevel", entry.evidenceLevel),
  ("evidenceStatus", entry.evidenceStatus),
  ("owner", entry.owner),
  ("evidence", entry.evidence.toJson),
]

def json (semanticHash catalogHash : String) : String :=
  (Lean.Json.mkObj [
    ("formatVersion", "umpire3/parity-ledger/v3"),
    ("resultClass", "metadata-validated"),
    ("trustBadge", "kernel"),
    ("semanticHash", semanticHash),
    ("catalogHash", catalogHash),
    ("entries", Lean.Json.arr (ledger.entries.map Entry.toJson).toArray),
  ]).compress

end Umpire3.Temporal.Parity
