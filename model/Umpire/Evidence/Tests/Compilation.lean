import Umpire.Evidence.Reading.Check
import Umpire.Evidence.Tests.Fixtures
import Umpire.Model.Tests.Fixtures

/-! Deterministic checked-plan identity and exact structural compilation failures. -/

namespace Umpire.EvidenceTests

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

def projectedProfileSpec : ObservationProfileSpec := {
  id := profileId
  source
  kinds := [{
    id := eventKind
    fields := [nameFieldSpec, secretFieldSpec, hashedFieldSpec, rejectedFieldSpec]
  }]
}

def projectedInitialRuleSpec : ObservationRuleSpec := {
  id := initialRule.id
  output := operationState
  outputKind := .state
  field := nameFieldSpec
  condition := initialRule.condition
}

def projectedMappingSpec : Evidence.ReadingSpec := {
  id := baseDeclaration.id
  source
  profile := profileId
  digestPolicies := [digestPolicy]
  bindings := [normalizedName]
  rules := [initialRule, contributionRule, digestRule]
  ordering := baseDeclaration.ordering
  closures := [{ kind := eventKind }]
  dispositions := [
    (nameFieldSpec, .retain),
    (secretFieldSpec, .redact),
    (hashedFieldSpec, .hash (some digestPolicyId)),
    (rejectedFieldSpec, .reject)
  ]
  evidenceBound := { value := 10, unit := .evidenceRecords }
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

/-- Profile, rule, and mapping specifications project exact existing declaration values. -/
example :
    projectedProfileSpec.declaration = evidenceProfile ∧
    projectedInitialRuleSpec.declaration = {
      initialRule with value := .portable nameFieldSpec.expression
    } ∧
    projectedMappingSpec.declaration = baseDeclaration := by
  exact ⟨rfl, rfl, rfl⟩

def checkedProjectedPlan : Evidence.CheckedReading :=
  projectedMappingSpec.checked context (by native_decide)

/-- Specification checking and checked extraction delegate to the existing checker exactly once. -/
example :
    projectedMappingSpec.check context = Evidence.checkReading context baseDeclaration ∧
    checkedProjectedPlan =
      (Evidence.checkReading context baseDeclaration).toOption.get (by native_decide) := by
  exact ⟨rfl, rfl⟩

#guard_msgs (error, substring := true) in
def projectedMappingWithoutValidityProof : Evidence.CheckedReading :=
  projectedMappingSpec.checked context

def checkedBasePlan : Evidence.CheckedReading :=
  Evidence.checkedReading context baseDeclaration (by native_decide)

/-- Checked Observation authoring returns the typed checker's complete canonical plan. -/
example : checkedBasePlan =
    (Evidence.checkReading context baseDeclaration).toOption.get (by native_decide) := by
  native_decide

#guard_msgs (error, substring := true) in
def observationWithoutValidityProof : Evidence.CheckedReading :=
  Evidence.checkedReading context baseDeclaration

def reorderedInitialRule : ObservationRule := {
  initialRule with
  condition := some (.portable (.and
    (.equals (.boolean true) (.boolean true))
    (.present (field nameFieldSpec))))
}

def reorderedDeclaration : Evidence.Reading := {
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
  let plan ← (Evidence.checkReading context baseDeclaration).toOption
  let binding ← plan.bindings.find? fun binding => binding.id == normalizedName.id
  match binding.expression with
  | .normalize .textTrimV1 (.field reference .text .retain) =>
      pure (reference == { kind := eventKind, field := nameField })
  | _ => pure false

/-- Evaluation can consume the checked expression tree without parsing its canonical identity. -/
example : checkedNormalizedNameIsTyped = some true := by
  native_decide

def connectedContext : Option Evidence.ReadingContext :=
  ((checkModel (DraftModel.make Umpire.ModelTests.testTarget) |>.mapError LocatedError.error)).toOption.map fun target =>
    Evidence.ReadingContext.ofTarget target [evidenceProfile]

def reconciledMappingSpec : Evidence.ReadingSpec := {
  baseSpec with
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

def reconciledMapping : Evidence.Reading :=
  reconciledMappingSpec.declaration

def reconciledMeaningDigest : Option String := do
  let checkContext ← connectedContext
  let plan ← (Evidence.checkReading checkContext reconciledMapping).toOption
  let rule ← plan.rules.find? fun rule => rule.id == id "test.rule.reconciled"
  pure rule.meaning.behaviorVersion

/-- Connected target meanings compile under the connector's reconciled semantic identity. -/
example : reconciledMeaningDigest = some "test-shared-connector/reconciled-v1" := by
  native_decide

def providerResolutionFailures :
    Option DefinitionErrorKind × Option Evidence.ReadingErrorKind :=
  (Umpire.ModelTests.errorOf ((checkModel (DraftModel.make Umpire.ModelTests.conflictingTarget) |>.mapError LocatedError.error))
      |>.map DefinitionError.kind,
    errorKindOf (Evidence.checkReading { context with meanings := [] } reconciledMapping))

/-- Conflicting providers fail before Observation construction; unresolved meaning stays fail-closed. -/
example : providerResolutionFailures =
    (some .conflictingProviders, some .unknownSemanticDeclaration) := by
  native_decide

/-- Every consumed field has one checked disposition in the canonical plan. -/
example : (Evidence.checkReading context baseDeclaration).toOption.map
    (fun plan => plan.dispositions.length) = some 4 := by
  native_decide

def withSingleRuleExpression
    (expression : ObservationExpressionAuthoring) : Evidence.Reading := {
  baseDeclaration with
  rules := [{ initialRule with value := expression, condition := none }]
  ordering := []
}

def emptyProfileContext : Evidence.ReadingContext := {
  context with profiles := [{ evidenceProfile with id := id "" }]
}

def invalidProfileContext : Evidence.ReadingContext := {
  context with profiles := [{ evidenceProfile with id := id "profile" }]
}

def duplicateProfileContext : Evidence.ReadingContext := {
  context with profiles := [evidenceProfile, evidenceProfile]
}

def emptyFieldContext : Evidence.ReadingContext := {
  context with profiles := [{
    evidenceProfile with kinds := [{
      id := eventKind
      fields := [{ id := id "", valueType := .text }]
    }]
  }]
}

def invalidFieldContext : Evidence.ReadingContext := {
  context with profiles := [{
    evidenceProfile with kinds := [{
      id := eventKind
      fields := [{ id := id "field", valueType := .text }]
    }]
  }]
}

def duplicateFieldContext : Evidence.ReadingContext := {
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
    (declaration : EvidenceFieldDeclaration) : Evidence.ReadingContext := {
  context with profiles := [{
    evidenceProfile with kinds := evidenceProfile.kinds.map fun kind => {
      kind with fields := kind.fields.map fun field =>
        if field.id == nameField then declaration else field
    }
  }]
}

def contextWithProjectedFields
    (fields : List EvidenceFieldDeclaration) : Evidence.ReadingContext := {
  context with profiles := [{
    evidenceProfile with kinds := [{ id := eventKind, fields }]
  }]
}

def projectedFieldFailures : List (Option Evidence.ReadingErrorKind) := [
  errorKindOf (Evidence.checkReading
    (contextWithProjectedNameDeclaration { projectedNameField with field := id "" }.declaration)
    baseDeclaration),
  errorKindOf (Evidence.checkReading
    (contextWithProjectedNameDeclaration { projectedNameField with field := id "field" }.declaration)
    baseDeclaration),
  errorKindOf (Evidence.checkReading
    (contextWithProjectedFields [projectedNameField.declaration, projectedNameField.declaration])
    baseDeclaration),
  errorKindOf (Evidence.checkReading context (withSingleRuleExpression (.portable
    { projectedNameField with kind := id "test.kind.unknown" }.expression))),
  errorKindOf (Evidence.checkReading context (withSingleRuleExpression (.portable
    { projectedNameField with field := id "test.field.unknown" }.expression))),
  errorKindOf (Evidence.checkReading
    (contextWithProjectedNameDeclaration { projectedNameField with valueType := .boolean }.declaration)
    baseDeclaration),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with
    rules := [{ initialRule with value := .portable projectedNameField.expression }]
    ordering := []
    dispositions := baseDeclaration.dispositions.filter fun disposition =>
      disposition.field != projectedNameField.reference
  }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with dispositions := baseDeclaration.dispositions ++
      [projectedNameField.disposition .retain]
  }),
  errorKindOf (Evidence.checkReading context {
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

def structuralFailures : List (Option Evidence.ReadingErrorKind) := [
  errorKindOf (Evidence.checkReading emptyProfileContext { baseDeclaration with profile := id "" }),
  errorKindOf (Evidence.checkReading invalidProfileContext { baseDeclaration with profile := id "profile" }),
  errorKindOf (Evidence.checkReading duplicateProfileContext baseDeclaration),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with rules := [{ initialRule with id := id "" }], ordering := [] }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with rules := [{ initialRule with id := id "rule" }], ordering := [] }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with rules := [initialRule, initialRule], ordering := [] }),
  errorKindOf (Evidence.checkReading emptyFieldContext baseDeclaration),
  errorKindOf (Evidence.checkReading invalidFieldContext baseDeclaration),
  errorKindOf (Evidence.checkReading duplicateFieldContext baseDeclaration),
  errorKindOf (Evidence.checkReading context { baseDeclaration with profile := id "test.profile.unknown" }),
  errorKindOf (Evidence.checkReading context (withSingleRuleExpression
    (.portable (.field { kind := id "test.kind.unknown", field := nameField })))),
  errorKindOf (Evidence.checkReading context (withSingleRuleExpression
    (.portable (.field { kind := eventKind, field := id "test.field.unknown" })))),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with bindings := [{ normalizedName with
      expression := .portable
        (.normalize { name := "text.unknown", version := 1 } (field nameFieldSpec)) }]
  }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with bindings := [{ normalizedName with
      expression := .portable
        (.normalize { name := "text.trim", version := 2 } (field nameFieldSpec)) }]
  }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with bindings := [{ normalizedName with
      expression := .portable
        (.normalize { name := "text.trim", version := 1 } (.boolean true)) }]
  }),
  errorKindOf (Evidence.checkReading context (withSingleRuleExpression (.callback "forbidden"))),
  errorKindOf (Evidence.checkReading context (withSingleRuleExpression (.recursive initialRule.id))),
  errorKindOf (Evidence.checkReading context (withSingleRuleExpression (field secretFieldSpec))),
  errorKindOf (Evidence.checkReading context (withSingleRuleExpression (field hashedFieldSpec))),
  errorKindOf (Evidence.checkReading context (withSingleRuleExpression (field rejectedFieldSpec))),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with
    rules := [{ initialRule with output := id "test.state.unknown" }]
    ordering := []
  }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with
    rules := [{
      initialRule with
      output := unauthorizedObservation
      outputKind := .fact
    }]
    ordering := []
  }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with
    rules := [{ initialRule with outputKind := .action }]
    ordering := []
  }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with
    dispositions := baseDeclaration.dispositions.filter fun disposition =>
      disposition.field.field != nameField
  }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with
    dispositions := baseDeclaration.dispositions ++
      [{ field := { kind := eventKind, field := nameField }, disposition := .retain }]
  }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with
    rules := [initialRule, {
      contributionRule with
      output := operationState
      outputKind := .state
    }]
    ordering := []
  }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with bindings := [{ normalizedName with valueType := .boolean }]
  }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with ordering := [
      { before := initialRule.id, after := contributionRule.id },
      { before := contributionRule.id, after := initialRule.id }
    ]
  }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with ordering := [
      { before := initialRule.id, after := contributionRule.id },
      { before := contributionRule.id, after := digestRule.id },
      { before := digestRule.id, after := initialRule.id }
    ]
  }),
  errorKindOf (Evidence.checkReading context { baseDeclaration with closures := [] }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with closures := [{ kind := eventKind }, { kind := eventKind }]
  }),
  errorKindOf (Evidence.checkReading context {
    baseDeclaration with evidenceBound := { value := 0, unit := .evidenceRecords }
  }),
  errorKindOf (Evidence.checkReading context {
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
  outputKind := .fact
  value := .portable (.text ruleId.value)
}

def divergentCycleContext : Evidence.ReadingContext := {
  context with
  definitions := context.definitions ++ [cycleA, cycleB, cycleC, cycleD].map fun ruleId =>
    metadata (cycleOutput ruleId).value .fact
  meanings := context.meanings ++ [cycleA, cycleB, cycleC, cycleD].map fun ruleId => {
    definitionId := cycleOutput ruleId
    kind := .fact
    behaviorVersion := (cycleOutput ruleId).value ++ "/meaning-v1"
  }
}

def divergentCycleDeclaration : Evidence.Reading := {
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

def mixedGraphAndBoundFaultDeclaration : Evidence.Reading := {
  divergentCycleDeclaration with
  evidenceBound := { value := 0, unit := .evidenceRecords }
}

def multipleGraphFaultDeclaration : Evidence.Reading := {
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
    (result : Except Evidence.ReadingError Evidence.CheckedReading) : Option String :=
  match result with
  | .ok _ => none
  | .error failure => some (Evidence.canonicalReadingErrorJson failure)

example : (
    compileErrorJson (Evidence.checkReading divergentCycleContext mixedGraphAndBoundFaultDeclaration),
    compileErrorJson (Evidence.checkReading context multipleGraphFaultDeclaration),
    compileErrorJson (Evidence.checkReading divergentCycleContext divergentCycleDeclaration)
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

/-! Structural cost inventory for the inert helper layer:

`ObservationKindSpec.declaration` calls `List.map` once over its explicit field collection.
`ObservationProfileSpec.declaration` calls `List.map` once over its explicit kind collection and
then the kind helper once per kind. `ObservationRuleSpec.declaration` calls
`ObservationFieldSpec.expression` and performs record assembly without a collection traversal.
`Evidence.ReadingSpec.declaration` calls `List.map` once over explicit disposition choices and
otherwise performs record assembly. `Evidence.ReadingSpec.check` and `.checked` each delegate to
one `Evidence.checkReading` call; neither normalizes, rescans, nor duplicates checker work. The
independent 1×/10× specimens below therefore add exactly one copy of that wrapper work per input;
the unchanged checker complexity is outside this construction inventory. -/

def oneIndependentObservationConstruction :
    List (EvidenceProfileDeclaration × ObservationRule × Evidence.Reading) := [
  (projectedProfileSpec.declaration, projectedInitialRuleSpec.declaration,
    projectedMappingSpec.declaration)
]

def tenIndependentObservationConstructions :
    List (EvidenceProfileDeclaration × ObservationRule × Evidence.Reading) :=
  (List.range 10).map fun index =>
    let suffix := toString index
    let profile := {
      projectedProfileSpec with id := id ("test.evidence.profile.scale-" ++ suffix)
    }
    let rule := {
      projectedInitialRuleSpec with id := id ("test.rule.scale-" ++ suffix)
    }
    let mapping := {
      projectedMappingSpec with id := id ("test.mapping.scale-" ++ suffix)
    }
    (profile.declaration, rule.declaration, mapping.declaration)

example : oneIndependentObservationConstruction.length = 1 ∧
    tenIndependentObservationConstructions.length = 10 := by
  native_decide

#print axioms ObservationKindSpec.declaration
#print axioms ObservationProfileSpec.declaration
#print axioms ObservationRuleSpec.declaration
#print axioms Evidence.ReadingSpec.declaration
#print axioms Evidence.ReadingSpec.check
#print axioms Evidence.checkedReading
#print axioms Evidence.ReadingSpec.checked

end Umpire.EvidenceTests
