import Umpire.Artifact.Runtime
import Umpire.Examples.Switch

/-! RuntimeConfiguration and ExperimentRun exact v2 bytes, checksums, matrices, and closures. -/

namespace Umpire.Artifact.Tests.Runtime

open Umpire
open Umpire.Examples.Switch

#check (RuntimeConfiguration.knownGaps : RuntimeConfiguration → KnownGapSet)
#check (ExperimentRun.knownGaps : ExperimentRun → KnownGapSet)

private def id (value : String) : DefinitionId := DefinitionId.of value

private def fingerprint (value : String) : BehaviorFingerprint :=
  (BehaviorFingerprint.parse? value).getD (behaviorFingerprintOf "invalid")

private def checksum (value : String) : ArtifactChecksum :=
  (ArtifactChecksum.parse? value).getD (drivePlanChecksumOf "invalid")

private def emptyChecksum : ArtifactChecksum :=
  checksum "sha256:0000000000000000000000000000000000000000000000000000000000000000"

private def interpretationKnownGaps : KnownGapSet :=
  (KnownGapSet.ofUnordered [{
    kind := .interpretation
    code := id "switch.gap.interpretation"
  }]).toOption.getD KnownGapSet.empty

private def experimentBinding : ArtifactBinding := {
  formatVersion := compiledArtifact.formatVersion
  artifactChecksum := compiledArtifact.artifactChecksum
  behaviorFingerprint := compiledArtifact.queryBehaviorFingerprint
  provenanceChecksum := compiledArtifact.provenance.expectedChecksum
}

private def phaseLimits : List PhaseLimit := [
  { phase := .preparation, durationMilliseconds := 30000, maxAttempts := 1,
    maxRecords := 128, maxBytes := 1048576 },
  { phase := .realization, durationMilliseconds := 30000, maxAttempts := 1,
    maxRecords := 128, maxBytes := 1048576 },
  { phase := .observation, durationMilliseconds := 30000, maxAttempts := 1,
    maxRecords := 3584, maxBytes := 12582912 },
  { phase := .isolation, durationMilliseconds := 15000, maxAttempts := 1,
    maxRecords := 128, maxBytes := 1048576 },
  { phase := .cleanup, durationMilliseconds := 15000, maxAttempts := 1,
    maxRecords := 128, maxBytes := 1048576 }
]

private def runtimeConfigurationDraft : RuntimeConfiguration := {
  formatVersion := "umpire-runtime-configuration/v2"
  configurationDefinitionId := id "switch.runtime.configuration"
  behaviorFingerprint :=
    fingerprint "sha256:6b81f3a1bc1b67f699b5f2dd7bd030e08c4bcf52c656274d4b25abb374bb87df"
  experiment := experimentBinding
  authorityProfile := {
    definitionId := id "switch.runtime.profile"
    version := 2
    behaviorFingerprint :=
      fingerprint "sha256:44c1679e8770f2be478d5d4bbda5566b9a369df11fe98717bf22e665681a99ec"
    requiredCapabilityDefinitionIds := []
  }
  phaseLimits
  observation := {
    profileDefinitionId := id "switch.observation.profile"
    profileBehaviorFingerprint :=
      fingerprint "sha256:90445782b6263c18353c89a4ae0f08f7ef84317e2192e2e4dec4fb28754632f5"
    programDefinitionId := id "switch.observation.program"
    programBehaviorFingerprint :=
      fingerprint "sha256:9115e2c6fa437e90ee8bbbb326a489a01653f92c6289c23fb9b65a4fab044efc"
    mappingDefinitionId := id "switch.observation.mapping"
    mappingBehaviorFingerprint :=
      fingerprint "sha256:f9f81b529dde00a6f3ce17a0a68a039d2b61bb0d332fdd7ec26cecd7cb56c63c"
  }
  participantBindings := [{
    participantDefinitionId := id "switch.participant"
    protocolDefinitionId := id "switch.protocol"
    protocolVersion := 2
    programDefinitionId := id "switch.participant.program"
    programBehaviorFingerprint :=
      fingerprint "sha256:92489e8192608a0a88e319591737e528194b9c1239ac3d08f92bdc692aca3d31"
    capabilityDefinitionIds := [id "switch.capability.state"]
  }]
  knownGaps := KnownGapSet.empty
  provenance := {
    sourceDefinitionIds := [
      id "switch.observation.mapping",
      id "switch.observation.profile",
      id "switch.observation.program",
      id "switch.participant.program",
      id "switch.runtime.configuration",
      id "switch.runtime.profile"
    ]
    sourceLocations := [{
      path := "Umpire/Artifact/Tests/Runtime.lean"
      line := 1
      column := 1
      provenance := "lean-model"
    }]
  }
  provenanceChecksum := emptyChecksum
  artifactChecksum := emptyChecksum
}

def runtimeConfiguration : RuntimeConfiguration := runtimeConfigurationDraft.seal

private def phaseOutcome
    (phase : ExecutionPhase)
    (started finished : Nat) : PhaseOutcome := {
  phase
  status := .succeeded
  startedAtUnixMillis := some started
  finishedAtUnixMillis := some finished
  code := none
}

private def notStartedPhaseOutcome (outcome : PhaseOutcome) : PhaseOutcome := {
  phase := outcome.phase
  status := .notStarted
  startedAtUnixMillis := none
  finishedAtUnixMillis := none
  code := none
}

private def experimentRunDraft : ExperimentRun := {
  formatVersion := "umpire-experiment-run/v2"
  runIdentity := id "switch.run.1"
  behaviorFingerprint :=
    fingerprint "sha256:41e30ef6849aec9841e5af3a478e7ca4062f5229142318572b8afd9f36ec7f07"
  experiment := experimentBinding
  runtimeConfiguration := runtimeConfiguration.artifactBinding
  attempt := 1
  operationalStatus := .succeeded
  phaseOutcomes := [
    phaseOutcome .preparation 1000 1100,
    phaseOutcome .realization 1100 1200,
    phaseOutcome .observation 1200 1300,
    phaseOutcome .isolation 1300 1400,
    phaseOutcome .cleanup 1400 1500
  ]
  controlAttempts := [{
    occurrenceDefinitionId := id "switch.occurrence.flip"
    actionDefinitionId := id "switch.action.flip"
    attempt := 1
    receiptFactDefinitionId := some (id "switch.evidence.control-receipt.1")
    status := .accepted
    code := none
  }]
  sourceClosures := [
    { sourceDefinitionId := id "umpire.evidence.source.cleanup", status := .closed,
      recordCount := 1, byteCount := 64 },
    { sourceDefinitionId := id "umpire.evidence.source.control-receipt", status := .closed,
      recordCount := 1, byteCount := 128 },
    { sourceDefinitionId := id "umpire.evidence.source.history", status := .closed,
      recordCount := 2, byteCount := 512 },
    { sourceDefinitionId := id "umpire.evidence.source.participant-output", status := .closed,
      recordCount := 2, byteCount := 256 }
  ]
  cleanup := { status := .complete, openHandleCount := 0, code := none }
  limits := phaseLimits
  knownGaps := KnownGapSet.empty
  provenance := {
    sourceDefinitionIds := [
      id "switch.run.1",
      id "umpire.evidence.source.cleanup",
      id "umpire.evidence.source.control-receipt",
      id "umpire.evidence.source.history",
      id "umpire.evidence.source.participant-output"
    ]
    sourceLocations := [{
      path := "Umpire/Artifact/Tests/Runtime.lean"
      line := 1
      column := 1
      provenance := "lean-model"
    }]
  }
  provenanceChecksum := emptyChecksum
  artifactChecksum := emptyChecksum
}

def experimentRun : ExperimentRun := experimentRunDraft.seal

/-! Lean owns the authoritative RuntimeConfiguration fixture bytes. -/
example : canonicalRuntimeConfigurationBytes runtimeConfiguration =
    include_str "Fixtures/RuntimeConfigurationV2.json" := by
  native_decide

/-! Lean owns the authoritative ExperimentRun fixture bytes. -/
example : canonicalExperimentRunBytes experimentRun =
    include_str "Fixtures/ExperimentRunV2.json" := by
  native_decide

/-! Both checksum layers use the exact pretty-byte preimages. -/
example : runtimeConfiguration.hasValidChecksums && experimentRun.hasValidChecksums &&
    runtimeConfiguration.provenanceChecksum.render =
      "sha256:09745642d54e6faf89fd0c5a1a848d62fab3d8e472cc653db4fd02a96ff9e34e" &&
    runtimeConfiguration.artifactChecksum.render =
      "sha256:c4aaec3cec49a58cfb6cc085447afe5c197c2a0b0920cd7ae751a7e997858870" &&
    experimentRun.provenanceChecksum.render =
      "sha256:b879d5eba0c02a60c52e59a009c79f953310a6c49e3453ea863fddcbb07a75a9" &&
    experimentRun.artifactChecksum.render =
      "sha256:42dac535e8e87f4545ce96e0bf1de5bb947f02929c75f1978dfaafdde104ea93" := by
  native_decide

/-! The canonical values close over the exact Experiment, configuration, Limits, and controls. -/
example : runtimeConfiguration.isValidTransport &&
    runtimeConfiguration.closesExperiment compiledArtifact &&
    experimentRun.isValidTransport && experimentRun.closes compiledArtifact runtimeConfiguration := by
  native_decide

/-! Stage bounds and control receipt/code matrices reject one-field inconsistencies. -/
example :
    !({
      runtimeConfiguration with phaseLimits := runtimeConfiguration.phaseLimits.drop 1
    } : RuntimeConfiguration).seal.isValidTransport &&
    !({ experimentRun with controlAttempts := experimentRun.controlAttempts.map fun
      (attempt : ControlAttempt) =>
      { attempt with receiptFactDefinitionId := none } }).seal.isValidTransport &&
    !({ experimentRun with controlAttempts := experimentRun.controlAttempts.map fun
      (attempt : ControlAttempt) =>
      { attempt with status := .notAttempted } }).seal.isValidTransport := by
  native_decide

/-! Terminal phase/source/cleanup and exact cross-bindings reject inconsistent values. -/
example :
    !({ experimentRun with phaseOutcomes := experimentRun.phaseOutcomes.map fun
      (outcome : PhaseOutcome) =>
      if outcome.phase == ExecutionPhase.observation then
        { outcome with status := .failed } else outcome
    }).seal.isValidTransport &&
    !({ experimentRun with sourceClosures := experimentRun.sourceClosures.reverse }).seal.isValidTransport &&
    !({ experimentRun with cleanup := { status := .complete, openHandleCount := 1, code := none }
    }).seal.isValidTransport &&
    !({ experimentRun with runtimeConfiguration := experimentBinding }).seal.closes
      compiledArtifact runtimeConfiguration := by
  native_decide

/-! Operational precedence rejects contradictory summaries and preserves hard-failure dominance. -/
example :
    !({ experimentRun with operationalStatus := .failed }).seal.isValidTransport &&
    !({ experimentRun with phaseOutcomes := experimentRun.phaseOutcomes.map fun
      (outcome : PhaseOutcome) =>
      if outcome.phase == ExecutionPhase.realization then
        { outcome with status := .failed, code := some (id "switch.phase.failed") }
      else outcome
    }).seal.isValidTransport &&
    !({ experimentRun with sourceClosures := experimentRun.sourceClosures.map fun
      (closure : SourceClosure) =>
      if closure.sourceDefinitionId == id "umpire.evidence.source.cleanup" then
        { closure with status := .partiallyClosed }
      else closure
    }).seal.isValidTransport &&
    !({ experimentRun with
      operationalStatus := .incomplete
      phaseOutcomes := experimentRun.phaseOutcomes.map fun (outcome : PhaseOutcome) =>
        if outcome.phase == ExecutionPhase.realization then
          { outcome with status := .failed, code := some (id "switch.phase.failed") }
        else outcome
      cleanup := {
        status := .incomplete
        openHandleCount := 0
        code := some (id "switch.cleanup.incomplete")
      }
    }).seal.isValidTransport := by
  native_decide

/-! Known Gaps remain independent from a Run's operational summary. -/
example :
    (let configuration := { runtimeConfiguration with knownGaps := interpretationKnownGaps }.seal;
      configuration.isValidTransport &&
        (canonicalRuntimeConfigurationBytes configuration).contains
          "\"knownGaps\": [\n    {\n      \"kind\": \"interpretation\",\n      \"code\": \"switch.gap.interpretation\",\n      \"subject\": null,\n      \"detail\": null\n    }\n  ]") &&
    (let run := { experimentRun with knownGaps := interpretationKnownGaps }.seal;
      run.isValidTransport &&
        (canonicalExperimentRunBytes run).contains
          "\"knownGaps\": [\n    {\n      \"kind\": \"interpretation\",\n      \"code\": \"switch.gap.interpretation\",\n      \"subject\": null,\n      \"detail\": null\n    }\n  ]") := by
  native_decide

/-! A Run rejects phase progressions that cannot arise from the five-phase execution contract. -/
example :
    !({ experimentRun with
      operationalStatus := .incomplete
      phaseOutcomes := experimentRun.phaseOutcomes.map fun (outcome : PhaseOutcome) =>
        if outcome.phase == ExecutionPhase.preparation ||
            outcome.phase == ExecutionPhase.realization ||
            outcome.phase == ExecutionPhase.observation then
          notStartedPhaseOutcome outcome
        else outcome
    }).seal.isValidTransport &&
    !({ experimentRun with
      operationalStatus := .incomplete
      phaseOutcomes := experimentRun.phaseOutcomes.map fun (outcome : PhaseOutcome) =>
        if outcome.phase == ExecutionPhase.realization then
          notStartedPhaseOutcome outcome
        else outcome
    }).seal.isValidTransport &&
    !({ experimentRun with
      operationalStatus := .incomplete
      phaseOutcomes := experimentRun.phaseOutcomes.map fun (outcome : PhaseOutcome) =>
        if outcome.phase == ExecutionPhase.cleanup then
          notStartedPhaseOutcome outcome
        else outcome
    }).seal.isValidTransport := by
  native_decide

/-! Runtime provenance rejects the same empty, blank, and zero-position rows as Go admission. -/
example :
    !({ runtimeConfiguration with provenance := {
      runtimeConfiguration.provenance with sourceLocations := []
    }} : RuntimeConfiguration).seal.isValidTransport &&
    !({ runtimeConfiguration with provenance := {
      runtimeConfiguration.provenance with sourceLocations :=
        runtimeConfiguration.provenance.sourceLocations.map fun (source : SourceLocation) =>
          { source with path := "" }
    }} : RuntimeConfiguration).seal.isValidTransport &&
    !({ runtimeConfiguration with provenance := {
      runtimeConfiguration.provenance with sourceLocations :=
        runtimeConfiguration.provenance.sourceLocations.map fun (source : SourceLocation) =>
          { source with path := String.singleton (Char.ofNat 0x00a0) }
    }} : RuntimeConfiguration).seal.isValidTransport &&
    !({ runtimeConfiguration with provenance := {
      runtimeConfiguration.provenance with sourceLocations :=
        runtimeConfiguration.provenance.sourceLocations.map fun (source : SourceLocation) =>
          { source with path := String.singleton (Char.ofNat 0x000b) }
    }} : RuntimeConfiguration).seal.isValidTransport &&
    !({ runtimeConfiguration with provenance := {
      runtimeConfiguration.provenance with sourceLocations :=
        runtimeConfiguration.provenance.sourceLocations.map fun (source : SourceLocation) =>
          { source with path := String.singleton (Char.ofNat 0x000c) }
    }} : RuntimeConfiguration).seal.isValidTransport &&
    !({ runtimeConfiguration with provenance := {
      runtimeConfiguration.provenance with sourceLocations :=
        runtimeConfiguration.provenance.sourceLocations.map fun (source : SourceLocation) =>
          { source with provenance := "" }
    }} : RuntimeConfiguration).seal.isValidTransport &&
    !({ runtimeConfiguration with provenance := {
      runtimeConfiguration.provenance with sourceLocations :=
        runtimeConfiguration.provenance.sourceLocations.map fun (source : SourceLocation) =>
          { source with line := 0 }
    }} : RuntimeConfiguration).seal.isValidTransport &&
    !({ runtimeConfiguration with provenance := {
      runtimeConfiguration.provenance with sourceLocations :=
        runtimeConfiguration.provenance.sourceLocations.map fun (source : SourceLocation) =>
          { source with column := 0 }
    }} : RuntimeConfiguration).seal.isValidTransport := by
  native_decide

end Umpire.Artifact.Tests.Runtime
