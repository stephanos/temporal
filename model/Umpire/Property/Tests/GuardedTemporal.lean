import Umpire.Property.Tests.Fixtures

/-! Trigger-frozen guarded bounded Property admission, evaluation, provenance, and agreement. -/

namespace Umpire.PropertyTests

open Umpire

private def guardAtom
    (field : PropertyPredicateField)
    (reference : DefinitionId)
    (literal : PropertyLiteral) : PropertyPredicate :=
  .atom { field, reference, constraint := .equals literal }

private def requestGuard : PropertyPredicate :=
  .all [
    guardAtom .priorState pendingCount (.natural 0),
    guardAtom .selectedAction requestCancel (.text "request")
  ]

private def pendingOne : PropertyPredicate :=
  guardAtom .priorState pendingCount (.natural 1)

private def temporalException : PropertyUnless := {
  id := id "test.property.guarded-temporal.exception.pending"
  source
  condition := pendingOne
}

private def parentException : PropertyUnless := {
  id := id "test.property.guarded-temporal.exception.parent"
  source
  condition := guardAtom .priorState pendingCount (.natural 2)
}

private def caseException : PropertyUnless := {
  id := id "test.property.guarded-temporal.exception.case"
  source
  condition := pendingOne
}

private def guardedEventually
    (exception : Option PropertyUnless := none)
    (limit : Limit := { value := 1, unit := .semanticTransitions }) :
    PropertyClause :=
  .eventuallyWithin
      (id := (id "test.property.guarded-temporal.eventually"))
      (source := source)
      (guard := some requestGuard)
      («unless» := exception)
      (trigger := (pattern .observation cancelRequested))
      (response := (pattern .observation cancelDelivered))
      (limit := limit)

private def guardedQuiescent
    (exception : Option PropertyUnless := none) : PropertyClause :=
  .neverWithin
      (id := (id "test.property.guarded-temporal.quiescent"))
      (source := source)
      (guard := some requestGuard)
      («unless» := exception)
      (trigger := (pattern .observation cancelRequested))
      (forbidden := (pattern .observation cancelDelivered))
      (limit := { value := 1, unit := .semanticTransitions })

private def declaration
    (clauses : List PropertyClause := [guardedEventually]) : Property := {
  portableProperty with
  id := id "test.property.guarded-temporal"
  version := 2
  clauses
}

private def caseTemporalGroup
    (parentException? : Option PropertyUnless := none)
    (caseException? : Option PropertyUnless := none)
    (caseId : DefinitionId := id "test.property.guarded-temporal.case")
    (clauseId : DefinitionId := id "test.property.guarded-temporal.case.eventually") :
    PropertyClause :=
  .branches {
    id := id "test.property.guarded-temporal.group"
    source
    guard := guardAtom .selectedAction requestCancel (.text "request")
    exception := parentException?
    cases := [{
      id := caseId
      source
      guard := guardAtom .priorState pendingCount (.natural 0)
      exception := caseException?
      clauses := []
      temporalClauses := [.eventuallyWithin clauseId source
        (pattern .observation cancelRequested)
        (pattern .observation cancelDelivered)
        { value := 1, unit := .semanticTransitions }]
    }]
  }

private def trace
    (initial : Nat)
    (firstObservations secondObservations : List ModelValue)
    (afterFirst : Nat := 1) : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := value pendingCount (toString initial)
  steps := [
    {
      selectedAction := value requestCancel "request"
      outcome := value deliveredOutcome "delivered"
      state := value pendingCount (toString afterFirst)
      facts := firstObservations
    },
    {
      selectedAction := value tick "tick"
      outcome := value deliveredOutcome "delivered"
      state := value pendingCount (toString afterFirst)
      facts := secondObservations
    }
  ]
}

private def trigger := value cancelRequested "request-1"
private def response := value cancelDelivered "request-1"

private def checked? (authored : Property := declaration) : Option CheckedProperty :=
  (Property.check context (authored)).toOption

private def fingerprintOf
    (authored : Property := declaration) : Option BehaviorFingerprint :=
  (checked? authored).map CheckedProperty.behaviorFingerprint

private def satisfied?
    (authored : Property)
    (modelTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue) : Option Bool := do
  let property ← checked? authored
  let result ← (evaluatePropertyOnTrace property modelTrace).toOption
  pure result.satisfied

/- Guarded bounded forms are admitted only as version-two checked Property data. -/
#guard (checked?).map (fun property =>
    (property.version, property.clauses.map CheckedPropertyClause.id,
      property.canonicalMetadata.contains "guarded-eventually-within")) ==
  some (2, [id "test.property.guarded-temporal.eventually"], true)

/- Named cases retain their parent/case shape while using a legacy single-pattern response. -/
#guard (checked? (declaration [caseTemporalGroup])).map (fun property =>
    (property.canonicalMetadata.contains "branches",
      property.canonicalMetadata.contains "guarded-eventually-within")) ==
  some (true, true)

/- The two guarded temporal kinds remain canonically and behaviorally distinguishable. -/
#guard (do
    let eventually ← checked? (declaration [guardedEventually])
    let quiescent ← checked? (declaration [guardedQuiescent])
    pure (
      eventually.behaviorFingerprint != quiescent.behaviorFingerprint,
      eventually.canonicalMetadata.contains "guarded-eventually-within",
      quiescent.canonicalMetadata.contains "guarded-never-within")) ==
  some (true, true, true)

private def changedBoundDeclaration :=
  declaration [guardedEventually (limit := { value := 2, unit := .semanticTransitions })]

private def changedTriggerDeclaration :=
  declaration [.eventuallyWithin
      (id := (id "test.property.guarded-temporal.eventually"))
      (source := source)
      (guard := some requestGuard)
      («unless» := none)
      (trigger := (pattern .observation cancelDelivered))
      (response := (pattern .observation cancelRequested))
      (limit := { value := 1, unit := .semanticTransitions })]

private def changedExceptionDeclaration :=
  declaration [guardedEventually (some temporalException)]

/- Guards, exceptions, references, kinds, and resolved bounds remain semantic identity inputs. -/
#guard [
    changedBoundDeclaration,
    changedTriggerDeclaration,
    changedExceptionDeclaration,
    declaration [guardedQuiescent]
  ].all fun authored => fingerprintOf declaration != fingerprintOf authored

private def changedSourceDeclaration : Property := {
  declaration with source := { source with line := source.line + 1 }
}

private def changedDocumentationDeclaration : Property := {
  declaration with documentation := "Updated guarded temporal documentation."
}

private def changedClauseSourceDeclaration :=
  declaration [.eventuallyWithin
      (id := (id "test.property.guarded-temporal.eventually"))
      (source := { source with line := source.line + 1 })
      (guard := some requestGuard)
      («unless» := none)
      (trigger := (pattern .observation cancelRequested))
      (response := (pattern .observation cancelDelivered))
      (limit := { value := 1, unit := .semanticTransitions })]

#guard [
    changedSourceDeclaration,
    changedDocumentationDeclaration,
    changedClauseSourceDeclaration
  ].all fun authored =>
  fingerprintOf declaration == fingerprintOf authored

/- The response is inclusive at the trigger, permitted one transition later, and required. -/
#guard [
    trace 0 [trigger, response] [],
    trace 0 [trigger] [response],
    trace 0 [trigger] []
  ].map (satisfied? declaration) == [some true, some true, some false]

/- A zero bound preserves its unit and excludes a response on the following transition. -/
private def zeroBoundDeclaration :=
  declaration [guardedEventually (limit := { value := 0, unit := .semanticTransitions })]

private def zeroBoundResult : Option (Bool × Option Limit) := do
  let property ← checked? zeroBoundDeclaration
  let evaluation ← (evaluatePropertyOnTrace property (trace 0 [trigger] [response])).toOption
  let clause ← evaluation.clauses.head?
  pure (evaluation.satisfied, clause.evaluatedLimit)

#guard zeroBoundResult == some (false, some { value := 0, unit := .semanticTransitions })

private def twoTriggerTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := value pendingCount "0"
  steps := [
    {
      selectedAction := value requestCancel "request"
      outcome := value deliveredOutcome "delivered"
      state := value pendingCount "0"
      facts := [trigger]
    },
    {
      selectedAction := value requestCancel "request"
      outcome := value deliveredOutcome "delivered"
      state := value pendingCount "0"
      facts := [trigger, response]
    }
  ]
}

/- Every applicable trigger creates an obligation; a later passing trigger cannot hide an earlier failure. -/
#guard satisfied? zeroBoundDeclaration twoTriggerTrace == some false

/- Exception truth is frozen at the triggering step; later state change cannot erase the work. -/
#guard [
    satisfied? (declaration [guardedEventually (some temporalException)])
      (trace 1 [trigger] []),
    satisfied? (declaration [guardedEventually (some temporalException)])
      (trace 0 [trigger] [])
  ] == [some true, some false]

/- Parent and case exceptions are independently frozen where the temporal trigger occurs. -/
#guard [
    satisfied? (declaration [caseTemporalGroup (some {
      parentException with condition := guardAtom .priorState pendingCount (.natural 0)
    })]) (trace 0 [trigger] []),
    satisfied? (declaration [caseTemporalGroup none (some {
      caseException with condition := guardAtom .priorState pendingCount (.natural 0)
    })]) (trace 0 [trigger] []),
    satisfied? (declaration [caseTemporalGroup (some parentException) (some caseException)])
      (trace 0 [trigger] []),
    satisfied? (declaration [caseTemporalGroup (some parentException) (some caseException)])
      (trace 0 [trigger] [] (afterFirst := 2))
  ] == [some true, some true, some false, some false]

private def parentExcludedTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := value pendingCount "0"
  steps := [{
    selectedAction := value tick "tick"
    outcome := value deliveredOutcome "delivered"
    state := value pendingCount "1"
    facts := [trigger]
  }]
}

/- A false parent guard and a true named exception independently exclude the bounded clause. -/
#guard [
    satisfied? declaration parentExcludedTrace,
    satisfied? (declaration [guardedEventually (some temporalException)])
      (trace 1 [trigger] [])
  ] == [some true, some true]

/- Excluding the temporal clause supplies no replacement and cannot waive an independent clause. -/
#guard satisfied?
    (declaration [cancelIsUnique, guardedEventually (some temporalException)])
    (trace 1 [trigger] [] (afterFirst := 2)) == some false

/- Source order cannot choose between an excluded clause and an independent failing invariant. -/
#guard [
    [cancelIsUnique, guardedEventually (some temporalException)],
    [guardedEventually (some temporalException), cancelIsUnique]
  ].map (fun clauses =>
    satisfied? (declaration clauses) (trace 1 [trigger] [] (afterFirst := 2))) ==
  [some false, some false]

private def overlappingTemporalCases :=
  declaration [
    .branches {
      id := id "test.property.guarded-temporal.overlap"
      source
      guard := requestGuard
      cases := [
        {
          id := id "test.property.guarded-temporal.case.passing"
          source
          guard := guardAtom .priorState pendingCount (.natural 0)
          clauses := []
          temporalClauses := [.eventuallyWithin
            (id "test.property.guarded-temporal.case.passing.eventually") source
            (pattern .observation cancelRequested)
            (pattern .observation cancelDelivered)
            { value := 1, unit := .semanticTransitions }]
        },
        {
          id := id "test.property.guarded-temporal.case.failing"
          source
          guard := guardAtom .priorState pendingCount (.natural 0)
          clauses := []
          temporalClauses := [.neverWithin
            (id "test.property.guarded-temporal.case.failing.quiescent") source
            (pattern .observation cancelRequested)
            (pattern .observation cancelDelivered)
            { value := 1, unit := .semanticTransitions }]
        }
      ]
    }
  ]

/- Applicable sibling temporal cases are conjoined rather than selected by source order. -/
#guard satisfied? overlappingTemporalCases (trace 0 [trigger, response] []) == some false

/- Guarded quiescence uses the same frozen trigger applicability and inclusive bound. -/
#guard [
    satisfied? (declaration [guardedQuiescent]) (trace 0 [trigger] []),
    satisfied? (declaration [guardedQuiescent]) (trace 0 [trigger, response] []),
    satisfied? (declaration [guardedQuiescent (some temporalException)])
      (trace 1 [trigger, response] [])
  ] == [some true, some false, some true]

private def unknownPriorTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := value hiddenObservation "0"
  steps := [{
    selectedAction := value requestCancel "request"
    outcome := value deliveredOutcome "delivered"
    state := value pendingCount "1"
    facts := [trigger]
  }]
}

private def unknownThroughNegation :=
  declaration [.eventuallyWithin
      (id := (id "test.property.guarded-temporal.unknown-negation"))
      (source := source)
      (guard := some (.not pendingOne))
      («unless» := none)
      (trigger := (pattern .observation cancelRequested))
      (response := (pattern .observation cancelDelivered))
      (limit := { value := 1, unit := .semanticTransitions })]

private def evaluationError? : Option (PropertyErrorKind × Option SourceLocation) := do
  let property ← checked? unknownThroughNegation
  match evaluatePropertyOnTrace property unknownPriorTrace with
  | .ok _ => none
  | .error error => some (error.kind, error.sourceLocation)

/- Negation cannot turn an unavailable trigger input into applicable truth. -/
#guard evaluationError? == some (.missingPredicateInput, some source)

private def invalidFutureGuard :=
  declaration [.eventuallyWithin
      (id := (id "test.property.guarded-temporal.future-guard"))
      (source := source)
      (guard := some (guardAtom .resultingState pendingCount (.natural 1)))
      («unless» := none)
      (trigger := (pattern .observation cancelRequested))
      (response := (pattern .observation cancelDelivered))
      (limit := { value := 1, unit := .semanticTransitions })]

/- Initial guards cannot inspect the result whose later response they govern. -/
#guard (match Property.check context (invalidFutureGuard) with
  | .error error => some (error.kind, error.sourceLocation)
  | .ok _ => none) == some (.invalidPredicateContext, some source)

#guard_msgs (error, substring := true) in
#check PropertyClause.guardedEventuallyUntil

#guard_msgs (error, substring := true) in
def compoundTemporalResponse : PropertyClause :=
  .eventuallyWithin
      (id := (id "test.property.guarded-temporal.compound"))
      (source := source)
      (guard := some requestGuard)
      («unless» := none)
      (trigger := (pattern .observation cancelRequested))
      (response := (.all [guardAtom .resultingState pendingCount (.natural 1)]))
      (limit := { value := 1, unit := .semanticTransitions })

private def failedTemporalIdentity : Option PropertyClauseIdentity := do
  let property ← checked? (declaration [guardedEventually (some temporalException)])
  let evaluation ← (evaluatePropertyOnTrace property (trace 0 [trigger] [])).toOption
  let result ← evaluation.clauses.head?
  result.failedObligations.head?

/- A failed pending obligation retains its Property, exception, clause, and source identities. -/
#guard failedTemporalIdentity.map (fun failure =>
    (failure.parentId, failure.caseId, failure.clauseId, failure.source,
      failure.relatedDefinitionIds)) == some (
    (declaration).id,
    none,
    id "test.property.guarded-temporal.eventually",
    source,
    [temporalException.id])

private def failedCaseTemporalIdentity : Option PropertyClauseIdentity := do
  let property ← checked? (declaration [caseTemporalGroup
    (some parentException) (some caseException)])
  let evaluation ← (evaluatePropertyOnTrace property (trace 0 [trigger] [])).toOption
  let result ← evaluation.clauses.head?
  result.failedObligations.head?

/- A case temporal failure retains both applicability exceptions and every qualified identity. -/
#guard failedCaseTemporalIdentity.map (fun failure =>
    (failure.parentId, failure.caseId, failure.clauseId, failure.source,
      failure.evaluatedLimit, failure.relatedDefinitionIds)) == some (
    id "test.property.guarded-temporal.group",
    some (id "test.property.guarded-temporal.case"),
    id "test.property.guarded-temporal.case.eventually",
    source,
    some { value := 1, unit := .semanticTransitions },
    [parentException.id, caseException.id])

example (property : CheckedProperty) (input : CheckedPropertyEvaluationInput property) :
    (evaluateProperty property input).satisfied = true ↔ property.denote input :=
  evaluateProperty_agrees property input

#print axioms Umpire.evaluatePropertyClause_agrees
#print axioms Umpire.evaluateProperty_agrees

end Umpire.PropertyTests
