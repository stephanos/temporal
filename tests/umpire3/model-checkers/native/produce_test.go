package native

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/protocol"
)

func TestParallelProducerIsDeterministicAndExploresTenTimesThePilot(t *testing.T) {
	t.Parallel()

	view := soundView(t)
	serial, err := Produce(context.Background(), view, testOptions(1), nil)
	require.NoError(t, err)
	parallel, err := Produce(context.Background(), view, testOptions(8), nil)
	require.NoError(t, err)
	require.Equal(t, serial, parallel)
	require.Equal(t, len(view.Oracle.States)*10, parallel.Statistics.ExpandedStates)
	require.Equal(t, len(view.Oracle.States), parallel.Statistics.RepresentativeStates)
	require.Equal(t, parallel.Statistics.RepresentativeStates,
		parallel.Closure.ClosedRepresentatives)
	require.Equal(t, parallel.Statistics.ExpandedStates,
		parallel.Symmetry.ExpandedStates)
}

func TestCheckpointResumeMatchesUninterruptedProduction(t *testing.T) {
	t.Parallel()

	view := soundView(t)
	expected, err := Produce(context.Background(), view, testOptions(4), nil)
	require.NoError(t, err)
	checkpointPath := filepath.Join(t.TempDir(), "search", "checkpoint.json")
	stop := errors.New("simulated producer crash")
	options := testOptions(3)
	options.Checkpoint = func(checkpoint Checkpoint) error {
		require.NoError(t, SaveCheckpoint(checkpointPath, checkpoint))
		if checkpoint.CompletedDepth >= 1 {
			return stop
		}
		return nil
	}
	_, err = Produce(context.Background(), view, options, nil)
	require.ErrorIs(t, err, stop)
	checkpoint, err := LoadCheckpoint(checkpointPath, protocol.DefaultDecodeLimit)
	require.NoError(t, err)

	resumedOptions := testOptions(7)
	resumed, err := Produce(context.Background(), view, resumedOptions, &checkpoint)
	require.NoError(t, err)
	require.Equal(t, expected, resumed)
	info, err := os.Stat(checkpointPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestCorruptedAndPartialCertificatesFailClosed(t *testing.T) {
	t.Parallel()

	view := soundView(t)
	certificate, err := Produce(context.Background(), view, testOptions(2), nil)
	require.NoError(t, err)
	encoded, err := certificate.CanonicalJSON(view)
	require.NoError(t, err)
	decoded, err := DecodeCertificate(bytes.NewReader(encoded), protocol.DefaultDecodeLimit, view)
	require.NoError(t, err)
	require.Equal(t, certificate, decoded)

	corrupted := certificate
	corrupted.Nodes[1].Action = protocol.ActionKindPersistSuccess
	require.Error(t, corrupted.Validate(view))
	partial := certificate
	partial.Nodes = partial.Nodes[:len(partial.Nodes)-1]
	require.Error(t, partial.Validate(view))
}

func TestResourceLimitPublishesLastCompleteCheckpoint(t *testing.T) {
	t.Parallel()

	view := soundView(t)
	options := testOptions(2)
	options.Limits.MaxStates = options.Replicas
	var checkpoint Checkpoint
	options.Checkpoint = func(value Checkpoint) error {
		checkpoint = value
		return nil
	}
	_, err := Produce(context.Background(), view, options, nil)
	var resourceErr *ResourceError
	require.ErrorAs(t, err, &resourceErr)
	require.Equal(t, "states", resourceErr.Resource)
	require.Len(t, checkpoint.Nodes, options.Replicas)
	require.Equal(t, -1, checkpoint.CompletedDepth)
}

func BenchmarkParallelProducerTenReplicas(b *testing.B) {
	view, found, err := protocol.DefaultFirstOrderView(protocol.TargetIDNexusCancellation, "sound")
	require.NoError(b, err)
	require.True(b, found)
	options := testOptions(8)
	b.ResetTimer()
	for range b.N {
		certificate, err := Produce(context.Background(), view, options, nil)
		require.NoError(b, err)
		encoded, err := certificate.CanonicalJSON(view)
		require.NoError(b, err)
		b.ReportMetric(float64(certificate.Statistics.StateBytes), "state-bytes")
		b.ReportMetric(float64(len(encoded)), "certificate-bytes")
	}
}

func BenchmarkLeanChecksTenReplicaCertificate(b *testing.B) {
	command := os.Getenv("UMPIRE3_NATIVE_CERTIFICATE_CHECK")
	if command == "" {
		b.Skip("canonical native certificate checker executable is not configured")
	}
	view := soundView(b)
	certificate, err := Produce(context.Background(), view, testOptions(8), nil)
	require.NoError(b, err)
	b.ResetTimer()
	for range b.N {
		_, err := CheckCertificate(context.Background(), []string{command}, view, certificate)
		require.NoError(b, err)
	}
}

func soundView(t testing.TB) protocol.FirstOrderView {
	t.Helper()
	view, found, err := protocol.DefaultFirstOrderView(protocol.TargetIDNexusCancellation, "sound")
	require.NoError(t, err)
	require.True(t, found)
	return view
}

func testOptions(workers int) Options {
	return Options{
		Workers: workers, Replicas: 10,
		Limits: SearchLimits{
			MaxDepth: 32, MaxStates: 1024, MaxTransitions: 16384, MaxStateBytes: 1 << 20,
		},
	}
}
