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

def reorderedProperty : Property := {
  portableProperty with
  clauses := portableProperty.clauses.reverse
}

def canonicalOf
    (check : Except PropertyError CheckedProperty) : Option String :=
  check.toOption.map canonicalPropertyJson

def fingerprintOf
    (check : Except PropertyError CheckedProperty) : Option BehaviorFingerprint :=
  check.toOption.map CheckedProperty.behaviorFingerprint

example : canonicalOf (Property.check context authoredProperty) =
      canonicalOf (Property.check reorderedContext (reorderedProperty)) ∧
    fingerprintOf (Property.check context authoredProperty) =
      fingerprintOf (Property.check reorderedContext (reorderedProperty)) := by
  native_decide

def changedSourceProperty : Property := {
  portableProperty with
  source := { source with line := source.line + 1 }
}

def changedDocumentationProperty : Property := {
  portableProperty with
  documentation := "Updated Property documentation."
}

example : [
    fingerprintOf (Property.check context (changedSourceProperty)),
    fingerprintOf (Property.check context (changedDocumentationProperty))
  ] = [
    fingerprintOf (Property.check context authoredProperty),
    fingerprintOf (Property.check context authoredProperty)
  ] := by
  native_decide

example : [
    canonicalOf (Property.check context (changedSourceProperty)),
    canonicalOf (Property.check context (changedDocumentationProperty))
  ].all fun changed => changed.isSome && changed != canonicalOf (Property.check context authoredProperty) := by
  native_decide

/- The legacy semantic identity remains an exact compatibility boundary. -/
#guard (fingerprintOf (Property.check context authoredProperty)).map BehaviorFingerprint.render ==
  some "sha256:d4f5e1474603bf339c82b2e93313854128a90c5032476e1b34de0551094857a3"

def changedCapabilityContext : PropertyCheckContext := {
  context with
  providers := context.providers.map fun capability =>
    if capability.id == cancellationCapability then
      { capability with behaviorVersion := "test-cancellation/v2" }
    else
      capability
}

example : fingerprintOf (Property.check context authoredProperty) ≠
    fingerprintOf (Property.check changedCapabilityContext authoredProperty) := by
  native_decide

def changedConstructor : Property := {
  portableProperty with
  clauses := portableProperty.clauses.map fun clause =>
    if clause.id == honoredDelivery.id then
      .neverWithin honoredDelivery.id
        (pattern .observation cancelRequested)
        (pattern .observation cancelDelivered)
        cancelBudget
    else
      clause
}

def changedReference : Property := {
  portableProperty with
  clauses := portableProperty.clauses.map fun clause =>
    if clause.id == cancelIsUnique.id then
      .stateInvariant cancelIsUnique.id
        (pattern .state cancellationPhase (.naturalAtMost 1))
    else
      clause
}

def changedBound : Property := {
  portableProperty with
  clauses := portableProperty.clauses.map fun clause =>
    if clause.id == honoredDelivery.id then
      .eventuallyWithin honoredDelivery.id
        (pattern .observation cancelRequested)
        (pattern .observation cancelDelivered)
        { value := 3, unit := .observationPositions }
    else
      clause
}

example : fingerprintOf (Property.check context authoredProperty) ≠
    fingerprintOf (Property.check context (changedConstructor)) := by
  native_decide

example : fingerprintOf (Property.check context authoredProperty) ≠
    fingerprintOf (Property.check context (changedReference)) := by
  native_decide

example : fingerprintOf (Property.check context authoredProperty) ≠
    fingerprintOf (Property.check context (changedBound)) := by
  native_decide

def unsupportedMajorProperty : Property := {
  portableProperty with
  version := 3
}

def unsupportedMajorWithInvalidBody : Property := {
  unsupportedMajorProperty with
  clauses := [cancelIsUnique, cancelIsUnique]
}

/- Unknown Property majors fail before an old reader can accept their clauses as version one. -/
#guard match Property.check context (unsupportedMajorProperty) with
  | .error error =>
      error.kind == .unsupportedPropertyVersion &&
      error.definitionId == portableProperty.id &&
      error.sourcePath == portableProperty.source.displayPath &&
      error.offendingValue == "supported versions are 1 and 2, found 3" &&
      error.relatedDefinitionIds == [portableProperty.id]
  | .ok _ => false

/- Version classification precedes an unknown major's body without changing supported-version
diagnostic order. -/
#guard match Property.check context (unsupportedMajorWithInvalidBody) with
  | .error error => error.kind == .unsupportedPropertyVersion
  | .ok _ => false

end Umpire.PropertyTests
