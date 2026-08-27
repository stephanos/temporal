import Umpire.Planning

/-! Shared deterministic model, checked query, incremental kernel, and runner fixtures. -/

namespace Umpire.PlanningTests

open Umpire

def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Umpire/Planning/Tests.lean"
  line := 1
  column := 1
  provenance := "lean-test"
}

def phase : DeclarationId := id "planner.state.phase"
def request : DeclarationId := id "planner.action.request"
def accepted : DeclarationId := id "planner.outcome.accepted"
def observed : DeclarationId := id "planner.observation.accepted"
def role : DeclarationId := id "planner.role.operation"
def occurrence : DeclarationId := id "planner.occurrence.request"

def value (identity : DeclarationId) (payload : String) : SemanticValue := {
  identity
  value := payload
}

def initial : SemanticValue := value phase "initial"
def completed : SemanticValue := value phase "completed"
def requestValue : SemanticValue := value request "request"
def acceptedValue : SemanticValue := value accepted "accepted"
def observedValue : SemanticValue := value observed "accepted"
def setup : List RoleBinding := [{ role, value := value phase "operation-a" }]

def transition (_index : Nat) : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := acceptedValue
  resultingState := completed
  observations := [observedValue]
}

def transitions (width : Nat) : List (TransitionResult SemanticValue SemanticValue SemanticValue) :=
  (List.range (width + 1)).map transition

def kernel (width : Nat) : TransitionKernel
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  metadata := {
    id := id "planner.kernel.fixture"
    contractDigest := "planner-kernel/v1"
    source
  }
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
}

def finitePlanning (width : Nat) : FinitePlanningCapability (kernel width).authoritativeStep := {
  actions := [requestValue]
  roleDomainDigest := "role-domain/v1"
  actionDomainDigest := "action-domain/v1"
  actionSound := by
    intro action member
    simp only [List.mem_cons, List.not_mem_nil, or_false] at member
    subst action
    exact ⟨initial, transition 0, rfl, rfl, rfl⟩
  actionComplete := by
    intro state action result admitted
    simp [admitted.2.1]
}

def target (width : Nat) : QueryTarget (fun _ => True) := {
  id := id "planner.target.fixture"
  source
  declarations := []
  requiredCapabilities := []
  providers := []
  connectors := []
  resolvedSetups := [setup]
  kernel := kernel width
  planning := .available (finitePlanning width)
  canonicalMetadata := "target-metadata"
  semanticDigest := "target/v1"
}

def property : CheckedProperty := {
  id := id "planner.property.fixture"
  source
  version := 1
  requires := []
  clauses := []
  access := { capabilities := [], meanings := [], logicalTimeSource := none }
  documentation := "property documentation"
  canonicalMetadata := "property-metadata"
  semanticDigest := "property/v1"
}

def behavior : CheckedBehavior := {
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
  semanticDigest := "behavior/v1"
}

def bounds (budget : Nat := 10) : QueryBounds := {
  behavior := {
    transitions := { value := 1, unit := .semanticTransitions }
    selectedActions := { value := 1, unit := .selectedActions }
  }
  search := { value := budget, unit := .candidateEvaluations }
}

def policy (strategy : SearchStrategy) (seed : Nat := 17) : PlannerPolicy := {
  strategy
  seed
  tieBreak := .semanticIdentity
}

def checkedQuery
    (width : Nat)
    (form : QueryForm)
    (strategy : SearchStrategy)
    (budget : Nat := 10)
    (seed : Nat := 17)
    (withCompleteness : Bool := true)
    (selectedBehavior : CheckedBehavior := behavior) : CheckedQuery (fun _ => True) := {
  id := id "planner.query.fixture"
  source
  version := 1
  form
  quantifier := form.quantifier
  claim := form.claim
  behavior := selectedBehavior
  target := target width
  bounds := bounds budget
  policy := policy strategy seed
  targetComposition := []
  completeness := if withCompleteness then
    (CheckedQueryTarget.ofTarget (target width)).completeness
  else
    none
  documentation := "query documentation"
  canonicalMetadata := "query-metadata"
  semanticDigest := "query/v1:" ++ strategy.name ++ ":" ++ toString seed ++ ":" ++
    selectedBehavior.semanticDigest
}

def orderedQuery (width : Nat) : CheckedQuery (fun _ => True) :=
  checkedQuery width (.witness property) .shortest

def incrementalKernel? (width : Nat) : Option (IncrementalPlannerKernel (target width)) :=
  IncrementalPlannerKernel.ofCheckedQuery? (orderedQuery width)
    (by
      intro evidence evidenceEq
      simp [orderedQuery, checkedQuery, CheckedQueryTarget.ofTarget, target, finitePlanning] at evidenceEq
      cases Option.some.inj evidenceEq
      simp)
    (by
      intro _ _ candidate
      simp only [orderedQuery, checkedQuery, target, kernel]
      split <;> simp)
    (by
      intro _ _ state action
      simp only [orderedQuery, checkedQuery, target, kernel]
      split
      · rw [List.pairwise_iff_getElem]
        intro first second firstBound secondBound earlier
        simp [transitions, transition]
      · simp)

private theorem incrementalKernel?_isSome (width : Nat) :
    (incrementalKernel? width).isSome = true := by
  rfl

def incrementalKernel (width : Nat) : IncrementalPlannerKernel (target width) :=
  (incrementalKernel? width).get (incrementalKernel?_isSome width)

def run
    (width : Nat)
    (form : QueryForm)
    (strategy : SearchStrategy)
    (budget : Nat := 10)
    (seed : Nat := 17)
    (withCompleteness : Bool := true)
    (selectedBehavior : CheckedBehavior := behavior) : PlannerRun :=
  plan (checkedQuery width form strategy budget seed withCompleteness selectedBehavior)
    (incrementalKernel width)

end Umpire.PlanningTests
