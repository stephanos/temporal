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
    (evaluatePropertyOnTrace property intendedTrace.trace).toOption.map
        PropertyEvaluation.satisfied = some true ∧
    (evaluatePropertyOnTrace property wrongOutcomeTrace.trace).toOption.map
        PropertyEvaluation.satisfied = some false := by
  native_decide

theorem behaviorSeparatesSelectedAction : behavior.admits intendedTrace = true ∧
    behavior.admits wrongOutcomeTrace = true ∧
    behavior.admits wrongActionTrace = false := by
  native_decide

theorem planningRunIsDeterministic :
    run.toOption.map (fun planned => planned.result.outcome.name) = some "found" ∧
      (run.toOption == repeatedRun.toOption) = true := by
  native_decide

theorem artifactRetainsExpectedPlanShape : run.toOption.bind (fun planned =>
  planned.artifact.map (fun artifact =>
    (artifact.plan.requestedActions,
      artifact.plan.modelOutcomes,
      artifact.plan.resultingStates,
      artifact.plan.checkpoints.map ObservationCheckpoint.observations))) =
    some ([cancelAction], [canceledOutcome], [canceledState], [[canceledObservation]]) := by
  native_decide

end Cancellation

end Temporal.Feature.Nexus.OperationsTests
