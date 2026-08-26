import Umpire.Property.Tests.Fixtures

/-! Logical-time Property variants, traces, and evaluation checks. -/

namespace Umpire.PropertyTests

open Umpire

def logicalEventuallyProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.logical-eventually"
  logicalTimeSource := some logicalTime
  clauses := [
    .eventuallyWithin (id "test.property.logical-eventually.clause")
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      (.exact { value := 1, unit := .logicalTime })
  ]
}

def logicalQuiescentProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.logical-quiescent"
  logicalTimeSource := some logicalTime
  clauses := [
    .quiescentWithin (id "test.property.logical-quiescent.clause")
      (pattern .observation cancelDelivered)
      (pattern .observation cancelRequested)
      (.exact { value := 0, unit := .logicalTime })
  ]
}

def traceWithLogicalTime
    (first second : String) :
    SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue := {
  positiveTrace with
  steps := positiveTrace.steps.mapIdx fun index step => {
    step with
    observations := step.observations ++ [value logicalTime (if index == 0 then first else second)]
  }
}

example :
    (evaluationOf logicalEventuallyProperty (traceWithLogicalTime "1" "2")).map
      PropertyEvaluation.satisfied = some true := by
  native_decide

example :
    (evaluationOf logicalQuiescentProperty (traceWithLogicalTime "1" "2")).map
      PropertyEvaluation.satisfied = some true := by
  native_decide

example :
    (evaluationOf logicalEventuallyProperty positiveTrace).map PropertyEvaluation.satisfied =
      some false := by
  native_decide

example :
    (evaluationOf logicalQuiescentProperty positiveTrace).map PropertyEvaluation.satisfied =
      some false := by
  native_decide

example :
    (evaluationOf logicalEventuallyProperty (traceWithLogicalTime "not-a-time" "2")).map
      PropertyEvaluation.satisfied = some false := by
  native_decide

example :
    (evaluationOf logicalQuiescentProperty (traceWithLogicalTime "not-a-time" "2")).map
      PropertyEvaluation.satisfied = some false := by
  native_decide

example :
    ((evaluationOf logicalEventuallyProperty (traceWithLogicalTime "2" "1")).map
        PropertyEvaluation.satisfied,
      (evaluationOf logicalQuiescentProperty (traceWithLogicalTime "2" "1")).map
        PropertyEvaluation.satisfied) = (some false, some false) := by
  native_decide

end Umpire.PropertyTests
