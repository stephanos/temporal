import Umpire.Artifact
import Umpire.Space

/-! Pure checked inputs shared by bounded Experiment Space exploration policies. -/

namespace Umpire

/-- The two retained deterministic policies over one checked finite Experiment Space. -/
inductive ExplorationPolicy where
  | exhaustive
  | uncoveredCoordinate (coordinate : ModelCoordinate)
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable serialized name of an Exploration policy. -/
def ExplorationPolicy.name : ExplorationPolicy → String
  | .exhaustive => "exhaustive"
  | .uncoveredCoordinate _ => "uncovered-coordinate"

/-- Unchecked inputs for one bounded selection over exactly one checked Experiment Space. -/
structure ExplorationRequest (LawStatement : LawDefinition → Prop) where
  space : CheckedExperimentSpace LawStatement
  policy : ExplorationPolicy
  limit : Limit
  pinned : List ExperimentSpec := []

/-- Stable categories for bounded Exploration request failures. -/
inductive ExplorationErrorKind where
  | emptySpace
  | spacePointLimitExceeded
  | invalidLimitValue
  | invalidLimitUnit
  | unknownCoordinate
  | invalidPinnedArtifact
  | duplicatePinnedIdentity
  | incompatiblePinnedContract
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable serialized name of an Exploration request failure. -/
def ExplorationErrorKind.name : ExplorationErrorKind → String
  | .emptySpace => "empty-space"
  | .spacePointLimitExceeded => "space-point-limit-exceeded"
  | .invalidLimitValue => "invalid-limit-value"
  | .invalidLimitUnit => "invalid-limit-unit"
  | .unknownCoordinate => "unknown-coordinate"
  | .invalidPinnedArtifact => "invalid-pinned-artifact"
  | .duplicatePinnedIdentity => "duplicate-pinned-identity"
  | .incompatiblePinnedContract => "incompatible-pinned-contract"

/-- Canonical typed failure returned before candidate compilation or selection. -/
structure ExplorationError where
  kind : ExplorationErrorKind
  definitionId : DefinitionId
  sourcePath : String
  offendingValue : String
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Stable name of one Model Coordinate used in Exploration diagnostics. -/
def ModelCoordinate.name : ModelCoordinate → String
  | .initialState => "initial-state"
  | .selectedAction step => "selected-action:" ++ toString step
  | .modelOutcome step => "model-outcome:" ++ toString step
  | .resultingState step => "resulting-state:" ++ toString step
  | .observation step position => "observation:" ++ toString step ++ ":" ++ toString position

end Umpire
