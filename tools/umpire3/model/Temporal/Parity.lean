import Lean.Data.Json
import Temporal.Inventory
import Temporal.Monitors
import Temporal.Families.TaskAcknowledgement.Feature
import Temporal.Families.TaskAcknowledgement.Refinement
import Temporal.Families
import Temporal.Families.NexusProgress.Refinement
import Temporal.Families
import Temporal.Families.NexusProgress.System
import Umpire3.Registration

namespace Umpire3.Temporal.Parity

structure Evidence where
  proof : ResolvedDeclaration
  executable : ResolvedDeclaration
  monitor : ResolvedDeclaration
  negativeControl : ResolvedDeclaration
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
      entry.evidence.proof.name != "" && entry.evidence.executable.name != "" &&
        entry.evidence.monitor.name != "" && entry.evidence.negativeControl.name != ""
    else
      (entry.fidelity = "partial" || entry.fidelity = "inventory-only") &&
        entry.evidenceStatus = "metadata-missing"

def Ledger.metadataValid (ledger : Ledger) : Bool :=
  ledger.entries.length = 22 && ledger.entries.all entryMetadataValid &&
    (ledger.entries.map (fun entry => entry.category ++ ":" ++ entry.legacyName)).eraseDups.length =
      ledger.entries.length

def Ledger.MetadataValid (ledger : Ledger) : Prop := ledger.metadataValid = true

private def target (legacyName semanticIdentifier owner : String) (proof executable monitor
    negativeControl : ResolvedDeclaration) : Entry := {
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
  owner := "Temporal.Refinement.TaskAck"
  evidence := {
    proof := resolved_declaration% Umpire3.Temporal.Refinement.TaskAck.soundSimulations
    executable := resolved_declaration% Umpire3.Temporal.System.TaskAck.executableViews
    monitor := resolved_declaration% Umpire3.Temporal.Monitors.taskAcknowledgement_monitor_equivalent
    negativeControl := resolved_declaration% Umpire3.Temporal.Refinement.TaskAck.nonVacuity
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
  owner := "Temporal.Feature.NexusClosure"
  evidence := {
    proof := resolved_declaration% Umpire3.Temporal.Feature.NexusClosure.closureSafe
    executable := resolved_declaration% Umpire3.Temporal.Feature.NexusClosure.bounded
    monitor := resolved_declaration% Umpire3.Temporal.Monitors.nexusOperationClosure_monitor_equivalent
    negativeControl := resolved_declaration% Umpire3.Temporal.Feature.NexusClosure.unsafeClosureMutation
  }
}

private def exactProperty (legacyName semanticIdentifier owner : String) (proof executable monitor
    negativeControl : ResolvedDeclaration) : Entry := {
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

private def exactTarget (legacyName semanticIdentifier owner : String) (proof executable monitor
    negativeControl : ResolvedDeclaration) : Entry := {
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
      "Temporal.Feature.SpeculativeTask"
      (resolved_declaration% Umpire3.Temporal.Feature.SpeculativeTask.speculativeCreationSafe)
      (resolved_declaration% Umpire3.Temporal.Feature.SpeculativeTask.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.speculativeTask_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Feature.SpeculativeTask.orphanedTaskMutationNegativeControl),
    nexusClosureProperty,
    exactProperty "NexusOperationProgress" "nexus-operation.progress"
      "Temporal.Feature.NexusProgress"
      (resolved_declaration% Umpire3.Temporal.Feature.NexusProgress.progressSafe)
      (resolved_declaration% Umpire3.Temporal.Feature.NexusProgress.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.nexusOperationProgress_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Feature.NexusProgress.stuckAfterDeadlineMutationNegativeControl),
    exactProperty "NexusActivityLinkConsistency" "nexus-activity.link-consistency"
      "Temporal.Feature.NexusActivityLink"
      (resolved_declaration% Umpire3.Temporal.Feature.NexusActivityLink.linkConsistencySafe)
      (resolved_declaration% Umpire3.Temporal.Feature.NexusActivityLink.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.nexusActivityLink_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Feature.NexusActivityLink.missingReverseMutationNegativeControl),
    exactProperty "NexusOperationTimeoutSemantics" "nexus-operation.timeout-semantics"
      "Temporal.Feature.NexusTimeout"
      (resolved_declaration% Umpire3.Temporal.Feature.NexusTimeout.timeoutSemanticsSafe)
      (resolved_declaration% Umpire3.Temporal.Feature.NexusTimeout.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.nexusTimeout_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Feature.NexusTimeout.timeoutMetadataMutationNegativeControl),
    exactProperty "CallbackReferenceConsistency" "callback.reference-consistency"
      "Temporal.Feature.CallbackReference"
      (resolved_declaration% Umpire3.Temporal.Feature.CallbackReference.referenceConsistencySafe)
      (resolved_declaration% Umpire3.Temporal.Feature.CallbackReference.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.callbackReference_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Feature.CallbackReference.wrongReferenceMutationNegativeControl),
    exactProperty "CallbackResponseConsistency" "callback.response-consistency"
      "Temporal.Feature.CallbackResponse"
      (resolved_declaration% Umpire3.Temporal.Feature.CallbackResponse.responseConsistencySafe)
      (resolved_declaration% Umpire3.Temporal.Feature.CallbackResponse.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.callbackResponse_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Feature.CallbackResponse.conflictingResponseMutationNegativeControl),
    exactProperty "WorkflowTaskStarvation" "workflow-task.starvation"
      "Temporal.Feature.WorkflowProgress"
      (resolved_declaration% Umpire3.Temporal.Feature.WorkflowProgress.workflowTaskStarvationSafe)
      (resolved_declaration% Umpire3.Temporal.Feature.WorkflowProgress.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.workflowTaskStarvation_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Feature.WorkflowProgress.starvationMutationNegativeControl),
    exactProperty "EntityProgress" "entity.progress"
      "Temporal.Feature.WorkflowProgress"
      (resolved_declaration% Umpire3.Temporal.Feature.WorkflowProgress.entityProgressSafe)
      (resolved_declaration% Umpire3.Temporal.Feature.WorkflowProgress.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.entityProgress_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Feature.WorkflowProgress.progressMutationNegativeControl),
    exactTarget "feature-nexus" "feature-nexus"
      "Temporal.Refinement.NexusClosure"
      (resolved_declaration% Umpire3.Temporal.Refinement.NexusClosure.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.NexusClosure.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.nexusOperationClosure_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.NexusClosure.mutationBreaksDeclaredSimulation),
    exactTarget "feature-nexus-progress" "feature-nexus-progress"
      "Temporal.Refinement.NexusProgress"
      (resolved_declaration% Umpire3.Temporal.Refinement.NexusProgress.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.NexusProgress.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.nexusOperationProgress_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.NexusProgress.mutationBreaksDeclaredSimulation),
    exactTarget "feature-workflow-speculative-delivery" "feature-workflow-speculative-delivery"
      "Temporal.Refinement.SpeculativeTask"
      (resolved_declaration% Umpire3.Temporal.Refinement.SpeculativeTask.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.SpeculativeTask.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.speculativeTask_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.SpeculativeTask.mutationBreaksDeclaredSimulation),
    taskAckTarget,
    exactTarget "foundation-delivery-safety" "foundation-delivery-safety"
      "Temporal.Refinement.WorkflowProgress"
      (resolved_declaration% Umpire3.Temporal.Refinement.WorkflowProgress.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.WorkflowProgress.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.entityProgress_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.WorkflowProgress.entityMutationBreaksDeclaredSimulation),
    exactTarget "foundation-ownership-fencing" "foundation-ownership-fencing"
      "Temporal.Refinement.WorkflowOwnership"
      (resolved_declaration% Umpire3.Temporal.Refinement.WorkflowOwnership.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.WorkflowOwnership.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.workflowOwnership_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.WorkflowOwnership.mutationBreaksDeclaredSimulation),
    exactTarget "foundation-routing-isolation" "foundation-routing-isolation"
      "Temporal.Refinement.RoutingIsolation"
      (resolved_declaration% Umpire3.Temporal.Refinement.RoutingIsolation.soundSimulations)
      (resolved_declaration% Umpire3.Temporal.System.RoutingIsolation.executableViews)
      (resolved_declaration% Umpire3.Temporal.Monitors.workflowRouting_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.RoutingIsolation.mutationsBreakDeclaredSimulations),
    exactTarget "integration-activity-delivery" "integration-activity-delivery"
      "Temporal.Refinement.WorkflowProgress"
      (resolved_declaration% Umpire3.Temporal.Refinement.WorkflowProgress.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.WorkflowProgress.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.entityProgress_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.WorkflowProgress.entityMutationBreaksDeclaredSimulation),
    exactTarget "integration-callback-nexus" "integration-callback-nexus"
      "Temporal.Refinement.CallbackReference"
      (resolved_declaration% Umpire3.Temporal.Refinement.CallbackReference.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.CallbackReference.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.callbackReference_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.CallbackReference.mutationBreaksDeclaredSimulation),
    exactTarget "integration-callback-workflow" "integration-callback-workflow"
      "Temporal.Refinement.CallbackResponse"
      (resolved_declaration% Umpire3.Temporal.Refinement.CallbackResponse.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.CallbackResponse.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.callbackResponse_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.CallbackResponse.mutationBreaksDeclaredSimulation),
    exactTarget "integration-nexus-activity" "integration-nexus-activity"
      "Temporal.Refinement.NexusActivityLink"
      (resolved_declaration% Umpire3.Temporal.Refinement.NexusActivityLink.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.NexusActivityLink.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.nexusActivityLink_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.NexusActivityLink.mutationBreaksDeclaredSimulation),
    exactTarget "integration-workflow-delivery" "integration-workflow-delivery"
      "Temporal.Refinement.WorkflowProgress"
      (resolved_declaration% Umpire3.Temporal.Refinement.WorkflowProgress.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.WorkflowProgress.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.workflowTaskStarvation_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.WorkflowProgress.starvationMutationBreaksDeclaredSimulation),
    exactTarget "protocol-atomic" "protocol-atomic"
      "Temporal.Refinement.CallbackResponse"
      (resolved_declaration% Umpire3.Temporal.Refinement.CallbackResponse.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.CallbackResponse.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.callbackResponse_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.CallbackResponse.mutationBreaksDeclaredSimulation),
  ]

theorem ledgerMetadataValid : ledger.MetadataValid := by
  change ledger.metadataValid = true
  decide

private def Evidence.toJson (evidence : Evidence) : Lean.Json := Lean.Json.mkObj [
  ("proof", evidence.proof.toJson),
  ("executable", evidence.executable.toJson),
  ("monitor", evidence.monitor.toJson),
  ("negativeControl", evidence.negativeControl.toJson),
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

private def allEvidence : List ResolvedDeclaration :=
  ledger.entries.flatMap fun entry => [entry.evidence.proof, entry.evidence.executable,
    entry.evidence.monitor, entry.evidence.negativeControl]

def json (semanticHash dependencyHash catalogHash : String) : String :=
  (Lean.Json.mkObj [
    ("formatVersion", "umpire3/parity-ledger/v4"),
    ("resultClass", "evidence-resolved"),
    ("trustBadge", if allEvidence.all (·.axioms.isEmpty) then
      "kernel" else "kernel-with-declared-axioms"),
    ("semanticHash", semanticHash),
    ("sourceDigest", semanticHash),
    ("dependencyDigest", dependencyHash),
    ("artifactDigest", "derived"),
    ("catalogHash", catalogHash),
    ("entries", Lean.Json.arr (ledger.entries.map Entry.toJson).toArray),
  ]).compress

end Umpire3.Temporal.Parity
