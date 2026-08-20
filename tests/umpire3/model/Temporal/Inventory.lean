import Umpire3.Catalog
import Temporal.Product.CallbackReference
import Temporal.Product.CallbackResponse
import Temporal.Product.NexusActivityLink
import Temporal.Product.NexusClosure
import Temporal.Product.NexusTimeout
import Temporal.Product.SpeculativeTask
import Temporal.Product.WorkflowLineage
import Temporal.Product.WorkflowOwnership
import Temporal.Product.WorkflowProgress
import Temporal.Product.WorkflowRouting

namespace Umpire3.Temporal.Inventory

private def action (identifier description : String)
    (capabilities : List String) : ActionDeclaration := {
  identifier, description, parameters := [], requiredCapabilities := capabilities
}

private def entity (identifier description : String) : EntityDeclaration :=
  { identifier, description }

private def observation (identifier description : String) : ObservationDeclaration :=
  { identifier, description }

private def property (identifier description : String)
    (proof : ResolvedTheorem) : PropertyDeclaration :=
  (RegisteredProperty.mk identifier description ["causal", "identity-lineage"] proof).declaration

def entities : List EntityDeclaration := [
  entity "namespace" "Temporal Namespace",
  entity "task-queue" "Temporal Task Queue",
  entity "workflow-run" "A concrete Workflow Execution run",
  entity "activity" "Activity command identity",
  entity "activity-execution" "A concrete Activity attempt",
  entity "callback" "A callback registered by a Temporal operation",
]

def relations : List RelationDeclaration := [
  { identifier := "namespace.contains-task-queue", source := "namespace", target := "task-queue",
    description := "A Task Queue belongs to a Namespace" },
  { identifier := "workflow.has-run", source := "workflow", target := "workflow-run",
    description := "A Workflow identity has concrete runs" },
  { identifier := "workflow-run.uses-task-queue", source := "workflow-run", target := "task-queue",
    description := "A Workflow run routes Workflow Tasks through a Task Queue" },
  { identifier := "workflow-run.has-task", source := "workflow-run", target := "workflow-task",
    description := "A Workflow Task belongs to a Workflow run" },
  { identifier := "workflow-run.has-activity", source := "workflow-run", target := "activity",
    description := "An Activity belongs to a Workflow run" },
  { identifier := "activity.has-execution", source := "activity", target := "activity-execution",
    description := "An Activity has concrete attempts" },
  { identifier := "nexus-operation.links-activity", source := "nexus-operation", target := "activity",
    description := "Nexus and Activity references agree in both directions" },
  { identifier := "nexus-operation.has-callback", source := "nexus-operation", target := "callback",
    description := "A Nexus operation registers a callback" },
  { identifier := "workflow-run.has-callback", source := "workflow-run", target := "callback",
    description := "A Workflow run registers a callback" },
  { identifier := "workflow-run.continues-as", source := "workflow-run", target := "workflow-run",
    description := "A continued or reset run retains explicit lineage" },
]

def actions : List ActionDeclaration := [
  action "create-speculative-workflow-task" "Create a speculative Workflow Task" ["workflow-task-control"],
  action "commit-speculative-workflow-task" "Commit a speculative Workflow Task" ["history-observation"],
  { identifier := "close-nexus-operation", description := "Close a Nexus operation",
    requiredCapabilities := ["nexus"], footprint := [
      { protocol := "grpc", service := "history", route := "RecordWorkflowTaskStarted" },
      { protocol := "grpc", service := "history", route := "RespondWorkflowTaskCompleted" },
      { protocol := "grpc", service := "history", route := "UpdateWorkflowExecution" },
      { protocol := "grpc", service := "matching", route := "AddWorkflowTask" },
      { protocol := "http", service := "nexus", route := "/service/operation" },
    ] },
  action "link-nexus-activity" "Create bidirectional Nexus and Activity links" ["history-observation"],
  action "timeout-nexus-operation" "Record Nexus timeout semantics" ["nexus-observation"],
  action "register-callback" "Register a callback reference" ["history-observation"],
  action "record-callback-response" "Record a callback response" ["history-observation"],
  action "dispatch-assurance-workflow-task" "Dispatch a Workflow Task for progress" ["workflow-task-control"],
  action "progress-entity" "Advance an entity lifecycle" ["history-observation"],
  action "continue-workflow" "Continue a Workflow as a new run with explicit lineage" ["workflow-task-control", "history-observation"],
  action "reset-workflow" "Reset a Workflow into a new run with explicit lineage" ["workflow-task-control", "history-observation"],
  action "route-workflow-task" "Route a Workflow Task through its declared Task Queue" ["workflow-task-control", "history-observation"],
  action "fence-workflow-owner" "Reject completion from a superseded Workflow Task owner" ["workflow-task-control", "history-observation"],
]

def observations : List ObservationDeclaration := [
  observation "speculative-task-valid" "Speculative Workflow Task creation is valid",
  observation "nexus-operation-closed" "A closed Nexus operation is terminal",
  observation "nexus-activity-links-consistent" "Nexus and Activity links agree",
  observation "nexus-timeout-valid" "Nexus timeout state is terminal and recorded",
  observation "callback-reference-valid" "Callback reference resolves to its owner",
  observation "callback-response-consistent" "Callback response agrees with its reference",
  observation "workflow-task-not-starved" "An available worker permits Workflow Task progress",
  observation "entity-progressed" "The selected entity made lifecycle progress",
  observation "workflow-continuation-lineage-valid" "The continued run names its predecessor and chain root",
  observation "workflow-reset-lineage-valid" "The reset run names the reset predecessor and chain root",
  observation "workflow-routing-isolated" "Workflow Tasks were scheduled on and completed from the declared Task Queue",
  observation "workflow-ownership-fenced" "A superseded Workflow Task completion was rejected before the current owner completed",
]

def properties : List PropertyDeclaration := [
  property "workflow-task.speculative-creation" "SpeculativeTaskCreation"
    (resolved_theorem% Umpire3.Temporal.Product.SpeculativeTask.speculativeCreationSafe),
  property "nexus-operation.closure" "NexusOperationClosure"
    (resolved_theorem% Umpire3.Temporal.Product.NexusClosure.closureSafe),
  property "nexus-activity.link-consistency" "NexusActivityLinkConsistency"
    (resolved_theorem% Umpire3.Temporal.Product.NexusActivityLink.linkConsistencySafe),
  property "nexus-operation.timeout-semantics" "NexusOperationTimeoutSemantics"
    (resolved_theorem% Umpire3.Temporal.Product.NexusTimeout.timeoutSemanticsSafe),
  property "callback.reference-consistency" "CallbackReferenceConsistency"
    (resolved_theorem% Umpire3.Temporal.Product.CallbackReference.referenceConsistencySafe),
  property "callback.response-consistency" "CallbackResponseConsistency"
    (resolved_theorem% Umpire3.Temporal.Product.CallbackResponse.responseConsistencySafe),
  property "workflow-task.starvation" "WorkflowTaskStarvation"
    (resolved_theorem% Umpire3.Temporal.Product.WorkflowProgress.workflowTaskStarvationSafe),
  property "entity.progress" "EntityProgress"
    (resolved_theorem% Umpire3.Temporal.Product.WorkflowProgress.entityProgressSafe),
  property "workflow-run.continuation-lineage" "ContinuationLineage"
    (resolved_theorem% Umpire3.Temporal.Product.WorkflowLineage.continuationLineageSafe),
  property "workflow-run.reset-lineage" "ResetLineage"
    (resolved_theorem% Umpire3.Temporal.Product.WorkflowLineage.resetLineageSafe),
  property "workflow-task.routing-isolation" "WorkflowRoutingIsolation"
    (resolved_theorem% Umpire3.Temporal.Product.WorkflowRouting.routingIsolationSafe),
  property "workflow-task.ownership-fencing" "WorkflowOwnershipFencing"
    (resolved_theorem% Umpire3.Temporal.Product.WorkflowOwnership.ownershipFencingSafe),
]

def nexusClosureProductModule : ModuleDeclaration := {
  identifier := "Temporal.Product.NexusClosure"
  description := "Caller Workflow and related Nexus operation closure contract"
}

def nexusLifecycleProductModule : ModuleDeclaration := {
  identifier := "Temporal.Product.NexusLifecycle"
  description := "Complete Nexus operation lifecycle and generated transition coverage denominator"
}

def nexusClosureSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.NexusClosure"
  description := "Nexus task, ownership epoch, completion, and Workflow closure mechanism"
}

def nexusClosureRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.NexusClosure"
  description := "Nexus closure system-to-product refinement"
}

def nexusTimeoutProductModule : ModuleDeclaration := {
  identifier := "Temporal.Product.NexusTimeout"
  description := "Nexus timeout configuration, outcome, and evidence relation contract"
}

def nexusTimeoutSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.NexusTimeout"
  description := "Ordered Nexus timeout history observation mechanism"
}

def nexusTimeoutRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.NexusTimeout"
  description := "Nexus timeout observation system-to-product refinement"
}

def nexusActivityLinkProductModule : ModuleDeclaration := {
  identifier := "Temporal.Product.NexusActivityLink"
  description := "Reciprocal Nexus operation and Activity link contract"
}

def nexusActivityLinkSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.NexusActivityLink"
  description := "Nexus and Activity link observation and redelivery mechanism"
}

def nexusActivityLinkRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.NexusActivityLink"
  description := "Nexus Activity link observation system-to-product refinement"
}

def callbackReferenceProductModule : ModuleDeclaration := {
  identifier := "Temporal.Product.CallbackReference"
  description := "Callback, Nexus operation, handler run, reference, and causal-order contract"
}

def callbackReferenceSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.CallbackReference"
  description := "Independent callback attachment and Nexus start observation mechanism"
}

def callbackReferenceRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.CallbackReference"
  description := "Callback reference observation system-to-product refinement"
}

def callbackResponseProductModule : ModuleDeclaration := {
  identifier := "Temporal.Product.CallbackResponse"
  description := "Callback delivery, accepted response, idempotency, and operation-lifetime contract"
}

def callbackResponseSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.CallbackResponse"
  description := "Callback registration, settlement, response, and redelivery observation mechanism"
}

def callbackResponseRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.CallbackResponse"
  description := "Callback response observation system-to-product refinement"
}

def workflowLineageProductModule : ModuleDeclaration := {
  identifier := "Temporal.Product.WorkflowLineage"
  description := "Workflow continuation and reset predecessor, original, and chain-root contract"
}

def workflowLineageSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.WorkflowLineage"
  description := "Workflow start-history lineage observation and redelivery mechanism"
}

def workflowLineageRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.WorkflowLineage"
  description := "Workflow lineage observation system-to-product refinement"
}

def workflowRoutingProductModule : ModuleDeclaration := {
  identifier := "Temporal.Product.WorkflowRouting"
  description := "Workflow Task, poller, reservation, and Task Queue route-isolation contract"
}

def workflowRoutingSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.WorkflowRouting"
  description := "Workflow Task and poller route plus reservation observation mechanism"
}

def workflowRoutingRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.WorkflowRouting"
  description := "Workflow routing observation system-to-product refinement"
}

def workflowOwnershipProductModule : ModuleDeclaration := {
  identifier := "Temporal.Product.WorkflowOwnership"
  description := "Workflow Task attempt epoch and stale-owner fencing contract"
}

def workflowOwnershipSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.WorkflowOwnership"
  description := "Workflow Task ownership failure, rotation, rejection, and completion observation mechanism"
}

def workflowOwnershipRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.WorkflowOwnership"
  description := "Workflow ownership observation system-to-product refinement"
}

def speculativeTaskProductModule : ModuleDeclaration := {
  identifier := "Temporal.Product.SpeculativeTask"
  description := "Update-linked speculative Workflow Task creation and commitment contract"
}

def speculativeTaskSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.SpeculativeTask"
  description := "Update request, speculative task creation, commit, and redelivery observation mechanism"
}

def speculativeTaskRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.SpeculativeTask"
  description := "Speculative Workflow Task observation system-to-product refinement"
}

def workflowProgressProductModule : ModuleDeclaration := {
  identifier := "Temporal.Product.WorkflowProgress"
  description := "Workflow Task availability deadline and task-to-entity progress contract"
}

def workflowProgressSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.WorkflowProgress"
  description := "Task enqueue, worker availability, dispatch, completion, and redelivery observation mechanism"
}

def workflowProgressRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.WorkflowProgress"
  description := "Workflow progress observation system-to-product refinement"
}

def modules : List ModuleDeclaration := [
  nexusLifecycleProductModule,
  nexusClosureProductModule,
  nexusClosureSystemModule,
  nexusClosureRefinementModule,
  nexusTimeoutProductModule,
  nexusTimeoutSystemModule,
  nexusTimeoutRefinementModule,
  nexusActivityLinkProductModule,
  nexusActivityLinkSystemModule,
  nexusActivityLinkRefinementModule,
  callbackReferenceProductModule,
  callbackReferenceSystemModule,
  callbackReferenceRefinementModule,
  callbackResponseProductModule,
  callbackResponseSystemModule,
  callbackResponseRefinementModule,
  workflowLineageProductModule,
  workflowLineageSystemModule,
  workflowLineageRefinementModule,
  workflowRoutingProductModule,
  workflowRoutingSystemModule,
  workflowRoutingRefinementModule,
  workflowOwnershipProductModule,
  workflowOwnershipSystemModule,
  workflowOwnershipRefinementModule,
  speculativeTaskProductModule,
  speculativeTaskSystemModule,
  speculativeTaskRefinementModule,
  workflowProgressProductModule,
  workflowProgressSystemModule,
  workflowProgressRefinementModule,
]

private def routingTarget : TargetDeclaration := {
  identifier := "foundation-routing-isolation"
  modules := [workflowLineageProductModule.identifier, workflowLineageSystemModule.identifier,
    workflowLineageRefinementModule.identifier, workflowRoutingProductModule.identifier,
    workflowRoutingSystemModule.identifier, workflowRoutingRefinementModule.identifier]
  properties := [
    "workflow-task.routing-isolation",
    "workflow-run.continuation-lineage",
    "workflow-run.reset-lineage",
  ]
}

def targets : List TargetDeclaration := [
  {
    identifier := "feature-nexus"
    modules := [nexusLifecycleProductModule.identifier, nexusClosureProductModule.identifier, nexusClosureSystemModule.identifier,
      nexusClosureRefinementModule.identifier]
    properties := ["nexus-operation.closure"]
  },
  {
    identifier := "feature-workflow-speculative-delivery"
    modules := [speculativeTaskProductModule.identifier, speculativeTaskSystemModule.identifier,
      speculativeTaskRefinementModule.identifier]
    properties := ["workflow-task.speculative-creation"]
  },
  {
    identifier := "foundation-delivery-safety"
    modules := [workflowProgressProductModule.identifier, workflowProgressSystemModule.identifier,
      workflowProgressRefinementModule.identifier]
    properties := ["entity.progress"]
  },
  {
    identifier := "foundation-ownership-fencing"
    modules := [workflowOwnershipProductModule.identifier, workflowOwnershipSystemModule.identifier,
      workflowOwnershipRefinementModule.identifier]
    properties := ["workflow-task.ownership-fencing"]
  },
  routingTarget,
  {
    identifier := "integration-activity-delivery"
    modules := [workflowProgressProductModule.identifier, workflowProgressSystemModule.identifier,
      workflowProgressRefinementModule.identifier]
    properties := ["entity.progress"]
  },
  {
    identifier := "integration-callback-nexus"
    modules := [callbackReferenceProductModule.identifier, callbackReferenceSystemModule.identifier,
      callbackReferenceRefinementModule.identifier]
    properties := ["callback.reference-consistency"]
  },
  {
    identifier := "integration-callback-workflow"
    modules := [callbackResponseProductModule.identifier, callbackResponseSystemModule.identifier,
      callbackResponseRefinementModule.identifier]
    properties := ["callback.response-consistency"]
  },
  {
    identifier := "integration-nexus-activity"
    modules := [nexusActivityLinkProductModule.identifier, nexusActivityLinkSystemModule.identifier,
      nexusActivityLinkRefinementModule.identifier]
    properties := ["nexus-activity.link-consistency"]
  },
  {
    identifier := "integration-nexus-timeout"
    modules := [nexusTimeoutProductModule.identifier, nexusTimeoutSystemModule.identifier,
      nexusTimeoutRefinementModule.identifier]
    properties := ["nexus-operation.timeout-semantics"]
  },
  {
    identifier := "integration-workflow-delivery"
    modules := [workflowProgressProductModule.identifier, workflowProgressSystemModule.identifier,
      workflowProgressRefinementModule.identifier]
    properties := ["workflow-task.starvation"]
  },
  {
    identifier := "protocol-atomic"
    modules := [callbackResponseProductModule.identifier, callbackResponseSystemModule.identifier,
      callbackResponseRefinementModule.identifier]
    properties := ["callback.response-consistency"]
  },
]

end Umpire3.Temporal.Inventory
