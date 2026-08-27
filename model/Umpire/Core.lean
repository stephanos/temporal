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

structure SourceLocation where
  path : String
  line : Nat := 0
  column : Nat := 0
  provenance : String := "authored"
  deriving BEq, DecidableEq, Repr

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
  deriving BEq, DecidableEq, Ord, Repr

def LimitUnit.name : LimitUnit → String
  | .semanticTransitions => "semantic-transitions"
  | .selectedActions => "selected-actions"
  | .observationPositions => "observation-positions"
  | .logicalTime => "logical-time"
  | .candidateEvaluations => "candidate-evaluations"

structure Limit where
  value : Nat
  unit : LimitUnit
  deriving BEq, DecidableEq, Ord, Repr

structure ModelValue where
  definitionId : DefinitionId
  value : String
  deriving BEq, DecidableEq, Ord, Repr

structure ModelTraceStep (State Action Outcome Observation : Type) where
  selectedAction : Action
  modelOutcome : Outcome
  resultingState : State
  observations : List Observation
  deriving BEq, DecidableEq, Repr

/-- Pure model data only. Execution Evidence and Claim Assessment are deliberately absent. -/
structure ModelTrace (State Action Outcome Observation : Type) where
  initialState : State
  steps : List (ModelTraceStep State Action Outcome Observation)
  deriving BEq, DecidableEq, Repr

structure TransitionResult (State Outcome Observation : Type) where
  modelOutcome : Outcome
  resultingState : State
  observations : List Observation
  deriving BEq, DecidableEq, Repr

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
