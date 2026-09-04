import Umpire.Property.Check
import Umpire.Property.Tests.Fixtures

/-! Malformed Property definitions and authoring-mode validation checks. -/

namespace Umpire.PropertyTests

open Umpire

private def characterizedErrorOf
    (result : Except PropertyError CheckedProperty) : Option (PropertyError × String) :=
  match result with
  | .ok _ => none
  | .error error => some (error, canonicalPropertyErrorJson error)

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
    characterizedErrorOf (checkProperty context (.portable mixedUnitProperty)) = some ({
      kind := .unitMismatch
      definitionId := id "test.property.mixed-unit"
      sourcePath := "Umpire/Property/Tests.lean"
      offendingValue :=
        "test.limit.cancel-budget: expected selected-actions, found observation-positions"
      relatedDefinitionIds := [cancelBudget.id]
    }, "{\"kind\":\"unit-mismatch\",\"definitionId\":\"test.property.mixed-unit\"," ++
      "\"sourcePath\":\"Umpire/Property/Tests.lean\",\"offendingValue\":" ++
      "\"test.limit.cancel-budget: expected selected-actions, found observation-positions\"," ++
      "\"relatedDefinitionIds\":[\"test.limit.cancel-budget\"]}") := by
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
    characterizedErrorOf (checkProperty context (.portable missingLogicalTimeProperty)) = some ({
      kind := .missingLogicalTimeSource
      definitionId := id "test.property.missing-logical-time"
      sourcePath := "Umpire/Property/Tests.lean"
      offendingValue := "logical-time"
      relatedDefinitionIds := []
    }, "{\"kind\":\"missing-logical-time-source\"," ++
      "\"definitionId\":\"test.property.missing-logical-time\"," ++
      "\"sourcePath\":\"Umpire/Property/Tests.lean\"," ++
      "\"offendingValue\":\"logical-time\",\"relatedDefinitionIds\":[]}") := by
  native_decide

example :
    characterizedErrorOf (checkProperty context
      (.opaque (id "test.property.expert-only") source)) = some ({
      kind := .opaqueDeclaration
      definitionId := id "test.property.expert-only"
      sourcePath := "Umpire/Property/Tests.lean"
      offendingValue := "test.property.expert-only"
      relatedDefinitionIds := [id "test.property.expert-only"]
    }, "{\"kind\":\"opaque-declaration\",\"definitionId\":\"test.property.expert-only\"," ++
      "\"sourcePath\":\"Umpire/Property/Tests.lean\"," ++
      "\"offendingValue\":\"test.property.expert-only\"," ++
      "\"relatedDefinitionIds\":[\"test.property.expert-only\"]}") := by
  native_decide

def unknownCapabilityProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.unknown-capability"
  requires := [id "test.capability.unknown"]
}

example :
    characterizedErrorOf (checkProperty context (.portable unknownCapabilityProperty)) = some ({
      kind := .unknownCapability
      definitionId := unknownCapabilityProperty.id
      sourcePath := "Umpire/Property/Tests.lean"
      offendingValue := "test.capability.unknown"
      relatedDefinitionIds := [id "test.capability.unknown"]
    }, "{\"kind\":\"unknown-capability\",\"definitionId\":" ++
      "\"test.property.unknown-capability\",\"sourcePath\":" ++
      "\"Umpire/Property/Tests.lean\",\"offendingValue\":\"test.capability.unknown\"," ++
      "\"relatedDefinitionIds\":[\"test.capability.unknown\"]}") := by
  native_decide

def wrongCapabilityKindProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.wrong-capability-kind"
  requires := [pendingCount]
}

example :
    characterizedErrorOf (checkProperty context (.portable wrongCapabilityKindProperty)) = some ({
      kind := .wrongReferenceKind
      definitionId := wrongCapabilityKindProperty.id
      sourcePath := "Umpire/Property/Tests.lean"
      offendingValue := "test.state.pending-count: expected capability, found state"
      relatedDefinitionIds := [pendingCount]
    }, "{\"kind\":\"wrong-reference-kind\",\"definitionId\":" ++
      "\"test.property.wrong-capability-kind\",\"sourcePath\":" ++
      "\"Umpire/Property/Tests.lean\",\"offendingValue\":" ++
      "\"test.state.pending-count: expected capability, found state\"," ++
      "\"relatedDefinitionIds\":[\"test.state.pending-count\"]}") := by
  native_decide

def missingCapabilityContext : PropertyCheckContext := {
  context with
  providers := context.providers.filter fun capability => capability.id != cancellationCapability
}

def missingCapabilityProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.missing-capability"
}

example :
    characterizedErrorOf
      (checkProperty missingCapabilityContext (.portable missingCapabilityProperty)) = some ({
        kind := .missingCapability
        definitionId := missingCapabilityProperty.id
        sourcePath := "Umpire/Property/Tests.lean"
        offendingValue := "test.capability.cancellation"
        relatedDefinitionIds := [cancellationCapability]
      }, "{\"kind\":\"missing-capability\",\"definitionId\":" ++
        "\"test.property.missing-capability\",\"sourcePath\":" ++
        "\"Umpire/Property/Tests.lean\",\"offendingValue\":" ++
        "\"test.capability.cancellation\",\"relatedDefinitionIds\":" ++
        "[\"test.capability.cancellation\"]}") := by
  native_decide

def unknownReferenceProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.unknown-reference"
  clauses := [
    .stateInvariant (id "test.property.unknown-reference.clause")
      (pattern .state (id "test.state.unknown"))
  ]
}

example :
    characterizedErrorOf (checkProperty context (.portable unknownReferenceProperty)) = some ({
      kind := .unknownReference
      definitionId := unknownReferenceProperty.id
      sourcePath := "Umpire/Property/Tests.lean"
      offendingValue := "test.state.unknown"
      relatedDefinitionIds := [id "test.state.unknown"]
    }, "{\"kind\":\"unknown-reference\",\"definitionId\":" ++
      "\"test.property.unknown-reference\",\"sourcePath\":" ++
      "\"Umpire/Property/Tests.lean\",\"offendingValue\":\"test.state.unknown\"," ++
      "\"relatedDefinitionIds\":[\"test.state.unknown\"]}") := by
  native_decide

def wrongReferenceKindProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.wrong-reference-kind"
  clauses := [
    .stateInvariant (id "test.property.wrong-reference-kind.clause")
      (pattern .state requestCancel)
  ]
}

example :
    characterizedErrorOf (checkProperty context (.portable wrongReferenceKindProperty)) = some ({
      kind := .wrongReferenceKind
      definitionId := wrongReferenceKindProperty.id
      sourcePath := "Umpire/Property/Tests.lean"
      offendingValue := "test.action.request-cancel: expected state, found action"
      relatedDefinitionIds := [requestCancel]
    }, "{\"kind\":\"wrong-reference-kind\",\"definitionId\":" ++
      "\"test.property.wrong-reference-kind\",\"sourcePath\":" ++
      "\"Umpire/Property/Tests.lean\",\"offendingValue\":" ++
      "\"test.action.request-cancel: expected state, found action\"," ++
      "\"relatedDefinitionIds\":[\"test.action.request-cancel\"]}") := by
  native_decide

def undeclaredReferenceProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.undeclared-reference"
  clauses := [
    .inputOutput (id "test.property.undeclared-reference.clause")
      (pattern .selectedAction requestCancel)
      (pattern .observation hiddenObservation)
  ]
}

example :
    characterizedErrorOf (checkProperty context (.portable undeclaredReferenceProperty)) = some ({
      kind := .undeclaredReference
      definitionId := undeclaredReferenceProperty.id
      sourcePath := "Umpire/Property/Tests.lean"
      offendingValue := "test.observation.hidden-record"
      relatedDefinitionIds := [hiddenObservation]
    }, "{\"kind\":\"undeclared-reference\",\"definitionId\":" ++
      "\"test.property.undeclared-reference\",\"sourcePath\":" ++
      "\"Umpire/Property/Tests.lean\",\"offendingValue\":" ++
      "\"test.observation.hidden-record\",\"relatedDefinitionIds\":" ++
      "[\"test.observation.hidden-record\"]}") := by
  native_decide

def unknownLimitProfileProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.unknown-limit-profile"
  clauses := [
    .eventuallyWithin (id "test.property.unknown-limit-profile.clause")
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      (.named (id "test.limit.unknown") .observationPositions)
  ]
}

example :
    characterizedErrorOf (checkProperty context (.portable unknownLimitProfileProperty)) = some ({
      kind := .unknownLimitProfile
      definitionId := unknownLimitProfileProperty.id
      sourcePath := "Umpire/Property/Tests.lean"
      offendingValue := "test.limit.unknown"
      relatedDefinitionIds := [id "test.limit.unknown"]
    }, "{\"kind\":\"unknown-limit-profile\",\"definitionId\":" ++
      "\"test.property.unknown-limit-profile\",\"sourcePath\":" ++
      "\"Umpire/Property/Tests.lean\",\"offendingValue\":\"test.limit.unknown\"," ++
      "\"relatedDefinitionIds\":[\"test.limit.unknown\"]}") := by
  native_decide

def invalidClauseProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.invalid-clause"
  clauses := [
    .stateInvariant (id "test.property.invalid-clause.clause")
      (pattern .observation cancelRequested)
  ]
}

example :
    characterizedErrorOf (checkProperty context (.portable invalidClauseProperty)) = some ({
      kind := .invalidClause
      definitionId := invalidClauseProperty.id
      sourcePath := "Umpire/Property/Tests.lean"
      offendingValue := "test.property.invalid-clause.clause: observation"
      relatedDefinitionIds := [id "test.property.invalid-clause.clause"]
    }, "{\"kind\":\"invalid-clause\",\"definitionId\":" ++
      "\"test.property.invalid-clause\",\"sourcePath\":" ++
      "\"Umpire/Property/Tests.lean\",\"offendingValue\":" ++
      "\"test.property.invalid-clause.clause: observation\",\"relatedDefinitionIds\":" ++
      "[\"test.property.invalid-clause.clause\"]}") := by
  native_decide

def duplicateLimitProfileContext : PropertyCheckContext := {
  context with limitProfiles := [cancelBudget, cancelBudget]
}

def duplicateLimitProfileProperty : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.duplicate-limit-profile"
}

example :
    characterizedErrorOf
      (checkProperty duplicateLimitProfileContext (.portable duplicateLimitProfileProperty)) =
      some ({
        kind := .duplicateDefinitionId
        definitionId := duplicateLimitProfileProperty.id
        sourcePath := "Umpire/Property/Tests.lean"
        offendingValue := "test.limit.cancel-budget"
        relatedDefinitionIds := [cancelBudget.id]
      }, "{\"kind\":\"duplicate-definition-id\",\"definitionId\":" ++
        "\"test.property.duplicate-limit-profile\",\"sourcePath\":" ++
        "\"Umpire/Property/Tests.lean\",\"offendingValue\":" ++
        "\"test.limit.cancel-budget\",\"relatedDefinitionIds\":" ++
        "[\"test.limit.cancel-budget\"]}") := by
  native_decide

end Umpire.PropertyTests
