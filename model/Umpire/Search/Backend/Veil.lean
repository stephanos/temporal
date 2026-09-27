import Umpire.Search.Product
import Veil.Core.Tools.ModelChecker.Concrete.Checker

/-!
# The `veil` search backend

Veil's concrete breadth-first checker searches a Query's product state space
(`Umpire.Search.Product`) and reports a `BackendResult`, which `Umpire.Search.finalizeBackendResult`
turns into the Query's `PlanResult` as it does the reference backend's. This is the only module
that imports `Veil.*`; `Umpire.Search.Selection` is the only module that imports it
(`search-backend-isolation`, `ModelLint.ImportGraph`).

## The transition system

`system` is the product as a Veil `EnumerableTransitionSystem` over `Located` states, a product
state with the depth it was reached at. Roots come in the product's order and successors in
`Product.successors` order, which is the reference search's key order. The depth bound
(`maximumDepth`, the smaller of the `steps` and `actions` Limits) is a Veil state constraint, so a
state past it is never enumerated. `transition_equivalence`, `initial_equivalence` and
`assumptions_equivalence` prove that Veil's relational reading of `system` is the product's own
relation (`relation`), in both directions.

## Running the checker

Veil's entry point, `findReachable`, runs in `IO` with an unfueled loop. The backend is a pure
total function, so it drives Veil's pure breadth-first step, `SequentialSearchContext.bfsStep`,
itself: once per product state, at most `Limits.search` times. The frontier is Veil's FIFO queue,
single-threaded, with first-discovery parents, so the first state found at the smallest depth is
the one whose path is least in the reference key order.

Deduplication is exact: the fingerprint of a `Located` state is its product `State`, not a 64-bit
hash, so two distinct product states are never merged. It ignores the depth, which is sound because
a breadth-first frontier reaches every state first at its minimal depth.

## What the backend observes

The one Veil invariant says a state is not an admitted endpoint: the Scenario accepts it and, under
a `terminal` ending, its Model state is terminal. Veil records every state breaking it, and each one
is read as the reference reads an admitted candidate: its Property answers from the monitors, its
fired clauses from the fired-clause bitset, and the form's stopping rule. Trigger evidence is one
witness per clause, the first endpoint at which the clause fired, from the evaluator on the decoded
trace.

- `violationFound`: `find`, `findViolation` or `pick` stopped at an endpoint.
- `complete`: the frontier emptied, so every state within the depth bound was visited. Under `verify`
  a violation does not stop the search; the first one found is the observations' counterexample,
  which is the shortest.
- `stateBound`: `Limits.search` product states were visited and the frontier is not empty.
- `invalid`: evaluating a Property on a witness trace failed.
-/

namespace Umpire.Search.Backend.Veil

open Umpire
open Umpire.Search.Product
open Veil.ModelChecker
open Veil.ModelChecker.Concrete

/-- The pinned Veil revision this backend runs; `make umpire-check-veil-pin` fails when it differs
from the revision `model/lakefile.lean` requires or `model/lake-manifest.json` resolves. -/
def commit : String := "517f2badbf9a7ba2b18a72242351ff20943cbdd7"

/-- A product state and the depth, in transitions from its root, it was reached at. -/
structure Located (Monitors : Type) where
  depth : Nat
  state : State Monitors
  deriving DecidableEq, Repr

variable {LawStatement : Law → Prop} {target : QueryModel LawStatement} {Monitors : Type}

/-- The product as Veil's enumerable transition system. Every transition succeeds; the theory is
`Unit`. -/
def system (product : Product target Monitors) :
    Veil.EnumerableTransitionSystem Unit (List Unit) (Located Monitors) (List (Located Monitors))
      Int Transition (List (Transition × Veil.ExecutionOutcome Int (Located Monitors))) () where
  initStates := product.initialStates.map ({ depth := 0, state := · })
  tr _ located := (product.successors located.state).map fun (transition, next) =>
    (transition, .success { depth := located.depth + 1, state := next })

/-- The product's own relation, read off `Product.initialStates` and `Product.successors`. -/
def relation (product : Product target Monitors) :
    Veil.RelationalTransitionSystem Unit (Located Monitors) Transition where
  assumptions _ := True
  init _ located := located.depth = 0 ∧ located.state ∈ product.initialStates
  tr _ before transition after :=
    after.depth = before.depth + 1 ∧ (transition, after.state) ∈ product.successors before.state

/-- Every edge of Veil's relational reading of the adapter is a product edge, and the converse. -/
theorem transition_equivalence (product : Product target Monitors)
    (before after : Located Monitors) (transition : Transition) :
    (system product).toRelational.tr () before transition after ↔
      (relation product).tr () before transition after := by
  cases after
  simp only [system, relation, Veil.EnumerableTransitionSystem.toRelational, List.mem_map,
    Prod.mk.injEq, Veil.ExecutionOutcome.success, Veil.ExecutionResult.success.injEq,
    Located.mk.injEq, true_and, Prod.exists]
  constructor
  · rintro ⟨label, next, member, rfl, rfl, rfl⟩
    exact ⟨rfl, member⟩
  · rintro ⟨rfl, member⟩
    exact ⟨_, _, member, rfl, rfl, rfl⟩

/-- Veil's relational reading of the adapter and the product admit the same initial states. -/
theorem initial_equivalence (product : Product target Monitors) (located : Located Monitors) :
    (system product).toRelational.init () located ↔ (relation product).init () located := by
  cases located
  simp only [system, relation, Veil.EnumerableTransitionSystem.toRelational, List.mem_map,
    Located.mk.injEq]
  constructor
  · rintro ⟨root, member, rfl, rfl⟩
    exact ⟨rfl, member⟩
  · rintro ⟨rfl, member⟩
    exact ⟨_, member, rfl, rfl⟩

/-- Both readings accept the one theory. -/
theorem assumptions_equivalence (product : Product target Monitors) (theory : Unit) :
    (system product).toRelational.assumptions theory ↔ (relation product).assumptions theory := by
  simp [system, relation, Veil.EnumerableTransitionSystem.toRelational]

/-! ### Driving the checker -/

/-- Exact deduplication: a `Located` state's fingerprint is its product state. -/
local instance locatedFingerprint [DecidableEq Monitors] [Hashable Monitors] :
    StateFingerprint (Located Monitors) (State Monitors) where
  beq left right := decide (left = right)
  rfl := decide_eq_true rfl
  eq_of_beq equal := of_decide_eq_true equal
  hash_eq {left right} equal := by rw [of_decide_eq_true equal]
  view := Located.state

/-- The backend keeps no per-action statistics. -/
local instance : ActionStatUpdate Transition Unit where
  empty := ()
  increment _ _ _ := ()
  merge _ _ := ()
  dump _ := []

private abbrev Monitored := List ClauseState

private abbrev Context :=
  SequentialSearchContext (Located Monitored) Transition (State Monitored) Unit

/-- The smallest of the depth Limits, as the reference search bounds its iterative deepening. -/
def maximumDepth (query : CheckedQuery LawStatement) : Nat :=
  Nat.min query.limits.steps.value query.limits.actions.value

/-- Whether a trace reaching this state is one the Query's form reads: the Scenario admits it and,
under a `terminal` ending, its Model state is terminal. -/
private def isEndpoint (query : CheckedQuery LawStatement)
    (monitored : MonitoredProduct query.target) (state : State Monitored) : Bool :=
  monitored.product.accepts state &&
    (query.ending != .terminal || query.target.isTerminal state.model)

private def parameters (query : CheckedQuery LawStatement)
    (monitored : MonitoredProduct query.target) : SearchParameters Unit (Located Monitored) := {
  invariants := [{
    name := `notEndpoint
    property := fun _ located => isEndpoint query monitored located.state = false }]
  stateConstraints := [{
    name := `withinDepth
    property := fun _ located => located.depth ≤ maximumDepth query }]
  earlyTerminationConditions := []
}

/-- The trace the checker's parent log records for a visited state. -/
private def traceOf (context : Context) (state : State Monitored) : Scenario.Trace :=
  letI : BEq (State Monitored) := locatedFingerprint.toBEq
  let (root, steps) := retraceSteps context.1.log state
  decode root (steps.map (·.transitionLabel))

/-- What the backend has observed so far, beside the `PlanningObservations` it reports. -/
private structure Observer where
  observations : PlanningObservations
  /-- The clauses with trigger evidence, as `(Property, clause)`. -/
  witnessed : List (DefinitionId × DefinitionId) := []
  /-- The last endpoint Veil recorded, to tell a newly recorded one. -/
  lastEndpoint : Option (State Monitored) := none
  processed : Nat := 0
  roots : Nat := 0
  endpoints : Nat := 0

private def Observer.explored (query : CheckedQuery LawStatement) (observer : Observer) :
    ExploredCounts := {
  setups := observer.roots
  traces := observer.processed
  transitions := observer.processed - observer.roots
  propertyEvaluations := observer.endpoints * query.form.properties.length
}

/-- The evaluator's trigger evidence for one clause on a witness trace. -/
private def clauseEvidence (query : CheckedQuery LawStatement) (trace : Scenario.Trace)
    (propertyId clauseId : DefinitionId) : Except QueryError (List PlanningTriggerEvidence) := do
  let some property := query.form.properties.find? (·.id == propertyId) | pure []
  let input ← (checkPropertyEvaluationInput property trace.trace
      query.target.stateFields).mapError fun error => {
    kind := QueryErrorKind.propertyEvaluationFailure
    definitionId := query.id
    sourcePath := error.sourcePath
    offendingValue := error.kind.name ++ ":" ++ error.offendingValue
    relatedDefinitionIds := DefinitionId.canonicalSet
      (property.id :: property.guardedClauseIds ++ error.relatedDefinitionIds)
  }
  let evaluation := evaluatePropertyEndpoint property input (query.ending == .«partial»)
  pure <| ((evaluation.realizedTriggers.filter (·.clauseId == clauseId)).take 1).map
    ({ trace, trigger := · })

/-- Read one endpoint the checker recorded as the reference reads an admitted candidate. Returns the
updated observer and, when the form stops here, the stopping trace. -/
private def observeEndpoint (query : CheckedQuery LawStatement)
    (monitored : MonitoredProduct query.target) (context : Context) (observer : Observer)
    (state : State Monitored) : Except QueryError (Observer × Option Scenario.Trace) := do
  let answers := monitored.answers state
  let violated := answers.contains .violated
  let unresolved := answers.contains .unresolved
  let requested := monitored.monitors.requested
  let fired := monitored.monitors.firedClauses state.fired
  let covered := !query.requireFiring || requested.all fired.contains
  let trace := traceOf context state
  let mut triggers := #[]
  let mut witnessed := observer.witnessed
  for (propertyId, clauseId) in fired do
    unless witnessed.contains (propertyId, clauseId) do
      triggers := triggers ++ (← clauseEvidence query trace propertyId clauseId)
      witnessed := witnessed ++ [(propertyId, clauseId)]
  let observations := observer.observations
  let observations := { observations with
    nonempty := true
    unresolved := observations.unresolved || unresolved
    required := (observations.required ++ requested).eraseDups
    triggers := observations.triggers ++ triggers.toList
    counterexample := if violated && observations.counterexample.isNone then some trace
      else observations.counterexample }
  let stops := match query.form with
    | .verify _ => false
    | .findViolation _ => violated
    | .find _ => !violated && !unresolved && covered
    | .pick _ => covered
  pure ({ observer with observations, witnessed, endpoints := observer.endpoints + 1 },
    if stops then some trace else none)

private def Observer.finish (query : CheckedQuery LawStatement) (observer : Observer)
    (context : Context) : PlanningObservations := { observer.observations with
  explored := observer.explored query
  instrumentation := {
    enumeratorPulls := observer.processed
    generatedCandidates := context.1.statesFound
    peakActiveFrontierDepth := context.1.currentFrontierDepth + 1
    searchBackend := .veil commit
    searchUnit := .states } }

/-- Run the checker one product state at a time, at most `remaining` more states. -/
private def drive (query : CheckedQuery LawStatement) (monitored : MonitoredProduct query.target)
    (sys : Veil.EnumerableTransitionSystem Unit (List Unit) (Located Monitored)
      (List (Located Monitored)) Int Transition
      (List (Transition × Veil.ExecutionOutcome Int (Located Monitored))) ())
    (remaining : Nat) (context : Context) (observer : Observer) : BackendResult :=
  match context.2.dequeue? with
  | none => .complete (observer.finish query context)
  | some (item, _) =>
      match remaining with
      | 0 => .stateBound observer.processed (observer.finish query context)
      | remaining + 1 =>
          let next := context.bfsStep (parameters query monitored) () sys.tr
          let observer := { observer with
            processed := observer.processed + 1
            roots := observer.roots + if item.depth == 0 then 1 else 0 }
          let recorded := match next.1.violatingStates with
            | (state, _) :: _ => if observer.lastEndpoint == some state then none else some state
            | [] => none
          match recorded with
          | none => drive query monitored sys remaining next observer
          | some state =>
              match observeEndpoint query monitored next { observer with lastEndpoint := state }
                  state with
              | .error error => .invalid error (observer.finish query next)
              | .ok (observer, some trace) => .violationFound trace (observer.finish query next)
              | .ok (observer, none) => drive query monitored sys remaining next observer

/-- Search a Query's product with Veil's checker. -/
def run (query : CheckedQuery LawStatement) (monitored : MonitoredProduct query.target) :
    BackendResult :=
  let parameters := parameters query monitored
  let sys := restrictSystemByStateConstraints (system monitored.product) parameters ()
  drive query monitored sys query.limits.search.value (SequentialSearchContext.initial sys) {
    observations := {} }

/-- The `veil` backend: search the Query's product with Veil's checker, or name the Property clause
or Scenario construct its product cannot encode. -/
def backend? (query : CheckedQuery LawStatement) (view : SearchView query.target) :
    Except ProductUnsupported BackendResult :=
  (run query ·) <$> MonitoredProduct.build query view

end Umpire.Search.Backend.Veil
