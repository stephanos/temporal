import Temporal.Families
import Temporal.Families.NexusProgress.System
import Temporal.Families.TaskAcknowledgement.System
import Umpire3.FiniteReplayView

namespace Umpire3.Temporal.Targets.FiniteReplay

private def limits : Exact.Limits where
  maxDepth := 16
  maxStates := 64
  maxTransitions := 512
  maxStateBytes := 4096
  maxWork := 1024

private def outcomes : List ActionOutcome := [
  .applied, .suppressed, .rejected, .retried, .faultIntercepted,
]

private def attempt (action : String) (transitions : List String := []) : FiniteReplayAttempt where
  action := action
  outcomes := outcomes
  appliedPaths := [transitions]

private def attemptPaths (action : String) (paths : List (List String)) : FiniteReplayAttempt where
  action := action
  outcomes := outcomes
  appliedPaths := paths

private def artifact (target property canonicalModel : String)
    (attempts : List FiniteReplayAttempt) : FiniteReplayArtifact where
  target := target
  property := property
  world := "smoke"
  variant := "sound"
  canonicalModel := canonicalModel
  attempts := attempts

private def ownershipActionName : Umpire3.Temporal.System.WorkflowOwnership.Action → String
  | .dispatchCurrent => "dispatch-current"
  | .failCurrent => "fail-current"
  | .rotateOwner => "rotate-owner"
  | .rejectStale => "reject-stale"
  | .completeCurrent => "complete-current"
  | .completeStale => "complete-stale"

private def ownershipFinite : FiniteView Umpire3.Temporal.System.WorkflowOwnership.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.WorkflowOwnership.executable inferInstance
    ownershipActionName (by
      intro left right
      cases left <;> cases right <;> simp_all [ownershipActionName])

def ownership : FiniteReplayView Umpire3.Temporal.System.WorkflowOwnership.behavior () where
  artifact := artifact "foundation-ownership-fencing" "workflow-task.ownership-fencing"
    "Umpire3.Temporal.System.WorkflowOwnership.behavior" [
      attempt "fence-workflow-owner" [
        "dispatch-current", "fail-current", "rotate-owner", "reject-stale", "complete-current",
      ],
      attempt "task-delivery.environment-tick",
    ]
  finite := ownershipFinite
  property := fun state => decide (state ≠ .staleCompleted)
  valid := by decide

private def lineageActionName : Umpire3.Temporal.System.WorkflowLineage.Action → String
  | .observeContinuation => "observe-continuation"
  | .observeReset => "observe-reset"
  | .observeInvalidContinuation => "observe-invalid-continuation"

private def lineageFinite : FiniteView Umpire3.Temporal.System.WorkflowLineage.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.WorkflowLineage.executable inferInstance
    lineageActionName (by
      intro left right
      cases left <;> cases right <;> simp_all [lineageActionName])

private def lineageArtifact (property : String) : FiniteReplayArtifact :=
  artifact "foundation-routing-isolation" property
    "Umpire3.Temporal.System.WorkflowLineage.behavior" [
      attempt "route-workflow-task",
      attempt "continue-workflow" ["observe-continuation"],
      attempt "reset-workflow" ["observe-reset"],
    ]

def continuationLineage : FiniteReplayView Umpire3.Temporal.System.WorkflowLineage.behavior () where
  artifact := lineageArtifact "workflow-run.continuation-lineage"
  finite := lineageFinite
  property := fun state => decide (state ≠ .invalidContinuation)
  valid := by decide

def resetLineage : FiniteReplayView Umpire3.Temporal.System.WorkflowLineage.behavior () where
  artifact := lineageArtifact "workflow-run.reset-lineage"
  finite := lineageFinite
  property := fun state => decide (state ≠ .invalidContinuation)
  valid := by decide

private def routingActionName : Umpire3.Temporal.System.WorkflowRouting.Action → String
  | .assignTask => "assign-task"
  | .registerMatchingPoller => "register-matching-poller"
  | .registerCrossingPoller => "register-crossing-poller"
  | .reserveMatching => "reserve-matching"
  | .reserveCrossing => "reserve-crossing"

private def routingFinite : FiniteView Umpire3.Temporal.System.WorkflowRouting.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.WorkflowRouting.executable inferInstance
    routingActionName (by
      intro left right
      cases left <;> cases right <;> simp_all [routingActionName])

def routing : FiniteReplayView Umpire3.Temporal.System.WorkflowRouting.behavior () where
  artifact := artifact "foundation-routing-isolation" "workflow-task.routing-isolation"
    "Umpire3.Temporal.System.WorkflowRouting.behavior" [
      attemptPaths "route-workflow-task" [
        ["assign-task", "register-matching-poller", "reserve-matching"],
        ["assign-task", "register-crossing-poller"],
      ],
      attempt "continue-workflow",
      attempt "reset-workflow",
    ]
  finite := routingFinite
  property := fun state => decide (state ≠ .crossingReservation)
  valid := by decide

private def speculativeActionName : Umpire3.Temporal.System.SpeculativeTask.Action → String
  | .requestUpdate => "request-update"
  | .createTask => "create-task"
  | .commitTask => "commit-task"
  | .createOrphan => "create-orphan"

private def speculativeFinite : FiniteView Umpire3.Temporal.System.SpeculativeTask.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.SpeculativeTask.executable inferInstance
    speculativeActionName (by
      intro left right
      cases left <;> cases right <;> simp_all [speculativeActionName])

def speculative : FiniteReplayView Umpire3.Temporal.System.SpeculativeTask.behavior () where
  artifact := artifact "feature-workflow-speculative-delivery" "workflow-task.speculative-creation"
    "Umpire3.Temporal.System.SpeculativeTask.behavior" [
      attempt "create-speculative-workflow-task" ["request-update", "create-task"],
      attempt "commit-speculative-workflow-task" ["commit-task"],
    ]
  finite := speculativeFinite
  property := fun state => decide (state ≠ .orphanedTask)
  valid := by decide

private def callbackReferenceActionName : Umpire3.Temporal.System.CallbackReference.Action → String
  | .observeAttachment => "observe-attachment"
  | .observeMatchingOperation => "observe-matching-operation"
  | .observeWrongOperation => "observe-wrong-operation"

private def callbackReferenceFinite : FiniteView Umpire3.Temporal.System.CallbackReference.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.CallbackReference.executable inferInstance
    callbackReferenceActionName (by
      intro left right
      cases left <;> cases right <;> simp_all [callbackReferenceActionName])

def callbackReference : FiniteReplayView Umpire3.Temporal.System.CallbackReference.behavior () where
  artifact := artifact "integration-callback-nexus" "callback.reference-consistency"
    "Umpire3.Temporal.System.CallbackReference.behavior" [
      attempt "register-callback" ["observe-attachment", "observe-matching-operation"],
    ]
  finite := callbackReferenceFinite
  property := fun state => decide (state ≠ .wrongOperationObserved)
  valid := by decide

private def callbackResponseActionName : Umpire3.Temporal.System.CallbackResponse.Action → String
  | .register => "register"
  | .settle => "settle"
  | .respond => "respond"
  | .conflict => "conflict"

private def callbackResponseFinite : FiniteView Umpire3.Temporal.System.CallbackResponse.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.CallbackResponse.executable inferInstance
    callbackResponseActionName (by
      intro left right
      cases left <;> cases right <;> simp_all [callbackResponseActionName])

private def callbackResponseArtifact (target : String)
    (attempts : List FiniteReplayAttempt) : FiniteReplayArtifact :=
  artifact target "callback.response-consistency"
    "Umpire3.Temporal.System.CallbackResponse.behavior" attempts

def callbackResponse : FiniteReplayView Umpire3.Temporal.System.CallbackResponse.behavior () where
  artifact := callbackResponseArtifact "integration-callback-workflow" [
    attempt "register-callback" ["register", "settle"],
    attempt "record-callback-response" ["respond"],
  ]
  finite := callbackResponseFinite
  property := fun state => decide (state ≠ .conflictingResponse)
  valid := by decide

def atomicCallbackResponse : FiniteReplayView Umpire3.Temporal.System.CallbackResponse.behavior () where
  artifact := callbackResponseArtifact "protocol-atomic" [
    attempt "record-callback-response" ["register", "settle", "respond"],
  ]
  finite := callbackResponseFinite
  property := fun state => decide (state ≠ .conflictingResponse)
  valid := by decide

private def timeoutActionName : Umpire3.Temporal.System.NexusTimeout.Action → String
  | .configure => "configure-timeout"
  | .recordTimeout => "record-timeout"
  | .recordMalformedTimeout => "record-malformed-timeout"

private def timeoutFinite : FiniteView Umpire3.Temporal.System.NexusTimeout.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.NexusTimeout.executable inferInstance timeoutActionName (by
    intro left right
    cases left <;> cases right <;> simp_all [timeoutActionName])

def timeout : FiniteReplayView Umpire3.Temporal.System.NexusTimeout.behavior () where
  artifact := artifact "integration-nexus-timeout" "nexus-operation.timeout-semantics"
    "Umpire3.Temporal.System.NexusTimeout.behavior" [
      attempt "schedule-operation" ["configure-timeout"],
      attempt "timeout-nexus-operation" ["record-timeout"],
    ]
  finite := timeoutFinite
  property := fun state => decide (state ≠ .malformedTimeout)
  valid := by decide

private def closureActionName : Umpire3.Temporal.System.NexusClosure.Action → String
  | .schedule => "schedule"
  | .start => "start"
  | .settle => "settle"
  | .close => "close"
  | .closeWhileRunning => "close-while-running"

private def closureFinite : FiniteView Umpire3.Temporal.System.NexusClosure.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.NexusClosure.executable inferInstance closureActionName (by
    intro left right
    cases left <;> cases right <;> simp_all [closureActionName])

def closure : FiniteReplayView Umpire3.Temporal.System.NexusClosure.behavior () where
  artifact := artifact "feature-nexus" "nexus-operation.closure"
    "Umpire3.Temporal.System.NexusClosure.behavior" [
      attempt "schedule-operation" ["schedule"],
      attempt "dispatch-task" ["start"],
      attempt "worker-returns-success",
      attempt "persist-success" ["settle"],
      attempt "request-cancellation",
      attempt "commit-cancellation",
      attempt "acquire-ownership",
      attempt "retry-task",
      attemptPaths "close-nexus-operation" [
        ["schedule", "start", "settle", "close"],
        ["start", "settle", "close"],
        ["settle", "close"],
        ["close"],
      ],
      attempt "timeout-nexus-operation",
      attempt "register-callback",
    ]
  finite := closureFinite
  property := fun state => decide (state ≠ .closedWhileRunning)
  valid := by decide

private def nexusProgressActionName : Umpire3.Temporal.System.NexusProgress.Action → String
  | .schedule => "schedule"
  | .failRetryably => "fail-retryably"
  | .elapseWithinDeadline => "elapse-within-deadline"
  | .settle => "settle"
  | .exceedDeadline => "exceed-deadline"

private def nexusProgressFinite : FiniteView Umpire3.Temporal.System.NexusProgress.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.NexusProgress.executable inferInstance
    nexusProgressActionName (by
      intro left right
      cases left <;> cases right <;> simp_all [nexusProgressActionName])

def nexusProgress : FiniteReplayView Umpire3.Temporal.System.NexusProgress.behavior () where
  artifact := artifact "feature-nexus-progress" "nexus-operation.progress"
    "Umpire3.Temporal.System.NexusProgress.behavior" [
      attempt "close-nexus-operation" [
        "schedule", "fail-retryably", "elapse-within-deadline", "settle",
      ],
    ]
  finite := nexusProgressFinite
  property := fun state => decide (state ≠ .stuckAfterDeadline)
  valid := by decide

private def activityLinkActionName : Umpire3.Temporal.System.NexusActivityLink.Action → String
  | .observeOperation => "observe-operation"
  | .observeLinkedActivity => "observe-linked-activity"
  | .observeOneSidedActivity => "observe-one-sided-activity"

private def activityLinkFinite : FiniteView Umpire3.Temporal.System.NexusActivityLink.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.NexusActivityLink.executable inferInstance
    activityLinkActionName (by
      intro left right
      cases left <;> cases right <;> simp_all [activityLinkActionName])

def activityLink : FiniteReplayView Umpire3.Temporal.System.NexusActivityLink.behavior () where
  artifact := artifact "integration-nexus-activity" "nexus-activity.link-consistency"
    "Umpire3.Temporal.System.NexusActivityLink.behavior" [
      attempt "link-nexus-activity" ["observe-operation", "observe-linked-activity"],
    ]
  finite := activityLinkFinite
  property := fun state => decide (state ≠ .oneSided)
  valid := by decide

private def progressActionName : Umpire3.Temporal.System.WorkflowProgress.Action → String
  | .enqueue => "enqueue"
  | .makeWorkerAvailable => "make-worker-available"
  | .wait => "wait"
  | .dispatch => "dispatch"
  | .complete => "complete"
  | .waitAgain => "wait-again"
  | .completeWrongEntity => "complete-wrong-entity"

private def progressFinite : FiniteView Umpire3.Temporal.System.WorkflowProgress.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.WorkflowProgress.executable inferInstance progressActionName (by
    intro left right
    cases left <;> cases right <;> simp_all [progressActionName])

private def progressTransitions : List String := [
  "enqueue", "make-worker-available", "wait", "dispatch", "complete",
]

private def entityProgressArtifact (target : String) : FiniteReplayArtifact :=
  artifact target "entity.progress"
    "Umpire3.Temporal.System.WorkflowProgress.behavior" [
      attempt "crash-owner",
      attempt "progress-entity" progressTransitions,
      attempt "recover-owner",
    ]

def deliveryProgress : FiniteReplayView Umpire3.Temporal.System.WorkflowProgress.behavior () where
  artifact := entityProgressArtifact "foundation-delivery-safety"
  finite := progressFinite
  property := fun state => decide (state ≠ .wrongEntityCompleted)
  valid := by decide

def activityProgress : FiniteReplayView Umpire3.Temporal.System.WorkflowProgress.behavior () where
  artifact := entityProgressArtifact "integration-activity-delivery"
  finite := progressFinite
  property := fun state => decide (state ≠ .wrongEntityCompleted)
  valid := by decide

def workflowDelivery : FiniteReplayView Umpire3.Temporal.System.WorkflowProgress.behavior () where
  artifact := artifact "integration-workflow-delivery" "workflow-task.starvation"
    "Umpire3.Temporal.System.WorkflowProgress.behavior" [
      attempt "dispatch-assurance-workflow-task" progressTransitions,
    ]
  finite := progressFinite
  property := fun state => decide (state ≠ .starved)
  valid := by decide

private def updateActionName : Umpire3.Temporal.System.UpdateLifecycle.Action → String
  | .start => "start"
  | .dispatchTask => "dispatch-task"
  | .accept => "accept"
  | .recordHistory => "record-history"
  | .completeWorkflowTask => "complete-workflow-task"
  | .complete => "complete"
  | .completeWithoutHistory => "complete-without-history"

private def updateFinite : FiniteView Umpire3.Temporal.System.UpdateLifecycle.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.UpdateLifecycle.executable inferInstance updateActionName (by
    intro left right
    cases left <;> cases right <;> simp_all [updateActionName])

def update : FiniteReplayView Umpire3.Temporal.System.UpdateLifecycle.behavior () where
  artifact := artifact "workflow-update-lifecycle"
    "workflow-update.accepted-completes-through-history"
    "Umpire3.Temporal.System.UpdateLifecycle.behavior" [
      attempt "start-update" ["start"],
      attempt "dispatch-workflow-task" ["dispatch-task"],
      attempt "accept-update" ["accept"],
      attempt "record-update-history" ["record-history"],
      attempt "complete-workflow-task" ["complete-workflow-task"],
      attempt "complete-update" ["complete"],
      attempt "task-delivery.environment-tick",
    ]
  finite := updateFinite
  property := fun state => decide (state ≠ .completedWithoutHistory)
  valid := by decide

private def taskAckActionName : Umpire3.Temporal.System.TaskAck.Protocol.Action → String
  | .enqueueMessage => "enqueue-message"
  | .issueDelivery => "issue-delivery"
  | .storeCompletion => "store-completion"
  | .storeCompletionWithoutRemovingBacklog => "store-completion-with-backlog"

private def taskAckFinite : FiniteView Umpire3.Temporal.System.TaskAck.Protocol.behavior () :=
  FiniteView.ofExecutable Umpire3.Temporal.System.TaskAck.Protocol.executable inferInstance taskAckActionName (by
    intro left right
    cases left <;> cases right <;> simp_all [taskAckActionName])

def backlogAcknowledgement : FiniteReplayView Umpire3.Temporal.System.TaskAck.Protocol.behavior () where
  artifact := artifact "foundation-backlog-ack" "task-delivery.acknowledged-removes-backlog"
    "Umpire3.Temporal.System.TaskAck.Protocol.behavior" [
      attempt "enqueue-workflow-task" ["enqueue-message"],
      attempt "deliver-workflow-task" ["issue-delivery"],
      attempt "acknowledge-workflow-task" ["store-completion"],
    ]
  finite := taskAckFinite
  property := fun state => decide (state ≠ .completionStoredWithBacklog)
  valid := by decide

def targetsJson? (semanticHash : String) : Option (List Lean.Json) := do
  let ownershipJson ← ownership.toJson? (resolved_declaration% ownership) limits semanticHash
  let continuationJson ← continuationLineage.toJson?
    (resolved_declaration% continuationLineage) limits semanticHash
  let resetJson ← resetLineage.toJson? (resolved_declaration% resetLineage) limits semanticHash
  let routingJson ← routing.toJson? (resolved_declaration% routing) limits semanticHash
  let speculativeJson ← speculative.toJson? (resolved_declaration% speculative) limits semanticHash
  let callbackReferenceJson ← callbackReference.toJson?
    (resolved_declaration% callbackReference) limits semanticHash
  let callbackResponseJson ← callbackResponse.toJson?
    (resolved_declaration% callbackResponse) limits semanticHash
  let atomicCallbackResponseJson ← atomicCallbackResponse.toJson?
    (resolved_declaration% atomicCallbackResponse) limits semanticHash
  let timeoutJson ← timeout.toJson? (resolved_declaration% timeout) limits semanticHash
  let closureJson ← closure.toJson? (resolved_declaration% closure) limits semanticHash
  let nexusProgressJson ← nexusProgress.toJson?
    (resolved_declaration% nexusProgress) limits semanticHash
  let activityLinkJson ← activityLink.toJson?
    (resolved_declaration% activityLink) limits semanticHash
  let deliveryProgressJson ← deliveryProgress.toJson?
    (resolved_declaration% deliveryProgress) limits semanticHash
  let activityProgressJson ← activityProgress.toJson?
    (resolved_declaration% activityProgress) limits semanticHash
  let workflowDeliveryJson ← workflowDelivery.toJson?
    (resolved_declaration% workflowDelivery) limits semanticHash
  let updateJson ← update.toJson? (resolved_declaration% update) limits semanticHash
  let backlogJson ← backlogAcknowledgement.toJson?
    (resolved_declaration% backlogAcknowledgement) limits semanticHash
  pure [
    ownershipJson, continuationJson, resetJson, routingJson, speculativeJson,
    callbackReferenceJson, callbackResponseJson, atomicCallbackResponseJson, timeoutJson,
    closureJson, nexusProgressJson, activityLinkJson, deliveryProgressJson, activityProgressJson,
    workflowDeliveryJson, updateJson, backlogJson,
  ]

end Umpire3.Temporal.Targets.FiniteReplay
