import Umpire.Search.Admission
import Umpire.Search.Tests.Fixtures
import Umpire.Examples.Switch

/-!
# Backend selection and the kernel replay gate

Selection sends a Query to `veil` or `reference` and records why, one test per reason value. Kernel
replay inside `finalizeBackendResult` accepts a backend's witness only when it is a trace the
Query's own search could report: a faulty backend returning any other trace yields `invalid` with
`unreplayableWitness`, the diagnostic and the trace in `offendingValue`. The module also pins the
public surface Selection and replay add and the line count of the frozen `Search.lean` (R12).
-/

namespace Umpire.SearchTests.Replay

open Umpire
open Umpire.Search.Selection

private def query (form : Query.Form) (strategy : SearchStrategy := .exhaustive)
    (selectedBehavior : CheckedScenario := behavior) : CheckedQuery (fun _ => True) :=
  fixtureQuery 0 form strategy (selectedBehavior := selectedBehavior)

private def view : SearchView (target 0) := incrementalKernel 0

/-! ### Selection -/

private def selected (query : CheckedQuery (fun _ => True)) (view : SearchView query.target) :
    String :=
  match select query view with
  | .veil _ => "veil"
  | .reference reason => "reference " ++ reason.name

/-- A clause bounded in logical time, which no version-one monitor encodes. -/
private def logicalTimeProperty : CheckedProperty := {
  property with
  clauses := [.eventuallyWithin (id "replay.property.response")
    { field := .selectedAction, reference := request, constraint := .equals "request" }
    { field := .observation, reference := observed, constraint := .equals "accepted" }
    { value := 1, unit := .logicalTime }]
  access := { capabilities := [], logicalTimeSource := none, meanings := [
    { definitionId := request, kind := .action, behaviorVersion := "request" },
    { definitionId := observed, kind := .fact, behaviorVersion := "observed" }] }
}

/-- An ordering over a free schedule, which the version-one progress automaton does not encode. -/
private def freeOrdering : CheckedScenario := {
  behavior with
  actionsExactly := none
  ordering := [{ before := occurrence, after := occurrence }]
}

/-! Every form on a supported Query goes to `veil`; the seeded strategy, a clause no monitor
encodes, and a free-schedule ordering each go to `reference` with their reason. No Query form is
unsupported in version one, so `unsupported-form` has no Query to test. -/
#guard [Query.Form.verify property, .find property, .findViolation property, .pick [property]].all
  fun form => selected (query form) view == "veil"

#guard [selected (query (.find property) .seeded) view,
    selected (query (.find logicalTimeProperty)) view,
    selected (query (.find property) (selectedBehavior := freeOrdering)) view] ==
  ["reference unsupported-strategy:seeded", "reference unsupported-clause:logical-time-limit",
    "reference unsupported-scenario:ordering"]

private def backendOf (run : Except KnownGapError PlanResult) : Option (String × String) :=
  run.toOption.map fun run =>
    (run.instrumentation.searchBackend.name, run.instrumentation.backendReason.name)

/-! `searchWith .veil` runs `veil` where Selection allows it and falls back to `reference`, with the
reason, where it does not; `searchWith .reference` is the reference search. -/
#guard [searchWith .veil (query (.find property)) view,
    searchWith .veil (query (.find property) .seeded) view,
    searchWith .reference (query (.find property)) view].map backendOf ==
  [some ("veil", "default"), some ("reference", "unsupported-strategy:seeded"),
    some ("reference", "default")]

#guard (searchWith .reference (query (.find property)) view).toOption ==
  (Umpire.search (query (.find property)) view).toOption

/-! Until the cutover, `search` runs every Query on `reference`: a Query Selection sends to `veil`
searches exactly as the reference does, and one it rules out records its reason. -/
#guard cutover == false

#guard (Search.Selection.search (query (.find property)) view).toOption ==
  (Umpire.search (query (.find property)) view).toOption

#guard backendOf (Search.Selection.search (query (.find logicalTimeProperty)) view) ==
  some ("reference", "unsupported-clause:logical-time-limit")

/-! An admitted Query selects the same way: the Switch's exact-action Query is supported, and
`searchWith` runs either backend on its own view. -/
#guard [Examples.Switch.exactActionAdmitted.search,
    Examples.Switch.exactActionAdmitted.searchWith .reference,
    Examples.Switch.exactActionAdmitted.searchWith .veil].map backendOf ==
  [some ("reference", "default"), some ("reference", "default"), some ("veil", "default")]

#guard (Examples.Switch.exactActionAdmitted.searchWith .veil).toOption.map
    (·.result.outcome.name) ==
  (Examples.Switch.exactActionAdmitted.search).toOption.map (·.result.outcome.name)

/-! ### Kernel replay -/

private def step (state : ModelValue := completed) (action : ModelValue := requestValue) :
    ModelTraceStep ModelValue ModelValue ModelValue ModelValue :=
  { selectedAction := action, outcome := acceptedValue, state, facts := [observedValue] }

private def witness (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
    (initialState : ModelValue := initial) (roles : List RoleBinding := setup) : Scenario.Trace :=
  { setup := roles, trace := { initialState, steps } }

/-- A faulty backend: it reports `trace` as the stopping witness whatever the Query. -/
private def faulty (trace : Scenario.Trace) : Backend (fun _ => True) := fun _ _ =>
  .violationFound trace {}

private def finalized (form : Query.Form) (result : BackendResult) : Option PlanResult :=
  (finalizeBackendResult (query form) view result).toOption

private def outcomeOf (form : Query.Form) (result : BackendResult) : String :=
  match (finalized form result).map (·.result.outcome) with
  | some (PlanningOutcome.invalid error) => error.kind.name ++ " " ++ error.offendingValue.takeWhile (· != ':')
  | some outcome => outcome.name
  | none => "known-gap"

private def findWith (trace : Scenario.Trace) : String :=
  outcomeOf (.find property) (faulty trace (query (.find property)) view)

/-! The one-step request is the fixture's `find` witness and replays. Every other trace the faulty
backend reports is `invalid` with `unreplayable-witness`, naming the first check it fails: the
Scenario does not admit the bare root, a step whose result or action the search view does not
offer, a trace past the depth bound, an initial state or a setup the search never starts from. -/
#guard [witness [step], witness [], witness [step initial],
    witness [step completed (value request "other")], witness [step, step],
    witness [step] completed, witness [step] initial []].map findWith ==
  ["found",
    "unreplayable-witness the trace is not an admitted endpoint",
    "unreplayable-witness step 1 is not a search-view step",
    "unreplayable-witness step 1 selects an action outside the search view",
    "unreplayable-witness the trace has 2 steps, past the depth bound 1",
    "unreplayable-witness the initial state is not a search-view initial state",
    "unreplayable-witness the setup is not one the search draws roots from"]

/-! The negative control in full: the diagnostic and the rendered trace are the offending value, and
the result is never `found`. -/
#guard (finalized (.find property) (faulty (witness [step initial]) (query (.find property)) view)).map
    (fun run => match run.result.outcome with
      | .invalid error => (error.kind, error.definitionId, error.offendingValue)
      | _ => (.emptyDefinitionId, id "", "")) ==
  some (.unreplayableWitness, id "planner.query.fixture",
    "step 1 is not a search-view step: setup [planner.role.operation:planner.state.phase=" ++
      "operation-a] initial planner.state.phase=initial -> planner.action.request=request / " ++
      "planner.outcome.accepted=accepted / planner.state.phase=initial " ++
      "[planner.observation.accepted=accepted]")

/-- A bounded response to the request that never comes: every admitted trace violates it. -/
private def unansweredProperty : CheckedProperty := {
  logicalTimeProperty with
  clauses := [.eventuallyWithin (id "replay.property.unanswered")
    { field := .selectedAction, reference := request, constraint := .equals "request" }
    { field := .observation, reference := observed, constraint := .equals "absent" }
    { value := 1, unit := .steps }]
}

/-! A member trace the Query does not stop at fails the form's decision: `find-violation` needs a
violation where the fixture Property has none, and a `verify` counterexample must violate. A real
counterexample replays and becomes the selected trace. -/
#guard [outcomeOf (.findViolation property) (.violationFound (witness [step]) {}),
    outcomeOf (.verify property) (.complete { counterexample := some (witness [step]) }),
    outcomeOf (.verify unansweredProperty)
      (.complete { nonempty := true, counterexample := some (witness [step]) })] ==
  ["unreplayable-witness the find-violation Query does not stop at it",
    "unreplayable-witness the verify Query finds no violation on it",
    "found"]

/-! A rejected witness stays `invalid` even where finalization would otherwise conclude from the
Query alone, as it does for a Scenario known unsatisfiable. -/
#guard (finalizeBackendResult
    (query (.find property) (selectedBehavior := { behavior with spaceStatus := .unsatisfiable }))
    view (.violationFound (witness [step]) {})).toOption.map (·.result.outcome.name) ==
  some "invalid"

/-! A rejected `verify` counterexample is dropped, so it can never become the selected trace. -/
#guard (finalized (.verify property)
    (.complete { nonempty := true, counterexample := some (witness [step]) })).map
    (·.artifact.isNone) == some true

/-! ### Surface (R12)

Selection and replay add these public names. Replay itself stays private to `Search.lean`. -/
#check isAdmittedEndpoint
#check EndpointDecision
#check endpointDecision
#check projectPlanRequest
#check Search.Selection.BackendName
#check Search.Selection.Selection
#check Search.Selection.select
#check Search.Selection.reasonOf
#check Search.Selection.Selection.run
#check Search.Selection.cutover
#check Search.Selection.search
#check Search.Selection.searchWith
#check AdmittedQuery.searchWith

/--
error: Unknown identifier `Umpire.replayFailure`
-/
#guard_msgs (error, substring := true) in
#check Umpire.replayFailure

/-! `Search.lean` is frozen: its line count is pinned, so any growth is a visible diff. -/
/-- info: 1428 -/
#guard_msgs in
#eval show Lean.Elab.Command.CommandElabM Unit from do
  let file : System.FilePath := ← Lean.getFileName
  let search := file.parent.bind (·.parent) |>.bind (·.parent) |>.map (· / "Search.lean")
  let some search := search | throwError "no Search.lean beside {file}"
  Lean.logInfo m!"{(← IO.FS.lines search).size}"

end Umpire.SearchTests.Replay
