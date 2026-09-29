import Umpire.Search.Product.MonitorProofs
import Umpire.Search.Tests.Product
import Umpire.Shared.Test

/-!
# Property monitors against the evaluator

The R6 evidence for `Umpire.Search.Product.Monitor`: for every clause kind version one lowers, each
monitor's answer equals `evaluatePropertyEndpoint` under both endings (closed and `partial`), and
its fired bit equals whether the evaluator realized the clause's trigger, on every trace compared.
First exhaustively over every trace of at most four steps drawn from a synthetic alphabet, from two
initial states, for a table of clauses covering every kind, every trace field, both counting units
and bounds zero to two; then over every trace within the Limits of the Switch, Search-fixture and
parameterized Models, for clauses generated from the values those traces carry; and through
`MonitoredProduct` over the Switch Queries' own Property. The answer counts per kind are pinned so
the comparison is not vacuous. `monitorsAgree` is public so the Temporal feature test modules can
run the same comparison over their own Models.
-/

namespace Umpire.SearchTests.Monitor

open Umpire
open Umpire.Search.Product
open Umpire.SearchTests.Product (allTraces)

/-! ### The comparison -/

/-- Run a Query's monitors along a whole trace: their states at its end and every fired bit. -/
def observe (monitors : QueryMonitors)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) : List ClauseState × Nat :=
  let (states, fired) := monitors.start trace.initialState
  let (states, fired, _) := trace.steps.foldl (fun (states, fired, prior) step =>
    let (next, bits) := monitors.advance states prior (Transition.ofTraceStep step)
    (next, fired ||| bits, step.state)) (states, fired, trace.initialState)
  (states, fired)

/-- The evaluator's answer per Property, by Definition ID, and its realized `(Property, clause)`
triggers, as the reference search computes them. -/
def evaluated (properties : List CheckedProperty) (stateFields : ModelValue → List ModelValue)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) (partialTrace : Bool) :
    Option (List PropertyEndpointAnswer × List (DefinitionId × DefinitionId)) := do
  let evaluations ← (CheckedProperty.sortedById properties).mapM fun property =>
    (checkPropertyEvaluationInput property trace stateFields).toOption.map fun input =>
      (property.id, evaluatePropertyEndpoint property input partialTrace)
  pure (evaluations.map (·.2.answer),
    (evaluations.flatMap fun (propertyId, evaluation) =>
      evaluation.realizedTriggers.map fun trigger => (propertyId, trigger.clauseId)).eraseDups)

private def sameSet (left right : List (DefinitionId × DefinitionId)) : Bool :=
  left.all right.contains && right.all left.contains

/-- The monitors of `properties` agree with the evaluator on `trace` under both endings. -/
def agreesOn (monitors : QueryMonitors) (properties : List CheckedProperty)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) : Bool :=
  let (states, fired) := observe monitors trace
  [false, true].all fun partialTrace =>
    match evaluated properties monitors.stateFields trace partialTrace with
    | none => false
    | some (answers, realized) =>
        answers == monitors.answers states partialTrace &&
          sameSet realized (monitors.firedClauses fired)

/-- The monitors of `properties` lower, and agree with the evaluator on every trace. -/
def monitorsAgree (properties : List CheckedProperty) (stateFields : ModelValue → List ModelValue)
    (traces : List (ModelTrace ModelValue ModelValue ModelValue ModelValue)) : Bool :=
  match QueryMonitors.lower properties stateFields with
  | .error _ => false
  | .ok monitors => traces.all (agreesOn monitors properties)

/-! ### A synthetic alphabet -/

private def stateId : DefinitionId := id "monitor.state"
private def loadId : DefinitionId := id "monitor.field.load"
private def actionA : DefinitionId := id "monitor.action.a"
private def actionB : DefinitionId := id "monitor.action.b"
private def outcomeId : DefinitionId := id "monitor.outcome"
private def factId : DefinitionId := id "monitor.fact"
private def hiddenId : DefinitionId := id "monitor.hidden"

private def idle : ModelValue := value stateId "idle"
private def busy : ModelValue := value stateId "busy"

/-- Each state holds a load field and a field no Property may read. -/
private def stateFields (state : ModelValue) : List ModelValue :=
  if state == busy then [value loadId "2", value hiddenId "h"]
  else if state == idle then [value loadId "0", value hiddenId "h"]
  else []

private def stepOf (action : DefinitionId) (outcome : String) (state : ModelValue)
    (facts : List ModelValue) : ModelTraceStep ModelValue ModelValue ModelValue ModelValue :=
  { selectedAction := value action "call", outcome := value outcomeId outcome, state, facts }

private def alphabet : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue) := [
  stepOf actionA "ok" busy [value factId "ping"],
  stepOf actionA "fail" idle [],
  stepOf actionB "ok" idle [value factId "pong", value hiddenId "x"],
  stepOf actionB "fail" busy [value factId "ping", value factId "pong"]
]

private def sequencesUpTo : Nat →
    List (List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
  | 0 => [[]]
  | length + 1 => [] :: (alphabet.flatMap fun first => (sequencesUpTo length).map (first :: ·))

private def syntheticTraces : List (ModelTrace ModelValue ModelValue ModelValue ModelValue) :=
  [idle, busy].flatMap fun initialState =>
    (sequencesUpTo 4).map fun steps => { initialState, steps }

#guard syntheticTraces.length == 2 * 341

/-! ### The clause table -/

private def meaning (definitionId : DefinitionId) (kind : DefinitionKind) : Meaning :=
  { definitionId, kind, behaviorVersion := "monitor" }

private def fullAccess : PropertyCapabilityView := {
  capabilities := [], logicalTimeSource := none
  meanings := [meaning stateId .state, meaning loadId .state, meaning actionA .action,
    meaning actionB .action, meaning outcomeId .outcome, meaning factId .fact] }

/-- A view that admits no state: its fields go with it. -/
private def statelessAccess : PropertyCapabilityView :=
  { fullAccess with meanings := fullAccess.meanings.filter (·.definitionId != stateId) }

private def pattern (field : PropertyTraceField) (reference : DefinitionId)
    (constraint : ValueConstraint := .present) : PropertyPattern :=
  { field, reference, constraint }

private def isIdle (field : PropertyTraceField) : PropertyPattern :=
  pattern field stateId (.equals "idle")
private def isBusy (field : PropertyTraceField) : PropertyPattern :=
  pattern field stateId (.equals "busy")

private def propertyOf (key : String) (clauses : List CheckedPropertyClause)
    (access : PropertyCapabilityView := fullAccess) : CheckedProperty :=
  { SearchTests.property with id := id ("monitor.property." ++ key), clauses, access }

private def clauseId (key : String) : DefinitionId := id ("monitor.clause." ++ key)

/-- Pairs of patterns the one-step and positional kinds relate, chosen so positions coincide,
lag and lead: a prior-state pattern sits one position back in `steps`. -/
private def pairs : List (PropertyPattern × PropertyPattern) := [
  (pattern .selectedAction actionA, pattern .observation factId (.equals "pong")),
  (isBusy .state, isIdle .priorState),
  (isBusy .priorState, isBusy .resultingState),
  (isIdle .resultingState, isIdle .priorState),
  (pattern .outcome outcomeId (.equals "fail"), isBusy .state),
  (pattern .priorState loadId (.naturalAtLeast 1), pattern .relation factId (.equals "ping"))
]

private def indexed {α : Type} (items : List α) : List (α × String) :=
  items.zipIdx.map fun (item, index) => (item, toString index)

private def stateInvariants : List CheckedProperty :=
  [(isIdle .state, fullAccess), (pattern .state loadId (.naturalAtMost 1), fullAccess),
    (pattern .state actionA, fullAccess), (pattern .state stateId, statelessAccess),
    (pattern .state loadId (.naturalAtMost 2), fullAccess)] |> indexed |>.map
    fun ((state, access), key) =>
      propertyOf ("invariant." ++ key) [.stateInvariant (clauseId ("invariant." ++ key)) state]
        access

private def contracts : List CheckedProperty :=
  (indexed pairs).flatMap fun ((trigger, response), key) => [
    propertyOf ("contract." ++ key)
      [.transitionContract (clauseId ("contract." ++ key)) trigger response],
    propertyOf ("io." ++ key) [.inputOutput (clauseId ("io." ++ key)) trigger response]]

private def relations : List CheckedProperty :=
  [pattern .observation factId (.equals "ping"), isBusy .state, isBusy .resultingState,
    pattern .priorState loadId (.equals "2"), pattern .outcome outcomeId (.equals "fail"),
    pattern .relation factId (.equals "pong"), pattern .selectedAction actionB,
    pattern .state loadId (.equals "0")] |> indexed |>.map fun (relation, key) =>
      propertyOf ("relation." ++ key) [.identityRelation (clauseId ("relation." ++ key)) relation]

private def units : List LimitUnit := [.steps, .actions]

private def ordereds : List CheckedProperty :=
  (indexed pairs).flatMap fun ((before, after), key) => units.flatMap fun unit =>
    let key := "ordered." ++ key ++ "." ++ unit.name
    [propertyOf key [.ordered (clauseId key) before after unit],
      propertyOf (key ++ ".reversed") [.ordered (clauseId (key ++ ".reversed")) after before unit]]

private def bounded (kind : String)
    (clause :
      DefinitionId → PropertyPattern → PropertyPattern → Limit → CheckedPropertyClause) :
    List CheckedProperty :=
  (indexed pairs).flatMap fun ((trigger, response), key) => units.flatMap fun unit =>
    [0, 1, 2].map fun bound =>
      let key := s!"{kind}.{key}.{unit.name}.{bound}"
      propertyOf key [clause (clauseId key) trigger response { value := bound, unit }]

private def eventuallies : List CheckedProperty := bounded "eventually" .eventuallyWithin
private def nevers : List CheckedProperty := bounded "never" .neverWithin

/-! ### Every kind, against the evaluator -/

#guard monitorsAgree stateInvariants stateFields syntheticTraces
#guard monitorsAgree contracts stateFields syntheticTraces
#guard monitorsAgree relations stateFields syntheticTraces
#guard monitorsAgree ordereds stateFields syntheticTraces
#guard monitorsAgree eventuallies stateFields syntheticTraces
#guard monitorsAgree nevers stateFields syntheticTraces

/-- One Query's Properties together: out of Definition ID order, several clauses to a Property, and
a Property with none, so the per-Property combination and the fired-bit indices are compared
too. -/
private def mixed : List CheckedProperty := [
  propertyOf "mixed.z" [
    .eventuallyWithin (clauseId "mixed.z.eventually") (isBusy .state) (isIdle .priorState)
      { value := 1, unit := .steps },
    .stateInvariant (clauseId "mixed.z.invariant") (pattern .state loadId (.naturalAtMost 1))],
  propertyOf "mixed.a" [
    .transitionContract (clauseId "mixed.a.contract") (pattern .selectedAction actionA)
      (pattern .outcome outcomeId (.equals "ok")),
    .neverWithin (clauseId "mixed.a.never") (pattern .selectedAction actionB)
      (pattern .observation factId (.equals "ping")) { value := 1, unit := .actions },
    .ordered (clauseId "mixed.a.ordered") (isIdle .resultingState) (isBusy .state) .steps],
  propertyOf "mixed.m" []
]

#guard monitorsAgree mixed stateFields syntheticTraces

/-! How often each kind answers what, closed and then partial, over the synthetic traces: every
kind reaches `satisfied` and `violated`, and every kind but the one-step contracts `unresolved`,
so the comparison above is not vacuous. -/
private def tally (properties : List CheckedProperty) (partialTrace : Bool) : Nat × Nat × Nat :=
  let answers := properties.flatMap fun property => syntheticTraces.filterMap fun trace =>
    (evaluated [property] stateFields trace partialTrace).bind (·.1.head?)
  (answers.count .satisfied, answers.count .violated, answers.count .unresolved)

#guard [stateInvariants, contracts, relations, ordereds, eventuallies, nevers].map
    (fun properties => (tally properties false, tally properties true)) == [
  ((744, 2666, 0), (744, 1302, 1364)),
  ((2168, 6016, 0), (2168, 6016, 0)),
  ((5022, 434, 0), (5022, 0, 434)),
  ((10764, 5604, 0), (10764, 0, 5604)),
  ((11086, 13466, 0), (11086, 9618, 3848)),
  ((8172, 16380, 0), (6184, 16380, 1988))]

/-! ### Unsupported clauses -/

private def guard : CheckedPropertyPredicate .before :=
  ⟨id "monitor.guard", SearchTests.source, .all [], fullAccess⟩

private def temporal (forbidden : Bool) : CheckedPropertyTemporalClause := {
  id := clauseId "guarded", source := SearchTests.source, parentId := clauseId "guarded.parent"
  caseId := none, guard, exception := none, forbidden
  trigger := pattern .selectedAction actionA, response := isIdle .state
  limit := { value := 1, unit := .steps } }

private def lowerError (property : CheckedProperty) : Option MonitorUnsupported :=
  match QueryMonitors.lower [property] stateFields with
  | .ok _ => none
  | .error unsupported => some unsupported

private def unsupportedKind (clause : CheckedPropertyClause) : Option String :=
  (lowerError (propertyOf "unsupported" [clause])).map (·.kind.name)

#guard [
    .branches { id := clauseId "branches", source := SearchTests.source, guard, exception := none
                cases := [], complete := false, exclusive := false },
    .guardedEventuallyWithin (temporal false), .guardedNeverWithin (temporal true),
    .eventuallyWithin (clauseId "time") (isBusy .state) (isIdle .state)
      { value := 1, unit := .logicalTime },
    .neverWithin (clauseId "time") (isBusy .state) (isIdle .state)
      { value := 1, unit := .logicalTime },
    .ordered (clauseId "time") (isBusy .state) (isIdle .state) .logicalTime,
    .ordered (clauseId "search") (isBusy .state) (isIdle .state) .search
  ].map unsupportedKind ==
    [some "branches", some "guardedEventuallyWithin", some "guardedNeverWithin",
      some "logical-time-limit", some "logical-time-limit", some "logical-time-limit",
      some "search-limit"]

/-! The error names the Property and the clause, and the first unsupported clause is reported. -/
#guard lowerError (propertyOf "unsupported" [
    .stateInvariant (clauseId "fine") (isIdle .state),
    .ordered (clauseId "time") (isBusy .state) (isIdle .state) .logicalTime,
    .guardedNeverWithin (temporal true)]) ==
  some {
    propertyId := id "monitor.property.unsupported"
    clauseId := clauseId "time"
    source := SearchTests.source
    kind := .limitUnit .logicalTime }

/-! Every supported kind lowers to its own monitor and records `testing`. -/
#guard (stateInvariants ++ contracts ++ relations ++ ordereds ++ eventuallies ++ nevers).all
  fun property => (lowerError property).isNone
#guard MonitorKind.all.map (fun kind => (kind.name, kind.trustBasis.name)) == [
  ("stateInvariant", "testing"), ("transitionContract", "testing"),
  ("identityRelation", "testing"), ("inputOutput", "testing"), ("ordered", "testing"),
  ("eventuallyWithin", "testing"), ("neverWithin", "testing")]

/-! ### Every trace of the checked-in Models -/

/-- A clause table built from the values a Model's traces carry: exact patterns on every field
that can read them, paired with the next pattern, under every kind and both units, bounds zero and
one. The Property may read every definition the traces carry. -/
def generatedProperties (stateFields : ModelValue → List ModelValue)
    (traces : List (ModelTrace ModelValue ModelValue ModelValue ModelValue)) :
    List CheckedProperty :=
  let states := (traces.flatMap fun trace =>
    trace.initialState :: trace.steps.map (·.state)).eraseDups
  let stateValues := (states ++ states.flatMap stateFields).eraseDups
  let actions := (traces.flatMap fun trace => trace.steps.map (·.selectedAction)).eraseDups
  let outcomes := (traces.flatMap fun trace => trace.steps.map (·.outcome)).eraseDups
  let facts := (traces.flatMap fun trace => trace.steps.flatMap (·.facts)).eraseDups
  let exact (field : PropertyTraceField) (value : ModelValue) : PropertyPattern :=
    pattern field value.definitionId (.equals value.value)
  let patterns :=
    ([PropertyTraceField.state, .priorState, .resultingState].flatMap fun field =>
      stateValues.map (exact field)) ++
    actions.map (exact .selectedAction) ++ outcomes.map (exact .outcome) ++
    ([PropertyTraceField.observation, .relation].flatMap fun field => facts.map (exact field))
  let access : PropertyCapabilityView := {
    capabilities := [], logicalTimeSource := none
    meanings := (stateValues ++ actions ++ outcomes ++ facts).map fun value =>
      meaning value.definitionId .state }
  let related := patterns.zip (patterns.drop 1 ++ patterns.take 1)
  let property (key : String) (clause : DefinitionId → CheckedPropertyClause) : CheckedProperty :=
    propertyOf ("generated." ++ key) [clause (clauseId ("generated." ++ key))] access
  (indexed patterns).flatMap (fun (single, key) => [
    property ("invariant." ++ key) (.stateInvariant · single),
    property ("relation." ++ key) (.identityRelation · single)]) ++
  (indexed related).flatMap fun ((first, second), key) =>
    [property ("contract." ++ key) (.transitionContract · first second),
      property ("io." ++ key) (.inputOutput · first second)] ++
    units.flatMap fun unit =>
      property s!"ordered.{key}.{unit.name}" (.ordered · first second unit) ::
        [0, 1].flatMap fun bound =>
          let limit : Limit := { value := bound, unit }
          [property s!"eventually.{key}.{unit.name}.{bound}"
              (.eventuallyWithin · first second limit),
            property s!"never.{key}.{unit.name}.{bound}" (.neverWithin · first second limit)]

/-- Every trace of a Model's view within `depth`, against a generated clause table and the Query's
own Properties. -/
def modelAgrees (query : CheckedQuery LawStatement) (view : SearchView query.target)
    (depth : Nat) : Bool :=
  let setups := match query.completeness with
    | some evidence => evidence.roleAssignments
    | none => query.target.resolvedSetups
  let traces := (allTraces view setups depth).map (·.trace)
  let stateFields := query.target.stateFields
  !traces.isEmpty && monitorsAgree query.form.properties stateFields traces &&
    (generatedProperties stateFields traces).all fun property =>
      monitorsAgree [property] stateFields traces

/-! The Search fixture: one request with three equal results. -/
#guard modelAgrees (SearchTests.orderedQuery 2) (SearchTests.incrementalKernel 2) 1

section Switch

open Umpire.Examples.Switch

private def switchView (query : CheckedQuery LawStatement) : Option (SearchView query.target) :=
  (SearchView.ofCheckedQuery query.target.id query).toOption

/-! The Switch within its Limits (one step) and past them; its state holds a `power` field. -/
#guard [exploratoryQuery, exactActionQuery, exactTraceQuery].all fun query =>
  (switchView query).any fun view =>
    modelAgrees query view (Nat.min query.limits.steps.value query.limits.actions.value) &&
      modelAgrees query view 3

/-- Every product path of a Query's `MonitoredProduct` within `depth`: the answers read off its
last state and its fired bits are the evaluator's on the trace it decodes to. -/
def productAgrees {LawStatement : Law → Prop} (query : CheckedQuery LawStatement)
    (view : SearchView query.target)
    (depth : Nat) : Bool :=
  match MonitoredProduct.build query view with
  | .error _ => false
  | .ok monitored =>
      let product := monitored.product
      let rec paths : Nat → State (List ClauseState) → List Transition →
          List (State (List ClauseState) × List Transition)
        | 0, state, reversed => [(state, reversed)]
        | depth + 1, state, reversed =>
            (state, reversed) :: (product.successors state).flatMap fun (transition, next) =>
              paths depth next (transition :: reversed)
      let checked := product.initialStates.flatMap fun root =>
        (paths depth root []).map fun (state, reversed) =>
          let trace := (decode root reversed.reverse).trace
          match evaluated query.form.properties query.target.stateFields trace
              monitored.partialTrace with
          | none => false
          | some (answers, realized) =>
              answers == monitored.answers state &&
                sameSet realized (monitored.monitors.firedClauses state.fired)
      !checked.isEmpty && checked.all (·)

#guard [exploratoryQuery, exactActionQuery, exactTraceQuery].all fun query =>
  (switchView query).any fun view => productAgrees query view 3 &&
    productAgrees { query with ending := .«partial» } view 3

/-! The Switch's Property is a one-step contract; its lowered family has one clause. -/
#guard (QueryMonitors.lower [flipProperty] Examples.Switch.target.stateFields).toOption.map
  (·.clauses.map (·.monitor.kind.name)) == some ["transitionContract"]

/-! A Query with an unsupported clause has no product, and the error names the clause. -/
#guard (switchView exactActionQuery).any fun view =>
  match MonitoredProduct.build
      { exactActionQuery with form := .find { flipProperty with
        clauses := [.guardedNeverWithin (temporal true)] } } view with
  | .error (.clause unsupported) => unsupported.kind == .guardedNeverWithin
  | _ => false

end Switch

section Parameterized

private def parameterizedQuery (model : QueryModel (fun _ => True)) : Query := {
  id := .of "example.query.monitor", source := ModelTests.Parameterized.source
  target := model.id
  form := .verify SearchTests.property
  limits := Limits.bounded 3 3 100
  policy := .exhaustive
  behavior := { SearchTests.behavior with
    roles := [], requiredOccurrences := [], allowedActions := [], actionsExactly := none }
}

/-! The parameterized Model: two actions under one Definition ID, and a cycle. -/
#guard (do
    let model ← (ModelTests.Parameterized.target .samplesOnly).toOption
    let query ← (Query.check (.ofTarget model) (parameterizedQuery model)).toOption
    let view ← (SearchView.ofCheckedQuery query.target.id query).toOption
    pure (modelAgrees query view 3)) == some true

end Parameterized

/-! ### Axiom inventories of the monitor theorems -/

assert_axioms [Umpire.Search.Product.ClauseState.answer_closed_ne_unresolved,
  Umpire.Search.Product.Monitor.start_within, Umpire.Search.Product.Monitor.advance_within,
  Umpire.Search.Product.Monitor.eventuallyStep_within, Umpire.Search.Product.Monitor.neverStep_within,
  Umpire.Search.Product.Monitor.advance_fired, Umpire.Search.Product.Monitor.never_fires]
  allowing [propext, Classical.choice, Quot.sound]

end Umpire.SearchTests.Monitor
