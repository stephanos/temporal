import Temporal.Experiment.Planner

namespace Temporal.Experiment.PlannerTests

open Temporal.Experiment

def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Temporal/Experiment/PlannerTests.lean"
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

def incrementalKernel (width : Nat) : IncrementalPlannerKernel (target width) := {
  actionLimit := 1
  actionAt := fun index => if index = 0 then some requestValue else none
  initialLimit := fun candidate => if candidate = setup then 1 else 0
  initialAt := fun candidate index =>
    if candidate = setup ∧ index = 0 then some initial else none
  stepLimit := fun state action =>
    if state = initial ∧ action = requestValue then width + 1 else 0
  stepAt := fun state action index =>
    if state = initial ∧ action = requestValue ∧ index < width + 1 then
      some (transition index)
    else
      none
  actionSound := by
    intro index action inBounds emitted
    simp only [Nat.lt_one_iff] at inBounds
    subst index
    simp at emitted
    subst action
    exact ⟨initial, transition 0, rfl, rfl, rfl⟩
  actionComplete := by
    intro state action result admitted
    exact ⟨0, by simp, by simp [admitted.2.1]⟩
  initialSound := by
    intro candidate index state inBounds emitted
    change candidate = setup ∧ state = initial
    by_cases selected : candidate = setup ∧ index = 0
    · simp [selected] at emitted
      exact ⟨selected.1, emitted.symm⟩
    · simp [selected] at emitted
  initialComplete := by
    intro candidate state admitted
    exact ⟨0, by simp [admitted.1], by simp [admitted.1, admitted.2]⟩
  stepSound := by
    intro state action index result inBounds emitted
    change state = initial ∧ action = requestValue ∧ result = transition 0
    by_cases selected : state = initial ∧ action = requestValue ∧ index < width + 1
    · simp [selected] at emitted
      exact ⟨selected.1, selected.2.1, by simpa [transition] using emitted.symm⟩
    · simp [selected] at emitted
  stepComplete := by
    intro state action result admitted
    exact ⟨0, by simp [admitted.1, admitted.2.1], by simp [admitted.1, admitted.2]⟩
  actionOrdered := by intros; simp_all [semanticValueOrderKey]
  initialOrdered := by intros; simp_all [semanticValueOrderKey]
  stepOrdered := by intros; simp_all [transitionResultOrderKey, transition]
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

def outcomeName
    (form : QueryForm)
    (strategy : SearchStrategy)
    (withCompleteness : Bool := true) : String :=
  (run 2 form strategy 10 17 withCompleteness).result.outcome.name

/-! Each Query form preserves its exact result semantics over the same deterministic kernel. -/
example : [
    outcomeName (.verify property) .exhaustive,
    outcomeName (.witness property) .shortest false,
    outcomeName (.counterexample property) .exhaustive,
    outcomeName (.select [property]) .breadthFirst false
  ] = [
    "verified-within-bounds",
    "found",
    "no-such-trace-within-complete-bounds",
    "found"
  ] := by
  native_decide

/-! Complete absence and exhausted effort remain distinct while retaining counts and bounds. -/
example :
    let complete := run 0 (.counterexample property) .exhaustive
    let exhausted := run 64 (.counterexample property) .shortest 1 17 false
    (complete.result.outcome.name, complete.result.metadata.completeness.established,
      complete.result.metadata.completeness.bounds,
      exhausted.result.outcome.name, exhausted.result.metadata.completeness.established) =
      ("no-such-trace-within-complete-bounds", true, bounds,
        "budget-exhausted", false) := by
  native_decide

def witnessSpec (seed : Nat := 17) : Option ExperimentSpec :=
  (run 2 (.witness property) .shortest 10 seed false).artifact

def incidentalWitnessSpec : Option ExperimentSpec :=
  let query := checkedQuery 2 (.witness property) .shortest 10 17 false
  let incidental : CheckedQuery (fun _ => True) := {
    query with
    documentation := "changed query documentation"
    behavior := { query.behavior with documentation := "changed behavior documentation" }
    form := .witness { property with documentation := "changed property documentation" }
  }
  (plan incidental (incrementalKernel 2)).artifact

def selectedArtifactIsInspectable : Bool :=
  match witnessSpec with
  | none => false
  | some spec =>
      spec.plan.initialState == initial &&
      spec.plan.requestedActions == [requestValue] &&
      spec.plan.modelOutcomes == [acceptedValue] &&
      spec.plan.linearExtension.map PlannedOccurrence.identity == [occurrence] &&
      spec.plan.linearExtension.map PlannedOccurrence.action == [request] &&
      spec.plan.linearExtension.length == spec.plan.requestedActions.length &&
      spec.plan.bindings == setup &&
      spec.plan.symbolicRoles == [] &&
      spec.plan.expandedBounds == bounds &&
      spec.plan.selectionReason == .satisfyingWitness &&
      spec.plan.checkpoints.length == 1 &&
      spec.plan.omissions == canonicalPlannerOmissions &&
      spec.properties.map PortableProperty.identity == [property.id]

/-! A selected trace is compiled into an inspectable plan that separates requests from outcomes. -/
example : selectedArtifactIsInspectable := by
  native_decide

def optionalBehavior : CheckedBehavior := {
  behavior with
  requiredOccurrences := []
  semanticDigest := "behavior/optional-v1"
}

/-! The linear extension contains every selected action, including optional occurrences. -/
example :
    ((run 2 (.select [property]) .shortest 10 17 false optionalBehavior).artifact.map fun spec =>
      (spec.plan.linearExtension.length,
        spec.plan.linearExtension.map PlannedOccurrence.action)) =
      some (1, [request]) := by
  native_decide

def targetRelativeEmptyBehavior : CheckedBehavior := {
  behavior with
  actionsExactly := some [request, request]
  semanticDigest := "behavior/target-relative-empty-v1"
}

/-! Exhaustive completion with no Behavior-admitted target trace is unsatisfiable, not proof. -/
example :
    let planned := run 0 (.verify property) .exhaustive 10 17 true targetRelativeEmptyBehavior
    (planned.result.outcome.name, planned.result.isVerified,
      planned.result.metadata.completeness.established) =
      ("unsatisfiable", false, false) := by
  native_decide

def staticallyUnsatisfiableBehavior : CheckedBehavior := {
  behavior with
  spaceStatus := .unsatisfiable
  semanticDigest := "behavior/statically-unsatisfiable-v1"
}

/-! Empty behavior is unsatisfiable, while an incomplete search is budget exhaustion; neither
can be observed as verification. -/
example :
    let empty := run 0 (.verify property) .exhaustive 10 17 true staticallyUnsatisfiableBehavior
    let exhausted := run 64 (.counterexample property) .shortest 1 17 false
    (empty.result.outcome.name, empty.result.isVerified,
      exhausted.result.outcome.name, exhausted.result.isVerified) =
      ("unsatisfiable", false, "budget-exhausted", false) := by
  native_decide

/-! Independent planning and rendering of semantically identical checked inputs is byte-identical. -/
example :
    witnessSpec.map canonicalExperimentSpecJson =
      incidentalWitnessSpec.map canonicalExperimentSpecJson := by
  native_decide

/-! A meaning-bearing Query input is part of the artifact semantic identity. -/
example :
    witnessSpec.map ExperimentSpec.semanticIdentity !=
      (witnessSpec 18).map ExperimentSpec.semanticIdentity := by
  native_decide

/-!
The cursor instrumentation catches eager full-space production: a two-candidate budget over a
high-branching step pulls the root and one child, retains no pending candidates, and cannot
materialize siblings or upgrade the exhausted prefix into completeness.
-/
example :
    let planned := run 64 (.counterexample property) .shortest 2 17 false
    (planned.result.outcome.name, planned.result.metadata.completeness.established,
      planned.instrumentation.generatedCandidates,
      planned.instrumentation.retainedPendingCandidates,
      planned.instrumentation.peakActiveFrontierDepth,
      planned.instrumentation.stepKernelPulls,
      planned.result.metadata.explored.transitions) =
    ("budget-exhausted", false, 2, 0, 2, 1, 1) := by
  native_decide

end Temporal.Experiment.PlannerTests
