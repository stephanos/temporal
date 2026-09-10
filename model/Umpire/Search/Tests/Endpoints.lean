import Umpire.Search.Tests.Fixtures
import Umpire.Model.Table

/-! Endpoint interpretation and exact candidate-budget boundaries. -/

namespace Umpire.SearchTests

#guard (run 0 (.verify property) .exhaustive 2).toOption.map
  (fun run => run.result.outcome.name) == some "verified-within-limits"

private def temporalProperty (triggered responds : Bool) (bound : Nat := 1) : CheckedProperty := {
  property with
  clauses := [.eventuallyWithin (id "planner.property.response")
    { field := .selectedAction, reference := request,
      constraint := .equals (if triggered then "request" else "absent") }
    { field := .observation, reference := observed,
      constraint := .equals (if responds then "accepted" else "absent") }
    { value := bound, unit := .steps }]
  access := { capabilities := [], logicalTimeSource := none, meanings := [
    { definitionId := request, kind := .action, behaviorVersion := "request" },
    { definitionId := observed, kind := .fact, behaviorVersion := "observed" }] }
}

private def endpointRun
    (ending : Query.Ending) (requireFiring : Bool)
    (form : Query.Form) (width : Nat := 0) (budget : Nat := 10) : Option PlanResult :=
  (search { fixtureQuery width form .exhaustive budget with ending, requireFiring }
    (incrementalKernel width)).toOption

#guard (endpointRun .final true
  (.verify (temporalProperty false false))).map (·.result.outcome.name) ==
    some "never-triggered"

#guard (endpointRun .final true
  (.find (temporalProperty false false))).map (·.result.outcome.name) ==
    some "never-triggered"

#guard (endpointRun .final true
  (.find (temporalProperty true true))).map (·.result.metadata.validity.answer) == some .witness

#guard (endpointRun .final true
  (.verify (temporalProperty true true))).map (fun run =>
    (run.result.metadata.validity.satisfiability, run.result.metadata.validity.coverage,
      run.result.metadata.validity.answer, run.result.metadata.validity.searchComplete)) ==
    some (.nonempty, .exercised, .verified, true)

#guard (endpointRun .«partial» true
  (.verify (temporalProperty true false))).map (fun run =>
    (run.result.outcome.name, run.result.isVerified, run.result.metadata.validity.searchComplete)) ==
    some ("still-pending", false, true)

#guard (endpointRun .«partial» true
  (.verify (temporalProperty true false 0))).map (·.result.metadata.validity.answer) ==
    some .counterexample

#guard (endpointRun .final true
  (.verify (temporalProperty true false)) 10 2).map (fun run =>
    (run.result.metadata.validity.answer, run.result.metadata.validity.searchComplete,
      run.result.metadata.completeness.established,
      run.artifact.isSome, run.result.metadata.explored.traces)) ==
    some (.counterexample, false, false, true, 2)

#guard (endpointRun .final true
  (.verify (temporalProperty false false)) 10 2).map (fun run =>
    (run.result.outcome.name, run.result.metadata.validity.satisfiability,
      run.result.metadata.validity.coverage, run.result.metadata.validity.answer)) ==
    some ("limit-reached", .nonempty, .unknown, .unknown)

#guard (endpointRun .terminal false (.verify property)).map
  (·.result.metadata.validity.satisfiability) == some .impossible

#guard (endpointRun .final true
  (.verify (temporalProperty true true)) 2).map (fun run =>
    run.result.metadata.validity.triggers.map (fun evidence =>
      (evidence.trace.trace.steps.length, evidence.trigger.transitionPosition,
        evidence.trigger.occurrence.value))) ==
    some [(1, 1, some requestValue), (1, 1, some requestValue), (1, 1, some requestValue)]

#guard (endpointRun .final true
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

private def terminalRun : Option PlanResult := do
  let target ← terminalTarget [[completed], [completed, initial]]
  let query := { fixtureQuery 0 (.verify property) .exhaustive with
    target
    completeness := (ModelCompleteness.ofTarget target).completeness
    ending := .terminal }
  let kernel ← (SearchView.ofCheckedQuery target.id query).toOption
  (search query kernel).toOption

#guard terminalRun.map (·.result.outcome.name) == some "verified-within-limits"

private def authoredQuery : Query := {
  id := id "planner.query.endpoints"
  source
  target := targetId
  form := .verify property
  behavior
  limits := limits
  policy := .exhaustive
}

private def admittedQuery (ending : Query.Ending) (requireFiring : Bool) :
    Option (CheckedQuery (fun _ => True)) :=
  (Query.check (.ofTarget baseTarget) { authoredQuery with ending, requireFiring }).toOption

#guard (admittedQuery .final false).map (fun query =>
  query.canonicalMetadata.contains "endingPolicy/v1") == some false
#guard (admittedQuery .«partial» true).map (fun query =>
  (query.ending, query.requireFiring, query.canonicalMetadata.contains "endingPolicy/v1")) ==
    some (.«partial», true, true)
#guard (admittedQuery .«partial» true).map (·.behaviorFingerprint) !=
  (admittedQuery .terminal true).map (·.behaviorFingerprint)

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

private def convergingRun : Option PlanResult := do
  let target ← convergingTarget
  let query := { fixtureQuery 0 (.verify (temporalProperty true true)) .exhaustive with
    target
    completeness := (ModelCompleteness.ofTarget target).completeness }
  let kernel ← (SearchView.ofCheckedQuery target.id query).toOption
  (search query kernel).toOption

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
    let query ← (Query.check (.ofTarget target) { authoredQuery with
      form := .findViolation (temporalProperty true true)
      behavior := { behavior with traceExactly := some selected } }).toOption
    let kernel ← (SearchView.ofCheckedQuery target.id query).toOption
    let replayed ← (search query kernel).toOption
    pure (replayed.artifact.map (·.plan.requestedActions),
      replayed.result.metadata.validity.answer)) == some (some [requestValue], .counterexample)

#guard (let query := { fixtureQuery 0 (.pick [temporalProperty true false]) .exhaustive with
    ending := .«partial» }
  let analysis := analyzeBranches query (incrementalKernel 0)
  (analysis.joint.status, analysis.propertyEvaluations.map (·.endpointAnswer))) ==
    (.limitReached, [.unresolved])

#guard (terminalTarget [[completed]]).map (·.behaviorFingerprint.render) ==
  some "sha256:0359b9a0d4340f3d4c04b85b6ac0950c2a563076febbacffddfaac2e506f42b5"
#guard (admittedQuery .final false).map (·.behaviorFingerprint.render) ==
  some "sha256:78423dad6c6efb84c4943fb87a2ff16a127ad4c2fb7b105638f068a7954de242"
#guard (endpointRun .final true
  (.verify (temporalProperty true false)) 10 2).map
    (·.result.metadata.validity.searchTermination) == some "limit-reached"

end Umpire.SearchTests
