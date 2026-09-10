import Umpire.Property.Tests.Fixtures

/-! Same-step guarded case admission, evaluation, identity, and agreement. -/

namespace Umpire.PropertyTests

open Umpire

private def guardAtom
    (field : PropertyPredicateField)
    (reference : DefinitionId)
    (literal : PropertyLiteral) : PropertyPredicate :=
  .atom { field, reference, constraint := .equals literal }

private def expectation
    (clauseId : String)
    (field : PropertyPredicateField)
    (reference : DefinitionId)
    (constraint : PropertyAtomConstraint) : PropertySameStepClause := {
  id := id clauseId
  source
  expectation := .atom { field, reference, constraint }
}

private def requestGuard : PropertyPredicate :=
  guardAtom .selectedAction requestCancel (.text "request")

private def tickGuard : PropertyPredicate :=
  guardAtom .selectedAction tick (.text "tick")

private def pendingOne : PropertyPredicate :=
  guardAtom .priorState pendingCount (.natural 1)

private def requestCase : PropertyCase := {
  id := id "test.property.case.request"
  source
  guard := requestGuard
  clauses := [
    expectation "test.property.case.request.state" .resultingState pendingCount
      (.equals (.natural 1)),
    expectation "test.property.case.request.outcome" .modelOutcome deliveredOutcome
      (.equals (.text "delivered"))
  ]
}

private def resolutionCase : PropertyCase := {
  id := id "test.property.case.resolution"
  source
  guard := tickGuard
  clauses := [
    expectation "test.property.case.resolution.fact" .expectationFact cancelDelivered .present
  ]
}

private def guardedGroup
    (cases : List PropertyCase := [requestCase, resolutionCase])
    (complete : Bool := true)
    (exclusive : Bool := true)
    (exception : Option PropertyException := none) : PropertyCaseGroup := {
  id := id "test.property.case-group.delivery"
  source
  guard := .any [requestGuard, tickGuard]
  exception
  cases
  complete
  exclusive
}

private def guardedDeclaration
    (group : PropertyCaseGroup := guardedGroup) : PropertyDeclaration := {
  portableProperty with
  id := id "test.property.guarded-delivery"
  version := 2
  clauses := [cancelIsUnique, .sameStepCases group]
}

private def requestTrace (before after : Nat) : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := value pendingCount (toString before)
  steps := [{
    selectedAction := value requestCancel "request"
    outcome := value deliveredOutcome "delivered"
    state := value pendingCount (toString after)
    facts := [value cancelRequested "request-1"]
  }]
}

private def evaluation?
    (declaration : PropertyDeclaration)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    Option PropertyEvaluation := do
  let property ← (checkProperty context (.portable declaration)).toOption
  (evaluatePropertyOnTrace property trace).toOption

#guard (evaluation? guardedDeclaration (requestTrace 0 1)).map
  PropertyEvaluation.satisfied == some true

example (property : CheckedProperty) (input : CheckedPropertyEvaluationInput property) :
    (evaluateProperty property input).satisfied = true ↔ property.denote input :=
  evaluateProperty_agrees property input

private def exceptedRequestCase : PropertyCase := {
  requestCase with
  exception := some {
    id := id "test.property.case.request.exception.pending"
    source
    condition := pendingOne
  }
}

private def replacementCase : PropertyCase := {
  id := id "test.property.case.request.replacement"
  source
  guard := .all [requestGuard, pendingOne]
  clauses := [
    expectation "test.property.case.request.replacement.state" .resultingState pendingCount
      (.equals (.natural 0))
  ]
}

private def missingReplacementDeclaration : PropertyDeclaration :=
  guardedDeclaration (guardedGroup [exceptedRequestCase])

private def replacementDeclaration : PropertyDeclaration :=
  guardedDeclaration (guardedGroup [exceptedRequestCase, replacementCase])

#guard [
    (evaluation? missingReplacementDeclaration (requestTrace 1 0)).map
      PropertyEvaluation.satisfied,
    (evaluation? replacementDeclaration (requestTrace 1 0)).map PropertyEvaluation.satisfied
  ] == [some false, some true]

private def parentExcludedDeclaration : PropertyDeclaration :=
  let group : PropertyCaseGroup := {
    guardedGroup with exception := some {
      id := id "test.property.case-group.delivery.exception.tick"
      source
      condition := tickGuard
    }
  }
  guardedDeclaration group

private def tickTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := value pendingCount "2"
  steps := [{
    selectedAction := value tick "tick"
    outcome := value deliveredOutcome "delivered"
    state := value pendingCount "2"
    facts := []
  }]
}

/- A parent exception excludes every child but cannot waive the independent invariant. -/
#guard (evaluation? parentExcludedDeclaration tickTrace).map
  PropertyEvaluation.satisfied == some false

private def groupOnlyDeclaration (group : PropertyCaseGroup) : PropertyDeclaration := {
  guardedDeclaration group with clauses := [.sameStepCases group]
}

private def falseParentDeclaration : PropertyDeclaration :=
  groupOnlyDeclaration { guardedGroup with guard := tickGuard }

private def incompleteAllowedDeclaration : PropertyDeclaration :=
  groupOnlyDeclaration (guardedGroup [exceptedRequestCase] (complete := false))

/- Parent and case exceptions are independently false, true exclusions are vacuous only for the
guarded group, and disabling completeness explicitly permits an unreplaced exception. -/
#guard [
    (evaluation? falseParentDeclaration (requestTrace 0 1)).map PropertyEvaluation.satisfied,
    (evaluation? (groupOnlyDeclaration { guardedGroup with exception := some {
      id := id "test.property.case-group.delivery.exception.tick"
      source
      condition := tickGuard
    } }) (requestTrace 0 1)).map PropertyEvaluation.satisfied,
    (evaluation? (groupOnlyDeclaration (guardedGroup [exceptedRequestCase]))
      (requestTrace 0 1)).map PropertyEvaluation.satisfied,
    (evaluation? (groupOnlyDeclaration { guardedGroup with exception := some {
      id := id "test.property.case-group.delivery.exception.tick"
      source
      condition := tickGuard
    } }) tickTrace).map PropertyEvaluation.satisfied,
    (evaluation? incompleteAllowedDeclaration (requestTrace 1 0)).map
      PropertyEvaluation.satisfied
  ] == [some true, some true, some true, some true, some true]

private def compatibleOverlapDeclaration : PropertyDeclaration :=
  let duplicate := { requestCase with id := id "test.property.case.request.also" }
  guardedDeclaration (guardedGroup [requestCase, duplicate] (exclusive := false))

private def exclusiveOverlapDeclaration : PropertyDeclaration :=
  let duplicate := { requestCase with id := id "test.property.case.request.also" }
  guardedDeclaration (guardedGroup [requestCase, duplicate])

#guard [
    (evaluation? compatibleOverlapDeclaration (requestTrace 0 1)).map
      PropertyEvaluation.satisfied,
    (evaluation? exclusiveOverlapDeclaration (requestTrace 0 1)).map
      PropertyEvaluation.satisfied
  ] == [some true, some false]

private def failingOverlapCase : PropertyCase := {
  id := id "test.property.case.request.failing-overlap"
  source
  guard := requestGuard
  clauses := [
    expectation "test.property.case.request.failing-overlap.state" .resultingState pendingCount
      (.equals (.natural 0))
  ]
}

/- Every overlapping case remains conjunctive under either source order; no case wins priority. -/
#guard [
    guardedGroup [requestCase, failingOverlapCase] (exclusive := false),
    guardedGroup [failingOverlapCase, requestCase] (exclusive := false)
  ].map (fun group =>
    (evaluation? (guardedDeclaration group) (requestTrace 0 1)).map
      PropertyEvaluation.satisfied) == [some false, some false]

/-- Missing fields remain typed errors before any false guard or negation can make them vacuous. -/
private def incompleteTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
      initialState := value pendingCount "0"
      steps := [{
        selectedAction := value (id "test.action.unknown") "request"
        outcome := value deliveredOutcome "delivered"
        state := value pendingCount "1"
        facts := []
      }]
    }

private def evaluationErrorKind?
    (declaration : PropertyDeclaration)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) : Option PropertyErrorKind := do
  let property ← (checkProperty context (.portable declaration)).toOption
  match evaluatePropertyOnTrace property trace with
  | .ok _ => none
  | .error error => some error.kind

#guard evaluationErrorKind? guardedDeclaration incompleteTrace == some .missingPredicateInput

/- Known absence is an ordinary false expectation rather than an unknown-input diagnostic. -/
#guard (evaluation? guardedDeclaration tickTrace).map PropertyEvaluation.satisfied == some false

private def guardedTemporalDeclaration : PropertyDeclaration := {
  guardedDeclaration with
  id := id "test.property.guarded-temporal"
  clauses := [
    .guardedEventuallyWithin (id "test.property.guarded-temporal.clause") source
      requestGuard none
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      (.exact { value := 1, unit := .semanticTransitions })
  ]
}

private def guardedQuiescentDeclaration : PropertyDeclaration := {
  guardedTemporalDeclaration with
  id := id "test.property.guarded-quiescent"
  clauses := [
    .guardedQuiescentWithin (id "test.property.guarded-quiescent.clause") source
      requestGuard none
      (pattern .observation cancelDelivered)
      (pattern .observation cancelRequested)
      (.exact { value := 1, unit := .semanticTransitions })
  ]
}

#guard [guardedTemporalDeclaration, guardedQuiescentDeclaration].map (fun declaration =>
    (checkProperty context (.portable declaration)).toOption.map fun property =>
      property.clauses.map ResolvedPropertyClause.id) == [
    some [id "test.property.guarded-temporal.clause"],
    some [id "test.property.guarded-quiescent.clause"]
  ]

private def reorderedDeclaration : PropertyDeclaration :=
  guardedDeclaration (guardedGroup [resolutionCase, requestCase])

private def reorderedClausesDeclaration : PropertyDeclaration := {
  guardedDeclaration with clauses := guardedDeclaration.clauses.reverse
}

private def changedGuardDeclaration : PropertyDeclaration :=
  guardedDeclaration { guardedGroup with guard := requestGuard }

private def changedCaseGuardDeclaration : PropertyDeclaration :=
  guardedDeclaration (guardedGroup [{ requestCase with guard := pendingOne }, resolutionCase])

private def changedExceptionDeclaration : PropertyDeclaration :=
  guardedDeclaration (guardedGroup (exception := some {
    id := id "test.property.case-group.delivery.exception"
    source
    condition := pendingOne
  }))

private def changedExpectationDeclaration : PropertyDeclaration :=
  let changedRequest := { requestCase with clauses := [
    expectation "test.property.case.request.state" .resultingState pendingCount
      (.equals (.natural 2)),
    expectation "test.property.case.request.outcome" .modelOutcome deliveredOutcome
      (.equals (.text "delivered"))
  ] }
  guardedDeclaration (guardedGroup [changedRequest, resolutionCase])

private def changedCompleteDeclaration : PropertyDeclaration :=
  guardedDeclaration (guardedGroup (complete := false))

private def changedExclusiveDeclaration : PropertyDeclaration :=
  guardedDeclaration (guardedGroup (exclusive := false))

private def changedGuardedSourceDeclaration : PropertyDeclaration := {
  guardedDeclaration with source := { source with line := source.line + 1 }
}

private def changedGuardedDocumentationDeclaration : PropertyDeclaration := {
  guardedDeclaration with documentation := "Updated guarded Property documentation."
}

private def changedNestedSourceDeclaration : PropertyDeclaration :=
  let nestedSource := { source with line := source.line + 1 }
  let changedRequest := {
    requestCase with
    source := nestedSource
    clauses := requestCase.clauses.map fun clause => { clause with source := nestedSource }
  }
  guardedDeclaration {
    guardedGroup [changedRequest, resolutionCase] with source := nestedSource
  }

private def fingerprintOf (declaration : PropertyDeclaration) : Option BehaviorFingerprint :=
  (checkProperty context (.portable declaration)).toOption.map CheckedProperty.behaviorFingerprint

#guard [reorderedDeclaration, reorderedClausesDeclaration].all fun declaration =>
  fingerprintOf guardedDeclaration == fingerprintOf declaration
#guard fingerprintOf guardedDeclaration != fingerprintOf changedGuardDeclaration
#guard [
    changedCaseGuardDeclaration,
    changedExceptionDeclaration,
    changedExpectationDeclaration,
    changedCompleteDeclaration,
    changedExclusiveDeclaration
  ].all fun declaration => fingerprintOf guardedDeclaration != fingerprintOf declaration
#guard [
    changedGuardedSourceDeclaration,
    changedGuardedDocumentationDeclaration,
    changedNestedSourceDeclaration
  ].all fun declaration =>
  fingerprintOf guardedDeclaration == fingerprintOf declaration
#guard (checkProperty context (.portable guardedDeclaration)).toOption.map
    (fun property => !property.canonicalMetadata.contains "temporalClauses") == some true
#guard (fingerprintOf guardedDeclaration).map BehaviorFingerprint.render ==
  some "sha256:2bdf8bbc4a76122f44ba9b5bb6254bcc6924b99830871bbeead8e7c70a92271c"

private def checkError?
    (declaration : PropertyDeclaration) : Option PropertyError :=
  match checkProperty context (.portable declaration) with
  | .ok _ => none
  | .error error => some error

private def withGroup (group : PropertyCaseGroup) : PropertyDeclaration :=
  guardedDeclaration group

private def malformedSource : SourceLocation :=
  {
    path := "Umpire/Property/Tests/MalformedGuardedCases.lean"
    line := 41
    column := 7
    provenance := "generated-test"
  }

private def duplicateSource : SourceLocation :=
  { malformedSource with line := 52, column := 9 }

private def referenceSource : SourceLocation :=
  { malformedSource with line := 63, column := 11 }

private def duplicateCases : PropertyDeclaration :=
  withGroup (guardedGroup [requestCase, requestCase])

private def malformedParent : PropertyDeclaration :=
  withGroup { guardedGroup with id := DefinitionId.of "" }

private def malformedCase : PropertyDeclaration :=
  withGroup (guardedGroup [{ requestCase with id := DefinitionId.of "Bad Case" }])

private def malformedClause : PropertyDeclaration :=
  withGroup (guardedGroup [{ requestCase with clauses := [
    expectation "" .resultingState pendingCount (.equals (.natural 1))
  ] }])

private def duplicateException : PropertyDeclaration :=
  withGroup (guardedGroup [{
    requestCase with exception := some {
      id := requestCase.id
      source
      condition := pendingOne
    }
  }])

private def emptyGroup : PropertyDeclaration :=
  withGroup (guardedGroup [])

private def emptyCase : PropertyDeclaration :=
  withGroup (guardedGroup [{ requestCase with clauses := [] }])

private def wrongReferenceKind : PropertyDeclaration :=
  withGroup { guardedGroup with guard := guardAtom .selectedAction pendingCount (.text "1") }

private def unknownReference : PropertyDeclaration :=
  withGroup { guardedGroup with guard :=
    (guardAtom .selectedAction (id "test.action.missing") (.text "missing")) }

private def undeclaredReference : PropertyDeclaration :=
  withGroup (guardedGroup [{ requestCase with clauses := [
    expectation "test.property.case.hidden" .expectationFact hiddenObservation .present
  ] }])

private def invalidGuardContext : PropertyDeclaration :=
  withGroup { guardedGroup with guard :=
    (guardAtom .resultingState pendingCount (.natural 1)) }

private def invalidExpectationContext : PropertyDeclaration :=
  withGroup (guardedGroup [{ requestCase with clauses := [
    expectation "test.property.case.invalid-expectation" .selectedAction requestCancel .present
  ] }])

private def emptyOperator : PropertyDeclaration :=
  withGroup { guardedGroup with guard := .all [] }

private def mixedLiteralTypes : PropertyDeclaration :=
  withGroup { guardedGroup with guard := .atom {
    field := .selectedAction
    reference := requestCancel
    constraint := .oneOf [.text "request", .natural 1]
  } }

private def wrongTemporalUnit : PropertyDeclaration := {
  guardedTemporalDeclaration with clauses := [
    .guardedEventuallyWithin (id "test.property.guarded-temporal.bad-unit") source
      requestGuard none
      (pattern .observation cancelRequested)
      (pattern .observation cancelDelivered)
      (.exact { value := 1, unit := .candidateEvaluations })
  ]
}

private def legacyVersion : PropertyDeclaration :=
  { guardedDeclaration with version := 1 }

private def malformedCaseAtOwnedSource : PropertyDeclaration :=
  withGroup (guardedGroup [{ requestCase with
    id := DefinitionId.of "Bad Case"
    source := malformedSource
  }])

private def duplicateClauseAtOwnedSource : PropertyDeclaration :=
  let first := {
    expectation "test.property.case.duplicate-source" .resultingState pendingCount .present with
    source := malformedSource
  }
  let second := { first with source := duplicateSource }
  withGroup (guardedGroup [{ requestCase with clauses := [first, second] }])

private def unknownReferenceAtOwnedSource : PropertyDeclaration :=
  withGroup { guardedGroup with
    source := referenceSource
    guard := guardAtom .selectedAction (id "test.action.missing") (.text "missing")
  }

/- Every guarded structural, reference, context, operator, unit, and version failure is typed. -/
#guard [
    duplicateCases,
    malformedParent,
    malformedCase,
    malformedClause,
    duplicateException,
    emptyGroup,
    emptyCase,
    wrongReferenceKind,
    unknownReference,
    undeclaredReference,
    invalidGuardContext,
    invalidExpectationContext,
    emptyOperator,
    mixedLiteralTypes,
    wrongTemporalUnit,
    legacyVersion
  ].map (fun declaration => (checkError? declaration).map PropertyError.kind) == [
    some .duplicateDefinitionId,
    some .emptyDefinitionId,
    some .invalidDefinitionId,
    some .emptyDefinitionId,
    some .duplicateDefinitionId,
    some .emptyCaseGroup,
    some .emptyCase,
    some .wrongReferenceKind,
    some .unknownReference,
    some .undeclaredReference,
    some .invalidPredicateContext,
    some .invalidPredicateContext,
    some .emptyBooleanGroup,
    some .typeMismatch,
    some .unitMismatch,
    some .unsupportedPropertyVersion
  ]

/- Malformed, duplicate, and reference diagnostics retain the offending declaration's complete
source coordinate and exact related identity. -/
#guard [
    malformedCaseAtOwnedSource,
    duplicateClauseAtOwnedSource,
    unknownReferenceAtOwnedSource
  ].map (fun declaration => (checkError? declaration).map fun error =>
    (error.kind, error.sourceLocation, error.relatedDefinitionIds)) == [
    some (.invalidDefinitionId, some malformedSource, [DefinitionId.of "Bad Case"]),
    some (.duplicateDefinitionId, some duplicateSource,
      [id "test.property.case.duplicate-source"]),
    some (.unknownReference, some referenceSource, [id "test.action.missing"])
  ]

#guard (checkError? malformedCaseAtOwnedSource).map canonicalPropertyErrorJson == some
  ("{\"kind\":\"invalid-definition-id\",\"definitionId\":\"test.property.guarded-delivery\"," ++
    "\"sourcePath\":\"Umpire/Property/Tests/MalformedGuardedCases.lean\"," ++
    "\"source\":{\"path\":\"Umpire/Property/Tests/MalformedGuardedCases.lean\"," ++
    "\"line\":41,\"column\":7,\"provenance\":\"generated-test\"}," ++
    "\"offendingValue\":\"Bad Case\",\"relatedDefinitionIds\":[\"Bad Case\"]}")

private def failedCaseClauseIdentity : Option PropertyClauseIdentity := do
  let evaluation ← evaluation? guardedDeclaration (requestTrace 0 2)
  let groupResult ← evaluation.clauses.find? fun result =>
    result.clauseId == (guardedGroup).id
  groupResult.failedObligations.head?

/- Case identity is part of a failed expectation even when local clause keys may repeat. -/
#guard failedCaseClauseIdentity.map (fun failure =>
    (failure.parentId, failure.caseId, failure.clauseId, failure.kind)) ==
  some ((guardedGroup).id, some requestCase.id,
    id "test.property.case.request.state", .clause)

private def repeatedLocalClauseIdentity : Option
    (List (Option DefinitionId × DefinitionId)) := do
  let localClause := expectation "test.property.case.shared.state" .resultingState pendingCount
    (.equals (.natural 1))
  let first := { requestCase with clauses := [localClause] }
  let second := {
    requestCase with
    id := id "test.property.case.request.sibling"
    clauses := [localClause]
  }
  let declaration := groupOnlyDeclaration
    (guardedGroup [first, second] (exclusive := false))
  let evaluation ← evaluation? declaration (requestTrace 0 2)
  let result ← evaluation.clauses.find? fun item => item.clauseId == (guardedGroup).id
  pure (result.failedObligations.map fun failure => (failure.caseId, failure.clauseId))

/- Equal local clause IDs in distinct cases remain distinct full clause identities. -/
#guard repeatedLocalClauseIdentity == some [
  (some requestCase.id, id "test.property.case.shared.state"),
  (some (id "test.property.case.request.sibling"), id "test.property.case.shared.state")
]

end Umpire.PropertyTests
