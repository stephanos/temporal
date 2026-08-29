// Package local owns the one closed, loopback-only Temporal execution authority.
package local

import (
	"slices"

	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

const (
	// ProfileDefinitionID is the exact model-owned local profile identity.
	ProfileDefinitionID = "temporal.runtime-profile.ephemeral-local"
	// ProfileVersion is the exact model-owned local profile version.
	ProfileVersion uint64 = 2
	// ProfileBehaviorFingerprint is derived from the model-owned pretty Generated View.
	ProfileBehaviorFingerprint = "sha256:dd92f1ee14df101f2ea4abb4439f4722de8c061292a4fdd6b6476c7ca7e09b31"
)

var requiredCapabilityDefinitionIDs = [...]string{
	"umpire.runtime.capability.complete-workflow-history-read",
	"umpire.runtime.capability.ephemeral-server-lifecycle",
	"umpire.runtime.capability.sdk-worker-lifecycle",
}

// RequiredCapabilityDefinitionIDs returns the exact generic local authority capabilities.
func RequiredCapabilityDefinitionIDs() []string {
	return slices.Clone(requiredCapabilityDefinitionIDs[:])
}

// NewAuthority closes runtime authority construction over the sole local profile.
// Configuration and participant identities remain inert model-owned bindings; no
// endpoint, namespace, credential, executable, callback, or runtime option enters here.
func NewAuthority(
	configurationDefinitionID string,
	configurationBehaviorFingerprint string,
	participantDefinitionID string,
	protocolDefinitionID string,
	program umpireruntime.Program,
) (umpireruntime.Authority, error) {
	return umpireruntime.NewAuthority(
		ProfileDefinitionID,
		ProfileVersion,
		ProfileBehaviorFingerprint,
		configurationDefinitionID,
		configurationBehaviorFingerprint,
		RequiredCapabilityDefinitionIDs(),
		[]string{},
		umpireruntime.CanonicalPhaseLimits(),
		0,
		1,
		participantDefinitionID,
		protocolDefinitionID,
		2,
		1,
		1,
		program,
	)
}

func exactAuthority(authority umpireruntime.Authority) bool {
	return authority.DefinitionID() == ProfileDefinitionID &&
		authority.Version() == ProfileVersion &&
		authority.BehaviorFingerprint() == ProfileBehaviorFingerprint &&
		slices.Equal(authority.RequiredCapabilityDefinitionIDs(), requiredCapabilityDefinitionIDs[:]) &&
		slices.Equal(authority.PhaseLimits(), umpireruntime.CanonicalPhaseLimits()) &&
		authority.Seed() == 0 && authority.Attempt() == 1 &&
		authority.ParticipantCount() == 1 && authority.ProgramCount() == 1 &&
		authority.ProtocolVersion() == 2
}
