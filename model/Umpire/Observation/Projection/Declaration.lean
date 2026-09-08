import Umpire.Observation.Evaluation.Types

/-!
Closed declarations for run-local evidence projection. A submission is an authorized Action request,
never a confirmed Model Outcome. Confirmation mappings name Target results which admission checks
against the authoritative kernel. Source ordinals and explicit parents are the only ordering inputs.
-/

namespace Umpire.Observation.Projection

/-- Identity within declared Run bindings; independent sources have independent zero-based ordinals. -/
structure Identity where
  scope : List (DefinitionId × String)
  source : DefinitionId
  ordinal : Nat
  deriving BEq, DecidableEq, Repr

/-- Only an explicitly retained value may be supplied; redacted fields carry no raw value. -/
structure Field where
  id : DefinitionId
  value : Option EvidenceValue
  deriving BEq, DecidableEq, Repr

/-- Immutable source evidence and its supporting Run Event sequences, distinct from source order. -/
structure Event where
  identity : Identity
  operation : String
  kind : DefinitionId
  parents : List Identity := []
  runSequences : List Nat
  fields : List Field := []
  deriving BEq, DecidableEq, Repr

/-- Explicit admission and retention ceilings, independent of semantic transition counts. -/
structure Limits where
  events : Nat
  buffered : Nat
  keys : Nat
  support : Nat
  work : Nat
  eventSize : Nat
  deriving BEq, DecidableEq, Repr

/-- Closed ownership mapping; only confirmation can release semantic steps. -/
inductive Meaning (State Action Outcome Fact : Type) where
  | irrelevant
  | submission (action : Action)
  | confirmed (submission : Option Action)
      (steps : List (Action × TransitionResult State Outcome Fact))
  deriving BEq, DecidableEq, Repr

/-- An evidence kind's complete authorized field policy and semantic interpretation. -/
structure Rule (State Action Outcome Fact : Type) where
  kind : DefinitionId
  fields : List (EvidenceFieldDeclaration × FieldDisposition) := []
  meaning : Meaning State Action Outcome Fact
  deriving BEq, DecidableEq, Repr

/-- Version one admits retain/redact/reject policies; digest processing remains with Observation. -/
structure Declaration (State Action Outcome Fact : Type) where
  id : DefinitionId
  version : Nat := 1
  scopeFields : List DefinitionId
  operationField : DefinitionId
  sources : List DefinitionId
  rules : List (Rule State Action Outcome Fact)
  limits : Limits
  deriving BEq, DecidableEq, Repr

/-- Admission failures never imply a Property verdict or return partially released steps. -/
inductive Error where
  | unsupportedVersion
  | invalidDeclaration
  | unsupportedDisposition
  | invalidInitialState
  | wrongScope
  | unknownSource
  | identityConflict (identity : Identity)
  | wrongOperation (identity parent : Identity)
  | unsupportedEvidence (kind : DefinitionId)
  | unauthorizedField (field : DefinitionId)
  | missingSubmission (identity : Identity)
  | invalidTransition (identity : Identity)
  | causalCycle
  | incomparableOrder (identity previous : Identity)
  | eventsExhausted
  | bufferExhausted
  | keysExhausted
  | supportExhausted
  | workExhausted
  | eventSizeExhausted
  | invalidEvidenceSupport
  | incomplete (pending : List Identity)
  | nonterminal (operations : List String)
  | closed
  deriving BEq, DecidableEq, Repr

/-- Pending is distinct from stutter; emission has a nonempty head/tail representation. -/
inductive Progress (Step : Type) where
  | pending (identities : List Identity)
  | stutter
  | emitted (first : Step) (rest : List Step)

/-- Whether this append admitted no semantic change and left no new pending evidence. -/
def Progress.isStutter : Progress Step → Bool
  | .stutter => true
  | _ => false

/-- The newly emitted steps, never historical emissions. -/
def Progress.emissions : Progress Step → List Step
  | .emitted first rest => first :: rest
  | _ => []

end Umpire.Observation.Projection
