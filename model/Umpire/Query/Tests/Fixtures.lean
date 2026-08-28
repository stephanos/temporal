import Umpire.Query
import Umpire.Shared.Test

/-! Shared semantic model, checked definitions, completeness evidence, and Query helpers. -/

namespace Umpire.QueryTests

open Umpire

def id (value : String) : DefinitionId := Shared.Test.definitionId value

def source : SourceLocation := Shared.Test.sourceLocation "Umpire/Query/Tests.lean"

def phase : DefinitionId := id "query.state.phase"
def request : DefinitionId := id "query.action.request"
def accepted : DefinitionId := id "query.outcome.accepted"
def observed : DefinitionId := id "query.observation.accepted"
def role : DefinitionId := id "query.role.operation"
def targetId : DefinitionId := id "query.target.fixture"
def kernelId : DefinitionId := id "query.kernel.fixture"
def extraCapabilityId : DefinitionId := id "query.capability.extra"
def extraProviderId : DefinitionId := id "query.provider.extra"

def metadata
    (definitionId : DefinitionId)
    (kind : DefinitionKind)
    (canonicalBehavior : String) : DefinitionMetadata :=
  { Shared.Test.definitionMetadata definitionId.value kind source canonicalBehavior with
    id := definitionId
    documentation := "query fixture"
  }

def value (definitionId : DefinitionId) (payload : String) : ModelValue := {
  definitionId
  value := payload
}

def initial : ModelValue := value phase "initial"
def completed : ModelValue := value phase "completed"
def requestValue : ModelValue := value request "request"
def acceptedValue : ModelValue := value accepted "accepted"
def observedValue : ModelValue := value observed "accepted"
def setup : List RoleBinding := [{ role, value := value phase "operation-a" }]

private def encodeValue (modelValue : ModelValue) : String :=
  modelValue.definitionId.value ++ ":" ++ modelValue.value

private def encodeSetup (bindings : List RoleBinding) : String :=
  String.intercalate "|" (bindings.map fun binding =>
    binding.role.value ++ "=" ++ encodeValue binding.value)

def transition : TransitionResult ModelValue ModelValue ModelValue := {
  modelOutcome := acceptedValue
  resultingState := completed
  observations := [observedValue]
}

def kernel : TransitionKernel
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
    if state = initial ∧ action = requestValue then [transition] else []
  authoritativeStep := fun state action result =>
    state = initial ∧ action = requestValue ∧ result = transition
  stepSound := by
    intro state action result member
    by_cases selected : state = initial ∧ action = requestValue
    · rw [if_pos selected] at member
      exact ⟨selected.1, selected.2, List.mem_singleton.mp member⟩
    · rw [if_neg selected] at member
      exact (List.not_mem_nil member).elim
  stepComplete := by
    intro state action result admitted
    rcases admitted with ⟨rfl, rfl, rfl⟩
    simp
  behaviorDomain := .complete {
    setups := [setup]
    states := [initial, completed]
    actions := [requestValue]
    outcomes := [acceptedValue]
    observations := [observedValue]
    encodeSetup
    encodeState := encodeValue
    encodeAction := encodeValue
    encodeOutcome := encodeValue
    encodeObservation := encodeValue
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
        simp [List.mem_singleton.mp member, transition]
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
    outcomeCoverage := by
      intro state action result member
      by_cases selected : state = initial ∧ action = requestValue
      · rw [if_pos selected] at member
        simp [List.mem_singleton.mp member, transition]
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
    observationCoverage := by
      intro state action result observation member observationMember
      by_cases selected : state = initial ∧ action = requestValue
      · rw [if_pos selected] at member
        simp [List.mem_singleton.mp member, transition] at observationMember ⊢
        exact observationMember
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
  }
}

def finitePlanning : FinitePlanningCapability kernel.authoritativeStep := {
  actions := [requestValue]
  actionSound := by
    intro action member
    simp only [List.mem_cons, List.not_mem_nil, or_false] at member
    subst action
    exact ⟨initial, transition, rfl, rfl, rfl⟩
  actionComplete := by
    intro state action result admitted
    simp [admitted.2.1]
}

def extraProvider : CapabilityProvider (fun _ => True) := {
  id := extraProviderId
  source
  contract := {
    id := extraCapabilityId
    canonicalBehavior := "query-extra-capability/v1"
    requiredLaws := []
  }
  meanings := []
  lawWitnesses := []
}

def targetDefinitions : List DefinitionMetadata := [
  metadata targetId .target "query-target/v1",
  metadata kernelId .kernel "query-kernel/v1",
  metadata extraCapabilityId .capability "query-extra-capability/v1",
  metadata extraProviderId .provider "query-extra-provider/v1"
]

def targetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetId
  source
  definitions := targetDefinitions
  requiredCapabilities := []
  resolvedSetups := [setup]
  kernel := .checked kernel
}

def targetComposition : TargetComposition (fun _ => True) :=
  TargetComposition.empty |>.provide extraProvider

def targetAuthoring : AuthoredTarget (fun _ => True)
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  AuthoredTarget.make targetDefinition targetComposition
    (.available kernel rfl finitePlanning)

def target : QueryTarget (fun _ => True) := checkedTarget targetAuthoring

def targetWithoutPlanning : QueryTarget (fun _ => True) :=
  checkedTarget targetAuthoring.withoutPlanning

def checkedProperty : CheckedProperty := {
  id := id "query.property.fixture"
  source
  version := 1
  requires := []
  clauses := []
  access := { capabilities := [], meanings := [], logicalTimeSource := none }
  documentation := "property documentation"
  canonicalMetadata := "property-metadata"
  behaviorFingerprint := behaviorFingerprintOf "property/v1"
}

def checkedBehavior : CheckedBehavior := {
  id := id "query.behavior.fixture"
  source
  version := 1
  requires := []
  roles := [{ id := role, valueKind := .state }]
  setup := []
  allowedActions := [request]
  requiredOccurrences := []
  forbiddenActions := []
  occurrenceBounds := []
  ordering := []
  sequences := []
  adjacencies := []
  actionsExactly := none
  traceExactly := none
  spaceStatus := .unclassified
  documentation := "behavior documentation"
  canonicalMetadata := "behavior-metadata"
  behaviorFingerprint := behaviorFingerprintOf "behavior/v1"
}

def limits : QueryLimits := {
  behavior := {
    transitions := { value := 1, unit := .semanticTransitions }
    selectedActions := { value := 1, unit := .selectedActions }
  }
  search := { value := 10, unit := .candidateEvaluations }
}

def exhaustivePolicy : PlannerPolicy := {
  strategy := .exhaustive
  seed := 17
  tieBreak := .definitionId
}

def searchPolicy : PlannerPolicy := {
  strategy := .shortest
  seed := 17
  tieBreak := .definitionId
}

def context : QueryCheckContext (fun _ => True) :=
  .ofTarget targetWithoutPlanning

def exhaustiveContext : QueryCheckContext (fun _ => True) := .ofTarget target

def declaration
    (form : QueryForm)
    (policy : PlannerPolicy := searchPolicy)
    (behavior : CheckedBehavior := checkedBehavior) : QueryDeclaration := {
  id := id "query.declaration.fixture"
  source
  target := target.id
  form
  behavior
  limits
  policy
  documentation := "query documentation"
}

def errorKindOf
    (result : Except QueryError (CheckedQuery (fun _ => True))) : Option QueryErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

end Umpire.QueryTests
