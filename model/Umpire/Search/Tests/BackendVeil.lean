import Umpire.Search.Backend.Veil
import Umpire.Search.Tests.Fixtures
import Umpire.Model.Tests.Parameterized

/-!
# The `veil` backend

Veil-only tests, in their own module so the rollback drill deletes them as a unit. They pin that
`Umpire.Search.Backend.Veil` returns each `BackendResult` ending on fixture products, that exhausting
the depth bound is `complete`, that its results finalize to the reference's outcome and witness on
the Search fixture's Queries wherever the reference terminates, that a `veil` run's receipt carries
the pinned commit, and the axiom inventories of the adapter's equivalence theorems (R7).
-/

namespace Umpire.SearchTests.BackendVeil

open Umpire
open Umpire.Search.Backend

private def ending (result : BackendResult) : String :=
  match result with
  | .violationFound trace _ => s!"violationFound {trace.trace.steps.length}"
  | .complete observations => s!"complete {observations.explored.traces}"
  | .stateBound visited _ => s!"stateBound {visited}"
  | .invalid _ _ => "invalid"

private def veilEnding (query : CheckedQuery LawStatement) (view : SearchView query.target) :
    String :=
  match Veil.backend? query view with
  | .error _ => "unsupported"
  | .ok result => ending result

/-! ### The endings

On the Search fixture (one request, then a completed state with no successor) `verify` visits both
states and completes, `find` stops at the one-step witness, and a search bound of zero or one stops
before the frontier empties. -/
#guard [veilEnding (fixtureQuery 0 (.verify property) .exhaustive 10) (incrementalKernel 0),
    veilEnding (fixtureQuery 0 (.find property) .exhaustive 10) (incrementalKernel 0),
    veilEnding (fixtureQuery 0 (.verify property) .exhaustive 0) (incrementalKernel 0),
    veilEnding (fixtureQuery 2 (.verify property) .exhaustive 1) (incrementalKernel 2)] ==
  ["complete 2", "violationFound 1", "stateBound 0", "stateBound 1"]

/-! The width-two fixture offers the request three times with equal results; exact deduplication
keeps one product state for all three, so `veil` visits two states where the reference counts four
candidates. -/
#guard veilEnding (fixtureQuery 2 (.verify property) .exhaustive 10) (incrementalKernel 2) ==
  "complete 2"

/-! The parameterized Model has a cycle, so every depth has a successor; an `atMost 3` bound on its
action makes the product grow until the count saturates. Each run ends `complete` at its depth
bound, and a deeper bound visits more states until the product stops growing: exhausting the depth
bound is `complete`, never `stateBound`. -/
private def parameterizedQuery (model : QueryModel (fun _ => True)) (depth : Nat) : Query := {
  id := .of "example.query.backend-veil", source := ModelTests.Parameterized.source
  target := model.id
  form := .verify SearchTests.property
  limits := Limits.bounded depth depth 100
  policy := .exhaustive
  behavior := { SearchTests.behavior with
    roles := [], requiredOccurrences := [], allowedActions := [], actionsExactly := none
    occurrenceBounds := [.atMost ModelTests.Parameterized.actionId 3] }
}

private def parameterized (depth : Nat) : Option String := do
  let model ← (ModelTests.Parameterized.target .samplesOnly).toOption
  let query ← (Query.check (.ofTarget model) (parameterizedQuery model depth)).toOption
  let view ← (SearchView.ofCheckedQuery query.target.id query).toOption
  pure (veilEnding query view)

#guard [1, 2, 3, 4].map parameterized ==
  [some "complete 3", some "complete 5", some "complete 7", some "complete 7"]

/-! ### Finalized against the reference

Every form at every budget, on the fixture with one and with three equal requests. The outcomes
agree wherever the reference terminates; the two differences are the width-two fixture at budget
two, where the reference's four candidates exceed the bound and `veil`'s two states do not. -/
private def finalized (query : CheckedQuery LawStatement) (view : SearchView query.target) :
    Option (PlanResult × PlanResult) := do
  let reference ← (search query view).toOption
  let backend ← (Veil.backend? query view).toOption
  let veil ← (finalizeBackendResult query view backend).toOption
  pure (reference, veil)

private def forms : List Query.Form :=
  [.verify property, .find property, .findViolation property, .pick [property]]

private def fixtureRuns : List (Option (PlanResult × PlanResult)) :=
  forms.flatMap fun form => [0, 1, 2, 10].flatMap fun budget => [0, 2].map fun width =>
    finalized (fixtureQuery width form .exhaustive budget) (incrementalKernel width)

#guard fixtureRuns.all Option.isSome

private def differingOutcomes : List (String × String) :=
  (fixtureRuns.reduceOption.map fun (reference, veil) =>
    (reference.result.outcome.name, veil.result.outcome.name)).filter fun (left, right) =>
      left != right

#guard differingOutcomes ==
  [("limit-reached", "verified-within-limits"), ("limit-reached", "none-found")]

/-! Where both find a trace, it is the same trace. -/
#guard fixtureRuns.reduceOption.all fun (reference, veil) =>
  match reference.result.outcome, veil.result.outcome with
  | .found referenceTrace _, .found veilTrace _ => referenceTrace == veilTrace
  | .found _ _, _ | _, .found _ _ => reference.result.outcome.name == "limit-reached"
  | _, _ => true

/-! ### Endpoints and trigger evidence

A bounded response clause triggered by the request: its trigger fires, it answers `unresolved`
under a partial ending when nothing responds, and a trace on which it never fires leaves the
coverage unexercised. The validity dimensions agree with the reference; the trigger evidence is one
witness for the clause, where the reference keeps one per admitted candidate. -/
private def temporalProperty (triggered responds : Bool) : CheckedProperty := {
  property with
  clauses := [.eventuallyWithin (id "planner.property.response")
    { field := .selectedAction, reference := request,
      constraint := .equals (if triggered then "request" else "absent") }
    { field := .observation, reference := observed,
      constraint := .equals (if responds then "accepted" else "absent") }
    { value := 1, unit := .steps }]
  access := { capabilities := [], logicalTimeSource := none, meanings := [
    { definitionId := request, kind := .action, behaviorVersion := "request" },
    { definitionId := observed, kind := .fact, behaviorVersion := "observed" }] }
}

private def validity (run : PlanResult) :=
  let validity := run.result.metadata.validity
  (run.result.outcome.name, validity.satisfiability, validity.coverage, validity.answer,
    validity.searchComplete, validity.requestedTriggers)

private def endpointRuns : List (Option (PlanResult × PlanResult)) :=
  [(Query.Ending.final, Query.Form.verify (temporalProperty false false)),
    (.final, .find (temporalProperty false false)),
    (.final, .find (temporalProperty true true)),
    (.final, .verify (temporalProperty true true)),
    (.«partial», .verify (temporalProperty true false)),
    (.final, .findViolation (temporalProperty true false))].map fun (ending, form) =>
    finalized { fixtureQuery 2 form .exhaustive 10 with ending, requireFiring := true }
      (incrementalKernel 2)

#guard endpointRuns.all fun runs =>
  runs.any fun (reference, veil) => validity reference == validity veil

#guard endpointRuns.filterMap (·.map fun (_, veil) => veil.result.outcome.name) ==
  ["never-triggered", "never-triggered", "found", "verified-within-limits", "still-pending",
    "found"]

#guard endpointRuns[3]!.map (fun (reference, veil) =>
    (reference.result.metadata.validity.triggers.length,
      veil.result.metadata.validity.triggers.map fun evidence =>
        (evidence.trace.trace.steps.length, evidence.trigger.clauseId,
          evidence.trigger.transitionPosition))) ==
  some (3, [(1, id "planner.property.response", 1)])

/-! ### Receipt

A `veil` run names its backend, the unit its bound counts and the pinned commit, which only the
`veil` backend carries. -/
#guard (fixtureRuns.head?.join).map (fun (reference, veil) =>
    (reference.instrumentation.searchBackend, veil.instrumentation.searchBackend,
      veil.instrumentation.searchUnit,
      (canonicalPlanningReceiptJson veil).contains
        s!"\"veilCommit\":\"{Veil.commit}\"",
      (canonicalPlanningReceiptJson veil).contains "\"searchBackend\":\"veil\"",
      (canonicalPlanningReceiptJson reference).contains "veilCommit")) ==
  some (.reference, .veil Veil.commit, .states, true, true, false)

#guard Veil.commit == "517f2badbf9a7ba2b18a72242351ff20943cbdd7"

/-! ### Axiom inventories of the adapter theorems (R7) -/

/-- info: 'Umpire.Search.Backend.Veil.transition_equivalence' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Search.Backend.Veil.transition_equivalence

/-- info: 'Umpire.Search.Backend.Veil.initial_equivalence' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Search.Backend.Veil.initial_equivalence

/-- info: 'Umpire.Search.Backend.Veil.assumptions_equivalence' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Search.Backend.Veil.assumptions_equivalence

end Umpire.SearchTests.BackendVeil
