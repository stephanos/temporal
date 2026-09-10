import Umpire.Core

/-!
Pure Target inputs, behavior rows, and relation-indexed finite planning contracts.
Occurrence and diagnostic rows contain only inert source data; captured syntax belongs to Frontend.
Checked construction and admission remain together in Semantics.
-/

namespace Umpire

structure TargetDeclaration
    (LawStatement : LawDefinition → Prop)
    (Setup State Action Outcome Observation : Type) where
  id : DefinitionId
  source : SourceLocation
  definitions : List DefinitionMetadata
  requiredCapabilities : List DefinitionId
  providers : List (CapabilityProvider LawStatement)
  connectors : List (CapabilityConnector LawStatement)
  resolvedSetups : List Setup
  /-- One eligible-state set per constituent; every set must match. Empty metadata never closes. -/
  terminalConditions : List (List State) := []
  kernel : MachineAvailability Setup State Action Outcome Observation

/-- Optional finite planning is tied propositionally to the exact authoritative target kernel. -/
structure FinitePlanningCapability
    {State Action Outcome Observation : Type}
    (authoritativeStep : State → Action → Step State Outcome Observation → Prop) where
  actions : List Action
  actionSound : ∀ action, action ∈ actions →
    ∃ state result, authoritativeStep state action result
  actionComplete : ∀ state action result,
    authoritativeStep state action result → action ∈ actions

inductive FinitePlanningAvailability
    {State Action Outcome Observation : Type}
    (authoritativeStep : State → Action → Step State Outcome Observation → Prop) where
  | unavailable
  | available (capability : FinitePlanningCapability authoritativeStep)

inductive AuthoredPlanningCapability
    {Setup State Action Outcome Observation : Type}
    (availability : MachineAvailability Setup State Action Outcome Observation) where
  | unavailable
  | available
      (kernel : Machine Setup State Action Outcome Observation)
      (kernelEq : availability = .checked kernel)
      (capability : FinitePlanningCapability kernel.authoritativeStep)

structure TargetInitialStateRow where
  setup : String
  state : String
  deriving BEq, DecidableEq, Ord, Repr

structure TargetTransitionRow where
  state : String
  action : String
  modelOutcome : String
  resultingState : String
  observations : List String
  deriving BEq, DecidableEq, Ord, Repr

/-- Canonical executable behavior evaluated over the complete finite Target Behavior Domain. -/
structure TargetBehaviorDescription where
  setups : List String
  states : List String
  actions : List String
  outcomes : List String
  observations : List String
  initialStates : List TargetInitialStateRow
  transitions : List TargetTransitionRow
  terminalConditions : List (List String) := []
  deriving BEq, DecidableEq, Repr

/-- Closed authoring roles keep compiler locations separate from Model Definition IDs. -/
inductive AuthoringOccurrenceRole where
  | definitionMetadata
  | targetDefinition
  | providerDefinition
  | providerReference
  | connectorDefinition
  | connectorReference
  | capabilityRequirement
  | lawRequirement
  | lawWitness
  | meaning
  | reconciliation
  | kernel
  deriving BEq, DecidableEq, Ord, Repr

def AuthoringOccurrenceRole.name : AuthoringOccurrenceRole → String
  | .definitionMetadata => "definition-metadata"
  | .targetDefinition => "target-definition"
  | .providerDefinition => "provider-definition"
  | .providerReference => "provider-reference"
  | .connectorDefinition => "connector-definition"
  | .connectorReference => "connector-reference"
  | .capabilityRequirement => "capability-requirement"
  | .lawRequirement => "law-requirement"
  | .lawWitness => "law-witness"
  | .meaning => "meaning"
  | .reconciliation => "reconciliation"
  | .kernel => "kernel"

/-- The owner makes a nested occurrence path unambiguous when identities are reused. -/
inductive AuthoringOccurrenceContext where
  | direct
  | reconciliation (definitionId : DefinitionId)
  deriving BEq, DecidableEq, Repr

structure AuthoringOccurrencePath where
  role : AuthoringOccurrenceRole
  owner : DefinitionId
  context : AuthoringOccurrenceContext := .direct
  deriving BEq, DecidableEq, Repr

/-- Nonsemantic occurrence identity derived from a source span and its local ordinal. -/
structure AuthoringOccurrenceId where
  sourcePath : String
  line : Nat
  column : Nat
  endLine : Nat
  endColumn : Nat
  localOrdinal : Nat
  deriving BEq, DecidableEq, Repr

structure AuthoringOccurrence where
  id : AuthoringOccurrenceId
  definitionId : DefinitionId
  path : AuthoringOccurrencePath
  deriving BEq, DecidableEq, Repr

structure AuthoringDiagnostic where
  error : DefinitionError
  path : AuthoringOccurrencePath
  original : Option AuthoringOccurrenceId
  offending : AuthoringOccurrenceId
  deriving BEq, DecidableEq, Repr

/-- Ordinary target input keeps semantic definitions explicit without exposing checked fields. -/
structure TargetDefinition
    (Setup State Action Outcome Observation : Type) where
  id : DefinitionId
  source : SourceLocation
  definitions : List DefinitionMetadata
  requiredCapabilities : List DefinitionId
  resolvedSetups : List Setup
  /-- One eligible-state set per constituent; every set must match. Empty metadata never closes. -/
  terminalConditions : List (List State) := []
  kernel : MachineAvailability Setup State Action Outcome Observation

end Umpire
