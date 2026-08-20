package temporal

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tests/umpire3/environment"
	"go.temporal.io/server/tests/umpire3/protocol"
	"go.temporal.io/server/tests/umpire3/regress"
	regressnexus "go.temporal.io/server/tests/umpire3/regress/nexus"
	umpire3runtime "go.temporal.io/server/tests/umpire3/runtime"
	"go.temporal.io/server/tests/umpire3/umpire3test"
)

func TestNexusFactoryPreparationFailure(t *testing.T) {
	factory := NewNexusFactory(func(context.Context) (ClusterInfo, error) {
		return ClusterInfo{}, errors.New("cluster unavailable")
	}, NexusOptions{})

	_, err := factory.Prepare(context.Background(), loadNexusExperiment(t))
	require.ErrorContains(t, err, "cluster unavailable")
}

func TestNexusFactoryLearnsMintedIdentity(t *testing.T) {
	experiment := loadNexusExperiment(t)
	experiment.Actions[1].Bindings = []protocol.Binding{{
		Symbol: "learned-operation", Type: "identity", Projection: "operation-id",
	}}
	factory := NewNexusFactory(func(context.Context) (ClusterInfo, error) {
		return ClusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "server-operation"}, nil
	}, NexusOptions{})
	session, err := factory.Prepare(context.Background(), experiment)
	require.NoError(t, err)

	_, err = session.Realize(context.Background(), experiment.Actions[0], environment.Bindings{})
	require.NoError(t, err)
	evidence, err := session.Realize(context.Background(), experiment.Actions[1], environment.Bindings{})
	require.NoError(t, err)
	require.Equal(t, "server-operation", evidence.GroundedBindings["learned-operation"])
}

func TestEveryNexusRealizerHonorsCancellation(t *testing.T) {
	experiment := loadNexusExperiment(t)
	factory := NewNexusFactory(func(context.Context) (ClusterInfo, error) {
		return ClusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "operation"}, nil
	}, NexusOptions{})
	for _, action := range experiment.Actions {
		session, err := factory.Prepare(context.Background(), experiment)
		require.NoError(t, err)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = session.Realize(ctx, action, environment.Bindings{})
		require.ErrorIs(t, err, context.Canceled, action.Kind)
	}
}

func TestNexusNegativeControlChangesImplementationBehavior(t *testing.T) {
	experiment := loadNexusExperiment(t)
	probe := func(context.Context) (ClusterInfo, error) {
		return ClusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "server-operation"}, nil
	}

	sound, err := umpire3runtime.Run(context.Background(), umpire3runtime.Request{
		Experiment:  experiment,
		Environment: NewNexusFactory(probe, NexusOptions{}),
	})
	require.NoError(t, err)
	require.Equal(t, umpire3runtime.ClaimConforming, sound.Claim.Kind)
	require.Equal(t, "build", sound.Environment.BuildID)

	faulty, err := umpire3runtime.Run(context.Background(), umpire3runtime.Request{
		Experiment:  experiment,
		Environment: NewNexusFactory(probe, NexusOptions{AllowStaleSuccess: true}),
	})
	require.NoError(t, err)
	require.Equal(t, umpire3runtime.ClaimViolating, faulty.Claim.Kind)
}

func TestTypedNexusRegressionFacadeRuns(t *testing.T) {
	operation := regressnexus.Operation("operation")
	scenario := regressnexus.Regression("typed-nexus-cancellation", operation,
		regress.OnePath(operation.CancelWithRetry(), operation.CancellationSafety()))
	factory := NewNexusFactory(func(context.Context) (ClusterInfo, error) {
		return ClusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "operation"}, nil
	}, NexusOptions{})

	umpire3test.RequireRegression(t, scenario, umpire3test.WithEnvironment(factory))
}

func TestNexusTaskTransportDeterminesVisibleCompletion(t *testing.T) {
	experiment := loadNexusExperiment(t)
	probe := func(context.Context) (ClusterInfo, error) {
		return ClusterInfo{BuildID: "build", Namespace: "namespace"}, nil
	}

	for _, testCase := range []struct {
		name              string
		allowStaleSuccess bool
		wantClaim         umpire3runtime.ClaimKind
		wantReportSuccess bool
	}{
		{name: "sound worker", wantClaim: umpire3runtime.ClaimConforming},
		{
			name:              "faulty worker",
			allowStaleSuccess: true,
			wantClaim:         umpire3runtime.ClaimViolating,
			wantReportSuccess: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			transport := &fakeNexusTaskTransport{}
			result, err := umpire3runtime.Run(context.Background(), umpire3runtime.Request{
				Experiment: experiment,
				Environment: NewNexusFactory(probe, NexusOptions{
					AllowStaleSuccess: testCase.allowStaleSuccess,
					TaskTransport:     transport,
				}),
			})
			require.NoError(t, err)
			require.Equal(t, testCase.wantClaim, result.Claim.Kind)
			require.Equal(t, "server-minted-operation", result.Bindings["operation"])
			require.Equal(t, []bool{testCase.wantReportSuccess}, transport.reportedSuccess)
			require.True(t, transport.cleaned)
		})
	}
}

func TestNexusObservationsCarryCausalEvidence(t *testing.T) {
	experiment := loadNexusExperiment(t)
	factory := NewNexusFactory(func(context.Context) (ClusterInfo, error) {
		return ClusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "operation"}, nil
	}, NexusOptions{})
	result, err := umpire3runtime.Run(context.Background(), umpire3runtime.Request{
		Experiment:  experiment,
		Environment: factory,
	})
	require.NoError(t, err)
	for _, observation := range result.Observations {
		require.NotEmpty(t, observation.Source)
		require.NotEmpty(t, observation.CausalReference)
	}
}

func TestNexusViolationMinimizesWithoutChangingCheckpoint(t *testing.T) {
	experiment := loadNexusExperiment(t)
	experiment.Scope.Bounds.MaxDepth = 10
	experiment.Actions = append(experiment.Actions[:5], append([]protocol.Action{
		{Identifier: "irrelevant-crash", Kind: "crash-owner", RequiredCapabilities: []string{"failover-control"}},
		{Identifier: "irrelevant-recover", Kind: "recover-owner", RequiredCapabilities: []string{"failover-control"}},
	}, experiment.Actions[5:]...)...)
	probe := func(context.Context) (ClusterInfo, error) {
		return ClusterInfo{BuildID: "build", Namespace: "namespace", MintedOperationID: "operation"}, nil
	}

	minimized, err := umpire3runtime.MinimizeActions(context.Background(), experiment,
		func(ctx context.Context, candidate protocol.Experiment) (umpire3runtime.Result, error) {
			return umpire3runtime.Run(ctx, umpire3runtime.Request{
				Experiment:  candidate,
				Environment: NewNexusFactory(probe, NexusOptions{AllowStaleSuccess: true}),
			})
		})
	require.NoError(t, err)
	for _, action := range minimized.Actions {
		require.NotEqual(t, "crash-owner", action.Kind)
		require.NotEqual(t, "recover-owner", action.Kind)
	}
	require.Less(t, len(minimized.Actions), len(experiment.Actions))
}

func loadNexusExperiment(t *testing.T) protocol.Experiment {
	t.Helper()
	encoded, err := os.ReadFile("../testdata/nexus-cancellation.json")
	require.NoError(t, err)
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(encoded), protocol.DefaultDecodeLimit)
	require.NoError(t, err)
	return experiment
}

type fakeNexusTaskTransport struct {
	reportedSuccess []bool
	cleaned         bool
}

func (f *fakeNexusTaskTransport) Dispatch(context.Context) (NexusTask, error) {
	return NexusTask{
		OperationID: "server-minted-operation",
		Source:      "fake-matching",
		Reference:   "fake/task",
	}, nil
}

func (f *fakeNexusTaskTransport) Complete(_ context.Context, completion NexusTaskCompletion) (NexusTaskOutcome, error) {
	f.reportedSuccess = append(f.reportedSuccess, completion.ReportSuccess)
	return NexusTaskOutcome{
		SuccessVisible: completion.ReportSuccess,
		Source:         "fake-matching",
		Reference:      "fake/result",
	}, nil
}

func (f *fakeNexusTaskTransport) Cleanup(context.Context) error {
	f.cleaned = true
	return nil
}
