package replay

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestReproduceReproducesBoundResult(t *testing.T) {
	experiment := replayExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	baseline := execution.Result{
		FormatVersion: execution.ResultFormatVersion, ExperimentDigest: digest,
		Environment: execution.EnvironmentIdentity{Name: "local-in-process", Capabilities: []protocol.CapabilityID{"update"}},
		Claim:       execution.Claim{Kind: execution.ClaimViolating, Property: experiment.Property.Identifier},
	}
	baseline.DeriveAssurance()
	bundle := Bundle{
		FormatVersion: BundleFormatVersion, Experiment: experiment, Result: baseline,
		Replay: Metadata{Profile: "local-in-process", Capabilities: []protocol.CapabilityID{"update"}, Seed: experiment.Scope.Seed, Bounds: experiment.Scope.Bounds, Command: "umpire3 replay"},
	}
	report, err := Reproduce(context.Background(), bundle,
		func(context.Context, protocol.Experiment) (execution.Result, error) { return baseline, nil })
	require.NoError(t, err)
	require.True(t, report.Reproduced)
	require.Empty(t, report.Drift)
}

func TestReproduceReportsProfileDrift(t *testing.T) {
	experiment := replayExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	baseline := execution.Result{
		FormatVersion: execution.ResultFormatVersion, ExperimentDigest: digest,
		Environment: execution.EnvironmentIdentity{Name: "remote-deployment", Capabilities: []protocol.CapabilityID{"update"}},
	}
	baseline.DeriveAssurance()
	bundle := Bundle{
		Experiment: experiment, Result: baseline,
		Replay: Metadata{Profile: "remote-deployment", Capabilities: []protocol.CapabilityID{"update"}},
	}
	current := baseline
	current.Environment.Name = "local-in-process"
	report, err := Reproduce(context.Background(), bundle,
		func(context.Context, protocol.Experiment) (execution.Result, error) { return current, nil })
	require.NoError(t, err)
	require.False(t, report.Reproduced)
	require.Contains(t, report.Drift, Drift{Kind: DriftRealization, Detail: "environment profile changed"})
}

func replayExperiment(t *testing.T) protocol.Experiment {
	t.Helper()
	encoded, err := os.ReadFile("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}
