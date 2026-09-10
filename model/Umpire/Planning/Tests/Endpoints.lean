import Umpire.Planning.Tests.Fixtures
import Umpire.Model.Table

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
    { definitionId := request, kind := .action, behaviorVersion := "request" },
    { definitionId := observed, kind := .fact, behaviorVersion := "observed" }] }
}

private def endpointRun
    (endpoint : QueryEndpoint) (exercise : QueryExercisePolicy)
    (form : QueryForm) (width : Nat := 0) (budget : Nat := 10) : Option PlannerRun :=
  (plan { checkedQuery width form .exhaustive budget with endpoint, exercise }
    (incrementalKernel width)).toOption

#guard (endpointRun .final .requireAllTriggers
  (.verify (temporalProperty false false))).map (·.result.outcome.name) ==
    some "nonempty-unexercised"

#guard (endpointRun .final .requireAllTriggers
  (.witness (temporalProperty false false))).map (·.result.outcome.name) ==
    some "nonempty-unexercised"

#guard (endpointRun .final .requireAllTriggers
  (.witness (temporalProperty true true))).map (·.result.metadata.validity.answer) == some .witness

#guard (endpointRun .final .requireAllTriggers
  (.verify (temporalProperty true true))).map (fun run =>
    (run.result.metadata.validity.satisfiability, run.result.metadata.validity.coverage,
      run.result.metadata.validity.answer, run.result.metadata.validity.searchComplete)) ==
    some (.nonempty, .exercised, .verified, true)

#guard (endpointRun .«partial» .requireAllTriggers
  (.verify (temporalProperty true false))).map (fun run =>
    (run.result.outcome.name, run.result.isVerified, run.result.metadata.validity.searchComplete)) ==
    some ("unresolved-prefix", false, true)

#guard (endpointRun .«partial» .requireAllTriggers
  (.verify (temporalProperty true false 0))).map (·.result.metadata.validity.answer) ==
    some .counterexample

#guard (endpointRun .final .requireAllTriggers
  (.verify (temporalProperty true false)) 10 2).map (fun run =>
    (run.result.metadata.validity.answer, run.result.metadata.validity.searchComplete,
      run.result.metadata.completeness.established,
      run.artifact.isSome, run.result.metadata.explored.traces)) ==
    some (.counterexample, false, false, true, 2)

#guard (endpointRun .final .requireAllTriggers
  (.verify (temporalProperty false false)) 10 2).map (fun run =>
    (run.result.outcome.name, run.result.metadata.validity.satisfiability,
      run.result.metadata.validity.coverage, run.result.metadata.validity.answer)) ==
    some ("limit-reached", .nonempty, .unknown, .unknown)

#guard (endpointRun .terminalModel .allowVacuous (.verify property)).map
  (·.result.metadata.validity.satisfiability) == some .impossible

#guard (endpointRun .final .requireAllTriggers
  (.verify (temporalProperty true true)) 2).map (fun run =>
    run.result.metadata.validity.triggers.map (fun evidence =>
      (evidence.trace.trace.steps.length, evidence.trigger.transitionPosition,
        evidence.trigger.occurrence.value))) ==
    some [(1, 1, some requestValue), (1, 1, some requestValue), (1, 1, some requestValue)]

#guard (endpointRun .final .requireAllTriggers
  (.verify (temporalProperty true true))).map (fun run =>
    (canonicalPlanningReceiptJson run.result).contains "checked-finite-enumeration/v1") == some true

#print axioms evaluatePropertyEndpoint_closed
#print axioms PlanningOutcome.constructorClassifiers_exactlyOne

private def terminalTarget (conditions : List (List ModelValue)) : Option (QueryModel (fun _ => True)) :=
  (checkModel (DraftModel.make { modelSpec 0 with terminalConditions := conditions }
    Providers.empty (.available (kernel 0) rfl (finitePlanning 0)))).toOption

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
    completeness := (CheckedQueryModel.ofTarget target).completeness
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

#guard (admittedQuery .final .allowVacuous).map (fun query =>
  query.canonicalMetadata.contains "endpointPolicy/v1") == some false
#guard (admittedQuery .«partial» .requireAllTriggers).map (fun query =>
  (query.endpoint, query.exercise, query.canonicalMetadata.contains "endpointPolicy/v1")) ==
    some (.«partial», .requireAllTriggers, true)
#guard (admittedQuery .«partial» .requireAllTriggers).map (·.behaviorFingerprint) !=
  (admittedQuery .terminalModel .requireAllTriggers).map (·.behaviorFingerprint)

private def convergingTarget : Option (QueryModel (fun _ => True)) :=
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
  (FiniteTable.checkTypedModel table {
    id := targetId
    source
    definitions := (modelSpec 0).definitions
    requiredCapabilities := []
    metadata := (kernel 0).metadata
  } Providers.empty).toOption

private def convergingRun : Option PlannerRun := do
  let target ← convergingTarget
  let query := { checkedQuery 0 (.verify (temporalProperty true true)) .exhaustive with
    target
    completeness := (CheckedQueryModel.ofTarget target).completeness }
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
    let selected : Scenario.Trace := {
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
    endpoint := .«partial» }
  let analysis := analyzeCases query (incrementalKernel 0)
  (analysis.joint.status, analysis.propertyEvaluations.map (·.endpointAnswer))) ==
    (.limitReached, [.unresolved])

#guard (terminalTarget [[completed]]).map (·.behaviorFingerprint.render) ==
  some "sha256:0359b9a0d4340f3d4c04b85b6ac0950c2a563076febbacffddfaac2e506f42b5"
#guard (admittedQuery .final .allowVacuous).map (·.behaviorFingerprint.render) ==
  some "sha256:be5535ef4d07080095b565f71a70233db180f95447d86414ae57330d27a9e43f"
#guard (endpointRun .final .requireAllTriggers
  (.verify (temporalProperty true false)) 10 2).map
    (·.result.metadata.validity.searchTermination) == some "limit-reached"

end Umpire.PlanningTests
