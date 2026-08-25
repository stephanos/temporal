import Temporal.Experiment.Query

namespace Temporal.Experiment.QueryTests

open Temporal.Experiment

def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Temporal/Experiment/QueryTests.lean"
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

def exactTrace (outcome : SemanticValue := acceptedValue) : BehaviorTrace := {
  setup
  trace := {
    initialState := initial
    steps := [{
      selectedAction := requestValue
      modelOutcome := outcome
      resultingState := completed
      observations := [observedValue]
    }]
  }
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

def summaryOf
    (result : Except QueryError (CheckedQuery (fun _ => True))) :
    Option (QueryQuantifier × QueryClaim) :=
  result.toOption.map fun query => (query.quantifier, query.claim)

/-! Every public form fixes its quantifier and claim before planning. -/
example : [
    summaryOf (checkQuery exhaustiveContext
      (declaration (.verify checkedProperty) exhaustivePolicy)),
    summaryOf (checkQuery context (declaration (.witness checkedProperty))),
    summaryOf (checkQuery context (declaration (.counterexample checkedProperty))),
    summaryOf (checkQuery context (declaration (.select [checkedProperty])))
  ] = [
    some (.universal, .verifiedWithinBounds),
    some (.existential, .satisfyingWitness),
    some (.existential, .violatingCounterexample),
    some (.exploratory, .boundedSelection)
  ] := by
  native_decide

def errorKindOf
    (result : Except QueryError (CheckedQuery (fun _ => True))) : Option QueryErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

def incompleteContext (missing : CompletenessRequirement) :
    QueryCheckContext (fun _ => True) := {
  target := .incomplete target.id source [missing]
}

/-! Exhaustive mode fails closed for every missing finite or kernel-completeness obligation. -/
example : [
    .roleDomain,
    .actionDomain,
    .initialEnumeration,
    .stepEnumeration,
    .kernelRelation
  ].all (fun missing =>
    errorKindOf (checkQuery (incompleteContext missing)
      (declaration (.verify checkedProperty) exhaustivePolicy)) ==
        some .missingFiniteCompleteness) := by
  native_decide

def noFiniteDomains : QueryCheckContext (fun _ => True) := {
  target := .checked { target, completeness := none }
}

example : errorKindOf (checkQuery noFiniteDomains
    (declaration (.verify checkedProperty) exhaustivePolicy)) =
      some .missingFiniteCompleteness := by
  native_decide

/-! The checked planner input retains the exact certified domains, not only their digests. -/
example : ((checkQuery exhaustiveContext
    (declaration (.verify checkedProperty) exhaustivePolicy)).toOption.bind fun query =>
      query.completeness.map fun evidence =>
        (evidence.roleAssignments.length, evidence.actions.length)) = some (1, 1) := by
  native_decide

/-! Completeness follows the exhaustive strategy, not a particular query form. -/
example : [
    QueryForm.verify checkedProperty,
    .witness checkedProperty,
    .counterexample checkedProperty,
    .select [checkedProperty]
  ].all (fun form =>
    errorKindOf (checkQuery noFiniteDomains (declaration form exhaustivePolicy)) ==
      some .missingFiniteCompleteness) := by
  native_decide

def unsatisfiableBehavior : CheckedBehavior := {
  checkedBehavior with
  spaceStatus := .unsatisfiable
  semanticDigest := "behavior/unsatisfiable-v1"
}

def checkedUnsatisfiable : Option (CheckedQuery (fun _ => True)) :=
  (checkQuery context
    (declaration (.witness checkedProperty) searchPolicy unsatisfiableBehavior)).toOption

def explored : ExploredCounts := {
  setups := 2
  traces := 7
  transitions := 11
  propertyEvaluations := 7
}

/-! Empty behavior is unsatisfiable, while an incomplete search is budget exhaustion; neither
can be observed as verification. -/
example :
    (checkedUnsatisfiable.map (fun query => finalizePlanning query explored (.complete false))).map
        (fun result => (result.outcome.name, result.isVerified)) =
      some ("unsatisfiable", false) := by
  native_decide

def checkedWitness : Option (CheckedQuery (fun _ => True)) :=
  (checkQuery context (declaration (.witness checkedProperty))).toOption

example : checkedWitness.map (fun query =>
    let result := finalizePlanning query explored .budgetExhausted
    (result.outcome.name, result.isVerified, result.metadata.completeness.established,
      result.metadata.explored.traces)) =
      some ("budget-exhausted", false, false, 7) := by
  native_decide

def checkedExhaustiveWitness : Option (CheckedQuery (fun _ => True)) :=
  (checkQuery exhaustiveContext
    (declaration (.witness checkedProperty) exhaustivePolicy)).toOption

/-! Complete absence and exhausted effort remain distinct while retaining counts and bounds. -/
example : checkedExhaustiveWitness.map (fun query =>
    let absent := finalizePlanning query explored (.complete true)
    let exhausted := finalizePlanning query explored .budgetExhausted
    (absent.outcome.name, absent.metadata.completeness.established,
      absent.metadata.explored.traces, absent.metadata.completeness.bounds,
      exhausted.outcome.name, exhausted.metadata.completeness.established)) =
      some ("no-such-trace-within-complete-bounds", true, 7, bounds,
        "budget-exhausted", false) := by
  native_decide

/-! A backend completion signal cannot manufacture proof from a non-exhaustive or empty space. -/
example :
    let nonExhaustive := checkedWitness.map fun query =>
      let result := finalizePlanning query explored (.complete true)
      (result.outcome.name, result.isVerified)
    let emptyVerification :=
      (checkQuery exhaustiveContext
        (declaration (.verify checkedProperty) exhaustivePolicy unsatisfiableBehavior)).toOption.map
          fun query =>
            let result := finalizePlanning query explored (.complete false)
            (result.outcome.name, result.isVerified)
    (nonExhaustive, emptyVerification) =
      (some ("budget-exhausted", false), some ("unsatisfiable", false)) := by
  native_decide

def canonicalOf
    (queryContext : QueryCheckContext (fun _ => True))
    (queryDeclaration : QueryDeclaration) : Option String :=
  (checkQuery queryContext queryDeclaration).toOption.map canonicalQueryJson

def digestOf
    (queryContext : QueryCheckContext (fun _ => True))
    (queryDeclaration : QueryDeclaration) : Option String :=
  (checkQuery queryContext queryDeclaration).toOption.map CheckedQuery.semanticDigest

def reorderedTarget : QueryTarget (fun _ => True) := {
  target with declarations := target.declarations.reverse
}

def incidentalContext : QueryCheckContext (fun _ => True) := {
  target := .checked { target := reorderedTarget, completeness := none }
}

def incidentalDeclaration : QueryDeclaration := {
  declaration (.witness { checkedProperty with documentation := "changed docs" }) with
  behavior := { checkedBehavior with documentation := "changed docs" }
  documentation := "changed query docs"
}

example : canonicalOf context (declaration (.witness checkedProperty)) =
    canonicalOf incidentalContext incidentalDeclaration := by
  native_decide

def changedTarget
    (digest : String := target.semanticDigest)
    (kernelDigest : String := kernel.metadata.contractDigest)
    (composition : List DeclarationId := []) : QueryTarget (fun _ => True) := {
  target with
  semanticDigest := digest
  requiredCapabilities := composition
  kernel := { kernel with metadata := { kernel.metadata with contractDigest := kernelDigest } }
}

def contextFor (candidate : QueryTarget (fun _ => True)) : QueryCheckContext (fun _ => True) := {
  target := .checked { target := candidate, completeness := none }
}

def changedProperty : CheckedProperty := {
  checkedProperty with semanticDigest := "property/v2"
}

def changedBehavior : CheckedBehavior := {
  checkedBehavior with semanticDigest := "behavior/v2"
}

def changedBounds : QueryBounds := {
  bounds with behavior := { bounds.behavior with transitions := { value := 2, unit := .semanticTransitions } }
}

def changedBoundsDeclaration : QueryDeclaration := {
  declaration (.witness checkedProperty) with bounds := changedBounds
}

def changedStrategyDeclaration : QueryDeclaration := {
  declaration (.witness checkedProperty) with policy := { searchPolicy with strategy := .breadthFirst }
}

def changedSeedDeclaration : QueryDeclaration := {
  declaration (.witness checkedProperty) with policy := { searchPolicy with seed := 18 }
}

/-! Every consumed semantic input changes Query identity. -/
example :
    let baseline := digestOf context (declaration (.witness checkedProperty))
    [
      digestOf context (declaration (.witness changedProperty)),
      digestOf context (declaration (.witness checkedProperty) searchPolicy changedBehavior),
      digestOf context changedBoundsDeclaration,
      digestOf context changedStrategyDeclaration,
      digestOf context changedSeedDeclaration,
      digestOf (contextFor (changedTarget "target/v2"))
        (declaration (.witness checkedProperty)),
      digestOf (contextFor (changedTarget (composition := [id "query.capability.extra"])))
        (declaration (.witness checkedProperty)),
      digestOf (contextFor (changedTarget (kernelDigest := "query-kernel/v2")))
        (declaration (.witness checkedProperty))
    ].all (fun changed => changed.isSome && changed != baseline) := by
  native_decide

def invalidExactBehavior : CheckedBehavior := {
  checkedBehavior with
  traceExactly := some (exactTrace (value accepted "not-admitted"))
  semanticDigest := "behavior/invalid-exact-v1"
}

/-! Structural exactness is insufficient: the selected kernel must admit the complete step. -/
example : errorKindOf (checkQuery context
    (declaration (.select [checkedProperty]) searchPolicy invalidExactBehavior)) =
      some .targetKernelMismatch := by
  native_decide

def backend : PlannerBackend Nat Nat Nat := {
  start := fun query => query
  pull := fun _ state =>
    if state == 0 then .complete else .yield state (state - 1)
}

/-! The backend exposes one candidate per pull and retains only its continuation state. -/
example : backend.pull 2 (backend.start 2) = PlannerPull.yield 2 1 := by
  native_decide

end Temporal.Experiment.QueryTests
