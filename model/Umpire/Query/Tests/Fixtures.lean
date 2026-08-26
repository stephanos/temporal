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

def phase := id "query.state.phase"
def request := id "query.action.request"
def accepted := id "query.outcome.accepted"
def observed := id "query.observation.accepted"
def role := id "query.role.operation"

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

def transition : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := acceptedValue
  resultingState := completed
  observations := [observedValue]
}

def kernel : TransitionKernel
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  metadata := {
    id := id "query.kernel.fixture"
    contractDigest := "query-kernel/v1"
    source
  }
  initialStates := fun candidate => if candidate = setup then [initial] else []
  authoritativeInitial := fun candidate state => candidate = setup ∧ state = initial
  initialSound := by intros; split at * <;> simp_all
  initialComplete := by intros; simp_all
  steps := fun state action =>
    if state = initial ∧ action = requestValue then [transition] else []
  authoritativeStep := fun state action result =>
    state = initial ∧ action = requestValue ∧ result = transition
  stepSound := by intros; split at * <;> simp_all
  stepComplete := by intros; simp_all
}

def target : QueryTarget (fun _ => True) := {
  id := id "query.target.fixture"
  source
  declarations := []
  requiredCapabilities := []
  providers := []
  connectors := []
  resolvedSetups := [setup]
  kernel
  canonicalMetadata := "target-metadata"
  semanticDigest := "target/v1"
}

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

def completeness : FiniteCompletenessEvidence (fun _ => True) target := {
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
    exact ⟨initial, transition, rfl, rfl, rfl⟩
  actionComplete := by
    intro state action result admitted
    simp [admitted.2.1]
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

def context : QueryCheckContext (fun _ => True) := {
  target := .checked { target, completeness := none }
}

def exhaustiveContext : QueryCheckContext (fun _ => True) := {
  target := .checked { target, completeness := some completeness }
}

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
