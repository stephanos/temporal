import Umpire.Property.Tests.Fixtures

/-! Checked prefix evaluation preserves closed truth and inclusive semantic deadlines. -/

namespace Umpire.PropertyTests

private def endpointAnswer (clause : PropertyClause)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (partialTrace : Bool) (logicalTimeSource : Option DefinitionId := none) : Option PropertyEndpointAnswer := do
  let property ← (Property.check context ({ portableProperty with version := 2, clauses := [clause], logicalTimeSource })).toOption
  let input ← (checkPropertyEvaluationInput property trace).toOption
  pure (evaluatePropertyEndpoint property input partialTrace).answer

private def response (bound : Nat) : PropertyClause :=
  .eventuallyWithin (id "test.property.endpoint.response")
    (pattern .observation cancelRequested) (pattern .observation cancelDelivered)
    { value := bound, unit := .steps }

private def selectedPrefix : ModelTrace ModelValue ModelValue ModelValue ModelValue :=
  { positiveTrace with steps := positiveTrace.steps.take 1 }

#guard endpointAnswer (response 1) selectedPrefix true == some .unresolved
#guard endpointAnswer (response 1) selectedPrefix false == some .violated
#guard endpointAnswer (response 0) selectedPrefix true == some .violated
#guard endpointAnswer (response 1) positiveTrace true == some .satisfied
#guard endpointAnswer (response 0) positiveTrace true == some .violated

private def quiescent : PropertyClause :=
  .neverWithin (id "test.property.endpoint.quiet")
    (pattern .observation cancelRequested) (pattern .observation cancelDelivered)
    { value := 1, unit := .steps }

#guard endpointAnswer quiescent selectedPrefix true == some .unresolved
#guard endpointAnswer quiescent selectedPrefix false == some .satisfied
#guard endpointAnswer quiescent positiveTrace true == some .violated

private def guardedResponse : PropertyClause :=
  .eventuallyWithin
      (id := (id "test.property.endpoint.guarded"))
      (source := source)
      (guard := some (.atom {
      field := .selectedAction
      reference := requestCancel
      constraint := .equals (.text "request")
    }))
      (exception := none)
      (trigger := (pattern .observation cancelRequested))
      (response := (pattern .observation cancelDelivered))
      (limit := { value := 1, unit := .steps })

#guard endpointAnswer guardedResponse selectedPrefix true == some .unresolved
#guard endpointAnswer guardedResponse positiveTrace true == some .satisfied

private def guardedLogicalResponse : PropertyClause :=
  .eventuallyWithin
      (id := (id "test.property.endpoint.guarded-logical"))
      (source := source)
      (guard := some (.atom {
      field := .selectedAction
      reference := requestCancel
      constraint := .equals (.text "request")
    }))
      (exception := none)
      (trigger := (pattern .observation cancelRequested))
      (response := (pattern .observation cancelDelivered))
      (limit := { value := 1, unit := .logicalTime })

#guard ([none, some "not-a-time"] : List (Option String)).all fun coordinate =>
  let trace := { selectedPrefix with steps := selectedPrefix.steps.map fun step =>
    { step with facts := step.facts ++
      (coordinate.toList.map fun time => value logicalTime time) } }
  endpointAnswer guardedLogicalResponse trace true (some logicalTime) == some .unresolved

#guard endpointAnswer cancelIsUnique
  { initialState := value cancellationPhase "initial", steps := [] } true == some .unresolved
#guard endpointAnswer cancelIsUnique
  { initialState := value pendingCount "2", steps := [] } true == some .violated

#print axioms Umpire.evaluateProperty
#print axioms Umpire.evaluatePropertyEndpoint
#print axioms Umpire.evaluatePropertyEndpoint_closed

end Umpire.PropertyTests
