import Temporal.Feature.Nexus.Operations.Cancellation
import Umpire.Json

namespace Temporal.Feature.Nexus.OperationsTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Operations

private def expectedCancellationQueryJson : String :=
  include_str "../Fixtures/OperationsCancellationQuery.json"

namespace Cancellation

open Temporal.Feature.Nexus.Operations.Cancellation

theorem declarationsCheckSuccessfully :
    propertyResult.isOk = true ∧ behaviorResult.isOk = true ∧ queryResult.isOk = true := by
  native_decide

theorem queryUsesLifecycleTarget : query.target = target := by
  rfl

theorem queryJsonRemainsCanonical :
    Json.prettyBytes (canonicalQueryJson query) = expectedCancellationQueryJson := by
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
    some ([cancelAction], [canceledOutcome], [canceledState], [[canceledObservation]]) := by
  native_decide

end Cancellation

end Temporal.Feature.Nexus.OperationsTests
