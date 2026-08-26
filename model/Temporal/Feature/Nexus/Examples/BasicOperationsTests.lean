import Temporal.Feature.Nexus.Examples.BasicOperations

namespace Temporal.Feature.Nexus.Examples.BasicOperationsTests

open Umpire
open Temporal.Feature.Nexus.Examples.BasicLifecycle
open Temporal.Feature.Nexus.Examples.BasicOperations

namespace AsyncStart

open Temporal.Feature.Nexus.Examples.BasicOperations.AsyncStart

example : propertyResult.isOk = true ∧ behaviorResult.isOk = true ∧ queryResult.isOk = true := by
  native_decide

example : (evaluateProperty property intendedTrace.trace).satisfied = true ∧
    (evaluateProperty property wrongOutcomeTrace.trace).satisfied = false := by
  native_decide

example : behavior.admits intendedTrace = true ∧
    behavior.admits wrongOutcomeTrace = true ∧
    behavior.admits wrongActionTrace = false := by
  native_decide

example : run.result.outcome.name = "found" ∧ run = repeatedRun := by
  native_decide

example : run.artifact.map (fun artifact =>
    (artifact.plan.requestedActions,
      artifact.plan.modelOutcomes,
      artifact.plan.resultingStates,
      artifact.plan.checkpoints.map ObservationCheckpoint.observations)) =
    some ([startAction], [startedOutcome], [startedState], [[startedObservation]]) := by
  native_decide

end AsyncStart

namespace SuccessfulCompletion

open Temporal.Feature.Nexus.Examples.BasicOperations.SuccessfulCompletion

example : propertyResult.isOk = true ∧ behaviorResult.isOk = true ∧ queryResult.isOk = true := by
  native_decide

example : (evaluateProperty property intendedTrace.trace).satisfied = true ∧
    (evaluateProperty property wrongOutcomeTrace.trace).satisfied = false := by
  native_decide

example : behavior.admits intendedTrace = true ∧
    behavior.admits wrongOutcomeTrace = true ∧
    behavior.admits wrongActionTrace = false := by
  native_decide

example : run.result.outcome.name = "found" ∧ run = repeatedRun := by
  native_decide

example : run.artifact.map (fun artifact =>
    (artifact.plan.requestedActions,
      artifact.plan.modelOutcomes,
      artifact.plan.resultingStates,
      artifact.plan.checkpoints.map ObservationCheckpoint.observations)) =
    some ([reportSuccessAction], [succeededOutcome], [succeededState], [[succeededObservation]]) := by
  native_decide

end SuccessfulCompletion

end Temporal.Feature.Nexus.Examples.BasicOperationsTests
