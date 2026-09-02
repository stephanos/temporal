import Umpire.Property.Tests.Fixtures

/-! Malformed Property definitions and authoring-mode validation checks. -/

namespace Umpire.PropertyTests

open Umpire

def mixedUnitProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.mixed-unit"
  clauses := [
    .eventuallyWithin (id "test.property.mixed-unit.clause")
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      (.named cancelBudget.id .selectedActions)
  ]
}

example :
    errorKindOf (checkProperty context (.portable mixedUnitProperty)) = some .unitMismatch := by
  native_decide

def candidateEvaluationLimit : PropertyLimitProfile := {
  id := id "test.limit.candidate-evaluations"
  source
  limit := { value := 2, unit := .candidateEvaluations }
}

def candidateEvaluationContext : PropertyCheckContext := {
  context with limitProfiles := candidateEvaluationLimit :: context.limitProfiles
}

def candidateEvaluationProperty (limit : PropertyLimit) : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.candidate-evaluations"
  clauses := [
    .eventuallyWithin (id "test.property.candidate-evaluations.clause")
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      limit
  ]
}

/-- Query's candidate-evaluation Limit is rejected in both exact and named Property forms. -/
example : [
    errorKindOf (checkProperty context (.portable <|
      candidateEvaluationProperty (.exact candidateEvaluationLimit.limit))),
    errorKindOf (checkProperty candidateEvaluationContext (.portable <|
      candidateEvaluationProperty (.named candidateEvaluationLimit.id .candidateEvaluations)))
  ] = [some .unitMismatch, some .unitMismatch] := by
  native_decide

/-! Exploration's ExperimentSpec Limit is not a Property position unit. -/
example : errorKindOf (checkProperty context (.portable <|
    candidateEvaluationProperty (.exact { value := 2, unit := .experimentSpecs }))) =
    some .unitMismatch := by
  native_decide

def missingLogicalTimeProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.missing-logical-time"
  clauses := [
    .eventuallyWithin (id "test.property.missing-logical-time.clause")
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      (.exact { value := 1, unit := .logicalTime })
  ]
}

example :
    errorKindOf (checkProperty context (.portable missingLogicalTimeProperty)) =
      some .missingLogicalTimeSource := by
  native_decide

example :
    errorKindOf (checkProperty context
      (.opaque (id "test.property.expert-only") source)) = some .opaqueDeclaration := by
  native_decide

end Umpire.PropertyTests
