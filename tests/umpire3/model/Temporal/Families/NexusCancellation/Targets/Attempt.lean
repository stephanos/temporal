import Temporal.Families.NexusCancellation.Targets.FirstOrder
import Umpire3.AttemptView

namespace Umpire3.Temporal.Targets.NexusCancellationFencing

private def attemptField (identifier : String) : FirstOrderTerm := .field identifier

private def attemptValue (sort identifier : String) : FirstOrderTerm := .value sort identifier

private def cancelled : FirstOrderFormula :=
  .equal (attemptField "lifecycle") (attemptValue "lifecycle" "cancelled")

private def applied (action : String) : AttemptOutcome where
  outcome := .applied
  guard := .truth
  transitions := [action]

private def stutter (outcome : ActionOutcome) (guard : FirstOrderFormula) : AttemptOutcome where
  outcome := outcome
  guard := guard
  transitions := []

private def modeledAttempt (action : String) : AttemptMapping where
  action := action
  outcomes := [applied action]

private def guardedAttempt (action : String) : AttemptMapping where
  action := action
  outcomes := [
    applied action,
    stutter .suppressed cancelled,
    stutter .rejected cancelled,
    stutter .retried cancelled,
    stutter .faultIntercepted cancelled,
  ]

private def liveAttempt (action : String) (outcomes : List ActionOutcome := []) : AttemptMapping where
  action := action
  outcomes := { outcome := .applied, guard := .truth, transitions := [] } ::
    outcomes.map fun outcome => stutter outcome .truth

private def artifact (variant : String) : AttemptArtifact where
  target := "nexus-cancellation"
  property := "nexus.cancellation.won-excludes-success"
  world := "smoke"
  variant := variant
  canonicalModel := "Umpire3.Temporal.System.NexusCancellationFencing.behavior"
  attempts := [
    modeledAttempt "dispatch-task",
    modeledAttempt "request-cancellation",
    modeledAttempt "acquire-ownership",
    modeledAttempt "commit-cancellation",
    guardedAttempt "worker-returns-success",
    guardedAttempt "persist-success",
    liveAttempt "schedule-operation",
    liveAttempt "retry-task" [.retried],
  ]

def soundAttemptArtifact : AttemptArtifact := artifact "sound"

def mutatedAttemptArtifact : AttemptArtifact := artifact "stale-completion-guard-removed"

def soundAttemptView : AttemptView soundFirstOrderView where
  artifact := soundAttemptArtifact
  valid := by decide

def mutatedAttemptView : AttemptView mutatedFirstOrderView where
  artifact := mutatedAttemptArtifact
  valid := by decide

def soundAttemptExport : AttemptExport where
  view := resolved_attempt% soundAttemptView

def mutatedAttemptExport : AttemptExport where
  view := resolved_attempt% mutatedAttemptView

end Umpire3.Temporal.Targets.NexusCancellationFencing
