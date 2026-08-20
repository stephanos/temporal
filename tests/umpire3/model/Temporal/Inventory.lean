import Umpire3.Catalog

namespace Umpire3.Temporal.Inventory

private def action (identifier description : String)
    (capabilities : List String) : ActionDeclaration := {
  identifier, description, parameters := [], requiredCapabilities := capabilities
}

private def entity (identifier description : String) : EntityDeclaration :=
  { identifier, description }

private def observation (identifier description : String) : ObservationDeclaration :=
  { identifier, description }

private def property (identifier description statementHash : String) : PropertyDeclaration := {
  identifier
  description
  statementHash
  evidence := ["causal", "identity-lineage"]
}

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
    "sha256:93dde5178008a5a6c769d3980af88c43d8ffbaa31a4a7395f26cc3e1b872e6c5",
  property "nexus-operation.closure" "NexusOperationClosure"
    "sha256:f793044a1875cebac67335b7d653bd9eba07b63576f2be5e9dee47e4463baebb",
  property "nexus-activity.link-consistency" "NexusActivityLinkConsistency"
    "sha256:12fc8f2fc279123239747d42076963b622fdef3580bd8e7bdab3f2bd10c0d97d",
  property "nexus-operation.timeout-semantics" "NexusOperationTimeoutSemantics"
    "sha256:495ef4b393d35f35a9212c599dc2c2de340c9de7ce39dd4750a8c45caa102455",
  property "callback.reference-consistency" "CallbackReferenceConsistency"
    "sha256:3f85e1a045dab782595876f7123d8d91728b92019601fc4aa34c1b3b448cad5a",
  property "callback.response-consistency" "CallbackResponseConsistency"
    "sha256:60885f3085fba30e9f07872f3d6d0fd30ee4fea127451a217333770fcda31d8f",
  property "workflow-task.starvation" "WorkflowTaskStarvation"
    "sha256:467947d799f5bf5f4a7801065a13919e10936735699f707d54925df0a4f4da3f",
  property "entity.progress" "EntityProgress"
    "sha256:7a2ae813c7b09d9a4f6c152469aad915a1be0c8b0c87b7f1a4524ec702486e2e",
  property "workflow-run.continuation-lineage" "ContinuationLineage"
    "sha256:1c4ab49d19cc38f6c23286ff949d33bc52d05437f587ee9ff964de71845ff0ec",
  property "workflow-run.reset-lineage" "ResetLineage"
    "sha256:0d6fafb264288dc1f8de1cc9923a95e54ae6baae1697ccb67eca86e3f1c70570",
  property "workflow-task.routing-isolation" "WorkflowRoutingIsolation"
    "sha256:27ce559f16b78db637e1d35a0fab320da8382a15bb49f09e25178b55f1b45d6c",
  property "workflow-task.ownership-fencing" "WorkflowOwnershipFencing"
    "sha256:0ef8b9936c01a1d30d187d27045659bbb5622616b5fd7ae5d093118497d0139f",
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
