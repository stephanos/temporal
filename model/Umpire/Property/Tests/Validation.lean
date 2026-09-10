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

def candidateEvaluationProperty (limit : Limit) : Property := {
  portableProperty with
  id := id "test.property.candidate-evaluations"
  clauses := [
    .eventuallyWithin (id "test.property.candidate-evaluations.clause")
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      limit
  ]
}

/-- Query's candidate-evaluation Limit is not a Property position unit. -/
example :
    errorKindOf (Property.check context
      (candidateEvaluationProperty { value := 2, unit := .candidateEvaluations })) =
      some .unitMismatch := by
  native_decide

/-! Exploration's ExperimentSpec Limit is not a Property position unit. -/
example : errorKindOf (Property.check context (
    candidateEvaluationProperty { value := 2, unit := .experimentSpecs })) =
    some .unitMismatch := by
  native_decide

def missingLogicalTimeProperty : Property := {
  portableProperty with
  id := id "test.property.missing-logical-time"
  clauses := [
    .eventuallyWithin (id "test.property.missing-logical-time.clause")
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      { value := 1, unit := .logicalTime }
  ]
}

example :
    characterizedErrorOf (Property.check context (missingLogicalTimeProperty)) = some ({
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

/-- The opaque-declaration diagnostic outlives the deleted opaque authoring wrapper: it stays a
plain error a producer may report, with its canonical spelling unchanged. -/
example : PropertyErrorKind.name .opaqueDeclaration = "opaque-declaration" := by
  native_decide

def unknownCapabilityProperty : Property := {
  portableProperty with
  id := id "test.property.unknown-capability"
  requires := [id "test.capability.unknown"]
}

example :
    characterizedErrorOf (Property.check context (unknownCapabilityProperty)) = some ({
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

def wrongCapabilityKindProperty : Property := {
  portableProperty with
  id := id "test.property.wrong-capability-kind"
  requires := [pendingCount]
}

example :
    characterizedErrorOf (Property.check context (wrongCapabilityKindProperty)) = some ({
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

def missingCapabilityProperty : Property := {
  portableProperty with
  id := id "test.property.missing-capability"
}

example :
    characterizedErrorOf
      (Property.check missingCapabilityContext (missingCapabilityProperty)) = some ({
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

def unknownReferenceProperty : Property := {
  portableProperty with
  id := id "test.property.unknown-reference"
  clauses := [
    .stateInvariant (id "test.property.unknown-reference.clause")
      (pattern .state (id "test.state.unknown"))
  ]
}

example :
    characterizedErrorOf (Property.check context (unknownReferenceProperty)) = some ({
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

def wrongReferenceKindProperty : Property := {
  portableProperty with
  id := id "test.property.wrong-reference-kind"
  clauses := [
    .stateInvariant (id "test.property.wrong-reference-kind.clause")
      (pattern .state requestCancel)
  ]
}

example :
    characterizedErrorOf (Property.check context (wrongReferenceKindProperty)) = some ({
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

def undeclaredReferenceProperty : Property := {
  portableProperty with
  id := id "test.property.undeclared-reference"
  clauses := [
    .inputOutput (id "test.property.undeclared-reference.clause")
      (pattern .selectedAction requestCancel)
      (pattern .observation hiddenObservation)
  ]
}

example :
    characterizedErrorOf (Property.check context (undeclaredReferenceProperty)) = some ({
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

def invalidClauseProperty : Property := {
  portableProperty with
  id := id "test.property.invalid-clause"
  clauses := [
    .stateInvariant (id "test.property.invalid-clause.clause")
      (pattern .observation cancelRequested)
  ]
}

example :
    characterizedErrorOf (Property.check context (invalidClauseProperty)) = some ({
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

end Umpire.PropertyTests
