import Temporal.Families.TaskAcknowledgement.Feature
import Temporal.Families.TaskAcknowledgement.Refinement
import Temporal.Families.NexusClosure.Lifecycle
import Temporal.Inventory
import Temporal.Families
import Umpire3.Composition

namespace Umpire3.Temporal.Composition

private def presentObligation (identifier kind detail : String) : ModelObligation := {
  identifier, kind, detail, status := "metadata-present"
}

def deliveryProvider : ModuleContract where
  identifier := "Temporal.Mechanisms.TaskDelivery"
  rank := 0
  owns := ["relation:task-delivery.current-completion"]
  provides := [ContractGuarantee.ofGuarantee Mechanisms.TaskDelivery.guarantee]
  interferenceActions := ["task-delivery.environment-tick"]
  obligations := [presentObligation "task-delivery.guarantee" "guarantee"
    "Current-owner completion is versioned and proved against stale completion."]

private def updateDeliveryRequirement : ContractRequirement :=
  ContractRequirement.ofRequirement deliveryProvider.identifier Mechanisms.TaskDelivery.guarantee
    System.UpdateLifecycle.deliveryRequirement

private def workflowOwnershipDeliveryRequirement : ContractRequirement :=
  ContractRequirement.ofRequirement deliveryProvider.identifier Mechanisms.TaskDelivery.guarantee
    System.WorkflowOwnership.deliveryRequirement

structure DeliveryProjection (Epoch : Type) where
  completionEpoch : Option Epoch
  ownerEpoch : Epoch
  environmentVersion : Nat

def DeliveryProjection.Current {Epoch : Type} (state : DeliveryProjection Epoch) : Prop :=
  Mechanisms.TaskDelivery.CurrentCompletionOf state.completionEpoch state.ownerEpoch

def DeliveryEnvironmentInterference {Epoch : Type}
    (before after : DeliveryProjection Epoch) : Prop :=
  after.completionEpoch = before.completionEpoch ∧ after.ownerEpoch = before.ownerEpoch

theorem deliveryInterferencePreservesCurrentCompletion {Epoch : Type}
    {before after : DeliveryProjection Epoch}
    (interference : DeliveryEnvironmentInterference before after)
    (current : before.Current) : after.Current := by
  rcases interference with ⟨completionUnchanged, ownerUnchanged⟩
  simp only [DeliveryProjection.Current]
  rw [completionUnchanged, ownerUnchanged]
  exact current

def SharedDeliveryComposition : Prop :=
  Mechanisms.TaskDelivery.guarantee.Claim ∧
  Mechanisms.TaskDelivery.guarantee.Claim ∧
  (∀ {Epoch : Type} {before after : DeliveryProjection Epoch},
    DeliveryEnvironmentInterference before after → before.Current → after.Current)

theorem sharedDeliveryCompositionSound : SharedDeliveryComposition :=
  ⟨System.UpdateLifecycle.deliveryRequirement.proof,
    System.WorkflowOwnership.deliveryRequirement.proof,
    deliveryInterferencePreservesCurrentCompletion⟩

def featureNexus : ModuleContract where
  identifier := "Temporal.Feature.NexusCancellationFencing"
  rank := 0
  owns := ["entity:nexus-operation", "property:nexus.cancellation.won-excludes-success"]
  obligations := [presentObligation "nexus.feature" "feature"
    "Feature lifecycle, executable transition relation, and safety theorem are checked."]

def systemNexus : ModuleContract where
  identifier := "Temporal.System.NexusCancellationFencing"
  rank := 1
  owns := ["mechanism:nexus-task-delivery"]
  obligations := [presentObligation "nexus.system" "mechanism"
    "Task ownership and stale completion mechanism is executable."]

def refinementNexus : ModuleContract where
  identifier := "Temporal.Refinement.NexusCancellationFencing"
  rank := 2
  obligations := [presentObligation "nexus.refinement" "refinement"
    "Every system step refines the Nexus feature or stutters."]

def featureUpdate : ModuleContract where
  identifier := "Temporal.Feature.UpdateLifecycle"
  rank := 0
  owns := ["entity:workflow-update", "property:workflow-update.accepted-completes-through-history"]
  obligations := [presentObligation "update.feature" "feature"
    "Update lifecycle and completion stability are checked."]

def systemUpdate : ModuleContract where
  identifier := "Temporal.System.UpdateLifecycle"
  rank := 1
  owns := ["mechanism:update-task-delivery"]
  requires := [updateDeliveryRequirement]
  interferenceActions := deliveryProvider.interferenceActions
  obligations := [presentObligation "update.system" "mechanism"
    "The independent Update task and history mechanism is exactly executable."]

def refinementUpdate : ModuleContract where
  identifier := "Temporal.Refinement.UpdateLifecycle"
  rank := 2
  obligations := [presentObligation "update.refinement" "refinement"
    "Every independent Update system step refines the history-backed feature or stutters."]

def featureTaskAck : ModuleContract where
  identifier := Feature.TaskAck.declaration.module.identifier
  rank := 0
  owns := ["entity:workflow-task", "property:task-delivery.acknowledged-removes-backlog"]
  obligations := [
    presentObligation "task-ack.feature" "feature"
      "The lifecycle, executable equivalence, and acknowledgement theorem are checked.",
    presentObligation "task-ack.live-realization" "realization"
      "The public Workflow Task protocol adapter realizes enqueue, delivery, acknowledgement, and cleanup.",
  ]

def systemTaskAck : ModuleContract where
  identifier := "Temporal.System.TaskAck"
  rank := 1
  owns := ["mechanism:workflow-task-acknowledgement"]
  obligations := [
    presentObligation "task-ack.protocol-system" "mechanism"
      "The message delivery and completion-storage mechanism is independently executable.",
    presentObligation "task-ack.history-system" "mechanism"
      "The public-history observation mechanism is independently executable.",
  ]

def refinementTaskAck : ModuleContract where
  identifier := "Temporal.Refinement.TaskAck"
  rank := 2
  obligations := [
    presentObligation "task-ack.protocol-refinement" "refinement"
      "The protocol mechanism refines the acknowledgement contract.",
    presentObligation "task-ack.history-refinement" "refinement"
      "The history-observation mechanism independently refines the same contract.",
    presentObligation "task-ack.mutation" "non-vacuity"
      "Both mechanisms have an executable backlog-retention mutation that breaks refinement.",
  ]

def featureNexusClosure : ModuleContract where
  identifier := Inventory.nexusClosureFeatureModule.identifier
  rank := 0
  owns := ["relation:nexus-operation.caller-workflow", "property:nexus-operation.closure"]
  obligations := [
    presentObligation "nexus-closure.feature" "feature"
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
      "Every system transition refines the relational feature or stutters.",
    presentObligation "nexus-closure.monitor" "monitor"
      "The generated monitor is equivalent to the feature predicate and the SDK adapter orders operation and caller terminal history.",
  ]

def featureNexusProgress : ModuleContract where
  identifier := Inventory.nexusProgressFeatureModule.identifier
  rank := 0
  owns := ["property:nexus-operation.progress"]
  obligations := [
    presentObligation "nexus-progress.feature" "feature"
      "Retryable failure, a bounded wait, settlement, and deadline expiry are executable and safe.",
    presentObligation "nexus-progress.non-vacuity" "non-vacuity"
      "A retrying operation can settle while a second bounded wait violates the progress property.",
  ]

def systemNexusProgress : ModuleContract where
  identifier := Inventory.nexusProgressSystemModule.identifier
  rank := 1
  owns := ["mechanism:nexus-operation.retry-progress"]
  obligations := [presentObligation "nexus-progress.system" "mechanism"
    "Scheduling, retryable handler failure, deadline passage, and settlement are independently executable."]

def refinementNexusProgress : ModuleContract where
  identifier := Inventory.nexusProgressRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "nexus-progress.refinement" "refinement"
      "Every independent progress step refines the bounded feature transition.",
    presentObligation "nexus-progress.monitor" "monitor"
      "The generated monitor requires typed retry and closed-deadline evidence from the live adapter.",
  ]

def featureNexusTimeout : ModuleContract where
  identifier := Inventory.nexusTimeoutFeatureModule.identifier
  rank := 0
  owns := ["relation:nexus-operation.timeout-evidence", "property:nexus-operation.timeout-semantics"]
  obligations := [
    presentObligation "nexus-timeout.feature" "feature"
      "Configured operations, timeout kinds, failure metadata, evidence identity, and validity are relational and executable.",
    presentObligation "nexus-timeout.non-vacuity" "non-vacuity"
      "A valid timeout reaches the antecedent while wrong kind or message mutations are rejected by the weakened guard model.",
  ]

def systemNexusTimeout : ModuleContract where
  identifier := Inventory.nexusTimeoutSystemModule.identifier
  rank := 1
  owns := ["mechanism:nexus-timeout.history-observation"]
  obligations := [presentObligation "nexus-timeout.system" "mechanism"
    "Ordered and duplicate history observations execute independently of the feature contract."]

def refinementNexusTimeout : ModuleContract where
  identifier := Inventory.nexusTimeoutRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "nexus-timeout.refinement" "refinement"
      "Every timeout observation step refines the feature or stutters.",
    presentObligation "nexus-timeout.monitor" "monitor"
      "Lean and SDK monitors require start-to-close kind and operation-timeout failure metadata.",
  ]

def featureNexusActivityLink : ModuleContract where
  identifier := Inventory.nexusActivityLinkFeatureModule.identifier
  rank := 0
  owns := ["relation:nexus-operation.links-activity", "property:nexus-activity.link-consistency"]
  obligations := [
    presentObligation "nexus-activity-link.feature" "feature"
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
      "Every link observation step refines the reciprocal feature or stutters.",
    presentObligation "nexus-activity-link.monitor" "monitor"
      "Lean and SDK monitors require independently observed forward and reverse public links.",
  ]

def featureCallbackReference : ModuleContract where
  identifier := Inventory.callbackReferenceFeatureModule.identifier
  rank := 0
  owns := ["relation:callback-operation-handler-reference", "property:callback.reference-consistency"]
  obligations := [
    presentObligation "callback-reference.feature" "feature"
      "Two callback, operation, and handler identities plus reference kind, value, and causal position are relational and executable.",
    presentObligation "callback-reference.non-vacuity" "non-vacuity"
      "Matching event and request references are admitted while mismatched handler, value, kind, order, and malformed evidence are rejected.",
  ]

def systemCallbackReference : ModuleContract where
  identifier := Inventory.callbackReferenceSystemModule.identifier
  rank := 1
  owns := ["mechanism:callback-reference.history-observation"]
  obligations := [presentObligation "callback-reference.system" "mechanism"
    "Attachment, Nexus start, and duplicate attachment observations execute independently of the feature contract."]

def refinementCallbackReference : ModuleContract where
  identifier := Inventory.callbackReferenceRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "callback-reference.refinement" "refinement"
      "Every callback reference observation step refines the feature or stutters.",
    presentObligation "callback-reference.monitor" "monitor"
      "Lean and live SDK monitors require a mechanism-qualified callback receipt and matching public history evidence.",
  ]

def featureCallbackResponse : ModuleContract where
  identifier := Inventory.callbackResponseFeatureModule.identifier
  rank := 0
  owns := ["relation:callback-delivery-response", "property:callback.response-consistency"]
  obligations := [
    presentObligation "callback-response.feature" "feature"
      "Two callback, operation, and delivery identities plus accepted response fingerprints and causal settlement order are relational and executable.",
    presentObligation "callback-response.non-vacuity" "non-vacuity"
      "Idempotent duplicates and terminal late responses are admitted while conflicting responses and non-terminal late responses are rejected.",
  ]

def systemCallbackResponse : ModuleContract where
  identifier := Inventory.callbackResponseSystemModule.identifier
  rank := 1
  owns := ["mechanism:callback-response.delivery-observation"]
  obligations := [presentObligation "callback-response.system" "mechanism"
    "Registration, settlement, response, and duplicate response observations execute independently of the feature contract."]

def refinementCallbackResponse : ModuleContract where
  identifier := Inventory.callbackResponseRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "callback-response.refinement" "refinement"
      "Every callback response observation step refines the feature or stutters.",
    presentObligation "callback-response.monitor" "monitor"
      "Lean and live SDK monitors require independently qualified registration and response receipts with terminal history.",
  ]

def featureWorkflowLineage : ModuleContract where
  identifier := Inventory.workflowLineageFeatureModule.identifier
  rank := 0
  owns := ["relation:workflow-run.continues-as", "property:workflow-run.continuation-lineage",
    "property:workflow-run.reset-lineage"]
  obligations := [
    presentObligation "workflow-lineage.feature" "feature"
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
      "Every Workflow lineage observation step refines the feature or stutters.",
    presentObligation "workflow-lineage.monitor" "monitor"
      "Lean and SDK monitors agree on continuation and reset lineage and require typed live mechanism receipts.",
  ]

def featureWorkflowRouting : ModuleContract where
  identifier := Inventory.workflowRoutingFeatureModule.identifier
  rank := 0
  owns := ["relation:workflow-task.route", "relation:poller.route",
    "property:workflow-task.routing-isolation"]
  obligations := [
    presentObligation "workflow-routing.feature" "feature"
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
      "Every Workflow routing observation step refines the feature or stutters.",
    presentObligation "workflow-routing.monitor" "monitor"
      "Lean and SDK monitors require matching task/poller routes, a typed routing receipt, and public Task Queue history.",
  ]

def featureRoutingIsolation : ModuleContract where
  identifier := Inventory.routingIsolationFeatureModule.identifier
  rank := 0
  obligations := [
    presentObligation "routing-isolation.feature" "feature"
      "Workflow lineage and Task Queue routing form one feature boundary for the routing-isolation target.",
  ]

def systemRoutingIsolation : ModuleContract where
  identifier := Inventory.routingIsolationSystemModule.identifier
  rank := 1
  obligations := [
    presentObligation "routing-isolation.system" "mechanism"
      "Workflow lineage and Task Queue routing observations execute independently of the feature contract.",
  ]

def refinementRoutingIsolation : ModuleContract where
  identifier := Inventory.routingIsolationRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "routing-isolation.refinement" "refinement"
      "The independent lineage and routing systems refine their combined feature boundary.",
  ]

def featureWorkflowOwnership : ModuleContract where
  identifier := Inventory.workflowOwnershipFeatureModule.identifier
  rank := 0
  owns := ["relation:workflow-task.attempt-epoch", "property:workflow-task.ownership-fencing"]
  obligations := [
    presentObligation "workflow-ownership.feature" "feature"
      "Two tasks, attempts, and ownership epochs are relational, executable, and fenced after rotation.",
    presentObligation "workflow-ownership.non-vacuity" "non-vacuity"
      "A stale owner is rejected before current-owner completion while the stale-completion mutation violates fencing.",
  ]

def systemWorkflowOwnership : ModuleContract where
  identifier := Inventory.workflowOwnershipSystemModule.identifier
  rank := 1
  owns := ["mechanism:workflow-ownership.task-history-observation"]
  requires := [workflowOwnershipDeliveryRequirement]
  interferenceActions := deliveryProvider.interferenceActions
  obligations := [presentObligation "workflow-ownership.system" "mechanism"
    "Independent dispatch, failure, epoch rotation, stale rejection, and completion are exactly executable."]

def refinementWorkflowOwnership : ModuleContract where
  identifier := Inventory.workflowOwnershipRefinementModule.identifier
  rank := 2
  obligations := [
    presentObligation "workflow-ownership.refinement" "refinement"
      "Every Workflow ownership observation step refines the feature or stutters.",
    presentObligation "workflow-ownership.monitor" "monitor"
      "Lean and SDK monitors require typed fencing receipts and ordered failed-before-completed history.",
  ]

def featureSpeculativeTask : ModuleContract where
  identifier := Inventory.speculativeTaskFeatureModule.identifier
  rank := 0
  owns := ["relation:workflow-task.requested-by-update", "property:workflow-task.speculative-creation"]
  obligations := [
    presentObligation "speculative-task.feature" "feature"
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
      "Every speculative task observation step refines the feature or stutters.",
    presentObligation "speculative-task.monitor" "monitor"
      "Lean and SDK monitors require update-linked typed receipts and Update plus Workflow Task history.",
  ]

def featureWorkflowProgress : ModuleContract where
  identifier := Inventory.workflowProgressFeatureModule.identifier
  rank := 0
  owns := ["relation:workflow-task.progresses-entity", "property:workflow-task.starvation",
    "property:entity.progress"]
  obligations := [
    presentObligation "workflow-progress.feature" "feature"
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
      "Every Workflow progress observation step refines the feature or stutters.",
    presentObligation "workflow-progress.monitor" "monitor"
      "Lean and SDK monitors require typed progress receipts, ordered Workflow Task history, and terminal entity history.",
  ]

def nexusTarget : TargetProjection where
  identifier := "nexus-cancellation"
  modules := [featureNexus.identifier, deliveryProvider.identifier, systemNexus.identifier,
    refinementNexus.identifier]
  properties := ["nexus.cancellation.won-excludes-success"]
  retainedActions := [
    "schedule-operation", "dispatch-task", "request-cancellation", "commit-cancellation",
    "acquire-ownership", "retry-task", "worker-returns-success", "persist-success",
    "task-delivery.environment-tick",
  ]
  omissions := [{
    identifier := "unselected-update-interference"
    reason := "The bounded Nexus target excludes independent Update feature actions."
    maxCount := 100
  }]

def updateTarget : TargetProjection where
  identifier := "workflow-update-lifecycle"
  modules := [featureUpdate.identifier, deliveryProvider.identifier, systemUpdate.identifier,
    refinementUpdate.identifier]
  properties := ["workflow-update.accepted-completes-through-history"]
  retainedActions := [
    "start-update", "dispatch-workflow-task", "accept-update", "record-update-history",
    "complete-workflow-task", "complete-update", "task-delivery.environment-tick",
  ]
  omissions := [{
    identifier := "unselected-nexus-interference"
    reason := "The bounded Update target excludes independent Nexus feature actions."
    maxCount := 100
  }]

def taskAckTarget : TargetProjection where
  identifier := Feature.TaskAck.declaration.target.identifier
  modules := [featureTaskAck.identifier, systemTaskAck.identifier, refinementTaskAck.identifier]
  properties := ["task-delivery.acknowledged-removes-backlog"]
  retainedActions := ["enqueue-workflow-task", "deliver-workflow-task", "acknowledge-workflow-task"]
  omissions := []

private def routingTarget : TargetProjection := {
  identifier := "foundation-routing-isolation"
  modules := [featureRoutingIsolation.identifier, systemRoutingIsolation.identifier,
    refinementRoutingIsolation.identifier]
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

def featureNexusLifecycle : ModuleContract where
  identifier := Inventory.nexusLifecycleFeatureModule.identifier
  rank := 0
  owns := ["lifecycle:nexus-operation"]
  obligations := [
    presentObligation "nexus-lifecycle.feature" "feature"
      "All 17 Nexus lifecycle edges are executable, unique, and exported as the coverage denominator.",
    presentObligation "nexus-lifecycle.non-vacuity" "non-vacuity"
      "Direct and asynchronous settlement, retry, timeout, termination, and rejection edges are reachable in the model.",
  ]

private def ownershipTarget : TargetProjection := {
  identifier := "foundation-ownership-fencing"
  modules := [featureWorkflowOwnership.identifier, deliveryProvider.identifier,
    systemWorkflowOwnership.identifier,
    refinementWorkflowOwnership.identifier]
  properties := ["workflow-task.ownership-fencing"]
  retainedActions := ["fence-workflow-owner", "task-delivery.environment-tick"]
  omissions := [{
    identifier := "foundation-ownership-fencing.unselected-interference"
    reason := "Independent entity actions are bounded outside this focused target projection."
    maxCount := 100
  }]
}

private def speculativeTaskTarget : TargetProjection := {
  identifier := "feature-workflow-speculative-delivery"
  modules := [featureSpeculativeTask.identifier, systemSpeculativeTask.identifier,
    refinementSpeculativeTask.identifier]
  properties := ["workflow-task.speculative-creation"]
  retainedActions := ["create-speculative-workflow-task", "commit-speculative-workflow-task"]
  omissions := [{
    identifier := "feature-workflow-speculative-delivery.unselected-interference"
    reason := "Independent entity actions are bounded outside this focused target projection."
    maxCount := 100
  }]
}

private def workflowProgressTarget (identifier property : String)
    (actions : List String) : TargetProjection := {
  identifier
  modules := [featureWorkflowProgress.identifier, systemWorkflowProgress.identifier,
    refinementWorkflowProgress.identifier]
  properties := [property]
  retainedActions := actions
  omissions := [{
    identifier := identifier ++ ".unselected-interference"
    reason := "Independent entity actions are bounded outside this focused target projection."
    maxCount := 100
  }]
}

def parityTargets : List TargetProjection := [
  {
    identifier := "feature-nexus"
    modules := [featureNexusLifecycle.identifier, featureNexusClosure.identifier, systemNexusClosure.identifier,
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
  {
    identifier := "feature-nexus-progress"
    modules := [featureNexusProgress.identifier, systemNexusProgress.identifier,
      refinementNexusProgress.identifier]
    properties := ["nexus-operation.progress"]
    retainedActions := ["close-nexus-operation"]
    omissions := [{
      identifier := "feature-nexus-progress.unselected-interference"
      reason := "Independent entity actions are bounded outside this focused target projection."
      maxCount := 100
    }]
  },
  speculativeTaskTarget,
  workflowProgressTarget "foundation-delivery-safety" "entity.progress"
    ["crash-owner", "progress-entity", "recover-owner"],
  ownershipTarget,
  routingTarget,
  workflowProgressTarget "integration-activity-delivery" "entity.progress"
    ["crash-owner", "progress-entity", "recover-owner"],
  {
    identifier := "integration-callback-nexus"
    modules := [featureCallbackReference.identifier, systemCallbackReference.identifier,
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
    modules := [featureCallbackResponse.identifier, systemCallbackResponse.identifier,
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
    modules := [featureNexusActivityLink.identifier, systemNexusActivityLink.identifier,
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
    modules := [featureNexusTimeout.identifier, systemNexusTimeout.identifier,
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
    ["dispatch-assurance-workflow-task"],
  {
    identifier := "protocol-atomic"
    modules := [featureCallbackResponse.identifier, systemCallbackResponse.identifier,
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
  proof := resolved_theorem% sharedDeliveryCompositionSound
  modules := [deliveryProvider, featureNexus, systemNexus, refinementNexus,
    featureUpdate, systemUpdate, refinementUpdate, featureTaskAck, systemTaskAck, refinementTaskAck,
    featureNexusLifecycle, featureNexusClosure, systemNexusClosure, refinementNexusClosure,
    featureNexusProgress, systemNexusProgress, refinementNexusProgress,
    featureNexusTimeout, systemNexusTimeout, refinementNexusTimeout,
    featureNexusActivityLink, systemNexusActivityLink, refinementNexusActivityLink,
    featureCallbackReference, systemCallbackReference, refinementCallbackReference,
    featureCallbackResponse, systemCallbackResponse, refinementCallbackResponse,
    featureWorkflowLineage, systemWorkflowLineage, refinementWorkflowLineage,
    featureWorkflowRouting, systemWorkflowRouting, refinementWorkflowRouting,
    featureRoutingIsolation, systemRoutingIsolation, refinementRoutingIsolation,
    featureWorkflowOwnership, systemWorkflowOwnership, refinementWorkflowOwnership,
    featureSpeculativeTask, systemSpeculativeTask, refinementSpeculativeTask,
    featureWorkflowProgress, systemWorkflowProgress, refinementWorkflowProgress]
  targets := [nexusTarget, updateTarget, taskAckTarget] ++ parityTargets

set_option maxRecDepth 100000 in
theorem compositionMetadataValid : composition.MetadataValid := by
  change composition.metadataValid = true
  decide

def weakenedDeliveryProvider : ModuleContract := {
  deliveryProvider with
  provides := [{ ContractGuarantee.ofGuarantee Mechanisms.TaskDelivery.guarantee with
    theoremName := "weakened" }]
}

def weakenedComposition : Umpire3.Composition := {
  composition with
  modules := weakenedDeliveryProvider :: composition.modules.tail
}

example : weakenedComposition.metadataValid = false := by decide

end Umpire3.Temporal.Composition
