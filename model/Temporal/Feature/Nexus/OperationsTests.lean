import Temporal.Feature.Nexus.Operations

namespace Temporal.Feature.Nexus.OperationsTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Operations

/-- Every live ordinary Nexus consumer of the shared Lifecycle target. -/
def compatibilityConsumers : List String := [
  "nexus-operations-async-start",
  "nexus-operations-cancellation",
  "nexus-operations-successful-completion"
]

private def expectedAsyncStartQueryJson : String :=
  include_str "Fixtures/OperationsAsyncStartQuery.json"

private def expectedCancellationQueryJson : String :=
  include_str "Fixtures/OperationsCancellationQuery.json"

private def expectedSuccessfulCompletionQueryJson : String :=
  include_str "Fixtures/OperationsSuccessfulCompletionQuery.json"

namespace AsyncStart

open Temporal.Feature.Nexus.Operations.AsyncStart

example : propertyResult.isOk = true ∧ behaviorResult.isOk = true ∧ queryResult.isOk = true := by
  native_decide

example : query.target = target := by
  rfl

example : query.completeness.map (fun evidence =>
      (evidence.roleAssignments, evidence.actions,
        evidence.roleDomainDigest, evidence.actionDomainDigest)) =
      some ([scheduledSetup, startedSetup], [cancelAction, startAction, reportSuccessAction],
        "temporal-nexus-basic-lifecycle-role-domain/v1",
        "temporal-nexus-basic-lifecycle-action-domain/v2") := by
  native_decide

example : canonicalQueryJson query ++ "\n" = expectedAsyncStartQueryJson := by
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

example : incrementalKernel.actionLimit = 3 ∧
    incrementalKernel.actionAt 0 = some cancelAction ∧
    incrementalKernel.actionAt 1 = some startAction ∧
    incrementalKernel.actionAt 2 = some reportSuccessAction ∧
    incrementalKernel.actionAt 3 = none ∧
    incrementalKernel.initialAt scheduledSetup 0 = some scheduledState ∧
    incrementalKernel.initialAt startedSetup 0 = some startedState ∧
    incrementalKernel.stepAt scheduledState startAction 0 = some startedResult ∧
    incrementalKernel.stepAt startedState cancelAction 0 = some canceledResult ∧
    incrementalKernel.stepAt startedState reportSuccessAction 0 = some succeededResult ∧
    incrementalKernel.stepAt startedState startAction 0 = none := by
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

example : query.target = target := by
  rfl

example : canonicalQueryJson query ++ "\n" = expectedCancellationQueryJson := by
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

example : query.target = target := by
  rfl

example : canonicalQueryJson query ++ "\n" = expectedSuccessfulCompletionQueryJson := by
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

example : [
    AsyncStart.query,
    Cancellation.query,
    SuccessfulCompletion.query
  ].map (fun checked =>
    (checked.id.value, checked.target.id.value, checked.target.semanticDigest,
      checked.completeness.map fun evidence =>
        (evidence.roleDomainDigest, evidence.actionDomainDigest))) = [
    ("temporal.nexus.basic-lifecycle.query.async-start",
      "temporal.nexus.basic-lifecycle.target", target.semanticDigest,
      some ("temporal-nexus-basic-lifecycle-role-domain/v1",
        "temporal-nexus-basic-lifecycle-action-domain/v2")),
    ("temporal.nexus.basic-lifecycle.query.cancellation",
      "temporal.nexus.basic-lifecycle.target", target.semanticDigest,
      some ("temporal-nexus-basic-lifecycle-role-domain/v1",
        "temporal-nexus-basic-lifecycle-action-domain/v2")),
    ("temporal.nexus.basic-lifecycle.query.successful-completion",
      "temporal.nexus.basic-lifecycle.target", target.semanticDigest,
      some ("temporal-nexus-basic-lifecycle-role-domain/v1",
        "temporal-nexus-basic-lifecycle-action-domain/v2"))
  ] := by
  native_decide

/-! Independent runs preserve canonical artifact bytes for every ordinary lifecycle consumer. -/
example : [
    AsyncStart.run.artifact.map canonicalExperimentSpecJson,
    Cancellation.run.artifact.map canonicalExperimentSpecJson,
    SuccessfulCompletion.run.artifact.map canonicalExperimentSpecJson
  ] = [
    AsyncStart.repeatedRun.artifact.map canonicalExperimentSpecJson,
    Cancellation.repeatedRun.artifact.map canonicalExperimentSpecJson,
    SuccessfulCompletion.repeatedRun.artifact.map canonicalExperimentSpecJson
  ] := by
  native_decide

example : compatibilityConsumers = [
    "nexus-operations-async-start",
    "nexus-operations-cancellation",
    "nexus-operations-successful-completion"
  ] := by
  rfl

end Temporal.Feature.Nexus.OperationsTests
