package execution

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tools/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tools/umpire3/protocol/experiment"
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

func (f *identityFactory) Capabilities() []protocolcatalog.CapabilityID {
	return append([]protocolcatalog.CapabilityID(nil), f.identity.Capabilities...)
}

func (f *identityFactory) Prepare(context.Context, protocolexperiment.Experiment) (PreparedEnvironment, error) {
	f.prepares++
	return PreparedEnvironment{Session: f.session, Identity: f.identity}, nil
}
