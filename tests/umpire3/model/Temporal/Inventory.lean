import Umpire3.Catalog
import Temporal.Families.CallbackReference.Feature
import Temporal.Families.CallbackResponse.Feature
import Temporal.Families.NexusActivityLink.Feature
import Temporal.Families.NexusClosure.Feature
import Temporal.Families.NexusProgress.Feature
import Temporal.Families.NexusTimeout.Feature
import Temporal.Families.SpeculativeTask.Feature
import Temporal.Families.WorkflowRoutingIsolation.LineageFeature
import Temporal.Families.WorkflowOwnership.Feature
import Temporal.Families.WorkflowProgress.Feature
import Temporal.Families.WorkflowRoutingIsolation.RoutingFeature

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
    parameters := [{
      name := "nexus-completion"
      type := "string"
      required := false
      allowedValues := ["failed", "open-at-caller-close", "retry-stuck", "retry-then-success"]
    }],
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
  observation "nexus-operation-progressed" "A retrying Nexus operation settled within its progress deadline",
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
    (resolved_theorem% Umpire3.Temporal.Feature.SpeculativeTask.speculativeCreationSafe),
  property "nexus-operation.closure" "NexusOperationClosure"
    (resolved_theorem% Umpire3.Temporal.Feature.NexusClosure.closureSafe),
  property "nexus-operation.progress" "NexusOperationProgress"
    (resolved_theorem% Umpire3.Temporal.Feature.NexusProgress.progressSafe),
  property "nexus-activity.link-consistency" "NexusActivityLinkConsistency"
    (resolved_theorem% Umpire3.Temporal.Feature.NexusActivityLink.linkConsistencySafe),
  property "nexus-operation.timeout-semantics" "NexusOperationTimeoutSemantics"
    (resolved_theorem% Umpire3.Temporal.Feature.NexusTimeout.timeoutSemanticsSafe),
  property "callback.reference-consistency" "CallbackReferenceConsistency"
    (resolved_theorem% Umpire3.Temporal.Feature.CallbackReference.referenceConsistencySafe),
  property "callback.response-consistency" "CallbackResponseConsistency"
    (resolved_theorem% Umpire3.Temporal.Feature.CallbackResponse.responseConsistencySafe),
  property "workflow-task.starvation" "WorkflowTaskStarvation"
    (resolved_theorem% Umpire3.Temporal.Feature.WorkflowProgress.workflowTaskStarvationSafe),
  property "entity.progress" "EntityProgress"
    (resolved_theorem% Umpire3.Temporal.Feature.WorkflowProgress.entityProgressSafe),
  property "workflow-run.continuation-lineage" "ContinuationLineage"
    (resolved_theorem% Umpire3.Temporal.Feature.WorkflowLineage.continuationLineageSafe),
  property "workflow-run.reset-lineage" "ResetLineage"
    (resolved_theorem% Umpire3.Temporal.Feature.WorkflowLineage.resetLineageSafe),
  property "workflow-task.routing-isolation" "WorkflowRoutingIsolation"
    (resolved_theorem% Umpire3.Temporal.Feature.WorkflowRouting.routingIsolationSafe),
  property "workflow-task.ownership-fencing" "WorkflowOwnershipFencing"
    (resolved_theorem% Umpire3.Temporal.Feature.WorkflowOwnership.ownershipFencingSafe),
]

def nexusClosureFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.NexusClosure"
  description := "Caller Workflow and related Nexus operation closure contract"
}

def nexusLifecycleFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.NexusLifecycle"
  description := "Complete Nexus operation lifecycle and generated transition coverage denominator"
}

def nexusClosureSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.NexusClosure"
  description := "Independent Nexus lifecycle and Workflow closure mechanism"
}

def nexusClosureRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.NexusClosure"
  description := "Independent Nexus closure system-to-feature refinement"
}

def nexusProgressFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.NexusProgress"
  description := "Bounded Nexus retry progress contract"
}

def nexusProgressSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.NexusProgress"
  description := "Independent Nexus retry and deadline observation mechanism"
}

def nexusProgressRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.NexusProgress"
  description := "Nexus retry progress system-to-feature refinement"
}

def nexusTimeoutFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.NexusTimeout"
  description := "Nexus timeout configuration, outcome, and evidence relation contract"
}

def nexusTimeoutSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.NexusTimeout"
  description := "Ordered Nexus timeout history observation mechanism"
}

def nexusTimeoutRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.NexusTimeout"
  description := "Nexus timeout observation system-to-feature refinement"
}

def nexusActivityLinkFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.NexusActivityLink"
  description := "Reciprocal Nexus operation and Activity link contract"
}

def nexusActivityLinkSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.NexusActivityLink"
  description := "Nexus and Activity link observation and redelivery mechanism"
}

def nexusActivityLinkRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.NexusActivityLink"
  description := "Nexus Activity link observation system-to-feature refinement"
}

def callbackReferenceFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.CallbackReference"
  description := "Callback, Nexus operation, handler run, reference, and causal-order contract"
}

def callbackReferenceSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.CallbackReference"
  description := "Independent callback attachment and Nexus start observation mechanism"
}

def callbackReferenceRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.CallbackReference"
  description := "Callback reference observation system-to-feature refinement"
}

def callbackResponseFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.CallbackResponse"
  description := "Callback delivery, accepted response, idempotency, and operation-lifetime contract"
}

def callbackResponseSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.CallbackResponse"
  description := "Callback registration, settlement, response, and redelivery observation mechanism"
}

def callbackResponseRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.CallbackResponse"
  description := "Callback response observation system-to-feature refinement"
}

def workflowLineageFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.WorkflowLineage"
  description := "Workflow continuation and reset predecessor, original, and chain-root contract"
}

def workflowLineageSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.WorkflowLineage"
  description := "Workflow start-history lineage observation and redelivery mechanism"
}

def workflowLineageRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.WorkflowLineage"
  description := "Workflow lineage observation system-to-feature refinement"
}

def workflowRoutingFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.WorkflowRouting"
  description := "Workflow Task, poller, reservation, and Task Queue route-isolation contract"
}

def workflowRoutingSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.WorkflowRouting"
  description := "Workflow Task and poller route plus reservation observation mechanism"
}

def workflowRoutingRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.WorkflowRouting"
  description := "Workflow routing observation system-to-feature refinement"
}

def routingIsolationFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.RoutingIsolation"
  description := "Workflow lineage and Task Queue routing isolation contract"
}

def routingIsolationSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.RoutingIsolation"
  description := "Independent Workflow lineage and Task Queue routing observation mechanisms"
}

def routingIsolationRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.RoutingIsolation"
  description := "Workflow routing isolation system-to-feature refinements"
}

def workflowOwnershipFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.WorkflowOwnership"
  description := "Workflow Task attempt epoch and stale-owner fencing contract"
}

def workflowOwnershipSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.WorkflowOwnership"
  description := "Workflow Task ownership failure, rotation, rejection, and completion observation mechanism"
}

def workflowOwnershipRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.WorkflowOwnership"
  description := "Workflow ownership observation system-to-feature refinement"
}

def speculativeTaskFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.SpeculativeTask"
  description := "Update-linked speculative Workflow Task creation and commitment contract"
}

def speculativeTaskSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.SpeculativeTask"
  description := "Update request, speculative task creation, commit, and redelivery observation mechanism"
}

def speculativeTaskRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.SpeculativeTask"
  description := "Speculative Workflow Task observation system-to-feature refinement"
}

def workflowProgressFeatureModule : ModuleDeclaration := {
  identifier := "Temporal.Feature.WorkflowProgress"
  description := "Workflow Task availability deadline and task-to-entity progress contract"
}

def workflowProgressSystemModule : ModuleDeclaration := {
  identifier := "Temporal.System.WorkflowProgress"
  description := "Task enqueue, worker availability, dispatch, completion, and redelivery observation mechanism"
}

def workflowProgressRefinementModule : ModuleDeclaration := {
  identifier := "Temporal.Refinement.WorkflowProgress"
  description := "Workflow progress observation system-to-feature refinement"
}

def modules : List ModuleDeclaration := [
  nexusLifecycleFeatureModule,
  nexusClosureFeatureModule,
  nexusClosureSystemModule,
  nexusClosureRefinementModule,
  nexusProgressFeatureModule,
  nexusProgressSystemModule,
  nexusProgressRefinementModule,
  nexusTimeoutFeatureModule,
  nexusTimeoutSystemModule,
  nexusTimeoutRefinementModule,
  nexusActivityLinkFeatureModule,
  nexusActivityLinkSystemModule,
  nexusActivityLinkRefinementModule,
  callbackReferenceFeatureModule,
  callbackReferenceSystemModule,
  callbackReferenceRefinementModule,
  callbackResponseFeatureModule,
  callbackResponseSystemModule,
  callbackResponseRefinementModule,
  workflowLineageFeatureModule,
  workflowLineageSystemModule,
  workflowLineageRefinementModule,
  workflowRoutingFeatureModule,
  workflowRoutingSystemModule,
  workflowRoutingRefinementModule,
  routingIsolationFeatureModule,
  routingIsolationSystemModule,
  routingIsolationRefinementModule,
  workflowOwnershipFeatureModule,
  workflowOwnershipSystemModule,
  workflowOwnershipRefinementModule,
  speculativeTaskFeatureModule,
  speculativeTaskSystemModule,
  speculativeTaskRefinementModule,
  workflowProgressFeatureModule,
  workflowProgressSystemModule,
  workflowProgressRefinementModule,
]

private def routingTarget : TargetDeclaration := {
  identifier := "foundation-routing-isolation"
  modules := [routingIsolationFeatureModule.identifier, routingIsolationSystemModule.identifier,
    routingIsolationRefinementModule.identifier]
  properties := [
    "workflow-task.routing-isolation",
    "workflow-run.continuation-lineage",
    "workflow-run.reset-lineage",
  ]
}

def targets : List TargetDeclaration := [
  {
    identifier := "feature-nexus"
    modules := [nexusClosureFeatureModule.identifier, nexusClosureSystemModule.identifier,
      nexusClosureRefinementModule.identifier]
    properties := ["nexus-operation.closure"]
  },
  {
    identifier := "feature-nexus-progress"
    modules := [nexusProgressFeatureModule.identifier, nexusProgressSystemModule.identifier,
      nexusProgressRefinementModule.identifier]
    properties := ["nexus-operation.progress"]
  },
  {
    identifier := "feature-workflow-speculative-delivery"
    modules := [speculativeTaskFeatureModule.identifier, speculativeTaskSystemModule.identifier,
      speculativeTaskRefinementModule.identifier]
    properties := ["workflow-task.speculative-creation"]
  },
  {
    identifier := "foundation-delivery-safety"
    modules := [workflowProgressFeatureModule.identifier, workflowProgressSystemModule.identifier,
      workflowProgressRefinementModule.identifier]
    properties := ["entity.progress"]
  },
  {
    identifier := "foundation-ownership-fencing"
    modules := [workflowOwnershipFeatureModule.identifier, workflowOwnershipSystemModule.identifier,
      workflowOwnershipRefinementModule.identifier]
    properties := ["workflow-task.ownership-fencing"]
  },
  routingTarget,
  {
    identifier := "integration-activity-delivery"
    modules := [workflowProgressFeatureModule.identifier, workflowProgressSystemModule.identifier,
      workflowProgressRefinementModule.identifier]
    properties := ["entity.progress"]
  },
  {
    identifier := "integration-callback-nexus"
    modules := [callbackReferenceFeatureModule.identifier, callbackReferenceSystemModule.identifier,
      callbackReferenceRefinementModule.identifier]
    properties := ["callback.reference-consistency"]
  },
  {
    identifier := "integration-callback-workflow"
    modules := [callbackResponseFeatureModule.identifier, callbackResponseSystemModule.identifier,
      callbackResponseRefinementModule.identifier]
    properties := ["callback.response-consistency"]
  },
  {
    identifier := "integration-nexus-activity"
    modules := [nexusActivityLinkFeatureModule.identifier, nexusActivityLinkSystemModule.identifier,
      nexusActivityLinkRefinementModule.identifier]
    properties := ["nexus-activity.link-consistency"]
  },
  {
    identifier := "integration-nexus-timeout"
    modules := [nexusTimeoutFeatureModule.identifier, nexusTimeoutSystemModule.identifier,
      nexusTimeoutRefinementModule.identifier]
    properties := ["nexus-operation.timeout-semantics"]
  },
  {
    identifier := "integration-workflow-delivery"
    modules := [workflowProgressFeatureModule.identifier, workflowProgressSystemModule.identifier,
      workflowProgressRefinementModule.identifier]
    properties := ["workflow-task.starvation"]
  },
  {
    identifier := "protocol-atomic"
    modules := [callbackResponseFeatureModule.identifier, callbackResponseSystemModule.identifier,
      callbackResponseRefinementModule.identifier]
    properties := ["callback.response-consistency"]
  },
]

end Umpire3.Temporal.Inventory
