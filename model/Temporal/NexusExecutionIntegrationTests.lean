import Temporal.Feature.Nexus.Experimental.CallerClosure
import Temporal.Feature.Nexus.Experimental.CallerClosureFault
import Temporal.System.Execution.Nexus

namespace Temporal.NexusExecutionIntegrationTests

open _root_.Umpire
open Temporal.System.Execution.Nexus

private def id (value : String) : DefinitionId := DefinitionId.of value

private def canonicalExperiment : ExperimentSpec :=
  Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact

private def canonicalExecutionDefinition : ExecutionDefinition :=
  executionDefinitionFor canonicalExperiment

private def canonicalRuntimeConfiguration : RuntimeConfiguration :=
  runtimeConfigurationFor canonicalExperiment

private def faultedExperiments : List ExperimentSpec :=
  Temporal.Feature.Nexus.Experimental.CallerClosureFault.batchResult.toOption.get
    (by native_decide)

private def faultedExperiment : ExperimentSpec :=
  faultedExperiments.tail.head?.get (by native_decide)

private def faultedExecutionDefinition : ExecutionDefinition :=
  duplicateDeliveryExecutionDefinitionFor faultedExperiment

private theorem duplicateDeliveryExecution_isSome :
    (checkExecution faultedExecutionDefinition).toOption.isSome = true := by
  native_decide

private def duplicateDeliveryExecution : CheckedExecution :=
  (checkExecution faultedExecutionDefinition).toOption.get
    duplicateDeliveryExecution_isSome

private theorem callerClosureExecution_isSome :
    (checkExecution canonicalExecutionDefinition).toOption.isSome = true := by
  native_decide

private def callerClosureExecution : CheckedExecution :=
  (checkExecution canonicalExecutionDefinition).toOption.get callerClosureExecution_isSome

private def errorKindOf
    (result : Except NexusExecutionError CheckedExecution) : Option NexusExecutionErrorKind :=
  match result with
  | .error error => some error.kind
  | .ok _ => none

private def mapOccurrence
    (change : ProgramOccurrence → ProgramOccurrence)
    (program : ParticipantProgramDefinition) : ParticipantProgramDefinition := {
  program with occurrences := program.occurrences.map change
}

private def mapParticipantBinding
    (change : ParticipantBinding → ParticipantBinding)
    (configuration : RuntimeConfiguration) : RuntimeConfiguration :=
  { configuration with participantBindings := configuration.participantBindings.map change } |>.seal

private def expectedExperimentJson : String :=
  include_str "../../tools/umpire/temporal/nexus/testdata/caller-closure-input-set/artifacts/experiment.json"

private def expectedRuntimeConfigurationJson : String :=
  include_str "../../tools/umpire/temporal/nexus/testdata/caller-closure-input-set/artifacts/runtime-configuration.json"

private def expectedManifestJson : String :=
  include_str "../../tools/umpire/temporal/nexus/testdata/caller-closure-input-set/manifest.json"

private def expectedDuplicateDeliveryExperimentJson : String :=
  include_str "../../tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set/artifacts/experiment.json"

private def expectedDuplicateDeliveryRuntimeConfigurationJson : String :=
  include_str "../../tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set/artifacts/runtime-configuration.json"

private def expectedDuplicateDeliveryManifestJson : String :=
  include_str "../../tools/umpire/temporal/nexus/testdata/caller-closure-duplicate-delivery-input-set/manifest.json"

private def actualManifestJson : String :=
  match callerClosureExecution.artifactSet.manifest? with
  | some manifest => canonicalArtifactSetManifestBytes manifest
  | none => ""

private def actualDuplicateDeliveryManifestJson : String :=
  match duplicateDeliveryExecution.artifactSet.manifest? with
  | some manifest => canonicalArtifactSetManifestBytes manifest
  | none => ""

private def programIdentityChanges (candidate : ParticipantProgramDefinition) : Bool :=
  canonicalParticipantProgramDefinition.expectedBehaviorFingerprint !=
    candidate.expectedBehaviorFingerprint

private def configurationIdentityChanges (candidate : RuntimeConfiguration) : Bool :=
  expectedRuntimeConfigurationBehaviorFingerprint canonicalRuntimeConfiguration !=
    expectedRuntimeConfigurationBehaviorFingerprint candidate

#check (callerClosureProgram : CheckedParticipantProgram)
#check (callerClosureExecution : CheckedExecution)
#check (callerClosureExecution.configuration : RuntimeConfiguration)
#check (callerClosureExecution.artifactSet : ArtifactSet)
#check (callerClosureExecution.observationProgram : ObservationProgramDefinition)

example : callerClosureExecution.experiment =
      Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact ∧
    callerClosureExecution.program.targetDefinitionIds =
      [Temporal.Feature.Nexus.Experimental.CallerClosure.targetId] ∧
    callerClosureExecution.program.actionDefinitionIds =
      [Temporal.Feature.Nexus.Experimental.CallerClosure.forceCloseActionId] ∧
    callerClosureExecution.program.occurrences = [{
      definitionId := id "workflow-nexus.occurrence.force-close"
      actionDefinitionId :=
        Temporal.Feature.Nexus.Experimental.CallerClosure.forceCloseActionId
      position := 1
    }] ∧
    callerClosureExecution.program.requestedFaultDefinitionIds = [] ∧
    callerClosureExecution.program.commands = [.prepare, .realize, .observe, .cleanup] := by
  native_decide

example : callerClosureExecution.configuration.isValidTransport ∧
    callerClosureExecution.configuration.closesExperiment
      Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact ∧
    callerClosureExecution.artifactSet.isValidClosure ∧
    callerClosureExecution.artifactSet.manifest?.isSome := by
  native_decide

example : canonicalExperimentSpecBytes callerClosureExecution.experiment = expectedExperimentJson ∧
    canonicalRuntimeConfigurationBytes callerClosureExecution.configuration =
      expectedRuntimeConfigurationJson ∧
    actualManifestJson = expectedManifestJson := by
  native_decide

example : errorKindOf (checkExecution canonicalExecutionDefinition) = none := by
  native_decide

example : errorKindOf (checkExecution faultedExecutionDefinition) = none ∧
    duplicateDeliveryExecution.experiment = faultedExperiment ∧
    duplicateDeliveryExecution.program = duplicateDeliveryProgram.definition ∧
    duplicateDeliveryExecution.observationProgram =
      duplicateDeliveryObservationProgramDefinition ∧
    duplicateDeliveryExecution.configuration =
      duplicateDeliveryRuntimeConfigurationFor faultedExperiment ∧
    duplicateDeliveryExecution.artifactSet.isValidClosure := by
  native_decide

example : canonicalExperimentSpecBytes duplicateDeliveryExecution.experiment =
      expectedDuplicateDeliveryExperimentJson ∧
    canonicalRuntimeConfigurationBytes duplicateDeliveryExecution.configuration =
      expectedDuplicateDeliveryRuntimeConfigurationJson ∧
    actualDuplicateDeliveryManifestJson = expectedDuplicateDeliveryManifestJson ∧
    canonicalExperimentSpecBytes callerClosureExecution.experiment = expectedExperimentJson ∧
    canonicalRuntimeConfigurationBytes callerClosureExecution.configuration =
      expectedRuntimeConfigurationJson ∧
    actualManifestJson = expectedManifestJson := by
  native_decide

example : duplicateDeliveryExecution.experiment.artifactBinding !=
      callerClosureExecution.experiment.artifactBinding ∧
    duplicateDeliveryExecution.program.reference !=
      callerClosureExecution.program.reference ∧
    duplicateDeliveryExecution.configuration.artifactBinding !=
      callerClosureExecution.configuration.artifactBinding ∧
    duplicateDeliveryExecution.artifactSet.manifest? !=
      callerClosureExecution.artifactSet.manifest? := by
  native_decide

example :
    errorKindOf (checkExecution {
      canonicalExecutionDefinition with experiment := faultedExperiment
    }) = some .fault ∧
    errorKindOf (checkExecution {
      faultedExecutionDefinition with experiment := canonicalExperiment
    }) = some .fault ∧
    errorKindOf (checkExecution {
      faultedExecutionDefinition with
      participantProgram := canonicalParticipantProgramDefinition
    }) = some .fault ∧
    errorKindOf (checkExecution {
      faultedExecutionDefinition with
      observationProgram := canonicalObservationProgramDefinition
    }) = some .reference ∧
    errorKindOf (checkExecution {
      faultedExecutionDefinition with configuration := canonicalRuntimeConfiguration
    }) = some .inputSet := by
  native_decide

example :
    errorKindOf (checkExecution {
      faultedExecutionDefinition with
      experiment := {
        faultedExperiment with
        plan := { faultedExperiment.plan with requestedFaults := [] }
      }
    }) = some .fault ∧
    errorKindOf (checkExecution {
      faultedExecutionDefinition with
      experiment := {
        faultedExperiment with
        plan := { faultedExperiment.plan with requestedFaults :=
          faultedExperiment.plan.requestedFaults ++ faultedExperiment.plan.requestedFaults }
      }
    }) = some .fault ∧
    errorKindOf (checkExecution {
      faultedExecutionDefinition with
      experiment := {
        faultedExperiment with
        plan := { faultedExperiment.plan with requestedFaults :=
          faultedExperiment.plan.requestedFaults.map fun fault =>
            { fault with definitionId := id "temporal.nexus.caller-closure.fault.other" } }
      }
    }) = some .fault ∧
    errorKindOf (checkExecution {
      faultedExecutionDefinition with
      experiment := {
        faultedExperiment with
        plan := { faultedExperiment.plan with requestedFaults :=
          faultedExperiment.plan.requestedFaults.map fun fault =>
            { fault with value := "workflow-nexus.occurrence.other" } }
      }
    }) = some .fault := by
  native_decide

example : callerClosureExecution.observationProgram.profile.behaviorFingerprint.render =
      "sha256:ac3cf245ad3e4a311eb6372be9caf49301c7e8ad3ee1b1875a53ea69d1ddc105" ∧
    callerClosureExecution.observationProgram.reference.behaviorFingerprint.render =
      "sha256:1ab36fdcd2978dec901678491646ec67fe0fc1d3bd1883e599bc2c53810b3480" ∧
    callerClosureExecution.observationProgram.mapping.behaviorFingerprint.render =
      "sha256:608e4db6c3a29d0f953640621ee34d34e16b0090309e85804e21f0cb21be30a2" ∧
    callerClosureExecution.program.reference.behaviorFingerprint.render =
      "sha256:f2f1a9a1346576b4d8c6b0b4f7f6c8a138461f90c168ab57747b316807666e56" ∧
    callerClosureExecution.configuration.behaviorFingerprint.render =
      "sha256:7c4c35a8031d07ff55ef5e83b90c64e63cbc6b196642c379ed75b5fc461f3a67" := by
  native_decide

example :
    errorKindOf (checkExecution {
      canonicalExecutionDefinition with
      experiment := {
        canonicalExecutionDefinition.experiment with
        plan := {
          canonicalExecutionDefinition.experiment.plan with
          targetDefinitionId := id "workflow-nexus.target.unsupported"
        }
      }
    }) = some .target ∧
    errorKindOf (checkExecution {
      canonicalExecutionDefinition with
      experiment := {
        canonicalExecutionDefinition.experiment with
        plan := {
          canonicalExecutionDefinition.experiment.plan with
          requestedActions := canonicalExecutionDefinition.experiment.plan.requestedActions ++
            [{ definitionId := id "workflow.action.unsupported", value := "unsupported" }]
        }
      }
    }) = some .action ∧
    errorKindOf (checkExecution {
      canonicalExecutionDefinition with
      experiment := {
        canonicalExecutionDefinition.experiment with
        plan := {
          canonicalExecutionDefinition.experiment.plan with
          linearExtension := canonicalExecutionDefinition.experiment.plan.linearExtension.map fun item =>
            { item with position := 2 }
        }
      }
    }) = some .occurrence ∧
    errorKindOf (checkExecution {
      canonicalExecutionDefinition with
      experiment := {
        canonicalExecutionDefinition.experiment with
        plan := {
          canonicalExecutionDefinition.experiment.plan with
          requestedFaults := [{
            definitionId := id "workflow-nexus.fault.unsupported"
            value := "unsupported"
          }]
        }
      }
    }) = some .fault := by
  native_decide

example :
    errorKindOf (checkExecution {
      canonicalExecutionDefinition with
      participantProgram := {
        canonicalParticipantProgramDefinition with
        participantDefinitionIds := canonicalParticipantProgramDefinition.participantDefinitionIds ++
          [id "temporal.nexus.participant.extra"]
      }
    }) = some .participant ∧
    errorKindOf (checkExecution {
      canonicalExecutionDefinition with
      participantProgram := {
        canonicalParticipantProgramDefinition with
        protocolVersion := 3
      }
    }) = some .protocol ∧
    errorKindOf (checkExecution {
      canonicalExecutionDefinition with
      participantProgram := {
        canonicalParticipantProgramDefinition with
        capabilityDefinitionIds := canonicalParticipantProgramDefinition.capabilityDefinitionIds.drop 1
      }
    }) = some .capability ∧
    errorKindOf (checkExecution {
      canonicalExecutionDefinition with
      participantProgram := {
        canonicalParticipantProgramDefinition with
        occurrences := canonicalParticipantProgramDefinition.occurrences.map fun occurrence =>
          { occurrence with actionDefinitionId := id "workflow.action.unsupported" }
      }
    }) = some .action := by
  native_decide

example :
    errorKindOf (checkExecution {
      canonicalExecutionDefinition with
      observationProgram := {
        canonicalObservationProgramDefinition with
        mapping := {
          canonicalObservationProgramDefinition.mapping with
          definitionId := id "temporal.nexus.synthetic.basic-lifecycle.mapping.drift"
        }
      }
    }) = some .reference ∧
    errorKindOf (checkExecution {
      canonicalExecutionDefinition with
      configuration := {
        canonicalRuntimeConfiguration with
        participantBindings := canonicalRuntimeConfiguration.participantBindings ++
          canonicalRuntimeConfiguration.participantBindings
      } |>.seal
    }) = some .participant ∧
    errorKindOf (checkExecution {
      canonicalExecutionDefinition with
      configuration := mapParticipantBinding (fun binding => {
        binding with programBehaviorFingerprint := behaviorFingerprintOf "stale-program"
      }) canonicalRuntimeConfiguration
    }) = some .program := by
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

example : [
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with formatVersion := "umpire-runtime-configuration/v3"
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with
      configurationDefinitionId := id "temporal.nexus.runtime-configuration.mutated"
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with experiment := {
        canonicalRuntimeConfiguration.experiment with
        artifactChecksum := drivePlanChecksumOf "mutated-experiment"
      }
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with authorityProfile := {
        canonicalRuntimeConfiguration.authorityProfile with
        definitionId := id "temporal.runtime-profile.mutated"
      }
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with authorityProfile := {
        canonicalRuntimeConfiguration.authorityProfile with version := 3
      }
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with authorityProfile := {
        canonicalRuntimeConfiguration.authorityProfile with
        behaviorFingerprint := behaviorFingerprintOf "mutated-profile"
      }
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with authorityProfile := {
        canonicalRuntimeConfiguration.authorityProfile with
        requiredCapabilityDefinitionIds := [id "workflow.capability.mutated"]
      }
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with phaseLimits := canonicalRuntimeConfiguration.phaseLimits.reverse
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with observation := {
        canonicalRuntimeConfiguration.observation with
        profileDefinitionId := id "temporal.nexus.observation-profile.mutated"
      }
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with observation := {
        canonicalRuntimeConfiguration.observation with
        profileBehaviorFingerprint := behaviorFingerprintOf "mutated-observation-profile"
      }
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with observation := {
        canonicalRuntimeConfiguration.observation with
        programDefinitionId := id "temporal.nexus.observation-program.mutated"
      }
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with observation := {
        canonicalRuntimeConfiguration.observation with
        programBehaviorFingerprint := behaviorFingerprintOf "mutated-observation-program"
      }
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with observation := {
        canonicalRuntimeConfiguration.observation with
        mappingDefinitionId := id "temporal.nexus.observation-mapping.mutated"
      }
    },
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with observation := {
        canonicalRuntimeConfiguration.observation with
        mappingBehaviorFingerprint := behaviorFingerprintOf "mutated-observation-mapping"
      }
    },
    configurationIdentityChanges (mapParticipantBinding (fun binding => {
      binding with participantDefinitionId := id "temporal.nexus.participant.mutated"
    }) canonicalRuntimeConfiguration),
    configurationIdentityChanges (mapParticipantBinding (fun binding => {
      binding with protocolDefinitionId := id "umpire.participant-protocol.mutated"
    }) canonicalRuntimeConfiguration),
    configurationIdentityChanges (mapParticipantBinding (fun binding => {
      binding with protocolVersion := 3
    }) canonicalRuntimeConfiguration),
    configurationIdentityChanges (mapParticipantBinding (fun binding => {
      binding with programDefinitionId := id "temporal.nexus.participant-program.mutated"
    }) canonicalRuntimeConfiguration),
    configurationIdentityChanges (mapParticipantBinding (fun binding => {
      binding with programBehaviorFingerprint := behaviorFingerprintOf "mutated-participant-program"
    }) canonicalRuntimeConfiguration),
    configurationIdentityChanges (mapParticipantBinding (fun binding => {
      binding with capabilityDefinitionIds := binding.capabilityDefinitionIds.drop 1
    }) canonicalRuntimeConfiguration),
    configurationIdentityChanges {
      canonicalRuntimeConfiguration with knownGaps := [{
        kind := .input
        code := id "umpire.known-gap.mutated"
      }]
    }
  ].all (fun changed => changed) ∧
    expectedRuntimeConfigurationBehaviorFingerprint canonicalRuntimeConfiguration =
      expectedRuntimeConfigurationBehaviorFingerprint ({ canonicalRuntimeConfiguration with
        provenance := {
          canonicalRuntimeConfiguration.provenance with
          sourceLocations := [{ canonicalConfigurationSource with line := 2 }]
        }
      } : RuntimeConfiguration) ∧
    expectedRuntimeConfigurationBehaviorFingerprint canonicalRuntimeConfiguration =
      expectedRuntimeConfigurationBehaviorFingerprint ({ canonicalRuntimeConfiguration with
        provenance := {
          canonicalRuntimeConfiguration.provenance with
          sourceDefinitionIds := [id "temporal.nexus.provenance.mutated"]
        }
      } : RuntimeConfiguration) := by
  native_decide

example : callerClosureExecution.configuration.authorityProfile.requiredCapabilityDefinitionIds = [] ∧
    callerClosureExecution.configuration.participantBindings.flatMap
      ParticipantBinding.capabilityDefinitionIds =
        Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact.plan.capabilityRequirementDefinitionIds := by
  native_decide

end Temporal.NexusExecutionIntegrationTests
