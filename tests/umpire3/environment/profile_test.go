package environment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvidenceProfilesAreExplicit(t *testing.T) {
	t.Parallel()

	for _, evidenceProfile := range []string{
		EvidenceProfilePublicGRPC,
		EvidenceProfilePublicGRPCHistory,
		EvidenceProfileTelemetry,
		EvidenceProfileInProcessHooks,
	} {
		profile := Profile{
			Name: "profile", BuildID: "build", ConfigurationIdentity: "config",
			EvidenceProfile: evidenceProfile, DrivingAuthority: "test-driver",
			ObservationAuthority: "selected-source", FaultAuthority: "none",
			IsolationIdentity: "namespace", RetentionClass: "semantic-only",
		}
		require.NoError(t, profile.Validate())
	}
}
