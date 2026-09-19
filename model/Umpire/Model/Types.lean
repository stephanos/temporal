import Umpire.Core

/-!
Pure model inputs, behavior rows, and relation-indexed finite planning contracts.
Source-reference and diagnostic rows contain only inert source data; captured syntax belongs to
`Model.Elab`. Checked construction and admission remain together in `Model.Check`.
-/

namespace Umpire

/-- The author's model record. Providers and connectors default to empty so an ordinary model
states only its semantic definitions; `Providers` collects them when a model has any. -/
structure ModelSpec
    (LawStatement : Law → Prop)
    (Setup State Action Outcome Observation : Type) where
  id : DefinitionId
  source : SourceLocation
  definitions : List DefinitionMetadata
  requiredCapabilities : List DefinitionId
  providers : List (Provider LawStatement) := []
  connectors : List (Connector LawStatement) := []
  resolvedSetups : List Setup
  /-- One eligible-state set per constituent; every set must match. Empty metadata never closes. -/
  terminalConditions : List (List State) := []
  machine : MachineAvailability Setup State Action Outcome Observation

/-- Optional finite planning is tied propositionally to the exact authoritative model Machine. -/
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
      (machineEq : availability = .checked kernel)
      (capability : FinitePlanningCapability kernel.authoritativeStep)

structure BehaviorInitialStateRow where
  setup : String
  state : String
  deriving BEq, DecidableEq, Ord, Repr

structure BehaviorTransitionRow where
  priorState : String
  action : String
  outcome : String
  state : String
  facts : List String
  deriving BEq, DecidableEq, Ord, Repr

/-- Canonical executable behavior evaluated over the complete finite Target Behavior Domain. -/
structure BehaviorTable where
  setups : List String
  states : List String
  actions : List String
  outcomes : List String
  observations : List String
  initialStates : List BehaviorInitialStateRow
  transitions : List BehaviorTransitionRow
  terminalConditions : List (List String) := []
  deriving BEq, DecidableEq, Repr

/-- Closed authoring roles keep compiler locations separate from Model Definition IDs. -/
inductive SourceRefRole where
  | definitionMetadata
  | modelSpec
  | providerDefinition
  | providerReference
  | connectorDefinition
  | connectorReference
  | capabilityRequirement
  | lawRequirement
  | lawProof
  | meaning
  | reconciliation
  | machine
  deriving BEq, DecidableEq, Ord, Repr

def SourceRefRole.name : SourceRefRole → String
  | .definitionMetadata => "definition-metadata"
  | .modelSpec => "model-spec"
  | .providerDefinition => "provider-definition"
  | .providerReference => "provider-reference"
  | .connectorDefinition => "connector-definition"
  | .connectorReference => "connector-reference"
  | .capabilityRequirement => "capability-requirement"
  | .lawRequirement => "law-requirement"
  | .lawProof => "law-proof"
  | .meaning => "meaning"
  | .reconciliation => "reconciliation"
  | .machine => "machine"

/-- The owner makes a nested occurrence path unambiguous when identities are reused. -/
inductive SourceRefContext where
  | direct
  | reconciliation (definitionId : DefinitionId)
  deriving BEq, DecidableEq, Repr

structure SourceRefPath where
  role : SourceRefRole
  owner : DefinitionId
  context : SourceRefContext := .direct
  deriving BEq, DecidableEq, Repr

/-- Nonsemantic occurrence identity derived from a source span and its local ordinal. -/
structure SourceSpan where
  sourcePath : String
  line : Nat
  column : Nat
  endLine : Nat
  endColumn : Nat
  localOrdinal : Nat
  deriving BEq, DecidableEq, Repr

structure SourceRef where
  id : SourceSpan
  definitionId : DefinitionId
  path : SourceRefPath
  deriving BEq, DecidableEq, Repr

structure LocatedError where
  error : DefinitionError
  path : SourceRefPath
  original : Option SourceSpan
  offending : SourceSpan
  deriving BEq, DecidableEq, Repr

end Umpire
