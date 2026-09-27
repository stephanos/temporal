import Umpire.Search.Product
import Umpire.Search.Tests.Fixtures
import Umpire.Examples.Switch
import Umpire.Model.Tests.Parameterized

/-!
# The Scenario progress automaton and the product state space

The R14 evidence for `Umpire.Search.Product.Scenario`: `ScenarioAutomaton.admits` equals
`CheckedScenario.admits` on every trace, compared exhaustively. First over every action sequence of
length at most four, from three setups, for a table of Scenarios that exercises every version-one
construct; then over every trace of the checked-in Switch Scenarios and the Search fixture's, and of
the parameterized Model, within their Limits and a little past them. `admitsPrefix` stays the
pruning oracle: every prefix it rejects, the automaton drops.

The product checks: its accepted paths, decoded, are exactly the admitted traces of the Model in
the reference key order; `ordering` and `adjacencies` over a free schedule are `Unsupported`; states
two paths reach alike are equal and hash alike; the monitor family's fired bits accumulate.
-/

namespace Umpire.SearchTests.Product

open Umpire
open Umpire.Search.Product

/-! ### A synthetic alphabet -/

private def actionA : DefinitionId := id "product.action.a"
private def actionB : DefinitionId := id "product.action.b"
private def actionC : DefinitionId := id "product.action.c"
private def stateValue : ModelValue := value phase "idle"
private def firstBinding : RoleBinding := { role, value := value phase "operation-a" }
private def secondBinding : RoleBinding := { role, value := value phase "operation-b" }

private def transitionOf (action : DefinitionId) (outcome : String) : Search.Product.Transition := {
  action := value action "call"
  result := { outcome := value accepted outcome, state := stateValue, facts := [] }
}

/-- The four transitions every synthetic trace draws from: `a` has two outcomes, so an exact trace
can differ from a candidate in its outcome alone. -/
private def alphabet : List Search.Product.Transition := [
  transitionOf actionA "one", transitionOf actionA "two",
  transitionOf actionB "one", transitionOf actionC "one"
]

private def sequencesUpTo : Nat → List (List Search.Product.Transition)
  | 0 => [[]]
  | length + 1 =>
      [] :: (alphabet.flatMap fun first => (sequencesUpTo length).map (first :: ·))

/-- A complete setup, an incomplete one, and one binding the role twice. -/
private def setups : List (List RoleBinding) := [[firstBinding], [], [firstBinding, secondBinding]]

private def traceOf (setup : List RoleBinding) (path : List Search.Product.Transition) :
    Scenario.Trace := {
  setup
  trace := { initialState := stateValue, steps := path.map (·.toTraceStep) }
}

private def syntheticTraces : List Scenario.Trace :=
  setups.flatMap fun setup => (sequencesUpTo 4).map (traceOf setup)

/-! ### The Scenario table -/

private def base : CheckedScenario := {
  SearchTests.behavior with
  requiredOccurrences := []
  allowedActions := []
  actionsExactly := none
}

private def occurrence (key : String) (action : DefinitionId) : Scenario.Step :=
  { id := id ("product.occurrence." ++ key), action }

private def exactTrace (outcome : String) : Scenario.Trace :=
  traceOf [firstBinding] [transitionOf actionA outcome, transitionOf actionB "one"]

private def scenarios : List CheckedScenario := [
  base,
  { base with allowedActions := [actionA, actionB] },
  { base with forbiddenActions := [actionC] },
  { base with occurrenceBounds := [{ action := actionA, minimum := 1, maximum := some 2 }] },
  { base with occurrenceBounds := [.atLeast actionB 2, .atMost actionC 0] },
  { base with occurrenceBounds := [.atLeast actionA 1, .atMost actionA 1] },
  { base with requiredOccurrences := [occurrence "first" actionA, occurrence "second" actionA] },
  { base with sequences := [[actionA, actionB], [actionB, actionA]] },
  { base with sequences := [[actionA, actionA, actionB]] },
  { base with
    allowedActions := [actionA, actionB]
    occurrenceBounds := [.atMost actionB 1]
    requiredOccurrences := [occurrence "first" actionA]
    sequences := [[actionA, actionB]] },
  { base with
    requiredOccurrences := [occurrence "first" actionA, occurrence "second" actionB,
      occurrence "third" actionA]
    ordering := [
      { before := (occurrence "first" actionA).id, after := (occurrence "second" actionB).id },
      { before := (occurrence "second" actionB).id, after := (occurrence "third" actionA).id }]
    actionsExactly := some [actionA, actionB, actionA] },
  { base with
    requiredOccurrences := [occurrence "first" actionA, occurrence "second" actionB]
    ordering := [
      { before := (occurrence "second" actionB).id, after := (occurrence "first" actionA).id }]
    actionsExactly := some [actionA, actionB] },
  { base with adjacencies := [[actionA, actionB]], actionsExactly := some [actionA, actionB] },
  { base with adjacencies := [[actionB, actionA]], actionsExactly := some [actionA, actionB] },
  { base with forbiddenActions := [actionC], actionsExactly := some [actionA, actionC] },
  { base with traceExactly := some (exactTrace "one") },
  { base with traceExactly := some (exactTrace "two"), actionsExactly := some [actionA, actionB] },
  { base with traceExactly := some (exactTrace "one"), actionsExactly := some [actionB, actionA] },
  { base with spaceStatus := .unsatisfiable },
  { base with setup := [.roleEquals (id "product.setup.first") role firstBinding.value] }
]

private def actionsOf (trace : Scenario.Trace) : List DefinitionId :=
  trace.trace.steps.map (·.selectedAction.definitionId)

/-- The automaton agrees with `admits` on every trace, and drops every prefix `admitsPrefix`
rejects. -/
private def agrees (behavior : CheckedScenario) (traces : List Scenario.Trace) : Bool :=
  match ScenarioAutomaton.lower behavior with
  | .error _ => false
  | .ok automaton =>
      traces.all fun trace =>
        automaton.admits trace == behavior.admits trace &&
          (behavior.admitsPrefix (actionsOf trace) || (automaton.run trace).isNone)

#guard syntheticTraces.length == 3 * 341

#guard scenarios.all fun behavior => agrees behavior syntheticTraces

/-! How many synthetic traces each Scenario admits, which keeps the comparison above from being
vacuous: only the five Scenarios built to admit nothing -- an impossible ordering or adjacency, a
forbidden exact action, contradictory exact schedules, an unsatisfiable space -- admit no trace. -/
#guard scenarios.map (fun behavior => syntheticTraces.countP behavior.admits) ==
  [341, 121, 121, 222, 41, 98, 212, 66, 48, 34, 4, 0, 2, 0, 0, 1, 1, 0, 0, 341]

/-! ### Constructs version one does not encode -/

private def lowerError (behavior : CheckedScenario) : Option Unsupported :=
  match ScenarioAutomaton.lower behavior with
  | .ok _ => none
  | .error unsupported => some unsupported

private def freeOrdering : CheckedScenario := {
  base with
  requiredOccurrences := [occurrence "first" actionA, occurrence "second" actionB]
  ordering := [
    { before := (occurrence "first" actionA).id, after := (occurrence "second" actionB).id }]
}

#guard lowerError freeOrdering ==
  some { scenarioId := base.id, source := base.source, construct := .ordering }
#guard lowerError { base with adjacencies := [[actionA, actionB]] } ==
  some { scenarioId := base.id, source := base.source, construct := .adjacencies }
#guard (lowerError { freeOrdering with adjacencies := [[actionA, actionB]] }).map
  (·.construct.name) == some "ordering"
#guard (lowerError { freeOrdering with actionsExactly := some [actionA, actionB] }).isNone

/-! ### Every trace of a Model, against the product's accepted paths -/

private def appendStep (trace : Scenario.Trace) (action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue) : Scenario.Trace :=
  { trace with trace := { trace.trace with
    steps := trace.trace.steps ++ [ModelTraceStep.result action result] } }

/-- Every trace of the view from `trace` within `depth` more steps, in the reference key order. -/
private def tracesFrom {target : QueryModel LawStatement} (view : SearchView target) :
    Nat → Scenario.Trace → ModelValue → List Scenario.Trace
  | 0, trace, _ => [trace]
  | depth + 1, trace, state =>
      trace :: (List.range view.actionLimit).flatMap fun actionIndex =>
        match view.actionAt actionIndex with
        | none => []
        | some action =>
            (List.range (view.stepLimit state action)).flatMap fun outcomeIndex =>
              match view.stepAt state action outcomeIndex with
              | none => []
              | some result => tracesFrom view depth (appendStep trace action result) result.state

private def allTraces {target : QueryModel LawStatement} (view : SearchView target)
    (setups : List (List RoleBinding)) (depth : Nat) : List Scenario.Trace :=
  setups.flatMap fun setup =>
    (List.range (view.initialLimit setup)).flatMap fun index =>
      match view.initialAt setup index with
      | none => []
      | some initialState =>
          tracesFrom view depth { setup, trace := { initialState, steps := [] } } initialState

/-- Every product path from `state` within `depth` more steps, reversed, in successor order. -/
private def pathsFrom {target : QueryModel LawStatement}
    (product : Search.Product.Product target Unit) :
    Nat → Search.Product.State Unit → List Search.Product.Transition →
      List (Search.Product.State Unit × List Search.Product.Transition)
  | 0, state, reversed => [(state, reversed)]
  | depth + 1, state, reversed =>
      (state, reversed) :: (product.successors state).flatMap fun (transition, next) =>
        pathsFrom product depth next (transition :: reversed)

/-- The product's accepted paths decode to exactly the Model's admitted traces, in order, and the
automaton agrees with `admits` on every trace, admitted or not. -/
private def productAgrees (query : CheckedQuery LawStatement) (view : SearchView query.target)
    (depth : Nat) : Bool :=
  match Search.Product.Product.build query view .none with
  | .error _ => false
  | .ok product =>
      let traces := allTraces view product.setups depth
      let accepted := product.initialStates.flatMap fun root =>
        (pathsFrom product depth root []).filterMap fun (state, reversed) =>
          if product.accepts state then some (decode root reversed.reverse) else none
      agrees query.behavior traces && accepted == traces.filter query.behavior.admits &&
        !accepted.isEmpty

/-- The query with its Scenario replaced; the view over its Model stays valid. -/
private def withBehavior (query : CheckedQuery LawStatement) (behavior : CheckedScenario) :
    CheckedQuery LawStatement :=
  { query with behavior }

/-! The Search fixture: one request with three equal results, the product's first dedup case. -/
#guard productAgrees (SearchTests.orderedQuery 2) (SearchTests.incrementalKernel 2) 1

section Switch

open Umpire.Examples.Switch

private def switchAgrees (query : CheckedQuery LawStatement) (depth : Nat) : Bool :=
  match SearchView.ofCheckedQuery query.target.id query with
  | .error _ => false
  | .ok view => productAgrees query view depth

/-! The Switch's three checked-in Scenarios, each over its own Query, within the Limits (one step)
and past them. -/
#guard [exploratoryQuery, exactActionQuery, exactTraceQuery].all fun query =>
  switchAgrees query (Nat.min query.limits.steps.value query.limits.actions.value) &&
    switchAgrees query 3

/-! The free-schedule constructs over the Switch's own flip. -/
#guard [
    { exploratoryBehavior with actionsExactly := none, ordering := [] },
    { exploratoryBehavior with
      actionsExactly := none, ordering := [], occurrenceBounds := [.atMost flipActionId 2] },
    { exploratoryBehavior with
      actionsExactly := none, ordering := [], occurrenceBounds := []
      sequences := [[flipActionId, flipActionId]] }
  ].all fun behavior => switchAgrees (withBehavior exploratoryQuery behavior) 3

end Switch

section Parameterized


private def parameterizedAction : DefinitionId := ModelTests.Parameterized.actionId

private def parameterizedQuery (model : QueryModel (fun _ => True)) : Query := {
  id := .of "example.query.product", source := ModelTests.Parameterized.source
  target := model.id
  form := .verify SearchTests.property
  limits := Limits.bounded 3 3 100
  policy := .exhaustive
  behavior := { base with roles := [] }
}

/-- The parameterized Model has two actions under one Definition ID and a cycle, so every depth
has traces; two different actions reach one state. -/
private def parameterizedAgrees (behavior : CheckedScenario) : Bool :=
  (do
    let model ← (ModelTests.Parameterized.target .samplesOnly).toOption
    let query ← (Query.check (.ofTarget model) (parameterizedQuery model)).toOption
    let view ← (SearchView.ofCheckedQuery query.target.id query).toOption
    pure (productAgrees (withBehavior query behavior) view 3)) == some true

#guard [
    { base with roles := [] },
    { base with roles := [], occurrenceBounds := [.atMost parameterizedAction 2] },
    { base with roles := [], sequences := [[parameterizedAction, parameterizedAction]] },
    { base with roles := [], allowedActions := [actionA] }
  ].all parameterizedAgrees

/-- Two actions from the idle root reach the same product state; the states are equal and hash
alike, while their transitions differ. A monitor family that counts steps keeps them apart only by
what it records, and ORs its fired bits along the path. -/
private def countingMonitors : Search.Product.MonitorFamily Nat where
  start _ _ := (0, 1)
  advance count _ _ := (count + 1, 2 ^ (count + 1))

#guard (do
    let model ← (ModelTests.Parameterized.target .samplesOnly).toOption
    let query ← (Query.check (.ofTarget model) (parameterizedQuery model)).toOption
    let view ← (SearchView.ofCheckedQuery query.target.id query).toOption
    let product ← (Search.Product.Product.build query view countingMonitors).toOption
    let root ← product.initialStates.head?
    let successors := product.successors root
    let accepted := successors.filter fun (transition, _) =>
      transition.result.outcome.value == "accepted"
    match accepted with
    | [(first, left), (second, right)] =>
        let grandchild := (product.successors left).head?.map (·.2.fired)
        pure (first != second, left == right, hash left == hash right, left.fired, grandchild)
    | _ => none) == some (true, true, true, 3, some 7)

end Parameterized

/-! ### Decoding -/

#guard
  let root : Search.Product.State Unit :=
    { setup := [firstBinding], model := stateValue, progress := {}, monitors := (), fired := 0 }
  let path := [transitionOf actionA "two", transitionOf actionC "one"]
  decode root path == traceOf [firstBinding] path &&
    (decode root path).trace.steps.map Search.Product.Transition.ofTraceStep == path

/-! ### Axiom inventories of the product theorems -/

/-- info: 'Umpire.Search.Product.decode_lossless' depends on axioms: [propext, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Search.Product.decode_lossless

/-- info: 'Umpire.Search.Product.Product.run_progress' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Search.Product.Product.run_progress

/-- info: 'Umpire.Search.Product.Product.accepts_iff_admits' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Search.Product.Product.accepts_iff_admits

/-- info: 'Umpire.Search.Product.Product.mem_successors' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Search.Product.Product.mem_successors

end Umpire.SearchTests.Product
