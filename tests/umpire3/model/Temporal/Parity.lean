import Lean.Data.Json
import Temporal.Inventory
import Temporal.Monitors
import Temporal.Product.TaskAck
import Temporal.Refinement.TaskAck
import Temporal.Refinement.MigratedFamilies
import Temporal.System.MigratedFamilies
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
  ledger.entries.length = 20 && ledger.entries.all entryMetadataValid &&
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
  owner := "Temporal.Product.NexusClosure"
  evidence := {
    proof := resolved_declaration% Umpire3.Temporal.Product.NexusClosure.closureSafe
    executable := resolved_declaration% Umpire3.Temporal.Product.NexusClosure.bounded
    monitor := resolved_declaration% Umpire3.Temporal.Monitors.nexusOperationClosure_monitor_equivalent
    negativeControl := resolved_declaration% Umpire3.Temporal.Product.NexusClosure.unsafeClosureMutation
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
      "Temporal.Product.SpeculativeTask"
      (resolved_declaration% Umpire3.Temporal.Product.SpeculativeTask.speculativeCreationSafe)
      (resolved_declaration% Umpire3.Temporal.Product.SpeculativeTask.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.speculativeTask_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Product.SpeculativeTask.orphanedTaskMutationNegativeControl),
    nexusClosureProperty,
    exactProperty "NexusActivityLinkConsistency" "nexus-activity.link-consistency"
      "Temporal.Product.NexusActivityLink"
      (resolved_declaration% Umpire3.Temporal.Product.NexusActivityLink.linkConsistencySafe)
      (resolved_declaration% Umpire3.Temporal.Product.NexusActivityLink.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.nexusActivityLink_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Product.NexusActivityLink.missingReverseMutationNegativeControl),
    exactProperty "NexusOperationTimeoutSemantics" "nexus-operation.timeout-semantics"
      "Temporal.Product.NexusTimeout"
      (resolved_declaration% Umpire3.Temporal.Product.NexusTimeout.timeoutSemanticsSafe)
      (resolved_declaration% Umpire3.Temporal.Product.NexusTimeout.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.nexusTimeout_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Product.NexusTimeout.timeoutMetadataMutationNegativeControl),
    exactProperty "CallbackReferenceConsistency" "callback.reference-consistency"
      "Temporal.Product.CallbackReference"
      (resolved_declaration% Umpire3.Temporal.Product.CallbackReference.referenceConsistencySafe)
      (resolved_declaration% Umpire3.Temporal.Product.CallbackReference.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.callbackReference_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Product.CallbackReference.wrongReferenceMutationNegativeControl),
    exactProperty "CallbackResponseConsistency" "callback.response-consistency"
      "Temporal.Product.CallbackResponse"
      (resolved_declaration% Umpire3.Temporal.Product.CallbackResponse.responseConsistencySafe)
      (resolved_declaration% Umpire3.Temporal.Product.CallbackResponse.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.callbackResponse_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Product.CallbackResponse.conflictingResponseMutationNegativeControl),
    exactProperty "WorkflowTaskStarvation" "workflow-task.starvation"
      "Temporal.Product.WorkflowProgress"
      (resolved_declaration% Umpire3.Temporal.Product.WorkflowProgress.workflowTaskStarvationSafe)
      (resolved_declaration% Umpire3.Temporal.Product.WorkflowProgress.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.workflowTaskStarvation_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Product.WorkflowProgress.starvationMutationNegativeControl),
    exactProperty "EntityProgress" "entity.progress"
      "Temporal.Product.WorkflowProgress"
      (resolved_declaration% Umpire3.Temporal.Product.WorkflowProgress.entityProgressSafe)
      (resolved_declaration% Umpire3.Temporal.Product.WorkflowProgress.bounded)
      (resolved_declaration% Umpire3.Temporal.Monitors.entityProgress_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Product.WorkflowProgress.progressMutationNegativeControl),
    exactTarget "feature-nexus" "feature-nexus"
      "Temporal.Refinement.MigratedFamilies.NexusClosure"
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.NexusClosure.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.MigratedFamilies.NexusClosure.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.nexusOperationClosure_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.NexusClosure.mutationBreaksDeclaredSimulation),
    exactTarget "feature-workflow-speculative-delivery" "feature-workflow-speculative-delivery"
      "Temporal.Refinement.MigratedFamilies.SpeculativeTask"
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.SpeculativeTask.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.MigratedFamilies.SpeculativeTask.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.speculativeTask_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.SpeculativeTask.mutationBreaksDeclaredSimulation),
    taskAckTarget,
    exactTarget "foundation-delivery-safety" "foundation-delivery-safety"
      "Temporal.Refinement.MigratedFamilies.WorkflowProgress"
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.WorkflowProgress.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.MigratedFamilies.WorkflowProgress.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.entityProgress_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.WorkflowProgress.entityMutationBreaksDeclaredSimulation),
    exactTarget "foundation-ownership-fencing" "foundation-ownership-fencing"
      "Temporal.Refinement.MigratedFamilies.WorkflowOwnership"
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.WorkflowOwnership.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.MigratedFamilies.WorkflowOwnership.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.workflowOwnership_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.WorkflowOwnership.mutationBreaksDeclaredSimulation),
    exactTarget "foundation-routing-isolation" "foundation-routing-isolation"
      "Temporal.Refinement.MigratedFamilies.RoutingIsolation"
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.RoutingIsolation.soundSimulations)
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.RoutingIsolation.executableViews)
      (resolved_declaration% Umpire3.Temporal.Monitors.workflowRouting_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.RoutingIsolation.mutationsBreakDeclaredSimulations),
    exactTarget "integration-activity-delivery" "integration-activity-delivery"
      "Temporal.Refinement.MigratedFamilies.WorkflowProgress"
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.WorkflowProgress.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.MigratedFamilies.WorkflowProgress.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.entityProgress_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.WorkflowProgress.entityMutationBreaksDeclaredSimulation),
    exactTarget "integration-callback-nexus" "integration-callback-nexus"
      "Temporal.Refinement.MigratedFamilies.CallbackReference"
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.CallbackReference.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.MigratedFamilies.CallbackReference.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.callbackReference_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.CallbackReference.mutationBreaksDeclaredSimulation),
    exactTarget "integration-callback-workflow" "integration-callback-workflow"
      "Temporal.Refinement.MigratedFamilies.CallbackResponse"
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.CallbackResponse.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.MigratedFamilies.CallbackResponse.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.callbackResponse_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.CallbackResponse.mutationBreaksDeclaredSimulation),
    exactTarget "integration-nexus-activity" "integration-nexus-activity"
      "Temporal.Refinement.MigratedFamilies.NexusActivityLink"
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.NexusActivityLink.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.MigratedFamilies.NexusActivityLink.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.nexusActivityLink_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.NexusActivityLink.mutationBreaksDeclaredSimulation),
    exactTarget "integration-workflow-delivery" "integration-workflow-delivery"
      "Temporal.Refinement.MigratedFamilies.WorkflowProgress"
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.WorkflowProgress.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.MigratedFamilies.WorkflowProgress.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.workflowTaskStarvation_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.WorkflowProgress.starvationMutationBreaksDeclaredSimulation),
    exactTarget "protocol-atomic" "protocol-atomic"
      "Temporal.Refinement.MigratedFamilies.CallbackResponse"
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.CallbackResponse.soundSimulation)
      (resolved_declaration% Umpire3.Temporal.System.MigratedFamilies.CallbackResponse.executable)
      (resolved_declaration% Umpire3.Temporal.Monitors.callbackResponse_monitor_equivalent)
      (resolved_declaration% Umpire3.Temporal.Refinement.MigratedFamilies.CallbackResponse.mutationBreaksDeclaredSimulation),
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
