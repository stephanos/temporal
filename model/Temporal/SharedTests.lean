import Temporal.Feature.Nexus.Operations.AsyncStart
import Temporal.Feature.Nexus.Operations.Cancellation
import Umpire.Query.Authoring

/-! Temporal-owned identity, source, and named Query Limit contracts.

The 1×/10× specimen counts independent family construction. Its added work is one
`Temporal.Shared.definitionFamily` plus one `DefinitionFamily.id` call per declaration. Source and
Limit construction add one `Temporal.Shared.sourceLocation` and one `QueryLimitSpec.toQueryLimits`
record assembly per declaration. None traverses a declaration collection; the owning language
checkers remain separate and their existing work is excluded from this structural count. -/

namespace Temporal.SharedTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Operations

def family : DefinitionFamily :=
  Temporal.Shared.definitionFamily "nexus.basic-lifecycle"

example : family.root = DefinitionId.of "temporal.nexus.basic-lifecycle" ∧
    family.id "query" "async-start" =
      DefinitionId.of "temporal.nexus.basic-lifecycle.query.async-start" := by
  exact ⟨rfl, rfl⟩

#guard_msgs (error, substring := true) in
def foreignRootFamily : DefinitionFamily :=
  Temporal.Shared.definitionFamily
    (root := DefinitionId.of "foreign.nexus.basic-lifecycle") "nexus.basic-lifecycle"

example : Internal.family = family ∧
    Internal.family.id "query" "async-start" = AsyncStart.queryId ∧
    Internal.queryLimitSpec.toQueryLimits = limits := by
  exact ⟨rfl, rfl, rfl⟩

private def queryErrorOf
    (declaration : QueryDeclaration) : Option QueryError :=
  match checkQuery queryContext declaration with
  | .error error => some error
  | .ok _ => none

private def baseDeclaration : QueryDeclaration :=
  Internal.queryDeclaration AsyncStart.queryId AsyncStart.property AsyncStart.behavior

def malformedIdError : Option QueryError :=
  queryErrorOf { baseDeclaration with id := DefinitionId.of "temporal" }

def duplicateReferenceError : Option QueryError :=
  queryErrorOf {
    baseDeclaration with
    form := .select [AsyncStart.property, AsyncStart.property]
  }

def crossedReferenceError : Option QueryError :=
  queryErrorOf {
    baseDeclaration with
    target := Cancellation.propertyId
  }

example : malformedIdError.map (fun error =>
      (error.kind, error.definitionId, error.sourcePath, error.relatedDefinitionIds)) =
      some (.invalidDefinitionId, DefinitionId.of "temporal",
        "Temporal/Feature/Nexus/Operations.lean", [DefinitionId.of "temporal"]) ∧
    duplicateReferenceError.map (fun error =>
      (error.kind, error.definitionId, error.sourcePath, error.relatedDefinitionIds)) =
      some (.duplicateProperty, AsyncStart.queryId,
        "Temporal/Feature/Nexus/Operations.lean", [AsyncStart.propertyId]) ∧
    crossedReferenceError.map (fun error =>
      (error.kind, error.definitionId, error.sourcePath, error.relatedDefinitionIds)) =
      some (.targetMismatch, AsyncStart.queryId,
        "Temporal/Feature/Nexus/Operations.lean",
        [Cancellation.propertyId, target.id]) := by
  native_decide

private def declarationWithLimitSpec (spec : QueryLimitSpec) : QueryDeclaration :=
  { baseDeclaration with limits := spec.toQueryLimits }

private def wrongUnitLimits : QueryLimits := {
  Internal.queryLimitSpec.toQueryLimits with
  behavior := {
    Internal.queryLimitSpec.toQueryLimits.behavior with
    transitions := { value := 1, unit := .selectedActions }
  }
}

def limitErrors : List (Option (QueryErrorKind × String × String)) := [
  queryErrorOf (declarationWithLimitSpec {
    Internal.queryLimitSpec with transitions := 0
  }) |>.map fun error => (error.kind, error.sourcePath, error.offendingValue),
  queryErrorOf (declarationWithLimitSpec {
    Internal.queryLimitSpec with selectedActions := 0
  }) |>.map fun error => (error.kind, error.sourcePath, error.offendingValue),
  queryErrorOf (declarationWithLimitSpec {
    Internal.queryLimitSpec with candidateEvaluations := 0
  }) |>.map fun error => (error.kind, error.sourcePath, error.offendingValue),
  queryErrorOf { baseDeclaration with limits := wrongUnitLimits } |>.map fun error =>
    (error.kind, error.sourcePath, error.offendingValue)
]

example : limitErrors = [
    some (.invalidLimit, "Temporal/Feature/Nexus/Operations.lean", "behavior.transitions=0"),
    some (.invalidLimit, "Temporal/Feature/Nexus/Operations.lean", "behavior.selectedActions=0"),
    some (.invalidLimit, "Temporal/Feature/Nexus/Operations.lean",
      "search.candidateEvaluations=0"),
    some (.unitMismatch, "Temporal/Feature/Nexus/Operations.lean",
      "behavior.transitions:selected-actions")
  ] := by
  native_decide

def alternateSource : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/SharedTests/Alternate.lean"

private def orderedDeclaration (properties : List CheckedProperty) (querySource : SourceLocation) :
    QueryDeclaration := {
  baseDeclaration with
  source := querySource
  form := .select properties
}

private def checkedIdentity
    (declaration : QueryDeclaration) : Option (DefinitionId × BehaviorFingerprint × SourceLocation) :=
  (checkQuery queryContext declaration).toOption.map fun checked =>
    (checked.id, checked.behaviorFingerprint, checked.source)

example :
    let first := checkedIdentity <| orderedDeclaration
      [AsyncStart.property, Cancellation.property] Temporal.Feature.Nexus.Operations.source
    let second := checkedIdentity <| orderedDeclaration
      [Cancellation.property, AsyncStart.property] alternateSource
    first.map (fun checked => (checked.1, checked.2.1)) =
      second.map (fun checked => (checked.1, checked.2.1)) ∧
    first.map (fun checked => checked.2.2) = some Temporal.Feature.Nexus.Operations.source ∧
    second.map (fun checked => checked.2.2) = some alternateSource := by
  native_decide

def oneIndependentIdentity : List DefinitionId :=
  [Temporal.Shared.definitionFamily "nexus.family-0" |>.id "query" "operation"]

def tenIndependentIdentities : List DefinitionId :=
  (List.range 10).map fun index =>
    Temporal.Shared.definitionFamily ("nexus.family-" ++ toString index) |>.id
      "query" "operation"

example : oneIndependentIdentity.length = 1 ∧ tenIndependentIdentities.length = 10 := by
  native_decide

#print axioms Temporal.Shared.definitionFamily
#print axioms Internal.family
#print axioms Internal.queryLimitSpec

end Temporal.SharedTests
