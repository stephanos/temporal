import Umpire.Search.Product.Monitor

/-!
# The product state space

A state-space search over a checked Query explores the product of four things: the Model state the
`SearchView` steps, the Scenario's `Progress` (`Umpire.Search.Product.Scenario`), the states of the
Property monitors, and a bitset of the clauses whose trigger has fired. `Product` builds that space
from a `SearchView`, a `CheckedScenario` and a supplied `MonitorFamily`; its transitions are
labelled with the Model `Step` taken, so a product path decodes to a `Scenario.Trace` with nothing
lost.

## Invariant: a product state determines the future verdict

Everything a Query's verdict reads of a trace's past is in its last product `State`: the Model
state the future steps from, the Scenario progress that `ScenarioAutomaton.accepts` reads, the
monitor states the Property answers are read from, and the fired-clause bitset that coverage reads.
The setup is kept too, because the Scenario's root decision and a decoded witness read it. Two
paths that reach equal states therefore have the same admitted extensions with the same verdicts,
which is what makes deduplicating a visited set sound. `State` is `BEq` and `Hashable` for that
visited set.

The state does not record its depth. Under a depth bound (`maximumDepth`), two equal states reached
at different depths have the same extensions but different remaining budgets, so dedup keeps every
admitted trace within the bound only when a state is first reached at its minimal depth, which a
breadth-first frontier guarantees.

## Order

`Product.initialStates` lists roots by sorted setup and then initial index, and
`Product.successors` lists transitions by action index and then outcome index: the key order the
reference search enumerates candidates in, so a backend that keeps first-discovery parents reports
the witness the reference would.

The product never enumerates what the Scenario rejects: a root or a transition the automaton drops
has no admitted extension, so it is not a product state. The search strategy's seed is not read;
seeded search stays on the reference backend.

## The Query's own monitors

`MonitoredProduct.build` builds the product a Query's search runs: its Scenario automaton beside
the Property monitors of `Umpire.Search.Product.Monitor` (`MonitorFamily.ofQuery`), or the first
Property clause or Scenario construct version one cannot encode.
-/

namespace Umpire.Search.Product

-- The product is `Umpire.Search.Product.Product`: its namespace names the module the whole state
-- space lives in, and renaming the structure would move every caller of `Product.build`.
set_option linter.extra.dupNamespace false

/-- The Property monitors a product runs beside the Scenario automaton. `start` gives the monitor
states at a root and `advance` steps them along one transition from a Model state; each also
returns the bits of the clauses whose trigger fired, which the product ORs into its fired-clause
bitset. -/
structure MonitorFamily (Monitors : Type) where
  start : List RoleBinding → ModelValue → Monitors × Nat
  advance : Monitors → ModelValue → Transition → Monitors × Nat

/-- The family with no monitors, for a product that tracks the Scenario alone. -/
def MonitorFamily.empty : MonitorFamily Unit where
  start _ _ := ((), 0)
  advance _ _ _ := ((), 0)

/-- The family of a Query's lowered Property monitors; clause `i` owns fired bit `i`. -/
def MonitorFamily.ofQuery (monitors : QueryMonitors) : MonitorFamily (List ClauseState) where
  start _ initialState := monitors.start initialState
  advance := monitors.advance

/-- One product state. See the module header for why it determines the future verdict. -/
structure State (Monitors : Type) where
  setup : List RoleBinding
  model : ModelValue
  progress : Progress
  monitors : Monitors
  fired : Nat
  deriving BEq, DecidableEq, Repr

private def hashValue (value : ModelValue) : UInt64 :=
  mixHash (hash value.definitionId) (hash value.value)

private def hashSetup (setup : List RoleBinding) : UInt64 :=
  setup.foldl (fun accumulated binding =>
    mixHash accumulated (mixHash (hash binding.role) (hashValue binding.value))) 7

instance [Hashable Monitors] : Hashable (State Monitors) where
  hash state :=
    mixHash (hashSetup state.setup) <|
      mixHash (hashValue state.model) <|
        mixHash (hash state.progress) <|
          mixHash (hash state.monitors) (hash state.fired)

/-- The product of one checked Query's search view, Scenario automaton and Property monitors. -/
structure Product {LawStatement : Law → Prop} (target : QueryModel LawStatement)
    (Monitors : Type) where
  view : SearchView target
  /-- The setups roots are drawn from, sorted. -/
  setups : List (List RoleBinding)
  scenario : ScenarioAutomaton
  monitors : MonitorFamily Monitors

variable {LawStatement : Law → Prop} {target : QueryModel LawStatement} {Monitors : Type}

private def setupLe (left right : List RoleBinding) : Bool :=
  compare left right != .gt

/-- Build the product for one checked Query, or name the Scenario construct it cannot encode. The
setups are the Query's finite role assignments, or the Model's resolved setups without finite
evidence, sorted as the reference search sorts them. -/
def Product.build
    (query : CheckedQuery LawStatement)
    (view : SearchView query.target)
    (monitors : MonitorFamily Monitors) : Except Unsupported (Product query.target Monitors) := do
  let scenario ← ScenarioAutomaton.lower query.behavior
  let setups := match query.completeness with
    | some evidence => evidence.roleAssignments
    | none => query.target.resolvedSetups
  pure { view, setups := setups.mergeSort setupLe, scenario, monitors }

namespace Product

variable (product : Product target Monitors)

/-- The root at this setup and initial state, or `none` when the Scenario admits no trace from
it. -/
def root? (setup : List RoleBinding) (initialState : ModelValue) : Option (State Monitors) :=
  (product.scenario.start setup initialState).map fun progress =>
    let (monitors, fired) := product.monitors.start setup initialState
    { setup, model := initialState, progress, monitors, fired }

/-- The roots, by sorted setup and then initial index. -/
def initialStates : List (State Monitors) :=
  product.setups.flatMap fun setup =>
    (List.range (product.view.initialLimit setup)).filterMap fun index =>
      (product.view.initialAt setup index).bind (product.root? setup)

/-- The state one transition leads to, or `none` when the Scenario admits no trace continuing this
way. It does not check that the transition is the Model's; `successors` only offers ones that
are. -/
def follow (state : State Monitors) (transition : Transition) : Option (State Monitors) :=
  (product.scenario.step state.progress transition).map fun progress =>
    let (monitors, bits) := product.monitors.advance state.monitors state.model transition
    { state with
      model := transition.result.state
      progress
      monitors
      fired := state.fired ||| bits }

/-- The transitions out of a state with the states they lead to, by action index and then outcome
index. -/
def successors (state : State Monitors) : List (Transition × State Monitors) :=
  (List.range product.view.actionLimit).flatMap fun actionIndex =>
    match product.view.actionAt actionIndex with
    | none => []
    | some action =>
        (List.range (product.view.stepLimit state.model action)).filterMap fun outcomeIndex =>
          (product.view.stepAt state.model action outcomeIndex).bind fun result =>
            let transition := { action, result }
            (product.follow state transition).map (transition, ·)

/-- Whether the Scenario admits a trace that reached this state. -/
def accepts (state : State Monitors) : Bool :=
  product.scenario.accepts state.progress

/-- Follow a whole path from a state. -/
def run (state : State Monitors) (path : List Transition) : Option (State Monitors) :=
  path.foldlM product.follow state

end Product

/-- The trace a product path from `root` records: the root's setup and Model state, then one step
per transition. -/
def decode (root : State Monitors) (path : List Transition) : Scenario.Trace := {
  setup := root.setup
  trace := { initialState := root.model, steps := path.map Transition.toTraceStep }
}

/-- Decoding loses nothing: the setup, initial Model state and path come back out of the trace. -/
theorem decode_lossless (root : State Monitors) (path : List Transition) :
    (decode root path).setup = root.setup ∧
      (decode root path).trace.initialState = root.model ∧
      (decode root path).trace.steps.map Transition.ofTraceStep = path := by
  refine ⟨rfl, rfl, ?_⟩
  simp [decode, Function.comp_def]

namespace Product

variable (product : Product target Monitors)

/-- The Scenario progress of a followed transition is the automaton's step. -/
theorem follow_progress (state : State Monitors) (transition : Transition) :
    (product.follow state transition).map State.progress =
      product.scenario.step state.progress transition := by
  simp only [follow, Option.map_map]
  cases product.scenario.step state.progress transition <;> rfl

/-- The Scenario progress at the end of a followed path is the automaton run along that path. -/
theorem run_progress (state : State Monitors) (path : List Transition) :
    (product.run state path).map State.progress =
      product.scenario.runFrom state.progress path := by
  induction path generalizing state with
  | nil => rfl
  | cons transition rest ih =>
      have step := product.follow_progress state transition
      simp only [run, ScenarioAutomaton.runFrom, List.foldlM_cons] at ih ⊢
      cases next : product.follow state transition with
      | none =>
          rw [next] at step
          simp only [Option.map_none] at step
          simp [← step]
      | some following =>
          rw [next] at step
          simp only [Option.map_some] at step
          simp only [Option.bind_eq_bind, Option.bind_some, ← step]
          exact ih following

/-- A followed path is admitted exactly when the automaton run along it from the state's progress
accepts. -/
theorem run_accepts (state : State Monitors) (path : List Transition) :
    (product.run state path).any product.accepts =
      (product.scenario.runFrom state.progress path).any product.scenario.accepts := by
  rw [← product.run_progress]
  cases product.run state path <;> rfl

/-- A product path from a root is admitted exactly when the Scenario automaton admits the trace it
decodes to: the product's acceptance is the automaton's, whatever the monitors do. -/
theorem accepts_iff_admits
    (setup : List RoleBinding) (initialState : ModelValue) (path : List Transition) :
    ((product.root? setup initialState).bind (product.run · path)).any product.accepts =
      product.scenario.admits
        { setup, trace := { initialState, steps := path.map Transition.toTraceStep } } := by
  simp only [ScenarioAutomaton.admits, ScenarioAutomaton.run, root?]
  cases start : product.scenario.start setup initialState with
  | none => rfl
  | some progress =>
      simp only [Option.map_some, Option.bind_some, List.map_map, Function.comp_def,
        Transition.ofTraceStep_toTraceStep, List.map_id']
      exact product.run_accepts _ path

/-- Every successor is the state `follow` gives for a transition the search view offers. -/
theorem mem_successors {state next : State Monitors} {transition : Transition}
    (member : (transition, next) ∈ product.successors state) :
    product.follow state transition = some next ∧
      (∃ index, index < product.view.actionLimit ∧
        product.view.actionAt index = some transition.action) ∧
      ∃ index, index < product.view.stepLimit state.model transition.action ∧
        product.view.stepAt state.model transition.action index = some transition.result := by
  simp only [successors, List.mem_flatMap, List.mem_range] at member
  obtain ⟨actionIndex, actionBound, member⟩ := member
  split at member
  · simp at member
  · rename_i action selected
    simp only [List.mem_filterMap, List.mem_range] at member
    obtain ⟨outcomeIndex, outcomeBound, member⟩ := member
    cases stepped : product.view.stepAt state.model action outcomeIndex with
    | none => simp [stepped] at member
    | some result =>
        simp only [stepped, Option.bind_some] at member
        cases followed : product.follow state { action, result } with
        | none => simp [followed] at member
        | some following =>
            simp only [followed, Option.map_some, Option.some.injEq, Prod.mk.injEq] at member
            obtain ⟨rfl, rfl⟩ := member
            exact ⟨followed, ⟨actionIndex, actionBound, selected⟩,
              ⟨outcomeIndex, outcomeBound, stepped⟩⟩

end Product

/-- Why a Query has no version-one product: its Scenario or one of its Property clauses. -/
inductive ProductUnsupported where
  | scenario (unsupported : Unsupported)
  | clause (unsupported : MonitorUnsupported)
  deriving BEq, DecidableEq, Repr

/-- A Query's product with its own Property monitors, and how to read its answers. -/
structure MonitoredProduct (target : QueryModel LawStatement) where
  product : Product target (List ClauseState)
  monitors : QueryMonitors
  /-- Whether the Query's ending is `partial`, under which a monitor may answer `unresolved`. -/
  partialTrace : Bool

/-- Build the product a Query's search runs, or name the first Property clause or, failing that,
the Scenario construct version one cannot encode. -/
def MonitoredProduct.build
    (query : CheckedQuery LawStatement)
    (view : SearchView query.target) :
    Except ProductUnsupported (MonitoredProduct query.target) := do
  let monitors ← (QueryMonitors.lower query.form.properties query.target.stateFields).mapError
    .clause
  let product ← (Product.build query view (.ofQuery monitors)).mapError .scenario
  pure { product, monitors, partialTrace := query.ending == .«partial» }

/-- Each Property's answer, by Definition ID, on the trace that reached this state. -/
def MonitoredProduct.answers (monitored : MonitoredProduct target)
    (state : State (List ClauseState)) : List PropertyEndpointAnswer :=
  monitored.monitors.answers state.monitors monitored.partialTrace

end Umpire.Search.Product
