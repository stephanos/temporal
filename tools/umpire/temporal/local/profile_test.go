package local

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestAuthorityUsesTheExactModelOwnedLocalProfile(t *testing.T) {
	program := testProgram(t)
	authority, err := NewAuthority(
		"switch.runtime.configuration",
		"sha256:6b81f3a1bc1b67f699b5f2dd7bd030e08c4bcf52c656274d4b25abb374bb87df",
		"switch.participant",
		"switch.protocol",
		program,
	)
	require.NoError(t, err)
	require.Equal(t, ProfileDefinitionID, authority.DefinitionID())
	require.Equal(t, ProfileVersion, authority.Version())
	require.Equal(t, ProfileBehaviorFingerprint, authority.BehaviorFingerprint())
	require.Equal(t, RequiredCapabilityDefinitionIDs(), authority.RequiredCapabilityDefinitionIDs())
	require.Equal(t, umpireruntime.CanonicalPhaseLimits(), authority.PhaseLimits())
	require.EqualValues(t, 0, authority.Seed())
	require.EqualValues(t, 1, authority.Attempt())
}

func TestRequiredCapabilitiesReturnsAnImmutableCopy(t *testing.T) {
	capabilities := RequiredCapabilityDefinitionIDs()
	capabilities[0] = "changed.capability"
	require.Equal(t, []string{
		"umpire.runtime.capability.complete-workflow-history-read",
		"umpire.runtime.capability.ephemeral-server-lifecycle",
		"umpire.runtime.capability.sdk-worker-lifecycle",
	}, RequiredCapabilityDefinitionIDs())
}
