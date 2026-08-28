import Temporal.Feature.Nexus.Operations

namespace Temporal.Feature.Nexus.OperationsTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Operations

theorem sourceRemainsAnchoredToOperationsFacade : Temporal.Feature.Nexus.Operations.source = {
    path := "Temporal/Feature/Nexus/Operations.lean"
    line := 1
    column := 1
    provenance := "lean-model"
  } := by
  native_decide

theorem propertiesRetainCanonicalMetadata : [
    (AsyncStart.property.id, AsyncStart.property.source, AsyncStart.property.version,
      AsyncStart.property.documentation),
    (Cancellation.property.id, Cancellation.property.source, Cancellation.property.version,
      Cancellation.property.documentation),
    (SuccessfulCompletion.property.id, SuccessfulCompletion.property.source,
      SuccessfulCompletion.property.version, SuccessfulCompletion.property.documentation)
  ] = [
    (AsyncStart.propertyId, Temporal.Feature.Nexus.Operations.source, 1,
      "Starting a scheduled Nexus operation produces the target-owned started result."),
    (Cancellation.propertyId, Temporal.Feature.Nexus.Operations.source, 1,
      "Canceling a started Nexus operation produces the target-owned canceled result."),
    (SuccessfulCompletion.propertyId, Temporal.Feature.Nexus.Operations.source, 1,
      "Reporting success for a started Nexus operation produces the target-owned succeeded result.")
  ] := by
  native_decide

theorem behaviorsRetainCanonicalMetadata : [
    (AsyncStart.behavior.id, AsyncStart.behavior.source, AsyncStart.behavior.version,
      AsyncStart.behavior.documentation),
    (Cancellation.behavior.id, Cancellation.behavior.source, Cancellation.behavior.version,
      Cancellation.behavior.documentation),
    (SuccessfulCompletion.behavior.id, SuccessfulCompletion.behavior.source,
      SuccessfulCompletion.behavior.version, SuccessfulCompletion.behavior.documentation)
  ] = [
    (AsyncStart.behaviorId, Temporal.Feature.Nexus.Operations.source, 1,
      "Select exactly one start action and leave its result to the Nexus model."),
    (Cancellation.behaviorId, Temporal.Feature.Nexus.Operations.source, 1,
      "Select exactly one cancel action and leave its result to the Nexus model."),
    (SuccessfulCompletion.behaviorId, Temporal.Feature.Nexus.Operations.source, 1,
      "Select exactly one success report and leave its result to the Nexus model.")
  ] := by
  native_decide

theorem queriesRetainCanonicalMetadata : [
    (AsyncStart.query.id, AsyncStart.query.source, AsyncStart.query.version,
      AsyncStart.query.documentation),
    (Cancellation.query.id, Cancellation.query.source, Cancellation.query.version,
      Cancellation.query.documentation),
    (SuccessfulCompletion.query.id, SuccessfulCompletion.query.source,
      SuccessfulCompletion.query.version, SuccessfulCompletion.query.documentation)
  ] = [
    (AsyncStart.queryId, Temporal.Feature.Nexus.Operations.source, 1, ""),
    (Cancellation.queryId, Temporal.Feature.Nexus.Operations.source, 1, ""),
    (SuccessfulCompletion.queryId, Temporal.Feature.Nexus.Operations.source, 1, "")
  ] := by
  native_decide

/-- Every live ordinary Nexus consumer of the shared Lifecycle target. -/
def compatibilityConsumers : List String := [
  "nexus-operations-async-start",
  "nexus-operations-cancellation",
  "nexus-operations-successful-completion"
]

private def expectedAsyncStartArtifactJson : String :=
  include_str "../Fixtures/OperationsAsyncStartArtifact.json"

private def expectedCancellationArtifactJson : String :=
  include_str "../Fixtures/OperationsCancellationArtifact.json"

private def expectedSuccessfulCompletionArtifactJson : String :=
  include_str "../Fixtures/OperationsSuccessfulCompletionArtifact.json"

theorem incrementalKernelRetainsFiniteLifecycleDomain : incrementalKernel.actionLimit = 3 ∧
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

theorem queryIdentitiesAndFingerprintsRemainShared :
  let domainFingerprints := (CheckedQueryTarget.ofTarget target).completeness.map fun evidence =>
    (evidence.roleDomainFingerprint, evidence.actionDomainFingerprint)
  [
    AsyncStart.query,
    Cancellation.query,
    SuccessfulCompletion.query
  ].map (fun checked =>
    (checked.id.value, checked.target.id.value, checked.target.behaviorFingerprint,
      checked.completeness.map fun evidence =>
        (evidence.roleDomainFingerprint, evidence.actionDomainFingerprint))) = [
    ("temporal.nexus.basic-lifecycle.query.async-start",
      "temporal.nexus.basic-lifecycle.target", target.behaviorFingerprint,
      domainFingerprints),
    ("temporal.nexus.basic-lifecycle.query.cancellation",
      "temporal.nexus.basic-lifecycle.target", target.behaviorFingerprint,
      domainFingerprints),
    ("temporal.nexus.basic-lifecycle.query.successful-completion",
      "temporal.nexus.basic-lifecycle.target", target.behaviorFingerprint,
      domainFingerprints)
  ] := by
  native_decide

/-! Golden artifacts preserve canonical bytes for every ordinary lifecycle consumer. -/
theorem artifactsRetainCanonicalBytes : [
    AsyncStart.run.artifact.map canonicalExperimentSpecBytes,
    Cancellation.run.artifact.map canonicalExperimentSpecBytes,
    SuccessfulCompletion.run.artifact.map
      canonicalExperimentSpecBytes
  ] = [
    some expectedAsyncStartArtifactJson,
    some expectedCancellationArtifactJson,
    some expectedSuccessfulCompletionArtifactJson
  ] := by
  native_decide

theorem compatibilityConsumersRetainMigrationBoundary : compatibilityConsumers = [
    "nexus-operations-async-start",
    "nexus-operations-cancellation",
    "nexus-operations-successful-completion"
  ] := by
  rfl

end Temporal.Feature.Nexus.OperationsTests
