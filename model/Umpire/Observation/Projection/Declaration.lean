import Umpire.Observation.Evaluation.Types
import Shared.ScopedProjection

/-!
Closed declarations for run-local evidence projection. A submission is an authorized Action request,
never a confirmed Model Outcome. Confirmation mappings name Target results which admission checks
against the authoritative kernel. Source ordinals and explicit parents are the only ordering inputs.
-/

namespace Umpire.Observation.Projection

/-- Identity within declared Run bindings; independent sources have independent zero-based ordinals. -/
abbrev Identity := Shared.ScopedProjection.Identity DefinitionId
abbrev Identity.mk := Shared.ScopedProjection.Identity.mk (Id := DefinitionId)
abbrev Identity.scope (identity : Identity) := Shared.ScopedProjection.Identity.scope identity
abbrev Identity.source (identity : Identity) := Shared.ScopedProjection.Identity.source identity
abbrev Identity.ordinal (identity : Identity) := Shared.ScopedProjection.Identity.ordinal identity

/-- Only an explicitly retained value may be supplied; redacted fields carry no raw value. -/
abbrev Field := Shared.ScopedProjection.EvidenceField DefinitionId EvidenceValue

/-- Immutable source evidence and its supporting Run Event sequences, distinct from source order. -/
abbrev Event := Shared.ScopedProjection.Event DefinitionId Field

/-- Explicit admission and retention ceilings, independent of semantic transition counts. -/
abbrev Limits := Shared.ScopedProjection.Limits

/-- Closed ownership mapping; only confirmation can release semantic steps. -/
abbrev Meaning (State Action Outcome Fact : Type) :=
  Shared.ScopedProjection.Meaning Action (Step State Outcome Fact)

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
abbrev Error := Shared.ScopedProjection.Error DefinitionId
abbrev Error.unsupportedVersion := Shared.ScopedProjection.Error.unsupportedVersion (Id := DefinitionId)
abbrev Error.invalidDeclaration := Shared.ScopedProjection.Error.invalidDeclaration (Id := DefinitionId)
abbrev Error.unsupportedDisposition := Shared.ScopedProjection.Error.unsupportedDisposition (Id := DefinitionId)
abbrev Error.invalidInitialState := Shared.ScopedProjection.Error.invalidInitialState (Id := DefinitionId)
abbrev Error.wrongScope := Shared.ScopedProjection.Error.wrongScope (Id := DefinitionId)
abbrev Error.unknownSource := Shared.ScopedProjection.Error.unknownSource (Id := DefinitionId)
abbrev Error.identityConflict := Shared.ScopedProjection.Error.identityConflict (Id := DefinitionId)
abbrev Error.wrongOperation := Shared.ScopedProjection.Error.wrongOperation (Id := DefinitionId)
abbrev Error.unsupportedEvidence := Shared.ScopedProjection.Error.unsupportedEvidence (Id := DefinitionId)
abbrev Error.unauthorizedField := Shared.ScopedProjection.Error.unauthorizedField (Id := DefinitionId)
abbrev Error.missingSubmission := Shared.ScopedProjection.Error.missingSubmission (Id := DefinitionId)
abbrev Error.invalidTransition := Shared.ScopedProjection.Error.invalidTransition (Id := DefinitionId)
abbrev Error.causalCycle := Shared.ScopedProjection.Error.causalCycle (Id := DefinitionId)
abbrev Error.incomparableOrder := Shared.ScopedProjection.Error.incomparableOrder (Id := DefinitionId)
abbrev Error.eventsExhausted := Shared.ScopedProjection.Error.eventsExhausted (Id := DefinitionId)
abbrev Error.bufferExhausted := Shared.ScopedProjection.Error.bufferExhausted (Id := DefinitionId)
abbrev Error.keysExhausted := Shared.ScopedProjection.Error.keysExhausted (Id := DefinitionId)
abbrev Error.supportExhausted := Shared.ScopedProjection.Error.supportExhausted (Id := DefinitionId)
abbrev Error.workExhausted := Shared.ScopedProjection.Error.workExhausted (Id := DefinitionId)
abbrev Error.eventSizeExhausted := Shared.ScopedProjection.Error.eventSizeExhausted (Id := DefinitionId)
abbrev Error.invalidEvidenceSupport := Shared.ScopedProjection.Error.invalidEvidenceSupport (Id := DefinitionId)
abbrev Error.incomplete := Shared.ScopedProjection.Error.incomplete (Id := DefinitionId)
abbrev Error.nonterminal := Shared.ScopedProjection.Error.nonterminal (Id := DefinitionId)
abbrev Error.closed := Shared.ScopedProjection.Error.closed (Id := DefinitionId)

/-- Pending is distinct from stutter; emission has a nonempty head/tail representation. -/
inductive Progress (Step : Type) where
  | pending (identities : List Identity)
  | stutter
  | emitted (first : Step) (rest : List Step)

/-- Whether this append admitted no semantic change and left no new pending evidence. -/
def Progress.isStutter {Step : Type} : Progress Step → Bool
  | .stutter => true
  | _ => false

/-- The newly emitted steps, never historical emissions. -/
def Progress.emissions {Step : Type} : Progress Step → List Step
  | .emitted first rest => first :: rest
  | _ => []

end Umpire.Observation.Projection
