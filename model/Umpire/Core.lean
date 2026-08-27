import Lean.Data.Json
import Std

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
  contractDigest : String
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

inductive BoundUnit where
  | semanticTransitions
  | selectedActions
  | observationPositions
  | logicalTime
  deriving BEq, DecidableEq, Ord, Repr

def BoundUnit.name : BoundUnit → String
  | .semanticTransitions => "semantic-transitions"
  | .selectedActions => "selected-actions"
  | .observationPositions => "observation-positions"
  | .logicalTime => "logical-time"

structure TypedBound where
  value : Nat
  unit : BoundUnit
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

/-- Pure model data only. Execution evidence and qualification are deliberately absent. -/
structure ModelTrace (State Action Outcome Observation : Type) where
  initialState : State
  steps : List (ModelTraceStep State Action Outcome Observation)
  deriving BEq, DecidableEq, Repr

structure TransitionResult (State Outcome Observation : Type) where
  modelOutcome : Outcome
  resultingState : State
  observations : List Observation
  deriving BEq, DecidableEq, Repr

structure KernelMetadata where
  id : DefinitionId
  version : Nat := 1
  contractDigest : String
  source : SourceLocation
  deriving BEq, DecidableEq, Repr

/--
The target-owned finite transition kernel. The proof fields make every emitted initial state and
step sound, and make each authoritative relation complete with respect to the finite enumerators.
-/
structure TransitionKernel (Setup State Action Outcome Observation : Type) where
  metadata : KernelMetadata
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

/-- Missing proof obligations are representable only before target composition. -/
inductive KernelAvailability (Setup State Action Outcome Observation : Type) where
  | checked (kernel : TransitionKernel Setup State Action Outcome Observation)
  | incomplete (metadata : KernelMetadata) (missingProofs : List DefinitionId)

structure LawRequirement where
  id : DefinitionId
  semanticDigest : String
  deriving BEq, DecidableEq, Ord, Repr

/-- A law witness retains its portable Definition ID while proving the target's authoritative proposition. -/
structure LawWitness (LawStatement : DefinitionId → Prop) where
  requirement : LawRequirement
  proof : LawStatement requirement.id

structure CapabilityContract where
  id : DefinitionId
  version : Nat := 1
  semanticDigest : String
  requiredLaws : List LawRequirement
  deriving BEq, DecidableEq, Repr

structure MeaningProvision where
  definitionId : DefinitionId
  kind : DefinitionKind
  semanticDigest : String
  deriving BEq, DecidableEq, Repr

structure CapabilityProvider (LawStatement : DefinitionId → Prop) where
  id : DefinitionId
  source : SourceLocation
  contract : CapabilityContract
  meanings : List MeaningProvision
  lawWitnesses : List (LawWitness LawStatement)

structure Reconciliation where
  definitionId : DefinitionId
  kind : DefinitionKind
  providers : List DefinitionId
  semanticDigest : String
  deriving BEq, DecidableEq, Repr

structure CapabilityConnector (LawStatement : DefinitionId → Prop) where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  semanticDigest : String
  reconciliations : List Reconciliation
  requiredLaws : List LawRequirement
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

structure DefinitionError where
  kind : DefinitionErrorKind
  definitionId : DefinitionId
  sourcePath : String
  offendingValue : String
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr


def semanticDigestOf (canonicalSemanticValue : String) : String :=
  "umpire-semantic/v1:" ++ canonicalSemanticValue

private def quote (value : String) : String := Lean.Json.compress (.str value)

def canonicalTypedBoundJson (bound : TypedBound) : String :=
  "{\"value\":" ++ toString bound.value ++ ",\"unit\":" ++ quote bound.unit.name ++ "}"

end Umpire
