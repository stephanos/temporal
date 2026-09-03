import Lean.Data.Json
import Std
import Umpire.Fingerprint

namespace Umpire

/-! The common, pure model substrate shared by the Umpire authoring languages. -/

structure DefinitionId where
  value : String
  deriving BEq, DecidableEq, Hashable, Ord, Repr

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
  | observation
  | relation
  | capability
  | provider
  | law
  | connector
  | target
  | kernel
  | experimentSpace
  | variationAxis
  | choice
  | fault
  | coverageGoal
  deriving BEq, DecidableEq, Ord, Repr

def DefinitionKind.name : DefinitionKind → String
  | .state => "state"
  | .action => "action"
  | .outcome => "outcome"
  | .observation => "observation"
  | .relation => "relation"
  | .capability => "capability"
  | .provider => "provider"
  | .law => "law"
  | .connector => "connector"
  | .target => "target"
  | .kernel => "kernel"
  | .experimentSpace => "experiment-space"
  | .variationAxis => "variation-axis"
  | .choice => "choice"
  | .fault => "fault"
  | .coverageGoal => "coverage-goal"

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

structure ModelValue where
  definitionId : DefinitionId
  value : String
  deriving BEq, DecidableEq, Ord, Repr

/-- Construct a Model Value from an explicit Definition ID and value without validation or inference. -/
def ModelValue.named (definitionId : DefinitionId) (value : String) : ModelValue := {
  definitionId
  value
}

structure ModelTraceStep (State Action Outcome Observation : Type) where
  selectedAction : Action
  modelOutcome : Outcome
  resultingState : State
  observations : List Observation
  deriving BEq, DecidableEq, Repr

/-- One stable, one-based location of a Model Fact in a Model Trace. -/
inductive ModelCoordinate where
  | initialState
  | selectedAction (step : Nat)
  | modelOutcome (step : Nat)
  | resultingState (step : Nat)
  | observation (step position : Nat)
  deriving BEq, DecidableEq, Ord, Repr

/-- Pure model data only. Execution Evidence and Claim Assessment are deliberately absent. -/
structure ModelTrace (State Action Outcome Observation : Type) where
  initialState : State
  steps : List (ModelTraceStep State Action Outcome Observation)
  deriving BEq, DecidableEq, Repr

/-- Return the Definition kind selected by a Model Trace coordinate. -/
def ModelCoordinate.definitionKind : ModelCoordinate → DefinitionKind
  | .initialState | .resultingState _ => .state
  | .selectedAction _ => .action
  | .modelOutcome _ => .outcome
  | .observation _ _ => .observation

/-- Enumerate every Model Trace coordinate in canonical source order. -/
def ModelTrace.coordinates
    {State Action Outcome Observation : Type}
    (trace : ModelTrace State Action Outcome Observation) : List ModelCoordinate :=
  .initialState :: (trace.steps.mapIdx fun index step =>
    let stepPosition := index + 1
    [.selectedAction stepPosition, .modelOutcome stepPosition, .resultingState stepPosition] ++
      step.observations.mapIdx fun observationIndex _ =>
        .observation stepPosition (observationIndex + 1)).flatten

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
  | .modelOutcome step => do
      if step == 0 then none else
        let traceStep ← trace.steps[step - 1]?
        pure traceStep.modelOutcome
  | .resultingState step => do
      if step == 0 then none else
        let traceStep ← trace.steps[step - 1]?
        pure traceStep.resultingState
  | .observation step position => do
      if step == 0 || position == 0 then none else
        let traceStep ← trace.steps[step - 1]?
        traceStep.observations[position - 1]?

structure TransitionResult (State Outcome Observation : Type) where
  modelOutcome : Outcome
  resultingState : State
  observations : List Observation
  deriving BEq, DecidableEq, Repr

/-- Build one Model Trace step from its selected Action and model-owned transition result. -/
def ModelTraceStep.result
    {State Action Outcome Observation : Type}
    (selectedAction : Action)
    (result : TransitionResult State Outcome Observation) :
    ModelTraceStep State Action Outcome Observation := {
  selectedAction
  modelOutcome := result.modelOutcome
  resultingState := result.resultingState
  observations := result.observations
}

/-- A step built from a transition result retains the selected Action. -/
@[simp] theorem ModelTraceStep.result_selectedAction
    {State Action Outcome Observation : Type}
    (selectedAction : Action)
    (result : TransitionResult State Outcome Observation) :
    (ModelTraceStep.result selectedAction result).selectedAction = selectedAction := rfl

/-- A step built from a transition result retains its Model Outcome. -/
@[simp] theorem ModelTraceStep.result_modelOutcome
    {State Action Outcome Observation : Type}
    (selectedAction : Action)
    (result : TransitionResult State Outcome Observation) :
    (ModelTraceStep.result selectedAction result).modelOutcome = result.modelOutcome := rfl

/-- A step built from a transition result retains its resulting state. -/
@[simp] theorem ModelTraceStep.result_resultingState
    {State Action Outcome Observation : Type}
    (selectedAction : Action)
    (result : TransitionResult State Outcome Observation) :
    (ModelTraceStep.result selectedAction result).resultingState = result.resultingState := rfl

/-- A step built from a transition result retains its observations. -/
@[simp] theorem ModelTraceStep.result_observations
    {State Action Outcome Observation : Type}
    (selectedAction : Action)
    (result : TransitionResult State Outcome Observation) :
    (ModelTraceStep.result selectedAction result).observations = result.observations := rfl

/-- Map each semantic component of a transition result without changing its structure. -/
def TransitionResult.map
    {State Outcome Observation MappedState MappedOutcome MappedObservation : Type}
    (result : TransitionResult State Outcome Observation)
    (mapState : State → MappedState)
    (mapOutcome : Outcome → MappedOutcome)
    (mapObservation : Observation → MappedObservation) :
    TransitionResult MappedState MappedOutcome MappedObservation := {
  modelOutcome := mapOutcome result.modelOutcome
  resultingState := mapState result.resultingState
  observations := result.observations.map mapObservation
}

/-- Mapping a transition result maps its Model Outcome. -/
@[simp] theorem TransitionResult.map_modelOutcome
    {State Outcome Observation MappedState MappedOutcome MappedObservation : Type}
    (result : TransitionResult State Outcome Observation)
    (mapState : State → MappedState)
    (mapOutcome : Outcome → MappedOutcome)
    (mapObservation : Observation → MappedObservation) :
    (result.map mapState mapOutcome mapObservation).modelOutcome =
      mapOutcome result.modelOutcome := rfl

/-- Mapping a transition result maps its resulting state. -/
@[simp] theorem TransitionResult.map_resultingState
    {State Outcome Observation MappedState MappedOutcome MappedObservation : Type}
    (result : TransitionResult State Outcome Observation)
    (mapState : State → MappedState)
    (mapOutcome : Outcome → MappedOutcome)
    (mapObservation : Observation → MappedObservation) :
    (result.map mapState mapOutcome mapObservation).resultingState =
      mapState result.resultingState := rfl

/-- Mapping a transition result maps its observations in their existing order. -/
@[simp] theorem TransitionResult.map_observations
    {State Outcome Observation MappedState MappedOutcome MappedObservation : Type}
    (result : TransitionResult State Outcome Observation)
    (mapState : State → MappedState)
    (mapOutcome : Outcome → MappedOutcome)
    (mapObservation : Observation → MappedObservation) :
    (result.map mapState mapOutcome mapObservation).observations =
      result.observations.map mapObservation := rfl

/-- Authoritative finite-domain predicates, exhaustive enumerators, and canonical encoders for one Target. -/
structure TargetBehaviorDomain
    {Setup State Action Outcome Observation : Type}
    (setupDomain : Setup → Prop)
    (stateDomain : State → Prop)
    (actionDomain : Action → Prop)
    (outcomeDomain : Outcome → Prop)
    (observationDomain : Observation → Prop)
    (initialStates : Setup → List State)
    (steps : State → Action → List (TransitionResult State Outcome Observation)) where
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
    result ∈ steps state action → result.resultingState ∈ states
  outcomeCoverage : ∀ state action result,
    result ∈ steps state action → result.modelOutcome ∈ outcomes
  observationCoverage : ∀ state action result value,
    result ∈ steps state action → value ∈ result.observations → value ∈ observations

/-- Missing or incomplete finite coverage remains representable until Target checking. -/
inductive TargetBehaviorDomainAvailability
    {Setup State Action Outcome Observation : Type}
    (setupDomain : Setup → Prop)
    (stateDomain : State → Prop)
    (actionDomain : Action → Prop)
    (outcomeDomain : Outcome → Prop)
    (observationDomain : Observation → Prop)
    (initialStates : Setup → List State)
    (steps : State → Action → List (TransitionResult State Outcome Observation)) where
  | missing
  | incomplete (missingCoverage : List DefinitionId)
  | complete (domain : TargetBehaviorDomain setupDomain stateDomain actionDomain outcomeDomain
      observationDomain initialStates steps)

structure KernelMetadata where
  id : DefinitionId
  version : Nat := 1
  source : SourceLocation
  deriving BEq, DecidableEq, Repr

/--
The target-owned finite transition kernel. The proof fields make every admitted domain value and
emitted initial state or step sound, and make the authoritative relations exhaustively enumerable.
-/
structure TransitionKernel (Setup State Action Outcome Observation : Type) where
  metadata : KernelMetadata
  setupDomain : Setup → Prop
  stateDomain : State → Prop
  actionDomain : Action → Prop
  outcomeDomain : Outcome → Prop
  observationDomain : Observation → Prop
  initialStates : Setup → List State
  authoritativeInitial : Setup → State → Prop
  initialSound : ∀ setup state, state ∈ initialStates setup → authoritativeInitial setup state
  initialComplete : ∀ setup state, authoritativeInitial setup state → state ∈ initialStates setup
  steps : State → Action → List (TransitionResult State Outcome Observation)
  authoritativeStep :
    State → Action → TransitionResult State Outcome Observation → Prop
  stepSound : ∀ state action result,
    result ∈ steps state action → authoritativeStep state action result
  stepComplete : ∀ state action result,
    authoritativeStep state action result → result ∈ steps state action
  behaviorDomain : TargetBehaviorDomainAvailability setupDomain stateDomain actionDomain
    outcomeDomain observationDomain initialStates steps := .missing

/-- Complete behavior domains prove that every enumerated kernel result remains in-domain. -/
structure TargetBehaviorClosure
    {Setup State Action Outcome Observation : Type}
    (kernel : TransitionKernel Setup State Action Outcome Observation)
    (domain : TargetBehaviorDomain kernel.setupDomain kernel.stateDomain kernel.actionDomain
      kernel.outcomeDomain kernel.observationDomain kernel.initialStates kernel.steps) : Prop where
  initialState : ∀ setup state,
    state ∈ kernel.initialStates setup → state ∈ domain.states
  resultingState : ∀ state action result,
    result ∈ kernel.steps state action → result.resultingState ∈ domain.states
  outcome : ∀ state action result,
    result ∈ kernel.steps state action → result.modelOutcome ∈ domain.outcomes
  observation : ∀ state action result value,
    result ∈ kernel.steps state action → value ∈ result.observations → value ∈ domain.observations

theorem TargetBehaviorDomain.closure
    (kernel : TransitionKernel Setup State Action Outcome Observation)
    (domain : TargetBehaviorDomain kernel.setupDomain kernel.stateDomain kernel.actionDomain
      kernel.outcomeDomain kernel.observationDomain kernel.initialStates kernel.steps) :
    TargetBehaviorClosure kernel domain := {
  initialState := domain.initialStateCoverage
  resultingState := domain.resultingStateCoverage
  outcome := domain.outcomeCoverage
  observation := domain.observationCoverage
}

/-- Missing proof obligations are representable only before target composition. -/
inductive KernelAvailability (Setup State Action Outcome Observation : Type) where
  | checked (kernel : TransitionKernel Setup State Action Outcome Observation)
  | incomplete (metadata : KernelMetadata) (missingProofs : List DefinitionId)

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
