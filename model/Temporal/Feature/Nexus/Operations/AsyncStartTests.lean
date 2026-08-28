import Temporal.Feature.Nexus.Operations.AsyncStart
import Umpire.Json

namespace Temporal.Feature.Nexus.OperationsTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Operations

private def expectedAsyncStartQueryJson : String :=
  include_str "../Fixtures/OperationsAsyncStartQuery.json"

namespace AsyncStart

open Temporal.Feature.Nexus.Operations.AsyncStart

theorem declarationsCheckSuccessfully :
    propertyResult.isOk = true ∧ behaviorResult.isOk = true ∧ queryResult.isOk = true := by
  native_decide

theorem queryUsesLifecycleTarget : query.target = target := by
  rfl

theorem queryRetainsFiniteCompletenessEvidence : query.completeness.map (fun evidence =>
      (evidence.roleAssignments, evidence.actions,
        evidence.roleDomainFingerprint, evidence.actionDomainFingerprint)) =
      (CheckedQueryTarget.ofTarget target).completeness.map (fun evidence =>
        (evidence.roleAssignments, evidence.actions,
          evidence.roleDomainFingerprint, evidence.actionDomainFingerprint)) ∧
    query.completeness.map (fun evidence =>
      (evidence.roleAssignments, evidence.actions)) =
      some ([scheduledSetup, startedSetup], [cancelAction, startAction, reportSuccessAction]) := by
  native_decide

theorem queryJsonRemainsCanonical :
    Json.prettyBytes (canonicalQueryJson query) = expectedAsyncStartQueryJson := by
  native_decide

theorem propertySeparatesExpectedOutcome :
    (evaluateProperty property intendedTrace.trace).satisfied = true ∧
    (evaluateProperty property wrongOutcomeTrace.trace).satisfied = false := by
  native_decide

theorem behaviorSeparatesSelectedAction : behavior.admits intendedTrace = true ∧
    behavior.admits wrongOutcomeTrace = true ∧
    behavior.admits wrongActionTrace = false := by
  native_decide

theorem planningRunIsDeterministic : run.result.outcome.name = "found" ∧ run = repeatedRun := by
  native_decide

theorem artifactRetainsExpectedPlanShape : run.artifact.map (fun artifact =>
    (artifact.plan.requestedActions,
      artifact.plan.modelOutcomes,
      artifact.plan.resultingStates,
      artifact.plan.checkpoints.map ObservationCheckpoint.observations)) =
    some ([startAction], [startedOutcome], [startedState], [[startedObservation]]) := by
  native_decide

end AsyncStart

end Temporal.Feature.Nexus.OperationsTests
