import Temporal.System.Execution.Nexus

namespace Temporal.System.Execution.NexusTests

open _root_.Umpire
open Temporal.System.Execution.Nexus

private def id (value : String) : DefinitionId := DefinitionId.of value

private def programErrorKindOf
    (result : Except NexusExecutionError CheckedParticipantProgram) :
    Option NexusExecutionErrorKind :=
  match result with
  | .error error => some error.kind
  | .ok _ => none

private def mapOccurrence
    (change : ProgramOccurrence → ProgramOccurrence)
    (program : ParticipantProgramDefinition) : ParticipantProgramDefinition := {
  program with occurrences := program.occurrences.map change
}

private def programIdentityChanges (candidate : ParticipantProgramDefinition) : Bool :=
  canonicalParticipantProgramDefinition.expectedBehaviorFingerprint !=
    candidate.expectedBehaviorFingerprint

#check (callerClosureProgram : CheckedParticipantProgram)
#check (executionDefinitionFor : ExperimentSpec → ExecutionDefinition)
#check (checkExecution : ExecutionDefinition → Except NexusExecutionError CheckedExecution)
#check (CheckedExecution.artifactSet : CheckedExecution → ArtifactSet)

example : callerClosureProgram.definition = canonicalParticipantProgramDefinition ∧
    callerClosureProgram.definition.targetDefinitionIds = [targetDefinitionId] ∧
    callerClosureProgram.definition.actionDefinitionIds = [actionDefinitionId] ∧
    callerClosureProgram.definition.occurrences = [{
      definitionId := id "workflow-nexus.occurrence.force-close"
      actionDefinitionId := actionDefinitionId
      position := 1
    }] ∧
    callerClosureProgram.definition.requestedFaultDefinitionIds = [] ∧
    callerClosureProgram.definition.commands = [.prepare, .realize, .observe, .cleanup] ∧
    evidenceSourceDefinitionIds = [
      id "umpire.evidence.source.cleanup",
      id "umpire.evidence.source.control-receipt",
      id "umpire.evidence.source.history",
      id "umpire.evidence.source.participant-output"
    ] := by
  native_decide

example : experimentBinding.formatVersion = "umpire-experiment/v2" ∧
    experimentBinding.artifactChecksum.render =
      "sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8" ∧
    experimentBinding.behaviorFingerprint.render =
      "sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af" ∧
    experimentBinding.provenanceChecksum.render =
      "sha256:f7a6ebefca8202c6a7c467fd516e54d162c7d1f254c6c9a1f004a7f0b4135ab8" ∧
    canonicalObservationProgramDefinition.profile.behaviorFingerprint.render =
      "sha256:ac3cf245ad3e4a311eb6372be9caf49301c7e8ad3ee1b1875a53ea69d1ddc105" ∧
    canonicalObservationProgramDefinition.reference.behaviorFingerprint.render =
      "sha256:1ab36fdcd2978dec901678491646ec67fe0fc1d3bd1883e599bc2c53810b3480" ∧
    canonicalObservationProgramDefinition.mapping.behaviorFingerprint.render =
      "sha256:608e4db6c3a29d0f953640621ee34d34e16b0090309e85804e21f0cb21be30a2" ∧
    callerClosureProgram.definition.reference.behaviorFingerprint.render =
      "sha256:f2f1a9a1346576b4d8c6b0b4f7f6c8a138461f90c168ab57747b316807666e56" := by
  native_decide

example :
    programErrorKindOf (checkParticipantProgram {
      canonicalParticipantProgramDefinition with
      participantDefinitionIds := canonicalParticipantProgramDefinition.participantDefinitionIds ++
        [id "temporal.nexus.participant.extra"]
    }) = some .participant ∧
    programErrorKindOf (checkParticipantProgram {
      canonicalParticipantProgramDefinition with protocolVersion := 3
    }) = some .protocol ∧
    programErrorKindOf (checkParticipantProgram {
      canonicalParticipantProgramDefinition with
      capabilityDefinitionIds := canonicalParticipantProgramDefinition.capabilityDefinitionIds.drop 1
    }) = some .capability ∧
    programErrorKindOf (checkParticipantProgram {
      canonicalParticipantProgramDefinition with
      occurrences := canonicalParticipantProgramDefinition.occurrences.map fun occurrence =>
        { occurrence with actionDefinitionId := id "workflow.action.unsupported" }
    }) = some .action ∧
    programErrorKindOf (checkParticipantProgram {
      canonicalParticipantProgramDefinition with
      requestedFaultDefinitionIds := [id "workflow-nexus.fault.unsupported"]
    }) = some .fault := by
  native_decide

example : [
    programIdentityChanges {
      canonicalParticipantProgramDefinition with
      reference := {
        canonicalParticipantProgramDefinition.reference with
        definitionId := id "temporal.nexus.participant-program.mutated"
      }
    },
    programIdentityChanges {
      canonicalParticipantProgramDefinition with
      reference := { canonicalParticipantProgramDefinition.reference with version := 2 }
    },
    programIdentityChanges {
      canonicalParticipantProgramDefinition with
      participantDefinitionIds := [id "temporal.nexus.participant.mutated"]
    },
    programIdentityChanges {
      canonicalParticipantProgramDefinition with
      protocolDefinitionId := id "umpire.participant-protocol.mutated"
    },
    programIdentityChanges { canonicalParticipantProgramDefinition with protocolVersion := 3 },
    programIdentityChanges {
      canonicalParticipantProgramDefinition with
      targetDefinitionIds := [id "workflow-nexus.target.mutated"]
    },
    programIdentityChanges {
      canonicalParticipantProgramDefinition with
      actionDefinitionIds := [id "workflow.action.mutated"]
    },
    programIdentityChanges (mapOccurrence (fun occurrence => {
      occurrence with definitionId := id "workflow-nexus.occurrence.mutated"
    }) canonicalParticipantProgramDefinition),
    programIdentityChanges (mapOccurrence (fun occurrence => {
      occurrence with actionDefinitionId := id "workflow.action.mutated"
    }) canonicalParticipantProgramDefinition),
    programIdentityChanges (mapOccurrence (fun occurrence => { occurrence with position := 2 })
      canonicalParticipantProgramDefinition),
    programIdentityChanges {
      canonicalParticipantProgramDefinition with
      requestedFaultDefinitionIds := [id "workflow-nexus.fault.mutated"]
    },
    programIdentityChanges {
      canonicalParticipantProgramDefinition with
      capabilityDefinitionIds := canonicalParticipantProgramDefinition.capabilityDefinitionIds.drop 1
    },
    programIdentityChanges {
      canonicalParticipantProgramDefinition with commands := [.prepare, .realize, .cleanup]
    },
    programIdentityChanges {
      canonicalParticipantProgramDefinition with
      evidenceSourceDefinitionIds := canonicalParticipantProgramDefinition.evidenceSourceDefinitionIds.drop 1
    }
  ].all (fun changed => changed) ∧
    canonicalParticipantProgramDefinition.expectedBehaviorFingerprint =
      ({ canonicalParticipantProgramDefinition with
        provenance := [{ canonicalProgramSource with line := 2 }] } :
          ParticipantProgramDefinition).expectedBehaviorFingerprint := by
  native_decide

end Temporal.System.Execution.NexusTests
