import Temporal.Families.NexusClosure.Feature
import Temporal.Families.NexusProgress.Feature
import Temporal.Families.NexusActivityLink.Feature
import Temporal.Families.NexusTimeout.Feature
import Temporal.Families.CallbackReference.Feature
import Temporal.Families.CallbackResponse.Feature
import Temporal.Families.WorkflowRoutingIsolation.LineageFeature
import Temporal.Families.WorkflowRoutingIsolation.RoutingFeature
import Temporal.Families.WorkflowOwnership.Feature
import Temporal.Families.SpeculativeTask.Feature
import Temporal.Families.WorkflowProgress.Feature
import Temporal.Families.TaskAcknowledgement.Feature
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

def taskAcknowledgementObservations
    (state : Umpire3.Temporal.Feature.TaskAck.State) : List NormalizedObservation := [{
  identifier := "workflow-task-acknowledged"
  value := decide (state = .acknowledged)
}]

theorem taskAcknowledgement_monitor_equivalent
    (state : Umpire3.Temporal.Feature.TaskAck.State) :
    taskAcknowledgement.Holds (taskAcknowledgementObservations state) ↔ state = .acknowledged := by
  cases state <;> simp [MonitorDeclaration.Holds, taskAcknowledgement,
    taskAcknowledgementObservations, MonitorExpression.eval, lookupObservation]

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
    (state : Umpire3.Temporal.Feature.SpeculativeTask.State) : List NormalizedObservation := [{
  identifier := "speculative-task-valid"
  value := Umpire3.Temporal.Feature.SpeculativeTask.speculativeReadyB state &&
    Umpire3.Temporal.Feature.SpeculativeTask.speculativeCreationB state
}]

theorem speculativeTask_monitor_equivalent
    (state : Umpire3.Temporal.Feature.SpeculativeTask.State) :
    speculativeTaskCreation.Holds (speculativeTaskObservations state) ↔
      Umpire3.Temporal.Feature.SpeculativeTask.SpeculativeQualified state := by
  simp [MonitorDeclaration.Holds, speculativeTaskCreation, assuranceMonitor,
    speculativeTaskObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.SpeculativeTask.SpeculativeQualified,
    Umpire3.Temporal.Feature.SpeculativeTask.SpeculativeTaskCreation]

def nexusOperationClosure := assuranceMonitor
  "monitor.nexus-operation.closure"
  "nexus-operation.closure" "nexus-operation-closed"

def nexusClosureObservations
    (state : Umpire3.Temporal.Feature.NexusClosure.State) : List NormalizedObservation := [{
  identifier := "nexus-operation-closed"
  value := Umpire3.Temporal.Feature.NexusClosure.closureB state
}]

theorem nexusOperationClosure_monitor_equivalent
    (state : Umpire3.Temporal.Feature.NexusClosure.State) :
    nexusOperationClosure.Holds (nexusClosureObservations state) ↔
      Umpire3.Temporal.Feature.NexusClosure.Closure state := by
  simp [MonitorDeclaration.Holds, nexusOperationClosure, assuranceMonitor,
    nexusClosureObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.NexusClosure.Closure]

def nexusOperationProgress := assuranceMonitor
  "monitor.nexus-operation.progress"
  "nexus-operation.progress" "nexus-operation-progressed"

def nexusProgressObservations
    (state : Umpire3.Temporal.Feature.NexusProgress.State) : List NormalizedObservation := [{
  identifier := "nexus-operation-progressed"
  value := Umpire3.Temporal.Feature.NexusProgress.progressReadyB state &&
    Umpire3.Temporal.Feature.NexusProgress.progressB state
}]

theorem nexusOperationProgress_monitor_equivalent
    (state : Umpire3.Temporal.Feature.NexusProgress.State) :
    nexusOperationProgress.Holds (nexusProgressObservations state) ↔
      Umpire3.Temporal.Feature.NexusProgress.ProgressQualified state := by
  simp [MonitorDeclaration.Holds, nexusOperationProgress, assuranceMonitor,
    nexusProgressObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.NexusProgress.ProgressQualified,
    Umpire3.Temporal.Feature.NexusProgress.NexusOperationProgress]

def nexusActivityLinkConsistency := assuranceMonitor
  "monitor.nexus-activity.link-consistency"
  "nexus-activity.link-consistency" "nexus-activity-links-consistent"

def nexusActivityLinkObservations
    (state : Umpire3.Temporal.Feature.NexusActivityLink.State) : List NormalizedObservation := [{
  identifier := "nexus-activity-links-consistent"
  value := Umpire3.Temporal.Feature.NexusActivityLink.linkConsistencyB state
}]

theorem nexusActivityLink_monitor_equivalent
    (state : Umpire3.Temporal.Feature.NexusActivityLink.State) :
    nexusActivityLinkConsistency.Holds (nexusActivityLinkObservations state) ↔
      Umpire3.Temporal.Feature.NexusActivityLink.LinkConsistency state := by
  simp [MonitorDeclaration.Holds, nexusActivityLinkConsistency, assuranceMonitor,
    nexusActivityLinkObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.NexusActivityLink.LinkConsistency]

def nexusOperationTimeoutSemantics := assuranceMonitor
  "monitor.nexus-operation.timeout-semantics"
  "nexus-operation.timeout-semantics" "nexus-timeout-valid"

def nexusTimeoutObservations
    (state : Umpire3.Temporal.Feature.NexusTimeout.State) : List NormalizedObservation := [{
  identifier := "nexus-timeout-valid"
  value := Umpire3.Temporal.Feature.NexusTimeout.timeoutSemanticsB state
}]

theorem nexusTimeout_monitor_equivalent
    (state : Umpire3.Temporal.Feature.NexusTimeout.State) :
    nexusOperationTimeoutSemantics.Holds (nexusTimeoutObservations state) ↔
      Umpire3.Temporal.Feature.NexusTimeout.TimeoutSemantics state := by
  simp [MonitorDeclaration.Holds, nexusOperationTimeoutSemantics, assuranceMonitor,
    nexusTimeoutObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.NexusTimeout.TimeoutSemantics]

def callbackReferenceConsistency := assuranceMonitor
  "monitor.callback.reference-consistency"
  "callback.reference-consistency" "callback-reference-valid"

def callbackReferenceObservations
    (state : Umpire3.Temporal.Feature.CallbackReference.State) : List NormalizedObservation := [{
  identifier := "callback-reference-valid"
  value := Umpire3.Temporal.Feature.CallbackReference.referenceReadyB state &&
    Umpire3.Temporal.Feature.CallbackReference.referenceConsistencyB state
}]

theorem callbackReference_monitor_equivalent
    (state : Umpire3.Temporal.Feature.CallbackReference.State) :
    callbackReferenceConsistency.Holds (callbackReferenceObservations state) ↔
      Umpire3.Temporal.Feature.CallbackReference.ReferenceQualified state := by
  simp [MonitorDeclaration.Holds, callbackReferenceConsistency, assuranceMonitor,
    callbackReferenceObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.CallbackReference.ReferenceQualified,
    Umpire3.Temporal.Feature.CallbackReference.ReferenceConsistency]

def callbackResponseConsistency := assuranceMonitor
  "monitor.callback.response-consistency"
  "callback.response-consistency" "callback-response-consistent"

def callbackResponseObservations
    (state : Umpire3.Temporal.Feature.CallbackResponse.State) : List NormalizedObservation := [{
  identifier := "callback-response-consistent"
  value := Umpire3.Temporal.Feature.CallbackResponse.responseReadyB state &&
    Umpire3.Temporal.Feature.CallbackResponse.responseConsistencyB state
}]

theorem callbackResponse_monitor_equivalent
    (state : Umpire3.Temporal.Feature.CallbackResponse.State) :
    callbackResponseConsistency.Holds (callbackResponseObservations state) ↔
      Umpire3.Temporal.Feature.CallbackResponse.ResponseQualified state := by
  simp [MonitorDeclaration.Holds, callbackResponseConsistency, assuranceMonitor,
    callbackResponseObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.CallbackResponse.ResponseQualified,
    Umpire3.Temporal.Feature.CallbackResponse.ResponseConsistency]

def workflowTaskStarvation := assuranceMonitor
  "monitor.workflow-task.starvation"
  "workflow-task.starvation" "workflow-task-not-starved"

def workflowTaskStarvationObservations
    (state : Umpire3.Temporal.Feature.WorkflowProgress.State) : List NormalizedObservation := [{
  identifier := "workflow-task-not-starved"
  value := Umpire3.Temporal.Feature.WorkflowProgress.starvationReadyB state &&
    Umpire3.Temporal.Feature.WorkflowProgress.workflowTaskStarvationB state
}]

theorem workflowTaskStarvation_monitor_equivalent
    (state : Umpire3.Temporal.Feature.WorkflowProgress.State) :
    workflowTaskStarvation.Holds (workflowTaskStarvationObservations state) ↔
      Umpire3.Temporal.Feature.WorkflowProgress.StarvationQualified state := by
  simp [MonitorDeclaration.Holds, workflowTaskStarvation, assuranceMonitor,
    workflowTaskStarvationObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.WorkflowProgress.StarvationQualified,
    Umpire3.Temporal.Feature.WorkflowProgress.WorkflowTaskStarvation]

def entityProgress := assuranceMonitor
  "monitor.entity.progress" "entity.progress" "entity-progressed"

def entityProgressObservations
    (state : Umpire3.Temporal.Feature.WorkflowProgress.State) : List NormalizedObservation := [{
  identifier := "entity-progressed"
  value := Umpire3.Temporal.Feature.WorkflowProgress.progressReadyB state &&
    Umpire3.Temporal.Feature.WorkflowProgress.entityProgressB state
}]

theorem entityProgress_monitor_equivalent
    (state : Umpire3.Temporal.Feature.WorkflowProgress.State) :
    entityProgress.Holds (entityProgressObservations state) ↔
      Umpire3.Temporal.Feature.WorkflowProgress.ProgressQualified state := by
  simp [MonitorDeclaration.Holds, entityProgress, assuranceMonitor,
    entityProgressObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.WorkflowProgress.ProgressQualified,
    Umpire3.Temporal.Feature.WorkflowProgress.EntityProgress]

def continuationLineage := assuranceMonitor
  "monitor.workflow-run.continuation-lineage"
  "workflow-run.continuation-lineage" "workflow-continuation-lineage-valid"

def continuationLineageObservations
    (state : Umpire3.Temporal.Feature.WorkflowLineage.State) : List NormalizedObservation := [{
  identifier := "workflow-continuation-lineage-valid"
  value := Umpire3.Temporal.Feature.WorkflowLineage.continuationReadyB state &&
    Umpire3.Temporal.Feature.WorkflowLineage.continuationConsistencyB state
}]

theorem continuationLineage_monitor_equivalent
    (state : Umpire3.Temporal.Feature.WorkflowLineage.State) :
    continuationLineage.Holds (continuationLineageObservations state) ↔
      Umpire3.Temporal.Feature.WorkflowLineage.ContinuationQualified state := by
  simp [MonitorDeclaration.Holds, continuationLineage, assuranceMonitor,
    continuationLineageObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.WorkflowLineage.ContinuationQualified,
    Umpire3.Temporal.Feature.WorkflowLineage.ContinuationLineage]

def resetLineage := assuranceMonitor
  "monitor.workflow-run.reset-lineage"
  "workflow-run.reset-lineage" "workflow-reset-lineage-valid"

def resetLineageObservations
    (state : Umpire3.Temporal.Feature.WorkflowLineage.State) : List NormalizedObservation := [{
  identifier := "workflow-reset-lineage-valid"
  value := Umpire3.Temporal.Feature.WorkflowLineage.resetReadyB state &&
    Umpire3.Temporal.Feature.WorkflowLineage.resetConsistencyB state
}]

theorem resetLineage_monitor_equivalent
    (state : Umpire3.Temporal.Feature.WorkflowLineage.State) :
    resetLineage.Holds (resetLineageObservations state) ↔
      Umpire3.Temporal.Feature.WorkflowLineage.ResetQualified state := by
  simp [MonitorDeclaration.Holds, resetLineage, assuranceMonitor,
    resetLineageObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.WorkflowLineage.ResetQualified,
    Umpire3.Temporal.Feature.WorkflowLineage.ResetLineage]

def workflowRoutingIsolation := assuranceMonitor
  "monitor.workflow-task.routing-isolation"
  "workflow-task.routing-isolation" "workflow-routing-isolated"

def workflowRoutingObservations
    (state : Umpire3.Temporal.Feature.WorkflowRouting.State) : List NormalizedObservation := [{
  identifier := "workflow-routing-isolated"
  value := Umpire3.Temporal.Feature.WorkflowRouting.routingReadyB state &&
    Umpire3.Temporal.Feature.WorkflowRouting.routingIsolationB state
}]

theorem workflowRouting_monitor_equivalent
    (state : Umpire3.Temporal.Feature.WorkflowRouting.State) :
    workflowRoutingIsolation.Holds (workflowRoutingObservations state) ↔
      Umpire3.Temporal.Feature.WorkflowRouting.RoutingQualified state := by
  simp [MonitorDeclaration.Holds, workflowRoutingIsolation, assuranceMonitor,
    workflowRoutingObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.WorkflowRouting.RoutingQualified,
    Umpire3.Temporal.Feature.WorkflowRouting.RoutingIsolation]

def workflowOwnershipFencing := assuranceMonitor
  "monitor.workflow-task.ownership-fencing"
  "workflow-task.ownership-fencing" "workflow-ownership-fenced"

def workflowOwnershipObservations
    (state : Umpire3.Temporal.Feature.WorkflowOwnership.State) : List NormalizedObservation := [{
  identifier := "workflow-ownership-fenced"
  value := Umpire3.Temporal.Feature.WorkflowOwnership.ownershipReadyB state &&
    Umpire3.Temporal.Feature.WorkflowOwnership.ownershipFencingB state
}]

theorem workflowOwnership_monitor_equivalent
    (state : Umpire3.Temporal.Feature.WorkflowOwnership.State) :
    workflowOwnershipFencing.Holds (workflowOwnershipObservations state) ↔
      Umpire3.Temporal.Feature.WorkflowOwnership.OwnershipQualified state := by
  simp [MonitorDeclaration.Holds, workflowOwnershipFencing, assuranceMonitor,
    workflowOwnershipObservations, MonitorExpression.eval, lookupObservation,
    Umpire3.Temporal.Feature.WorkflowOwnership.OwnershipQualified,
    Umpire3.Temporal.Feature.WorkflowOwnership.OwnershipFencing]

def declarations : List MonitorDeclaration := [
  nexusCancellation,
  updateCompletion,
  taskAcknowledgement,
  speculativeTaskCreation,
  nexusOperationClosure,
  nexusOperationProgress,
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
