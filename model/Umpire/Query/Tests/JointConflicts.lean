import Umpire.Search.Tests.CaseAnalysis

/-! Bounded joint-Property conflict analysis over checked Query inputs. -/

namespace Umpire.QueryTests.JointConflicts

open Umpire
open Umpire.SearchTests

private def capability : DefinitionId := id "planner.capability.joint-analysis"

private def meaning (definitionId : DefinitionId) (kind : DefinitionKind) : Meaning := {
  definitionId
  kind
  behaviorVersion := definitionId.value ++ "/joint-analysis-v1"
}

private def propertyContext : PropertyCheckContext := {
  definitions := [
    metadata capability .capability "planner-joint-analysis/v1",
    metadata phase .state "planner-phase/v1",
    metadata request .action "planner-request/v1",
    metadata accepted .outcome "planner-accepted/v1",
    metadata observed .fact "planner-observed/v1"
  ]
  providers := [{
    id := capability
    version := 1
    behaviorVersion := "planner-joint-analysis/v1" }]
  meanings := [
    (capability, meaning phase .state),
    (capability, meaning request .action),
    (capability, meaning accepted .outcome),
    (capability, meaning observed .fact)
  ]
}

private def exactAtom
    (field : PropertyPredicateField)
    (reference : DefinitionId)
    (constraint : PropertyAtomConstraint) : PropertyPredicate :=
  .atom { field, reference, constraint }

private def requestGuard : PropertyPredicate :=
  exactAtom .selectedAction request (.equals (.text requestValue.value))

private def initialGuard : PropertyPredicate :=
  exactAtom .priorState phase (.equals (.text initial.value))

private def completedGuard : PropertyPredicate :=
  exactAtom .priorState phase (.equals (.text completed.value))

private def declaration
    (key : String)
    (constraints : List PropertyAtomConstraint)
    (field : PropertyPredicateField := .resultingState)
    (reference : DefinitionId := phase)
    (caseGuard : PropertyPredicate := initialGuard) : Property := {
  id := id ("planner.property.joint." ++ key)
  source
  version := 2
  requires := [capability]
  clauses := [.branches {
    id := id ("planner.property.joint." ++ key ++ ".group")
    source
    guard := requestGuard
    cases := [{
      id := id ("planner.property.joint." ++ key ++ ".case")
      source
      guard := caseGuard
      clauses := constraints.zipIdx.map fun (constraint, index) => {
        id := id ("planner.property.joint." ++ key ++ ".clause-" ++ toString index)
        source
        expectation := exactAtom field reference constraint
      }
    }]
  }]
}

private def checkedProperties? (declarations : List Property) : Option (List CheckedProperty) :=
  declarations.mapM fun selected =>
    (Property.check propertyContext (selected)).toOption

private def jointQuery
    (properties : List CheckedProperty)
    (budget : Nat := 8)
    (selectedBehavior : CheckedScenario := behavior) : CheckedQuery (fun _ => True) :=
  let form := QueryForm.select properties
  { SearchTests.checkedQuery 0 form .exhaustive budget 17 true selectedBehavior with
    form
    quantifier := form.quantifier
    claim := form.claim
  }

private def analyze?
    (declarations : List Property)
    (budget : Nat := 8)
    (selectedBehavior : CheckedScenario := behavior) : Option BranchAnalysisResult := do
  let properties ← checkedProperties? declarations
  let query := jointQuery properties budget selectedBehavior
  let kernel ← (SearchView.ofCheckedQuery query.target.id query).toOption
  pure (analyzeBranches query kernel)

private def textSet (values : List String) : PropertyAtomConstraint :=
  .oneOf (values.map PropertyLiteral.text)

/-! A single violated expectation remains ordinary Property evidence, not a joint conflict. -/
#guard (analyze? [declaration "single-violation" [.equals (.text "other")]]).map
    (fun result =>
      (result.joint.status.name, result.joint.logicalConflicts.isEmpty,
        result.joint.modelIncompatibilities.isEmpty,
        result.propertyEvaluations.any fun evaluation => !evaluation.evaluation.satisfied)) ==
  some ("unexercised", true, true, true)

/-! Compatible overlap remains conjoined and one admitted continuation satisfies every clause. -/
#guard (analyze? [declaration "compatible" [
      textSet [completed.value, "other"],
      textSet [completed.value, "third"]
    ]]).map (fun result =>
      (result.joint.status.name,
        result.joint.triggers.head?.map (fun trigger =>
          (trigger.expectations.length, trigger.satisfyingContinuations.length)))) ==
  some ("compatible-within-limits", some (2, 1))

/-! Logical contradiction aggregates every constraint; pairwise intersections are insufficient. -/
#guard (analyze? [declaration "three-way" [
      textSet ["a", "b"],
      textSet ["b", "c"],
      textSet ["a", "c"]
    ]]).map (fun result =>
      (result.joint.status.name,
        result.joint.logicalConflicts.head?.map fun conflict =>
          (conflict.field, conflict.reference, conflict.constraints.length,
            conflict.expectations.map OverlapExpectationEvidence.clauseId))) ==
  some ("logical-conflict", some (
    PropertyPredicateField.resultingState,
    phase,
    3,
    [SearchTests.id "planner.property.joint.three-way.clause-0",
      SearchTests.id "planner.property.joint.three-way.clause-1",
      SearchTests.id "planner.property.joint.three-way.clause-2"]))

/-! Set-valued fact projections can satisfy distinct equalities and are not scalar contradictions. -/
#guard (analyze? [declaration "set-valued-facts" [
      .equals (.text observedValue.value),
      .equals (.text "another")
    ] .expectationFact observed]).map (fun result =>
      (result.joint.logicalConflicts.isEmpty,
        result.joint.modelIncompatibilities.length)) == some (true, 1)

/-! A nonempty logical intersection can still be absent from every modeled continuation. -/
#guard (analyze? [declaration "model-relative" [
      textSet ["shared", "left"],
      textSet ["shared", "right"]
    ]]).map (fun result =>
      (result.joint.status.name,
        result.joint.logicalConflicts.isEmpty,
        result.joint.modelIncompatibilities.head?.map fun conflict =>
          (conflict.trigger.modeledPrefix.trace.steps.length,
            conflict.trigger.limits.search,
            conflict.admittedContinuations.length))) ==
  some ("model-incompatible", true, some (
    0,
    ({ value := 8, unit := .search } : Limit),
    1))

/-! No common trigger, no admitted Behavior trace, and exhausted work remain separate outcomes. -/
#guard (analyze? [declaration "no-trigger" [.present, .present]
    (caseGuard := completedGuard)]).map (fun result => result.joint.status.name) ==
  some "unexercised"

private def deadEndBehavior : CheckedScenario := {
  behavior with
  actionsExactly := some [request, request]
  behaviorFingerprint := behaviorFingerprintOf "behavior/joint-dead-end-v1"
}

#guard (analyze? [declaration "dead-end" [.present, .present]] 8 deadEndBehavior).map
    (fun result => (result.status.name, result.joint.status.name,
      result.joint.logicalConflicts.isEmpty, result.joint.modelIncompatibilities.isEmpty)) ==
  some ("unsatisfiable", "dead-end", true, true)

private def unsatisfiableBehavior : CheckedScenario := {
  deadEndBehavior with
  spaceStatus := .unsatisfiable
  behaviorFingerprint := behaviorFingerprintOf "behavior/joint-unsatisfiable-v1"
}

#guard (analyze? [declaration "unsatisfiable" [.present, .present]] 8
    unsatisfiableBehavior).map (fun result =>
      (result.joint.status.name, result.joint.logicalConflicts.isEmpty,
        result.joint.modelIncompatibilities.isEmpty)) == some ("unsatisfiable", true, true)

#guard (analyze? [declaration "limit" [.present, .present]] 1).map
    (fun result => (result.joint.status.name,
      result.metadata.completeness.established,
      result.joint.modelIncompatibilities.isEmpty)) ==
  some ("limit-reached", false, true)

/-! Unsupported Boolean formula classes are explicit while bounded Property evaluation remains. -/
private def withUnsupportedAny : Property :=
  let base := declaration "unsupported" [.present]
  { base with clauses := base.clauses.map fun clause => match clause with
    | .branches group => .branches { group with cases := group.cases.map fun item =>
        { item with clauses := item.clauses.map fun clause =>
          { clause with expectation := .any [
              exactAtom .resultingState phase (.equals (.text completed.value)),
              exactAtom .resultingState phase (.equals (.text "other"))
            ] } } }
    | other => other }

#guard (analyze? [withUnsupportedAny]).map (fun result =>
      (result.joint.status.name,
      result.joint.unsupported.map (fun unsupported => unsupported.formulaClass.name),
      result.propertyEvaluations.all fun evaluation => evaluation.evaluation.satisfied)) ==
  some ("unsupported", ["any"], true)

/-! Property declaration order cannot select a winner or change canonical joint evidence. -/
private def leftDeclaration : Property :=
  declaration "order-left" [textSet [completed.value, "left"]]

private def rightDeclaration : Property :=
  declaration "order-right" [textSet [completed.value, "right"]]

#guard (analyze? [leftDeclaration, rightDeclaration]).map (fun result => result.joint) ==
  (analyze? [rightDeclaration, leftDeclaration]).map (fun result => result.joint)

private def caseOrderDeclaration (reverse : Bool) : Property :=
  let base := declaration "case-order" [.equals (.text completed.value)]
  { base with clauses := base.clauses.map fun clause => match clause with
    | .branches group =>
        match group.cases with
        | first :: _ =>
            let second := { first with
              id := id "planner.property.joint.case-order.case-second"
              clauses := first.clauses.map fun clause => { clause with
                id := id "planner.property.joint.case-order.clause-second" }
            }
            .branches {
              group with cases := if reverse then [second, first] else [first, second]
            }
        | [] => .branches group
    | other => other }

/-! Case source order also preserves the complete joint evidence; neither branch has priority. -/
#guard (analyze? [caseOrderDeclaration false]).map (fun result => result.joint) ==
  (analyze? [caseOrderDeclaration true]).map (fun result => result.joint)

end Umpire.QueryTests.JointConflicts
