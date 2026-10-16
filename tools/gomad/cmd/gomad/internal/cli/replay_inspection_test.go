package cli

import (
	"bytes"
	"flag"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/runner"
)

func TestReplayInspectionFlagsBindTargetStopAndBounds(t *testing.T) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := newReplayInspectionFlags(flags)
	require.NoError(t, flags.Parse([]string{"--stop=choice=2", "--instance=a", "--incarnation=2", "--dump=state", "--probe=queue.*", "--debug-bytes=4MiB", "--dump-bytes=1MiB"}))
	request, err := options.request(flags, "/artifacts/failure/")
	require.NoError(t, err)
	require.Equal(t, "a", request.Instance)
	require.Equal(t, uint64(2), request.Incarnation)
	require.Equal(t, backends.InspectionStop{Kind: "choice", Value: 2}, request.Stop)
	require.Equal(t, []string{"state"}, request.Dumps)
	require.Equal(t, []string{"queue.*"}, request.Probes)
	require.Equal(t, uint64(4<<20), request.MaximumBytes)
	require.Equal(t, "/artifacts/failure.inspection", request.OutputDirectory)
}

func TestReplayInspectionFlagsRejectInvalidSelectorsAndSearchBounds(t *testing.T) {
	for _, arguments := range [][]string{{"--stop=choice=-1"}, {"--stop=time=-1ms"}, {"--stop=other"}, {"--stop=time=999999999999999999999ns"}, {"--instance=a"}, {"--incarnation=2"}, {"--check=healthy"}, {"--choice-range=0:2"}, {"--check=healthy", "--choice-range=3:2"}, {"--check=healthy", "--choice-range=0:2", "--search-attempts=0"}, {"--check=healthy", "--choice-range=0:2", "--search-timeout=0s"}, {"--check=healthy", "--choice-range=0:2", "--stop=failure"}, {"--debug-bytes=1MiB", "--dump-bytes=2MiB"}} {
		t.Run(arguments[0], func(t *testing.T) {
			flags := flag.NewFlagSet("test", flag.ContinueOnError)
			flags.SetOutput(io.Discard)
			options := newReplayInspectionFlags(flags)
			require.NoError(t, flags.Parse(arguments))
			_, err := options.request(flags, filepath.Join(t.TempDir(), "artifact"))
			require.Error(t, err)
		})
	}
}

func TestReplayInspectionReportDoesNotClaimFullReproduction(t *testing.T) {
	var output bytes.Buffer
	status, err := reportReplayResult(&output, runner.ReplayResult{Inspection: &backends.InspectionResult{Status: backends.InspectionStatusInspectedPrefix, PrefixVerified: true, WitnessMatch: true, Requested: backends.InspectionRequest{Stop: backends.InspectionStop{Kind: "failure"}}, Resolved: backends.InspectionBoundary{Instance: "a", Incarnation: 2, Choice: 3}}})
	require.NoError(t, err)
	require.Zero(t, status)
	require.Contains(t, output.String(), `"status":"inspected_prefix"`)
	require.Contains(t, output.String(), `"incarnation":2`)
	require.NotContains(t, output.String(), "reproduced")
}
