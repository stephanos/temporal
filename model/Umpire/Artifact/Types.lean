import Umpire.Planning.Types

namespace Umpire

/-! Portable, environment-independent products of pure model planning. -/

def SelectionReason.name : SelectionReason → String
  | .satisfyingWitness => "satisfying-witness"
  | .violatingCounterexample => "violating-counterexample"
  | .behaviorSelection => "behavior-selection"

structure ObservationCheckpoint where
  transition : Nat
  observations : List ModelValue
  deriving BEq, DecidableEq, Repr

structure PlannedOccurrence where
  definitionId : DefinitionId
  actionDefinitionId : DefinitionId
  position : Nat
  authoredDefinitionId : Option DefinitionId
  deriving BEq, DecidableEq, Repr

structure PortableProperty where
  definitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  requirementDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

structure ArtifactProvenance where
  sourceDefinitionIds : List DefinitionId
  sourceLocations : List SourceLocation
  deriving BEq, DecidableEq, Repr

/--
The runtime-facing request remains explicit model data. Requested actions and model-owned outcomes
are deliberately separate, and no field claims that runtime execution or evidence occurred.
-/
structure DrivePlan where
  formatVersion : String
  artifactChecksum : ArtifactChecksum
  queryDefinitionId : DefinitionId
  queryBehaviorFingerprint : BehaviorFingerprint
  behaviorDefinitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  targetDefinitionId : DefinitionId
  targetBehaviorFingerprint : BehaviorFingerprint
  kernelDefinitionId : DefinitionId
  kernelBehaviorFingerprint : BehaviorFingerprint
  bindings : List RoleBinding
  symbolicRoles : List ResourceRole
  modelPreconditions : List SetupConstraint
  initialState : ModelValue
  requestedActions : List ModelValue
  modelOutcomes : List ModelValue
  resultingStates : List ModelValue
  linearExtension : List PlannedOccurrence
  selectedChoices : List ModelValue
  selectedVariants : List ModelValue
  requestedFaults : List ModelValue
  capabilityRequirementDefinitionIds : List DefinitionId
  expandedLimits : QueryLimits
  checkpoints : List ObservationCheckpoint
  selectionReason : SelectionReason
  explored : ExploredCounts
  knownGaps : List KnownGap
  provenance : ArtifactProvenance
  deriving BEq, DecidableEq, Repr

/-- The portable envelope consumed by later execution, checking, replay, and generation work. -/
structure ExperimentSpec where
  formatVersion : String
  artifactChecksum : ArtifactChecksum
  queryBehaviorFingerprint : BehaviorFingerprint
  plan : DrivePlan
  properties : List PortableProperty
  observationRequirementDefinitionIds : List DefinitionId
  provenance : ArtifactProvenance
  deriving BEq, DecidableEq, Repr

/-- One requested fault bound to the authored occurrence and action it intends to affect. -/
structure ArtifactFaultIntent where
  definitionId : DefinitionId
  occurrenceDefinitionId : DefinitionId
  actionDefinitionId : DefinitionId
  capabilityDefinitionId : DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Checked semantic intent that may be projected onto one planner-selected Artifact. -/
structure ArtifactIntent where
  queryDefinitionId : DefinitionId
  queryBehaviorFingerprint : BehaviorFingerprint
  behaviorDefinitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  targetDefinitionId : DefinitionId
  targetBehaviorFingerprint : BehaviorFingerprint
  kernelDefinitionId : DefinitionId
  kernelBehaviorFingerprint : BehaviorFingerprint
  selectedChoices : List ModelValue
  selectedVariants : List RoleBinding
  requestedFaults : List ArtifactFaultIntent
  additionalCapabilityRequirementDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

inductive ArtifactIntentErrorKind where
  | invalidDefinitionId
  | duplicateEntry
  | identityDrift
  | missingOccurrence
  | occurrenceMismatch
  | invalidCapability
  | variantMismatch
  deriving BEq, DecidableEq, Ord, Repr

def ArtifactIntentErrorKind.name : ArtifactIntentErrorKind → String
  | .invalidDefinitionId => "invalid-definition-id"
  | .duplicateEntry => "duplicate-entry"
  | .identityDrift => "identity-drift"
  | .missingOccurrence => "missing-occurrence"
  | .occurrenceMismatch => "occurrence-mismatch"
  | .invalidCapability => "invalid-capability"
  | .variantMismatch => "variant-mismatch"

structure ArtifactIntentError where
  kind : ArtifactIntentErrorKind
  definitionId : DefinitionId
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

def canonicalPlannerKnownGaps : List KnownGap := [
  { kind := .input, code := DefinitionId.of "umpire.known-gap.execution-evidence" },
  { kind := .interpretation, code := DefinitionId.of "umpire.known-gap.artifact-migrations" },
  { kind := .interpretation, code := DefinitionId.of "umpire.known-gap.artifact-reading" },
  { kind := .interpretation, code := DefinitionId.of "umpire.known-gap.evidence-evaluation" },
  { kind := .interpretation, code := DefinitionId.of "umpire.known-gap.runtime-scheduler-order" },
  { kind := .interpretation, code := DefinitionId.of "umpire.known-gap.runtime-storage-order" },
  { kind := .interpretation, code := DefinitionId.of "umpire.known-gap.runtime-transport-order" },
  { kind := .claim, code := DefinitionId.of "umpire.known-gap.promotion" }
]

end Umpire
