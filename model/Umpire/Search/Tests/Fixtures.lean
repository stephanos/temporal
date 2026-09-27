import Umpire.Search
import Umpire.Search.Branches
import Umpire.Model.Table
import Umpire.Shared.Test

/-! Shared deterministic model, checked query, incremental kernel, and runner fixtures. -/

namespace Umpire.SearchTests

open Umpire

def id (value : String) : DefinitionId := Shared.Test.definitionId value

def source : SourceLocation := Shared.Test.sourceLocation "Umpire/Planning/Tests.lean"

def phase : DefinitionId := id "planner.state.phase"
def request : DefinitionId := id "planner.action.request"
def accepted : DefinitionId := id "planner.outcome.accepted"
def observed : DefinitionId := id "planner.observation.accepted"
def role : DefinitionId := id "planner.role.operation"
def occurrence : DefinitionId := id "planner.occurrence.request"
def targetId : DefinitionId := id "planner.target.fixture"
def kernelId : DefinitionId := id "planner.kernel.fixture"

def metadata
    (definitionId : DefinitionId)
    (kind : DefinitionKind)
    (behaviorVersion : String) : DefinitionMetadata :=
  { Shared.Test.definitionMetadata definitionId.value kind source behaviorVersion with
    id := definitionId
    documentation := "planning fixture"
  }

def value (definitionId : DefinitionId) (payload : String) : ModelValue :=
  ModelValue.named definitionId payload

def initial : ModelValue := value phase "initial"
def completed : ModelValue := value phase "completed"
def requestValue : ModelValue := value request "request"
def acceptedValue : ModelValue := value accepted "accepted"
def observedValue : ModelValue := value observed "accepted"
def setup : List RoleBinding := [{ role, value := value phase "operation-a" }]

def transition (_index : Nat) : Step ModelValue ModelValue ModelValue := {
  outcome := acceptedValue
  state := completed
  facts := [observedValue]
}

def transitions (width : Nat) : List (Step ModelValue ModelValue ModelValue) :=
  (List.range (width + 1)).map transition

def kernel (width : Nat) : Machine
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  metadata := {
    id := kernelId
    source
  }
  setupDomain := fun candidate => candidate = setup
  stateDomain := fun candidate => candidate = initial ∨ candidate = completed
  actionDomain := fun candidate => candidate = requestValue
  outcomeDomain := fun candidate => candidate = acceptedValue
  observationDomain := fun candidate => candidate = observedValue
  initialStates := fun candidate => if candidate = setup then [initial] else []
  authoritativeInitial := fun candidate state => candidate = setup ∧ state = initial
  initialSound := by
    intro candidate state member
    by_cases selected : candidate = setup
    · rw [if_pos selected] at member
      exact ⟨selected, List.mem_singleton.mp member⟩
    · rw [if_neg selected] at member
      exact (List.not_mem_nil member).elim
  initialComplete := by
    intro candidate state admitted
    rcases admitted with ⟨rfl, rfl⟩
    simp
  steps := fun state action =>
    if state = initial ∧ action = requestValue then transitions width else []
  authoritativeStep := fun state action result =>
    state = initial ∧ action = requestValue ∧ result = transition 0
  stepSound := by
    intro state action result member
    by_cases selected : state = initial ∧ action = requestValue
    · rw [if_pos selected] at member
      simp only [transitions] at member
      obtain ⟨index, _, rfl⟩ := List.mem_map.mp member
      exact ⟨selected.1, selected.2, rfl⟩
    · rw [if_neg selected] at member
      exact (List.not_mem_nil member).elim
  stepComplete := by
    intro state action result admitted
    rcases admitted with ⟨rfl, rfl, rfl⟩
    rw [if_pos ⟨rfl, rfl⟩]
    apply List.mem_map.mpr
    exact ⟨0, by simp, rfl⟩
  vocabulary := .complete {
    setups := [setup]
    states := [initial, completed]
    actions := [requestValue]
    outcomes := [acceptedValue]
    observations := [observedValue]
    encodeSetup := fun bindings => String.intercalate "|" (bindings.map fun binding =>
      binding.role.value ++ "=" ++ binding.value.definitionId.value ++ ":" ++ binding.value.value)
    encodeState := fun modelValue => modelValue.definitionId.value ++ ":" ++ modelValue.value
    encodeAction := fun modelValue => modelValue.definitionId.value ++ ":" ++ modelValue.value
    encodeOutcome := fun modelValue => modelValue.definitionId.value ++ ":" ++ modelValue.value
    encodeObservation := fun modelValue => modelValue.definitionId.value ++ ":" ++ modelValue.value
    setupSound := by intro candidate member; simpa using member
    setupComplete := by intro candidate admitted; simpa using admitted
    stateSound := by intro candidate member; simpa using member
    stateComplete := by intro candidate admitted; simpa using admitted
    actionSound := by intro candidate member; simpa using member
    actionComplete := by intro candidate admitted; simpa using admitted
    outcomeSound := by intro candidate member; simpa using member
    outcomeComplete := by intro candidate admitted; simpa using admitted
    observationSound := by intro candidate member; simpa using member
    observationComplete := by intro candidate admitted; simpa using admitted
    setupCoverage := by
      intro candidate state member
      by_cases selected : candidate = setup
      · simp [selected]
      · simp [selected] at member
    initialStateCoverage := by
      intro candidate state member
      by_cases selected : candidate = setup
      · rw [if_pos selected] at member
        simp [List.mem_singleton.mp member]
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
    transitionSourceCoverage := by
      intro state action result member
      by_cases selected : state = initial ∧ action = requestValue
      · simp [selected.1]
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
    actionCoverage := by
      intro state action result member
      by_cases selected : state = initial ∧ action = requestValue
      · simp [selected.2]
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
    resultingStateCoverage := by
      intro state action result member
      by_cases selected : state = initial ∧ action = requestValue
      · rw [if_pos selected] at member
        obtain ⟨index, _, rfl⟩ := List.mem_map.mp member
        simp [transition]
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
    outcomeCoverage := by
      intro state action result member
      by_cases selected : state = initial ∧ action = requestValue
      · rw [if_pos selected] at member
        obtain ⟨index, _, rfl⟩ := List.mem_map.mp member
        simp [transition]
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
    observationCoverage := by
      intro state action result observation member observationMember
      by_cases selected : state = initial ∧ action = requestValue
      · rw [if_pos selected] at member
        obtain ⟨index, _, rfl⟩ := List.mem_map.mp member
        simpa [transition] using observationMember
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
  }
}

def finitePlanning (width : Nat) : FinitePlanningCapability (kernel width).authoritativeStep := {
  actions := [requestValue]
  actionSound := by
    intro action member
    simp only [List.mem_cons, List.not_mem_nil, or_false] at member
    subst action
    exact ⟨initial, transition 0, rfl, rfl, rfl⟩
  actionComplete := by
    intro state action result admitted
    simp [admitted.2.1]
}

def modelSpec (width : Nat) : ModelSpec (fun _ => True)
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetId
  source
  definitions := [
    metadata targetId .target "planner-target/v1",
    metadata kernelId .machine "planner-kernel/v1"
  ]
  requiredCapabilities := []
  resolvedSetups := [setup]
  machine := .checked (kernel width)
}

def targetAuthoring : DraftModel (fun _ => True)
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  DraftModel.make (modelSpec 0) Providers.empty
    (.available (kernel 0) rfl (finitePlanning 0))

def baseTarget : QueryModel (fun _ => True) := model targetAuthoring

private theorem completed_ne_initial : completed ≠ initial := by
  native_decide

private theorem eraseDups_replicate_append_two
    [BEq α] (width : Nat) (value : α) (self : (value == value) = true) :
    (List.replicate width value ++ [value, value]).eraseDups = [value] := by
  cases width <;> simp [List.replicate_succ, List.eraseDups_cons, self]

private theorem kernelBehaviorDescription_eq (width : Nat) :
    (kernel width).behaviorTable? = (kernel 0).behaviorTable? := by
  induction width with
  | zero => rfl
  | succ width ih =>
      simp [Machine.behaviorTable?, Machine.describeBehavior, kernel,
        transitions, transition, completed_ne_initial, Function.comp_def, List.map_const',
        List.range_succ]
      rw [eraseDups_replicate_append_two _ _ (by native_decide)]
      native_decide

def target (width : Nat) : QueryModel (fun _ => True) :=
  baseTarget.withEquivalentMachine (kernel width)
    (by simp [baseTarget, model, targetAuthoring, DraftModel.make, modelSpec,
      kernel])
    (by simp [baseTarget, model, targetAuthoring, DraftModel.make, modelSpec,
      kernel])
    (by simp [baseTarget, model, targetAuthoring, DraftModel.make, modelSpec,
      kernel])
    (by simp [baseTarget, model, targetAuthoring, DraftModel.make, modelSpec,
      kernel])
    (kernelBehaviorDescription_eq width |>.trans (by native_decide))
    (.available (finitePlanning width))

def property : CheckedProperty := {
  id := id "planner.property.fixture"
  source
  version := 1
  requires := []
  clauses := []
  access := { capabilities := [], meanings := [], logicalTimeSource := none }
  documentation := "property documentation"
  canonicalMetadata := "property-metadata"
  behaviorFingerprint := behaviorFingerprintOf "property/v1"
}

def behavior : CheckedScenario := {
  id := id "planner.behavior.fixture"
  source
  version := 1
  requires := []
  roles := [{ id := role, valueKind := .state }]
  setup := []
  allowedActions := [request]
  requiredOccurrences := [{ id := occurrence, action := request }]
  forbiddenActions := []
  occurrenceBounds := []
  ordering := []
  sequences := []
  adjacencies := []
  actionsExactly := some [request]
  traceExactly := none
  spaceStatus := .unclassified
  documentation := "behavior documentation"
  canonicalMetadata := "behavior-metadata"
  behaviorFingerprint := behaviorFingerprintOf "behavior/v1"
}

def limits (budget : Nat := 10) : Limits := Limits.bounded 1 1 budget

def policy (strategy : SearchStrategy) (seed : Nat := 17) : PlannerPolicy :=
  match strategy with
  | .shortest => { PlannerPolicy.shortest with seed }
  | .exhaustive => { PlannerPolicy.exhaustive with seed }
  | .seeded => PlannerPolicy.seeded seed
  | .breadthFirst => {
      strategy
      seed
    }

def fixtureQuery
    (width : Nat)
    (form : Query.Form)
    (strategy : SearchStrategy)
    (budget : Nat := 10)
    (seed : Nat := 17)
    (withCompleteness : Bool := true)
    (selectedBehavior : CheckedScenario := behavior) : CheckedQuery (fun _ => True) := {
  id := id "planner.query.fixture"
  source
  version := 1
  form
  behavior := selectedBehavior
  target := target width
  limits := limits budget
  policy := policy strategy seed
  modelProviders := []
  completeness := if withCompleteness then
    (ModelCompleteness.ofTarget (target width)).completeness
  else
    none
  documentation := "query documentation"
  canonicalMetadata := "query-metadata"
  behaviorFingerprint := behaviorFingerprintOf <|
    "query/v1:" ++ strategy.name ++ ":" ++ toString seed ++ ":" ++
      selectedBehavior.behaviorFingerprint.render
}

def orderedQuery (width : Nat) : CheckedQuery (fun _ => True) :=
  fixtureQuery width (.find property) .shortest

def incrementalKernel? (width : Nat) : Option (SearchView (target width)) :=
  SearchView.ofCheckedQuery? (orderedQuery width)
    (by
      intro evidence evidenceEq
      simp [orderedQuery, fixtureQuery, policy, ModelCompleteness.ofTarget, target,
        CheckedModel.withEquivalentMachine, baseTarget, model, targetAuthoring,
        DraftModel.make, modelSpec, finitePlanning] at evidenceEq
      subst evidenceEq
      simp)
    (by
      intro _ _ candidate
      simp only [orderedQuery, fixtureQuery, policy, target, CheckedModel.withEquivalentMachine,
        baseTarget, model, targetAuthoring, DraftModel.make, modelSpec, kernel]
      split <;> simp)
    (by
      intro _ _ state action
      simp only [orderedQuery, fixtureQuery, policy, target, CheckedModel.withEquivalentMachine,
        baseTarget, model, targetAuthoring, DraftModel.make, modelSpec, kernel]
      split
      · rw [List.pairwise_iff_getElem]
        intro first second firstBound secondBound earlier
        simp [transitions, transition]
      · simp)

private theorem incrementalKernel?_isSome (width : Nat) :
    (incrementalKernel? width).isSome = true := by
  rfl

def incrementalKernel (width : Nat) : SearchView (target width) :=
  (incrementalKernel? width).get (incrementalKernel?_isSome width)

def run
    (width : Nat)
    (form : Query.Form)
    (strategy : SearchStrategy)
    (budget : Nat := 10)
    (seed : Nat := 17)
    (withCompleteness : Bool := true)
    (selectedBehavior : CheckedScenario := behavior) : Except KnownGapError PlanResult :=
  search (fixtureQuery width form strategy budget seed withCompleteness selectedBehavior)
    (incrementalKernel width)

/-! ### Table fixtures for the backend differential

Two Models written as finite tables, for the backend differential: one where two paths reach one
product state at the same depth, and three instances of a two-state machine, whose paths outgrow a
search bound its states never approach. Every state, action and outcome is named by its key under
the table's own family of Definition IDs, and nothing binds a setup role. -/

/-- A table's Model under the family `planner.<family>`. -/
def tableTarget (family : String)
    (table : FiniteTable Unit String String String String) : Option (QueryModel (fun _ => True)) :=
  let owned := fun (kind : String) => id s!"planner.{family}.{kind}"
  let identity : FiniteModelIdentity Unit String String String String := {
    setupBindings := fun _ => []
    stateId := fun _ => owned "state"
    actionId := fun action => owned ("action." ++ action)
    outcomeId := fun _ => owned "outcome"
    factId := fun _ => owned "fact"
  }
  let definition : TableModelSpec := {
    id := owned "target"
    source
    metadata := { id := owned "kernel", source }
    requiredCapabilities := []
    definitions := [
      metadata (owned "target") .target "target/v1",
      metadata (owned "kernel") .machine "kernel/v1",
      metadata (owned "state") .state "state/v1",
      metadata (owned "outcome") .outcome "outcome/v1"] ++
      table.actions.map fun action =>
        metadata (owned ("action." ++ action.key)) .action "action/v1"
  }
  (table.checkModel identity definition).toOption

private def catalog (keys : List String) : FiniteCatalog String :=
  keys.map fun key => { value := key, key }

private def row (source action target : String) : FiniteTransitionRow String String String String :=
  { key := source ++ "-" ++ action, source, action,
    results := [{ outcome := "accepted", state := target, facts := [] }] }

/-- `a` and `b` both lead from `idle` to `mid` with equal results, and `c` from `mid` to `done`: the
two paths to `mid` reach one product state at depth one. -/
def sameDepthTable : FiniteTable Unit String String String String := {
  setups := [{ value := (), key := "setup" }]
  states := catalog ["idle", "mid", "done"]
  actions := catalog ["a", "b", "c"]
  outcomes := catalog ["accepted"]
  facts := []
  initial := [{ setup := (), states := ["idle"] }]
  transitions := [row "idle" "a" "mid", row "idle" "b" "mid", row "mid" "c" "done"]
}

/-- The switch positions of three instances of a two-state machine, first instance first. -/
private def threeSwitches : List (List Bool) :=
  [false, true].flatMap fun first => [false, true].flatMap fun second =>
    [false, true].map fun third => [first, second, third]

private def switchesKey (switches : List Bool) : String :=
  "-".intercalate (switches.map fun on => if on then "on" else "off")

/-- Three instances of a two-state machine, each flipped by its own action: eight states, and
three enabled transitions from each, so the paths of each length triple. -/
def threeInstanceTable : FiniteTable Unit String String String String := {
  setups := [{ value := (), key := "setup" }]
  states := catalog (threeSwitches.map switchesKey)
  actions := catalog ["flip-1", "flip-2", "flip-3"]
  outcomes := catalog ["accepted"]
  facts := []
  initial := [{ setup := (), states := [switchesKey [false, false, false]] }]
  transitions := threeSwitches.flatMap fun switches =>
    (List.range 3).map fun slot =>
      row (switchesKey switches) s!"flip-{slot + 1}"
        (switchesKey (switches.modify slot (!·)))
}

/-- A Query over a table fixture's Model, with its finite completeness, and the search view over
it. -/
def tableQuery (model : QueryModel (fun _ => True)) (key : String) (form : Query.Form)
    (selectedBehavior : CheckedScenario) (limits : Limits) (strategy : SearchStrategy) :
    Option ((query : CheckedQuery (fun _ => True)) × SearchView query.target) := do
  let checked ← (Query.check (.ofTarget model) {
    id := id ("planner.query." ++ key)
    source
    target := model.id
    form
    limits
    policy := policy strategy
    behavior := selectedBehavior
  }).toOption
  let query := { checked with
    target := model
    completeness := (ModelCompleteness.ofTarget model).completeness }
  let view ← (SearchView.ofCheckedQuery query.target.id query).toOption
  pure ⟨query, view⟩

end Umpire.SearchTests
