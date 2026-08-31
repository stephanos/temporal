package nexus

import (
	"go.temporal.io/server/tools/umpire/artifact"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/local"
)

const (
	callerClosureConfigurationDefinitionID          = "temporal.nexus.runtime-configuration.caller-closure"
	callerClosureConfigurationBehaviorFingerprint   = "sha256:7c4c35a8031d07ff55ef5e83b90c64e63cbc6b196642c379ed75b5fc461f3a67"
	callerClosureParticipantDefinitionID            = "temporal.nexus.participant.caller-closure"
	callerClosureProtocolDefinitionID               = "umpire.participant-protocol.v2"
	callerClosureProgramDefinitionID                = "temporal.nexus.participant-program.caller-closure"
	callerClosureProgramVersion                     = 1
	callerClosureProgramBehaviorFingerprint         = "sha256:f2f1a9a1346576b4d8c6b0b4f7f6c8a138461f90c168ab57747b316807666e56"
	callerClosureTargetDefinitionID                 = "workflow-nexus.target.caller-closure"
	duplicateDeliveryConfigurationDefinitionID      = "temporal.nexus.runtime-configuration.caller-closure-duplicate-delivery"
	duplicateDeliveryConfigurationFingerprint       = "sha256:d88670a6766c2ef9037c82183f00c1c42179a7578c3c4c07714eadb5540750c0"
	duplicateDeliveryFaultDefinitionID              = "temporal.nexus.caller-closure.fault.duplicate-delivery-observation"
	duplicateDeliveryMappingDefinitionID            = "temporal.system.nexus.caller-closure.duplicate-delivery.mapping"
	duplicateDeliveryMappingFingerprint             = "sha256:cc5910e77e3d43f4cad56de88a68f099eea8b25bbbe0fde451a02b2afda01438"
	duplicateDeliveryObservationProfileDefinitionID = "temporal.system.nexus.caller-closure.duplicate-delivery.profile"
	duplicateDeliveryObservationProfileFingerprint  = "sha256:02517311485c8f87f13581d9381447ae34cb159526bdc865c1054efe2067acb8"
	duplicateDeliveryObservationProgramDefinitionID = "temporal.system.nexus.caller-closure.duplicate-delivery.observation-program"
	duplicateDeliveryObservationProgramFingerprint  = "sha256:7226f7762d3a21e7a66d460a4bf6b9d9a1d244bca847e4919cc0bc7debf432bd"
	duplicateDeliveryProgramDefinitionID            = "temporal.nexus.participant-program.caller-closure-duplicate-delivery"
	duplicateDeliveryProgramBehaviorFingerprint     = "sha256:3cd71d91c2ba9eef0e9b2a04cccf49d09b214282625ef815c2da8474ee49afee"
	forceCloseActionDefinitionID                    = "workflow.action.force-close"
	forceCloseOccurrenceDefinitionID                = "workflow-nexus.occurrence.force-close"
)

var callerClosureCapabilities = []string{
	"nexus.capability.cancellation",
	"workflow-nexus.capability.ownership",
	"workflow.capability.lifecycle",
}

// CheckRequest binds the admitted set to the sole System-owned caller-closure
// program and the closed local authority before any environment can exist.
func CheckRequest(
	admitted artifact.AdmittedSet,
	runIdentity string,
) (umpireruntime.CheckedRunRequest, error) {
	configurationDefinitionID := callerClosureConfigurationDefinitionID
	configurationBehaviorFingerprint := callerClosureConfigurationBehaviorFingerprint
	faulted := false
	if executable, ok := admitted.Executable(); ok &&
		executable.RuntimeConfiguration().ConfigurationDefinitionID == duplicateDeliveryConfigurationDefinitionID {
		configurationDefinitionID = duplicateDeliveryConfigurationDefinitionID
		configurationBehaviorFingerprint = duplicateDeliveryConfigurationFingerprint
		faulted = true
	}
	occurrence, err := umpireruntime.NewOccurrence(
		forceCloseOccurrenceDefinitionID,
		forceCloseActionDefinitionID,
		1,
	)
	if err != nil {
		return umpireruntime.CheckedRunRequest{}, err
	}
	program, err := normalCallerClosureProgram(occurrence)
	if faulted {
		program, err = duplicateDeliveryCallerClosureProgram(occurrence)
	}
	if err != nil {
		return umpireruntime.CheckedRunRequest{}, err
	}
	authority, err := local.NewAuthority(
		configurationDefinitionID,
		configurationBehaviorFingerprint,
		callerClosureParticipantDefinitionID,
		callerClosureProtocolDefinitionID,
		program,
	)
	if err != nil {
		return umpireruntime.CheckedRunRequest{}, err
	}
	return umpireruntime.CheckRequest(admitted, authority, runIdentity, 0, 1)
}

func normalCallerClosureProgram(occurrence umpireruntime.Occurrence) (umpireruntime.Program, error) {
	return umpireruntime.NewProgram(
		callerClosureProgramDefinitionID,
		callerClosureProgramVersion,
		callerClosureProgramBehaviorFingerprint,
		[]string{callerClosureTargetDefinitionID},
		[]string{forceCloseActionDefinitionID},
		[]umpireruntime.Occurrence{occurrence},
		callerClosureCapabilities,
	)
}

func duplicateDeliveryCallerClosureProgram(
	occurrence umpireruntime.Occurrence,
) (umpireruntime.Program, error) {
	observation, err := umpireruntime.NewObservationProgram(
		duplicateDeliveryObservationProfileDefinitionID,
		duplicateDeliveryObservationProfileFingerprint,
		duplicateDeliveryObservationProgramDefinitionID,
		duplicateDeliveryObservationProgramFingerprint,
		duplicateDeliveryMappingDefinitionID,
		duplicateDeliveryMappingFingerprint,
	)
	if err != nil {
		return umpireruntime.Program{}, err
	}
	return umpireruntime.NewProgramWithRequestedFault(
		duplicateDeliveryProgramDefinitionID,
		callerClosureProgramVersion,
		duplicateDeliveryProgramBehaviorFingerprint,
		[]string{callerClosureTargetDefinitionID},
		[]string{forceCloseActionDefinitionID},
		[]umpireruntime.Occurrence{occurrence},
		observation,
		duplicateDeliveryFaultDefinitionID,
		callerClosureCapabilities,
	)
}
