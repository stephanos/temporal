import Temporal.System.Execution.LocalProfile

namespace Temporal.System.Execution.LocalProfileTests

open _root_.Umpire

private def id (value : String) : DefinitionId := DefinitionId.of value

private def checksum : ArtifactChecksum :=
  drivePlanChecksumOf "temporal.system.execution.local-profile-tests/checksum"

private def fingerprint : BehaviorFingerprint :=
  behaviorFingerprintOf "temporal.system.execution.local-profile-tests/fingerprint"

private def configurationDraft : RuntimeConfiguration := {
  formatVersion := "umpire-runtime-configuration/v2"
  configurationDefinitionId := id "temporal.runtime-configuration.test"
  behaviorFingerprint := fingerprint
  experiment := {
    formatVersion := "umpire-experiment/v2"
    artifactChecksum := checksum
    behaviorFingerprint := fingerprint
    provenanceChecksum := checksum
  }
  authorityProfile := ephemeralLocalProfile.authorityProfile
  phaseLimits := ephemeralLocalProfile.phaseLimits
  observation := {
    profileDefinitionId := id "umpire.observation-profile.test"
    profileBehaviorFingerprint := fingerprint
    programDefinitionId := id "umpire.observation-program.test"
    programBehaviorFingerprint := fingerprint
    mappingDefinitionId := id "umpire.observation-mapping.test"
    mappingBehaviorFingerprint := fingerprint
  }
  participantBindings := [{
    participantDefinitionId := id "umpire.participant.test"
    protocolDefinitionId := id "umpire.participant-protocol.test"
    protocolVersion := 2
    programDefinitionId := id "umpire.participant-program.test"
    programBehaviorFingerprint := fingerprint
    capabilityDefinitionIds := [id "umpire.runtime.capability.program-test"]
  }]
  knownGaps := []
  provenance := {
    sourceDefinitionIds := [id "temporal.runtime-configuration.test"]
    sourceLocations := [{
      path := "Temporal/System/Execution/LocalProfileTests.lean"
      line := 1
      column := 1
      provenance := "lean-model"
    }]
  }
  provenanceChecksum := checksum
  artifactChecksum := checksum
}

private def configuration : RuntimeConfiguration := configurationDraft.seal

private def mutatePreparation
    (change : PhaseLimit → PhaseLimit)
    (limits : List PhaseLimit) : List PhaseLimit :=
  limits.map fun limit => if limit.phase == .preparation then change limit else limit

private def profileRejected (definition : LocalProfileDefinition) : Bool :=
  (checkLocalProfile definition).toOption.isNone

example : ephemeralLocalProfile.reference.definitionId =
    id "temporal.runtime-profile.ephemeral-local" ∧
    ephemeralLocalProfile.reference.version = 2 ∧
    ephemeralLocalProfile.reference.behaviorFingerprint.render =
      "sha256:dd92f1ee14df101f2ea4abb4439f4722de8c061292a4fdd6b6476c7ca7e09b31" ∧
    ephemeralLocalProfile.requiredCapabilities = [
      id "umpire.runtime.capability.complete-workflow-history-read",
      id "umpire.runtime.capability.ephemeral-server-lifecycle",
      id "umpire.runtime.capability.sdk-worker-lifecycle"
    ] := by
  native_decide

example : ephemeralLocalProfile.generatedViewBytes =
    "{\n" ++
    "  \"definitionId\": \"temporal.runtime-profile.ephemeral-local\",\n" ++
    "  \"version\": 2,\n" ++
    "  \"requiredCapabilityDefinitionIds\": [\n" ++
    "    \"umpire.runtime.capability.complete-workflow-history-read\",\n" ++
    "    \"umpire.runtime.capability.ephemeral-server-lifecycle\",\n" ++
    "    \"umpire.runtime.capability.sdk-worker-lifecycle\"\n" ++
    "  ],\n" ++
    "  \"phaseLimits\": [\n" ++
    "    {\n" ++
    "      \"phase\": \"preparation\",\n" ++
    "      \"durationMilliseconds\": 30000,\n" ++
    "      \"maxAttempts\": 1,\n" ++
    "      \"maxRecords\": 128,\n" ++
    "      \"maxBytes\": 1048576\n" ++
    "    },\n" ++
    "    {\n" ++
    "      \"phase\": \"realization\",\n" ++
    "      \"durationMilliseconds\": 30000,\n" ++
    "      \"maxAttempts\": 1,\n" ++
    "      \"maxRecords\": 128,\n" ++
    "      \"maxBytes\": 1048576\n" ++
    "    },\n" ++
    "    {\n" ++
    "      \"phase\": \"observation\",\n" ++
    "      \"durationMilliseconds\": 30000,\n" ++
    "      \"maxAttempts\": 1,\n" ++
    "      \"maxRecords\": 3584,\n" ++
    "      \"maxBytes\": 12582912\n" ++
    "    },\n" ++
    "    {\n" ++
    "      \"phase\": \"isolation\",\n" ++
    "      \"durationMilliseconds\": 15000,\n" ++
    "      \"maxAttempts\": 1,\n" ++
    "      \"maxRecords\": 128,\n" ++
    "      \"maxBytes\": 1048576\n" ++
    "    },\n" ++
    "    {\n" ++
    "      \"phase\": \"cleanup\",\n" ++
    "      \"durationMilliseconds\": 15000,\n" ++
    "      \"maxAttempts\": 1,\n" ++
    "      \"maxRecords\": 128,\n" ++
    "      \"maxBytes\": 1048576\n" ++
    "    }\n" ++
    "  ],\n" ++
    "  \"seed\": 0,\n" ++
    "  \"attempt\": 1,\n" ++
    "  \"participantProgramRequirements\": {\n" ++
    "    \"participantCount\": 1,\n" ++
    "    \"programCount\": 1,\n" ++
    "    \"protocolVersion\": 2,\n" ++
    "    \"commands\": [\n" ++
    "      \"prepare\",\n" ++
    "      \"realize\",\n" ++
    "      \"observe\",\n" ++
    "      \"cleanup\"\n" ++
    "    ]\n" ++
    "  }\n" ++
    "}\n" := by
  native_decide

example : ephemeralLocalProfile.phaseLimits.foldl
      (fun total limit => total + limit.durationMilliseconds) 0 = 120000 ∧
    ephemeralLocalProfile.phaseLimits.foldl
      (fun total limit => total + limit.maxAttempts) 0 = 5 ∧
    ephemeralLocalProfile.phaseLimits.foldl
      (fun total limit => total + limit.maxRecords) 0 = 4096 ∧
    ephemeralLocalProfile.phaseLimits.foldl
      (fun total limit => total + limit.maxBytes) 0 = 16777216 := by
  native_decide

example : configuration.isValidTransport ∧
    (ephemeralLocalProfile.checkRuntimeConfiguration configuration).toOption.isSome := by
  native_decide

example :
    profileRejected {
      canonicalLocalProfileDefinition with
      reference := {
        canonicalLocalProfileDefinition.reference with
        behaviorFingerprint := fingerprint
      }
    } ∧
    profileRejected {
      canonicalLocalProfileDefinition with
      requiredCapabilities := canonicalLocalProfileDefinition.requiredCapabilities.reverse
    } ∧
    profileRejected {
      canonicalLocalProfileDefinition with
      phaseLimits := canonicalLocalProfileDefinition.phaseLimits.drop 1
    } ∧
    profileRejected { canonicalLocalProfileDefinition with seed := 1 } ∧
    profileRejected { canonicalLocalProfileDefinition with attempt := 2 } ∧
    profileRejected {
      canonicalLocalProfileDefinition with
      authorityFieldDefinitionIds := [id "temporal.runtime.endpoint.remote"]
    } := by
  native_decide

example :
    profileRejected {
      canonicalLocalProfileDefinition with
      reference := {
        canonicalLocalProfileDefinition.reference with
        definitionId := id "temporal.runtime-profile.developer-server"
      }
    } ∧
    profileRejected {
      canonicalLocalProfileDefinition with
      reference := { canonicalLocalProfileDefinition.reference with version := 3 }
    } ∧
    profileRejected {
      canonicalLocalProfileDefinition with
      requiredCapabilities := canonicalLocalProfileDefinition.requiredCapabilities.drop 1
    } ∧
    profileRejected {
      canonicalLocalProfileDefinition with
      requiredCapabilities := canonicalLocalProfileDefinition.requiredCapabilities ++
        [id "umpire.runtime.capability.remote-server"]
    } := by
  native_decide

example :
    profileRejected {
      canonicalLocalProfileDefinition with
      phaseLimits := mutatePreparation (fun limit =>
        { limit with durationMilliseconds := 29999 }) canonicalLocalProfileDefinition.phaseLimits
    } ∧
    profileRejected {
      canonicalLocalProfileDefinition with
      phaseLimits := mutatePreparation (fun limit =>
        { limit with maxAttempts := 2 }) canonicalLocalProfileDefinition.phaseLimits
    } ∧
    profileRejected {
      canonicalLocalProfileDefinition with
      phaseLimits := mutatePreparation (fun limit =>
        { limit with maxRecords := 127 }) canonicalLocalProfileDefinition.phaseLimits
    } ∧
    profileRejected {
      canonicalLocalProfileDefinition with
      phaseLimits := mutatePreparation (fun limit =>
        { limit with maxBytes := 1048575 }) canonicalLocalProfileDefinition.phaseLimits
    } := by
  native_decide

example :
    profileRejected {
      canonicalLocalProfileDefinition with
      participantProgramRequirements := {
        canonicalLocalProfileDefinition.participantProgramRequirements with
        participantCount := 2
      }
    } ∧
    profileRejected {
      canonicalLocalProfileDefinition with
      participantProgramRequirements := {
        canonicalLocalProfileDefinition.participantProgramRequirements with
        programCount := 2
      }
    } ∧
    profileRejected {
      canonicalLocalProfileDefinition with
      participantProgramRequirements := {
        canonicalLocalProfileDefinition.participantProgramRequirements with
        protocolVersion := 1
      }
    } ∧
    profileRejected {
      canonicalLocalProfileDefinition with
      participantProgramRequirements := {
        canonicalLocalProfileDefinition.participantProgramRequirements with
        commands := [.prepare, .realize, .cleanup]
      }
    } := by
  native_decide

example :
    (ephemeralLocalProfile.checkRuntimeConfiguration <|
      ({ configuration with phaseLimits := configuration.phaseLimits.reverse }).seal).toOption.isNone ∧
    (ephemeralLocalProfile.checkRuntimeConfiguration <|
      ({ configuration with authorityProfile := {
        configuration.authorityProfile with
        requiredCapabilityDefinitionIds :=
          configuration.authorityProfile.requiredCapabilityDefinitionIds.drop 1
      }}).seal).toOption.isNone := by
  native_decide

end Temporal.System.Execution.LocalProfileTests
