import Umpire.Property.Tests.Fixtures

/-! Canonical ordering and semantic-digest sensitivity checks. -/

namespace Umpire.PropertyTests

open Umpire

def reorderedContext : PropertyCheckContext := {
  context with
  declarations := context.declarations.reverse
  providers := context.providers.reverse
  meanings := context.meanings.reverse
}

def reorderedProperty : PropertyDeclaration := {
  portableProperty with
  clauses := portableProperty.clauses.reverse
}

def canonicalOf
    (check : Except PropertyError CheckedProperty) : Option String :=
  check.toOption.map canonicalPropertyJson

def digestOf
    (check : Except PropertyError CheckedProperty) : Option String :=
  check.toOption.map CheckedProperty.semanticDigest

example : canonicalOf (checkProperty context authoredProperty) =
    canonicalOf (checkProperty reorderedContext (.portable reorderedProperty)) := by
  native_decide

def changedConstructor : PropertyDeclaration := {
  portableProperty with
  clauses := portableProperty.clauses.map fun clause =>
    if clause.id == honoredDelivery.id then
      .quiescentWithin honoredDelivery.id
        (pattern .observation cancelRequested)
        (pattern .observation cancelDelivered)
        (.exact cancelBudget.bound)
    else
      clause
}

def changedReference : PropertyDeclaration := {
  portableProperty with
  clauses := portableProperty.clauses.map fun clause =>
    if clause.id == cancelIsUnique.id then
      .stateInvariant cancelIsUnique.id
        (pattern .state cancellationPhase (.naturalAtMost 1))
    else
      clause
}

def changedBound : PropertyDeclaration := {
  portableProperty with
  clauses := portableProperty.clauses.map fun clause =>
    if clause.id == honoredDelivery.id then
      .eventuallyWithin honoredDelivery.id
        (pattern .observation cancelRequested)
        (pattern .observation cancelDelivered)
        (.exact { value := 3, unit := .observationPositions })
    else
      clause
}

example : digestOf (checkProperty context authoredProperty) ≠
    digestOf (checkProperty context (.portable changedConstructor)) := by
  native_decide

example : digestOf (checkProperty context authoredProperty) ≠
    digestOf (checkProperty context (.portable changedReference)) := by
  native_decide

example : digestOf (checkProperty context authoredProperty) ≠
    digestOf (checkProperty context (.portable changedBound)) := by
  native_decide

end Umpire.PropertyTests
