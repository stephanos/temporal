import Umpire.Artifact.Codecs

namespace Umpire

/-! Exact inert v2 transports for runtime configuration and one bounded execution Run. -/

private def quoteRuntime (value : String) : String := Lean.Json.compress (.str value)

private def runtimeArray (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def optionalRuntimeJson (value : Option String) : String :=
  value.getD "null"

private def optionalIdJson (value : Option DefinitionId) : String :=
  optionalRuntimeJson (value.map (quoteRuntime ∘ DefinitionId.value))

private def optionalNatJson (value : Option Nat) : String :=
  optionalRuntimeJson (value.map toString)

private def definitionIdLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def canonicalDefinitionIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort definitionIdLe |>.eraseDups

/-- An exact reference to one immutable Artifact. -/
structure ArtifactBinding where
  formatVersion : String
  artifactChecksum : ArtifactChecksum
  behaviorFingerprint : BehaviorFingerprint
  provenanceChecksum : ArtifactChecksum
  deriving BEq, DecidableEq, Repr

/-- One authority-profile identity and the capabilities declared by that profile. -/
structure AuthorityProfile where
  definitionId : DefinitionId
  version : Nat
  behaviorFingerprint : BehaviorFingerprint
  requiredCapabilityDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

/-- The five closed stages shared by RuntimeConfiguration and ExperimentRun. -/
inductive ExecutionPhase where
  | preparation
  | realization
  | observation
  | isolation
  | cleanup
  deriving BEq, DecidableEq, Ord, Repr

def ExecutionPhase.name : ExecutionPhase → String
  | .preparation => "preparation"
  | .realization => "realization"
  | .observation => "observation"
  | .isolation => "isolation"
  | .cleanup => "cleanup"

def executionPhases : List ExecutionPhase :=
  [.preparation, .realization, .observation, .isolation, .cleanup]

/-- Positive resource bounds scoped to exactly one execution stage. -/
structure PhaseLimit where
  phase : ExecutionPhase
  durationMilliseconds : Nat
  maxAttempts : Nat
  maxRecords : Nat
  maxBytes : Nat
  deriving BEq, DecidableEq, Repr

/-- Exact observation-profile, program, and mapping identities used by a Run. -/
structure ObservationConfiguration where
  profileDefinitionId : DefinitionId
  profileBehaviorFingerprint : BehaviorFingerprint
  programDefinitionId : DefinitionId
  programBehaviorFingerprint : BehaviorFingerprint
  mappingDefinitionId : DefinitionId
  mappingBehaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- One participant's protocol, program, and closed capability bindings. -/
structure ParticipantBinding where
  participantDefinitionId : DefinitionId
  protocolDefinitionId : DefinitionId
  protocolVersion : Nat
  programDefinitionId : DefinitionId
  programBehaviorFingerprint : BehaviorFingerprint
  capabilityDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Environment-independent inputs that bind one exact Experiment to bounded runtime programs. -/
structure RuntimeConfiguration where
  formatVersion : String
  configurationDefinitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  experiment : ArtifactBinding
  authorityProfile : AuthorityProfile
  phaseLimits : List PhaseLimit
  observation : ObservationConfiguration
  participantBindings : List ParticipantBinding
  knownGaps : List KnownGap
  provenance : ArtifactProvenance
  provenanceChecksum : ArtifactChecksum
  artifactChecksum : ArtifactChecksum
  deriving BEq, DecidableEq, Repr

/-- Closed terminal and not-started states for one execution stage. -/
inductive PhaseOutcomeStatus where
  | notStarted
  | succeeded
  | failed
  | timedOut
  | canceled
  deriving BEq, DecidableEq, Ord, Repr

def PhaseOutcomeStatus.name : PhaseOutcomeStatus → String
  | .notStarted => "not-started"
  | .succeeded => "succeeded"
  | .failed => "failed"
  | .timedOut => "timed-out"
  | .canceled => "canceled"

/-- One phase outcome with the exact timestamp/code matrix for its status. -/
structure PhaseOutcome where
  phase : ExecutionPhase
  status : PhaseOutcomeStatus
  startedAtUnixMillis : Option Nat
  finishedAtUnixMillis : Option Nat
  code : Option DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Closed transport states for one attempted or explicitly not-attempted control. -/
inductive ControlAttemptStatus where
  | accepted
  | rejected
  | unsupported
  | failed
  | canceled
  | notAttempted
  deriving BEq, DecidableEq, Ord, Repr

def ControlAttemptStatus.name : ControlAttemptStatus → String
  | .accepted => "accepted"
  | .rejected => "rejected"
  | .unsupported => "unsupported"
  | .failed => "failed"
  | .canceled => "canceled"
  | .notAttempted => "not-attempted"

/-- One planned occurrence's bounded control attempt and its receipt-fact identity. -/
structure ControlAttempt where
  occurrenceDefinitionId : DefinitionId
  actionDefinitionId : DefinitionId
  attempt : Nat
  receiptFactDefinitionId : Option DefinitionId
  status : ControlAttemptStatus
  code : Option DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Closed source-finalization states. -/
inductive SourceClosureStatus where
  | closed
  | partiallyClosed
  | failed
  deriving BEq, DecidableEq, Ord, Repr

def SourceClosureStatus.name : SourceClosureStatus → String
  | .closed => "closed"
  | .partiallyClosed => "partial"
  | .failed => "failed"

/-- Final counts for one canonically identified evidence source. -/
structure SourceClosure where
  sourceDefinitionId : DefinitionId
  status : SourceClosureStatus
  recordCount : Nat
  byteCount : Nat
  deriving BEq, DecidableEq, Repr

/-- Closed cleanup states and their remaining open-handle count. -/
inductive CleanupStatus where
  | complete
  | incomplete
  | failed
  deriving BEq, DecidableEq, Ord, Repr

def CleanupStatus.name : CleanupStatus → String
  | .complete => "complete"
  | .incomplete => "incomplete"
  | .failed => "failed"

structure CleanupOutcome where
  status : CleanupStatus
  openHandleCount : Nat
  code : Option DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Overall execution state derived from the closed outcome matrices. -/
inductive OperationalStatus where
  | succeeded
  | incomplete
  | failed
  deriving BEq, DecidableEq, Ord, Repr

def OperationalStatus.name : OperationalStatus → String
  | .succeeded => "succeeded"
  | .incomplete => "incomplete"
  | .failed => "failed"

/-- An inert record of one bounded Run; it carries neither Property nor Claim Assessment. -/
structure ExperimentRun where
  formatVersion : String
  runIdentity : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  experiment : ArtifactBinding
  runtimeConfiguration : ArtifactBinding
  attempt : Nat
  operationalStatus : OperationalStatus
  phaseOutcomes : List PhaseOutcome
  controlAttempts : List ControlAttempt
  sourceClosures : List SourceClosure
  cleanup : CleanupOutcome
  limits : List PhaseLimit
  knownGaps : List KnownGap
  provenance : ArtifactProvenance
  provenanceChecksum : ArtifactChecksum
  artifactChecksum : ArtifactChecksum
  deriving BEq, DecidableEq, Repr

private def artifactBindingJson (binding : ArtifactBinding) : String :=
  "{\"formatVersion\":" ++ quoteRuntime binding.formatVersion ++
    ",\"artifactChecksum\":" ++ quoteRuntime binding.artifactChecksum.render ++
    ",\"behaviorFingerprint\":" ++ quoteRuntime binding.behaviorFingerprint.render ++
    ",\"provenanceChecksum\":" ++ quoteRuntime binding.provenanceChecksum.render ++ "}"

private def authorityProfileJson (profile : AuthorityProfile) : String :=
  "{\"definitionId\":" ++ quoteRuntime profile.definitionId.value ++
    ",\"version\":" ++ toString profile.version ++
    ",\"behaviorFingerprint\":" ++ quoteRuntime profile.behaviorFingerprint.render ++
    ",\"requiredCapabilityDefinitionIds\":" ++ runtimeArray
      (profile.requiredCapabilityDefinitionIds.map (quoteRuntime ∘ DefinitionId.value)) ++ "}"

private def phaseLimitJson (limit : PhaseLimit) : String :=
  "{\"phase\":" ++ quoteRuntime limit.phase.name ++
    ",\"durationMilliseconds\":" ++ toString limit.durationMilliseconds ++
    ",\"maxAttempts\":" ++ toString limit.maxAttempts ++
    ",\"maxRecords\":" ++ toString limit.maxRecords ++
    ",\"maxBytes\":" ++ toString limit.maxBytes ++ "}"

private def observationConfigurationJson (observation : ObservationConfiguration) : String :=
  "{\"profileDefinitionId\":" ++ quoteRuntime observation.profileDefinitionId.value ++
    ",\"profileBehaviorFingerprint\":" ++ quoteRuntime observation.profileBehaviorFingerprint.render ++
    ",\"programDefinitionId\":" ++ quoteRuntime observation.programDefinitionId.value ++
    ",\"programBehaviorFingerprint\":" ++ quoteRuntime observation.programBehaviorFingerprint.render ++
    ",\"mappingDefinitionId\":" ++ quoteRuntime observation.mappingDefinitionId.value ++
    ",\"mappingBehaviorFingerprint\":" ++ quoteRuntime observation.mappingBehaviorFingerprint.render ++ "}"

private def participantBindingJson (binding : ParticipantBinding) : String :=
  "{\"participantDefinitionId\":" ++ quoteRuntime binding.participantDefinitionId.value ++
    ",\"protocolDefinitionId\":" ++ quoteRuntime binding.protocolDefinitionId.value ++
    ",\"protocolVersion\":" ++ toString binding.protocolVersion ++
    ",\"programDefinitionId\":" ++ quoteRuntime binding.programDefinitionId.value ++
    ",\"programBehaviorFingerprint\":" ++ quoteRuntime binding.programBehaviorFingerprint.render ++
    ",\"capabilityDefinitionIds\":" ++ runtimeArray
      (binding.capabilityDefinitionIds.map (quoteRuntime ∘ DefinitionId.value)) ++ "}"

private def runtimeConfigurationContentJson (configuration : RuntimeConfiguration) : String :=
  "{\"formatVersion\":" ++ quoteRuntime configuration.formatVersion ++
    ",\"configurationDefinitionId\":" ++ quoteRuntime configuration.configurationDefinitionId.value ++
    ",\"behaviorFingerprint\":" ++ quoteRuntime configuration.behaviorFingerprint.render ++
    ",\"experiment\":" ++ artifactBindingJson configuration.experiment ++
    ",\"authorityProfile\":" ++ authorityProfileJson configuration.authorityProfile ++
    ",\"phaseLimits\":" ++ runtimeArray (configuration.phaseLimits.map phaseLimitJson) ++
    ",\"observation\":" ++ observationConfigurationJson configuration.observation ++
    ",\"participantBindings\":" ++
      runtimeArray (configuration.participantBindings.map participantBindingJson) ++
    ",\"knownGaps\":" ++ runtimeArray (configuration.knownGaps.map canonicalKnownGapJson) ++
    ",\"provenance\":" ++ configuration.provenance.canonicalJson ++
    ",\"provenanceChecksum\":" ++ quoteRuntime configuration.provenanceChecksum.render ++ "}"

private def phaseOutcomeJson (outcome : PhaseOutcome) : String :=
  "{\"phase\":" ++ quoteRuntime outcome.phase.name ++
    ",\"status\":" ++ quoteRuntime outcome.status.name ++
    ",\"startedAtUnixMillis\":" ++ optionalNatJson outcome.startedAtUnixMillis ++
    ",\"finishedAtUnixMillis\":" ++ optionalNatJson outcome.finishedAtUnixMillis ++
    ",\"code\":" ++ optionalIdJson outcome.code ++ "}"

private def controlAttemptJson (attempt : ControlAttempt) : String :=
  "{\"occurrenceDefinitionId\":" ++ quoteRuntime attempt.occurrenceDefinitionId.value ++
    ",\"actionDefinitionId\":" ++ quoteRuntime attempt.actionDefinitionId.value ++
    ",\"attempt\":" ++ toString attempt.attempt ++
    ",\"receiptFactDefinitionId\":" ++ optionalIdJson attempt.receiptFactDefinitionId ++
    ",\"status\":" ++ quoteRuntime attempt.status.name ++
    ",\"code\":" ++ optionalIdJson attempt.code ++ "}"

private def sourceClosureJson (closure : SourceClosure) : String :=
  "{\"sourceDefinitionId\":" ++ quoteRuntime closure.sourceDefinitionId.value ++
    ",\"status\":" ++ quoteRuntime closure.status.name ++
    ",\"recordCount\":" ++ toString closure.recordCount ++
    ",\"byteCount\":" ++ toString closure.byteCount ++ "}"

private def cleanupOutcomeJson (cleanup : CleanupOutcome) : String :=
  "{\"status\":" ++ quoteRuntime cleanup.status.name ++
    ",\"openHandleCount\":" ++ toString cleanup.openHandleCount ++
    ",\"code\":" ++ optionalIdJson cleanup.code ++ "}"

private def experimentRunContentJson (run : ExperimentRun) : String :=
  "{\"formatVersion\":" ++ quoteRuntime run.formatVersion ++
    ",\"runIdentity\":" ++ quoteRuntime run.runIdentity.value ++
    ",\"behaviorFingerprint\":" ++ quoteRuntime run.behaviorFingerprint.render ++
    ",\"experiment\":" ++ artifactBindingJson run.experiment ++
    ",\"runtimeConfiguration\":" ++ artifactBindingJson run.runtimeConfiguration ++
    ",\"attempt\":" ++ toString run.attempt ++
    ",\"operationalStatus\":" ++ quoteRuntime run.operationalStatus.name ++
    ",\"phaseOutcomes\":" ++ runtimeArray (run.phaseOutcomes.map phaseOutcomeJson) ++
    ",\"controlAttempts\":" ++ runtimeArray (run.controlAttempts.map controlAttemptJson) ++
    ",\"sourceClosures\":" ++ runtimeArray (run.sourceClosures.map sourceClosureJson) ++
    ",\"cleanup\":" ++ cleanupOutcomeJson run.cleanup ++
    ",\"limits\":" ++ runtimeArray (run.limits.map phaseLimitJson) ++
    ",\"knownGaps\":" ++ runtimeArray (run.knownGaps.map canonicalKnownGapJson) ++
    ",\"provenance\":" ++ run.provenance.canonicalJson ++
    ",\"provenanceChecksum\":" ++ quoteRuntime run.provenanceChecksum.render ++ "}"

/-- Derive the checksum of exact pretty provenance bytes shared by all v2 runtime transports. -/
def ArtifactProvenance.expectedChecksum (provenance : ArtifactProvenance) : ArtifactChecksum :=
  provenanceChecksumOf (Json.prettyBytes provenance.canonicalJson)

def RuntimeConfiguration.expectedArtifactChecksum
    (configuration : RuntimeConfiguration) : ArtifactChecksum :=
  runtimeConfigurationChecksumOf (Json.prettyBytes (runtimeConfigurationContentJson configuration))

def RuntimeConfiguration.seal (configuration : RuntimeConfiguration) : RuntimeConfiguration :=
  let withProvenance := {
    configuration with provenanceChecksum := configuration.provenance.expectedChecksum
  }
  { withProvenance with artifactChecksum := withProvenance.expectedArtifactChecksum }

def RuntimeConfiguration.hasValidChecksums (configuration : RuntimeConfiguration) : Bool :=
  configuration.provenanceChecksum == configuration.provenance.expectedChecksum &&
    configuration.artifactChecksum == configuration.expectedArtifactChecksum

def canonicalRuntimeConfigurationJson (configuration : RuntimeConfiguration) : String :=
  let content := runtimeConfigurationContentJson configuration
  Json.pretty ((content.dropEnd 1).toString ++
    ",\"artifactChecksum\":" ++ quoteRuntime configuration.artifactChecksum.render ++ "}")

def canonicalRuntimeConfigurationBytes (configuration : RuntimeConfiguration) : String :=
  canonicalRuntimeConfigurationJson configuration ++ "\n"

def ExperimentRun.expectedArtifactChecksum (run : ExperimentRun) : ArtifactChecksum :=
  experimentRunChecksumOf (Json.prettyBytes (experimentRunContentJson run))

def ExperimentRun.seal (run : ExperimentRun) : ExperimentRun :=
  let withProvenance := { run with provenanceChecksum := run.provenance.expectedChecksum }
  { withProvenance with artifactChecksum := withProvenance.expectedArtifactChecksum }

def ExperimentRun.hasValidChecksums (run : ExperimentRun) : Bool :=
  run.provenanceChecksum == run.provenance.expectedChecksum &&
    run.artifactChecksum == run.expectedArtifactChecksum

def canonicalExperimentRunJson (run : ExperimentRun) : String :=
  let content := experimentRunContentJson run
  Json.pretty ((content.dropEnd 1).toString ++
    ",\"artifactChecksum\":" ++ quoteRuntime run.artifactChecksum.render ++ "}")

def canonicalExperimentRunBytes (run : ExperimentRun) : String :=
  canonicalExperimentRunJson run ++ "\n"

private def artifactBindingOfExperiment (experiment : ExperimentSpec) : ArtifactBinding := {
  formatVersion := experiment.formatVersion
  artifactChecksum := experiment.artifactChecksum
  behaviorFingerprint := experiment.queryBehaviorFingerprint
  provenanceChecksum := experiment.provenance.expectedChecksum
}

def RuntimeConfiguration.artifactBinding
    (configuration : RuntimeConfiguration) : ArtifactBinding := {
  formatVersion := configuration.formatVersion
  artifactChecksum := configuration.artifactChecksum
  behaviorFingerprint := configuration.behaviorFingerprint
  provenanceChecksum := configuration.provenanceChecksum
}

private def phaseLimitsValid (limits : List PhaseLimit) : Bool :=
  limits.map PhaseLimit.phase == executionPhases && limits.all fun limit =>
    limit.durationMilliseconds > 0 && limit.maxAttempts > 0 &&
      limit.maxRecords > 0 && limit.maxBytes > 0

private def idsCanonical (ids : List DefinitionId) : Bool :=
  ids == canonicalDefinitionIds ids && ids.all DefinitionId.isNamespaced

private def participantBindingLe (left right : ParticipantBinding) : Bool :=
  decide (left.participantDefinitionId.value ≤ right.participantDefinitionId.value)

private def participantBindingsValid (bindings : List ParticipantBinding) : Bool :=
  bindings != [] && bindings == bindings.mergeSort participantBindingLe &&
    (bindings.map ParticipantBinding.participantDefinitionId).eraseDups.length == bindings.length &&
    bindings.all fun binding =>
      binding.participantDefinitionId.isNamespaced && binding.protocolDefinitionId.isNamespaced &&
        binding.protocolVersion > 0 && binding.programDefinitionId.isNamespaced &&
        idsCanonical binding.capabilityDefinitionIds

private def sourceLocationLe (left right : SourceLocation) : Bool :=
  decide (left.path < right.path) ||
    (left.path == right.path && decide (left.line < right.line)) ||
    (left.path == right.path && left.line == right.line && decide (left.column < right.column)) ||
    (left.path == right.path && left.line == right.line && left.column == right.column &&
      decide (left.provenance ≤ right.provenance))

private def provenanceValid (provenance : ArtifactProvenance) : Bool :=
  idsCanonical provenance.sourceDefinitionIds &&
    provenance.sourceLocations != [] &&
    provenance.sourceLocations == provenance.sourceLocations.mergeSort sourceLocationLe &&
    provenance.sourceLocations.eraseDups.length == provenance.sourceLocations.length &&
    provenance.sourceLocations.all fun source =>
      !source.path.trimAscii.isEmpty && !source.provenance.trimAscii.isEmpty &&
        source.line > 0 && source.column > 0

private def knownGapsValid (knownGaps : List KnownGap) : Bool :=
  (validateKnownGaps knownGaps).isOk

/-- Check the closed RuntimeConfiguration transport without performing authorization. -/
def RuntimeConfiguration.isValidTransport (configuration : RuntimeConfiguration) : Bool :=
  configuration.formatVersion == "umpire-runtime-configuration/v2" &&
    configuration.configurationDefinitionId.isNamespaced &&
    configuration.experiment.formatVersion == "umpire-experiment/v2" &&
    configuration.authorityProfile.definitionId.isNamespaced &&
    configuration.authorityProfile.version > 0 &&
    idsCanonical configuration.authorityProfile.requiredCapabilityDefinitionIds &&
    phaseLimitsValid configuration.phaseLimits &&
    configuration.observation.profileDefinitionId.isNamespaced &&
    configuration.observation.programDefinitionId.isNamespaced &&
    configuration.observation.mappingDefinitionId.isNamespaced &&
    participantBindingsValid configuration.participantBindings &&
    knownGapsValid configuration.knownGaps && provenanceValid configuration.provenance &&
    configuration.hasValidChecksums

/-- Close the exact Experiment binding and its capability set. -/
def RuntimeConfiguration.closesExperiment
    (configuration : RuntimeConfiguration)
    (experiment : ExperimentSpec) : Bool :=
  let capabilities := canonicalDefinitionIds
    (configuration.authorityProfile.requiredCapabilityDefinitionIds ++
      configuration.participantBindings.flatMap ParticipantBinding.capabilityDefinitionIds)
  configuration.experiment == artifactBindingOfExperiment experiment &&
    capabilities == experiment.plan.capabilityRequirementDefinitionIds

private def terminalPhaseValid (outcome : PhaseOutcome) (requiresCode : Bool) : Bool :=
  match outcome.startedAtUnixMillis, outcome.finishedAtUnixMillis with
  | some started, some finished =>
      started ≤ finished && requiresCode == outcome.code.isSome &&
        outcome.code.all DefinitionId.isNamespaced
  | _, _ => false

private def phaseOutcomeValid (outcome : PhaseOutcome) : Bool :=
  match outcome.status with
  | .notStarted =>
      outcome.startedAtUnixMillis.isNone && outcome.finishedAtUnixMillis.isNone &&
        outcome.code.isNone
  | .succeeded => terminalPhaseValid outcome false
  | .failed | .timedOut | .canceled => terminalPhaseValid outcome true

private def phaseProgressionValid : List PhaseOutcome → Bool
  | preparation :: realization :: observation :: _isolation :: cleanup :: [] =>
      preparation.status != .notStarted &&
        (preparation.status == .succeeded || realization.status == .notStarted) &&
        ((realization.status == .notStarted) == (observation.status == .notStarted)) &&
        cleanup.status != .notStarted
  | _ => false

private def controlAttemptValid (runAttempt : Nat) (attempt : ControlAttempt) : Bool :=
  attempt.occurrenceDefinitionId.isNamespaced && attempt.actionDefinitionId.isNamespaced &&
    attempt.attempt > 0 && attempt.attempt == runAttempt &&
    match attempt.status with
    | .notAttempted => attempt.receiptFactDefinitionId.isNone && attempt.code.isNone
    | .accepted =>
        attempt.receiptFactDefinitionId.any DefinitionId.isNamespaced && attempt.code.isNone
    | .rejected | .unsupported | .failed | .canceled =>
        attempt.receiptFactDefinitionId.any DefinitionId.isNamespaced &&
          attempt.code.any DefinitionId.isNamespaced

private def controlAttemptLe (left right : ControlAttempt) : Bool :=
  decide (left.occurrenceDefinitionId.value < right.occurrenceDefinitionId.value) ||
    (left.occurrenceDefinitionId == right.occurrenceDefinitionId && left.attempt ≤ right.attempt)

private def controlAttemptsValid (runAttempt : Nat) (attempts : List ControlAttempt) : Bool :=
  let keys := attempts.map fun attempt => (attempt.occurrenceDefinitionId, attempt.attempt)
  let receipts := attempts.filterMap ControlAttempt.receiptFactDefinitionId
  attempts == attempts.mergeSort controlAttemptLe && keys.eraseDups.length == keys.length &&
    receipts.eraseDups.length == receipts.length && attempts.all (controlAttemptValid runAttempt)

private def sourceClosureLe (left right : SourceClosure) : Bool :=
  decide (left.sourceDefinitionId.value ≤ right.sourceDefinitionId.value)

private def sourceClosuresValid (closures : List SourceClosure) : Bool :=
  closures != [] && closures == closures.mergeSort sourceClosureLe &&
    (closures.map SourceClosure.sourceDefinitionId).eraseDups.length == closures.length &&
    closures.all fun closure => closure.sourceDefinitionId.isNamespaced

private def cleanupValid (cleanup : CleanupOutcome) : Bool :=
  match cleanup.status with
  | .complete => cleanup.openHandleCount == 0 && cleanup.code.isNone
  | .incomplete | .failed => cleanup.code.any DefinitionId.isNamespaced

/-! This validates the declared summary; it does not construct or normalize a Run. -/
private def expectedOperationalStatus (run : ExperimentRun) : OperationalStatus :=
  if run.phaseOutcomes.any fun outcome => outcome.status == .failed then .failed
  else if run.controlAttempts.any fun attempt =>
      attempt.status == .rejected || attempt.status == .unsupported || attempt.status == .failed
    then .failed
  else if run.sourceClosures.any fun closure => closure.status == .failed then .failed
  else if run.cleanup.status == .failed then .failed
  else if run.phaseOutcomes.any fun outcome => outcome.status != .succeeded then .incomplete
  else if run.controlAttempts.any fun attempt => attempt.status != .accepted then .incomplete
  else if run.sourceClosures.any fun closure => closure.status != .closed then .incomplete
  else if run.cleanup.status != .complete || run.knownGaps != [] then .incomplete
  else .succeeded

/-- Check the closed Run transport while leaving evidence-fact resolution to RawEvidence admission. -/
def ExperimentRun.isValidTransport (run : ExperimentRun) : Bool :=
  run.formatVersion == "umpire-experiment-run/v2" && run.runIdentity.isNamespaced &&
    run.experiment.formatVersion == "umpire-experiment/v2" &&
    run.runtimeConfiguration.formatVersion == "umpire-runtime-configuration/v2" &&
    run.attempt > 0 && run.phaseOutcomes.map PhaseOutcome.phase == executionPhases &&
    run.phaseOutcomes.all phaseOutcomeValid && phaseProgressionValid run.phaseOutcomes &&
    controlAttemptsValid run.attempt run.controlAttempts &&
    sourceClosuresValid run.sourceClosures && cleanupValid run.cleanup &&
    phaseLimitsValid run.limits && knownGapsValid run.knownGaps && provenanceValid run.provenance &&
    run.operationalStatus == expectedOperationalStatus run && run.hasValidChecksums

private def plannedControlLe (left right : DefinitionId × DefinitionId) : Bool :=
  decide (left.1.value ≤ right.1.value)

/-- Close a Run over the exact Experiment, RuntimeConfiguration, Limits, and planned controls. -/
def ExperimentRun.closes
    (run : ExperimentRun)
    (experiment : ExperimentSpec)
    (configuration : RuntimeConfiguration) : Bool :=
  let planned := experiment.plan.linearExtension.map fun occurrence =>
    (occurrence.definitionId, occurrence.actionDefinitionId)
  let attempted := run.controlAttempts.map fun attempt =>
    (attempt.occurrenceDefinitionId, attempt.actionDefinitionId)
  configuration.closesExperiment experiment && run.experiment == artifactBindingOfExperiment experiment &&
    run.runtimeConfiguration == configuration.artifactBinding && run.limits == configuration.phaseLimits &&
    attempted == planned.mergeSort plannedControlLe

end Umpire
