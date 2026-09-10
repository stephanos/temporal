import Lean.Data.Json
import Std
import Umpire.Fingerprint
import Shared.SemanticData

namespace Umpire

/-! The common, pure model substrate shared by the Umpire authoring languages. -/

abbrev DefinitionId := Shared.SemanticData.Name
abbrev DefinitionId.mk := Shared.SemanticData.Name.mk
abbrev DefinitionId.value (id : DefinitionId) : String := Shared.SemanticData.Name.value id

namespace DefinitionId

def of (value : String) : DefinitionId := ⟨value⟩

private def isIdentifierCharacter (character : Char) : Bool :=
  character.isAlphanum || character == '-' || character == '_'

private def isNamespaceSegment (segment : String) : Bool :=
  segment != "" && segment.toList.all isIdentifierCharacter

def isNamespaced (id : DefinitionId) : Bool :=
  let segments := id.value.splitOn "."
  segments.length > 1 && segments.all isNamespaceSegment

/-- Structural failures shared by authoring languages when validating a Definition ID. -/
inductive ValidationError where
  | empty
  | malformed
  deriving BEq, DecidableEq, Repr

private def lessOrEqual (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

/-- Sort Definition IDs by their string values and remove duplicates. -/
def canonicalSet (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort lessOrEqual |>.eraseDups

private def firstAdjacentDuplicate : List DefinitionId → Option DefinitionId
  | first :: second :: rest =>
      if first == second then some first else firstAdjacentDuplicate (second :: rest)
  | _ => none

/-- Return the lexicographically smallest Definition ID that occurs more than once. -/
def firstDuplicate (ids : List DefinitionId) : Option DefinitionId :=
  firstAdjacentDuplicate (ids.mergeSort lessOrEqual)

/-- Validate the shared syntax of a Definition ID without constructing a language-specific error. -/
def validate (id : DefinitionId) : Except ValidationError Unit :=
  if id.value == "" then
    .error .empty
  else if !id.isNamespaced then
    .error .malformed
  else
    .ok ()

end DefinitionId

inductive DefinitionKind where
  | state
  | action
  | outcome
  | fact
  | relation
  | capability
  | provider
  | law
  | connector
  | target
  | machine
  deriving BEq, DecidableEq, Ord, Repr

def DefinitionKind.name : DefinitionKind → String
  | .state => "state"
  | .action => "action"
  | .outcome => "outcome"
  | .fact => "fact"
  | .relation => "relation"
  | .capability => "capability"
  | .provider => "provider"
  | .law => "law"
  | .connector => "connector"
  | .target => "target"
  | .machine => "machine"

structure SourceLocation where
  path : String
  line : Nat := 0
  column : Nat := 0
  provenance : String := "authored"
  deriving BEq, DecidableEq, Repr

/-- Return the authored source path, or the stable fallback when no path is available. -/
def SourceLocation.displayPath (source : SourceLocation) : String :=
  if source.path == "" then "<unknown>" else source.path

structure DefinitionMetadata where
  id : DefinitionId
  kind : DefinitionKind
  source : SourceLocation
  version : Nat := 1
  canonicalBehavior : String
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

inductive LimitUnit where
  | semanticTransitions
  | selectedActions
  | observationPositions
  | logicalTime
  | candidateEvaluations
  | experimentSpecs
  deriving BEq, DecidableEq, Ord, Repr

def LimitUnit.name : LimitUnit → String
  | .semanticTransitions => "semantic-transitions"
  | .selectedActions => "selected-actions"
  | .observationPositions => "observation-positions"
  | .logicalTime => "logical-time"
  | .candidateEvaluations => "candidate-evaluations"
  | .experimentSpecs => "experiment-specs"

structure Limit where
  value : Nat
  unit : LimitUnit
  deriving BEq, DecidableEq, Ord, Repr

abbrev ModelValue := Shared.SemanticData.Atom
abbrev ModelValue.mk := Shared.SemanticData.Atom.mk
abbrev ModelValue.definitionId (value : ModelValue) : DefinitionId := Shared.SemanticData.Atom.definitionId value
abbrev ModelValue.value (value : ModelValue) : String := Shared.SemanticData.Atom.value value

/-- Construct a Model Value from an explicit Definition ID and value without validation or inference. -/
def ModelValue.named (definitionId : DefinitionId) (value : String) : ModelValue := {
  definitionId
  value
}

structure RoleBinding where
  role : DefinitionId
  value : ModelValue
  deriving BEq, DecidableEq, Ord, Repr

structure ModelTraceStep (State Action Outcome Fact : Type) where
  selectedAction : Action
  outcome : Outcome
  state : State
  facts : List Fact
  deriving BEq, DecidableEq, Repr

/-- One stable, one-based location of a Model Fact in a Model Trace. -/
inductive ModelCoordinate where
  | initialState
  | selectedAction (step : Nat)
  | outcome (step : Nat)
  | state (step : Nat)
  | fact (step position : Nat)
  deriving BEq, DecidableEq, Ord, Repr

/-- Pure model data only. Execution Evidence and Claim Assessment are deliberately absent. -/
structure ModelTrace (State Action Outcome Observation : Type) where
  initialState : State
  steps : List (ModelTraceStep State Action Outcome Observation)
  deriving BEq, DecidableEq, Repr

/-- Return the Definition kind selected by a Model Trace coordinate. -/
def ModelCoordinate.definitionKind : ModelCoordinate → DefinitionKind
  | .initialState | .state _ => .state
  | .selectedAction _ => .action
  | .outcome _ => .outcome
  | .fact _ _ => .fact

/-- Enumerate every Model Trace coordinate in canonical source order. -/
def ModelTrace.coordinates
    {State Action Outcome Observation : Type}
    (trace : ModelTrace State Action Outcome Observation) : List ModelCoordinate :=
  .initialState :: (trace.steps.mapIdx fun index step =>
    let stepPosition := index + 1
    [.selectedAction stepPosition, .outcome stepPosition, .state stepPosition] ++
      step.facts.mapIdx fun factIndex _ =>
        .fact stepPosition (factIndex + 1)).flatten

/-- Look up a Model Value at a strict one-based coordinate, rejecting zero and out-of-range
positions. -/
def ModelTrace.valueAt?
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (coordinate : ModelCoordinate) : Option ModelValue :=
  match coordinate with
  | .initialState => some trace.initialState
  | .selectedAction step => do
      if step == 0 then none else
        let traceStep ← trace.steps[step - 1]?
        pure traceStep.selectedAction
  | .outcome step => do
      if step == 0 then none else
        let traceStep ← trace.steps[step - 1]?
        pure traceStep.outcome
  | .state step => do
      if step == 0 then none else
        let traceStep ← trace.steps[step - 1]?
        pure traceStep.state
  | .fact step position => do
      if step == 0 || position == 0 then none else
        let traceStep ← trace.steps[step - 1]?
        traceStep.facts[position - 1]?

abbrev Step := Shared.SemanticData.Result
abbrev Step.mk := @Shared.SemanticData.Result.mk
@[simp] abbrev Step.outcome {State Outcome Fact : Type}
    (step : Step State Outcome Fact) : Outcome :=
  Shared.SemanticData.Result.outcome step
@[simp] abbrev Step.state {State Outcome Fact : Type}
    (step : Step State Outcome Fact) : State :=
  Shared.SemanticData.Result.state step
@[simp] abbrev Step.facts {State Outcome Fact : Type}
    (step : Step State Outcome Fact) : List Fact :=
  Shared.SemanticData.Result.facts step

/-- Build one Model Trace step from its selected Action and model-owned Step. -/
def ModelTraceStep.result
    {State Action Outcome Fact : Type}
    (selectedAction : Action)
    (step : Step State Outcome Fact) :
    ModelTraceStep State Action Outcome Fact := {
  selectedAction
  outcome := step.outcome
  state := step.state
  facts := step.facts
}

/-- A step built from a transition result retains the selected Action. -/
@[simp] theorem ModelTraceStep.result_selectedAction
    {State Action Outcome Observation : Type}
    (selectedAction : Action)
    (result : Step State Outcome Observation) :
    (ModelTraceStep.result selectedAction result).selectedAction = selectedAction := rfl

/-- A step built from a transition result retains its Model Outcome. -/
@[simp] theorem ModelTraceStep.result_modelOutcome
    {State Action Outcome Observation : Type}
    (selectedAction : Action)
    (result : Step State Outcome Observation) :
    (ModelTraceStep.result selectedAction result).outcome = result.outcome := rfl

/-- A step built from a transition result retains its resulting state. -/
@[simp] theorem ModelTraceStep.result_resultingState
    {State Action Outcome Observation : Type}
    (selectedAction : Action)
    (result : Step State Outcome Observation) :
    (ModelTraceStep.result selectedAction result).state = result.state := rfl

/-- A step built from a transition result retains its observations. -/
@[simp] theorem ModelTraceStep.result_observations
    {State Action Outcome Observation : Type}
    (selectedAction : Action)
    (result : Step State Outcome Observation) :
    (ModelTraceStep.result selectedAction result).facts = result.facts := rfl

/-- Map each semantic component of a transition result without changing its structure. -/
def Step.map
    {State Outcome Observation MappedState MappedOutcome MappedObservation : Type}
    (result : Step State Outcome Observation)
    (mapState : State → MappedState)
    (mapOutcome : Outcome → MappedOutcome)
    (mapObservation : Observation → MappedObservation) :
    Step MappedState MappedOutcome MappedObservation :=
  match result with
  | ⟨outcome, state, observations⟩ => ⟨mapOutcome outcome, mapState state, observations.map mapObservation⟩

/-- Mapping a transition result maps its Model Outcome. -/
@[simp] theorem Step.map_modelOutcome
    {State Outcome Observation MappedState MappedOutcome MappedObservation : Type}
    (result : Step State Outcome Observation)
    (mapState : State → MappedState)
    (mapOutcome : Outcome → MappedOutcome)
    (mapObservation : Observation → MappedObservation) :
    (result.map mapState mapOutcome mapObservation).outcome =
      mapOutcome result.outcome := rfl

/-- Mapping a transition result maps its resulting state. -/
@[simp] theorem Step.map_resultingState
    {State Outcome Observation MappedState MappedOutcome MappedObservation : Type}
    (result : Step State Outcome Observation)
    (mapState : State → MappedState)
    (mapOutcome : Outcome → MappedOutcome)
    (mapObservation : Observation → MappedObservation) :
    (result.map mapState mapOutcome mapObservation).state =
      mapState result.state := rfl

/-- Mapping a transition result maps its observations in their existing order. -/
@[simp] theorem Step.map_observations
    {State Outcome Observation MappedState MappedOutcome MappedObservation : Type}
    (result : Step State Outcome Observation)
    (mapState : State → MappedState)
    (mapOutcome : Outcome → MappedOutcome)
    (mapObservation : Observation → MappedObservation) :
    (result.map mapState mapOutcome mapObservation).facts =
      result.facts.map mapObservation := rfl

/-- Authoritative finite-domain predicates, exhaustive enumerators, and canonical encoders for one Target. -/
structure TargetBehaviorDomain
    {Setup State Action Outcome Observation : Type}
    (setupDomain : Setup → Prop)
    (stateDomain : State → Prop)
    (actionDomain : Action → Prop)
    (outcomeDomain : Outcome → Prop)
    (observationDomain : Observation → Prop)
    (initialStates : Setup → List State)
    (steps : State → Action → List (Step State Outcome Observation)) where
  setups : List Setup
  states : List State
  actions : List Action
  outcomes : List Outcome
  observations : List Observation
  encodeSetup : Setup → String
  encodeState : State → String
  encodeAction : Action → String
  encodeOutcome : Outcome → String
  encodeObservation : Observation → String
  setupSound : ∀ setup, setup ∈ setups → setupDomain setup
  setupComplete : ∀ setup, setupDomain setup → setup ∈ setups
  stateSound : ∀ state, state ∈ states → stateDomain state
  stateComplete : ∀ state, stateDomain state → state ∈ states
  actionSound : ∀ action, action ∈ actions → actionDomain action
  actionComplete : ∀ action, actionDomain action → action ∈ actions
  outcomeSound : ∀ outcome, outcome ∈ outcomes → outcomeDomain outcome
  outcomeComplete : ∀ outcome, outcomeDomain outcome → outcome ∈ outcomes
  observationSound : ∀ observation, observation ∈ observations → observationDomain observation
  observationComplete : ∀ observation, observationDomain observation → observation ∈ observations
  setupCoverage : ∀ setup state, state ∈ initialStates setup → setup ∈ setups
  initialStateCoverage : ∀ setup state, state ∈ initialStates setup → state ∈ states
  transitionSourceCoverage : ∀ state action result,
    result ∈ steps state action → state ∈ states
  actionCoverage : ∀ state action result, result ∈ steps state action → action ∈ actions
  resultingStateCoverage : ∀ state action result,
    result ∈ steps state action → result.state ∈ states
  outcomeCoverage : ∀ state action result,
    result ∈ steps state action → result.outcome ∈ outcomes
  observationCoverage : ∀ state action result value,
    result ∈ steps state action → value ∈ result.facts → value ∈ observations

/-- Missing or incomplete finite coverage remains representable until Target checking. -/
inductive TargetBehaviorDomainAvailability
    {Setup State Action Outcome Observation : Type}
    (setupDomain : Setup → Prop)
    (stateDomain : State → Prop)
    (actionDomain : Action → Prop)
    (outcomeDomain : Outcome → Prop)
    (observationDomain : Observation → Prop)
    (initialStates : Setup → List State)
    (steps : State → Action → List (Step State Outcome Observation)) where
  | missing
  | incomplete (missingCoverage : List DefinitionId)
  | complete (domain : TargetBehaviorDomain setupDomain stateDomain actionDomain outcomeDomain
      observationDomain initialStates steps)

structure MachineMetadata where
  id : DefinitionId
  version : Nat := 1
  source : SourceLocation
  deriving BEq, DecidableEq, Repr

/--
The target-owned finite transition kernel. The proof fields make every admitted domain value and
emitted initial state or step sound, and make the authoritative relations exhaustively enumerable.
-/
structure Machine (Setup State Action Outcome Observation : Type) where
  metadata : MachineMetadata
  setupDomain : Setup → Prop
  stateDomain : State → Prop
  actionDomain : Action → Prop
  outcomeDomain : Outcome → Prop
  observationDomain : Observation → Prop
  initialStates : Setup → List State
  authoritativeInitial : Setup → State → Prop
  initialSound : ∀ setup state, state ∈ initialStates setup → authoritativeInitial setup state
  initialComplete : ∀ setup state, authoritativeInitial setup state → state ∈ initialStates setup
  steps : State → Action → List (Step State Outcome Observation)
  authoritativeStep :
    State → Action → Step State Outcome Observation → Prop
  stepSound : ∀ state action result,
    result ∈ steps state action → authoritativeStep state action result
  stepComplete : ∀ state action result,
    authoritativeStep state action result → result ∈ steps state action
  behaviorDomain : TargetBehaviorDomainAvailability setupDomain stateDomain actionDomain
    outcomeDomain observationDomain initialStates steps := .missing

/-- Complete behavior domains prove that every enumerated kernel result remains in-domain. -/
structure TargetBehaviorClosure
    {Setup State Action Outcome Observation : Type}
    (kernel : Machine Setup State Action Outcome Observation)
    (domain : TargetBehaviorDomain kernel.setupDomain kernel.stateDomain kernel.actionDomain
      kernel.outcomeDomain kernel.observationDomain kernel.initialStates kernel.steps) : Prop where
  initialState : ∀ setup state,
    state ∈ kernel.initialStates setup → state ∈ domain.states
  resultingState : ∀ state action result,
    result ∈ kernel.steps state action → result.state ∈ domain.states
  outcome : ∀ state action result,
    result ∈ kernel.steps state action → result.outcome ∈ domain.outcomes
  observation : ∀ state action result value,
    result ∈ kernel.steps state action → value ∈ result.facts → value ∈ domain.observations

theorem TargetBehaviorDomain.closure
    (kernel : Machine Setup State Action Outcome Observation)
    (domain : TargetBehaviorDomain kernel.setupDomain kernel.stateDomain kernel.actionDomain
      kernel.outcomeDomain kernel.observationDomain kernel.initialStates kernel.steps) :
    TargetBehaviorClosure kernel domain := {
  initialState := domain.initialStateCoverage
  resultingState := domain.resultingStateCoverage
  outcome := domain.outcomeCoverage
  observation := domain.observationCoverage
}

/-- Missing proof obligations are representable only before target composition. -/
inductive MachineAvailability (Setup State Action Outcome Observation : Type) where
  | checked (kernel : Machine Setup State Action Outcome Observation)
  | incomplete (metadata : MachineMetadata) (missingProofs : List DefinitionId)

structure LawDefinition where
  id : DefinitionId
  body : String
  deriving BEq, DecidableEq, Ord, Repr

/-- A law witness retains its portable definition while proving the proposition interpreted from its body. -/
structure LawWitness (LawStatement : LawDefinition → Prop) where
  definition : LawDefinition
  proof : LawStatement definition

structure CapabilityContract where
  id : DefinitionId
  version : Nat := 1
  canonicalBehavior : String
  requiredLaws : List LawDefinition
  deriving BEq, DecidableEq, Repr

structure MeaningProvision where
  definitionId : DefinitionId
  kind : DefinitionKind
  canonicalBehavior : String
  deriving BEq, DecidableEq, Repr

structure CapabilityProvider (LawStatement : LawDefinition → Prop) where
  id : DefinitionId
  source : SourceLocation
  contract : CapabilityContract
  meanings : List MeaningProvision
  lawWitnesses : List (LawWitness LawStatement)

structure Reconciliation where
  definitionId : DefinitionId
  kind : DefinitionKind
  providers : List DefinitionId
  canonicalBehavior : String
  deriving BEq, DecidableEq, Repr

structure CapabilityConnector (LawStatement : LawDefinition → Prop) where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  canonicalBehavior : String
  reconciliations : List Reconciliation
  requiredLaws : List LawDefinition
  lawWitnesses : List (LawWitness LawStatement)

inductive DefinitionErrorKind where
  | emptyDefinitionId
  | invalidDefinitionId
  | duplicateDefinitionId
  | unknownDefinitionId
  | wrongKind
  | missingLaw
  | unexpectedLaw
  | lawContractMismatch
  | missingProvider
  | conflictingProviders
  | ambiguousConnector
  | incompleteKernel
  | missingBehaviorDomain
  | incompleteBehaviorDomain
  deriving BEq, DecidableEq, Ord, Repr

def DefinitionErrorKind.name : DefinitionErrorKind → String
  | .emptyDefinitionId => "empty-definition-id"
  | .invalidDefinitionId => "invalid-definition-id"
  | .duplicateDefinitionId => "duplicate-definition-id"
  | .unknownDefinitionId => "unknown-definition-id"
  | .wrongKind => "wrong-kind"
  | .missingLaw => "missing-law"
  | .unexpectedLaw => "unexpected-law"
  | .lawContractMismatch => "law-contract-mismatch"
  | .missingProvider => "missing-provider"
  | .conflictingProviders => "conflicting-providers"
  | .ambiguousConnector => "ambiguous-connector"
  | .incompleteKernel => "incomplete-kernel"
  | .missingBehaviorDomain => "missing-behavior-domain"
  | .incompleteBehaviorDomain => "incomplete-behavior-domain"

structure DefinitionError where
  kind : DefinitionErrorKind
  definitionId : DefinitionId
  sourcePath : String
  offendingValue : String
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr


private def quote (value : String) : String := Lean.Json.compress (.str value)

def canonicalLimitJson (limit : Limit) : String :=
  "{\"value\":" ++ toString limit.value ++ ",\"unit\":" ++ quote limit.unit.name ++ "}"

end Umpire
