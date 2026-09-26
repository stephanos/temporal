import Umpire.Search
import Umpire.Search.Branches

/-!
# Admitting a Query against one checked Model

Before anything can be searched, five checks run in a fixed order: the Property, the Scenario, the
authored Known Gaps, the Query that pairs them, and the search view over the Model. `Search.admit`
owns that chain. It returns either the first stage that rejected, as one `AdmissionDiagnostic`
carrying that stage's own typed error unchanged, or an `AdmittedQuery`.

An `AdmittedQuery` is indexed by the checked Model, like the search view it holds. The view is
private: `AdmittedQuery.search`, `searchWithIntent` and `analyzeBranches` are the only ways to use
it, so no caller rebuilds or transports a view by hand.

Two transports exist, and both live in this package. `SearchView.retarget` moves a view across a
proved Model equality. `AdmittedQuery.withQuery` re-pairs an admitted view with another checked
Query over the same Model, which is what the Variations compiler and Promotion do: they hold one
view and search many Queries derived from one base.
-/

namespace Umpire

/-- Everything an authored Query states apart from the Property and Scenario `Search.admit` checks
itself: the claim it makes of that Property, its identity, and the bounds and policy of its search.
Every field is one `CheckedQuery.id`, the canonical Query metadata, or the planner reads. -/
structure Query.Shape where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  target : DefinitionId
  /-- The claim the Query makes of the checked Property. -/
  form : CheckedProperty → Query.Form
  limits : Limits
  policy : PlannerPolicy
  ending : Query.Ending := .final
  requireFiring : Bool := false
  documentation : String := ""

/-- The authored Query this shape describes once its Property, Scenario and Known Gaps are
checked. -/
def Query.Shape.toQuery
    (shape : Query.Shape)
    (property : CheckedProperty)
    (behavior : CheckedScenario)
    (knownGaps : KnownGapSet := KnownGapSet.empty) : Query := {
  id := shape.id
  source := shape.source
  version := shape.version
  target := shape.target
  form := shape.form property
  behavior
  limits := shape.limits
  policy := shape.policy
  ending := shape.ending
  requireFiring := shape.requireFiring
  authoredKnownGaps := knownGaps
  documentation := shape.documentation
}

/-- The Scenario a Property-only Query searches, named after the Query. It declares each setup
role the Model binds, in first-binding order and with the kind of the value bound to it, and
constrains no Trace. A Model whose setups bind different role sets admits only the setups binding
all of them. -/
def Query.Shape.unconstrainedScenario
    (shape : Query.Shape)
    (target : QueryModel LawStatement) : Scenario := {
  id := DefinitionId.of (shape.id.value ++ ".scenario")
  source := shape.source
  roles := target.resolvedSetups.flatten.foldl (fun roles binding =>
    if roles.any (·.id == binding.role) then
      roles
    else
      let kind := (target.definitions.find? (·.id == binding.value.definitionId)).map
        DefinitionMetadata.kind
      roles ++ [{ id := binding.role, valueKind := kind.getD .state }]) []
}

/-- The stage of an admission that rejected. -/
inductive AdmissionStage where
  | property
  | scenario
  | knownGaps
  | query
  | searchView
  deriving BEq, DecidableEq, Repr

/-- The first admission stage that rejected, carrying the typed error that stage returns on its own.
Stages run in constructor order, so a declaration wrong in two places reports the earlier one. -/
inductive AdmissionDiagnostic where
  | property (error : PropertyError)
  | scenario (error : ScenarioError)
  | knownGaps (error : KnownGapError)
  | query (error : QueryError)
  | searchView (error : FiniteSearchAdmissionError)
  deriving BEq, DecidableEq, Repr

/-- Where one admission diagnostic belongs: the stage, the Definition ID it names, and the source
path the stage's error reports, when it reports one. -/
structure AdmissionLocation where
  stage : AdmissionStage
  definitionId : DefinitionId
  sourcePath : Option String
  deriving BEq, DecidableEq, Repr

/-- Locate a diagnostic at the declaration it rejects. A Known Gap names its own code, and a search
view names the Model the Query selected; neither error reports a source path. -/
def AdmissionDiagnostic.located : AdmissionDiagnostic → AdmissionLocation
  | .property error => {
      stage := .property, definitionId := error.definitionId, sourcePath := some error.sourcePath }
  | .scenario error => {
      stage := .scenario, definitionId := error.definitionId, sourcePath := some error.sourcePath }
  | .knownGaps error => { stage := .knownGaps, definitionId := error.code, sourcePath := none }
  | .query error =>
      { stage := .query, definitionId := error.definitionId, sourcePath := some error.sourcePath }
  | .searchView error =>
      { stage := .searchView, definitionId := error.actualTarget, sourcePath := none }

/-- A Query admitted against one checked Model, with the search view over that Model. -/
structure AdmittedQuery (target : QueryModel LawStatement) where
  private mk ::
  /-- The Property admission checked. A Query re-paired through `withQuery` keeps it. -/
  property : CheckedProperty
  query : CheckedQuery LawStatement
  targetEq : query.target = target
  private view : SearchView target

namespace AdmittedQuery

variable {LawStatement : Law → Prop} {target : QueryModel LawStatement}

/-- The checked Scenario the admitted Query searches. -/
def scenario (admitted : AdmittedQuery target) : CheckedScenario :=
  admitted.query.behavior

/-- Search the admitted Query through its own view. -/
def search (admitted : AdmittedQuery target) : Except KnownGapError PlanResult :=
  Umpire.search admitted.query (admitted.view.retarget admitted.targetEq.symm)

/-- Search the admitted Query and project the checked Artifact intent onto a selected Plan. -/
def searchWithIntent
    (admitted : AdmittedQuery target)
    (intent : PlanRequest) : Except PlanningRequestError PlanResult :=
  searchWithPlanRequest admitted.query (admitted.view.retarget admitted.targetEq.symm) intent

/-- Analyze the admitted Query's guarded cases over its bounded candidate stream. -/
def analyzeBranches (admitted : AdmittedQuery target) : BranchAnalysisResult :=
  Umpire.analyzeBranches admitted.query (admitted.view.retarget admitted.targetEq.symm)

/-- Pair this admission's view with another checked Query over the same Model. -/
def withQuery
    (admitted : AdmittedQuery target)
    (query : CheckedQuery LawStatement)
    (targetEq : query.target = target) : AdmittedQuery target :=
  { admitted with query, targetEq }

/-- Move an admitted Query across a proved Model equality. -/
def retarget
    {target' : QueryModel LawStatement}
    (admitted : AdmittedQuery target)
    (targetEq : target = target') : AdmittedQuery target' := {
  property := admitted.property
  query := admitted.query
  targetEq := admitted.targetEq.trans targetEq
  view := admitted.view.retarget targetEq
}

end AdmittedQuery

namespace Search

/--
Admit one Query against a checked Model. The stages run in order -- Property, Scenario, Known Gaps,
Query, search view -- and the first that rejects is the result, with its own typed error.

`scenario := none` admits a Property-only Query: it searches `Query.Shape.unconstrainedScenario`,
which constrains no Trace. The checked Query's Model is re-ascribed to `target`, as
`Query.checked` does, so the admitted view and Query share one index.
-/
def admit
    (target : QueryModel LawStatement)
    (property : Property)
    (scenario : Option Scenario)
    (shape : Query.Shape)
    (knownGaps : List KnownGap := []) :
    Except AdmissionDiagnostic (AdmittedQuery target) := do
  let checkedProperty ← Property.check (PropertyCheckContext.ofTarget target) property
    |>.mapError AdmissionDiagnostic.property
  let behavior ← Scenario.check (.ofTarget target)
      (scenario.getD (shape.unconstrainedScenario target))
    |>.mapError AdmissionDiagnostic.scenario
  let gaps ← KnownGapSet.checkCanonical knownGaps |>.mapError AdmissionDiagnostic.knownGaps
  let checked ← Query.check (.ofTarget target) (shape.toQuery checkedProperty behavior gaps)
    |>.mapError AdmissionDiagnostic.query
  let query : CheckedQuery LawStatement := {
    checked with
    target
    completeness := (ModelCompleteness.ofTarget target).completeness
  }
  let view ← SearchView.ofCheckedQuery target.id query |>.mapError AdmissionDiagnostic.searchView
  pure ⟨checkedProperty, query, rfl, view⟩

end Search

end Umpire
