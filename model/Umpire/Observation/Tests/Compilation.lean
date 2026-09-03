import Umpire.Observation.Tests.Fixtures
import Umpire.Target.Tests.Fixtures

/-! Deterministic checked-plan identity and exact structural compilation failures. -/

namespace Umpire.ObservationTests

open Umpire

def projectedNameField : ObservationFieldSpec := {
  kind := eventKind
  field := nameField
  valueType := .text
}

def projectedHashedField : ObservationFieldSpec := {
  kind := eventKind
  field := hashedField
  valueType := .text
}

/-- Field specifications reproduce the existing inert authoring records exactly. -/
example :
    (projectedNameField.declaration,
      projectedNameField.reference,
      projectedNameField.expression,
      projectedNameField.disposition .retain) =
    ({ id := nameField, valueType := .text },
      { kind := eventKind, field := nameField },
      .field { kind := eventKind, field := nameField },
      { field := { kind := eventKind, field := nameField }, disposition := .retain }) := by
  rfl

def checkedBasePlan : CheckedObservationPlan :=
  checkedObservation context baseDeclaration (by native_decide)

/-- Checked Observation authoring returns the typed checker's complete canonical plan. -/
example : checkedBasePlan =
    (checkObservation context baseDeclaration).toOption.get (by native_decide) := by
  native_decide

#guard_msgs (error, substring := true) in
def observationWithoutValidityProof : CheckedObservationPlan :=
  checkedObservation context baseDeclaration

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

/-- Reordering definitions and equivalent commutative expressions preserves checked identity. -/
example : planIdentityOf context baseDeclaration = planIdentityOf context reorderedDeclaration := by
  native_decide

/-- The positive evidence-record bound participates in semantic checked-plan identity. -/
example : planIdentityOf context baseDeclaration != planIdentityOf context {
    baseDeclaration with evidenceBound := { value := 11, unit := .evidenceRecords }
  } := by
  native_decide

def checkedNormalizedNameIsTyped : Option Bool := do
  let plan ← (checkObservation context baseDeclaration).toOption
  let binding ← plan.bindings.find? fun binding => binding.id == normalizedName.id
  match binding.expression with
  | .normalize .textTrimV1 (.field reference .text .retain) =>
      pure (reference == { kind := eventKind, field := nameField })
  | _ => pure false

/-- Evaluation can consume the checked expression tree without parsing its canonical identity. -/
example : checkedNormalizedNameIsTyped = some true := by
  native_decide

def connectedContext : Option ObservationCheckContext :=
  (composeTarget Umpire.TargetTests.testTarget).toOption.map fun target =>
    ObservationCheckContext.ofTarget target [evidenceProfile]

def reconciledMapping : ObservationMappingDeclaration := {
  baseDeclaration with
  id := id "test.mapping.reconciled"
  digestPolicies := []
  bindings := []
  rules := [{
    id := id "test.rule.reconciled"
    output := id "test.relation.shared"
    outputKind := .relation
    value := .portable (.text "shared")
  }]
  ordering := []
  dispositions := []
}

def reconciledMeaningDigest : Option String := do
  let checkContext ← connectedContext
  let plan ← (checkObservation checkContext reconciledMapping).toOption
  let rule ← plan.rules.find? fun rule => rule.id == id "test.rule.reconciled"
  pure rule.meaning.canonicalBehavior

/-- Connected target meanings compile under the connector's reconciled semantic identity. -/
example : reconciledMeaningDigest = some "test-shared-connector/reconciled-v1" := by
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

def contextWithProjectedNameDeclaration
    (declaration : EvidenceFieldDeclaration) : ObservationCheckContext := {
  context with profiles := [{
    evidenceProfile with kinds := evidenceProfile.kinds.map fun kind => {
      kind with fields := kind.fields.map fun field =>
        if field.id == nameField then declaration else field
    }
  }]
}

def contextWithProjectedFields
    (fields : List EvidenceFieldDeclaration) : ObservationCheckContext := {
  context with profiles := [{
    evidenceProfile with kinds := [{ id := eventKind, fields }]
  }]
}

def projectedFieldFailures : List (Option ObservationErrorKind) := [
  errorKindOf (checkObservation
    (contextWithProjectedNameDeclaration { projectedNameField with field := id "" }.declaration)
    baseDeclaration),
  errorKindOf (checkObservation
    (contextWithProjectedNameDeclaration { projectedNameField with field := id "field" }.declaration)
    baseDeclaration),
  errorKindOf (checkObservation
    (contextWithProjectedFields [projectedNameField.declaration, projectedNameField.declaration])
    baseDeclaration),
  errorKindOf (checkObservation context (withSingleRuleExpression (.portable
    { projectedNameField with kind := id "test.kind.unknown" }.expression))),
  errorKindOf (checkObservation context (withSingleRuleExpression (.portable
    { projectedNameField with field := id "test.field.unknown" }.expression))),
  errorKindOf (checkObservation
    (contextWithProjectedNameDeclaration { projectedNameField with valueType := .boolean }.declaration)
    baseDeclaration),
  errorKindOf (checkObservation context {
    baseDeclaration with
    rules := [{ initialRule with value := .portable projectedNameField.expression }]
    ordering := []
    dispositions := baseDeclaration.dispositions.filter fun disposition =>
      disposition.field != projectedNameField.reference
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with dispositions := baseDeclaration.dispositions ++
      [projectedNameField.disposition .retain]
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with
    digestPolicies := []
    dispositions := baseDeclaration.dispositions.map fun declaration =>
      if declaration.field == projectedHashedField.reference then
        projectedHashedField.disposition (.hash (some digestPolicyId))
      else
        declaration
  })
]

/-- Field projections leave identity, type, disposition, and digest failures to the checker. -/
example : projectedFieldFailures = [
  some .emptyDefinitionId,
  some .invalidDefinitionId,
  some .duplicateDefinitionId,
  some .unknownEvidenceKind,
  some .unknownEvidenceField,
  some .typeMismatch,
  some .missingDisposition,
  some .duplicateDisposition,
  some .missingDigestPolicy
] := by
  native_decide

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
    baseDeclaration with evidenceBound := { value := 0, unit := .evidenceRecords }
  }),
  errorKindOf (checkObservation context {
    baseDeclaration with digestPolicies := []
  })
]

def cycleA : DefinitionId := id "test.rule.a-tail"
def cycleB : DefinitionId := id "test.rule.b-cycle"
def cycleC : DefinitionId := id "test.rule.c-cycle"
def cycleD : DefinitionId := id "test.rule.d-cycle"

def cycleOutput (ruleId : DefinitionId) : DefinitionId :=
  id (ruleId.value ++ ".output")

def cycleRule (ruleId : DefinitionId) : ObservationRule := {
  id := ruleId
  output := cycleOutput ruleId
  outputKind := .observation
  value := .portable (.text ruleId.value)
}

def divergentCycleContext : ObservationCheckContext := {
  context with
  definitions := context.definitions ++ [cycleA, cycleB, cycleC, cycleD].map fun ruleId =>
    metadata (cycleOutput ruleId).value .observation
  meanings := context.meanings ++ [cycleA, cycleB, cycleC, cycleD].map fun ruleId => {
    definitionId := cycleOutput ruleId
    kind := .observation
    canonicalBehavior := (cycleOutput ruleId).value ++ "/meaning-v1"
  }
}

def divergentCycleDeclaration : ObservationMappingDeclaration := {
  baseDeclaration with
  id := id "test.mapping.divergent-cycle"
  digestPolicies := []
  bindings := []
  rules := [cycleRule cycleD, cycleRule cycleB, cycleRule cycleA, cycleRule cycleC]
  ordering := [
    { before := cycleC, after := cycleA },
    { before := cycleB, after := cycleC },
    { before := cycleC, after := cycleD },
    { before := cycleD, after := cycleB }
  ]
  dispositions := []
}

def mixedGraphAndBoundFaultDeclaration : ObservationMappingDeclaration := {
  divergentCycleDeclaration with
  evidenceBound := { value := 0, unit := .evidenceRecords }
}

def multipleGraphFaultDeclaration : ObservationMappingDeclaration := {
  baseDeclaration with
  ordering := [
    { before := digestRule.id, after := digestRule.id },
    { before := initialRule.id, after := contributionRule.id },
    { before := initialRule.id, after := contributionRule.id },
    { before := contributionRule.id, after := digestRule.id },
    { before := digestRule.id, after := initialRule.id },
    { before := id "test.rule.unknown", after := initialRule.id }
  ]
}

def compileErrorJson
    (result : Except ObservationError CheckedObservationPlan) : Option String :=
  match result with
  | .ok _ => none
  | .error failure => some (canonicalObservationErrorJson failure)

example : (
    compileErrorJson (checkObservation divergentCycleContext mixedGraphAndBoundFaultDeclaration),
    compileErrorJson (checkObservation context multipleGraphFaultDeclaration),
    compileErrorJson (checkObservation divergentCycleContext divergentCycleDeclaration)
  ) = (
    some "{\"kind\":\"invalid-bound-value\",\"definitionId\":\"test.mapping.divergent-cycle\",\"sourcePath\":\"Umpire/Observation/Tests/Fixtures.lean\",\"offendingValue\":\"0\",\"relatedDefinitionIds\":[]}",
    some "{\"kind\":\"contradictory-ordering\",\"definitionId\":\"test.mapping.lifecycle\",\"sourcePath\":\"Umpire/Observation/Tests/Fixtures.lean\",\"offendingValue\":\"test.rule.initial-state->test.rule.contribution\",\"relatedDefinitionIds\":[\"test.rule.contribution\",\"test.rule.initial-state\"]}",
    some "{\"kind\":\"cyclic-ordering\",\"definitionId\":\"test.mapping.divergent-cycle\",\"sourcePath\":\"Umpire/Observation/Tests/Fixtures.lean\",\"offendingValue\":\"test.rule.b-cycle\",\"relatedDefinitionIds\":[\"test.rule.b-cycle\"]}"
  ) := by
  native_decide

/-- Each R1 structural conflict reports its precise typed compile-error category. -/
example : structuralFailures = [
  some .emptyDefinitionId,
  some .invalidDefinitionId,
  some .duplicateDefinitionId,
  some .emptyDefinitionId,
  some .invalidDefinitionId,
  some .duplicateDefinitionId,
  some .emptyDefinitionId,
  some .invalidDefinitionId,
  some .duplicateDefinitionId,
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
  some .invalidBoundValue,
  some .missingDigestPolicy
] := by
  native_decide

end Umpire.ObservationTests
