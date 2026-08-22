package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	environment "go.temporal.io/server/tests/umpire3/execution"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func TestDeploymentProfilesHaveSeparatedAuthorities(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   Spec
		expected environment.EnvironmentIdentity
	}{
		{
			name: "local", config: Local("build", "namespace", "queue"),
			expected: environment.EnvironmentIdentity{
				Name: "local-in-process", BuildID: "build",
				EvidenceProfile:  environment.EvidenceProfileInProcessHooks,
				DrivingAuthority: "local-test-authority", ObservationAuthority: "local-server-hooks",
				FaultAuthority: "isolated-local-faults", IsolationIdentity: "namespace/queue",
				RetentionClass: "semantic-redacted",
			},
		},
		{
			name: "ci", config: CI("build", "namespace", "queue"),
			expected: environment.EnvironmentIdentity{
				Name: "ci-test-cluster", BuildID: "build",
				EvidenceProfile:  environment.EvidenceProfilePublicGRPCHistory,
				DrivingAuthority: "ci-test-cluster", ObservationAuthority: "ci-public-history",
				FaultAuthority: "isolated-ci-faults", IsolationIdentity: "namespace/queue",
				RetentionClass: "semantic-redacted",
			},
		},
		{
			name: "remote", config: Remote("https://temporal.example", "token", "build", "namespace", "queue"),
			expected: environment.EnvironmentIdentity{
				Name: "remote-deployment", BuildID: "build",
				EvidenceProfile:  environment.EvidenceProfilePublicGRPCHistory,
				DrivingAuthority: "remote-api", ObservationAuthority: "remote-public-history",
				FaultAuthority: "remote-approved-faults", IsolationIdentity: "namespace/queue",
				RetentionClass: "semantic-redacted",
			},
		},
		{
			name: "black box", config: BlackBox("https://temporal.example", "token", "build", "namespace", "queue"),
			expected: environment.EnvironmentIdentity{
				Name: "grpc-only-black-box", BuildID: "build", EvidenceProfile: environment.EvidenceProfilePublicGRPC,
				DrivingAuthority: "public-grpc", ObservationAuthority: "public-grpc",
				FaultAuthority: "none", IsolationIdentity: "namespace/queue", RetentionClass: "semantic-redacted",
			},
		},
		{
			name:   "canary",
			config: Canary("https://temporal.example", "token", "build", "namespace", "queue", []string{"worker"}),
			expected: environment.EnvironmentIdentity{
				Name: "production-canary", BuildID: "build",
				EvidenceProfile:      environment.EvidenceProfilePublicGRPCHistory,
				DrivingAuthority:     "approved-production-worker",
				ObservationAuthority: "production-public-history",
				FaultAuthority:       "approved-production-fault-controller", IsolationIdentity: "namespace/queue",
				RetentionClass: "semantic-redacted", HardExecutionBudget: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition, err := Define(test.config)
			require.NoError(t, err)
			test.expected.ConfigurationIdentity = definition.Environment.ConfigurationIdentity
			test.expected.Capabilities = definition.Capabilities
			require.Equal(t, test.expected, definition.Environment)
			require.NoError(t, definition.Environment.Validate())
		})
	}
}

func TestRemoteProfileDiagnosticsNeverContainCredentials(t *testing.T) {
	t.Parallel()

	const secret = "top-secret-token"
	definition, err := Define(Remote("https://temporal.example", secret, "build", "namespace", "queue"))
	require.NoError(t, err)
	encoded, err := json.Marshal(definition)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), secret)
	require.NotContains(t, definition.String(), secret)
}

func TestHardBudgetProfileRequiresKillableBoundary(t *testing.T) {
	t.Parallel()

	config := Remote("https://temporal.example", "token", "build", "namespace", "queue")
	config.HardExecutionBudget = true
	_, err := Define(config)
	require.ErrorContains(t, err, "worker command")
	config.WorkerCommand = []string{"umpire3-worker"}
	_, err = Define(config)
	require.NoError(t, err)
}

func TestHardBudgetProfileCannotBindInProcessFactory(t *testing.T) {
	t.Parallel()

	definition, err := Define(Canary("https://temporal.example", "token", "build", "namespace", "queue",
		[]string{"umpire3-worker"}))
	require.NoError(t, err)
	_, err = Bind(definition, &countingFactory{})
	require.ErrorContains(t, err, "killable worker")
}

func TestFactoryRejectsUnsupportedBeforeAllocation(t *testing.T) {
	t.Parallel()

	underlying := &countingFactory{}
	definition := mustDefine(t, Local("build", "namespace", "queue"))
	definition.Capabilities = nil
	factory, err := Bind(definition, underlying)
	require.NoError(t, err)
	_, err = factory.Prepare(context.Background(), validExperiment(t))
	require.ErrorContains(t, err, "unsupported capabilities")
	require.Zero(t, underlying.prepares)
}

func TestBoundProfileCannotBroadenAdapterFaultAuthority(t *testing.T) {
	t.Parallel()

	definition := mustDefine(t, Local("build", "namespace", "queue"))
	underlying := &identityFactory{capabilities: catalogCapabilities()}
	factory, err := Bind(definition, underlying)
	require.NoError(t, err)
	prepared, err := factory.Prepare(context.Background(), validExperiment(t))
	require.NoError(t, err)
	require.Equal(t, "none", prepared.Identity.FaultAuthority)
	require.Equal(t, factory.Capabilities(), prepared.Identity.Capabilities)
}

func TestSemanticExperimentDigestIsPortableAcrossProfiles(t *testing.T) {
	t.Parallel()

	experiment := validExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	for _, config := range []Spec{
		Local("build", "namespace", "queue"),
		CI("build", "namespace", "queue"),
		Remote("https://temporal.example", "token", "build", "namespace", "queue"),
		BlackBox("https://temporal.example", "token", "build", "namespace", "queue"),
		Canary("https://temporal.example", "token", "build", "namespace", "queue", []string{"worker"}),
	} {
		definition, defineErr := Define(config)
		require.NoError(t, defineErr)
		profileDigest, digestErr := definition.Digest()
		require.NoError(t, digestErr)
		require.NotEqual(t, digest, profileDigest)
		actual, experimentErr := experiment.Digest()
		require.NoError(t, experimentErr)
		require.Equal(t, digest, actual)
	}
}

type countingFactory struct {
	prepares int
}

type identityFactory struct {
	capabilities []protocolcatalog.CapabilityID
}

func (f *identityFactory) Capabilities() []protocolcatalog.CapabilityID {
	return append([]protocolcatalog.CapabilityID(nil), f.capabilities...)
}

func (*identityFactory) Prepare(context.Context, protocolexperiment.Experiment) (environment.PreparedEnvironment, error) {
	return environment.PreparedEnvironment{
		Session:  profileSession{},
		Identity: environment.EnvironmentIdentity{FaultAuthority: "none"},
	}, nil
}

type profileSession struct{}

func (profileSession) Realize(
	context.Context,
	protocolexperiment.Action,
	environment.Bindings,
) (environment.ActionEvidence, error) {
	return environment.ActionEvidence{Outcome: protocolexperiment.ActionOutcomeApplied}, nil
}

func (profileSession) Cleanup(context.Context) environment.CleanupResult {
	return environment.CleanupResult{Complete: true}
}

func (profileSession) RecoveryMetadata() map[string]string { return nil }

func (f *countingFactory) Capabilities() []protocolcatalog.CapabilityID {
	return []protocolcatalog.CapabilityID{protocolcatalog.CapabilityIDHistoryObservation}
}

func (f *countingFactory) Prepare(context.Context, protocolexperiment.Experiment) (environment.PreparedEnvironment, error) {
	f.prepares++
	return environment.PreparedEnvironment{}, errors.New("not implemented")
}

func mustDefine(t *testing.T, config Spec) Profile {
	t.Helper()
	definition, err := Define(config)
	require.NoError(t, err)
	return definition
}

func validExperiment(t *testing.T) protocolexperiment.Experiment {
	t.Helper()
	experiment, err := protocolexperiment.DecodeExperiment(
		mustOpen(t, "../testdata/generated/nexus-cancellation.json"), protocolexperiment.DefaultDecodeLimit,
	)
	require.NoError(t, err)
	return experiment
}

func mustOpen(t *testing.T, name string) *os.File {
	t.Helper()
	file, err := os.Open(name)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	return file
}
