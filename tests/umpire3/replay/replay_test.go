package replay

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/execution"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
)

func TestReproduceReproducesBoundResult(t *testing.T) {
	experiment := replayExperiment(t)
	digest, err := experiment.Digest()
	require.NoError(t, err)
	baseline := execution.Result{
		FormatVersion: execution.ResultFormatVersion, ExperimentDigest: digest,
		Environment: execution.EnvironmentIdentity{Name: "local-in-process", Capabilities: []protocolcatalog.CapabilityID{"update"}},
		Claim:       execution.Claim{Kind: execution.ClaimViolating, Property: experiment.Property.Identifier},
	}
	baseline.DeriveAssurance()
	bundle := Bundle{
		FormatVersion: BundleFormatVersion, Experiment: experiment, Result: baseline,
		Replay: Metadata{Profile: "local-in-process", Capabilities: []protocolcatalog.CapabilityID{"update"}, Seed: experiment.Scope.Seed, Bounds: experiment.Scope.Bounds, Command: "umpire3 replay"},
	}
	report, err := Reproduce(context.Background(), bundle,
		func(context.Context, protocolexperiment.Experiment) (execution.Result, error) { return baseline, nil })
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
		Environment: execution.EnvironmentIdentity{Name: "remote-deployment", Capabilities: []protocolcatalog.CapabilityID{"update"}},
	}
	baseline.DeriveAssurance()
	bundle := Bundle{
		Experiment: experiment, Result: baseline,
		Replay: Metadata{Profile: "remote-deployment", Capabilities: []protocolcatalog.CapabilityID{"update"}},
	}
	current := baseline
	current.Environment.Name = "local-in-process"
	report, err := Reproduce(context.Background(), bundle,
		func(context.Context, protocolexperiment.Experiment) (execution.Result, error) { return current, nil })
	require.NoError(t, err)
	require.False(t, report.Reproduced)
	require.Contains(t, report.Drift, Drift{Kind: DriftRealization, Detail: "environment profile changed"})
}

func replayExperiment(t *testing.T) protocolexperiment.Experiment {
	t.Helper()
	encoded, err := os.ReadFile("../testdata/generated/update-lifecycle.json")
	require.NoError(t, err)
	experiment, err := protocolexperiment.DecodeExperiment(bytes.NewReader(encoded), protocolexperiment.DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}
