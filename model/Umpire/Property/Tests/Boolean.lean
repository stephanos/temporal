import Umpire.Property.Tests.Fixtures

/-! Typed Boolean Property predicate checking, same-step contexts, and semantic agreement. -/

namespace Umpire.PropertyTests

open Umpire

private def predicateCheck
    (contextKind : PropertyPredicateContext)
    (predicate : PropertyPredicate) :
    Except PropertyError (CheckedPropertyPredicate contextKind) :=
  checkPropertyPredicate context portableProperty contextKind predicate

private def predicateErrorKind
    {contextKind : PropertyPredicateContext}
    (result : Except PropertyError (CheckedPropertyPredicate contextKind)) :
    Option PropertyErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

private def predicateInputErrorKind
    {contextKind : PropertyPredicateContext}
    (predicate : CheckedPropertyPredicate contextKind)
    (input : PropertyPredicateInput) : Option PropertyErrorKind :=
  match checkPropertyPredicateInput predicate input with
  | .ok _ => none
  | .error error => some error.kind

def guardPredicate : PropertyPredicate :=
  .all [
    .atom {
      field := .priorState
      reference := pendingCount
      constraint := .equals (.natural 0)
    },
    .any [
      .atom {
        field := .selectedAction
        reference := requestCancel
        constraint := .equals (.text "request")
      },
      .not (.atom {
        field := .selectedAction
        reference := tick
        constraint := .equals (.text "tick")
      })
    ]
  ]

def checkedGuardPredicate : CheckedPropertyPredicate .guard :=
  checkedPropertyPredicate context portableProperty .guard guardPredicate (by native_decide)

def completeGuardInput : PropertyPredicateInput := {
  context := .guard
  priorState := some (value pendingCount "0")
  selectedAction := some (value requestCancel "request")
}

def checkedGuardInput : CheckedPropertyPredicateInput checkedGuardPredicate :=
  checkedPropertyPredicateInput checkedGuardPredicate completeGuardInput (by native_decide)

example : evaluatePropertyPredicate checkedGuardPredicate checkedGuardInput = true := by
  native_decide

example :
    evaluatePropertyPredicate checkedGuardPredicate checkedGuardInput = true ↔
      checkedGuardPredicate.denote checkedGuardInput :=
  evaluatePropertyPredicate_agrees checkedGuardPredicate checkedGuardInput

def sameFieldOneOf : PropertyPredicate :=
  .atom {
    field := .selectedAction
    reference := requestCancel
    constraint := .oneOf [.text "defer", .text "request"]
  }

def checkedSameFieldOneOf : CheckedPropertyPredicate .guard :=
  checkedPropertyPredicate context portableProperty .guard sameFieldOneOf (by native_decide)

example :
    (checkPropertyPredicateInput checkedSameFieldOneOf completeGuardInput).toOption.map
      (evaluatePropertyPredicate checkedSameFieldOneOf) = some true := by
  native_decide

def alternateActionInput : PropertyPredicateInput := {
  completeGuardInput with selectedAction := some (value tick "tick")
}

/-- A valid selected alternative that does not match remains ordinary false. -/
example :
    (checkPropertyPredicateInput checkedSameFieldOneOf alternateActionInput).toOption.map
      (evaluatePropertyPredicate checkedSameFieldOneOf) = some false := by
  native_decide

example : [
    predicateErrorKind (predicateCheck .guard (.all [])),
    predicateErrorKind (predicateCheck .guard (.any [])),
    predicateErrorKind (predicateCheck .guard (.atom {
      field := .selectedAction
      reference := requestCancel
      constraint := .oneOf []
    }))
  ] = [some .emptyBooleanGroup, some .emptyBooleanGroup, some .emptyBooleanGroup] := by
  native_decide

def mixedLiteralTypes : PropertyPredicate :=
  .atom {
    field := .selectedAction
    reference := requestCancel
    constraint := .oneOf [.text "request", .natural 1]
  }

example : predicateErrorKind (predicateCheck .guard mixedLiteralTypes) = some .typeMismatch := by
  native_decide

def resultingStateGuard : PropertyPredicate :=
  .atom {
    field := .resultingState
    reference := pendingCount
    constraint := .equals (.natural 1)
  }

def priorStateExpectation : PropertyPredicate :=
  .atom {
    field := .priorState
    reference := pendingCount
    constraint := .equals (.natural 0)
  }

example : [
    predicateErrorKind (predicateCheck .guard resultingStateGuard),
    predicateErrorKind (predicateCheck .expectation priorStateExpectation)
  ] = [some .invalidPredicateContext, some .invalidPredicateContext] := by
  native_decide

def unknownPredicateReference : PropertyPredicate :=
  .atom {
    field := .priorState
    reference := id "test.state.unknown"
  }

def wrongKindPredicateReference : PropertyPredicate :=
  .atom {
    field := .priorState
    reference := requestCancel
  }

def undeclaredPredicateReference : PropertyPredicate :=
  .atom {
    field := .expectationFact
    reference := hiddenObservation
  }

example : [
    predicateErrorKind (predicateCheck .guard unknownPredicateReference),
    predicateErrorKind (predicateCheck .guard wrongKindPredicateReference),
    predicateErrorKind (predicateCheck .expectation undeclaredPredicateReference)
  ] = [some .unknownReference, some .wrongReferenceKind, some .undeclaredReference] := by
  native_decide

def eagerUnknownPredicate : PropertyPredicate :=
  .any [
    .atom {
      field := .priorState
      reference := pendingCount
      constraint := .equals (.natural 0)
    },
    unknownPredicateReference
  ]

example : predicateErrorKind (predicateCheck .guard eagerUnknownPredicate) =
    some .unknownReference := by
  native_decide

def predicateMissingCapabilityContext : PropertyCheckContext := {
  context with providers := context.providers.filter fun provider =>
    provider.id != cancellationCapability
}

def predicateWrongCapabilityOwner : PropertyDeclaration := {
  portableProperty with requires := [pendingCount]
}

example : [
    predicateErrorKind (checkPropertyPredicate predicateMissingCapabilityContext portableProperty
      .guard guardPredicate),
    predicateErrorKind (checkPropertyPredicate context predicateWrongCapabilityOwner
      .guard guardPredicate)
  ] = [some .missingCapability, some .wrongReferenceKind] := by
  native_decide

def missingActionInput : PropertyPredicateInput := {
  completeGuardInput with selectedAction := none
}

/-- Missing input remains a diagnostic beneath nested negation and disjunction. -/
def nestedMissingPredicate : CheckedPropertyPredicate .guard :=
  checkedPropertyPredicate context portableProperty .guard
    (.not (.any [
      .atom {
        field := .priorState
        reference := pendingCount
        constraint := .equals (.natural 0)
      },
      .not (.atom {
        field := .selectedAction
        reference := requestCancel
        constraint := .equals (.text "request")
      })
    ])) (by native_decide)

example : predicateInputErrorKind nestedMissingPredicate missingActionInput =
    some .missingPredicateInput := by
  native_decide

/-- A true disjunct cannot mask a malformed payload in another child. -/
def eagerPayloadPredicate : CheckedPropertyPredicate .guard :=
  checkedPropertyPredicate context portableProperty .guard
    (.any [
      .atom {
        field := .priorState
        reference := pendingCount
        constraint := .equals (.natural 0)
      },
      .atom {
        field := .selectedAction
        reference := requestCancel
        constraint := .equals (.natural 1)
      }
    ]) (by native_decide)

example : predicateInputErrorKind eagerPayloadPredicate completeGuardInput =
    some .invalidPredicatePayload := by
  native_decide

def unsupportedSelectedActionInput : PropertyPredicateInput := {
  completeGuardInput with selectedAction := some (value hiddenObservation "private-record")
}

example : predicateInputErrorKind checkedSameFieldOneOf unsupportedSelectedActionInput =
    some .unsupportedPredicateInput := by
  native_decide

def expectationPredicate : CheckedPropertyPredicate .expectation :=
  checkedPropertyPredicate context portableProperty .expectation
    (.all [
      .atom {
        field := .resultingState
        reference := pendingCount
        constraint := .oneOf [.natural 0, .natural 1]
      },
      .atom {
        field := .modelOutcome
        reference := deliveredOutcome
        constraint := .equals (.text "delivered")
      },
      .atom {
        field := .expectationFact
        reference := cancelRequested
        constraint := .present
      }
    ]) (by native_decide)

def completeExpectationInput : PropertyPredicateInput := {
  context := .expectation
  resultingState := some (value pendingCount "1")
  modelOutcome := some (value deliveredOutcome "delivered")
  facts := some [value cancelRequested "request-1"]
}

example :
    (checkPropertyPredicateInput expectationPredicate completeExpectationInput).toOption.map
      (evaluatePropertyPredicate expectationPredicate) = some true := by
  native_decide

def missingFactsInput : PropertyPredicateInput := {
  completeExpectationInput with facts := none
}

example : predicateInputErrorKind expectationPredicate missingFactsInput =
    some .missingPredicateInput := by
  native_decide

def knownAbsentFactInput : PropertyPredicateInput := {
  completeExpectationInput with facts := some []
}

/-- A complete fact collection with no matching fact is known false, not unknown. -/
example :
    (checkPropertyPredicateInput expectationPredicate knownAbsentFactInput).toOption.map
      (evaluatePropertyPredicate expectationPredicate) = some false := by
  native_decide

def wrongContextInput : PropertyPredicateInput := {
  completeExpectationInput with context := .guard
}

example : predicateInputErrorKind expectationPredicate wrongContextInput =
    some .invalidPredicateContext := by
  native_decide

/- Arbitrary callbacks, cross-field equality, and guarded clauses have no portable constructors. -/
#guard_msgs (error, substring := true) in
#check PropertyPredicate.callback

#guard_msgs (error, substring := true) in
#check PropertyPredicate.crossFieldEquals

#guard_msgs (error, substring := true) in
#check PropertyPredicate.eventuallyWithin

#guard_msgs (error, substring := true) in
#check PropertyPredicateField.futureState

#guard_msgs (error, substring := true) in
#check PropertyClause.guarded

#guard_msgs (error, substring := true) in
def mixedCheckedContexts : Bool :=
  evaluatePropertyPredicate expectationPredicate checkedGuardInput

#guard_msgs (error, substring := true) in
def reusedCheckedInput : Bool :=
  evaluatePropertyPredicate checkedSameFieldOneOf checkedGuardInput

#guard_msgs (error, substring := true) in
def forgedCheckedInput : CheckedPropertyPredicateInput checkedGuardPredicate := {
  input := completeGuardInput
}

#guard_msgs (error, substring := true) in
def forgedCheckedPredicate : CheckedPropertyPredicate .guard := {
  ownerId := portableProperty.id
  source := portableProperty.source
  predicate := guardPredicate
  access := {
    capabilities := []
    meanings := []
    logicalTimeSource := none
  }
}

#print axioms Umpire.evaluatePropertyPredicate_agrees

end Umpire.PropertyTests
