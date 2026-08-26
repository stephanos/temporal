import Umpire.Behavior.Tests.Fixtures

/-! Authoring errors, unsatisfiability, schedule contradictions, and occurrence guards. -/

namespace Umpire.BehaviorTests

open Umpire

def actualErrorKind : Except BehaviorError CheckedBehavior → Option BehaviorErrorKind
  | .ok _ => none
  | .error error => some error.kind

def cyclicDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  ordering := [
    { before := closeOccurrence.id, after := cancelOccurrence.id },
    { before := cancelOccurrence.id, after := closeOccurrence.id }
  ]
}

def invalidBindingDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  setup := [{
    id := id "test.setup.missing-role"
    relation := .equal
    left := .role (id "test.role.missing")
    right := .value operationA
  }]
}

def contradictoryCountDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  occurrenceBounds := [{ action := requestCancel, minimum := 2, maximum := some 1 }]
}

def forbiddenRequiredDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  allowedActions := [callerClose, tick]
  forbiddenActions := [requestCancel]
}

def incompleteExactDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  traceExactly := some {
    exactWitness with
    steps := exactWitness.steps.modifyHead fun step => { step with observations := none }
  }
}

example : [
    actualErrorKind (checkBehavior context cyclicDeclaration),
    actualErrorKind (checkBehavior context invalidBindingDeclaration),
    actualErrorKind (checkBehavior context contradictoryCountDeclaration),
    actualErrorKind (checkBehavior context forbiddenRequiredDeclaration),
    actualErrorKind (checkBehavior context incompleteExactDeclaration)
  ] = [
    some .cyclicOrdering,
    some .invalidBinding,
    some .contradictoryOccurrenceBounds,
    some .forbiddenRequired,
    some .incompleteExactTrace
  ] := by
  native_decide

def canonicalError (declaration : BehaviorDeclaration) : Option String :=
  match checkBehavior context declaration with
  | .ok _ => none
  | .error error => some (canonicalBehaviorErrorJson error)

example : canonicalError cyclicDeclaration = canonicalError {
    cyclicDeclaration with ordering := cyclicDeclaration.ordering.reverse
  } := by
  native_decide

/-- An empty semantic space is a checked result, distinct from invalid authoring. -/
def unsatisfiableDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with
  setup := [{
    id := id "test.setup.impossible"
    relation := .different
    left := .role operationRole.id
    right := .role operationRole.id
  }]
}

example : (checkBehavior context unsatisfiableDeclaration).toOption.map
    CheckedBehavior.isUnsatisfiable = some true := by
  native_decide

example : !checkedAdmits unsatisfiableDeclaration acceptedTrace := by native_decide

def pairedSetupConflict : BehaviorDeclaration := {
  constrainedDeclaration with
  setup := [
    setupEqualsA,
    {
      id := id "test.setup.resource-not-a"
      relation := .different
      left := .role operationRole.id
      right := .value operationA
    }
  ]
}

example : (checkBehavior context pairedSetupConflict).toOption.map
    CheckedBehavior.isUnsatisfiable = some true := by
  native_decide

def exactSequenceConflict : BehaviorDeclaration := {
  id := id "test.behavior.exact-sequence-conflict"
  source
  roles := [operationRole]
  actionsExactly := some [requestCancel]
  sequences := [[callerClose]]
}

def exactAdjacencyConflict : BehaviorDeclaration := {
  exactSequenceConflict with
  sequences := []
  adjacencies := [[requestCancel, callerClose]]
}

def exactOrderingConflict : BehaviorDeclaration := {
  exactSequenceConflict with
  requiredOccurrences := [cancelOccurrence, closeOccurrence]
  ordering := [{ before := cancelOccurrence.id, after := closeOccurrence.id }]
  actionsExactly := some [callerClose, requestCancel]
  sequences := []
}

def exactTraceSequenceConflict : BehaviorDeclaration := {
  constrainedDeclaration with
  traceExactly := some exactWitness
  sequences := [[callerClose, requestCancel]]
}

/-! Mechanically contradictory exact schedules and traces fail during Behavior checking. -/
example : [
    actualErrorKind (checkBehavior context exactSequenceConflict),
    actualErrorKind (checkBehavior context exactAdjacencyConflict),
    actualErrorKind (checkBehavior context exactOrderingConflict),
    actualErrorKind (checkBehavior context exactTraceSequenceConflict)
  ] = List.replicate 4 (some .contradictoryConstraint) := by
  native_decide

def manyCancelOccurrences : List NamedOccurrence :=
  (List.range 15).map fun index => {
    id := id ("test.occurrence.cancel-" ++ toString index)
    action := requestCancel
  }

def countDeficitDeclaration : BehaviorDeclaration := {
  constrainedDeclaration with requiredOccurrences := manyCancelOccurrences
}

/-- The checked authoring bound fails closed before occurrence-state exploration can explode. -/
example : actualErrorKind (checkBehavior context countDeficitDeclaration) =
    some .occurrenceLimitExceeded := by
  native_decide

end Umpire.BehaviorTests
