package family

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
)

func TestPlanSelectsOnlyTheFamilyCheckerPortfolio(t *testing.T) {
	t.Parallel()

	graph, err := protocolcatalog.DefaultFamilyDependencyGraph()
	require.NoError(t, err)

	nexus, err := PlanFor(graph, protocolcatalog.TargetIDNexusCancellation, "/repo")
	require.NoError(t, err)
	require.Contains(t, nexus.MakeTargets, "umpire3-check-native-results")
	require.Contains(t, nexus.MakeTargets, "umpire3-check-veil-results")
	require.NotContains(t, nexus.MakeTargets, "umpire3-check-temporal")
	require.Contains(t, nexus.GoPackages, "./tests/umpire3/checker/veil")
	require.NotEmpty(t, nexus.LeanModules)

	delivery, err := PlanFor(graph, protocolcatalog.TargetIDFoundationDeliverySafety, "/repo")
	require.NoError(t, err)
	require.Contains(t, delivery.MakeTargets, "umpire3-check-temporal")
	require.NotContains(t, delivery.MakeTargets, "umpire3-check-native-results")
	require.NotContains(t, delivery.MakeTargets, "umpire3-check-veil-results")
}

func TestPlanRejectsUnknownFamilyBeforeRunningCommands(t *testing.T) {
	t.Parallel()

	graph, err := protocolcatalog.DefaultFamilyDependencyGraph()
	require.NoError(t, err)
	_, err = PlanFor(graph, "unknown", "/repo")
	require.EqualError(t, err, `unknown Umpire3 model family "unknown"`)
}

func TestRunStopsAtTheFirstFailedGate(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{failAt: 2}
	err := Run(context.Background(), Plan{
		RepositoryRoot: "/repo", LeanModules: []string{"Temporal.Feature.Nexus"},
		MakeTargets: []string{"umpire3-check-finite-replay"},
		GoPackages:  []string{"./tests/umpire3/protocol/..."},
	}, runner)
	require.ErrorContains(t, err, "run generated-artifact gates")
	require.Len(t, runner.commands, 2)
}

type recordingRunner struct {
	commands []Command
	failAt   int
}

func (r *recordingRunner) Run(_ context.Context, command Command) error {
	r.commands = append(r.commands, command)
	if len(r.commands) == r.failAt {
		return errors.New("injected failure")
	}
	return nil
}
