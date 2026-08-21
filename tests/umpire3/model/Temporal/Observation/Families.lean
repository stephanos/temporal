import Umpire3.Observation

namespace Umpire3.Temporal.Observation.Families

open Umpire3.Observation

private def source (sequence : Nat) : Source where
  identity := "temporal/history"
  clockDomain := "temporal/history-event-id"
  sequence
  reference := "workflow/1/history/" ++ toString sequence
  entityIdentity := "workflow/1/run/1"
  lineage := ["namespace/1", "workflow/1", "run/1"]

private def historyFact (kind : String) (sequence : Nat) : Fact where
  identifier := "history/" ++ kind
  source := source sequence
  value := .history {
    eventType := kind
    eventID := sequence
    workflowID := some "workflow/1"
    runID := some "run/1"
  }

private def closedWindow (observation : String) (sequence : Nat) : Fact where
  identifier := "window/" ++ observation
  source := source sequence
  value := .window {
    purpose := observation
    closed := true
    throughSequence := sequence
  }

private def selector (kind : String) : Selector := {
  factType := "history-event"
  kind
}

private def mechanismSelector (kind : String) (outcome : Option String := none) : Selector := {
  factType := "mechanism-receipt"
  kind
  outcome
}

private def program (observation : String) (required : List String)
    (violation : String) : Program where
  identifier := "observation." ++ observation
  observation
  operation := .allExistAbsentWhenClosed
  matchers := required.map selector
  violations := [selector violation]
  closures := [{ factType := "evidence-window", kind := observation, closed := some true }]

private def fixture (observation : String) (required : List String) : Fixture :=
  let facts := (required.zip (List.range required.length)).map fun entry =>
    historyFact entry.1 (entry.2 + 1)
  let window := closedWindow observation (required.length + 1)
  {
    identifier := observation ++ ".established"
    observation
    facts := facts ++ [window]
    expected := {
      value := .true
      support := required.mergeSort.map ("history/" ++ ·) ++ ["window/" ++ observation]
    }
  }

private def existsProgram (observation kind : String) : Program where
  identifier := "observation." ++ observation
  observation
  operation := .exists
  matchers := [selector kind, mechanismSelector kind]

private def existsFixture (observation kind : String) : Fixture := {
  identifier := observation ++ ".established"
  observation
  facts := [historyFact kind 1]
  expected := { value := .true, support := ["history/" ++ kind] }
}

def updateAccepted : Program := existsProgram "update-accepted" "workflow-update-accepted"

def updateCompleted : Program := existsProgram "update-completed" "workflow-update-completed"

def workflowTaskAcknowledged : Program where
  identifier := "observation.workflow-task-acknowledged"
  observation := "workflow-task-acknowledged"
  operation := .allExistAbsentWhenClosed
  matchers := [mechanismSelector "workflow-task-acknowledged" (some "backlog-absent")]
  violations := [mechanismSelector "workflow-task-acknowledged" (some "backlog-present")]
  closures := [{
    factType := "evidence-window"
    kind := "workflow-task-acknowledged"
    closed := some true
  }]

def speculativeTaskValid : Program := program "speculative-task-valid" [
  "update-requested", "speculative-task-created", "speculative-task-committed",
] "speculative-task-orphaned"

def nexusOperationClosed : Program := program "nexus-operation-closed" [
  "nexus-operation-settled", "caller-workflow-closed",
] "caller-closed-with-open-operation"

def nexusActivityLinksConsistent : Program := program "nexus-activity-links-consistent" [
  "nexus-operation-linked-activity", "activity-linked-nexus-operation",
] "nexus-activity-link-one-sided"

def nexusTimeoutValid : Program := program "nexus-timeout-valid" [
  "nexus-timeout-configured", "nexus-operation-timed-out",
] "nexus-timeout-metadata-invalid"

def callbackReferenceValid : Program := program "callback-reference-valid" [
  "callback-attached", "nexus-operation-started",
] "callback-reference-mismatch"

def callbackResponseConsistent : Program := program "callback-response-consistent" [
  "callback-registered", "callback-operation-settled", "callback-response-recorded",
] "callback-response-conflict"

def workflowTaskNotStarved : Program := program "workflow-task-not-starved" [
  "workflow-task-queued", "workflow-worker-available", "workflow-task-completed",
] "workflow-task-starved"

def entityProgressed : Program := program "entity-progressed" [
  "entity-pending", "workflow-task-completed", "entity-progressed",
] "workflow-task-completed-wrong-entity"

def workflowContinuationLineageValid : Program := program
  "workflow-continuation-lineage-valid" ["workflow-continued", "workflow-lineage-recorded"]
  "workflow-continuation-lineage-invalid"

def workflowResetLineageValid : Program := program
  "workflow-reset-lineage-valid" ["workflow-reset", "workflow-lineage-recorded"]
  "workflow-reset-lineage-invalid"

def workflowRoutingIsolated : Program := program "workflow-routing-isolated" [
  "workflow-task-routed", "workflow-poller-routed", "workflow-task-reserved",
] "workflow-task-cross-route"

def workflowOwnershipFenced : Program := program "workflow-ownership-fenced" [
  "workflow-task-dispatched", "workflow-owner-rotated", "stale-completion-rejected",
  "current-completion-recorded",
] "stale-completion-recorded"

def programs : List Program := [
  updateAccepted,
  updateCompleted,
  workflowTaskAcknowledged,
  speculativeTaskValid,
  nexusOperationClosed,
  nexusActivityLinksConsistent,
  nexusTimeoutValid,
  callbackReferenceValid,
  callbackResponseConsistent,
  workflowTaskNotStarved,
  entityProgressed,
  workflowContinuationLineageValid,
  workflowResetLineageValid,
  workflowRoutingIsolated,
  workflowOwnershipFenced,
]

def fixtures : List Fixture := [
  existsFixture "update-accepted" "workflow-update-accepted",
  existsFixture "update-completed" "workflow-update-completed",
  {
    identifier := "workflow-task-acknowledged.established"
    observation := "workflow-task-acknowledged"
    facts := [
      {
        identifier := "mechanism/workflow-task-acknowledged"
        source := source 1
        value := .mechanism {
          action := "workflow-task-acknowledged"
          resource := "workflow/1"
          attempt := 1
          ownerEpoch := 0
          outcome := "backlog-absent"
        }
      },
      closedWindow "workflow-task-acknowledged" 2,
    ]
    expected := {
      value := .true
      support := ["mechanism/workflow-task-acknowledged", "window/workflow-task-acknowledged"]
    }
  },
  fixture "speculative-task-valid"
    ["update-requested", "speculative-task-created", "speculative-task-committed"],
  fixture "nexus-operation-closed" ["nexus-operation-settled", "caller-workflow-closed"],
  fixture "nexus-activity-links-consistent"
    ["nexus-operation-linked-activity", "activity-linked-nexus-operation"],
  fixture "nexus-timeout-valid" ["nexus-timeout-configured", "nexus-operation-timed-out"],
  fixture "callback-reference-valid" ["callback-attached", "nexus-operation-started"],
  fixture "callback-response-consistent"
    ["callback-registered", "callback-operation-settled", "callback-response-recorded"],
  fixture "workflow-task-not-starved"
    ["workflow-task-queued", "workflow-worker-available", "workflow-task-completed"],
  fixture "entity-progressed" ["entity-pending", "workflow-task-completed", "entity-progressed"],
  fixture "workflow-continuation-lineage-valid" ["workflow-continued", "workflow-lineage-recorded"],
  fixture "workflow-reset-lineage-valid" ["workflow-reset", "workflow-lineage-recorded"],
  fixture "workflow-routing-isolated"
    ["workflow-task-routed", "workflow-poller-routed", "workflow-task-reserved"],
  fixture "workflow-ownership-fenced" [
    "workflow-task-dispatched", "workflow-owner-rotated", "stale-completion-rejected",
    "current-completion-recorded",
  ],
]

end Umpire3.Temporal.Observation.Families
