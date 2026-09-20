import Umpire.Command
import Umpire.Examples.Conventions

/-!
# The switch: Umpire's worked example

A two-position switch, declared through the Model commands and nothing else: an `entity`, three
`enum` domains, one step function, a `machine`, a `property`, two `scenario`s, `limits` and a
`query`. It names no feature and no platform. It is the Model every Umpire test that needs one
reads, so what this module exports is the vocabulary those tests were written against -- the
switch's values, Definition IDs, checked Property, Scenarios, Queries, planner runs and compiled
artifact -- each defined from what the commands declared rather than beside it.

Two of its three Queries are not commands. The exploratory Query picks among Properties and the
exact-trace Query pins one trace, and the `query` command declares neither form; both are checked
here over the command's own Model, Property and Scenarios, which is what an authored Query is.
-/

namespace Umpire.Examples.Switch

open Umpire
open Umpire.Command

/-! ### The Model -/

/-- The switch. There is one, and a flip acts on it. -/
entity subject

/-- Where the switch stands. -/
enum Position
  | off
  | on

/-- One switch's state: where it stands. -/
structure SwitchState where
  power : Position
  deriving BEq, DecidableEq, Repr, Finite

/-- What a flip did: it moved the switch, or it was deferred and the switch stayed. -/
enum FlipOutcome
  | applied
  | deferred

/-- What a flip records: the position the switch shows once the flip is done. -/
enum Power
  | off
  | on

/-- The one action: flip the switch. -/
action flip
  party: operator
  on: subject

def Position.flip : Position → Position
  | .off => .on
  | .on => .off

/-- The position a step shows. -/
def Power.showing : Position → Power
  | .off => .off
  | .on => .on

/-- A flip moves the switch and shows its new position, or is deferred and shows the old one. The
applied result comes first, so the shortest witness of a flip is the flip that took. -/
def flipStep (state : SwitchState) : List (Step SwitchState FlipOutcome Power) :=
  let moved : SwitchState := { power := state.power.flip }
  [{ outcome := .applied, state := moved, facts := [.showing moved.power] },
   { outcome := .deferred, state, facts := [.showing state.power] }]

machine twoState
  for: subject
  state: SwitchState
  starts: [off]
  ends: [on]
  steps:
    flip: flipStep

/- A selected flip turns the switch on. -/
property flipTurnsOn
  machine: twoState
  when: flip
  holds: fun step => step.state.power == .on

/- One flip from off, its outcome left to the Model. -/
scenario oneFlip
  model: twoState
  starts: off
  actions: [flip]

/- The same one flip, explored: the Scenario the exploratory Query picks over. -/
scenario explore
  model: twoState
  starts: off
  actions: [flip]

limits one
  steps: 1
  actions: 1
  search: 8

query exactAction
  find: flipTurnsOn
  in: oneFlip
  limits: one

/-! ### What the commands declared, under the names the tests read

Everything below is a view: a value the commands computed, or a record assembled from such values.
The checked exact-action Query is the one the `query` command admitted while this file elaborated;
its Model is `target`, and the other Queries, the planner runs and the artifact are read through
it. -/

/-- The admitted exact-action Query. The `query` command evaluated the admission and reported any
refusal at its own lines, so the proof only re-reads what it found. -/
@[irreducible] private def checked : Umpire.Command.CheckedModel twoState :=
  exactAction.toOption.get (by native_decide)

def LawStatement : Law → Prop := twoState.lawStatement

def source : SourceLocation := twoState.origin.source

def target : QueryModel LawStatement := checked.target

/-- The switch's members as model values, resolved by spelling the way a Case reads them. -/
def vocabulary : ModelVocabulary := checked.vocabulary

def offState : ModelValue := vocabulary.namedState "off"
def onState : ModelValue := vocabulary.namedState "on"
def flipAction : ModelValue := vocabulary.namedAction "flip"
def appliedOutcome : ModelValue := vocabulary.namedOutcome "applied"
def deferredOutcome : ModelValue := vocabulary.namedOutcome "deferred"
def powerOffObservation : ModelValue := vocabulary.namedFact "off"
def powerOnObservation : ModelValue := vocabulary.namedFact "on"

def targetId : DefinitionId := twoState.targetId
def kernelId : DefinitionId := twoState.kernelId
def switchCapabilityId : DefinitionId := twoState.capabilityId
def switchProviderId : DefinitionId := twoState.providerId
def flipLawId : DefinitionId := twoState.lawId
def switchRoleId : DefinitionId := twoState.operationRoleId
/-- The `power` field's own definition. A state is one value under its own definition, and it
holds its position under this one, so a Property that names `power` reads the position apart from
the state -- which is what the one state definition named while a state was one value. -/
def powerStateId : DefinitionId := (twoState.stateFieldIds.lookup "power").getD unknownId
def flipActionId : DefinitionId := flipAction.definitionId
def appliedOutcomeId : DefinitionId := appliedOutcome.definitionId
def deferredOutcomeId : DefinitionId := deferredOutcome.definitionId
/-- Each recorded position is its own fact definition under the commands; the tests that read the
switch's observation through one definition record the switch off, so this is that fact's. -/
def powerObservationId : DefinitionId := powerOffObservation.definitionId
/-- The two enumerated rows, each its own relation definition: the flip from off and the flip from
on. -/
def relationIds : List DefinitionId := twoState.relationIds
def offFlipRelationId : DefinitionId := (relationIds[0]?).getD unknownId
def onFlipRelationId : DefinitionId := (relationIds[1]?).getD unknownId
def flipPropertyId : DefinitionId := twoState.origin.family.id "property" "flipTurnsOn"
/-- The one occurrence the exact-action Scenario selects: its first, which is its flip. -/
def flipOccurrenceId : DefinitionId :=
  twoState.origin.family.id "occurrence" (oneFlip.names.declaration ++ ".1")
def exploratoryBehaviorId : DefinitionId := twoState.origin.family.id "behavior" "explore"
def exactActionBehaviorId : DefinitionId := twoState.origin.family.id "behavior" "oneFlip"
def exactTraceBehaviorId : DefinitionId := twoState.origin.family.id "behavior" "exactTrace"
def exploratoryQueryId : DefinitionId := twoState.origin.family.id "query" "explore"
def exactActionQueryId : DefinitionId := twoState.origin.family.id "query" "exactAction"
def exactTraceQueryId : DefinitionId := twoState.origin.family.id "query" "exactTrace"

def flipLaw : Law := twoState.law

theorem flipLawProof : LawStatement flipLaw := twoState.lawProof

def switchSetup : List RoleBinding := [{ role := switchRoleId, value := offState }]

def appliedResult : Step ModelValue ModelValue ModelValue := {
  outcome := appliedOutcome
  state := onState
  facts := [powerOnObservation]
}

def deferredResult : Step ModelValue ModelValue ModelValue := {
  outcome := deferredOutcome
  state := offState
  facts := [powerOffObservation]
}

def appliedFromOnResult : Step ModelValue ModelValue ModelValue := {
  outcome := appliedOutcome
  state := offState
  facts := [powerOffObservation]
}

def deferredFromOnResult : Step ModelValue ModelValue ModelValue := {
  outcome := deferredOutcome
  state := onState
  facts := [powerOnObservation]
}

theorem offState_ne_onState : offState ≠ onState := by
  native_decide

theorem onState_ne_offState : onState ≠ offState := by
  native_decide

theorem appliedResult_ordered :
    stepOrderKey appliedResult ≤ stepOrderKey deferredResult := by
  native_decide

theorem appliedFromOnResult_ordered :
    stepOrderKey appliedFromOnResult ≤ stepOrderKey deferredFromOnResult := by
  native_decide

/-! ### The Model as `Umpire.Search` reads it

The target's machine is the finite table's kernel. The records the ordinary Target authoring
boundary takes -- the machine, its spec, its providers and its finite planning -- are read off the
checked target, so a test that composes a variant of the switch through `DraftModel.make` composes
the same kernel the commands checked. -/

def machine : Machine (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  target.machine

def initialStates (setup : List RoleBinding) : List ModelValue :=
  machine.initialStates setup

def authoritativeInitial (setup : List RoleBinding) (state : ModelValue) : Prop :=
  machine.authoritativeInitial setup state

def stepResults
    (state action : ModelValue) :
    List (Step ModelValue ModelValue ModelValue) :=
  machine.steps state action

def authoritativeStep
    (state action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue) : Prop :=
  machine.authoritativeStep state action result

def definitions : List DefinitionMetadata := twoState.modelSpec.definitions

def modelProviders : Providers LawStatement := twoState.composition

private theorem providers_present : target.providers.head?.isSome = true := by
  native_decide

/-- The one provider the machine composes: the finite table, meaning every member. -/
def switchProvider : Provider LawStatement := target.providers.head?.get providers_present

def modelSpec : ModelSpec LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := target.id
  source := target.source
  definitions
  requiredCapabilities := target.requiredCapabilities
  resolvedSetups := target.resolvedSetups
  terminalConditions := target.terminalConditions
  machine := .checked target.machine
}

private theorem planning_available :
    (match target.planning with
      | .available _ => true
      | .unavailable => false) = true := by
  native_decide

def finitePlanning : FinitePlanningCapability machine.authoritativeStep :=
  match available : target.planning with
  | .available capability => capability
  | .unavailable => by
      have complete := planning_available
      simp [available] at complete

def targetAuthoring : DraftModel LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  DraftModel.make modelSpec modelProviders (.available machine rfl finitePlanning)

theorem target_resolvedSetups : target.resolvedSetups = [switchSetup] := by
  native_decide

/-! ### What the target admits

The kernel's domains are membership in the table's catalogs, and the catalogs are read off the
machine's complete vocabulary; each lemma says what the switch admits, which is what a proof over
the target's domains needs. -/

private def catalogs : Option (List (List RoleBinding) × List ModelValue × List ModelValue ×
    List ModelValue × List ModelValue) :=
  match target.machine.vocabulary with
  | .complete domain =>
      some (domain.setups, domain.states, domain.actions, domain.outcomes, domain.observations)
  | _ => none

private theorem catalogs_pinned : catalogs = some ([switchSetup], [offState, onState],
    [flipAction], [appliedOutcome, deferredOutcome],
    [powerOffObservation, powerOnObservation]) := by
  native_decide

private theorem catalogs_complete : ∃ domain, target.machine.vocabulary = .complete domain ∧
    domain.setups = [switchSetup] ∧ domain.states = [offState, onState] ∧
    domain.actions = [flipAction] ∧ domain.outcomes = [appliedOutcome, deferredOutcome] ∧
    domain.observations = [powerOffObservation, powerOnObservation] := by
  have pinned := catalogs_pinned
  unfold catalogs at pinned
  match vocabulary : target.machine.vocabulary with
  | .complete domain =>
      simp only [vocabulary, Option.some.injEq, Prod.mk.injEq] at pinned
      exact ⟨domain, rfl, pinned.1, pinned.2.1, pinned.2.2.1, pinned.2.2.2.1, pinned.2.2.2.2⟩
  | .missing => simp [vocabulary] at pinned
  | .incomplete _ => simp [vocabulary] at pinned

theorem target_setupDomain
    (value : List RoleBinding)
    (admitted : target.machine.setupDomain value) : value = switchSetup := by
  obtain ⟨domain, _, setups, _, _, _, _⟩ := catalogs_complete
  have member := domain.setupComplete value admitted
  rw [setups] at member
  simpa using member

theorem target_stateDomain
    (value : ModelValue)
    (admitted : target.machine.stateDomain value) : value = offState ∨ value = onState := by
  obtain ⟨domain, _, _, states, _, _, _⟩ := catalogs_complete
  have member := domain.stateComplete value admitted
  rw [states] at member
  simpa using member

theorem target_actionDomain
    (value : ModelValue)
    (admitted : target.machine.actionDomain value) : value = flipAction := by
  obtain ⟨domain, _, _, _, actions, _, _⟩ := catalogs_complete
  have member := domain.actionComplete value admitted
  rw [actions] at member
  simpa using member

theorem target_outcomeDomain
    (value : ModelValue)
    (admitted : target.machine.outcomeDomain value) :
    value = appliedOutcome ∨ value = deferredOutcome := by
  obtain ⟨domain, _, _, _, _, outcomes, _⟩ := catalogs_complete
  have member := domain.outcomeComplete value admitted
  rw [outcomes] at member
  simpa using member

theorem target_observationDomain
    (value : ModelValue)
    (admitted : target.machine.observationDomain value) :
    value = powerOffObservation ∨ value = powerOnObservation := by
  obtain ⟨domain, _, _, _, _, _, observations⟩ := catalogs_complete
  have member := domain.observationComplete value admitted
  rw [observations] at member
  simpa using member

theorem target_initialStates : target.machine.initialStates switchSetup = [offState] := by
  native_decide

theorem target_initial
    (setup : List RoleBinding)
    (state : ModelValue)
    (admitted : target.machine.authoritativeInitial setup state) :
    setup = switchSetup ∧ state = offState := by
  obtain ⟨domain, _, setups, _, _, _, _⟩ := catalogs_complete
  have member := target.machine.initialComplete setup state admitted
  have setupEq : setup = switchSetup := by
    have covered := domain.setupCoverage setup state member
    rw [setups] at covered
    simpa using covered
  subst setupEq
  rw [target_initialStates] at member
  exact ⟨rfl, by simpa using member⟩

theorem target_steps :
    target.machine.steps offState flipAction = [appliedResult, deferredResult] ∧
      target.machine.steps onState flipAction = [appliedFromOnResult, deferredFromOnResult] := by
  native_decide

theorem target_step
    (state action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue)
    (admitted : target.machine.authoritativeStep state action result) :
    action = flipAction ∧
      ((state = offState ∧ (result = appliedResult ∨ result = deferredResult)) ∨
        (state = onState ∧
          (result = appliedFromOnResult ∨ result = deferredFromOnResult))) := by
  obtain ⟨domain, _, _, states, actions, _, _⟩ := catalogs_complete
  have member := target.machine.stepComplete state action result admitted
  have actionEq : action = flipAction := by
    have covered := domain.actionCoverage state action result member
    rw [actions] at covered
    simpa using covered
  subst actionEq
  have sourceMember := domain.transitionSourceCoverage state flipAction result member
  rw [states] at sourceMember
  refine ⟨rfl, ?_⟩
  simp only [List.mem_cons, List.not_mem_nil, or_false] at sourceMember
  rcases sourceMember with rfl | rfl
  · rw [target_steps.1] at member
    exact .inl ⟨rfl, by simpa using member⟩
  · rw [target_steps.2] at member
    exact .inr ⟨rfl, by simpa using member⟩

theorem target_off_flip_applied_authoritative :
    target.machine.authoritativeStep offState flipAction appliedResult :=
  target.machine.stepSound offState flipAction appliedResult (by rw [target_steps.1]; simp)

/-! ### The Property, the Scenarios and the Queries -/

def authoredProperty : Property := flipTurnsOn vocabulary

def propertyResult : Except PropertyError CheckedProperty :=
  Property.check (PropertyCheckContext.ofTarget target) authoredProperty

def flipProperty : CheckedProperty := checked.property

def switchRole : Scenario.Role := { id := switchRoleId, valueKind := .state }

/-- The one setup constraint a Scenario of the switch carries: the subject starts off. -/
def setupConstraint : SetupConstraint :=
  SetupConstraint.roleEquals
    (twoState.origin.ownedId "setup" oneFlip.names.declaration twoState.roleName)
    switchRoleId offState

def exploratoryBehaviorDeclaration : Scenario := explore vocabulary

def exactActionBehaviorDeclaration : Scenario := oneFlip vocabulary

def exactTrace : AuthoredExactTrace := {
  setup := switchSetup
  initialState := some offState
  steps := [{
    selectedAction := some flipAction
    outcome := some appliedOutcome
    resultingState := some onState
    observations := some appliedResult.facts
  }]
}

/-- The exact-action Scenario pinned to the applied flip: the one form the `scenario` command does
not declare. -/
def exactTraceBehaviorDeclaration : Scenario := {
  exactActionBehaviorDeclaration with
  id := exactTraceBehaviorId
  traceExactly := some exactTrace
}

private def checkBehaviorDeclaration
    (declaration : Scenario) : Except ScenarioError CheckedScenario :=
  Scenario.check (.ofTarget target) declaration

def exploratoryBehaviorResult : Except ScenarioError CheckedScenario :=
  checkBehaviorDeclaration exploratoryBehaviorDeclaration
def exactActionBehaviorResult : Except ScenarioError CheckedScenario :=
  checkBehaviorDeclaration exactActionBehaviorDeclaration
def exactTraceBehaviorResult : Except ScenarioError CheckedScenario :=
  checkBehaviorDeclaration exactTraceBehaviorDeclaration

def exploratoryBehavior : CheckedScenario :=
  Scenario.checked (.ofTarget target) exploratoryBehaviorDeclaration

def exactActionBehavior : CheckedScenario := checked.behavior

def exactTraceBehavior : CheckedScenario :=
  Scenario.checked (.ofTarget target) exactTraceBehaviorDeclaration

def appliedTrace : Scenario.Trace :=
  Scenario.Trace.singleStep switchSetup offState flipAction appliedResult

def deferredTrace : Scenario.Trace :=
  Scenario.Trace.singleStep switchSetup offState flipAction deferredResult

def limits : Limits := one

def shortestPolicy : PlannerPolicy := PlannerPolicy.shortest

def queryContext : QueryCheckContext LawStatement := .ofTarget target

private def authoredQuery
    (queryId : DefinitionId)
    (form : Query.Form)
    (behavior : CheckedScenario) : Query := {
  id := queryId
  source
  target := target.id
  form
  behavior
  limits
  policy := shortestPolicy
}

def exploratoryQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  Query.check queryContext
    (authoredQuery exploratoryQueryId (.pick [flipProperty]) exploratoryBehavior)

def exactActionQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  Query.check queryContext
    (authoredQuery exactActionQueryId (.find flipProperty) exactActionBehavior)

def exactTraceQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  Query.check queryContext
    (authoredQuery exactTraceQueryId (.find flipProperty) exactTraceBehavior)

/-- The exploratory Query picks among its Properties over the explored flip; `pick` is a form the
`query` command does not declare. -/
def exploratoryQuery : CheckedQuery LawStatement :=
  Query.checked target
    (authoredQuery exploratoryQueryId (.pick [flipProperty]) exploratoryBehavior)

/-- The command's own Query, re-addressed to `target` so that a proof that it searches `target`
is `rfl`; its finite completeness is the same target's. -/
def exactActionQuery : CheckedQuery LawStatement := {
  checked.query with
  target
  completeness := (ModelCompleteness.ofTarget target).completeness
}

def exactTraceQuery : CheckedQuery LawStatement :=
  Query.checked target
    (authoredQuery exactTraceQueryId (.find flipProperty) exactTraceBehavior)

/-- The exact-action Query admitted against the switch Model, with its search view. The exploratory
and exact-trace Queries search through the same view. -/
def exactActionAdmitted : AdmittedQuery target :=
  (Search.admit target authoredProperty (some exactActionBehaviorDeclaration) {
    id := exactActionQueryId
    source
    target := target.id
    form := .find
    limits
    policy := shortestPolicy
  }).toOption.get (by native_decide)

theorem exploratoryQuery_target : exploratoryQuery.target = target := by rfl
theorem exactActionQuery_target : exactActionQuery.target = target := by rfl
theorem exactTraceQuery_target : exactTraceQuery.target = target := by rfl

def exploratoryRun : Except KnownGapError PlanResult :=
  (exactActionAdmitted.withQuery exploratoryQuery exploratoryQuery_target).search

def exactActionRunResult : Except KnownGapError PlanResult :=
  exactActionAdmitted.search

def exactTraceRun : Except KnownGapError PlanResult :=
  (exactActionAdmitted.withQuery exactTraceQuery exactTraceQuery_target).search

/-- The command's own planner run: the one that found the applied flip. -/
def exactActionRun : PlanResult := checked.run

def artifact : Option Plan := exactActionRun.artifact

private theorem artifact_isSome : artifact.isSome = true := by
  native_decide

def compiledArtifact : Plan := artifact.get artifact_isSome

end Umpire.Examples.Switch
