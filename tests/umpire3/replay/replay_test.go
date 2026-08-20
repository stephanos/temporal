package replay

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/artifact"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
)

func TestRunReproducesBoundResult(t *testing.T) {
	experiment := replayExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	baseline := umpire3runtime.Result{
		FormatVersion: protocol.FormatVersion, ExperimentDigest: digest,
		Environment: umpire3runtime.EnvironmentProfile{Name: "local-in-process", Capabilities: []string{"update"}},
		Claim:       umpire3runtime.Claim{Kind: umpire3runtime.ClaimViolating, Property: experiment.Property.Identifier},
	}
	bundle := artifact.Record{
		FormatVersion: artifact.FormatVersion, Experiment: experiment, Result: baseline,
		Replay: artifact.ReplayMetadata{Profile: "local-in-process", Capabilities: []string{"update"}, Seed: experiment.Scope.Seed, Bounds: experiment.Scope.Bounds, Command: "umpire3 replay"},
	}
	report, err := Run(context.Background(), bundle,
		func(context.Context, protocol.Experiment) (umpire3runtime.Result, error) { return baseline, nil })
	require.NoError(t, err)
	require.True(t, report.Reproduced)
	require.Empty(t, report.Drift)
}

func TestRunReportsProfileDrift(t *testing.T) {
	experiment := replayExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	baseline := umpire3runtime.Result{
		ExperimentDigest: digest,
		Environment:      umpire3runtime.EnvironmentProfile{Name: "remote-deployment", Capabilities: []string{"update"}},
	}
	bundle := artifact.Record{
		Experiment: experiment, Result: baseline,
		Replay: artifact.ReplayMetadata{Profile: "remote-deployment", Capabilities: []string{"update"}},
	}
	current := baseline
	current.Environment.Name = "local-in-process"
	report, err := Run(context.Background(), bundle,
		func(context.Context, protocol.Experiment) (umpire3runtime.Result, error) { return current, nil })
	require.NoError(t, err)
	require.False(t, report.Reproduced)
	require.Contains(t, report.Drift, umpire3runtime.Drift{Kind: umpire3runtime.DriftRealization, Detail: "environment profile changed"})
}

func replayExperiment(t *testing.T) protocol.Experiment {
	t.Helper()
	encoded, err := os.ReadFile("../testdata/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}
