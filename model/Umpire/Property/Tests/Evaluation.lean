import Umpire.Property.Tests.Fixtures

/-! Successful checking, focused evaluation, boundaries, hidden observations, and result evidence. -/

namespace Umpire.PropertyTests

open Umpire

def negativeTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  positiveTrace with
  steps := positiveTrace.steps.mapIdx fun index step =>
    if index == 0 then { step with resultingState := value pendingCount "2" } else step
}

def uniquenessProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.uniqueness-only"
  clauses := [cancelIsUnique]
}

example : errorKindOf (checkProperty context authoredProperty) = none := by
  native_decide

example : (evaluationOf portableProperty positiveTrace).map PropertyEvaluation.satisfied = some true := by
  native_decide

example : (evaluationOf uniquenessProperty negativeTrace).map PropertyEvaluation.satisfied = some false := by
  native_decide

def samePositionBoundary : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.same-position-boundary"
  clauses := [
    .eventuallyWithin (id "test.property.same-position-boundary.clause")
      (pattern .observation cancelDelivered)
      (pattern .observation cancelDelivered)
      (.exact { value := 0, unit := .observationPositions })
  ]
}

example :
    (evaluationOf samePositionBoundary positiveTrace).map PropertyEvaluation.satisfied = some true := by
  native_decide

/-- The reusable theorem applies to positive, negative, and boundary fixtures without a
constructor-specific proof escape hatch. -/
example (clause : ResolvedPropertyClause) :
    ∀ view : PropertyTraceView,
      evaluatePropertyClause clause view = true ↔ clause.denote view := by
  intro view
  exact evaluatePropertyClause_agrees clause view

def hiddenReference : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.hidden-reference"
  clauses := [
    .identityRelation (id "test.property.hidden-reference.clause")
      (pattern .observation hiddenObservation)
  ]
}

example :
    errorKindOf (checkProperty context (.portable hiddenReference)) = some .undeclaredReference := by
  native_decide

def admittedObservationIds : Option (List DefinitionId) :=
  (checkProperty context authoredProperty).toOption.map fun property =>
    (property.traceView positiveTrace).steps.flatMap fun step =>
      step.observations.map ModelValue.definitionId

example : admittedObservationIds.map (fun ids => ids.contains hiddenObservation) = some false := by
  native_decide

def traceWithoutHidden : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  positiveTrace with
  steps := positiveTrace.steps.map fun step => {
    step with
    observations := step.observations.filter fun observation =>
      observation.definitionId != hiddenObservation
  }
}

example : evaluationOf portableProperty positiveTrace = evaluationOf portableProperty traceWithoutHidden := by
  native_decide

def focusedClauseResult : Option PropertyClauseResult := do
  let evaluation ← evaluationOf portableProperty positiveTrace
  evaluation.clauses.find? fun result => result.clauseId == honoredDelivery.id

example : focusedClauseResult.map PropertyClauseResult.evaluatedBound =
    some (some cancelBudget.bound) := by
  native_decide

example : focusedClauseResult.map (fun result =>
    result.traceSpan.isSome && !result.semanticProvenance.isEmpty) = some true := by
  native_decide

end Umpire.PropertyTests
