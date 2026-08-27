import Temporal.Feature.Nexus.Operations

namespace Temporal.Feature.Nexus.OperationsTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Operations

namespace AsyncStart

open Temporal.Feature.Nexus.Operations.AsyncStart

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

namespace Cancellation

open Temporal.Feature.Nexus.Operations.Cancellation

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
    some ([cancelAction], [canceledOutcome], [canceledState], [[canceledObservation]]) := by
  native_decide

end Cancellation

namespace SuccessfulCompletion

open Temporal.Feature.Nexus.Operations.SuccessfulCompletion

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

end Temporal.Feature.Nexus.OperationsTests
