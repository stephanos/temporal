package cli

import (
	"bytes"
	"context"
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

type stressInspectionBackend struct {
	commandBackend
	selected *bool
}

func (p stressInspectionBackend) ForStressReplay() (backends.Provider, error) {
	*p.selected = true
	return p, nil
}

func (p stressInspectionBackend) ForDeterminismFixtureReplay() (backends.Provider, error) {
	*p.selected = true
	return p, nil
}

func TestStressReplayFlagRequiresInspectionAndExplicitProvider(t *testing.T) {
	for _, test := range []struct {
		name      string
		arguments []string
		provider  bool
		status    int
	}{
		{"no-stop", []string{"--stress-jitter", "/artifact"}, true, 2},
		{"foreign-provider", []string{"--stress-jitter", "--stop=choice=2", "/artifact"}, false, 2},
		{"inspection", []string{"--stress-jitter", "--stop=choice=2", "/artifact"}, true, 0},
		{"fixture-no-stop", []string{"--determinism-fixture", "/artifact"}, true, 2},
		{"fixture-foreign-provider", []string{"--determinism-fixture", "--stop=choice=2", "/artifact"}, false, 2},
		{"fixture-inspection", []string{"--determinism-fixture", "--stop=choice=2", "/artifact"}, true, 0},
		{"mixed-roles", []string{"--determinism-fixture", "--stress-jitter", "--stop=choice=2", "/artifact"}, true, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			selected, replayed := false, false
			var output, diagnostic bytes.Buffer
			status := runReplayWith(test.arguments, &output, &diagnostic, replayDependencies{
				install: func(string) (installation, error) {
					var provider backends.Provider = commandBackend{}
					if test.provider {
						provider = stressInspectionBackend{selected: &selected}
					}
					return installation{backend: provider}, nil
				},
				replay: func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
					replayed = true
					require.True(t, selected)
					require.Equal(t, backends.InspectionStop{Kind: "choice", Value: 2}, spec.Inspection.Stop)
					return runner.ReplayResult{Inspection: &backends.InspectionResult{Status: backends.InspectionStatusInspectedPrefix, PrefixVerified: true, WitnessMatch: true}}, nil
				},
			})
			require.Equal(t, test.status, status, diagnostic.String())
			require.Equal(t, test.status == 0, replayed)
		})
	}
}
