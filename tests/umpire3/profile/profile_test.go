package profile

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestDeploymentProfilesHaveSeparatedAuthorities(t *testing.T) {
	t.Parallel()

	profiles := []Config{
		Local("build", "namespace", "queue"),
		CI("build", "namespace", "queue"),
		Remote("https://temporal.example", "token", "build", "namespace", "queue"),
		BlackBox("https://temporal.example", "token", "build", "namespace", "queue"),
	}
	for _, config := range profiles {
		definition, err := Define(config)
		require.NoError(t, err)
		require.NoError(t, definition.Environment.Validate())
		require.NotEmpty(t, definition.Environment.DrivingAuthority)
		require.NotEmpty(t, definition.Environment.ObservationAuthority)
		require.NotEmpty(t, definition.Environment.FaultAuthority)
	}
	require.Equal(t, environment.EvidenceProfilePublicGRPC, mustDefine(t, profiles[3]).Environment.EvidenceProfile)
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

func TestPairwiseMatrixIsDeterministicAndCoversEveryPair(t *testing.T) {
	t.Parallel()

	dimensions := Dimensions{
		"evidence": {"grpc", "history", "hooks"},
		"fault":    {"none", "rpc"},
		"profile":  {"local", "ci", "remote"},
	}
	first, err := Pairwise(dimensions)
	require.NoError(t, err)
	second, err := Pairwise(dimensions)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.True(t, CoversEveryPair(dimensions, first))
}

func TestSemanticExperimentDigestIsPortableAcrossProfiles(t *testing.T) {
	t.Parallel()

	experiment := validExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	for _, config := range []Config{
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

func (f *countingFactory) Capabilities() []string {
	return []string{string(protocol.CapabilityIDHistoryObservation)}
}

func (f *countingFactory) Prepare(context.Context, protocol.Experiment) (environment.Session, error) {
	f.prepares++
	return nil, errors.New("not implemented")
}

func mustDefine(t *testing.T, config Config) Definition {
	t.Helper()
	definition, err := Define(config)
	require.NoError(t, err)
	return definition
}

func validExperiment(t *testing.T) protocol.Experiment {
	t.Helper()
	experiment, err := protocol.DecodeExperiment(
		mustOpen(t, "../testdata/nexus-cancellation.json"), protocol.DefaultDecodeLimit,
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
