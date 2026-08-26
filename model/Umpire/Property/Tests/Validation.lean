import Umpire.Property.Tests.Fixtures

/-! Malformed Property declarations and authoring-mode validation checks. -/

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

def evidenceRecordBoundProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.evidence-record-bound"
  clauses := [
    .eventuallyWithin (id "test.property.evidence-record-bound.clause")
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      (.exact { value := 1, unit := .evidenceRecords })
  ]
}

/-- Evidence volume is an Observation boundary, never a semantic Property position. -/
example :
    errorKindOf (checkProperty context (.portable evidenceRecordBoundProperty)) =
      some .unitMismatch := by
  native_decide

example :
    errorKindOf (checkProperty context
      (.opaque (id "test.property.expert-only") source)) = some .opaqueDeclaration := by
  native_decide

end Umpire.PropertyTests
