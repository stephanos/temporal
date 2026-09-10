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

private def requestCase : PropertyBranch := {
  id := id "test.property.case.request"
  source
  guard := requestGuard
  clauses := [
    expectation "test.property.case.request.state" .resultingState pendingCount
      (.equals (.natural 1)),
    expectation "test.property.case.request.outcome" .outcome deliveredOutcome
      (.equals (.text "delivered"))
  ]
}

private def resolutionCase : PropertyBranch := {
  id := id "test.property.case.resolution"
  source
  guard := tickGuard
  clauses := [
    expectation "test.property.case.resolution.fact" .expectationFact cancelDelivered .present
  ]
}

private def guardedGroup
    (cases : List PropertyBranch := [requestCase, resolutionCase])
    (complete : Bool := true)
    (exclusive : Bool := true)
    (exception : Option PropertyUnless := none) : PropertyBranches := {
  id := id "test.property.case-group.delivery"
  source
  guard := .any [requestGuard, tickGuard]
  exception
  cases
  complete
  exclusive
}

private def guardedDeclaration
    (group : PropertyBranches := guardedGroup) : Property := {
  portableProperty with
  id := id "test.property.guarded-delivery"
  version := 2
  clauses := [cancelIsUnique, .branches group]
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
    (declaration : Property)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    Option PropertyEvaluation := do
  let property ← (Property.check context (declaration)).toOption
  (evaluatePropertyOnTrace property trace).toOption

#guard (evaluation? guardedDeclaration (requestTrace 0 1)).map
  PropertyEvaluation.satisfied == some true

example (property : CheckedProperty) (input : CheckedPropertyEvaluationInput property) :
    (evaluateProperty property input).satisfied = true ↔ property.denote input :=
  evaluateProperty_agrees property input

private def exceptedRequestCase : PropertyBranch := {
  requestCase with
  exception := some {
    id := id "test.property.case.request.exception.pending"
    source
    condition := pendingOne
  }
}

private def replacementCase : PropertyBranch := {
  id := id "test.property.case.request.replacement"
  source
  guard := .all [requestGuard, pendingOne]
  clauses := [
    expectation "test.property.case.request.replacement.state" .resultingState pendingCount
      (.equals (.natural 0))
  ]
}

private def missingReplacementDeclaration : Property :=
  guardedDeclaration (guardedGroup [exceptedRequestCase])

private def replacementDeclaration : Property :=
  guardedDeclaration (guardedGroup [exceptedRequestCase, replacementCase])

#guard [
    (evaluation? missingReplacementDeclaration (requestTrace 1 0)).map
      PropertyEvaluation.satisfied,
    (evaluation? replacementDeclaration (requestTrace 1 0)).map PropertyEvaluation.satisfied
  ] == [some false, some true]

private def parentExcludedDeclaration : Property :=
  let group : PropertyBranches := {
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

private def groupOnlyDeclaration (group : PropertyBranches) : Property := {
  guardedDeclaration group with clauses := [.branches group]
}

private def falseParentDeclaration : Property :=
  groupOnlyDeclaration { guardedGroup with guard := tickGuard }

private def incompleteAllowedDeclaration : Property :=
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

private def compatibleOverlapDeclaration : Property :=
  let duplicate := { requestCase with id := id "test.property.case.request.also" }
  guardedDeclaration (guardedGroup [requestCase, duplicate] (exclusive := false))

private def exclusiveOverlapDeclaration : Property :=
  let duplicate := { requestCase with id := id "test.property.case.request.also" }
  guardedDeclaration (guardedGroup [requestCase, duplicate])

#guard [
    (evaluation? compatibleOverlapDeclaration (requestTrace 0 1)).map
      PropertyEvaluation.satisfied,
    (evaluation? exclusiveOverlapDeclaration (requestTrace 0 1)).map
      PropertyEvaluation.satisfied
  ] == [some true, some false]

private def failingOverlapCase : PropertyBranch := {
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
    (declaration : Property)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) : Option PropertyErrorKind := do
  let property ← (Property.check context (declaration)).toOption
  match evaluatePropertyOnTrace property trace with
  | .ok _ => none
  | .error error => some error.kind

#guard evaluationErrorKind? guardedDeclaration incompleteTrace == some .missingPredicateInput

/- Known absence is an ordinary false expectation rather than an unknown-input diagnostic. -/
#guard (evaluation? guardedDeclaration tickTrace).map PropertyEvaluation.satisfied == some false

private def guardedTemporalDeclaration : Property := {
  guardedDeclaration with
  id := id "test.property.guarded-temporal"
  clauses := [
    .eventuallyWithin
      (id := (id "test.property.guarded-temporal.clause"))
      (source := source)
      (guard := some requestGuard)
      (exception := none)
      (trigger := (pattern .observation cancelRequested))
      (response := (pattern .observation cancelDelivered))
      (limit := { value := 1, unit := .steps })
  ]
}

private def guardedQuiescentDeclaration : Property := {
  guardedTemporalDeclaration with
  id := id "test.property.guarded-quiescent"
  clauses := [
    .neverWithin
      (id := (id "test.property.guarded-quiescent.clause"))
      (source := source)
      (guard := some requestGuard)
      (exception := none)
      (trigger := (pattern .observation cancelDelivered))
      (forbidden := (pattern .observation cancelRequested))
      (limit := { value := 1, unit := .steps })
  ]
}

#guard [guardedTemporalDeclaration, guardedQuiescentDeclaration].map (fun declaration =>
    (Property.check context (declaration)).toOption.map fun property =>
      property.clauses.map CheckedPropertyClause.id) == [
    some [id "test.property.guarded-temporal.clause"],
    some [id "test.property.guarded-quiescent.clause"]
  ]

private def reorderedDeclaration : Property :=
  guardedDeclaration (guardedGroup [resolutionCase, requestCase])

private def reorderedClausesDeclaration : Property := {
  guardedDeclaration with clauses := guardedDeclaration.clauses.reverse
}

private def changedGuardDeclaration : Property :=
  guardedDeclaration { guardedGroup with guard := requestGuard }

private def changedCaseGuardDeclaration : Property :=
  guardedDeclaration (guardedGroup [{ requestCase with guard := pendingOne }, resolutionCase])

private def changedExceptionDeclaration : Property :=
  guardedDeclaration (guardedGroup (exception := some {
    id := id "test.property.case-group.delivery.exception"
    source
    condition := pendingOne
  }))

private def changedExpectationDeclaration : Property :=
  let changedRequest := { requestCase with clauses := [
    expectation "test.property.case.request.state" .resultingState pendingCount
      (.equals (.natural 2)),
    expectation "test.property.case.request.outcome" .outcome deliveredOutcome
      (.equals (.text "delivered"))
  ] }
  guardedDeclaration (guardedGroup [changedRequest, resolutionCase])

private def changedCompleteDeclaration : Property :=
  guardedDeclaration (guardedGroup (complete := false))

private def changedExclusiveDeclaration : Property :=
  guardedDeclaration (guardedGroup (exclusive := false))

private def changedGuardedSourceDeclaration : Property := {
  guardedDeclaration with source := { source with line := source.line + 1 }
}

private def changedGuardedDocumentationDeclaration : Property := {
  guardedDeclaration with documentation := "Updated guarded Property documentation."
}

private def changedNestedSourceDeclaration : Property :=
  let nestedSource := { source with line := source.line + 1 }
  let changedRequest := {
    requestCase with
    source := nestedSource
    clauses := requestCase.clauses.map fun clause => { clause with source := nestedSource }
  }
  guardedDeclaration {
    guardedGroup [changedRequest, resolutionCase] with source := nestedSource
  }

private def fingerprintOf (declaration : Property) : Option BehaviorFingerprint :=
  (Property.check context (declaration)).toOption.map CheckedProperty.behaviorFingerprint

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
#guard (Property.check context (guardedDeclaration)).toOption.map
    (fun property => !property.canonicalMetadata.contains "temporalClauses") == some true
#guard (fingerprintOf guardedDeclaration).map BehaviorFingerprint.render ==
  some "sha256:1338637fd2543d225f061d77e6b15d4b9d35245ec301b95a540cc1bd4266e56f"

private def checkError?
    (declaration : Property) : Option PropertyError :=
  match Property.check context (declaration) with
  | .ok _ => none
  | .error error => some error

private def withGroup (group : PropertyBranches) : Property :=
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

private def duplicateCases : Property :=
  withGroup (guardedGroup [requestCase, requestCase])

private def malformedParent : Property :=
  withGroup { guardedGroup with id := DefinitionId.of "" }

private def malformedCase : Property :=
  withGroup (guardedGroup [{ requestCase with id := DefinitionId.of "Bad Case" }])

private def malformedClause : Property :=
  withGroup (guardedGroup [{ requestCase with clauses := [
    expectation "" .resultingState pendingCount (.equals (.natural 1))
  ] }])

private def duplicateException : Property :=
  withGroup (guardedGroup [{
    requestCase with exception := some {
      id := requestCase.id
      source
      condition := pendingOne
    }
  }])

private def emptyGroup : Property :=
  withGroup (guardedGroup [])

private def emptyCase : Property :=
  withGroup (guardedGroup [{ requestCase with clauses := [] }])

private def wrongReferenceKind : Property :=
  withGroup { guardedGroup with guard := guardAtom .selectedAction pendingCount (.text "1") }

private def unknownReference : Property :=
  withGroup { guardedGroup with guard :=
    (guardAtom .selectedAction (id "test.action.missing") (.text "missing")) }

private def undeclaredReference : Property :=
  withGroup (guardedGroup [{ requestCase with clauses := [
    expectation "test.property.case.hidden" .expectationFact hiddenObservation .present
  ] }])

private def invalidGuardContext : Property :=
  withGroup { guardedGroup with guard :=
    (guardAtom .resultingState pendingCount (.natural 1)) }

private def invalidExpectationContext : Property :=
  withGroup (guardedGroup [{ requestCase with clauses := [
    expectation "test.property.case.invalid-expectation" .selectedAction requestCancel .present
  ] }])

private def emptyOperator : Property :=
  withGroup { guardedGroup with guard := .all [] }

private def mixedLiteralTypes : Property :=
  withGroup { guardedGroup with guard := .atom {
    field := .selectedAction
    reference := requestCancel
    constraint := .oneOf [.text "request", .natural 1]
  } }

private def wrongTemporalUnit : Property := {
  guardedTemporalDeclaration with clauses := [
    .eventuallyWithin
      (id := (id "test.property.guarded-temporal.bad-unit"))
      (source := source)
      (guard := some requestGuard)
      (exception := none)
      (trigger := (pattern .observation cancelRequested))
      (response := (pattern .observation cancelDelivered))
      (limit := { value := 1, unit := .search })
  ]
}

private def legacyVersion : Property :=
  { guardedDeclaration with version := 1 }

private def malformedCaseAtOwnedSource : Property :=
  withGroup (guardedGroup [{ requestCase with
    id := DefinitionId.of "Bad Case"
    source := malformedSource
  }])

private def duplicateClauseAtOwnedSource : Property :=
  let first := {
    expectation "test.property.case.duplicate-source" .resultingState pendingCount .present with
    source := malformedSource
  }
  let second := { first with source := duplicateSource }
  withGroup (guardedGroup [{ requestCase with clauses := [first, second] }])

private def unknownReferenceAtOwnedSource : Property :=
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
