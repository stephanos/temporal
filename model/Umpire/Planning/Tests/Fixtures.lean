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

def phase := id "planner.state.phase"
def request := id "planner.action.request"
def accepted := id "planner.outcome.accepted"
def observed := id "planner.observation.accepted"
def role := id "planner.role.operation"
def occurrence := id "planner.occurrence.request"

def value (identity : DeclarationId) (payload : String) : SemanticValue := {
  identity
  value := payload
}

def initial := value phase "initial"
def completed := value phase "completed"
def requestValue := value request "request"
def acceptedValue := value accepted "accepted"
def observedValue := value observed "accepted"
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
  initialSound := by intros; split at * <;> simp_all
  initialComplete := by intros; simp_all
  steps := fun state action =>
    if state = initial ∧ action = requestValue then transitions width else []
  authoritativeStep := fun state action result =>
    state = initial ∧ action = requestValue ∧ result = transition 0
  stepSound := by
    intro state action result member
    split at member <;> simp_all [transitions, transition]
  stepComplete := by
    intro state action result admitted
    rcases admitted with ⟨rfl, rfl, rfl⟩
    rw [if_pos ⟨rfl, rfl⟩]
    apply List.mem_map.mpr
    exact ⟨0, by simp, rfl⟩
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

def completeness (width : Nat) : FiniteCompletenessEvidence (fun _ => True) (target width) := {
  roleAssignments := [setup]
  actions := [requestValue]
  roleDomainDigest := "role-domain/v1"
  actionDomainDigest := "action-domain/v1"
  roleSound := by simp [target]
  roleComplete := by simp [target]
  actionSound := by
    intro action member
    simp only [List.mem_cons, List.not_mem_nil, or_false] at member
    subst action
    exact ⟨initial, transition 0, rfl, rfl, rfl⟩
  actionComplete := by
    intro state action result admitted
    simp [admitted.2.1]
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
  completeness := if withCompleteness then some (completeness width) else none
  documentation := "query documentation"
  canonicalMetadata := "query-metadata"
  semanticDigest := "query/v1:" ++ strategy.name ++ ":" ++ toString seed ++ ":" ++
    selectedBehavior.semanticDigest
}

def incrementalKernel (width : Nat) : IncrementalPlannerKernel (target width) :=
  .ofFinite (completeness width) {
    action := by
      simp [completeness]
    initial := by
      intro candidate
      simp only [target, kernel]
      split <;> simp
    step := by
      intro state action
      simp only [target, kernel]
      split
      · rw [List.pairwise_iff_getElem]
        intro first second firstBound secondBound earlier
        simp [transitions, transition]
      · simp
  }

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
