import Umpire.Property.Tests.Fixtures

/-! Canonical ordering and Behavior Fingerprint sensitivity checks. -/

namespace Umpire.PropertyTests

open Umpire

def reorderedContext : PropertyCheckContext := {
  context with
  definitions := context.definitions.reverse
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

def fingerprintOf
    (check : Except PropertyError CheckedProperty) : Option BehaviorFingerprint :=
  check.toOption.map CheckedProperty.behaviorFingerprint

example : canonicalOf (checkProperty context authoredProperty) =
      canonicalOf (checkProperty reorderedContext (.portable reorderedProperty)) ∧
    fingerprintOf (checkProperty context authoredProperty) =
      fingerprintOf (checkProperty reorderedContext (.portable reorderedProperty)) := by
  native_decide

def changedSourceProperty : PropertyDeclaration := {
  portableProperty with
  source := { source with line := source.line + 1 }
}

def changedDocumentationProperty : PropertyDeclaration := {
  portableProperty with
  documentation := "Updated Property documentation."
}

example : [
    fingerprintOf (checkProperty context (.portable changedSourceProperty)),
    fingerprintOf (checkProperty context (.portable changedDocumentationProperty))
  ] = [
    fingerprintOf (checkProperty context authoredProperty),
    fingerprintOf (checkProperty context authoredProperty)
  ] := by
  native_decide

def changedCapabilityContext : PropertyCheckContext := {
  context with
  providers := context.providers.map fun capability =>
    if capability.id == cancellationCapability then
      { capability with canonicalBehavior := "test-cancellation/v2" }
    else
      capability
}

example : fingerprintOf (checkProperty context authoredProperty) ≠
    fingerprintOf (checkProperty changedCapabilityContext authoredProperty) := by
  native_decide

def changedConstructor : PropertyDeclaration := {
  portableProperty with
  clauses := portableProperty.clauses.map fun clause =>
    if clause.id == honoredDelivery.id then
      .quiescentWithin honoredDelivery.id
        (pattern .observation cancelRequested)
        (pattern .observation cancelDelivered)
        (.exact cancelBudget.limit)
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

example : fingerprintOf (checkProperty context authoredProperty) ≠
    fingerprintOf (checkProperty context (.portable changedConstructor)) := by
  native_decide

example : fingerprintOf (checkProperty context authoredProperty) ≠
    fingerprintOf (checkProperty context (.portable changedReference)) := by
  native_decide

example : fingerprintOf (checkProperty context authoredProperty) ≠
    fingerprintOf (checkProperty context (.portable changedBound)) := by
  native_decide

end Umpire.PropertyTests
