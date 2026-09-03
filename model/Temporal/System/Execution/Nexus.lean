import Temporal.System.Execution.LocalProfile
import Temporal.System.Nexus.Observation
import Umpire.Artifact.Set

namespace Temporal.System.Execution.Nexus

open _root_.Umpire

/-!
# Nexus caller-closure execution contract

This deep System module owns the inert execution program, configuration, and evidence-source
contract for one bounded Nexus participant. Its public composer accepts the complete checked
`ExperimentSpec`; it never imports Feature, reconstructs a smaller execution IR, performs runtime
IO, installs callbacks, evaluates evidence, or claims a model outcome. A neutral integration root
supplies the Feature-owned experiment and proves the exact fn-18 closure.
-/

private def id (value : String) : DefinitionId := DefinitionId.of value

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

/-- One stable identity for inert System-owned execution metadata. -/
structure ProgramReference where
  definitionId : DefinitionId
  version : Nat
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- One exact planned occurrence accepted by the participant program. -/
structure ProgramOccurrence where
  definitionId : DefinitionId
  actionDefinitionId : DefinitionId
  position : Nat
  deriving BEq, DecidableEq, Repr

/-- Fn-4's exact evidence-profile, Observation-program, and mapping references. -/
structure ObservationProgramDefinition where
  reference : ProgramReference
  profile : ProgramReference
  mapping : ProgramReference
  deriving BEq, DecidableEq, Repr

/--
Checked inert participant metadata for the four closed runtime commands.

`provenance` is deliberately excluded from the Behavior Fingerprint Generated View. Every other
field is execution meaning and therefore participates in the identity.
-/
structure ParticipantProgramDefinition where
  reference : ProgramReference
  participantDefinitionIds : List DefinitionId
  protocolDefinitionId : DefinitionId
  protocolVersion : Nat
  targetDefinitionIds : List DefinitionId
  actionDefinitionIds : List DefinitionId
  occurrences : List ProgramOccurrence
  requestedFaultDefinitionIds : List DefinitionId
  capabilityDefinitionIds : List DefinitionId
  commands : List ParticipantCommandKind
  evidenceSourceDefinitionIds : List DefinitionId
  provenance : List SourceLocation
  deriving BEq, DecidableEq, Repr

/-- The complete authored input checked before an adapter may perform IO. -/
structure ExecutionDefinition where
  experiment : ExperimentSpec
  localProfile : LocalProfileDefinition
  observationProgram : ObservationProgramDefinition
  participantProgram : ParticipantProgramDefinition
  configuration : RuntimeConfiguration
  deriving BEq, DecidableEq, Repr

/-- Stable preflight categories for the first Nexus execution composition. -/
inductive NexusExecutionErrorKind where
  | inputSet
  | profile
  | configuration
  | target
  | action
  | occurrence
  | fault
  | participant
  | protocol
  | capability
  | budget
  | program
  | reference
  | seed
  | attempt
  deriving BEq, DecidableEq, Ord, Repr

def NexusExecutionErrorKind.name : NexusExecutionErrorKind → String
  | .inputSet => "input-set"
  | .profile => "profile"
  | .configuration => "configuration"
  | .target => "target"
  | .action => "action"
  | .occurrence => "occurrence"
  | .fault => "fault"
  | .participant => "participant"
  | .protocol => "protocol"
  | .capability => "capability"
  | .budget => "budget"
  | .program => "program"
  | .reference => "reference"
  | .seed => "seed"
  | .attempt => "attempt"

/-- One deterministic composition failure with no partial checked value. -/
structure NexusExecutionError where
  kind : NexusExecutionErrorKind
  subject : DefinitionId
  deriving BEq, DecidableEq, Repr

private def executionError
    (kind : NexusExecutionErrorKind)
    (subject : DefinitionId) : NexusExecutionError := { kind, subject }

/-- Canonical provenance for the System-owned participant program. -/
def canonicalProgramSource : SourceLocation := {
  path := "Temporal/System/Execution/Nexus.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

/-- Canonical provenance for the fn-18 RuntimeConfiguration. -/
def canonicalConfigurationSource : SourceLocation := canonicalProgramSource

/-- The four exact evidence sources the later adapter must close. -/
def evidenceSourceDefinitionIds : List DefinitionId := [
  id "umpire.evidence.source.cleanup",
  id "umpire.evidence.source.control-receipt",
  id "umpire.evidence.source.history",
  id "umpire.evidence.source.participant-output"
]

private def exactBehaviorFingerprint
    (value : String)
    (valid : (BehaviorFingerprint.parse? value).isSome = true) : BehaviorFingerprint :=
  (BehaviorFingerprint.parse? value).get valid

private def exactArtifactChecksum
    (value : String)
    (valid : (ArtifactChecksum.parse? value).isSome = true) : ArtifactChecksum :=
  (ArtifactChecksum.parse? value).get valid

/-- Exact caller-closure target reference accepted by this execution contract. -/
def targetDefinitionId : DefinitionId := id "workflow-nexus.target.caller-closure"

/-- Exact caller-closure kernel reference accepted by this execution contract. -/
def kernelDefinitionId : DefinitionId := id "workflow-nexus.kernel.caller-closure"

/-- Exact caller-closure action reference accepted by this execution contract. -/
def actionDefinitionId : DefinitionId := id "workflow.action.force-close"

/-- Exact complete-experiment binding accepted by this execution contract. -/
def experimentBinding : ArtifactBinding := {
  formatVersion := "umpire-experiment/v2"
  artifactChecksum := exactArtifactChecksum
    "sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8"
    (by native_decide)
  behaviorFingerprint := exactBehaviorFingerprint
    "sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af"
    (by native_decide)
  provenanceChecksum := exactArtifactChecksum
    "sha256:f7a6ebefca8202c6a7c467fd516e54d162c7d1f254c6c9a1f004a7f0b4135ab8"
    (by native_decide)
}

/-- Exact fault-bearing caller-closure ExperimentSpec binding accepted by the negative control. -/
def duplicateDeliveryExperimentBinding : ArtifactBinding := {
  formatVersion := "umpire-experiment/v2"
  artifactChecksum := exactArtifactChecksum
    "sha256:09091758defd5ce50cc9acbba23a5c8499da4eef9b6e36878ac989ddea87fedf"
    (by native_decide)
  behaviorFingerprint := exactBehaviorFingerprint
    "sha256:eb6c9391f0bbd82effc5793d4b0650c3b01f2471b5f05838cdec7377a5931a91"
    (by native_decide)
  provenanceChecksum := exactArtifactChecksum
    "sha256:4136694dfede1044bbf391c5faba21a1a89a589890aa2915eb860a46942797c2"
    (by native_decide)
}

private def targetBehaviorFingerprint : BehaviorFingerprint := exactBehaviorFingerprint
  "sha256:22e49d60fb38ec52fd44f09549f28329d169605168dd6dc828f43941445faacd"
  (by native_decide)

private def requestedAction : ModelValue := ModelValue.named actionDefinitionId "force-close"

private def capabilityDefinitionIds : List DefinitionId := [
  id "nexus.capability.cancellation",
  id "workflow-nexus.capability.ownership",
  id "workflow.capability.lifecycle"
]

private def observationProfileReference : ProgramReference := {
  definitionId := id "temporal.nexus.synthetic.basic-lifecycle.profile"
  version := 1
  behaviorFingerprint := exactBehaviorFingerprint
    "sha256:ac3cf245ad3e4a311eb6372be9caf49301c7e8ad3ee1b1875a53ea69d1ddc105"
    (by native_decide)
}

private def observationMappingReference : ProgramReference := {
  definitionId := id "temporal.nexus.synthetic.basic-lifecycle.mapping"
  version := 1
  behaviorFingerprint := exactBehaviorFingerprint
    "sha256:608e4db6c3a29d0f953640621ee34d34e16b0090309e85804e21f0cb21be30a2"
    (by native_decide)
}

private def referenceJson (reference : ProgramReference) : String :=
  "{\"definitionId\":" ++ quote reference.definitionId.value ++
    ",\"version\":" ++ toString reference.version ++
    ",\"behaviorFingerprint\":" ++ quote reference.behaviorFingerprint.render ++ "}"

private def observationProgramGeneratedViewBytes
    (definition : ObservationProgramDefinition) : String :=
  Json.prettyBytes <|
    "{\"definitionId\":" ++ quote definition.reference.definitionId.value ++
      ",\"version\":" ++ toString definition.reference.version ++
      ",\"profile\":" ++ referenceJson definition.profile ++
      ",\"mapping\":" ++ referenceJson definition.mapping ++ "}"

private def canonicalObservationProgramDraft : ObservationProgramDefinition := {
  reference := {
    definitionId := id "temporal.nexus.observation-program.basic-lifecycle"
    version := 1
    behaviorFingerprint := behaviorFingerprintOf "unsealed-observation-program"
  }
  profile := observationProfileReference
  mapping := observationMappingReference
}

private def expectedObservationProgramBehaviorFingerprint
    (definition : ObservationProgramDefinition) : BehaviorFingerprint :=
  behaviorFingerprintOf (observationProgramGeneratedViewBytes definition)

/-- System-owned observation program bound to fn-4's exact profile and checked mapping. -/
def canonicalObservationProgramDefinition : ObservationProgramDefinition := {
  canonicalObservationProgramDraft with
  reference := {
    canonicalObservationProgramDraft.reference with
    behaviorFingerprint := expectedObservationProgramBehaviorFingerprint
      canonicalObservationProgramDraft
  }
}

/-- Task `.7`'s checked duplicate-delivery profile, program, and mapping references. -/
def duplicateDeliveryObservationProgramDefinition : ObservationProgramDefinition := {
  reference := {
    definitionId := Temporal.System.Nexus.Observation.DuplicateDelivery.programId
    version := Temporal.System.Nexus.Observation.DuplicateDelivery.programVersion
    behaviorFingerprint :=
      Temporal.System.Nexus.Observation.DuplicateDelivery.programBehaviorFingerprint
  }
  profile := {
    definitionId := Temporal.System.Nexus.Observation.DuplicateDelivery.Profile.id
    version := Temporal.System.Nexus.Observation.DuplicateDelivery.profileVersion
    behaviorFingerprint :=
      Temporal.System.Nexus.Observation.DuplicateDelivery.profileBehaviorFingerprint
  }
  mapping := {
    definitionId := Temporal.System.Nexus.Observation.DuplicateDelivery.Mapping.id
    version := Temporal.System.Nexus.Observation.DuplicateDelivery.mappingVersion
    behaviorFingerprint :=
      Temporal.System.Nexus.Observation.DuplicateDelivery.mappingBehaviorFingerprint
  }
}

private def occurrenceJson (occurrence : ProgramOccurrence) : String :=
  "{\"definitionId\":" ++ quote occurrence.definitionId.value ++
    ",\"actionDefinitionId\":" ++ quote occurrence.actionDefinitionId.value ++
    ",\"position\":" ++ toString occurrence.position ++ "}"

/-- The non-self-referential, provenance-free Generated View for a participant program. -/
def participantProgramGeneratedViewBytes (definition : ParticipantProgramDefinition) : String :=
  Json.prettyBytes <|
    "{\"definitionId\":" ++ quote definition.reference.definitionId.value ++
      ",\"version\":" ++ toString definition.reference.version ++
      ",\"participantDefinitionIds\":" ++
        array (definition.participantDefinitionIds.map (quote ∘ DefinitionId.value)) ++
      ",\"protocolDefinitionId\":" ++ quote definition.protocolDefinitionId.value ++
      ",\"protocolVersion\":" ++ toString definition.protocolVersion ++
      ",\"targetDefinitionIds\":" ++
        array (definition.targetDefinitionIds.map (quote ∘ DefinitionId.value)) ++
      ",\"actionDefinitionIds\":" ++
        array (definition.actionDefinitionIds.map (quote ∘ DefinitionId.value)) ++
      ",\"occurrences\":" ++ array (definition.occurrences.map occurrenceJson) ++
      ",\"requestedFaultDefinitionIds\":" ++
        array (definition.requestedFaultDefinitionIds.map (quote ∘ DefinitionId.value)) ++
      ",\"capabilityDefinitionIds\":" ++
        array (definition.capabilityDefinitionIds.map (quote ∘ DefinitionId.value)) ++
      ",\"commands\":" ++
        array (definition.commands.map (quote ∘ ParticipantCommandKind.name)) ++
      ",\"evidenceSourceDefinitionIds\":" ++
        array (definition.evidenceSourceDefinitionIds.map (quote ∘ DefinitionId.value)) ++ "}"

/-- Recompute a participant-program identity from every meaning-bearing field. -/
def ParticipantProgramDefinition.expectedBehaviorFingerprint
    (definition : ParticipantProgramDefinition) : BehaviorFingerprint :=
  behaviorFingerprintOf (participantProgramGeneratedViewBytes definition)

private def canonicalParticipantProgramDraft : ParticipantProgramDefinition := {
  reference := {
    definitionId := id "temporal.nexus.participant-program.caller-closure"
    version := 1
    behaviorFingerprint := behaviorFingerprintOf "unsealed-participant-program"
  }
  participantDefinitionIds := [id "temporal.nexus.participant.caller-closure"]
  protocolDefinitionId := id "umpire.participant-protocol.v2"
  protocolVersion := 2
  targetDefinitionIds := [targetDefinitionId]
  actionDefinitionIds := [actionDefinitionId]
  occurrences := [{
    definitionId := id "workflow-nexus.occurrence.force-close"
    actionDefinitionId := actionDefinitionId
    position := 1
  }]
  requestedFaultDefinitionIds := []
  capabilityDefinitionIds := capabilityDefinitionIds
  commands := [.prepare, .realize, .observe, .cleanup]
  evidenceSourceDefinitionIds
  provenance := [canonicalProgramSource]
}

private def duplicateDeliveryParticipantProgramDraft : ParticipantProgramDefinition := {
  canonicalParticipantProgramDraft with
  reference := {
    definitionId := id "temporal.nexus.participant-program.caller-closure-duplicate-delivery"
    version := 1
    behaviorFingerprint := behaviorFingerprintOf "unsealed-participant-program"
  }
  requestedFaultDefinitionIds := [
    Temporal.System.Nexus.Observation.DuplicateDelivery.faultDefinitionId
  ]
}

/-- Canonical unchecked fixture used by program/composition mutation tests. -/
def canonicalParticipantProgramDefinition : ParticipantProgramDefinition := {
  canonicalParticipantProgramDraft with
  reference := {
    canonicalParticipantProgramDraft.reference with
    behaviorFingerprint := canonicalParticipantProgramDraft.expectedBehaviorFingerprint
  }
}

/-- Canonical unchecked negative-control fixture used by program/composition mutation tests. -/
def duplicateDeliveryParticipantProgramDefinition : ParticipantProgramDefinition := {
  duplicateDeliveryParticipantProgramDraft with
  reference := {
    duplicateDeliveryParticipantProgramDraft.reference with
    behaviorFingerprint := duplicateDeliveryParticipantProgramDraft.expectedBehaviorFingerprint
  }
}

private structure CheckedParticipantProgramPayload where
  definition : ParticipantProgramDefinition

/-- A participant program whose entire inert command surface has passed the exact checker. -/
structure CheckedParticipantProgram where
  private mk ::
  private payload : CheckedParticipantProgramPayload

/-- Inspect the complete inert program metadata. -/
def CheckedParticipantProgram.definition
    (program : CheckedParticipantProgram) : ParticipantProgramDefinition :=
  program.payload.definition

private def participantProgramProvenanceValid
    (definition : ParticipantProgramDefinition) : Bool :=
  ({
    sourceDefinitionIds := [definition.reference.definitionId]
    sourceLocations := definition.provenance
  } : ArtifactProvenance).isValidTransport

/-- Check the complete inert Nexus participant-program contract before runtime IO. -/
private def checkParticipantProgramAgainst
    (expected : ParticipantProgramDefinition)
    (definition : ParticipantProgramDefinition) : Except NexusExecutionError CheckedParticipantProgram := do
  let subject := definition.reference.definitionId
  if definition.participantDefinitionIds !=
      expected.participantDefinitionIds then
    throw (executionError .participant subject)
  if definition.protocolDefinitionId != expected.protocolDefinitionId ||
      definition.protocolVersion != expected.protocolVersion then
    throw (executionError .protocol subject)
  if definition.targetDefinitionIds != expected.targetDefinitionIds then
    throw (executionError .target subject)
  if definition.actionDefinitionIds != expected.actionDefinitionIds ||
      definition.occurrences.any fun occurrence =>
        !definition.actionDefinitionIds.contains occurrence.actionDefinitionId then
    throw (executionError .action subject)
  if definition.occurrences != expected.occurrences then
    throw (executionError .occurrence subject)
  if definition.requestedFaultDefinitionIds != expected.requestedFaultDefinitionIds then
    throw (executionError .fault subject)
  if definition.capabilityDefinitionIds !=
      expected.capabilityDefinitionIds then
    throw (executionError .capability subject)
  if definition.reference.definitionId != expected.reference.definitionId ||
      definition.reference.version != expected.reference.version ||
      definition.commands != expected.commands ||
      definition.evidenceSourceDefinitionIds !=
        expected.evidenceSourceDefinitionIds ||
      !participantProgramProvenanceValid definition ||
      definition.reference.behaviorFingerprint != definition.expectedBehaviorFingerprint then
    throw (executionError .program subject)
  pure (.mk { definition })

/-- Check either exact closed Nexus participant-program contract before runtime IO. -/
def checkParticipantProgram
    (definition : ParticipantProgramDefinition) : Except NexusExecutionError CheckedParticipantProgram :=
  if definition.reference.definitionId == canonicalParticipantProgramDraft.reference.definitionId then
    checkParticipantProgramAgainst canonicalParticipantProgramDraft definition
  else if definition.reference.definitionId ==
      duplicateDeliveryParticipantProgramDraft.reference.definitionId then
    checkParticipantProgramAgainst duplicateDeliveryParticipantProgramDraft definition
  else
    .error (executionError .program definition.reference.definitionId)

private theorem canonicalParticipantProgram_isSome :
    (checkParticipantProgram canonicalParticipantProgramDefinition).toOption.isSome = true := by
  native_decide

/-- The sole checked participant program available to a runtime adapter. -/
def callerClosureProgram : CheckedParticipantProgram :=
  (checkParticipantProgram canonicalParticipantProgramDefinition).toOption.get
    canonicalParticipantProgram_isSome

private theorem duplicateDeliveryParticipantProgram_isSome :
    (checkParticipantProgram duplicateDeliveryParticipantProgramDefinition).toOption.isSome = true := by
  native_decide

/-- The sole checked duplicate-delivery program available to the bounded runtime adapter. -/
def duplicateDeliveryProgram : CheckedParticipantProgram :=
  (checkParticipantProgram duplicateDeliveryParticipantProgramDefinition).toOption.get
    duplicateDeliveryParticipantProgram_isSome

private def authorityProfileJson (profile : AuthorityProfile) : String :=
  "{\"definitionId\":" ++ quote profile.definitionId.value ++
    ",\"version\":" ++ toString profile.version ++
    ",\"behaviorFingerprint\":" ++ quote profile.behaviorFingerprint.render ++
    ",\"requiredCapabilityDefinitionIds\":" ++
      array (profile.requiredCapabilityDefinitionIds.map (quote ∘ DefinitionId.value)) ++ "}"

private def phaseLimitJson (limit : PhaseLimit) : String :=
  "{\"phase\":" ++ quote limit.phase.name ++
    ",\"durationMilliseconds\":" ++ toString limit.durationMilliseconds ++
    ",\"maxAttempts\":" ++ toString limit.maxAttempts ++
    ",\"maxRecords\":" ++ toString limit.maxRecords ++
    ",\"maxBytes\":" ++ toString limit.maxBytes ++ "}"

private def observationConfigurationJson (observation : ObservationConfiguration) : String :=
  "{\"profileDefinitionId\":" ++ quote observation.profileDefinitionId.value ++
    ",\"profileBehaviorFingerprint\":" ++ quote observation.profileBehaviorFingerprint.render ++
    ",\"programDefinitionId\":" ++ quote observation.programDefinitionId.value ++
    ",\"programBehaviorFingerprint\":" ++ quote observation.programBehaviorFingerprint.render ++
    ",\"mappingDefinitionId\":" ++ quote observation.mappingDefinitionId.value ++
    ",\"mappingBehaviorFingerprint\":" ++ quote observation.mappingBehaviorFingerprint.render ++ "}"

private def participantBindingJson (binding : ParticipantBinding) : String :=
  "{\"participantDefinitionId\":" ++ quote binding.participantDefinitionId.value ++
    ",\"protocolDefinitionId\":" ++ quote binding.protocolDefinitionId.value ++
    ",\"protocolVersion\":" ++ toString binding.protocolVersion ++
    ",\"programDefinitionId\":" ++ quote binding.programDefinitionId.value ++
    ",\"programBehaviorFingerprint\":" ++ quote binding.programBehaviorFingerprint.render ++
    ",\"capabilityDefinitionIds\":" ++
      array (binding.capabilityDefinitionIds.map (quote ∘ DefinitionId.value)) ++ "}"

/--
The RuntimeConfiguration Generated View excludes only provenance and transport checksums, including
the Behavior Fingerprint itself. Every execution-meaning field remains identity-bearing.
-/
def runtimeConfigurationGeneratedViewBytes (configuration : RuntimeConfiguration) : String :=
  Json.prettyBytes <|
    "{\"formatVersion\":" ++ quote configuration.formatVersion ++
      ",\"configurationDefinitionId\":" ++ quote configuration.configurationDefinitionId.value ++
      ",\"experiment\":" ++ configuration.experiment.canonicalJson ++
      ",\"authorityProfile\":" ++ authorityProfileJson configuration.authorityProfile ++
      ",\"phaseLimits\":" ++ array (configuration.phaseLimits.map phaseLimitJson) ++
      ",\"observation\":" ++ observationConfigurationJson configuration.observation ++
      ",\"participantBindings\":" ++
        array (configuration.participantBindings.map participantBindingJson) ++
      ",\"knownGaps\":" ++ array (configuration.knownGaps.toList.map canonicalKnownGapJson) ++ "}"

/-- Recompute the Nexus configuration identity from every meaning-bearing field. -/
def expectedRuntimeConfigurationBehaviorFingerprint
    (configuration : RuntimeConfiguration) : BehaviorFingerprint :=
  behaviorFingerprintOf (runtimeConfigurationGeneratedViewBytes configuration)

private def emptyChecksum : ArtifactChecksum := drivePlanChecksumOf ""

private def runtimeConfigurationDraft (experiment : ExperimentSpec) : RuntimeConfiguration := {
  formatVersion := "umpire-runtime-configuration/v2"
  configurationDefinitionId := id "temporal.nexus.runtime-configuration.caller-closure"
  behaviorFingerprint := behaviorFingerprintOf "unsealed-runtime-configuration"
  experiment := experiment.artifactBinding
  authorityProfile := ephemeralLocalProfile.authorityProfile
  phaseLimits := ephemeralLocalProfile.phaseLimits
  observation := {
    profileDefinitionId := canonicalObservationProgramDefinition.profile.definitionId
    profileBehaviorFingerprint := canonicalObservationProgramDefinition.profile.behaviorFingerprint
    programDefinitionId := canonicalObservationProgramDefinition.reference.definitionId
    programBehaviorFingerprint := canonicalObservationProgramDefinition.reference.behaviorFingerprint
    mappingDefinitionId := canonicalObservationProgramDefinition.mapping.definitionId
    mappingBehaviorFingerprint := canonicalObservationProgramDefinition.mapping.behaviorFingerprint
  }
  participantBindings := [{
    participantDefinitionId := id "temporal.nexus.participant.caller-closure"
    protocolDefinitionId := canonicalParticipantProgramDefinition.protocolDefinitionId
    protocolVersion := canonicalParticipantProgramDefinition.protocolVersion
    programDefinitionId := canonicalParticipantProgramDefinition.reference.definitionId
    programBehaviorFingerprint := canonicalParticipantProgramDefinition.reference.behaviorFingerprint
    capabilityDefinitionIds := canonicalParticipantProgramDefinition.capabilityDefinitionIds
  }]
  knownGaps := KnownGapSet.empty
  provenance := {
    sourceDefinitionIds := [
      canonicalObservationProgramDefinition.reference.definitionId,
      canonicalParticipantProgramDefinition.reference.definitionId,
      id "temporal.nexus.runtime-configuration.caller-closure",
      canonicalObservationProgramDefinition.mapping.definitionId,
      canonicalObservationProgramDefinition.profile.definitionId,
      ephemeralLocalProfile.reference.definitionId
    ]
    sourceLocations := [canonicalConfigurationSource]
  }
  provenanceChecksum := emptyChecksum
  artifactChecksum := emptyChecksum
}

private def duplicateDeliveryRuntimeConfigurationDraft
    (experiment : ExperimentSpec) : RuntimeConfiguration := {
  formatVersion := "umpire-runtime-configuration/v2"
  configurationDefinitionId :=
    id "temporal.nexus.runtime-configuration.caller-closure-duplicate-delivery"
  behaviorFingerprint := behaviorFingerprintOf "unsealed-runtime-configuration"
  experiment := experiment.artifactBinding
  authorityProfile := ephemeralLocalProfile.authorityProfile
  phaseLimits := ephemeralLocalProfile.phaseLimits
  observation := {
    profileDefinitionId := duplicateDeliveryObservationProgramDefinition.profile.definitionId
    profileBehaviorFingerprint :=
      duplicateDeliveryObservationProgramDefinition.profile.behaviorFingerprint
    programDefinitionId := duplicateDeliveryObservationProgramDefinition.reference.definitionId
    programBehaviorFingerprint :=
      duplicateDeliveryObservationProgramDefinition.reference.behaviorFingerprint
    mappingDefinitionId := duplicateDeliveryObservationProgramDefinition.mapping.definitionId
    mappingBehaviorFingerprint :=
      duplicateDeliveryObservationProgramDefinition.mapping.behaviorFingerprint
  }
  participantBindings := [{
    participantDefinitionId := id "temporal.nexus.participant.caller-closure"
    protocolDefinitionId := duplicateDeliveryParticipantProgramDefinition.protocolDefinitionId
    protocolVersion := duplicateDeliveryParticipantProgramDefinition.protocolVersion
    programDefinitionId := duplicateDeliveryParticipantProgramDefinition.reference.definitionId
    programBehaviorFingerprint :=
      duplicateDeliveryParticipantProgramDefinition.reference.behaviorFingerprint
    capabilityDefinitionIds := duplicateDeliveryParticipantProgramDefinition.capabilityDefinitionIds
  }]
  knownGaps := KnownGapSet.empty
  provenance := {
    sourceDefinitionIds := [
      duplicateDeliveryParticipantProgramDefinition.reference.definitionId,
      id "temporal.nexus.runtime-configuration.caller-closure-duplicate-delivery",
      ephemeralLocalProfile.reference.definitionId,
      duplicateDeliveryObservationProgramDefinition.mapping.definitionId,
      duplicateDeliveryObservationProgramDefinition.reference.definitionId,
      duplicateDeliveryObservationProgramDefinition.profile.definitionId,
    ]
    sourceLocations := [canonicalConfigurationSource]
  }
  provenanceChecksum := emptyChecksum
  artifactChecksum := emptyChecksum
}

/-- Compose the exact sealed fn-18 RuntimeConfiguration around one complete ExperimentSpec. -/
def runtimeConfigurationFor (experiment : ExperimentSpec) : RuntimeConfiguration :=
  { runtimeConfigurationDraft experiment with
      behaviorFingerprint := expectedRuntimeConfigurationBehaviorFingerprint
        (runtimeConfigurationDraft experiment) } |>.seal

/-- Compose the sealed negative-control RuntimeConfiguration around its exact ExperimentSpec. -/
def duplicateDeliveryRuntimeConfigurationFor
    (experiment : ExperimentSpec) : RuntimeConfiguration :=
  { duplicateDeliveryRuntimeConfigurationDraft experiment with
      behaviorFingerprint := expectedRuntimeConfigurationBehaviorFingerprint
        (duplicateDeliveryRuntimeConfigurationDraft experiment) } |>.seal

private def mapLocalProfileError (kind : LocalProfileErrorKind) : NexusExecutionErrorKind :=
  match kind with
  | .profile | .authority => .profile
  | .configuration => .configuration
  | .capability => .capability
  | .budget => .budget
  | .seed => .seed
  | .attempt => .attempt
  | .participant => .participant
  | .protocol => .protocol

private def checkExperimentAgainst
    (binding : ArtifactBinding)
    (requestedFaults : List ModelValue)
    (experiment : ExperimentSpec) : Except NexusExecutionError Unit := do
  let subject := experiment.plan.queryDefinitionId
  if experiment.plan.targetDefinitionId != targetDefinitionId ||
      experiment.plan.targetBehaviorFingerprint != targetBehaviorFingerprint ||
      experiment.plan.kernelDefinitionId != kernelDefinitionId ||
      experiment.plan.kernelBehaviorFingerprint != targetBehaviorFingerprint then
    throw (executionError .target subject)
  if experiment.plan.requestedFaults != requestedFaults then
    throw (executionError .fault subject)
  if experiment.plan.requestedActions != [requestedAction] then
    throw (executionError .action subject)
  if experiment.plan.linearExtension != [{
      definitionId := id "workflow-nexus.occurrence.force-close"
      actionDefinitionId := actionDefinitionId
      position := 1
      authoredDefinitionId := some (id "workflow-nexus.occurrence.force-close")
    }] then
    throw (executionError .occurrence subject)
  if experiment.plan.capabilityRequirementDefinitionIds !=
      canonicalParticipantProgramDefinition.capabilityDefinitionIds then
    throw (executionError .capability subject)
  if experiment.artifactBinding != binding || !experiment.isValidTransport then
    throw (executionError .inputSet subject)

private def checkObservationProgramAgainst
    (expected : ObservationProgramDefinition)
    (definition : ObservationProgramDefinition) : Except NexusExecutionError Unit := do
  let subject := definition.reference.definitionId
  if definition.profile != expected.profile || definition.mapping != expected.mapping then
    throw (executionError .reference subject)
  if definition.reference != expected.reference then
    throw (executionError .program subject)

private def checkConfiguration
    (configuration : RuntimeConfiguration)
    (experiment : ExperimentSpec)
    (observationProgram : ObservationProgramDefinition)
    (participantProgram : ParticipantProgramDefinition)
    (expectedConfiguration : RuntimeConfiguration)
    (profile : CheckedLocalProfile) : Except NexusExecutionError Unit := do
  let subject := configuration.configurationDefinitionId
  if configuration.experiment != experiment.artifactBinding then
    throw (executionError .inputSet subject)
  if configuration.authorityProfile.definitionId != profile.reference.definitionId ||
      configuration.authorityProfile.version != profile.reference.version ||
      configuration.authorityProfile.behaviorFingerprint != profile.reference.behaviorFingerprint then
    throw (executionError .profile subject)
  if configuration.authorityProfile.requiredCapabilityDefinitionIds != [] then
    throw (executionError .capability subject)
  if configuration.phaseLimits != profile.phaseLimits then
    throw (executionError .budget subject)
  if configuration.observation.profileDefinitionId != observationProgram.profile.definitionId ||
      configuration.observation.profileBehaviorFingerprint !=
        observationProgram.profile.behaviorFingerprint ||
      configuration.observation.mappingDefinitionId != observationProgram.mapping.definitionId ||
      configuration.observation.mappingBehaviorFingerprint !=
        observationProgram.mapping.behaviorFingerprint then
    throw (executionError .reference subject)
  if configuration.observation.programDefinitionId != observationProgram.reference.definitionId ||
      configuration.observation.programBehaviorFingerprint !=
        observationProgram.reference.behaviorFingerprint then
    throw (executionError .program subject)
  let binding ← match configuration.participantBindings with
    | [binding] =>
        if [binding.participantDefinitionId] != participantProgram.participantDefinitionIds then
          throw (executionError .participant subject)
        pure binding
    | _ => throw (executionError .participant subject)
  if binding.protocolDefinitionId != participantProgram.protocolDefinitionId ||
      binding.protocolVersion != participantProgram.protocolVersion then
    throw (executionError .protocol subject)
  if binding.programDefinitionId != participantProgram.reference.definitionId ||
      binding.programBehaviorFingerprint != participantProgram.reference.behaviorFingerprint then
    throw (executionError .program subject)
  if binding.capabilityDefinitionIds != participantProgram.capabilityDefinitionIds then
    throw (executionError .capability subject)
  if configuration.knownGaps.toList != [] || !configuration.isValidTransport ||
      configuration.behaviorFingerprint !=
        expectedRuntimeConfigurationBehaviorFingerprint configuration ||
      !configuration.closesExperiment experiment ||
      configuration != expectedConfiguration then
    throw (executionError .configuration subject)
  match profile.checkRuntimeConfiguration configuration with
  | .ok _ => pure ()
  | .error error => throw (executionError (mapLocalProfileError error.kind) subject)

private structure CheckedExecutionPayload where
  experiment : ExperimentSpec
  profile : CheckedLocalProfile
  observationProgram : ObservationProgramDefinition
  participantProgram : CheckedParticipantProgram
  configuration : RuntimeConfiguration

/-- Exact checked metadata available to the bounded local runner and no other authority. -/
structure CheckedExecution where
  private mk ::
  private payload : CheckedExecutionPayload

/-- Inspect the complete current ExperimentSpec without reconstructing omitted meaning. -/
def CheckedExecution.experiment (execution : CheckedExecution) : ExperimentSpec :=
  execution.payload.experiment

/-- Inspect the checked System-owned participant program. -/
def CheckedExecution.program (execution : CheckedExecution) : ParticipantProgramDefinition :=
  execution.payload.participantProgram.definition

/-- Inspect the exact fn-4 evidence-profile, Observation-program, and mapping references. -/
def CheckedExecution.observationProgram
    (execution : CheckedExecution) : ObservationProgramDefinition :=
  execution.payload.observationProgram

/-- Inspect the exact fn-18 RuntimeConfiguration. -/
def CheckedExecution.configuration (execution : CheckedExecution) : RuntimeConfiguration :=
  execution.payload.configuration

/-- Project the exact two-member fn-18 closure without adding an artifact family. -/
def CheckedExecution.artifactSet (execution : CheckedExecution) : ArtifactSet := {
  experiment := execution.experiment
  runtimeConfiguration := execution.configuration
}

/-- Compose and check the complete Nexus execution definition before runtime IO. -/
def checkExecution
    (definition : ExecutionDefinition) : Except NexusExecutionError CheckedExecution := do
  let profile ← match checkLocalProfile definition.localProfile with
    | .ok checked => pure checked
    | .error error =>
        throw (executionError (mapLocalProfileError error.kind) error.subject)
  let participantProgram ← checkParticipantProgram definition.participantProgram
  let expectedConfiguration ←
    if definition.participantProgram.reference.definitionId ==
        canonicalParticipantProgramDefinition.reference.definitionId then
      checkExperimentAgainst experimentBinding [] definition.experiment
      checkObservationProgramAgainst canonicalObservationProgramDefinition
        definition.observationProgram
      pure (runtimeConfigurationFor definition.experiment)
    else
      checkExperimentAgainst duplicateDeliveryExperimentBinding [{
        definitionId := Temporal.System.Nexus.Observation.DuplicateDelivery.faultDefinitionId
        value := id "workflow-nexus.occurrence.force-close" |>.value
      }] definition.experiment
      checkObservationProgramAgainst duplicateDeliveryObservationProgramDefinition
        definition.observationProgram
      pure (duplicateDeliveryRuntimeConfigurationFor definition.experiment)
  checkConfiguration definition.configuration definition.experiment definition.observationProgram
    definition.participantProgram expectedConfiguration profile
  pure (.mk {
    experiment := definition.experiment
    profile := profile
    observationProgram := definition.observationProgram
    participantProgram := participantProgram
    configuration := definition.configuration
  })

/-- Compose the unchecked System definition around one complete caller-closure ExperimentSpec. -/
def executionDefinitionFor (experiment : ExperimentSpec) : ExecutionDefinition := {
  experiment := experiment
  localProfile := canonicalLocalProfileDefinition
  observationProgram := canonicalObservationProgramDefinition
  participantProgram := canonicalParticipantProgramDefinition
  configuration := runtimeConfigurationFor experiment
}

/-- Compose the unchecked closed negative-control definition around its fault-bearing spec. -/
def duplicateDeliveryExecutionDefinitionFor (experiment : ExperimentSpec) : ExecutionDefinition := {
  experiment := experiment
  localProfile := canonicalLocalProfileDefinition
  observationProgram := duplicateDeliveryObservationProgramDefinition
  participantProgram := duplicateDeliveryParticipantProgramDefinition
  configuration := duplicateDeliveryRuntimeConfigurationFor experiment
}

end Temporal.System.Execution.Nexus
