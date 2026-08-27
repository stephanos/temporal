import Lean.Data.Json
import Std

namespace Umpire

/-! The common, pure semantic substrate shared by the Umpire authoring languages. -/

structure DeclarationId where
  value : String
  deriving BEq, DecidableEq, Hashable, Ord, Repr

namespace DeclarationId

def of (value : String) : DeclarationId := ⟨value⟩

private def isIdentifierCharacter (character : Char) : Bool :=
  character.isAlphanum || character == '-' || character == '_'

private def isNamespaceSegment (segment : String) : Bool :=
  segment != "" && segment.toList.all isIdentifierCharacter

def isNamespaced (id : DeclarationId) : Bool :=
  let segments := id.value.splitOn "."
  segments.length > 1 && segments.all isNamespaceSegment

end DeclarationId

inductive DeclarationKind where
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

def DeclarationKind.name : DeclarationKind → String
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

structure SemanticSource where
  path : String
  line : Nat := 0
  column : Nat := 0
  provenance : String := "authored"
  deriving BEq, DecidableEq, Repr

structure DeclarationMetadata where
  id : DeclarationId
  kind : DeclarationKind
  source : SemanticSource
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

structure SemanticValue where
  identity : DeclarationId
  value : String
  deriving BEq, DecidableEq, Ord, Repr

structure SemanticTraceStep (State Action Outcome Observation : Type) where
  selectedAction : Action
  modelOutcome : Outcome
  resultingState : State
  observations : List Observation
  deriving BEq, DecidableEq, Repr

/-- Pure model data only. Execution evidence and qualification are deliberately absent. -/
structure SemanticTrace (State Action Outcome Observation : Type) where
  initialState : State
  steps : List (SemanticTraceStep State Action Outcome Observation)
  deriving BEq, DecidableEq, Repr

structure TransitionResult (State Outcome Observation : Type) where
  modelOutcome : Outcome
  resultingState : State
  observations : List Observation
  deriving BEq, DecidableEq, Repr

structure KernelMetadata where
  id : DeclarationId
  version : Nat := 1
  contractDigest : String
  source : SemanticSource
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
  | incomplete (metadata : KernelMetadata) (missingProofs : List DeclarationId)

structure LawRequirement where
  id : DeclarationId
  semanticDigest : String
  deriving BEq, DecidableEq, Ord, Repr

/-- A law witness retains portable identity while proving the target's authoritative proposition. -/
structure LawWitness (LawStatement : DeclarationId → Prop) where
  requirement : LawRequirement
  proof : LawStatement requirement.id

structure CapabilityContract where
  id : DeclarationId
  version : Nat := 1
  semanticDigest : String
  requiredLaws : List LawRequirement
  deriving BEq, DecidableEq, Repr

structure MeaningProvision where
  declaration : DeclarationId
  kind : DeclarationKind
  semanticDigest : String
  deriving BEq, DecidableEq, Repr

structure CapabilityProvider (LawStatement : DeclarationId → Prop) where
  id : DeclarationId
  source : SemanticSource
  contract : CapabilityContract
  meanings : List MeaningProvision
  lawWitnesses : List (LawWitness LawStatement)

structure Reconciliation where
  declaration : DeclarationId
  kind : DeclarationKind
  providers : List DeclarationId
  semanticDigest : String
  deriving BEq, DecidableEq, Repr

structure CapabilityConnector (LawStatement : DeclarationId → Prop) where
  id : DeclarationId
  source : SemanticSource
  version : Nat := 1
  semanticDigest : String
  reconciliations : List Reconciliation
  requiredLaws : List LawRequirement
  lawWitnesses : List (LawWitness LawStatement)

inductive DeclarationErrorKind where
  | emptyIdentity
  | invalidIdentity
  | duplicateIdentity
  | unknownIdentity
  | wrongKind
  | missingLaw
  | unexpectedLaw
  | lawContractMismatch
  | missingProvider
  | conflictingProviders
  | ambiguousConnector
  | incompleteKernel
  deriving BEq, DecidableEq, Ord, Repr

def DeclarationErrorKind.name : DeclarationErrorKind → String
  | .emptyIdentity => "empty-identity"
  | .invalidIdentity => "invalid-identity"
  | .duplicateIdentity => "duplicate-identity"
  | .unknownIdentity => "unknown-identity"
  | .wrongKind => "wrong-kind"
  | .missingLaw => "missing-law"
  | .unexpectedLaw => "unexpected-law"
  | .lawContractMismatch => "law-contract-mismatch"
  | .missingProvider => "missing-provider"
  | .conflictingProviders => "conflicting-providers"
  | .ambiguousConnector => "ambiguous-connector"
  | .incompleteKernel => "incomplete-kernel"

structure DeclarationError where
  kind : DeclarationErrorKind
  declarationId : DeclarationId
  sourcePath : String
  offendingValue : String
  relatedIdentities : List DeclarationId
  deriving BEq, DecidableEq, Repr


def semanticDigestOf (canonicalSemanticValue : String) : String :=
  "umpire-semantic/v1:" ++ canonicalSemanticValue

private def quote (value : String) : String := Lean.Json.compress (.str value)

def canonicalTypedBoundJson (bound : TypedBound) : String :=
  "{\"value\":" ++ toString bound.value ++ ",\"unit\":" ++ quote bound.unit.name ++ "}"

end Umpire
