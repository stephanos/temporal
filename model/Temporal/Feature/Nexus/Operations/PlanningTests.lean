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

theorem operationQueriesRetainExplicitEmptyAuthoredKnownGaps : [
    AsyncStart.query.authoredKnownGaps.toList,
    Cancellation.query.authoredKnownGaps.toList,
    SuccessfulCompletion.query.authoredKnownGaps.toList
  ] = [[], [], []] := by
  native_decide

theorem constructorDeclarationsRetainPublishedIdentities : [
    (AsyncStart.propertyDeclaration.id,
      AsyncStart.propertyDeclaration.clauses.map PropertyClause.id,
      AsyncStart.behaviorDeclaration.id,
      AsyncStart.behaviorDeclaration.requiredOccurrences.map NamedOccurrence.id,
      AsyncStart.queryDeclaration.id),
    (Cancellation.propertyDeclaration.id,
      Cancellation.propertyDeclaration.clauses.map PropertyClause.id,
      Cancellation.behaviorDeclaration.id,
      Cancellation.behaviorDeclaration.requiredOccurrences.map NamedOccurrence.id,
      Cancellation.queryDeclaration.id),
    (SuccessfulCompletion.propertyDeclaration.id,
      SuccessfulCompletion.propertyDeclaration.clauses.map PropertyClause.id,
      SuccessfulCompletion.behaviorDeclaration.id,
      SuccessfulCompletion.behaviorDeclaration.requiredOccurrences.map NamedOccurrence.id,
      SuccessfulCompletion.queryDeclaration.id)
  ] = [
    (AsyncStart.propertyId, [
        Internal.id "temporal.nexus.basic-lifecycle.property.async-start.state",
        Internal.id "temporal.nexus.basic-lifecycle.property.async-start.outcome",
        Internal.id "temporal.nexus.basic-lifecycle.property.async-start.observation"],
      AsyncStart.behaviorId, [AsyncStart.occurrenceId], AsyncStart.queryId),
    (Cancellation.propertyId, [
        Internal.id "temporal.nexus.basic-lifecycle.property.cancellation.state",
        Internal.id "temporal.nexus.basic-lifecycle.property.cancellation.outcome",
        Internal.id "temporal.nexus.basic-lifecycle.property.cancellation.observation"],
      Cancellation.behaviorId, [Cancellation.occurrenceId], Cancellation.queryId),
    (SuccessfulCompletion.propertyId, [
        Internal.id "temporal.nexus.basic-lifecycle.property.successful-completion.state",
        Internal.id "temporal.nexus.basic-lifecycle.property.successful-completion.outcome",
        Internal.id "temporal.nexus.basic-lifecycle.property.successful-completion.observation"],
      SuccessfulCompletion.behaviorId, [SuccessfulCompletion.occurrenceId],
      SuccessfulCompletion.queryId)
  ] := by
  native_decide

theorem rawQueryDeclarationCompatibility :
    Internal.queryDeclaration AsyncStart.queryId AsyncStart.property AsyncStart.behavior =
      AsyncStart.queryDeclaration := by
  native_decide

private def propertyErrorKind (spec : PropertySpec) : Option PropertyErrorKind :=
  match spec.check (PropertyCheckContext.ofTarget target) with
  | .error error => some error.kind
  | .ok _ => none

private def behaviorSpaceStatus (spec : ExactSequenceSpec) : Option BehaviorSpaceStatus := do
  let checked ← spec.check (.ofTarget target) |>.toOption
  pure checked.spaceStatus

private def queryErrorKind (spec : QuerySpec) : Option QueryErrorKind :=
  match spec.check target with
  | .error error => some error.kind
  | .ok _ => none

theorem duplicateOperationClausesRetainTypedFailure :
    propertyErrorKind {
      AsyncStart.propertySpec with
      key := "async-start-duplicate-clause"
      clauses := AsyncStart.propertySpec.clauses ++ AsyncStart.propertySpec.clauses
    } = some .duplicateDefinitionId := by
  native_decide

theorem missingOperationCapabilityRetainsTypedFailure :
    propertyErrorKind {
      AsyncStart.propertySpec with
      key := "async-start-missing-capability"
      requires := []
    } = some .undeclaredReference := by
  native_decide

theorem contradictoryOperationBehaviorRemainsUnsatisfiable :
    behaviorSpaceStatus {
      AsyncStart.behaviorSpec with
      key := "async-start-contradictory"
      setup := AsyncStart.behaviorSpec.setup ++ [
        {
          id := Internal.family.id "setup" "scheduled-contradiction"
          relation := .different
          left := .role operationRoleId
          right := .value scheduledState
        }]
    } = some .unsatisfiable := by
  native_decide

theorem operationQueryTargetMismatchRetainsTypedFailure :
    queryErrorKind {
      AsyncStart.querySpec with
      key := "async-start-target-mismatch"
      target := Internal.id "temporal.nexus.basic-lifecycle.target.other"
    } = some .targetMismatch := by
  native_decide

theorem invalidOperationLimitsRetainTypedFailure :
    queryErrorKind {
      AsyncStart.querySpec with
      key := "async-start-invalid-limits"
      limits := { Internal.queryLimitSpec with transitions := 0 }
    } = some .invalidLimit := by
  native_decide

/-
error: type mismatch
-/
#guard_msgs (error, substring := true) in
def omittedPropertyEvidence : CheckedProperty :=
  AsyncStart.propertySpec.checked (PropertyCheckContext.ofTarget target)

/-
error: type mismatch
-/
#guard_msgs (error, substring := true) in
def omittedBehaviorEvidence : CheckedBehavior :=
  AsyncStart.behaviorSpec.checked (.ofTarget target)

/-
error: type mismatch
-/
#guard_msgs (error, substring := true) in
def omittedQueryEvidence : CheckedQuery LawStatement :=
  AsyncStart.querySpec.checked target

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

private def plannerAdmissionErrorKind
    {checkedTarget : QueryTarget LawStatement}
    (result : Except FinitePlannerAdmissionError (IncrementalPlannerKernel checkedTarget)) :
    Option FinitePlannerAdmissionErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

theorem checkedQueryPlannerAdmissionsPreserveFailures :
    AsyncStart.incrementalKernelResult.isOk = true ∧
    Cancellation.incrementalKernelResult.isOk = true ∧
    SuccessfulCompletion.incrementalKernelResult.isOk = true ∧
    plannerAdmissionErrorKind
        (IncrementalPlannerKernel.ofCheckedQuery
          (Internal.id "temporal.nexus.basic-lifecycle.target.other") AsyncStart.query) =
      some .targetMismatch ∧
    let incomplete := { AsyncStart.query with completeness := none }
    plannerAdmissionErrorKind
        (IncrementalPlannerKernel.ofCheckedQuery incomplete.target.id incomplete) =
      some .missingFiniteCompleteness := by
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
    AsyncStart.run.toOption.bind (fun run => run.artifact.map canonicalExperimentSpecBytes),
    Cancellation.run.toOption.bind (fun run => run.artifact.map canonicalExperimentSpecBytes),
    SuccessfulCompletion.run.toOption.bind (fun run => run.artifact.map
      canonicalExperimentSpecBytes)
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

#print axioms IncrementalPlannerKernel.ofCheckedQuery_isSome
#print axioms AsyncStart.property
#print axioms AsyncStart.behavior
#print axioms AsyncStart.query
#print axioms AsyncStart.incrementalKernel
#print axioms AsyncStart.run
#print axioms Cancellation.property
#print axioms Cancellation.behavior
#print axioms Cancellation.query
#print axioms Cancellation.incrementalKernel
#print axioms Cancellation.run
#print axioms SuccessfulCompletion.property
#print axioms SuccessfulCompletion.behavior
#print axioms SuccessfulCompletion.query
#print axioms SuccessfulCompletion.incrementalKernel
#print axioms SuccessfulCompletion.run

end Temporal.Feature.Nexus.OperationsTests
