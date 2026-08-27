import Umpire.Query

/-! Shared semantic model, checked declarations, completeness evidence, and Query helpers. -/

namespace Umpire.QueryTests

open Umpire

def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Umpire/Query/Tests.lean"
  line := 1
  column := 1
  provenance := "lean-test"
}

def phase : DeclarationId := id "query.state.phase"
def request : DeclarationId := id "query.action.request"
def accepted : DeclarationId := id "query.outcome.accepted"
def observed : DeclarationId := id "query.observation.accepted"
def role : DeclarationId := id "query.role.operation"
def targetId : DeclarationId := id "query.target.fixture"
def kernelId : DeclarationId := id "query.kernel.fixture"
def extraCapabilityId : DeclarationId := id "query.capability.extra"
def extraProviderId : DeclarationId := id "query.provider.extra"

def metadata
    (identity : DeclarationId)
    (kind : DeclarationKind)
    (contractDigest : String) : DeclarationMetadata := {
  id := identity
  kind
  version := 1
  contractDigest
  source
  documentation := "query fixture"
}

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

def transition : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := acceptedValue
  resultingState := completed
  observations := [observedValue]
}

def kernel : TransitionKernel
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  metadata := {
    id := kernelId
    contractDigest := "query-kernel/v1"
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
}

def finitePlanning : FinitePlanningCapability kernel.authoritativeStep := {
  actions := [requestValue]
  roleDomainDigest := "role-domain/v1"
  actionDomainDigest := "action-domain/v1"
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
    semanticDigest := "query-extra-capability/v1"
    requiredLaws := []
  }
  meanings := []
  lawWitnesses := []
}

def targetDeclarations : List DeclarationMetadata := [
  metadata targetId .target "query-target/v1",
  metadata kernelId .kernel "query-kernel/v1",
  metadata extraCapabilityId .capability "query-extra-capability/v1",
  metadata extraProviderId .provider "query-extra-provider/v1"
]

def targetDefinition : TargetDefinition
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  id := targetId
  source
  declarations := targetDeclarations
  requiredCapabilities := []
  resolvedSetups := [setup]
  kernel := .checked kernel
}

def targetComposition : TargetComposition (fun _ => True) :=
  TargetComposition.empty |>.provide extraProvider

def targetAuthoring : AuthoredTarget (fun _ => True)
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue :=
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
  semanticDigest := "property/v1"
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
  semanticDigest := "behavior/v1"
}

def bounds : QueryBounds := {
  behavior := {
    transitions := { value := 1, unit := .semanticTransitions }
    selectedActions := { value := 1, unit := .selectedActions }
  }
  search := { value := 10, unit := .candidateEvaluations }
}

def exhaustivePolicy : PlannerPolicy := {
  strategy := .exhaustive
  seed := 17
  tieBreak := .semanticIdentity
}

def searchPolicy : PlannerPolicy := {
  strategy := .shortest
  seed := 17
  tieBreak := .semanticIdentity
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
  bounds
  policy
  documentation := "query documentation"
}

def errorKindOf
    (result : Except QueryError (CheckedQuery (fun _ => True))) : Option QueryErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

end Umpire.QueryTests
