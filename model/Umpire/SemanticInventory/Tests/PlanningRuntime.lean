import Umpire.Artifact.Runtime
import Umpire.Planning.Engine

/-! Planning and runtime constructor catalogs retain their owner-local vocabularies. -/

namespace Umpire.SemanticInventoryTests.PlanningRuntime

open Umpire

private def value (suffix : String) : ModelValue := {
  definitionId := DefinitionId.of ("umpire.semantic-inventory.fixture." ++ suffix)
  value := suffix
}

private def trace (suffix : String) : BehaviorTrace := {
  setup := []
  trace := { initialState := value suffix, steps := [] }
}

private def queryError (suffix : String) : QueryError := {
  kind := .invalidDefinitionId
  definitionId := DefinitionId.of ("umpire.semantic-inventory.fixture." ++ suffix)
  sourcePath := suffix
  offendingValue := suffix
  relatedDefinitionIds := []
}

example : OutcomeConstructorClassifiers.names PlanningOutcome.constructorClassifiers = [
    "found",
    "verified-within-limits",
    "no-such-trace-within-complete-limits",
    "limit-reached",
    "unsatisfiable",
    "invalid"
  ] := by
  native_decide

example :
    OutcomeConstructorClassifiers.matchCount PlanningOutcome.constructorClassifiers
        (.found (trace "first") .satisfyingWitness) = 1 ∧
    OutcomeConstructorClassifiers.matchCount PlanningOutcome.constructorClassifiers
        (.found (trace "second") .behaviorSelection) = 1 ∧
    OutcomeConstructorClassifiers.matchCount PlanningOutcome.constructorClassifiers
        (.invalid (queryError "first")) = 1 ∧
    OutcomeConstructorClassifiers.matchCount PlanningOutcome.constructorClassifiers
        (.invalid (queryError "second")) = 1 := by
  native_decide

example : OutcomeConstructorClassifiers.ExactlyOne PlanningOutcome.constructorClassifiers :=
  PlanningOutcome.constructorClassifiers_exactlyOne

example : OutcomeConstructorClassifiers.names PhaseOutcomeStatus.constructorClassifiers =
    ["not-started", "succeeded", "failed", "timed-out", "canceled"] := by
  native_decide

example : OutcomeConstructorClassifiers.ExactlyOne PhaseOutcomeStatus.constructorClassifiers :=
  PhaseOutcomeStatus.constructorClassifiers_exactlyOne

example : OutcomeConstructorClassifiers.names ControlAttemptStatus.constructorClassifiers =
    ["accepted", "rejected", "unsupported", "failed", "canceled", "not-attempted"] := by
  native_decide

example : OutcomeConstructorClassifiers.ExactlyOne ControlAttemptStatus.constructorClassifiers :=
  ControlAttemptStatus.constructorClassifiers_exactlyOne

example : OutcomeConstructorClassifiers.names SourceClosureStatus.constructorClassifiers =
    ["closed", "partial", "failed"] := by
  native_decide

example : OutcomeConstructorClassifiers.ExactlyOne SourceClosureStatus.constructorClassifiers :=
  SourceClosureStatus.constructorClassifiers_exactlyOne

example : OutcomeConstructorClassifiers.names CleanupStatus.constructorClassifiers =
    ["complete", "incomplete", "failed"] := by
  native_decide

example : OutcomeConstructorClassifiers.ExactlyOne CleanupStatus.constructorClassifiers :=
  CleanupStatus.constructorClassifiers_exactlyOne

example : OutcomeConstructorClassifiers.names OperationalStatus.constructorClassifiers =
    ["succeeded", "incomplete", "failed"] := by
  native_decide

example : OutcomeConstructorClassifiers.ExactlyOne OperationalStatus.constructorClassifiers :=
  OperationalStatus.constructorClassifiers_exactlyOne

example : [
    KnownGapSourceShape.exactKnownGap,
    .generatedKnownGapFamily,
    .authoredImplementationLinkKnownGapFamily,
    .evidenceGapAdmissionProjection,
    .carriedCatalogEntry
  ].map KnownGapSourceShape.name = [
    "exact-known-gap",
    "generated-known-gap-family",
    "authored-implementation-link-known-gap-family",
    "evidence-gap-admission-projection",
    "carried-catalog-entry"
  ] := by
  native_decide

example : [KnownGapCarryMapping.exact, .observationAdmission].map KnownGapCarryMapping.name =
    [
      "kind -> kind; code -> code; subject -> subject; detail -> detail",
      "code -> code; subject.toList -> relatedDefinitionIds; kind -> absent; detail -> absent"
    ] := by
  native_decide

end Umpire.SemanticInventoryTests.PlanningRuntime
