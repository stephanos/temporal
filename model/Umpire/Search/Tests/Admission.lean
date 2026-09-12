import Umpire.Search.Admission
import Umpire.Examples.Switch

/-! One admission per stage: each rejection surfaces as its own diagnostic, carrying the error the
stage returns on its own, and an admitted Query searches exactly as the staged calls did. -/

namespace Umpire.SearchTests.Admission

open Umpire
open Umpire.Examples.Switch

private def errorOf : Except ε α → Option ε
  | .ok _ => none
  | .error error => some error

private def shape (queryId : DefinitionId := exactActionQueryId) : Query.Shape := {
  id := queryId
  source
  target := target.id
  form := .find
  limits
  policy := shortestPolicy
}

private def invalidProperty : Property := { authoredProperty with id := DefinitionId.of "" }

private def invalidScenario : Scenario :=
  { exactActionBehaviorDeclaration with id := DefinitionId.of "" }

private def knownGap : KnownGap := {
  kind := .claim
  code := DefinitionId.of "switch.known-gap.admission"
  subject := some flipPropertyId
  detail := some "An admission test limitation."
}

private def invalidLimitsShape : Query.Shape := { shape with limits := Limits.bounded 0 1 8 }

/-- The switch Model with no finite planning capability: its Queries check, but no search view
exists over it. -/
private def unsearchableTarget : QueryModel LawStatement :=
  model (DraftModel.make modelSpec modelProviders)

private def rejections : List (Option AdmissionDiagnostic) := [
  errorOf (Search.admit target invalidProperty (some exactActionBehaviorDeclaration) shape),
  errorOf (Search.admit target authoredProperty (some invalidScenario) shape),
  errorOf (Search.admit target authoredProperty (some exactActionBehaviorDeclaration) shape
    [knownGap, knownGap]),
  errorOf (Search.admit target authoredProperty (some exactActionBehaviorDeclaration)
    invalidLimitsShape),
  errorOf (Search.admit unsearchableTarget authoredProperty (some exactActionBehaviorDeclaration)
    shape)
]

/-! Each stage's rejection is its own constructor, located at that stage, carrying the typed error
the stage returns on its own. -/
example : rejections.map (·.map (·.located.stage)) =
    [some .property, some .scenario, some .knownGaps, some .query, some .searchView] := by
  native_decide

example : rejections = [
    (errorOf (Property.check (PropertyCheckContext.ofTarget target) invalidProperty)).map
      AdmissionDiagnostic.property,
    (errorOf (Scenario.check (.ofTarget target) invalidScenario)).map AdmissionDiagnostic.scenario,
    (errorOf (KnownGapSet.checkCanonical [knownGap, knownGap])).map AdmissionDiagnostic.knownGaps,
    (errorOf (Query.check (.ofTarget target) (invalidLimitsShape.toQuery flipProperty
      exactActionBehavior))).map AdmissionDiagnostic.query,
    (errorOf (SearchView.ofCheckedQuery unsearchableTarget.id
      (Query.checked unsearchableTarget (shape.toQuery
        (Property.checked (PropertyCheckContext.ofTarget unsearchableTarget) authoredProperty)
        (Scenario.checked (.ofTarget unsearchableTarget) exactActionBehaviorDeclaration))))).map
      AdmissionDiagnostic.searchView
  ] := by
  native_decide

/-! A located diagnostic names the declaration it rejects and the source path its stage reports. -/
example : (rejections.map (·.map fun diagnostic =>
    (diagnostic.located.definitionId.value, diagnostic.located.sourcePath))) = [
    some ("umpire.property.anonymous", some source.path),
    some ("umpire.behavior.anonymous", some source.path),
    some ("switch.known-gap.admission", none),
    some (exactActionQueryId.value, some source.path),
    some (targetId.value, none)
  ] := by
  native_decide

/-! Admit then search yields the staged `PlanResult` and the same checked declarations. -/
example :
    exactActionAdmitted.search.toOption = (search exactActionQuery
      ((SearchView.ofCheckedQuery target.id exactActionQuery).toOption.get
        (by native_decide))).toOption ∧
    (exactActionAdmitted.query.id, exactActionAdmitted.query.canonicalMetadata,
      exactActionAdmitted.scenario.id, exactActionAdmitted.property.id) =
    (exactActionQuery.id, exactActionQuery.canonicalMetadata, exactActionBehavior.id,
      flipProperty.id) := by
  native_decide

/-! A Property-only Query admits: it searches a Scenario that constrains no Trace. -/
example : (Search.admit target authoredProperty none
    (shape (DefinitionId.of "switch.query.property-only"))).toOption.map
    (fun admitted => (admitted.scenario.id.value,
      admitted.search.toOption.map fun (run : PlanResult) => run.result.outcome.name)) =
    some ("switch.query.property-only.scenario", some "found") := by
  native_decide

/-! `withQuery` searches another checked Query through the admitted view, as a fresh view would. -/
example :
    (exactActionAdmitted.withQuery exactTraceQuery rfl).search.toOption = exactTraceRun.toOption ∧
    (exactActionAdmitted.withQuery exploratoryQuery rfl).analyzeBranches =
      analyzeBranches exploratoryQuery
        ((SearchView.ofCheckedQuery target.id exploratoryQuery).toOption.get
          (by native_decide)) := by
  native_decide

end Umpire.SearchTests.Admission
