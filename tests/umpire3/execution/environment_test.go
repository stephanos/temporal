package execution

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestRunUsesIdentityReturnedByPrepare(t *testing.T) {
	experiment := loadExperiment(t)
	capabilities := uniqueSortedCapabilities(allCapabilities(experiment))
	identity := EnvironmentIdentity{
		Name: "test", BuildID: "build", ConfigurationIdentity: "configuration",
		EvidenceProfile: EvidenceProfilePublicGRPC, DrivingAuthority: "driver",
		ObservationAuthority: "observer", FaultAuthority: "none",
		IsolationIdentity: "namespace/queue", RetentionClass: "semantic-redacted",
		Capabilities: capabilities,
	}
	factory := &identityFactory{identity: identity, session: conformingSession(experiment)}

	result, err := Run(context.Background(), Request{Experiment: experiment, Environment: factory})
	require.NoError(t, err)
	require.Equal(t, identity, result.Environment)
	require.Equal(t, 1, factory.prepares)
}

type identityFactory struct {
	identity EnvironmentIdentity
	session  Session
	prepares int
}

func (f *identityFactory) Capabilities() []protocol.CapabilityID {
	return append([]protocol.CapabilityID(nil), f.identity.Capabilities...)
}

func (f *identityFactory) Prepare(context.Context, protocol.Experiment) (PreparedEnvironment, error) {
	f.prepares++
	return PreparedEnvironment{Session: f.session, Identity: f.identity}, nil
}
