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
def completed (index : Nat) := value phase ("completed-" ++ toString index)
def requestValue := value request "request"
def acceptedValue (index : Nat) := value accepted ("accepted-" ++ toString index)
def observedValue (index : Nat) := value observed ("accepted-" ++ toString index)
def setup : List RoleBinding := [{ role, value := value phase "operation-a" }]

def transition (index : Nat) : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := acceptedValue index
  resultingState := completed index
  observations := [observedValue index]
}

def transitions (width : Nat) : List (TransitionResult SemanticValue SemanticValue SemanticValue) :=
  transition 0 :: (List.range width).map fun index => transition (index + 1)

def kernel (width : Nat) : TransitionKernel
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  metadata := {
    id := id "planner.kernel.fixture"
    contractDigest := "planner-kernel/v1"
    source
  }
  initialStates := fun candidate => if candidate = setup then [initial] else []
  authoritativeInitial := fun candidate state =>
    state ∈ (if candidate = setup then [initial] else [])
  initialSound := by intros; assumption
  initialComplete := by intros; assumption
  steps := fun state action =>
    if state = initial ∧ action = requestValue then transitions width else []
  authoritativeStep := fun state action result =>
    result ∈ (if state = initial ∧ action = requestValue then transitions width else [])
  stepSound := by intros; assumption
  stepComplete := by intros; assumption
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
    exact ⟨initial, transition 0, by
      change transition 0 ∈
        (if initial = initial ∧ requestValue = requestValue then transitions width else [])
      simp [transitions]
    ⟩
  actionComplete := by
    intro state action result admitted
    change result ∈
      (if state = initial ∧ action = requestValue then transitions width else []) at admitted
    by_cases matched : state = initial ∧ action = requestValue
    · simp [matched.2]
    · simp [matched] at admitted
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

def declaration
    (form : QueryForm)
    (strategy : SearchStrategy)
    (budget : Nat := 10)
    (seed : Nat := 17) : QueryDeclaration := {
  id := id "planner.query.fixture"
  source
  target := (target 2).id
  form
  behavior
  bounds := bounds budget
  policy := policy strategy seed
  documentation := "query documentation"
}

def checkedQuery?
    (width : Nat)
    (form : QueryForm)
    (strategy : SearchStrategy)
    (budget : Nat := 10)
    (seed : Nat := 17) : Option (CheckedQuery (fun _ => True)) :=
  let context : QueryCheckContext (fun _ => True) := {
    target := .checked { target := target width, completeness := some (completeness width) }
  }
  (checkQuery context (declaration form strategy budget seed)).toOption

def run?
    (width : Nat)
    (form : QueryForm)
    (strategy : SearchStrategy)
    (budget : Nat := 10)
    (seed : Nat := 17) : Option PlannerRun :=
  (checkedQuery? width form strategy budget seed).map plan

def outcomeName?
    (form : QueryForm)
    (strategy : SearchStrategy) : Option String :=
  (run? 2 form strategy).map fun run => run.result.outcome.name

/-! Each Query form preserves its exact result semantics over the same deterministic kernel. -/
example : [
    outcomeName? (.verify property) .exhaustive,
    outcomeName? (.witness property) .shortest,
    outcomeName? (.counterexample property) .exhaustive,
    outcomeName? (.select [property]) .breadthFirst
  ] = [
    some "verified-within-bounds",
    some "found",
    some "no-such-trace-within-complete-bounds",
    some "found"
  ] := by
  native_decide

def witnessSpec? (seed : Nat := 17) : Option ExperimentSpec :=
  (run? 2 (.witness property) .shortest 10 seed).bind PlannerRun.artifact

def incidentalWitnessSpec? : Option ExperimentSpec :=
  (checkedQuery? 2 (.witness property) .shortest).bind fun query =>
    let incidental : CheckedQuery (fun _ => True) := {
      query with
      documentation := "changed query documentation"
      behavior := { query.behavior with documentation := "changed behavior documentation" }
      form := .witness { property with documentation := "changed property documentation" }
    }
    (plan incidental).artifact

def selectedArtifactIsInspectable : Bool :=
  match witnessSpec? with
  | none => false
  | some spec =>
      spec.plan.initialState == initial &&
      spec.plan.requestedActions == [requestValue] &&
      spec.plan.modelOutcomes == [acceptedValue 0] &&
      spec.plan.linearExtension == [occurrence] &&
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

/-! Independent planning and rendering of semantically identical checked inputs is byte-identical. -/
example :
    witnessSpec?.map canonicalExperimentSpecJson =
      incidentalWitnessSpec?.map canonicalExperimentSpecJson := by
  native_decide

/-! A meaning-bearing Query input is part of the artifact semantic identity. -/
example :
    witnessSpec?.map ExperimentSpec.semanticIdentity !=
      (witnessSpec? 18).map ExperimentSpec.semanticIdentity := by
  native_decide

/-!
The cursor instrumentation catches eager full-space production: a two-candidate budget over a
high-branching step generates the root and one child, retains no pending candidates, and cannot
upgrade the exhausted prefix into completeness.
-/
example : ((run? 64 (.counterexample property) .shortest 2).map fun run =>
    (run.result.outcome.name, run.result.metadata.completeness.established,
      run.instrumentation.generatedCandidates,
      run.instrumentation.retainedPendingCandidates,
      run.instrumentation.peakActiveFrontierDepth,
      run.result.metadata.explored.transitions)) =
    some ("budget-exhausted", false, 2, 0, 2, 1) := by
  native_decide

end Temporal.Experiment.PlannerTests
