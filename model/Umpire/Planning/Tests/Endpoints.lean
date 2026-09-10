import Umpire.Planning.Tests.Fixtures
import Umpire.Target.FiniteMachine

/-! Endpoint interpretation and exact candidate-budget boundaries. -/

namespace Umpire.PlanningTests

#guard (run 0 (.verify property) .exhaustive 2).toOption.map
  (fun run => run.result.outcome.name) == some "verified-within-limits"

private def temporalProperty (triggered responds : Bool) (bound : Nat := 1) : CheckedProperty := {
  property with
  clauses := [.eventuallyWithin (id "planner.property.response")
    { field := .selectedAction, reference := request,
      constraint := .equals (if triggered then "request" else "absent") }
    { field := .observation, reference := observed,
      constraint := .equals (if responds then "accepted" else "absent") }
    { value := bound, unit := .semanticTransitions }]
  access := { capabilities := [], logicalTimeSource := none, meanings := [
    { definitionId := request, kind := .action, canonicalBehavior := "request" },
    { definitionId := observed, kind := .fact, canonicalBehavior := "observed" }] }
}

private def endpointRun
    (endpoint : QueryEndpoint) (exercise : QueryExercisePolicy)
    (form : QueryForm) (width : Nat := 0) (budget : Nat := 10) : Option PlannerRun :=
  (plan { checkedQuery width form .exhaustive budget with endpoint, exercise }
    (incrementalKernel width)).toOption

#guard (endpointRun .deliberatelyClosed .requireAllTriggers
  (.verify (temporalProperty false false))).map (·.result.outcome.name) ==
    some "nonempty-unexercised"

#guard (endpointRun .deliberatelyClosed .requireAllTriggers
  (.witness (temporalProperty false false))).map (·.result.outcome.name) ==
    some "nonempty-unexercised"

#guard (endpointRun .deliberatelyClosed .requireAllTriggers
  (.witness (temporalProperty true true))).map (·.result.metadata.validity.answer) == some .witness

#guard (endpointRun .deliberatelyClosed .requireAllTriggers
  (.verify (temporalProperty true true))).map (fun run =>
    (run.result.metadata.validity.satisfiability, run.result.metadata.validity.coverage,
      run.result.metadata.validity.answer, run.result.metadata.validity.searchComplete)) ==
    some (.nonempty, .exercised, .verified, true)

#guard (endpointRun .runtimePrefix .requireAllTriggers
  (.verify (temporalProperty true false))).map (fun run =>
    (run.result.outcome.name, run.result.isVerified, run.result.metadata.validity.searchComplete)) ==
    some ("unresolved-prefix", false, true)

#guard (endpointRun .runtimePrefix .requireAllTriggers
  (.verify (temporalProperty true false 0))).map (·.result.metadata.validity.answer) ==
    some .counterexample

#guard (endpointRun .deliberatelyClosed .requireAllTriggers
  (.verify (temporalProperty true false)) 10 2).map (fun run =>
    (run.result.metadata.validity.answer, run.result.metadata.validity.searchComplete,
      run.result.metadata.completeness.established,
      run.artifact.isSome, run.result.metadata.explored.traces)) ==
    some (.counterexample, false, false, true, 2)

#guard (endpointRun .deliberatelyClosed .requireAllTriggers
  (.verify (temporalProperty false false)) 10 2).map (fun run =>
    (run.result.outcome.name, run.result.metadata.validity.satisfiability,
      run.result.metadata.validity.coverage, run.result.metadata.validity.answer)) ==
    some ("limit-reached", .nonempty, .unknown, .unknown)

#guard (endpointRun .terminalModel .allowVacuous (.verify property)).map
  (·.result.metadata.validity.satisfiability) == some .impossible

#guard (endpointRun .deliberatelyClosed .requireAllTriggers
  (.verify (temporalProperty true true)) 2).map (fun run =>
    run.result.metadata.validity.triggers.map (fun evidence =>
      (evidence.trace.trace.steps.length, evidence.trigger.transitionPosition,
        evidence.trigger.occurrence.value))) ==
    some [(1, 1, some requestValue), (1, 1, some requestValue), (1, 1, some requestValue)]

#guard (endpointRun .deliberatelyClosed .requireAllTriggers
  (.verify (temporalProperty true true))).map (fun run =>
    (canonicalPlanningReceiptJson run.result).contains "checked-finite-enumeration/v1") == some true

#print axioms evaluatePropertyEndpoint_closed
#print axioms PlanningOutcome.constructorClassifiers_exactlyOne

private def terminalTarget (conditions : List (List ModelValue)) : Option (QueryTarget (fun _ => True)) :=
  (checkTarget (AuthoredTarget.make { targetDefinition 0 with terminalConditions := conditions }
    TargetComposition.empty (.available (kernel 0) rfl (finitePlanning 0)))).toOption

#guard (terminalTarget [[completed], [initial, completed]]).map (fun target =>
    (target.isTerminal initial, target.isTerminal completed)) == some (false, true)

#guard (terminalTarget [[completed], []]).map (·.isTerminal completed) == some false
#guard (terminalTarget [[completed], [initial]]).map (·.isTerminal completed) == some false
#guard (terminalTarget []).map (·.isTerminal completed) == some false

#guard (terminalTarget [[completed], [initial, completed]]).map (·.behaviorFingerprint) ==
  (terminalTarget [[completed, initial], [completed]]).map (·.behaviorFingerprint)
#guard (terminalTarget []).map (·.behaviorFingerprint) == some baseTarget.behaviorFingerprint
#guard (terminalTarget [[completed]]).map (·.behaviorFingerprint) != some baseTarget.behaviorFingerprint

private def terminalRun : Option PlannerRun := do
  let target ← terminalTarget [[completed], [completed, initial]]
  let query := { checkedQuery 0 (.verify property) .exhaustive with
    target
    completeness := (CheckedQueryTarget.ofTarget target).completeness
    endpoint := .terminalModel }
  let kernel ← (IncrementalPlannerKernel.ofCheckedQuery target.id query).toOption
  (plan query kernel).toOption

#guard terminalRun.map (·.result.outcome.name) == some "verified-within-limits"

private def queryDeclaration : QueryDeclaration := {
  id := id "planner.query.endpoints"
  source
  target := targetId
  form := .verify property
  behavior
  limits := limits
  policy := .exhaustive
}

private def admittedQuery (endpoint : QueryEndpoint) (exercise : QueryExercisePolicy) :
    Option (CheckedQuery (fun _ => True)) :=
  (checkQuery (.ofTarget baseTarget) { queryDeclaration with endpoint, exercise }).toOption

#guard (admittedQuery .deliberatelyClosed .allowVacuous).map (fun query =>
  query.canonicalMetadata.contains "endpointPolicy/v1") == some false
#guard (admittedQuery .runtimePrefix .requireAllTriggers).map (fun query =>
  (query.endpoint, query.exercise, query.canonicalMetadata.contains "endpointPolicy/v1")) ==
    some (.runtimePrefix, .requireAllTriggers, true)
#guard (admittedQuery .runtimePrefix .requireAllTriggers).map (·.behaviorFingerprint) !=
  (admittedQuery .terminalModel .requireAllTriggers).map (·.behaviorFingerprint)

private def convergingTarget : Option (QueryTarget (fun _ => True)) :=
  let other := value request "other"
  let table : FiniteTable (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
    setups := [⟨setup, "setup"⟩]
    states := [⟨initial, "initial"⟩, ⟨completed, "completed"⟩]
    actions := [⟨other, "other"⟩, ⟨requestValue, "request"⟩]
    outcomes := [⟨acceptedValue, "accepted"⟩]
    facts := [⟨observedValue, "observed"⟩]
    initial := [⟨setup, [initial]⟩]
    transitions := [
      ⟨"other", initial, other, [transition 0]⟩,
      ⟨"request", initial, requestValue, [{ transition 0 with facts := [] }]⟩]
  }
  (FiniteTable.checkTarget table {
    id := targetId
    source
    definitions := (targetDefinition 0).definitions
    requiredCapabilities := []
    metadata := (kernel 0).metadata
  } TargetComposition.empty).toOption

private def convergingRun : Option PlannerRun := do
  let target ← convergingTarget
  let query := { checkedQuery 0 (.verify (temporalProperty true true)) .exhaustive with
    target
    completeness := (CheckedQueryTarget.ofTarget target).completeness }
  let kernel ← (IncrementalPlannerKernel.ofCheckedQuery target.id query).toOption
  (plan query kernel).toOption

#guard convergingRun.map (fun run =>
    (run.result.metadata.validity.answer, run.result.metadata.explored.traces,
      run.result.metadata.validity.searchComplete, run.result.metadata.completeness.established)) ==
    some (.counterexample, 3, true, true)

#guard (do
    let target ← convergingTarget
    let run ← convergingRun
    let artifact ← run.artifact
    let selected : BehaviorTrace := {
      setup := artifact.plan.bindings
      trace := { initialState := artifact.plan.initialState, steps := [
        { selectedAction := requestValue, outcome := acceptedValue,
          state := completed, facts := [] }] } }
    let query ← (checkQuery (.ofTarget target) { queryDeclaration with
      form := .counterexample (temporalProperty true true)
      behavior := { behavior with traceExactly := some selected } }).toOption
    let kernel ← (IncrementalPlannerKernel.ofCheckedQuery target.id query).toOption
    let replayed ← (plan query kernel).toOption
    pure (replayed.artifact.map (·.plan.requestedActions),
      replayed.result.metadata.validity.answer)) == some (some [requestValue], .counterexample)

#guard (let query := { checkedQuery 0 (.select [temporalProperty true false]) .exhaustive with
    endpoint := .runtimePrefix }
  let analysis := analyzeCases query (incrementalKernel 0)
  (analysis.joint.status, analysis.propertyEvaluations.map (·.endpointAnswer))) ==
    (.limitReached, [.unresolved])

#guard (terminalTarget [[completed]]).map (·.behaviorFingerprint.render) ==
  some "sha256:c82cf457e851b60b999b413cab51d48f6cf6878c3244a8d25a3ddef5c36da1c0"
#guard (admittedQuery .deliberatelyClosed .allowVacuous).map (·.behaviorFingerprint.render) ==
  some "sha256:97c95ff6a534e221dc0c9bdad1f00555087cc07a320a954f0c9213a556a94f68"
#guard (endpointRun .deliberatelyClosed .requireAllTriggers
  (.verify (temporalProperty true false)) 10 2).map
    (·.result.metadata.validity.searchTermination) == some "limit-reached"

end Umpire.PlanningTests
