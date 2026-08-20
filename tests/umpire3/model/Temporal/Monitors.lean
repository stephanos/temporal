import Temporal.Product.NexusClosure
import Temporal.Product.NexusActivityLink
import Temporal.Product.NexusTimeout
import Temporal.Product.CallbackReference
import Temporal.Product.CallbackResponse
import Temporal.Product.WorkflowLineage
import Temporal.Product.WorkflowRouting
import Temporal.Product.WorkflowOwnership
import Temporal.Product.SpeculativeTask
import Temporal.Product.WorkflowProgress
import Umpire3.Monitor

namespace Umpire3.Temporal.Monitors

def nexusCancellation : MonitorDeclaration where
  identifier := "monitor.nexus.cancellation.won-excludes-success"
  property := "nexus.cancellation.won-excludes-success"
  evidence := ["causal", "identity-lineage"]
  coverage := ["observation.cancellation-won", "observation.stale-success-absent"]
  expression := .all [
    .observation "cancellation-won" true,
    .observation "stale-success-absent" true,
  ]

def updateCompletion : MonitorDeclaration where
  identifier := "monitor.workflow-update.accepted-completes-through-history"
  property := "workflow-update.accepted-completes-through-history"
  evidence := ["source-sequence", "identity-lineage"]
  coverage := ["observation.update-accepted", "observation.update-completed"]
  expression := .all [
    .observation "update-accepted" true,
    .observation "update-completed" true,
  ]

def taskAcknowledgement : MonitorDeclaration where
  identifier := "monitor.task-delivery.acknowledged-removes-backlog"
  property := "task-delivery.acknowledged-removes-backlog"
  evidence := ["source-sequence", "identity-lineage"]
  coverage := ["observation.workflow-task-acknowledged"]
  expression := .observation "workflow-task-acknowledged" true

private def assuranceMonitor (identifier property observation : String) : MonitorDeclaration := {
  identifier
  property
  evidence := ["causal", "identity-lineage"]
  coverage := ["observation." ++ observation]
  expression := .observation observation true
}

def speculativeTaskCreation := assuranceMonitor
  "monitor.workflow-task.speculative-creation"
  "workflow-task.speculative-creation" "speculative-task-valid"

def speculativeTaskObservations
    (state : Umpire3.Temporal.Product.SpeculativeTask.State) : List NormalizedObservation := [{
  identifier := "speculative-task-valid"
  value := Umpire3.Temporal.Product.SpeculativeTask.speculativeReadyB state &&
    Umpire3.Temporal.Product.SpeculativeTask.speculativeCreationB state
}]

theorem speculativeTask_monitor_equivalent
    (state : Umpire3.Temporal.Product.SpeculativeTask.State) :
    speculativeTaskCreation.Holds (speculativeTaskObservations state) ↔
      Umpire3.Temporal.Product.SpeculativeTask.SpeculativeQualified state := by
  simp [MonitorDeclaration.Holds, speculativeTaskCreation, assuranceMonitor,
    speculativeTaskObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Product.SpeculativeTask.SpeculativeQualified,
    Umpire3.Temporal.Product.SpeculativeTask.SpeculativeTaskCreation]

def nexusOperationClosure := assuranceMonitor
  "monitor.nexus-operation.closure"
  "nexus-operation.closure" "nexus-operation-closed"

def nexusClosureObservations
    (state : Umpire3.Temporal.Product.NexusClosure.State) : List NormalizedObservation := [{
  identifier := "nexus-operation-closed"
  value := Umpire3.Temporal.Product.NexusClosure.closureB state
}]

theorem nexusOperationClosure_monitor_equivalent
    (state : Umpire3.Temporal.Product.NexusClosure.State) :
    nexusOperationClosure.Holds (nexusClosureObservations state) ↔
      Umpire3.Temporal.Product.NexusClosure.Closure state := by
  simp [MonitorDeclaration.Holds, nexusOperationClosure, assuranceMonitor,
    nexusClosureObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Product.NexusClosure.Closure]

def nexusActivityLinkConsistency := assuranceMonitor
  "monitor.nexus-activity.link-consistency"
  "nexus-activity.link-consistency" "nexus-activity-links-consistent"

def nexusActivityLinkObservations
    (state : Umpire3.Temporal.Product.NexusActivityLink.State) : List NormalizedObservation := [{
  identifier := "nexus-activity-links-consistent"
  value := Umpire3.Temporal.Product.NexusActivityLink.linkConsistencyB state
}]

theorem nexusActivityLink_monitor_equivalent
    (state : Umpire3.Temporal.Product.NexusActivityLink.State) :
    nexusActivityLinkConsistency.Holds (nexusActivityLinkObservations state) ↔
      Umpire3.Temporal.Product.NexusActivityLink.LinkConsistency state := by
  simp [MonitorDeclaration.Holds, nexusActivityLinkConsistency, assuranceMonitor,
    nexusActivityLinkObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Product.NexusActivityLink.LinkConsistency]

def nexusOperationTimeoutSemantics := assuranceMonitor
  "monitor.nexus-operation.timeout-semantics"
  "nexus-operation.timeout-semantics" "nexus-timeout-valid"

def nexusTimeoutObservations
    (state : Umpire3.Temporal.Product.NexusTimeout.State) : List NormalizedObservation := [{
  identifier := "nexus-timeout-valid"
  value := Umpire3.Temporal.Product.NexusTimeout.timeoutSemanticsB state
}]

theorem nexusTimeout_monitor_equivalent
    (state : Umpire3.Temporal.Product.NexusTimeout.State) :
    nexusOperationTimeoutSemantics.Holds (nexusTimeoutObservations state) ↔
      Umpire3.Temporal.Product.NexusTimeout.TimeoutSemantics state := by
  simp [MonitorDeclaration.Holds, nexusOperationTimeoutSemantics, assuranceMonitor,
    nexusTimeoutObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Product.NexusTimeout.TimeoutSemantics]

def callbackReferenceConsistency := assuranceMonitor
  "monitor.callback.reference-consistency"
  "callback.reference-consistency" "callback-reference-valid"

def callbackReferenceObservations
    (state : Umpire3.Temporal.Product.CallbackReference.State) : List NormalizedObservation := [{
  identifier := "callback-reference-valid"
  value := Umpire3.Temporal.Product.CallbackReference.referenceReadyB state &&
    Umpire3.Temporal.Product.CallbackReference.referenceConsistencyB state
}]

theorem callbackReference_monitor_equivalent
    (state : Umpire3.Temporal.Product.CallbackReference.State) :
    callbackReferenceConsistency.Holds (callbackReferenceObservations state) ↔
      Umpire3.Temporal.Product.CallbackReference.ReferenceQualified state := by
  simp [MonitorDeclaration.Holds, callbackReferenceConsistency, assuranceMonitor,
    callbackReferenceObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Product.CallbackReference.ReferenceQualified,
    Umpire3.Temporal.Product.CallbackReference.ReferenceConsistency]

def callbackResponseConsistency := assuranceMonitor
  "monitor.callback.response-consistency"
  "callback.response-consistency" "callback-response-consistent"

def callbackResponseObservations
    (state : Umpire3.Temporal.Product.CallbackResponse.State) : List NormalizedObservation := [{
  identifier := "callback-response-consistent"
  value := Umpire3.Temporal.Product.CallbackResponse.responseReadyB state &&
    Umpire3.Temporal.Product.CallbackResponse.responseConsistencyB state
}]

theorem callbackResponse_monitor_equivalent
    (state : Umpire3.Temporal.Product.CallbackResponse.State) :
    callbackResponseConsistency.Holds (callbackResponseObservations state) ↔
      Umpire3.Temporal.Product.CallbackResponse.ResponseQualified state := by
  simp [MonitorDeclaration.Holds, callbackResponseConsistency, assuranceMonitor,
    callbackResponseObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Product.CallbackResponse.ResponseQualified,
    Umpire3.Temporal.Product.CallbackResponse.ResponseConsistency]

def workflowTaskStarvation := assuranceMonitor
  "monitor.workflow-task.starvation"
  "workflow-task.starvation" "workflow-task-not-starved"

def workflowTaskStarvationObservations
    (state : Umpire3.Temporal.Product.WorkflowProgress.State) : List NormalizedObservation := [{
  identifier := "workflow-task-not-starved"
  value := Umpire3.Temporal.Product.WorkflowProgress.starvationReadyB state &&
    Umpire3.Temporal.Product.WorkflowProgress.workflowTaskStarvationB state
}]

theorem workflowTaskStarvation_monitor_equivalent
    (state : Umpire3.Temporal.Product.WorkflowProgress.State) :
    workflowTaskStarvation.Holds (workflowTaskStarvationObservations state) ↔
      Umpire3.Temporal.Product.WorkflowProgress.StarvationQualified state := by
  simp [MonitorDeclaration.Holds, workflowTaskStarvation, assuranceMonitor,
    workflowTaskStarvationObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Product.WorkflowProgress.StarvationQualified,
    Umpire3.Temporal.Product.WorkflowProgress.WorkflowTaskStarvation]

def entityProgress := assuranceMonitor
  "monitor.entity.progress" "entity.progress" "entity-progressed"

def entityProgressObservations
    (state : Umpire3.Temporal.Product.WorkflowProgress.State) : List NormalizedObservation := [{
  identifier := "entity-progressed"
  value := Umpire3.Temporal.Product.WorkflowProgress.progressReadyB state &&
    Umpire3.Temporal.Product.WorkflowProgress.entityProgressB state
}]

theorem entityProgress_monitor_equivalent
    (state : Umpire3.Temporal.Product.WorkflowProgress.State) :
    entityProgress.Holds (entityProgressObservations state) ↔
      Umpire3.Temporal.Product.WorkflowProgress.ProgressQualified state := by
  simp [MonitorDeclaration.Holds, entityProgress, assuranceMonitor,
    entityProgressObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Product.WorkflowProgress.ProgressQualified,
    Umpire3.Temporal.Product.WorkflowProgress.EntityProgress]

def continuationLineage := assuranceMonitor
  "monitor.workflow-run.continuation-lineage"
  "workflow-run.continuation-lineage" "workflow-continuation-lineage-valid"

def continuationLineageObservations
    (state : Umpire3.Temporal.Product.WorkflowLineage.State) : List NormalizedObservation := [{
  identifier := "workflow-continuation-lineage-valid"
  value := Umpire3.Temporal.Product.WorkflowLineage.continuationReadyB state &&
    Umpire3.Temporal.Product.WorkflowLineage.continuationConsistencyB state
}]

theorem continuationLineage_monitor_equivalent
    (state : Umpire3.Temporal.Product.WorkflowLineage.State) :
    continuationLineage.Holds (continuationLineageObservations state) ↔
      Umpire3.Temporal.Product.WorkflowLineage.ContinuationQualified state := by
  simp [MonitorDeclaration.Holds, continuationLineage, assuranceMonitor,
    continuationLineageObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Product.WorkflowLineage.ContinuationQualified,
    Umpire3.Temporal.Product.WorkflowLineage.ContinuationLineage]

def resetLineage := assuranceMonitor
  "monitor.workflow-run.reset-lineage"
  "workflow-run.reset-lineage" "workflow-reset-lineage-valid"

def resetLineageObservations
    (state : Umpire3.Temporal.Product.WorkflowLineage.State) : List NormalizedObservation := [{
  identifier := "workflow-reset-lineage-valid"
  value := Umpire3.Temporal.Product.WorkflowLineage.resetReadyB state &&
    Umpire3.Temporal.Product.WorkflowLineage.resetConsistencyB state
}]

theorem resetLineage_monitor_equivalent
    (state : Umpire3.Temporal.Product.WorkflowLineage.State) :
    resetLineage.Holds (resetLineageObservations state) ↔
      Umpire3.Temporal.Product.WorkflowLineage.ResetQualified state := by
  simp [MonitorDeclaration.Holds, resetLineage, assuranceMonitor,
    resetLineageObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Product.WorkflowLineage.ResetQualified,
    Umpire3.Temporal.Product.WorkflowLineage.ResetLineage]

def workflowRoutingIsolation := assuranceMonitor
  "monitor.workflow-task.routing-isolation"
  "workflow-task.routing-isolation" "workflow-routing-isolated"

def workflowRoutingObservations
    (state : Umpire3.Temporal.Product.WorkflowRouting.State) : List NormalizedObservation := [{
  identifier := "workflow-routing-isolated"
  value := Umpire3.Temporal.Product.WorkflowRouting.routingReadyB state &&
    Umpire3.Temporal.Product.WorkflowRouting.routingIsolationB state
}]

theorem workflowRouting_monitor_equivalent
    (state : Umpire3.Temporal.Product.WorkflowRouting.State) :
    workflowRoutingIsolation.Holds (workflowRoutingObservations state) ↔
      Umpire3.Temporal.Product.WorkflowRouting.RoutingQualified state := by
  simp [MonitorDeclaration.Holds, workflowRoutingIsolation, assuranceMonitor,
    workflowRoutingObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Product.WorkflowRouting.RoutingQualified,
    Umpire3.Temporal.Product.WorkflowRouting.RoutingIsolation]

def workflowOwnershipFencing := assuranceMonitor
  "monitor.workflow-task.ownership-fencing"
  "workflow-task.ownership-fencing" "workflow-ownership-fenced"

def workflowOwnershipObservations
    (state : Umpire3.Temporal.Product.WorkflowOwnership.State) : List NormalizedObservation := [{
  identifier := "workflow-ownership-fenced"
  value := Umpire3.Temporal.Product.WorkflowOwnership.ownershipReadyB state &&
    Umpire3.Temporal.Product.WorkflowOwnership.ownershipFencingB state
}]

theorem workflowOwnership_monitor_equivalent
    (state : Umpire3.Temporal.Product.WorkflowOwnership.State) :
    workflowOwnershipFencing.Holds (workflowOwnershipObservations state) ↔
      Umpire3.Temporal.Product.WorkflowOwnership.OwnershipQualified state := by
  simp [MonitorDeclaration.Holds, workflowOwnershipFencing, assuranceMonitor,
    workflowOwnershipObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Product.WorkflowOwnership.OwnershipQualified,
    Umpire3.Temporal.Product.WorkflowOwnership.OwnershipFencing]

def declarations : List MonitorDeclaration := [
  nexusCancellation,
  updateCompletion,
  taskAcknowledgement,
  speculativeTaskCreation,
  nexusOperationClosure,
  nexusActivityLinkConsistency,
  nexusOperationTimeoutSemantics,
  callbackReferenceConsistency,
  callbackResponseConsistency,
  workflowTaskStarvation,
  entityProgress,
  continuationLineage,
  resetLineage,
  workflowRoutingIsolation,
  workflowOwnershipFencing,
]

def nexusProperty (observations : List NormalizedObservation) : Prop :=
  nexusCancellation.expression.eval observations = some true

def updateProperty (observations : List NormalizedObservation) : Prop :=
  updateCompletion.expression.eval observations = some true

theorem nexus_monitor_equivalent (observations) :
    nexusCancellation.Holds observations ↔ nexusProperty observations := Iff.rfl

theorem update_monitor_equivalent (observations) :
    updateCompletion.Holds observations ↔ updateProperty observations := Iff.rfl

example : nexusCancellation.expression.eval [
    { identifier := "cancellation-won", value := true },
    { identifier := "stale-success-absent", value := false }
  ] = some false := by
    simp [nexusCancellation, MonitorExpression.eval, MonitorExpression.evalAll, lookupObservation]

example : updateCompletion.expression.eval [
    { identifier := "update-accepted", value := true },
    { identifier := "update-completed", value := true }
  ] = some true := by
    simp [updateCompletion, MonitorExpression.eval, MonitorExpression.evalAll, lookupObservation]

end Umpire3.Temporal.Monitors
