import Umpire.Observation.Tests.Fixtures

/-! Deterministic checked-plan identity and exact structural compilation failures. -/

namespace Umpire.ObservationTests

open Umpire

def reorderedInitialRule : ObservationRule := {
  initialRule with
  condition := some (.portable (.and
    (.equals (.boolean true) (.boolean true))
    (.present (field nameField))))
}

def reorderedDeclaration : ObservationMappingDeclaration := {
  baseDeclaration with
  digestPolicies := baseDeclaration.digestPolicies.reverse
  bindings := baseDeclaration.bindings.reverse
  rules := [digestRule, contributionRule, reorderedInitialRule]
  ordering := baseDeclaration.ordering.reverse
  closures := baseDeclaration.closures.reverse
  dispositions := baseDeclaration.dispositions.reverse
}

/-- Reordering declarations and equivalent commutative expressions preserves checked identity. -/
example : planIdentityOf context baseDeclaration = planIdentityOf context reorderedDeclaration := by
  native_decide

/-- The positive evidence-record bound participates in semantic checked-plan identity. -/
example : planIdentityOf context baseDeclaration != planIdentityOf context {
    baseDeclaration with evidenceBound := { value := 11, unit := .evidenceRecords }
  } := by
  native_decide

/-- Every consumed field has one checked disposition in the canonical plan. -/
example : (checkObservation context baseDeclaration).toOption.map
    (fun plan => plan.dispositions.length) = some 4 := by
  native_decide

def withSingleRuleExpression
    (expression : ObservationExpressionAuthoring) : ObservationMappingDeclaration := {
  baseDeclaration with
  rules := [{ initialRule with value := expression, condition := none }]
  ordering := []
}

def emptyProfileContext : ObservationCheckContext := {
  context with profiles := [{ evidenceProfile with id := id "" }]
}

def invalidProfileContext : ObservationCheckContext := {
  context with profiles := [{ evidenceProfile with id := id "profile" }]
}

def duplicateProfileContext : ObservationCheckContext := {
  context with profiles := [evidenceProfile, evidenceProfile]
}

def emptyFieldContext : ObservationCheckContext := {
  context with profiles := [{
    evidenceProfile with kinds := [{
      id := eventKind
      fields := [{ id := id "", valueType := .text }]
    }]
  }]
}

def invalidFieldContext : ObservationCheckContext := {
  context with profiles := [{
    evidenceProfile with kinds := [{
      id := eventKind
      fields := [{ id := id "field", valueType := .text }]
    }]
  }]
}

def duplicateFieldContext : ObservationCheckContext := {
  context with profiles := [{
    evidenceProfile with kinds := [{
      id := eventKind
      fields := [
        { id := nameField, valueType := .text },
        { id := nameField, valueType := .text }
      ]
    }]
  }]
}

def structuralFailures : List (Option ObservationErrorKind) := [
  errorKindOf (checkObservation emptyProfileContext { baseDeclaration with profile := id "" }),
  errorKindOf (checkObservation invalidProfileContext { baseDeclaration with profile := id "profile" }),
  errorKindOf (checkObservation duplicateProfileContext baseDeclaration),
  errorKindOf (checkObservation context {
    baseDeclaration with rules := [{ initialRule with id := id "" }], ordering := [] }),
  errorKindOf (checkObservation context {
    baseDeclaration with rules := [{ initialRule with id := id "rule" }], ordering := [] }),
  errorKindOf (checkObservation context {
    baseDeclaration with rules := [initialRule, initialRule], ordering := [] }),
  errorKindOf (checkObservation emptyFieldContext baseDeclaration),
  errorKindOf (checkObservation invalidFieldContext baseDeclaration),
  errorKindOf (checkObservation duplicateFieldContext baseDeclaration),
  errorKindOf (checkObservation context { baseDeclaration with profile := id "test.profile.unknown" }),
  errorKindOf (checkObservation context (withSingleRuleExpression
    (.portable (.field { kind := id "test.kind.unknown", field := nameField })))),
  errorKindOf (checkObservation context (withSingleRuleExpression
    (.portable (.field { kind := eventKind, field := id "test.field.unknown" })))),
  errorKindOf (checkObservation context {
    baseDeclaration with bindings := [{ normalizedName with
      expression := .portable
        (.normalize { name := "text.unknown", version := 1 } (field nameField)) }]
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with bindings := [{ normalizedName with
      expression := .portable
        (.normalize { name := "text.trim", version := 2 } (field nameField)) }]
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with bindings := [{ normalizedName with
      expression := .portable
        (.normalize { name := "text.trim", version := 1 } (.boolean true)) }]
  }),
  errorKindOf (checkObservation context (withSingleRuleExpression (.callback "forbidden"))),
  errorKindOf (checkObservation context (withSingleRuleExpression (.recursive initialRule.id))),
  errorKindOf (checkObservation context (withSingleRuleExpression (field secretField))),
  errorKindOf (checkObservation context (withSingleRuleExpression (field hashedField))),
  errorKindOf (checkObservation context (withSingleRuleExpression (field rejectedField))),
  errorKindOf (checkObservation context {
    baseDeclaration with
    rules := [{ initialRule with output := id "test.state.unknown" }]
    ordering := []
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with
    rules := [{
      initialRule with
      output := unauthorizedObservation
      outputKind := .observation
    }]
    ordering := []
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with
    rules := [{ initialRule with outputKind := .action }]
    ordering := []
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with
    dispositions := baseDeclaration.dispositions.filter fun disposition =>
      disposition.field.field != nameField
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with
    dispositions := baseDeclaration.dispositions ++
      [{ field := { kind := eventKind, field := nameField }, disposition := .retain }]
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with
    rules := [initialRule, {
      contributionRule with
      output := operationState
      outputKind := .state
    }]
    ordering := []
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with bindings := [{ normalizedName with valueType := .boolean }]
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with ordering := [
      { before := initialRule.id, after := contributionRule.id },
      { before := contributionRule.id, after := initialRule.id }
    ]
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with ordering := [
      { before := initialRule.id, after := contributionRule.id },
      { before := contributionRule.id, after := digestRule.id },
      { before := digestRule.id, after := initialRule.id }
    ]
  }),
  errorKindOf (checkObservation context { baseDeclaration with closures := [] }),
  errorKindOf (checkObservation context {
    baseDeclaration with closures := [{ kind := eventKind }, { kind := eventKind }]
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with evidenceBound := { value := 10, unit := .semanticTransitions }
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with evidenceBound := { value := 0, unit := .evidenceRecords }
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with digestPolicies := []
  })
]

/-- Each R1 structural conflict reports its precise typed compile-error category. -/
example : structuralFailures = [
  some .emptyIdentity,
  some .invalidIdentity,
  some .duplicateIdentity,
  some .emptyIdentity,
  some .invalidIdentity,
  some .duplicateIdentity,
  some .emptyIdentity,
  some .invalidIdentity,
  some .duplicateIdentity,
  some .unknownEvidenceProfile,
  some .unknownEvidenceKind,
  some .unknownEvidenceField,
  some .unknownOperator,
  some .unknownOperatorVersion,
  some .typeMismatch,
  some .callbackExpression,
  some .recursiveExpression,
  some .unauthorizedClearValueFlow,
  some .unauthorizedClearValueFlow,
  some .rejectedInputRead,
  some .unknownSemanticDeclaration,
  some .unauthorizedSemanticDeclaration,
  some .wrongOutputKind,
  some .missingDisposition,
  some .duplicateDisposition,
  some .overlappingOutputs,
  some .incompatibleBinding,
  some .contradictoryOrdering,
  some .cyclicOrdering,
  some .missingClosure,
  some .duplicateClosure,
  some .invalidBoundUnit,
  some .invalidBoundValue,
  some .missingDigestPolicy
] := by
  native_decide

end Umpire.ObservationTests
