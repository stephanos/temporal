package nexus

import (
	"go.temporal.io/server/tools/umpire/artifact"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/local"
)

const (
	callerClosureConfigurationDefinitionID        = "temporal.nexus.runtime-configuration.caller-closure"
	callerClosureConfigurationBehaviorFingerprint = "sha256:7c4c35a8031d07ff55ef5e83b90c64e63cbc6b196642c379ed75b5fc461f3a67"
	callerClosureParticipantDefinitionID          = "temporal.nexus.participant.caller-closure"
	callerClosureProtocolDefinitionID             = "umpire.participant-protocol.v2"
	callerClosureProgramDefinitionID              = "temporal.nexus.participant-program.caller-closure"
	callerClosureProgramVersion                   = 1
	callerClosureProgramBehaviorFingerprint       = "sha256:f2f1a9a1346576b4d8c6b0b4f7f6c8a138461f90c168ab57747b316807666e56"
	callerClosureTargetDefinitionID               = "workflow-nexus.target.caller-closure"
	forceCloseActionDefinitionID                  = "workflow.action.force-close"
	forceCloseOccurrenceDefinitionID              = "workflow-nexus.occurrence.force-close"
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
	occurrence, err := umpireruntime.NewOccurrence(
		forceCloseOccurrenceDefinitionID,
		forceCloseActionDefinitionID,
		1,
	)
	if err != nil {
		return umpireruntime.CheckedRunRequest{}, err
	}
	program, err := umpireruntime.NewProgram(
		callerClosureProgramDefinitionID,
		callerClosureProgramVersion,
		callerClosureProgramBehaviorFingerprint,
		[]string{callerClosureTargetDefinitionID},
		[]string{forceCloseActionDefinitionID},
		[]umpireruntime.Occurrence{occurrence},
		callerClosureCapabilities,
	)
	if err != nil {
		return umpireruntime.CheckedRunRequest{}, err
	}
	authority, err := local.NewAuthority(
		callerClosureConfigurationDefinitionID,
		callerClosureConfigurationBehaviorFingerprint,
		callerClosureParticipantDefinitionID,
		callerClosureProtocolDefinitionID,
		program,
	)
	if err != nil {
		return umpireruntime.CheckedRunRequest{}, err
	}
	return umpireruntime.CheckRequest(admitted, authority, runIdentity, 0, 1)
}
