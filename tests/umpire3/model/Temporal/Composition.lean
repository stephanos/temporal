import Temporal.Product.TaskAck
import Temporal.Product.NexusLifecycle
import Temporal.Inventory
import Temporal.Refinement.NexusClosure
import Temporal.Refinement.NexusActivityLink
import Temporal.Refinement.NexusTimeout
import Temporal.Refinement.CallbackReference
import Temporal.Refinement.CallbackResponse
import Temporal.Refinement.WorkflowLineage
import Temporal.Refinement.WorkflowRouting
import Temporal.Refinement.WorkflowOwnership
import Temporal.Refinement.SpeculativeTask
import Temporal.Refinement.WorkflowProgress
import Temporal.System.NexusTasks
import Temporal.System.UpdateTasks
import Umpire3.Composition

namespace Umpire3.Temporal.Composition

private def presentObligation (identifier kind detail : String) : ModelObligation := {
  identifier, kind, detail, status := "metadata-present"
}

def deliveryProvider : ModuleContract where
  identifier := "Temporal.System.TaskDelivery"
  rank := 0
  owns := ["relation:task-delivery.current-completion"]
  provides := [{
    identifier := System.TaskDelivery.guarantee.identifier
    statementHash := System.TaskDelivery.guarantee.statementHash
  }]
  interferenceActions := ["task-delivery.environment-tick"]
  obligations := [presentObligation "task-delivery.guarantee" "guarantee"
    "Current-owner completion is versioned and proved against stale completion."]

private def deliveryRequirement : ContractRequirement := {
  providerModule := deliveryProvider.identifier
  guarantee := System.TaskDelivery.guarantee.identifier
  statementHash := System.TaskDelivery.guarantee.statementHash
}

def productNexus : ModuleContract where
  identifier := "Temporal.Product.Nexus"
  rank := 0
  owns := ["entity:nexus-operation", "property:nexus.cancellation.won-excludes-success"]
  obligations := [presentObligation "nexus.product" "product"
    "Product lifecycle, executable transition relation, and safety theorem are checked."]

def systemNexus : ModuleContract where
  identifier := "Temporal.System.NexusTasks"
  rank := 1
  owns := ["mechanism:nexus-task-delivery"]
  requires := [deliveryRequirement]
  interferenceActions := deliveryProvider.interferenceActions
  obligations := [presentObligation "nexus.system" "mechanism"
    "Task ownership and stale completion mechanism is executable."]

def refinementNexus : ModuleContract where
  identifier := "Temporal.Refinement.NexusTasks"
  rank := 2
  obligations := [presentObligation "nexus.refinement" "refinement"
    "Every system step refines the Nexus product or stutters."]

def productUpdate : ModuleContract where
  identifier := "Temporal.Product.Update"
  rank := 0
  owns := ["entity:workflow-update", "property:workflow-update.accepted-completes-through-history"]
  obligations := [presentObligation "update.product" "product"
    "Update lifecycle and completion stability are checked."]

def systemUpdate : ModuleContract where
  identifier := "Temporal.System.UpdateTasks"
  rank := 1
  owns := ["mechanism:update-task-delivery"]
  requires := [deliveryRequirement]
  interferenceActions := deliveryProvider.interferenceActions
  obligations := [presentObligation "update.system" "mechanism"
    "Update task and history mechanism is executable."]

def refinementUpdate : ModuleContract where
  identifier := "Temporal.Refinement.UpdateTasks"
  rank := 2
  obligations := [presentObligation "update.refinement" "refinement"
    "Every Update system step refines the product or stutters."]

def productTaskAck : ModuleContract where
  identifier := Product.TaskAck.declaration.module.identifier
  rank := 0
  owns := ["entity:workflow-task", "property:task-delivery.acknowledged-removes-backlog"]
  obligations := [
    presentObligation "task-ack.product" "product"
      "The lifecycle, executable equivalence, and acknowledgement theorem are checked.",
    presentObligation "task-ack.live-realization" "realization"
      "The public Workflow Task protocol adapter realizes enqueue, delivery, acknowledgement, and cleanup.",
  ]

def productNexusClosure : ModuleContract where
  identifier := Inventory.nexusClosureProductModule.identifier
  rank := 0
  owns := ["relation:nexus-operation.caller-workflow", "property:nexus-operation.closure"]
  obligations := [
    presentObligation "nexus-closure.product" "product"
      "Two operation identities, their caller relation, all terminal outcomes, and the closure invariant are executable and safe.",
    presentObligation "nexus-closure.non-vacuity" "non-vacuity"
      "Permitted closure reaches the invariant antecedent and the guard-removal mutation violates it.",
  ]

def systemNexusClosure : ModuleContract where
  identifier := Inventory.nexusClosureSystemModule.identifier
  rank := 1
  owns := ["mechanism:nexus-operation.workflow-closure"]
  obligations := [presentObligation "nexus-closure.system" "mechanism"
    "Task dispatch, start observation, completion-before-start, ownership epochs, persistence, and caller closure are executable."]

def refinementNexusClosure : ModuleContract where
  identifier := Inventory.nexusClosureRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "nexus-closure.refinement" "refinement"
      "Every system transition refines the relational product or stutters.",
    presentObligation "nexus-closure.monitor" "monitor"
      "The generated monitor is equivalent to the product predicate and the SDK adapter orders operation and caller terminal history.",
  ]

def productNexusTimeout : ModuleContract where
  identifier := Inventory.nexusTimeoutProductModule.identifier
  rank := 0
  owns := ["relation:nexus-operation.timeout-evidence", "property:nexus-operation.timeout-semantics"]
  obligations := [
    presentObligation "nexus-timeout.product" "product"
      "Configured operations, timeout kinds, failure metadata, evidence identity, and validity are relational and executable.",
    presentObligation "nexus-timeout.non-vacuity" "non-vacuity"
      "A valid timeout reaches the antecedent while wrong kind or message mutations are rejected by the weakened guard model.",
  ]

def systemNexusTimeout : ModuleContract where
  identifier := Inventory.nexusTimeoutSystemModule.identifier
  rank := 1
  owns := ["mechanism:nexus-timeout.history-observation"]
  obligations := [presentObligation "nexus-timeout.system" "mechanism"
    "Ordered and duplicate history observations execute independently of the product contract."]

def refinementNexusTimeout : ModuleContract where
  identifier := Inventory.nexusTimeoutRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "nexus-timeout.refinement" "refinement"
      "Every timeout observation step refines the product or stutters.",
    presentObligation "nexus-timeout.monitor" "monitor"
      "Lean and SDK monitors require start-to-close kind and operation-timeout failure metadata.",
  ]

def productNexusActivityLink : ModuleContract where
  identifier := Inventory.nexusActivityLinkProductModule.identifier
  rank := 0
  owns := ["relation:nexus-operation.links-activity", "property:nexus-activity.link-consistency"]
  obligations := [
    presentObligation "nexus-activity-link.product" "product"
      "Two operations, two Activities, observation presence, and both link directions are relational and executable.",
    presentObligation "nexus-activity-link.non-vacuity" "non-vacuity"
      "A reciprocal pair is accepted while the one-sided-link weakened guard violates consistency.",
  ]

def systemNexusActivityLink : ModuleContract where
  identifier := Inventory.nexusActivityLinkSystemModule.identifier
  rank := 1
  owns := ["mechanism:nexus-activity-link.history-observation"]
  obligations := [presentObligation "nexus-activity-link.system" "mechanism"
    "Independent operation and Activity observations plus duplicate delivery are executable."]

def refinementNexusActivityLink : ModuleContract where
  identifier := Inventory.nexusActivityLinkRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "nexus-activity-link.refinement" "refinement"
      "Every link observation step refines the reciprocal product or stutters.",
    presentObligation "nexus-activity-link.monitor" "monitor"
      "Lean and SDK monitors require independently observed forward and reverse public links.",
  ]

def productCallbackReference : ModuleContract where
  identifier := Inventory.callbackReferenceProductModule.identifier
  rank := 0
  owns := ["relation:callback-operation-handler-reference", "property:callback.reference-consistency"]
  obligations := [
    presentObligation "callback-reference.product" "product"
      "Two callback, operation, and handler identities plus reference kind, value, and causal position are relational and executable.",
    presentObligation "callback-reference.non-vacuity" "non-vacuity"
      "Matching event and request references are admitted while mismatched handler, value, kind, order, and malformed evidence are rejected.",
  ]

def systemCallbackReference : ModuleContract where
  identifier := Inventory.callbackReferenceSystemModule.identifier
  rank := 1
  owns := ["mechanism:callback-reference.history-observation"]
  obligations := [presentObligation "callback-reference.system" "mechanism"
    "Attachment, Nexus start, and duplicate attachment observations execute independently of the product contract."]

def refinementCallbackReference : ModuleContract where
  identifier := Inventory.callbackReferenceRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "callback-reference.refinement" "refinement"
      "Every callback reference observation step refines the product or stutters.",
    presentObligation "callback-reference.monitor" "monitor"
      "Lean and live SDK monitors require a mechanism-qualified callback receipt and matching public history evidence.",
  ]

def productCallbackResponse : ModuleContract where
  identifier := Inventory.callbackResponseProductModule.identifier
  rank := 0
  owns := ["relation:callback-delivery-response", "property:callback.response-consistency"]
  obligations := [
    presentObligation "callback-response.product" "product"
      "Two callback, operation, and delivery identities plus accepted response fingerprints and causal settlement order are relational and executable.",
    presentObligation "callback-response.non-vacuity" "non-vacuity"
      "Idempotent duplicates and terminal late responses are admitted while conflicting responses and non-terminal late responses are rejected.",
  ]

def systemCallbackResponse : ModuleContract where
  identifier := Inventory.callbackResponseSystemModule.identifier
  rank := 1
  owns := ["mechanism:callback-response.delivery-observation"]
  obligations := [presentObligation "callback-response.system" "mechanism"
    "Registration, settlement, response, and duplicate response observations execute independently of the product contract."]

def refinementCallbackResponse : ModuleContract where
  identifier := Inventory.callbackResponseRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "callback-response.refinement" "refinement"
      "Every callback response observation step refines the product or stutters.",
    presentObligation "callback-response.monitor" "monitor"
      "Lean and live SDK monitors require independently qualified registration and response receipts with terminal history.",
  ]

def productWorkflowLineage : ModuleContract where
  identifier := Inventory.workflowLineageProductModule.identifier
  rank := 0
  owns := ["relation:workflow-run.continues-as", "property:workflow-run.continuation-lineage",
    "property:workflow-run.reset-lineage"]
  obligations := [
    presentObligation "workflow-lineage.product" "product"
      "Two run identities and continuation/reset predecessor, original-event, and first-run relations are executable and safe.",
    presentObligation "workflow-lineage.non-vacuity" "non-vacuity"
      "Valid continuation and reset histories are admitted while a copied-original continuation mutation is rejected.",
  ]

def systemWorkflowLineage : ModuleContract where
  identifier := Inventory.workflowLineageSystemModule.identifier
  rank := 1
  owns := ["mechanism:workflow-lineage.history-observation"]
  obligations := [presentObligation "workflow-lineage.system" "mechanism"
    "Successor start-history observations and duplicate delivery are executable."]

def refinementWorkflowLineage : ModuleContract where
  identifier := Inventory.workflowLineageRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "workflow-lineage.refinement" "refinement"
      "Every Workflow lineage observation step refines the product or stutters.",
    presentObligation "workflow-lineage.monitor" "monitor"
      "Lean and SDK monitors agree on continuation and reset lineage and require typed live mechanism receipts.",
  ]

def productWorkflowRouting : ModuleContract where
  identifier := Inventory.workflowRoutingProductModule.identifier
  rank := 0
  owns := ["relation:workflow-task.route", "relation:poller.route",
    "property:workflow-task.routing-isolation"]
  obligations := [
    presentObligation "workflow-routing.product" "product"
      "Two tasks, pollers, routes, and reservation attempts are relational and executable.",
    presentObligation "workflow-routing.non-vacuity" "non-vacuity"
      "A same-route reservation is admitted while a cross-route reservation is rejected by the checked guard.",
  ]

def systemWorkflowRouting : ModuleContract where
  identifier := Inventory.workflowRoutingSystemModule.identifier
  rank := 1
  owns := ["mechanism:workflow-routing.task-queue-observation"]
  obligations := [presentObligation "workflow-routing.system" "mechanism"
    "Task route, poller route, reservation, and duplicate reservation observations are executable."]

def refinementWorkflowRouting : ModuleContract where
  identifier := Inventory.workflowRoutingRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "workflow-routing.refinement" "refinement"
      "Every Workflow routing observation step refines the product or stutters.",
    presentObligation "workflow-routing.monitor" "monitor"
      "Lean and SDK monitors require matching task/poller routes, a typed routing receipt, and public Task Queue history.",
  ]

def productWorkflowOwnership : ModuleContract where
  identifier := Inventory.workflowOwnershipProductModule.identifier
  rank := 0
  owns := ["relation:workflow-task.attempt-epoch", "property:workflow-task.ownership-fencing"]
  obligations := [
    presentObligation "workflow-ownership.product" "product"
      "Two tasks, attempts, and ownership epochs are relational, executable, and fenced after rotation.",
    presentObligation "workflow-ownership.non-vacuity" "non-vacuity"
      "A stale owner is rejected before current-owner completion while the stale-completion mutation violates fencing.",
  ]

def systemWorkflowOwnership : ModuleContract where
  identifier := Inventory.workflowOwnershipSystemModule.identifier
  rank := 1
  owns := ["mechanism:workflow-ownership.task-history-observation"]
  obligations := [presentObligation "workflow-ownership.system" "mechanism"
    "Bootstrap, dispatch, failure, epoch rotation, stale rejection, completion, and redelivery are executable."]

def refinementWorkflowOwnership : ModuleContract where
  identifier := Inventory.workflowOwnershipRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "workflow-ownership.refinement" "refinement"
      "Every Workflow ownership observation step refines the product or stutters.",
    presentObligation "workflow-ownership.monitor" "monitor"
      "Lean and SDK monitors require typed fencing receipts and ordered failed-before-completed history.",
  ]

def productSpeculativeTask : ModuleContract where
  identifier := Inventory.speculativeTaskProductModule.identifier
  rank := 0
  owns := ["relation:workflow-task.requested-by-update", "property:workflow-task.speculative-creation"]
  obligations := [
    presentObligation "speculative-task.product" "product"
      "Two Workflow Tasks and Updates have explicit request, speculative, admitted, and committed relations.",
    presentObligation "speculative-task.non-vacuity" "non-vacuity"
      "An update-linked speculative task commits while an orphan task is rejected by the checked guard.",
  ]

def systemSpeculativeTask : ModuleContract where
  identifier := Inventory.speculativeTaskSystemModule.identifier
  rank := 1
  owns := ["mechanism:speculative-task.update-workflow-task"]
  obligations := [presentObligation "speculative-task.system" "mechanism"
    "Update request, speculative task creation, commit, and duplicate delivery are executable."]

def refinementSpeculativeTask : ModuleContract where
  identifier := Inventory.speculativeTaskRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "speculative-task.refinement" "refinement"
      "Every speculative task observation step refines the product or stutters.",
    presentObligation "speculative-task.monitor" "monitor"
      "Lean and SDK monitors require update-linked typed receipts and Update plus Workflow Task history.",
  ]

def productWorkflowProgress : ModuleContract where
  identifier := Inventory.workflowProgressProductModule.identifier
  rank := 0
  owns := ["relation:workflow-task.progresses-entity", "property:workflow-task.starvation",
    "property:entity.progress"]
  obligations := [
    presentObligation "workflow-progress.product" "product"
      "Two tasks, entities, and workers have explicit queue, availability, wait, dispatch, and completion relations.",
    presentObligation "workflow-progress.non-vacuity" "non-vacuity"
      "Bounded waiting reaches progress while an extra wait and a wrong-entity completion violate distinct properties.",
  ]

def systemWorkflowProgress : ModuleContract where
  identifier := Inventory.workflowProgressSystemModule.identifier
  rank := 1
  owns := ["mechanism:workflow-progress.task-history-observation"]
  obligations := [presentObligation "workflow-progress.system" "mechanism"
    "Task enqueue, worker availability, bounded wait, dispatch, completion, and redelivery are executable."]

def refinementWorkflowProgress : ModuleContract where
  identifier := Inventory.workflowProgressRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "workflow-progress.refinement" "refinement"
      "Every Workflow progress observation step refines the product or stutters.",
    presentObligation "workflow-progress.monitor" "monitor"
      "Lean and SDK monitors require typed progress receipts, ordered Workflow Task history, and terminal entity history.",
  ]

def nexusTarget : TargetProjection where
  identifier := "nexus-cancellation"
  modules := [productNexus.identifier, deliveryProvider.identifier, systemNexus.identifier,
    refinementNexus.identifier]
  properties := ["nexus.cancellation.won-excludes-success"]
  retainedActions := [
    "schedule-operation", "dispatch-task", "request-cancellation", "commit-cancellation",
    "acquire-ownership", "retry-task", "worker-returns-success", "persist-success",
    "task-delivery.environment-tick",
  ]
  omissions := [{
    identifier := "unselected-update-interference"
    reason := "The bounded Nexus target excludes independent Update product actions."
    maxCount := 100
  }]

def updateTarget : TargetProjection where
  identifier := "workflow-update-lifecycle"
  modules := [productUpdate.identifier, deliveryProvider.identifier, systemUpdate.identifier,
    refinementUpdate.identifier]
  properties := ["workflow-update.accepted-completes-through-history"]
  retainedActions := [
    "start-update", "dispatch-workflow-task", "accept-update", "record-update-history",
    "complete-workflow-task", "complete-update", "task-delivery.environment-tick",
  ]
  omissions := [{
    identifier := "unselected-nexus-interference"
    reason := "The bounded Update target excludes independent Nexus product actions."
    maxCount := 100
  }]

def taskAckTarget : TargetProjection where
  identifier := Product.TaskAck.declaration.target.identifier
  modules := [productTaskAck.identifier]
  properties := ["task-delivery.acknowledged-removes-backlog"]
  retainedActions := ["enqueue-workflow-task", "deliver-workflow-task", "acknowledge-workflow-task"]
  omissions := []

private def routingTarget : TargetProjection := {
  identifier := "foundation-routing-isolation"
  modules := [productWorkflowLineage.identifier, systemWorkflowLineage.identifier,
    refinementWorkflowLineage.identifier, productWorkflowRouting.identifier,
    systemWorkflowRouting.identifier, refinementWorkflowRouting.identifier]
  properties := [
    "workflow-task.routing-isolation",
    "workflow-run.continuation-lineage",
    "workflow-run.reset-lineage",
  ]
  retainedActions := ["route-workflow-task", "continue-workflow", "reset-workflow"]
  omissions := [{
    identifier := "foundation-routing-isolation.unselected-interference"
    reason := "Independent entity actions are bounded outside this focused target projection."
    maxCount := 100
  }]
}

def productNexusLifecycle : ModuleContract where
  identifier := Inventory.nexusLifecycleProductModule.identifier
  rank := 0
  owns := ["lifecycle:nexus-operation"]
  obligations := [
    presentObligation "nexus-lifecycle.product" "product"
      "All 17 Nexus lifecycle edges are executable, unique, and exported as the coverage denominator.",
    presentObligation "nexus-lifecycle.non-vacuity" "non-vacuity"
      "Direct and asynchronous settlement, retry, timeout, termination, and rejection edges are reachable in the model.",
  ]

private def ownershipTarget : TargetProjection := {
  identifier := "foundation-ownership-fencing"
  modules := [productWorkflowOwnership.identifier, systemWorkflowOwnership.identifier,
    refinementWorkflowOwnership.identifier]
  properties := ["workflow-task.ownership-fencing"]
  retainedActions := ["fence-workflow-owner"]
  omissions := [{
    identifier := "foundation-ownership-fencing.unselected-interference"
    reason := "Independent entity actions are bounded outside this focused target projection."
    maxCount := 100
  }]
}

private def speculativeTaskTarget : TargetProjection := {
  identifier := "feature-workflow-speculative-delivery"
  modules := [productSpeculativeTask.identifier, systemSpeculativeTask.identifier,
    refinementSpeculativeTask.identifier]
  properties := ["workflow-task.speculative-creation"]
  retainedActions := ["create-speculative-workflow-task", "commit-speculative-workflow-task"]
  omissions := [{
    identifier := "feature-workflow-speculative-delivery.unselected-interference"
    reason := "Independent entity actions are bounded outside this focused target projection."
    maxCount := 100
  }]
}

private def workflowProgressTarget (identifier property action : String) : TargetProjection := {
  identifier
  modules := [productWorkflowProgress.identifier, systemWorkflowProgress.identifier,
    refinementWorkflowProgress.identifier]
  properties := [property]
  retainedActions := [action]
  omissions := [{
    identifier := identifier ++ ".unselected-interference"
    reason := "Independent entity actions are bounded outside this focused target projection."
    maxCount := 100
  }]
}

def parityTargets : List TargetProjection := [
  {
    identifier := "feature-nexus"
    modules := [productNexusLifecycle.identifier, productNexusClosure.identifier, systemNexusClosure.identifier,
      refinementNexusClosure.identifier]
    properties := ["nexus-operation.closure"]
    retainedActions := [
      "schedule-operation", "dispatch-task", "worker-returns-success", "persist-success",
      "request-cancellation", "commit-cancellation", "acquire-ownership", "retry-task",
      "close-nexus-operation", "timeout-nexus-operation", "register-callback",
    ]
    omissions := [{
      identifier := "feature-nexus.unselected-interference"
      reason := "Independent entity actions are bounded outside this focused target projection."
      maxCount := 100
    }]
  },
  speculativeTaskTarget,
  workflowProgressTarget "foundation-delivery-safety" "entity.progress" "progress-entity",
  ownershipTarget,
  routingTarget,
  workflowProgressTarget "integration-activity-delivery" "entity.progress" "progress-entity",
  {
    identifier := "integration-callback-nexus"
    modules := [productCallbackReference.identifier, systemCallbackReference.identifier,
      refinementCallbackReference.identifier]
    properties := ["callback.reference-consistency"]
    retainedActions := ["register-callback"]
    omissions := [{
      identifier := "integration-callback-nexus.unselected-interference"
      reason := "Independent entity actions are bounded outside this focused target projection."
      maxCount := 100
    }]
  },
  {
    identifier := "integration-callback-workflow"
    modules := [productCallbackResponse.identifier, systemCallbackResponse.identifier,
      refinementCallbackResponse.identifier]
    properties := ["callback.response-consistency"]
    retainedActions := ["register-callback", "record-callback-response"]
    omissions := [{
      identifier := "integration-callback-workflow.unselected-interference"
      reason := "Independent entity actions are bounded outside this focused target projection."
      maxCount := 100
    }]
  },
  {
    identifier := "integration-nexus-activity"
    modules := [productNexusActivityLink.identifier, systemNexusActivityLink.identifier,
      refinementNexusActivityLink.identifier]
    properties := ["nexus-activity.link-consistency"]
    retainedActions := ["link-nexus-activity"]
    omissions := [{
      identifier := "integration-nexus-activity.unselected-interference"
      reason := "Independent entity actions are bounded outside this focused target projection."
      maxCount := 100
    }]
  },
  {
    identifier := "integration-nexus-timeout"
    modules := [productNexusTimeout.identifier, systemNexusTimeout.identifier,
      refinementNexusTimeout.identifier]
    properties := ["nexus-operation.timeout-semantics"]
    retainedActions := ["schedule-operation", "timeout-nexus-operation"]
    omissions := [{
      identifier := "integration-nexus-timeout.unselected-interference"
      reason := "Independent entity actions are bounded outside this focused target projection."
      maxCount := 100
    }]
  },
  workflowProgressTarget "integration-workflow-delivery" "workflow-task.starvation"
    "dispatch-assurance-workflow-task",
  {
    identifier := "protocol-atomic"
    modules := [productCallbackResponse.identifier, systemCallbackResponse.identifier,
      refinementCallbackResponse.identifier]
    properties := ["callback.response-consistency"]
    retainedActions := ["record-callback-response"]
    omissions := [{
      identifier := "protocol-atomic.unselected-interference"
      reason := "Independent entity actions are bounded outside this focused target projection."
      maxCount := 100
    }]
  },
]

def composition : Umpire3.Composition where
  modules := [deliveryProvider, productNexus, systemNexus, refinementNexus,
    productUpdate, systemUpdate, refinementUpdate, productTaskAck,
    productNexusLifecycle, productNexusClosure, systemNexusClosure, refinementNexusClosure,
    productNexusTimeout, systemNexusTimeout, refinementNexusTimeout,
    productNexusActivityLink, systemNexusActivityLink, refinementNexusActivityLink,
    productCallbackReference, systemCallbackReference, refinementCallbackReference,
    productCallbackResponse, systemCallbackResponse, refinementCallbackResponse,
    productWorkflowLineage, systemWorkflowLineage, refinementWorkflowLineage,
    productWorkflowRouting, systemWorkflowRouting, refinementWorkflowRouting,
    productWorkflowOwnership, systemWorkflowOwnership, refinementWorkflowOwnership,
    productSpeculativeTask, systemSpeculativeTask, refinementSpeculativeTask,
    productWorkflowProgress, systemWorkflowProgress, refinementWorkflowProgress]
  targets := [nexusTarget, updateTarget, taskAckTarget] ++ parityTargets

set_option maxRecDepth 100000 in
theorem compositionMetadataValid : composition.MetadataValid := by
  change composition.metadataValid = true
  decide

def weakenedDeliveryProvider : ModuleContract := {
  deliveryProvider with
  provides := [{
    identifier := System.TaskDelivery.guarantee.identifier
    statementHash := "sha256:weakened"
  }]
}

def weakenedComposition : Umpire3.Composition := {
  composition with
  modules := weakenedDeliveryProvider :: composition.modules.tail
}

example : weakenedComposition.metadataValid = false := by decide

end Umpire3.Temporal.Composition
